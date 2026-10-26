package conference

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/SV1Stail/online_school/internal/user"
)

type conferenceRepositoryStub struct {
	updateCalls int
	studentID   string
	updateInput UpdateInput
}

func (s *conferenceRepositoryStub) ListForStudent(context.Context, string) ([]Link, error) {
	panic("unexpected call")
}

func (s *conferenceRepositoryStub) ListForBooking(context.Context, string, *user.User) ([]Link, error) {
	panic("unexpected call")
}

func (s *conferenceRepositoryStub) CreateConferenceLink(context.Context, CreateInput) (*Link, error) {
	panic("unexpected call")
}

func (s *conferenceRepositoryStub) UpdateConferenceLink(
	_ context.Context,
	id, studentID string,
	input UpdateInput,
) (*Link, error) {
	s.updateCalls++
	s.studentID = studentID
	s.updateInput = input
	return &Link{ID: id, StudentID: studentID, URL: *input.URL}, nil
}

func (s *conferenceRepositoryStub) DeleteConferenceLink(context.Context, string, string) error {
	panic("unexpected call")
}

type conferenceAuditorStub struct{}

func (conferenceAuditorStub) Write(context.Context, string, string, string, string, map[string]any) error {
	return nil
}

func TestTeacherUpdatesLinksForSelectedStudent(t *testing.T) {
	t.Parallel()

	repository := &conferenceRepositoryStub{}
	service := NewService(repository, conferenceAuditorStub{})
	linkURL := "  https://meet.example.com/student-one  "

	updated, err := service.Update(
		context.Background(),
		&user.User{ID: "teacher-id", Role: user.RoleTeacher},
		"link-id",
		"student-id",
		UpdateInput{URL: &linkURL},
	)

	require.NoError(t, err)
	require.Equal(t, "student-id", updated.StudentID)
	require.Equal(t, "https://meet.example.com/student-one", updated.URL)
	require.Equal(t, 1, repository.updateCalls)
	require.Equal(t, "student-id", repository.studentID)
}

func TestStudentCannotManageConferenceLinks(t *testing.T) {
	t.Parallel()

	repository := &conferenceRepositoryStub{}
	service := NewService(repository, conferenceAuditorStub{})
	linkURL := "https://meet.example.com/student-one"

	_, err := service.Update(
		context.Background(),
		&user.User{ID: "student-id", Role: user.RoleStudent},
		"link-id",
		"student-id",
		UpdateInput{URL: &linkURL},
	)

	require.ErrorIs(t, err, ErrForbidden)
	require.Zero(t, repository.updateCalls)
}
