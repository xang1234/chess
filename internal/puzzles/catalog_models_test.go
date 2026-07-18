package puzzles

import (
	"errors"
	"reflect"
	"strings"
	"testing"

	"chess-trainer/internal/domain"
)

func TestSourceKindMismatchErrorPreservesDetails(t *testing.T) {
	err := error(&SourceKindMismatchError{
		SourceID:      "daily",
		ExistingKind:  "lichess",
		RequestedKind: "csv",
	})

	var mismatch *SourceKindMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("errors.As(%T) failed", err)
	}
	if mismatch.SourceID != "daily" || mismatch.ExistingKind != "lichess" || mismatch.RequestedKind != "csv" {
		t.Fatalf("mismatch details = %+v", mismatch)
	}
	for _, detail := range []string{"daily", "lichess", "csv"} {
		if !strings.Contains(err.Error(), detail) {
			t.Fatalf("error %q does not contain %q", err, detail)
		}
	}
}

func TestTrainingPuzzleKeepsSourceOccurrencesIndependent(t *testing.T) {
	core := PuzzleCore{
		Fingerprint:   "stable-core",
		DisplayedFEN:  "7k/5Q2/6K1/8/8/8/8/8 w - - 0 1",
		Solver:        domain.White,
		Solution:      []domain.MoveNode{{UCI: "f7f8"}},
		SolutionPlies: 1,
	}
	firstRating, firstPopularity, firstPlayCount := 1200, 80, 100
	secondRating, secondPopularity, secondPlayCount := 1800, 95, 900
	first := TrainingPuzzle{
		Core: core,
		Occurrence: PuzzleOccurrence{
			SourceID:    "community",
			SourceKind:  "csv",
			ExternalID:  "community-1",
			SourceFEN:   "first source fen",
			PreludeUCI:  "e2e4",
			Rating:      &firstRating,
			Popularity:  &firstPopularity,
			PlayCount:   &firstPlayCount,
			URL:         "https://example.test/community-1",
			Attribution: "Community author",
			Metadata:    map[string]any{"collection": "one"},
			Themes:      []string{"fork"},
			Ordinal:     10,
		},
	}
	second := TrainingPuzzle{
		Core: core,
		Occurrence: PuzzleOccurrence{
			SourceID:    "lichess",
			SourceKind:  "lichess",
			ExternalID:  "lichess-9",
			SourceFEN:   "second source fen",
			PreludeUCI:  "d2d4",
			Rating:      &secondRating,
			Popularity:  &secondPopularity,
			PlayCount:   &secondPlayCount,
			URL:         "https://example.test/lichess-9",
			Attribution: "Lichess",
			Metadata:    map[string]any{"collection": "two"},
			Themes:      []string{"pin"},
			Ordinal:     20,
		},
	}

	if !reflect.DeepEqual(first.Core, second.Core) {
		t.Fatalf("cores differ: first=%+v second=%+v", first.Core, second.Core)
	}
	if reflect.DeepEqual(first.Occurrence, second.Occurrence) {
		t.Fatalf("occurrences unexpectedly match: %+v", first.Occurrence)
	}
	if got, want := first.Key(), (PuzzleKey{Fingerprint: "stable-core", SourceID: "community"}); got != want {
		t.Fatalf("first.Key() = %+v, want %+v", got, want)
	}
	if got, want := second.Key(), (PuzzleKey{Fingerprint: "stable-core", SourceID: "lichess"}); got != want {
		t.Fatalf("second.Key() = %+v, want %+v", got, want)
	}
}

func TestLearnerRatingBoundsIncludeEveryRatedSource(t *testing.T) {
	jsonMin, jsonMax := 900, 2100
	lichessMin, lichessMax := 1100, 1800
	got := LearnerRatingBoundsFromSourceSummaries([]SourceSummary{
		{Kind: "canonical-json", MinimumRating: &jsonMin, MaximumRating: &jsonMax},
		{Kind: "lichess", MinimumRating: &lichessMin, MaximumRating: &lichessMax},
		{Kind: "tactical-pgn"},
	})
	if got != (RatingBounds{Minimum: 900, Maximum: 2100}) {
		t.Fatalf("bounds = %+v", got)
	}
}
