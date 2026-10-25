package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/SV1Stail/online_school/internal/schedule"
	"github.com/SV1Stail/online_school/internal/user"
)

const availabilityColumns = `id, teacher_id, starts_at, ends_at, allowed_durations, created_at`
const bookingColumns = `id, availability_id, teacher_id, student_id, starts_at, ends_at,
	duration_minutes, status, created_by, created_at, updated_at`

func (s *Store) CreateAvailability(
	ctx context.Context,
	input schedule.CreateAvailabilityInput,
	createdBy string,
) (*schedule.Availability, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO availability_windows (
			teacher_id, starts_at, ends_at, allowed_durations, created_by
		)
		SELECT $1, $2, $3, $4, $5
		FROM users
		WHERE id = $1 AND role = 'teacher' AND active = true
		RETURNING `+availabilityColumns,
		input.TeacherID,
		input.StartsAt.UTC(),
		input.EndsAt.UTC(),
		input.AllowedDurations,
		createdBy,
	)
	created := &schedule.Availability{}
	if err := scanAvailability(row, created); err != nil {
		if isNoRows(err) {
			return nil, schedule.ErrInvalidInput
		}
		if isExclusionViolation(err) {
			return nil, schedule.ErrConflict
		}
		return nil, fmt.Errorf("create availability: %w", err)
	}
	return created, nil
}

func (s *Store) ListAvailability(
	ctx context.Context,
	teacherID string,
	from, to time.Time,
) ([]schedule.Availability, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+availabilityColumns+`
		FROM availability_windows
		WHERE teacher_id = $1
			AND deleted_at IS NULL
			AND tstzrange(starts_at, ends_at, '[)') && tstzrange($2, $3, '[)')
		ORDER BY starts_at, id
	`, teacherID, from, to)
	if err != nil {
		return nil, fmt.Errorf("list availability: %w", err)
	}
	defer rows.Close()

	result := make([]schedule.Availability, 0)
	for rows.Next() {
		var item schedule.Availability
		if err := scanAvailability(rows, &item); err != nil {
			return nil, fmt.Errorf("scan availability: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate availability: %w", err)
	}
	return result, nil
}

func (s *Store) LoadSlotData(
	ctx context.Context,
	teacherID, studentID string,
	from, to time.Time,
) (*schedule.SlotData, error) {
	availability, err := s.ListAvailability(ctx, teacherID, from, to)
	if err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT starts_at, ends_at
		FROM bookings
		WHERE status IN ('pending', 'confirmed')
			AND starts_at < $4 AND ends_at > $3
			AND (teacher_id = $1 OR student_id = NULLIF($2, '')::uuid)
		ORDER BY starts_at, ends_at
	`, teacherID, studentID, from, to)
	if err != nil {
		return nil, fmt.Errorf("load blocked booking ranges: %w", err)
	}
	defer rows.Close()

	blocked := make([]schedule.TimeRange, 0)
	for rows.Next() {
		var item schedule.TimeRange
		if err := rows.Scan(&item.StartsAt, &item.EndsAt); err != nil {
			return nil, fmt.Errorf("scan blocked booking range: %w", err)
		}
		blocked = append(blocked, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate blocked booking ranges: %w", err)
	}
	return &schedule.SlotData{Availability: availability, Blocked: blocked}, nil
}

func (s *Store) DeleteAvailability(ctx context.Context, id, teacherID string, moderator bool) error {
	command, err := s.pool.Exec(ctx, `
		UPDATE availability_windows
		SET deleted_at = now()
		WHERE id = $1
			AND deleted_at IS NULL
			AND ($3 OR teacher_id = $2)
			AND NOT EXISTS (
				SELECT 1 FROM bookings
				WHERE availability_id = availability_windows.id
					AND status IN ('pending', 'confirmed')
			)
	`, id, teacherID, moderator)
	if err != nil {
		return fmt.Errorf("delete availability: %w", err)
	}
	if command.RowsAffected() == 0 {
		return schedule.ErrNotFound
	}
	return nil
}

