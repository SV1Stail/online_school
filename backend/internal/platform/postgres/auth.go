package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/SV1Stail/online_school/internal/auth"
	"github.com/SV1Stail/online_school/internal/user"
)

func (s *Store) CreateSession(ctx context.Context, userID string, tokenHash []byte, expiresAt time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (token_hash, user_id, expires_at)
		VALUES ($1, $2, $3)
	`, tokenHash, userID, expiresAt)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (s *Store) FindSession(ctx context.Context, tokenHash []byte, now time.Time) (*user.User, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT `+prefixUserColumns("u")+`
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > $2 AND u.active = true
	`, tokenHash, now)
	result := &user.User{}
	if err := scanUser(row, result); err != nil {
		if isNoRows(err) {
			return nil, auth.ErrUnauthorized
		}
		return nil, fmt.Errorf("find session: %w", err)
	}
	return result, nil
}

func (s *Store) DeleteSession(ctx context.Context, tokenHash []byte) error {
	if _, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (s *Store) DeleteUserSessions(ctx context.Context, userID string) error {
	if _, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("delete user sessions: %w", err)
	}
	return nil
}

func (s *Store) DeleteExpiredSessions(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at <= now()`); err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}
	return nil
}

func prefixUserColumns(alias string) string {
	return alias + `.id, ` + alias + `.login, ` + alias + `.full_name, ` + alias + `.role, ` +
		alias + `.timezone, ` + alias + `.active, ` + alias + `.must_change_password, ` +
		alias + `.created_at, ` + alias + `.updated_at`
}
