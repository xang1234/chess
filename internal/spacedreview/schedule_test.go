package spacedreview

import (
	"testing"
	"time"
)

func TestNextAdvancesNewAndExistingCleanReviews(t *testing.T) {
	now := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	scheduled := Next(now, State{IntervalIndex: -1}, Clean)
	if scheduled.State.IntervalIndex != 0 || scheduled.State.SuccessfulReviews != 1 ||
		scheduled.DueAt.Sub(now) != 24*time.Hour {
		t.Fatalf("new clean schedule = %+v", scheduled)
	}

	for index, days := range []int{3, 7, 21, 60} {
		now = scheduled.DueAt
		scheduled = Next(now, scheduled.State, Clean)
		if scheduled.State.IntervalIndex != index+1 ||
			scheduled.DueAt.Sub(now) != time.Duration(days)*24*time.Hour {
			t.Fatalf("clean schedule %d = %+v, want %d days", index, scheduled, days)
		}
	}
	now = scheduled.DueAt
	scheduled = Next(now, scheduled.State, Clean)
	if scheduled.State.IntervalIndex != 4 || scheduled.State.SuccessfulReviews != 6 ||
		scheduled.DueAt.Sub(now) != 60*24*time.Hour {
		t.Fatalf("maximum clean schedule = %+v", scheduled)
	}
}

func TestNextResetsNonCleanReviews(t *testing.T) {
	now := time.Date(2026, time.July, 19, 12, 0, 0, 0, time.UTC)
	for _, outcome := range []Outcome{Missed, Hinted, Revealed} {
		scheduled := Next(now, State{IntervalIndex: 4, SuccessfulReviews: 8}, outcome)
		if scheduled.State != (State{IntervalIndex: 0}) ||
			scheduled.DueAt.Sub(now) != 24*time.Hour {
			t.Fatalf("outcome %q schedule = %+v", outcome, scheduled)
		}
	}
}

func TestNextPanicsForUnknownOutcome(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Next() did not panic")
		}
	}()
	Next(time.Now(), State{}, Outcome("future"))
}
