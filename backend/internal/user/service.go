package user

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SV1Stail/online_school/internal/security"
)

var (
	ErrNotFound      = errors.New("user not found")
	ErrLoginConflict = errors.New("login already exists")
	ErrInvalidInput  = errors.New("invalid user input")
)

type SessionRevoker interface {
	DeleteUserSessions(ctx context.Context, userID string) error
}

type Auditor interface {
	Write(ctx context.Context, actorID, action, entityType, entityID string, details map[string]any) error
}

type Service struct {
	repository Repository
	sessions   SessionRevoker
	auditor    Auditor
}

func NewService(repository Repository, sessions SessionRevoker, auditor Auditor) *Service {
	return &Service{repository: repository, sessions: sessions, auditor: auditor}
}

func (s *Service) ListTeachers(ctx context.Context) ([]User, error) {
	return s.repository.ListTeachers(ctx)
}

func (s *Service) ListAll(ctx context.Context) ([]User, error) {
	return s.repository.List(ctx)
}

func (s *Service) ListStudents(ctx context.Context) ([]User, error) {
	return s.repository.ListStudents(ctx)
}

func (s *Service) Create(ctx context.Context, actorID string, input CreateInput) (*User, error) {
	input.Login = strings.ToLower(strings.TrimSpace(input.Login))
	input.FullName = strings.TrimSpace(input.FullName)
	input.Timezone = strings.TrimSpace(input.Timezone)
	if input.Login == "" || len(input.Login) > 100 || input.FullName == "" || len(input.FullName) > 200 {
		return nil, ErrInvalidInput
	}
	if !input.Role.Valid() {
		return nil, ErrInvalidInput
	}
	if _, err := time.LoadLocation(input.Timezone); err != nil {
		return nil, fmt.Errorf("%w: invalid timezone", ErrInvalidInput)
	}
	passwordHash, err := security.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	created, err := s.repository.CreateUser(ctx, CreateRecord{
		Login:              input.Login,
		PasswordHash:       passwordHash,
		FullName:           input.FullName,
		Role:               input.Role,
		Timezone:           input.Timezone,
		MustChangePassword: true,
	})
	if err != nil {
		return nil, err
	}
	if err := s.auditor.Write(ctx, actorID, "user.created", "user", created.ID, map[string]any{"role": created.Role}); err != nil {
		return nil, err
	}
	return created, nil
}

func (s *Service) Update(ctx context.Context, actorID, id string, input UpdateInput) (*User, error) {
	if input.FullName != nil {
		trimmed := strings.TrimSpace(*input.FullName)
		if trimmed == "" || len(trimmed) > 200 {
			return nil, ErrInvalidInput
		}
		input.FullName = &trimmed
	}
	if input.Timezone != nil {
		trimmed := strings.TrimSpace(*input.Timezone)
		if _, err := time.LoadLocation(trimmed); err != nil {
			return nil, fmt.Errorf("%w: invalid timezone", ErrInvalidInput)
		}
		input.Timezone = &trimmed
	}
	updated, err := s.repository.UpdateUser(ctx, id, input)
	if err != nil {
		return nil, err
	}
	if input.Active != nil && !*input.Active {
		if err := s.sessions.DeleteUserSessions(ctx, id); err != nil {
			return nil, err
		}
	}
	if err := s.auditor.Write(ctx, actorID, "user.updated", "user", id, nil); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) ResetPassword(ctx context.Context, actorID, id, password string) error {
	passwordHash, err := security.HashPassword(password)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	if err := s.repository.UpdatePassword(ctx, id, passwordHash, true); err != nil {
		return err
	}
	if err := s.sessions.DeleteUserSessions(ctx, id); err != nil {
		return err
	}
	return s.auditor.Write(ctx, actorID, "user.password_reset", "user", id, nil)
}

func (s *Service) ChangeOwnPassword(ctx context.Context, current *User, oldPassword, newPassword string) error {
	credentials, err := s.repository.FindByLogin(ctx, current.Login)
	if err != nil {
		return err
	}
	valid, err := security.VerifyPassword(oldPassword, credentials.PasswordHash)
	if err != nil || !valid {
		return ErrInvalidInput
	}
	passwordHash, err := security.HashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}
	if err := s.repository.UpdatePassword(ctx, current.ID, passwordHash, false); err != nil {
		return err
	}
	if err := s.sessions.DeleteUserSessions(ctx, current.ID); err != nil {
		return err
	}
	return s.auditor.Write(ctx, current.ID, "user.password_changed", "user", current.ID, nil)
}
