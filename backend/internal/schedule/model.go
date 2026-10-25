package schedule

import (
	"context"
	"time"

	"github.com/SV1Stail/online_school/internal/user"
)

type BookingStatus string

const (
	StatusPending            BookingStatus = "pending"
	StatusConfirmed          BookingStatus = "confirmed"
	StatusRejected           BookingStatus = "rejected"
	StatusCancelledByStudent BookingStatus = "cancelled_by_student"
	StatusCancelledByStaff   BookingStatus = "cancelled_by_staff"
)

func (s BookingStatus) BlocksTime() bool {
	return s == StatusPending || s == StatusConfirmed
}

type Availability struct {
	ID               string    `json:"id"`
	TeacherID        string    `json:"teacherId"`
	StartsAt         time.Time `json:"startsAt"`
	EndsAt           time.Time `json:"endsAt"`
	AllowedDurations []int16   `json:"allowedDurations"`
	CreatedAt        time.Time `json:"createdAt"`
}

type Booking struct {
	ID             string        `json:"id"`
	AvailabilityID string        `json:"availabilityId"`
	TeacherID      string        `json:"teacherId"`
	StudentID      string        `json:"studentId"`
	StartsAt       time.Time     `json:"startsAt"`
	EndsAt         time.Time     `json:"endsAt"`
	Duration       int16         `json:"durationMinutes"`
	Status         BookingStatus `json:"status"`
	CreatedBy      string        `json:"createdBy"`
	CreatedAt      time.Time     `json:"createdAt"`
	UpdatedAt      time.Time     `json:"updatedAt"`
}

type SlotOption struct {
	AvailabilityID   string    `json:"availabilityId"`
	TeacherID        string    `json:"teacherId"`
	StartsAt         time.Time `json:"startsAt"`
	AllowedDurations []int16   `json:"allowedDurations"`
}

type TimeRange struct {
	StartsAt time.Time
	EndsAt   time.Time
}

type SlotData struct {
	Availability []Availability
	Blocked      []TimeRange
}

type CreateAvailabilityInput struct {
	TeacherID        string    `json:"teacherId"`
	StartsAt         time.Time `json:"startsAt"`
	EndsAt           time.Time `json:"endsAt"`
	AllowedDurations []int16   `json:"allowedDurations"`
}

type CreateBookingInput struct {
	AvailabilityID string    `json:"availabilityId"`
	TeacherID      string    `json:"teacherId"`
	StudentID      string    `json:"studentId"`
	StartsAt       time.Time `json:"startsAt"`
	Duration       int16     `json:"durationMinutes"`
	IdempotencyKey string    `json:"-"`
}

type Repository interface {
	CreateAvailability(ctx context.Context, input CreateAvailabilityInput, createdBy string) (*Availability, error)
	ListAvailability(ctx context.Context, teacherID string, from, to time.Time) ([]Availability, error)
	LoadSlotData(ctx context.Context, teacherID, studentID string, from, to time.Time) (*SlotData, error)
	DeleteAvailability(ctx context.Context, id, teacherID string, moderator bool) error
	CreateBooking(ctx context.Context, input CreateBookingRecord) (*Booking, error)
	FindBooking(ctx context.Context, id string) (*Booking, error)
	ListBookings(ctx context.Context, actor *user.User, from, to time.Time) ([]Booking, error)
	UpdateBookingStatus(ctx context.Context, id string, from []BookingStatus, to BookingStatus) (*Booking, error)
}

type CreateBookingRecord struct {
	CreateBookingInput
	EndsAt    time.Time
	Status    BookingStatus
	CreatedBy string
}

type Auditor interface {
	Write(ctx context.Context, actorID, action, entityType, entityID string, details map[string]any) error
}
