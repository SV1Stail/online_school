package conference

import (
	"context"
	"errors"
	"net/url"
	"strings"

	"github.com/SV1Stail/online_school/internal/user"
)

var (
	ErrNotFound     = errors.New("conference link not found")
	ErrForbidden    = errors.New("conference action is forbidden")
	ErrInvalidInput = errors.New("invalid conference link")
	ErrConflict     = errors.New("conference link already belongs to another student")
)

type Service struct {
	repository Repository
	auditor    Auditor
}

func NewService(repository Repository, auditor Auditor) *Service {
	return &Service{repository: repository, auditor: auditor}
}

func (s *Service) ListForStudent(ctx context.Context, actor *user.User, studentID string) ([]Link, error) {
	if actor.Role == user.RoleModerator || actor.Role == user.RoleTeacher {
		return s.repository.ListForStudent(ctx, studentID, true)
	}
	if actor.Role == user.RoleStudent && actor.ID == studentID {
		return s.repository.ListForStudent(ctx, studentID, false)
	}
	return nil, ErrForbidden
}

func (s *Service) ListForBooking(ctx context.Context, actor *user.User, bookingID string) ([]Link, error) {
	return s.repository.ListForBooking(ctx, bookingID, actor)
}

func (s *Service) Create(ctx context.Context, actor *user.User, input CreateInput) (*Link, error) {
	if actor.Role != user.RoleTeacher && actor.Role != user.RoleModerator {
		return nil, ErrForbidden
	}
	if input.Provider != ProviderCustom {
		return nil, ErrInvalidInput
	}
	if err := normalizeAndValidateCreate(&input); err != nil {
		return nil, err
	}
	created, err := s.repository.CreateConferenceLink(ctx, input)
	if err != nil {
		return nil, err
	}
	if err := s.auditor.Write(ctx, actor.ID, "conference_link.created", "conference_link", created.ID, nil); err != nil {
		return nil, err
	}
	return created, nil
}

func (s *Service) Update(
	ctx context.Context,
	actor *user.User,
	id, studentID string,
	input UpdateInput,
) (*Link, error) {
	if actor.Role != user.RoleTeacher && actor.Role != user.RoleModerator {
		return nil, ErrForbidden
	}
	if err := normalizeAndValidateUpdate(&input); err != nil {
		return nil, err
	}
	updated, err := s.repository.UpdateConferenceLink(ctx, id, studentID, input)
	if err != nil {
		return nil, err
	}
	if err := s.auditor.Write(ctx, actor.ID, "conference_link.updated", "conference_link", id, nil); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) Delete(ctx context.Context, actor *user.User, id, studentID string) error {
	if actor.Role != user.RoleTeacher && actor.Role != user.RoleModerator {
		return ErrForbidden
	}
	if err := s.repository.DeleteConferenceLink(ctx, id, studentID); err != nil {
		return err
	}
	return s.auditor.Write(ctx, actor.ID, "conference_link.deleted", "conference_link", id, nil)
}

func normalizeAndValidateCreate(input *CreateInput) error {
	input.Title = strings.TrimSpace(input.Title)
	input.URL = strings.TrimSpace(input.URL)
	if input.StudentID == "" || input.Title == "" || len(input.Title) > 100 || input.DisplayOrder < 0 {
		return ErrInvalidInput
	}
	return validateURL(input.URL, input.Active)
}

func normalizeAndValidateUpdate(input *UpdateInput) error {
	if input.Title != nil {
		trimmed := strings.TrimSpace(*input.Title)
		if trimmed == "" || len(trimmed) > 100 {
			return ErrInvalidInput
		}
		input.Title = &trimmed
	}
	if input.URL != nil {
		trimmed := strings.TrimSpace(*input.URL)
		if err := validateURL(trimmed, input.Active == nil || *input.Active); err != nil {
			return err
		}
		input.URL = &trimmed
	}
	if input.DisplayOrder != nil && *input.DisplayOrder < 0 {
		return ErrInvalidInput
	}
	return nil
}

func validateURL(value string, required bool) error {
	if value == "" && !required {
		return nil
	}
	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed.Host == "" || parsed.Scheme != "https" && parsed.Scheme != "http" {
		return ErrInvalidInput
	}
	return nil
}
