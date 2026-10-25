package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/SV1Stail/online_school/internal/conference"
	"github.com/SV1Stail/online_school/internal/user"
)

const conferenceLinkColumns = `id, teacher_id, provider, title, url, active, display_order, created_at, updated_at`

func (s *Store) ListForTeacher(
	ctx context.Context,
	teacherID string,
	includeInactive bool,
) ([]conference.Link, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+conferenceLinkColumns+`
		FROM teacher_conference_links
		WHERE teacher_id = $1
			AND deleted_at IS NULL
			AND ($2 OR active = true)
		ORDER BY display_order, title, id
	`, teacherID, includeInactive)
	if err != nil {
		return nil, fmt.Errorf("list conference links: %w", err)
	}
	defer rows.Close()

	result := make([]conference.Link, 0)
	for rows.Next() {
		var item conference.Link
		if err := scanConferenceLink(rows, &item); err != nil {
			return nil, fmt.Errorf("scan conference link: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate conference links: %w", err)
	}
	return result, nil
}

func (s *Store) ListForBooking(
	ctx context.Context,
	bookingID string,
	actor *user.User,
) ([]conference.Link, error) {
	rows, err := s.pool.Query(ctx, `
		WITH target AS (
			SELECT b.teacher_id, b.student_id, b.starts_at
			FROM bookings b
			WHERE b.id = $1
				AND b.status = 'confirmed'
				AND (
					$2 = 'moderator'
					OR ($2 = 'teacher' AND b.teacher_id = $3)
					OR ($2 = 'student' AND b.student_id = $3)
				)
		), selected AS (
			SELECT l.id, l.teacher_id, l.provider,
				CASE WHEN target.starts_at < now() THEN v.title ELSE l.title END AS title,
				CASE WHEN target.starts_at < now() THEN v.url ELSE l.url END AS url,
				true AS active,
				CASE WHEN target.starts_at < now() THEN v.display_order ELSE l.display_order END AS display_order,
				CASE WHEN target.starts_at < now() THEN v.valid_from ELSE l.created_at END AS created_at,
				CASE WHEN target.starts_at < now() THEN COALESCE(v.valid_to, v.valid_from) ELSE l.updated_at END AS updated_at
			FROM target
			JOIN teacher_conference_links l ON l.teacher_id = target.teacher_id
			LEFT JOIN conference_link_versions v
				ON v.link_id = l.id
				AND target.starts_at < now()
				AND tstzrange(v.valid_from, v.valid_to, '[)') @> target.starts_at
			WHERE (
				target.starts_at >= now() AND l.deleted_at IS NULL AND l.active = true
			) OR (
				target.starts_at < now() AND v.active = true
			)
		)
		SELECT `+conferenceLinkColumns+` FROM selected
		ORDER BY display_order, title, id
	`, bookingID, actor.Role, actor.ID)
	if err != nil {
		return nil, fmt.Errorf("list booking conference links: %w", err)
	}
	defer rows.Close()

	result := make([]conference.Link, 0)
	for rows.Next() {
		var item conference.Link
		if err := scanConferenceLink(rows, &item); err != nil {
			return nil, fmt.Errorf("scan booking conference link: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate booking conference links: %w", err)
	}
	return result, nil
}

func (s *Store) CreateConferenceLink(ctx context.Context, input conference.CreateInput) (*conference.Link, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin create conference link: %w", err)
	}
	defer rollback(tx)

	row := tx.QueryRow(ctx, `
		INSERT INTO teacher_conference_links (teacher_id, provider, title, url, active, display_order)
		SELECT $1, $2, $3, $4, $5, $6
		FROM users
		WHERE id = $1 AND role = 'teacher' AND active = true
		RETURNING `+conferenceLinkColumns,
		input.TeacherID,
		input.Provider,
		input.Title,
		input.URL,
		input.Active,
		input.DisplayOrder,
	)
	created := &conference.Link{}
	if err := scanConferenceLink(row, created); err != nil {
		if isNoRows(err) {
			return nil, conference.ErrInvalidInput
		}
		return nil, fmt.Errorf("insert conference link: %w", err)
	}
	if err := insertConferenceVersion(ctx, tx, created, created.CreatedAt); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create conference link: %w", err)
	}
	return created, nil
}

func (s *Store) UpdateConferenceLink(
	ctx context.Context,
	id, teacherID string,
	moderator bool,
	input conference.UpdateInput,
) (*conference.Link, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin update conference link: %w", err)
	}
	defer rollback(tx)

	now := time.Now().UTC()
	row := tx.QueryRow(ctx, `
		UPDATE conference_link_versions v
		SET valid_to = $4
		FROM teacher_conference_links l
		WHERE l.id = $1
			AND l.deleted_at IS NULL
			AND ($3 OR l.teacher_id = $2)
			AND v.link_id = l.id
			AND v.valid_to IS NULL
		RETURNING l.teacher_id
	`, id, teacherID, moderator, now)
	var ownerID string
	if err := row.Scan(&ownerID); err != nil {
		if isNoRows(err) {
			return nil, conference.ErrNotFound
		}
		return nil, fmt.Errorf("close conference link version: %w", err)
	}

	row = tx.QueryRow(ctx, `
		UPDATE teacher_conference_links
		SET title = COALESCE($2, title),
			url = COALESCE($3, url),
			active = COALESCE($4, active),
			display_order = COALESCE($5, display_order),
			updated_at = $6
		WHERE id = $1
		RETURNING `+conferenceLinkColumns,
		id,
		input.Title,
		input.URL,
		input.Active,
		input.DisplayOrder,
		now,
	)
	updated := &conference.Link{}
	if err := scanConferenceLink(row, updated); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23514" {
			return nil, conference.ErrInvalidInput
		}
		return nil, fmt.Errorf("update conference link: %w", err)
	}
	if err := insertConferenceVersion(ctx, tx, updated, now); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit update conference link: %w", err)
	}
	return updated, nil
}

func (s *Store) DeleteConferenceLink(ctx context.Context, id, teacherID string, moderator bool) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin delete conference link: %w", err)
	}
	defer rollback(tx)

	now := time.Now().UTC()
	command, err := tx.Exec(ctx, `
		UPDATE teacher_conference_links
		SET deleted_at = $4, active = false, updated_at = $4
		WHERE id = $1
			AND provider = 'custom'
			AND deleted_at IS NULL
			AND ($3 OR teacher_id = $2)
	`, id, teacherID, moderator, now)
	if err != nil {
		return fmt.Errorf("delete conference link: %w", err)
	}
	if command.RowsAffected() == 0 {
		return conference.ErrNotFound
	}
	if _, err := tx.Exec(ctx, `
		UPDATE conference_link_versions SET valid_to = $2
		WHERE link_id = $1 AND valid_to IS NULL
	`, id, now); err != nil {
		return fmt.Errorf("close deleted conference link version: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit delete conference link: %w", err)
	}
	return nil
}

func insertConferenceVersion(ctx context.Context, tx pgx.Tx, link *conference.Link, validFrom time.Time) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO conference_link_versions (
			link_id, title, url, active, display_order, valid_from
		)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, link.ID, link.Title, link.URL, link.Active, link.DisplayOrder, validFrom)
	if err != nil {
		return fmt.Errorf("insert conference link version: %w", err)
	}
	return nil
}

func scanConferenceLink(row rowScanner, destination *conference.Link) error {
	return row.Scan(
		&destination.ID,
		&destination.TeacherID,
		&destination.Provider,
		&destination.Title,
		&destination.URL,
		&destination.Active,
		&destination.DisplayOrder,
		&destination.CreatedAt,
		&destination.UpdatedAt,
	)
}
