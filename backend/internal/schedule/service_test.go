package schedule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBuildSlotOptionsFiltersOverlapsAndUsesTenMinuteStep(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.October, 26, 15, 0, 0, 0, time.UTC)
	data := &SlotData{
		Availability: []Availability{{
			ID:               "availability-id",
			TeacherID:        "teacher-id",
			StartsAt:         start,
			EndsAt:           start.Add(2 * time.Hour),
			AllowedDurations: []int16{30, 60},
		}},
		Blocked: []TimeRange{{
			StartsAt: start.Add(30 * time.Minute),
			EndsAt:   start.Add(60 * time.Minute),
		}},
	}

	options := buildSlotOptions(data, start, start.Add(2*time.Hour))
	require.NotEmpty(t, options)
	require.Equal(t, start, options[0].StartsAt)
	require.Equal(t, []int16{30}, options[0].AllowedDurations)

	for _, option := range options {
		require.Zero(t, option.StartsAt.Sub(start)%(10*time.Minute))
		for _, duration := range option.AllowedDurations {
			end := option.StartsAt.Add(time.Duration(duration) * time.Minute)
			require.False(t, option.StartsAt.Before(start.Add(time.Hour)) && end.After(start.Add(30*time.Minute)))
		}
	}
}

func TestValidateAvailabilityRejectsDuplicateDuration(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.October, 26, 15, 0, 0, 0, time.UTC)
	err := validateAvailability(CreateAvailabilityInput{
		TeacherID:        "teacher-id",
		StartsAt:         start,
		EndsAt:           start.Add(time.Hour),
		AllowedDurations: []int16{30, 30},
	})
	require.ErrorIs(t, err, ErrInvalidInput)
}
