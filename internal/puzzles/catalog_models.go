package puzzles

import "chess-trainer/internal/domain"

type PuzzleKey struct {
	Fingerprint string
	SourceID    string
}

type PuzzleCore struct {
	Fingerprint   string
	DisplayedFEN  string
	Solver        domain.Color
	Solution      []domain.MoveNode
	SolutionPlies int
}

type PuzzleOccurrence struct {
	SourceID    string
	SourceKind  string
	ExternalID  string
	SourceFEN   string
	PreludeUCI  string
	Rating      *int
	Popularity  *int
	PlayCount   *int
	URL         string
	Attribution string
	Metadata    map[string]any
	Themes      []string
	Ordinal     int64
}

type TrainingPuzzle struct {
	Core       PuzzleCore
	Occurrence PuzzleOccurrence
}

func (p TrainingPuzzle) Key() PuzzleKey {
	return PuzzleKey{
		Fingerprint: p.Core.Fingerprint,
		SourceID:    p.Occurrence.SourceID,
	}
}

type SourceSummary struct {
	SourceID             string
	Kind                 string
	MinimumRating        *int
	MaximumRating        *int
	MaximumSolutionPlies int
}

type RatingBounds struct {
	Minimum int
	Maximum int
}

func DefaultLearnerRatingBounds() RatingBounds {
	return RatingBounds{Minimum: 400, Maximum: 3000}
}
