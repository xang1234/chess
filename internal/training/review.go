package training

import (
	"time"

	"chess-trainer/internal/spacedreview"
)

type Outcome string

const (
	OutcomeClean    Outcome = "clean"
	OutcomeMissed   Outcome = "missed"
	OutcomeHinted   Outcome = "hinted"
	OutcomeRevealed Outcome = "revealed"
)

type ReviewState struct {
	Fingerprint       string    `json:"fingerprint"`
	PreferredSourceID string    `json:"preferredSourceId"`
	DueAt             time.Time `json:"dueAt"`
	IntervalIndex     int       `json:"intervalIndex"`
	SuccessfulReviews int       `json:"successfulReviews"`
	LastOutcome       Outcome   `json:"lastOutcome"`
}

func NextReview(now time.Time, current ReviewState, outcome Outcome) ReviewState {
	current.LastOutcome = outcome
	scheduled := spacedreview.Next(now, spacedreview.State{
		IntervalIndex:     current.IntervalIndex,
		SuccessfulReviews: current.SuccessfulReviews,
	}, spacedReviewOutcome(outcome))
	current.IntervalIndex = scheduled.State.IntervalIndex
	current.SuccessfulReviews = scheduled.State.SuccessfulReviews
	current.DueAt = scheduled.DueAt
	return current
}

func spacedReviewOutcome(outcome Outcome) spacedreview.Outcome {
	switch outcome {
	case OutcomeClean:
		return spacedreview.Clean
	case OutcomeMissed:
		return spacedreview.Missed
	case OutcomeHinted:
		return spacedreview.Hinted
	case OutcomeRevealed:
		return spacedreview.Revealed
	default:
		panic("unknown review outcome")
	}
}
