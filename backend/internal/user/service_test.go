package user

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type updateRepositoryStub struct {
	updateCalls int
	input       UpdateInput
}

func (s *updateRepositoryStub) FindByLogin(context.Context, string) (*Credentials, error) {
	panic("unexpected call")
}

func (s *updateRepositoryStub) FindByID(context.Context, string) (*User, error) {
	panic("unexpected call")
}

func (s *updateRepositoryStub) List(context.Context) ([]User, error) {
	panic("unexpected call")
}

func (s *updateRepositoryStub) ListTeachers(context.Context) ([]User, error) {
	panic("unexpected call")
}

func (s *updateRepositoryStub) ListStudents(context.Context) ([]User, error) {
	panic("unexpected call")
}

func (s *updateRepositoryStub) CreateUser(context.Context, CreateRecord) (*User, error) {
	panic("unexpected call")
}

func (s *updateRepositoryStub) UpdateUser(_ context.Context, id string, input UpdateInput) (*User, error) {
	s.updateCalls++
	s.input = input
	return &User{ID: id, SubjectLabel: *input.SubjectLabel}, nil
}

func (s *updateRepositoryStub) UpdatePassword(context.Context, string, string, bool) error {
	panic("unexpected call")
}

type auditorStub struct{}

func (auditorStub) Write(context.Context, string, string, string, string, map[string]any) error {
	return nil
}

func TestUpdateSubjectLabel(t *testing.T) {
	t.Parallel()

	t.Run("trims subject label", func(t *testing.T) {
		t.Parallel()

		repository := &updateRepositoryStub{}
		service := NewService(repository, nil, auditorStub{})
		subjectLabel := "  Математика · физика  "

		updated, err := service.Update(
			context.Background(),
			"teacher-id",
			"teacher-id",
			UpdateInput{SubjectLabel: &subjectLabel},
		)

		require.NoError(t, err)
		require.Equal(t, "Математика · физика", updated.SubjectLabel)
		require.Equal(t, 1, repository.updateCalls)
	})

	t.Run("rejects more than 100 unicode characters", func(t *testing.T) {
		t.Parallel()

		repository := &updateRepositoryStub{}
		service := NewService(repository, nil, auditorStub{})
		subjectLabel := strings.Repeat("я", 101)

		_, err := service.Update(
			context.Background(),
			"teacher-id",
			"teacher-id",
			UpdateInput{SubjectLabel: &subjectLabel},
		)

		require.ErrorIs(t, err, ErrInvalidInput)
		require.Zero(t, repository.updateCalls)
	})
}
