package training

import (
	"testing"
	"time"
)

func TestNextReviewIntervalsAndReset(t *testing.T) {
	now := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	state := NextReview(now, ReviewState{}, OutcomeMissed)
	if got := state.DueAt.Sub(now); got != 24*time.Hour {
		t.Fatalf("miss due in %v", got)
	}

	for _, days := range []int{3, 7, 21, 60} {
		now = state.DueAt
		state = NextReview(now, state, OutcomeClean)
		if got := state.DueAt.Sub(now); got != time.Duration(days)*24*time.Hour {
			t.Fatalf("clean review due in %v, want %d days", got, days)
		}
	}

	now = state.DueAt
	state = NextReview(now, state, OutcomeMissed)
	if got := state.DueAt.Sub(now); got != 24*time.Hour || state.IntervalIndex != 0 {
		t.Fatalf("reset state=%+v", state)
	}
}

func TestHintedAndRevealedReviewsReturnTomorrow(t *testing.T) {
	now := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	for _, outcome := range []Outcome{OutcomeHinted, OutcomeRevealed} {
		state := NextReview(now, ReviewState{IntervalIndex: 4}, outcome)
		if got := state.DueAt.Sub(now); got != 24*time.Hour || state.IntervalIndex != 0 {
			t.Fatalf("outcome=%s state=%+v", outcome, state)
		}
	}
}
