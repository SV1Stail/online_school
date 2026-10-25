package conference

import (
	"context"
	"time"

	"github.com/SV1Stail/online_school/internal/user"
)

type Provider string

const (
	ProviderTeams      Provider = "teams"
	ProviderZoom       Provider = "zoom"
	ProviderTelemost   Provider = "telemost"
	ProviderGoogleMeet Provider = "google_meet"
	ProviderSberJazz   Provider = "sber_jazz"
	ProviderCustom     Provider = "custom"
)

func (p Provider) Valid() bool {
	switch p {
	case ProviderTeams, ProviderZoom, ProviderTelemost, ProviderGoogleMeet, ProviderSberJazz, ProviderCustom:
		return true
	default:
		return false
	}
}

type Link struct {
	ID           string    `json:"id"`
	TeacherID    string    `json:"teacherId"`
	Provider     Provider  `json:"provider"`
	Title        string    `json:"title"`
	URL          string    `json:"url"`
	Active       bool      `json:"active"`
	DisplayOrder int       `json:"displayOrder"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type CreateInput struct {
	TeacherID    string   `json:"teacherId"`
	Provider     Provider `json:"provider"`
	Title        string   `json:"title"`
	URL          string   `json:"url"`
	Active       bool     `json:"active"`
	DisplayOrder int      `json:"displayOrder"`
}

type UpdateInput struct {
	Title        *string `json:"title"`
	URL          *string `json:"url"`
	Active       *bool   `json:"active"`
	DisplayOrder *int    `json:"displayOrder"`
}

type Repository interface {
	ListForTeacher(ctx context.Context, teacherID string, includeInactive bool) ([]Link, error)
	ListForBooking(ctx context.Context, bookingID string, actor *user.User) ([]Link, error)
	CreateConferenceLink(ctx context.Context, input CreateInput) (*Link, error)
	UpdateConferenceLink(ctx context.Context, id, teacherID string, moderator bool, input UpdateInput) (*Link, error)
	DeleteConferenceLink(ctx context.Context, id, teacherID string, moderator bool) error
}

type Auditor interface {
	Write(ctx context.Context, actorID, action, entityType, entityID string, details map[string]any) error
}
