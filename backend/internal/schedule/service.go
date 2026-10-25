package schedule

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/SV1Stail/online_school/internal/user"
)

const (
	bookingCutoff      = 12 * time.Hour
	cancellationCutoff = 24 * time.Hour
	startStep          = 10 * time.Minute
)

var (
	ErrNotFound           = errors.New("schedule entity not found")
	ErrConflict           = errors.New("schedule conflict")
	ErrInvalidInput       = errors.New("invalid schedule input")
	ErrForbidden          = errors.New("schedule action is forbidden")
	ErrBookingClosed      = errors.New("booking is closed")
	ErrCancellationClosed = errors.New("cancellation is closed")
	ErrInvalidTransition  = errors.New("invalid booking status transition")
)

type Service struct {
	repository Repository
	auditor    Auditor
	now        func() time.Time
}

func NewService(repository Repository, auditor Auditor) *Service {
	return &Service{repository: repository, auditor: auditor, now: time.Now}
}

func (s *Service) CreateAvailability(
	ctx context.Context,
	actor *user.User,
	input CreateAvailabilityInput,
) (*Availability, error) {
	if actor.Role == user.RoleTeacher {
		input.TeacherID = actor.ID
	} else if actor.Role != user.RoleModerator {
		return nil, ErrForbidden
	}
	if err := validateAvailability(input); err != nil {
		return nil, err
	}
	created, err := s.repository.CreateAvailability(ctx, input, actor.ID)
	if err != nil {
		return nil, err
	}
	if err := s.auditor.Write(ctx, actor.ID, "availability.created", "availability", created.ID, nil); err != nil {
		return nil, err
	}
	return created, nil
}

func (s *Service) ListAvailability(
	ctx context.Context,
	teacherID string,
	from, to time.Time,
) ([]Availability, error) {
	if teacherID == "" || !to.After(from) || to.Sub(from) > 93*24*time.Hour {
		return nil, ErrInvalidInput
	}
	return s.repository.ListAvailability(ctx, teacherID, from.UTC(), to.UTC())
}

func (s *Service) ListAvailableSlots(
	ctx context.Context,
	actor *user.User,
	teacherID string,
	from, to time.Time,
) ([]SlotOption, error) {
	if teacherID == "" || !to.After(from) || to.Sub(from) > 93*24*time.Hour {
		return nil, ErrInvalidInput
	}
	studentID := ""
	if actor.Role == user.RoleStudent {
		studentID = actor.ID
	}
	data, err := s.repository.LoadSlotData(ctx, teacherID, studentID, from.UTC(), to.UTC())
	if err != nil {
		return nil, err
	}
	cutoff := from.UTC()
	if actor.Role == user.RoleStudent {
		minimum := s.now().UTC().Add(bookingCutoff)
		if minimum.After(cutoff) {
			cutoff = minimum
		}
	}
	return buildSlotOptions(data, cutoff, to.UTC()), nil
}

func (s *Service) DeleteAvailability(ctx context.Context, actor *user.User, id string) error {
	if actor.Role != user.RoleTeacher && actor.Role != user.RoleModerator {
		return ErrForbidden
	}
	if err := s.repository.DeleteAvailability(ctx, id, actor.ID, actor.Role == user.RoleModerator); err != nil {
		return err
	}
	return s.auditor.Write(ctx, actor.ID, "availability.deleted", "availability", id, nil)
}

func (s *Service) CreateBooking(
	ctx context.Context,
	actor *user.User,
	input CreateBookingInput,
) (*Booking, error) {
	if !validDuration(input.Duration) || input.IdempotencyKey == "" || len(input.IdempotencyKey) > 100 {
		return nil, ErrInvalidInput
	}
	if input.StartsAt.Second() != 0 || input.StartsAt.Nanosecond() != 0 {
		return nil, ErrInvalidInput
	}

	status := StatusConfirmed
	switch actor.Role {
	case user.RoleStudent:
		input.StudentID = actor.ID
		status = StatusPending
		if input.StartsAt.Before(s.now().UTC().Add(bookingCutoff)) {
			return nil, ErrBookingClosed
		}
	case user.RoleTeacher:
		input.TeacherID = actor.ID
	case user.RoleModerator:
	default:
		return nil, ErrForbidden
	}
	if input.TeacherID == "" || input.StudentID == "" || input.AvailabilityID == "" {
		return nil, ErrInvalidInput
	}
	if !input.StartsAt.After(s.now().UTC()) {
		return nil, ErrInvalidInput
	}

	endsAt := input.StartsAt.UTC().Add(time.Duration(input.Duration) * time.Minute)
	created, err := s.repository.CreateBooking(ctx, CreateBookingRecord{
		CreateBookingInput: input,
		EndsAt:             endsAt,
		Status:             status,
		CreatedBy:          actor.ID,
	})
	if err != nil {
		return nil, err
	}
	if err := s.auditor.Write(ctx, actor.ID, "booking.created", "booking", created.ID, map[string]any{
		"status": status,
	}); err != nil {
		return nil, err
	}
	return created, nil
}

func (s *Service) ListBookings(
	ctx context.Context,
	actor *user.User,
	from, to time.Time,
) ([]Booking, error) {
	if !to.After(from) || to.Sub(from) > 366*24*time.Hour {
		return nil, ErrInvalidInput
	}
	return s.repository.ListBookings(ctx, actor, from.UTC(), to.UTC())
}