func (s *Store) CreateBooking(ctx context.Context, input schedule.CreateBookingRecord) (*schedule.Booking, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return nil, fmt.Errorf("begin create booking: %w", err)
	}
	defer rollback(tx)

	existing, err := findBookingByIdempotency(ctx, tx, input.CreatedBy, input.IdempotencyKey)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("check booking idempotency: %w", err)
	}

	var availabilityStart time.Time
	var availabilityEnd time.Time
	var allowedDurations []int16
	err = tx.QueryRow(ctx, `
		SELECT aw.starts_at, aw.ends_at, aw.allowed_durations
		FROM availability_windows aw
		JOIN users teacher ON teacher.id = aw.teacher_id
		JOIN users student ON student.id = $3
		WHERE aw.id = $1
			AND aw.teacher_id = $2
			AND aw.deleted_at IS NULL
			AND teacher.role = 'teacher' AND teacher.active = true
			AND student.role = 'student' AND student.active = true
		FOR SHARE OF aw
	`, input.AvailabilityID, input.TeacherID, input.StudentID).Scan(
		&availabilityStart,
		&availabilityEnd,
		&allowedDurations,
	)
	if err != nil {
		if isNoRows(err) {
			return nil, schedule.ErrInvalidInput
		}
		return nil, fmt.Errorf("validate booking: %w", err)
	}
	if !containsDuration(allowedDurations, input.Duration) ||
		input.StartsAt.Before(availabilityStart) ||
		input.EndsAt.After(availabilityEnd) ||
		!schedule.IsAlignedWithStep(availabilityStart, input.StartsAt) {
		return nil, schedule.ErrInvalidInput
	}

	row := tx.QueryRow(ctx, `
		INSERT INTO bookings (
			availability_id, teacher_id, student_id, starts_at, ends_at,
			duration_minutes, status, created_by, idempotency_key
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING `+bookingColumns,
		input.AvailabilityID,
		input.TeacherID,
		input.StudentID,
		input.StartsAt.UTC(),
		input.EndsAt.UTC(),
		input.Duration,
		input.Status,
		input.CreatedBy,
		input.IdempotencyKey,
	)
	created := &schedule.Booking{}
	if err := scanBooking(row, created); err != nil {
		if isExclusionViolation(err) {
			return nil, schedule.ErrConflict
		}
		if isUniqueViolation(err) {
			existing, findErr := findBookingByIdempotency(ctx, tx, input.CreatedBy, input.IdempotencyKey)
			if findErr == nil {
				return existing, nil
			}
		}
		return nil, fmt.Errorf("insert booking: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create booking: %w", err)
	}
	return created, nil
}

func (s *Store) FindBooking(ctx context.Context, id string) (*schedule.Booking, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+bookingColumns+` FROM bookings WHERE id = $1`, id)
	result := &schedule.Booking{}
	if err := scanBooking(row, result); err != nil {
		if isNoRows(err) {
			return nil, schedule.ErrNotFound
		}
		return nil, fmt.Errorf("find booking: %w", err)
	}
	return result, nil
}

func (s *Store) ListBookings(
	ctx context.Context,
	actor *user.User,
	from, to time.Time,
) ([]schedule.Booking, error) {
	query := `
		SELECT ` + bookingColumns + `
		FROM bookings
		WHERE starts_at < $2 AND ends_at > $1`
	args := []any{from, to}
	switch actor.Role {
	case user.RoleStudent:
		query += ` AND student_id = $3`
		args = append(args, actor.ID)
	case user.RoleTeacher:
		query += ` AND teacher_id = $3`
		args = append(args, actor.ID)
	case user.RoleModerator:
	default:
		return nil, schedule.ErrForbidden
	}
	query += ` ORDER BY starts_at, id`

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list bookings: %w", err)
	}
	defer rows.Close()

	result := make([]schedule.Booking, 0)
	for rows.Next() {
		var item schedule.Booking
		if err := scanBooking(rows, &item); err != nil {
			return nil, fmt.Errorf("scan booking: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bookings: %w", err)
	}
	return result, nil
}

func (s *Store) UpdateBookingStatus(
	ctx context.Context,
	id string,
	from []schedule.BookingStatus,
	to schedule.BookingStatus,
) (*schedule.Booking, error) {
	statusValues := make([]string, len(from))
	for index, status := range from {
		statusValues[index] = string(status)
	}
	row := s.pool.QueryRow(ctx, `
		UPDATE bookings
		SET status = $3, updated_at = now()
		WHERE id = $1 AND status = ANY($2)
		RETURNING `+bookingColumns,
		id,
		statusValues,
		to,
	)
	updated := &schedule.Booking{}
	if err := scanBooking(row, updated); err != nil {
		if isNoRows(err) {
			return nil, schedule.ErrInvalidTransition
		}
		if isExclusionViolation(err) {
			return nil, schedule.ErrConflict
		}
		return nil, fmt.Errorf("update booking status: %w", err)
	}
	return updated, nil
}

func findBookingByIdempotency(
	ctx context.Context,
	tx pgx.Tx,
	createdBy, key string,
) (*schedule.Booking, error) {
	row := tx.QueryRow(ctx, `
		SELECT `+bookingColumns+`
		FROM bookings
		WHERE created_by = $1 AND idempotency_key = $2
	`, createdBy, key)
	result := &schedule.Booking{}
	if err := scanBooking(row, result); err != nil {
		return nil, err
	}
	return result, nil
}

func scanAvailability(row rowScanner, destination *schedule.Availability) error {
	return row.Scan(
		&destination.ID,
		&destination.TeacherID,
		&destination.StartsAt,
		&destination.EndsAt,
		&destination.AllowedDurations,
		&destination.CreatedAt,
	)
}

func scanBooking(row rowScanner, destination *schedule.Booking) error {
	return row.Scan(
		&destination.ID,
		&destination.AvailabilityID,
		&destination.TeacherID,
		&destination.StudentID,
		&destination.StartsAt,
		&destination.EndsAt,
		&destination.Duration,
		&destination.Status,
		&destination.CreatedBy,
		&destination.CreatedAt,
		&destination.UpdatedAt,
	)
}

func isExclusionViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23P01"
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func containsDuration(values []int16, expected int16) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
