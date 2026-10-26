package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/SV1Stail/online_school/internal/user"
)

const userColumns = `id, login, full_name, subject_label, role, timezone, active, must_change_password, created_at, updated_at`

func (s *Store) FindByLogin(ctx context.Context, login string) (*user.Credentials, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT `+userColumns+`, password_hash
		FROM users
		WHERE login = $1
	`, login)
	credentials := &user.Credentials{}
	if err := scanCredentials(row, credentials); err != nil {
		if isNoRows(err) {
			return nil, user.ErrNotFound
		}
		return nil, fmt.Errorf("find user by login: %w", err)
	}
	return credentials, nil
}

func (s *Store) FindByID(ctx context.Context, id string) (*user.User, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1`, id)
	result := &user.User{}
	if err := scanUser(row, result); err != nil {
		if isNoRows(err) {
			return nil, user.ErrNotFound
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return result, nil
}

func (s *Store) List(ctx context.Context) ([]user.User, error) {
	return s.listUsers(ctx, `SELECT `+userColumns+` FROM users ORDER BY created_at DESC`)
}

func (s *Store) ListTeachers(ctx context.Context) ([]user.User, error) {
	return s.listUsers(ctx, `
		SELECT `+userColumns+`
		FROM users
		WHERE role = 'teacher' AND active = true
		ORDER BY full_name, id
	`)
}

func (s *Store) ListStudents(ctx context.Context) ([]user.User, error) {
	return s.listUsers(ctx, `
		SELECT `+userColumns+`
		FROM users
		WHERE role = 'student' AND active = true
		ORDER BY full_name, id
	`)
}

func (s *Store) listUsers(ctx context.Context, query string) ([]user.User, error) {
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()

	result := make([]user.User, 0)
	for rows.Next() {
		var item user.User
		if err := scanUser(rows, &item); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		result = append(result, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}
	return result, nil
}

func (s *Store) CreateUser(ctx context.Context, input user.CreateRecord) (*user.User, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin create user: %w", err)
	}
	defer rollback(tx)

	row := tx.QueryRow(ctx, `
		INSERT INTO users (login, password_hash, full_name, role, timezone, must_change_password)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING `+userColumns,
		input.Login,
		input.PasswordHash,
		input.FullName,
		input.Role,
		input.Timezone,
		input.MustChangePassword,
	)
	created := &user.User{}
	if err := scanUser(row, created); err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, user.ErrLoginConflict
		}
		return nil, fmt.Errorf("insert user: %w", err)
	}

	if input.Role == user.RoleTeacher {
		if err := createDefaultConferenceLinks(ctx, tx, created.ID); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit create user: %w", err)
	}
	return created, nil
}

func (s *Store) UpdateUser(ctx context.Context, id string, input user.UpdateInput) (*user.User, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE users
		SET full_name = COALESCE($2, full_name),
			subject_label = COALESCE($3, subject_label),
			timezone = COALESCE($4, timezone),
			active = COALESCE($5, active),
			updated_at = now()
		WHERE id = $1
		RETURNING `+userColumns,
		id,
		input.FullName,
		input.SubjectLabel,
		input.Timezone,
		input.Active,
	)
	updated := &user.User{}
	if err := scanUser(row, updated); err != nil {
		if isNoRows(err) {
			return nil, user.ErrNotFound
		}
		return nil, fmt.Errorf("update user: %w", err)
	}
	return updated, nil
}

func (s *Store) UpdatePassword(ctx context.Context, id, passwordHash string, mustChange bool) error {
	command, err := s.pool.Exec(ctx, `
		UPDATE users
		SET password_hash = $2, must_change_password = $3, updated_at = now()
		WHERE id = $1
	`, id, passwordHash, mustChange)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}
	if command.RowsAffected() == 0 {
		return user.ErrNotFound
	}
	return nil
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanUser(row rowScanner, destination *user.User) error {
	return row.Scan(
		&destination.ID,
		&destination.Login,
		&destination.FullName,
		&destination.SubjectLabel,
		&destination.Role,
		&destination.Timezone,
		&destination.Active,
		&destination.MustChangePassword,
		&destination.CreatedAt,
		&destination.UpdatedAt,
	)
}

func scanCredentials(row rowScanner, destination *user.Credentials) error {
	return row.Scan(
		&destination.ID,
		&destination.Login,
		&destination.FullName,
		&destination.SubjectLabel,
		&destination.Role,
		&destination.Timezone,
		&destination.Active,
		&destination.MustChangePassword,
		&destination.CreatedAt,
		&destination.UpdatedAt,
		&destination.PasswordHash,
	)
}

func createDefaultConferenceLinks(ctx context.Context, tx pgx.Tx, teacherID string) error {
	_, err := tx.Exec(ctx, `
		WITH inserted AS (
			INSERT INTO teacher_conference_links (teacher_id, provider, title, display_order)
			VALUES
				($1, 'teams', 'Microsoft Teams', 10),
				($1, 'zoom', 'Zoom', 20),
				($1, 'telemost', 'Яндекс Телемост', 30),
				($1, 'google_meet', 'Google Meet', 40),
				($1, 'sber_jazz', 'СберJazz', 50)
			RETURNING id, title, url, active, display_order, created_at
		)
		INSERT INTO conference_link_versions (link_id, title, url, active, display_order, valid_from)
		SELECT id, title, url, active, display_order, created_at FROM inserted
	`, teacherID)
	if err != nil {
		return fmt.Errorf("create default conference links: %w", err)
	}
	return nil
}
