package training

import "time"

type Outcome string

const (
	OutcomeClean    Outcome = "clean"
	OutcomeMissed   Outcome = "missed"
	OutcomeHinted   Outcome = "hinted"
	OutcomeRevealed Outcome = "revealed"
)

var reviewIntervals = []time.Duration{
	24 * time.Hour,
	72 * time.Hour,
	7 * 24 * time.Hour,
	21 * 24 * time.Hour,
	60 * 24 * time.Hour,
}

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
	switch outcome {
	case OutcomeMissed, OutcomeHinted, OutcomeRevealed:
		current.IntervalIndex = 0
		current.SuccessfulReviews = 0
	case OutcomeClean:
		current.IntervalIndex = min(current.IntervalIndex+1, len(reviewIntervals)-1)
		current.SuccessfulReviews++
	default:
		panic("unknown review outcome")
	}
	current.DueAt = now.Add(reviewIntervals[current.IntervalIndex])
	return current
}
