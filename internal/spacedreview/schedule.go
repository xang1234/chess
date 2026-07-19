package spacedreview

import "time"

type Outcome string

const (
	Clean    Outcome = "clean"
	Missed   Outcome = "missed"
	Hinted   Outcome = "hinted"
	Revealed Outcome = "revealed"
)

var intervals = [...]time.Duration{
	24 * time.Hour,
	72 * time.Hour,
	7 * 24 * time.Hour,
	21 * 24 * time.Hour,
	60 * 24 * time.Hour,
}

type State struct {
	IntervalIndex     int
	SuccessfulReviews int
}

type Scheduled struct {
	State State
	DueAt time.Time
}

func Next(now time.Time, current State, outcome Outcome) Scheduled {
	if current.IntervalIndex < -1 || current.IntervalIndex >= len(intervals) {
		panic("invalid review interval index")
	}
	if current.SuccessfulReviews < 0 {
		panic("invalid successful review count")
	}
	switch outcome {
	case Missed, Hinted, Revealed:
		current.IntervalIndex = 0
		current.SuccessfulReviews = 0
	case Clean:
		current.IntervalIndex = min(current.IntervalIndex+1, len(intervals)-1)
		current.SuccessfulReviews++
	default:
		panic("unknown review outcome")
	}
	return Scheduled{
		State: current,
		DueAt: now.Add(intervals[current.IntervalIndex]),
	}
}
