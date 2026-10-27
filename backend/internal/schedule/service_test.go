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

func TestPlanAvailabilityCreationSplitsAroundMultipleOverlaps(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.October, 27, 15, 0, 0, 0, time.UTC)
	input := CreateAvailabilityInput{
		TeacherID:        "teacher-id",
		StartsAt:         start,
		EndsAt:           start.Add(6 * time.Hour),
		AllowedDurations: []int16{30, 60},
	}
	plan := PlanAvailabilityCreation(input, []TimeRange{
		{StartsAt: start.Add(time.Hour), EndsAt: start.Add(2 * time.Hour)},
		{StartsAt: start.Add(3 * time.Hour), EndsAt: start.Add(4*time.Hour + 30*time.Minute)},
		{StartsAt: start.Add(3*time.Hour + 30*time.Minute), EndsAt: start.Add(4 * time.Hour)},
	})

	require.Len(t, plan.Fragments, 3)
	require.Equal(t, start, plan.Fragments[0].StartsAt)
	require.Equal(t, start.Add(time.Hour), plan.Fragments[0].EndsAt)
	require.Equal(t, start.Add(2*time.Hour), plan.Fragments[1].StartsAt)
	require.Equal(t, start.Add(3*time.Hour), plan.Fragments[1].EndsAt)
	require.Equal(t, start.Add(4*time.Hour+30*time.Minute), plan.Fragments[2].StartsAt)
	require.Equal(t, start.Add(6*time.Hour), plan.Fragments[2].EndsAt)
	require.Len(t, plan.SkippedRanges, 2)
}

func TestPlanAvailabilityCreationFiltersDurationsForShortFragment(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.October, 27, 15, 0, 0, 0, time.UTC)
	input := CreateAvailabilityInput{
		TeacherID:        "teacher-id",
		StartsAt:         start,
		EndsAt:           start.Add(time.Hour),
		AllowedDurations: []int16{30, 60},
	}
	plan := PlanAvailabilityCreation(input, []TimeRange{{
		StartsAt: start.Add(45 * time.Minute),
		EndsAt:   start.Add(time.Hour),
	}})

	require.Len(t, plan.Fragments, 1)
	require.Equal(t, []int16{30}, plan.Fragments[0].AllowedDurations)
	require.Len(t, plan.SkippedRanges, 1)
	require.Equal(t, AvailabilitySkipOverlap, plan.SkippedRanges[0].Reason)
}

func TestPlanAvailabilityCreationSkipsRemaindersShorterThanAllDurations(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.October, 27, 15, 0, 0, 0, time.UTC)
	input := CreateAvailabilityInput{
		TeacherID:        "teacher-id",
		StartsAt:         start,
		EndsAt:           start.Add(time.Hour),
		AllowedDurations: []int16{30, 60},
	}
	plan := PlanAvailabilityCreation(input, []TimeRange{{
		StartsAt: start.Add(20 * time.Minute),
		EndsAt:   start.Add(40 * time.Minute),
	}})

	require.Empty(t, plan.Fragments)
	require.Len(t, plan.SkippedRanges, 3)
	require.Equal(t, AvailabilitySkipTooShort, plan.SkippedRanges[0].Reason)
	require.Equal(t, AvailabilitySkipOverlap, plan.SkippedRanges[1].Reason)
	require.Equal(t, AvailabilitySkipTooShort, plan.SkippedRanges[2].Reason)
}

func TestPlanAvailabilityCreationDoesNotDuplicateCoveredRange(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, time.October, 27, 15, 0, 0, 0, time.UTC)
	input := CreateAvailabilityInput{
		TeacherID:        "teacher-id",
		StartsAt:         start,
		EndsAt:           start.Add(time.Hour),
		AllowedDurations: []int16{30, 60},
	}
	plan := PlanAvailabilityCreation(input, []TimeRange{{
		StartsAt: start.Add(-time.Hour),
		EndsAt:   start.Add(2 * time.Hour),
	}})

	require.Empty(t, plan.Fragments)
	require.Equal(t, []SkippedAvailabilityRange{{
		StartsAt: start,
		EndsAt:   start.Add(time.Hour),
		Reason:   AvailabilitySkipOverlap,
	}}, plan.SkippedRanges)
}