func (s *Service) Confirm(ctx context.Context, actor *user.User, id string) (*Booking, error) {
	booking, err := s.authorizeStaffAction(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	updated, err := s.repository.UpdateBookingStatus(ctx, booking.ID, []BookingStatus{StatusPending}, StatusConfirmed)
	if err != nil {
		return nil, err
	}
	if err := s.auditor.Write(ctx, actor.ID, "booking.confirmed", "booking", id, nil); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) Reject(ctx context.Context, actor *user.User, id string) (*Booking, error) {
	booking, err := s.authorizeStaffAction(ctx, actor, id)
	if err != nil {
		return nil, err
	}
	updated, err := s.repository.UpdateBookingStatus(ctx, booking.ID, []BookingStatus{StatusPending}, StatusRejected)
	if err != nil {
		return nil, err
	}
	if err := s.auditor.Write(ctx, actor.ID, "booking.rejected", "booking", id, nil); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) Cancel(ctx context.Context, actor *user.User, id string) (*Booking, error) {
	booking, err := s.repository.FindBooking(ctx, id)
	if err != nil {
		return nil, err
	}
	var target BookingStatus
	var allowed []BookingStatus
	switch actor.Role {
	case user.RoleStudent:
		if booking.StudentID != actor.ID {
			return nil, ErrForbidden
		}
		if booking.Status == StatusConfirmed && booking.StartsAt.Before(s.now().UTC().Add(cancellationCutoff)) {
			return nil, ErrCancellationClosed
		}
		target = StatusCancelledByStudent
		allowed = []BookingStatus{StatusPending, StatusConfirmed}
	case user.RoleTeacher:
		if booking.TeacherID != actor.ID {
			return nil, ErrForbidden
		}
		target = StatusCancelledByStaff
		allowed = []BookingStatus{StatusPending, StatusConfirmed}
	case user.RoleModerator:
		target = StatusCancelledByStaff
		allowed = []BookingStatus{StatusPending, StatusConfirmed}
	default:
		return nil, ErrForbidden
	}
	updated, err := s.repository.UpdateBookingStatus(ctx, id, allowed, target)
	if err != nil {
		return nil, err
	}
	if err := s.auditor.Write(ctx, actor.ID, "booking.cancelled", "booking", id, map[string]any{
		"status": target,
	}); err != nil {
		return nil, err
	}
	return updated, nil
}

func (s *Service) authorizeStaffAction(ctx context.Context, actor *user.User, id string) (*Booking, error) {
	booking, err := s.repository.FindBooking(ctx, id)
	if err != nil {
		return nil, err
	}
	if actor.Role == user.RoleModerator || actor.Role == user.RoleTeacher && booking.TeacherID == actor.ID {
		return booking, nil
	}
	return nil, ErrForbidden
}

func validateAvailability(input CreateAvailabilityInput) error {
	if input.TeacherID == "" || !input.EndsAt.After(input.StartsAt) || input.EndsAt.Sub(input.StartsAt) > 24*time.Hour {
		return ErrInvalidInput
	}
	if input.StartsAt.Second() != 0 || input.StartsAt.Nanosecond() != 0 || input.EndsAt.Second() != 0 || input.EndsAt.Nanosecond() != 0 {
		return ErrInvalidInput
	}
	if len(input.AllowedDurations) == 0 || len(input.AllowedDurations) > 4 {
		return ErrInvalidInput
	}
	sort.Slice(input.AllowedDurations, func(i, j int) bool { return input.AllowedDurations[i] < input.AllowedDurations[j] })
	for index, duration := range input.AllowedDurations {
		if !validDuration(duration) || index > 0 && duration == input.AllowedDurations[index-1] {
			return fmt.Errorf("%w: invalid duration", ErrInvalidInput)
		}
	}
	return nil
}

func validDuration(duration int16) bool {
	return duration == 15 || duration == 30 || duration == 45 || duration == 60
}

func IsAlignedWithStep(availabilityStart, bookingStart time.Time) bool {
	return bookingStart.Sub(availabilityStart)%startStep == 0
}

func buildSlotOptions(data *SlotData, from, to time.Time) []SlotOption {
	result := make([]SlotOption, 0)
	for _, availability := range data.Availability {
		start := availability.StartsAt
		if start.Before(from) {
			delta := from.Sub(start)
			steps := (delta + startStep - 1) / startStep
			start = start.Add(steps * startStep)
		}
		limit := availability.EndsAt
		if to.Before(limit) {
			limit = to
		}
		for candidate := start; candidate.Before(limit); candidate = candidate.Add(startStep) {
			durations := availableDurations(candidate, limit, availability.AllowedDurations, data.Blocked)
			if len(durations) == 0 {
				continue
			}
			result = append(result, SlotOption{
				AvailabilityID:   availability.ID,
				TeacherID:        availability.TeacherID,
				StartsAt:         candidate,
				AllowedDurations: durations,
			})
		}
	}
	return result
}

func availableDurations(start, limit time.Time, durations []int16, blocked []TimeRange) []int16 {
	result := make([]int16, 0, len(durations))
	for _, duration := range durations {
		end := start.Add(time.Duration(duration) * time.Minute)
		if end.After(limit) || overlapsAny(start, end, blocked) {
			continue
		}
		result = append(result, duration)
	}
	return result
}

func overlapsAny(start, end time.Time, blocked []TimeRange) bool {
	for _, item := range blocked {
		if start.Before(item.EndsAt) && end.After(item.StartsAt) {
			return true
		}
	}
	return false
}
