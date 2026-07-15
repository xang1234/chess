package training

import (
	"context"
	"database/sql"
	"math/rand"
	"slices"
	"sort"
	"testing"
	"time"

	"chess-trainer/internal/domain"
	"chess-trainer/internal/puzzles"
)

type candidateCall struct {
	minimum  int
	maximum  int
	excluded []string
	limit    int
}

type schedulerCatalogFake struct {
	puzzles map[puzzles.PuzzleKey]puzzles.TrainingPuzzle
	calls   []candidateCall
}

func (f *schedulerCatalogFake) Get(_ context.Context, key puzzles.PuzzleKey) (puzzles.TrainingPuzzle, error) {
	puzzle, ok := f.puzzles[key]
	if !ok {
		return puzzles.TrainingPuzzle{}, sql.ErrNoRows
	}
	return puzzle, nil
}

func (f *schedulerCatalogFake) Resolve(
	ctx context.Context,
	fingerprint string,
	preferredSourceID string,
) (puzzles.TrainingPuzzle, error) {
	if preferredSourceID != "" {
		if puzzle, err := f.Get(ctx, puzzles.PuzzleKey{
			Fingerprint: fingerprint,
			SourceID:    preferredSourceID,
		}); err == nil {
			return puzzle, nil
		}
	}
	sourceIDs := make([]string, 0)
	for key := range f.puzzles {
		if key.Fingerprint == fingerprint {
			sourceIDs = append(sourceIDs, key.SourceID)
		}
	}
	if len(sourceIDs) == 0 {
		return puzzles.TrainingPuzzle{}, sql.ErrNoRows
	}
	sort.Strings(sourceIDs)
	return f.Get(ctx, puzzles.PuzzleKey{Fingerprint: fingerprint, SourceID: sourceIDs[0]})
}

func (f *schedulerCatalogFake) RatedCandidates(
	_ context.Context,
	minimum int,
	maximum int,
	excluded []string,
	limit int,
) ([]puzzles.TrainingPuzzle, error) {
	f.calls = append(f.calls, candidateCall{
		minimum: minimum, maximum: maximum, excluded: slices.Clone(excluded), limit: limit,
	})
	var count int
	switch maximum - 1500 {
	case 100:
		count = 2
	case 200:
		count = 4
	default:
		count = 6
	}
	result := make([]puzzles.TrainingPuzzle, 0, count)
	for index := 1; index <= count; index++ {
		fingerprint := "new" + string(rune('0'+index))
		if !slices.Contains(excluded, fingerprint) {
			result = append(result, f.puzzles[puzzles.PuzzleKey{
				Fingerprint: fingerprint,
				SourceID:    "lichess",
			}])
		}
	}
	return result, nil
}

func (*schedulerCatalogFake) FreePracticeCandidates(
	context.Context,
	string,
	*int,
	*int,
	[]string,
	*int,
	int,
) ([]puzzles.TrainingPuzzle, error) {
	return nil, nil
}

type schedulerUserFake struct {
	due         []ReviewState
	recent      []string
	reviewLimit int
}

func (f *schedulerUserFake) DueReviews(
	_ context.Context,
	_ time.Time,
	after *ReviewCursor,
	limit int,
) ([]ReviewState, error) {
	f.reviewLimit = limit
	due := slices.Clone(f.due)
	sort.SliceStable(due, func(i, j int) bool {
		return reviewCursorLess(reviewCursor(due[i]), reviewCursor(due[j]))
	})
	start := 0
	if after != nil {
		start = len(due)
		for index, review := range due {
			if reviewCursorLess(*after, reviewCursor(review)) {
				start = index
				break
			}
		}
	}
	return slices.Clone(due[start:min(start+limit, len(due))]), nil
}

func (f *schedulerUserFake) RecentFingerprints(context.Context, int) ([]string, error) {
	return slices.Clone(f.recent), nil
}

func scheduledTestPuzzle(fingerprint string, rating int) puzzles.TrainingPuzzle {
	return puzzles.TrainingPuzzle{
		Core: puzzles.PuzzleCore{
			Fingerprint:   fingerprint,
			DisplayedFEN:  "7k/5Q2/6K1/8/8/8/8/8 w - - 0 1",
			Solver:        domain.White,
			Solution:      []domain.MoveNode{{UCI: "f7f8"}},
			SolutionPlies: 1,
		},
		Occurrence: puzzles.PuzzleOccurrence{
			SourceID:   "lichess",
			SourceKind: "lichess",
			Rating:     &rating,
		},
	}
}

func TestSchedulerBuildsFourReviewsThenWidensForNewPuzzles(t *testing.T) {
	catalogPuzzles := make(map[puzzles.PuzzleKey]puzzles.TrainingPuzzle)
	due := make([]ReviewState, 0, 6)
	for index := 1; index <= 6; index++ {
		fingerprint := "review" + string(rune('0'+index))
		puzzle := scheduledTestPuzzle(fingerprint, 1400+index)
		catalogPuzzles[puzzle.Key()] = puzzle
		due = append(due, ReviewState{
			Fingerprint: fingerprint,
			DueAt:       time.Unix(int64(index), 0),
		})
	}
	for index := 1; index <= 6; index++ {
		fingerprint := "new" + string(rune('0'+index))
		puzzle := scheduledTestPuzzle(fingerprint, 1500)
		catalogPuzzles[puzzle.Key()] = puzzle
	}
	catalog := &schedulerCatalogFake{puzzles: catalogPuzzles}
	user := &schedulerUserFake{due: due, recent: []string{"recent"}}
	scheduler := Scheduler{Catalog: catalog, User: user}

	items, err := scheduler.BuildGuided(
		context.Background(),
		Profile{LearnerRating: 1500, SessionSize: 10},
		time.Unix(100, 0),
		rand.New(rand.NewSource(7)),
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 10 {
		t.Fatalf("items=%d", len(items))
	}
	if user.reviewLimit != 4 {
		t.Fatalf("DueReviews limit=%d", user.reviewLimit)
	}
	for index, item := range items[:4] {
		if item.Kind != ScheduledReview || item.UpdatesRating {
			t.Fatalf("item %d=%+v", index, item)
		}
	}
	for index, item := range items[4:] {
		if item.Kind != ScheduledNew || !item.UpdatesRating {
			t.Fatalf("new item %d=%+v", index, item)
		}
	}
	wantBands := [][2]int{{1400, 1600}, {1300, 1700}, {1200, 1800}}
	if len(catalog.calls) != len(wantBands) {
		t.Fatalf("calls=%+v", catalog.calls)
	}
	for index, band := range wantBands {
		call := catalog.calls[index]
		if call.minimum != band[0] || call.maximum != band[1] {
			t.Fatalf("call %d=%+v", index, call)
		}
	}
	if !slices.Contains(catalog.calls[0].excluded, "recent") ||
		!slices.Contains(catalog.calls[0].excluded, "review1") {
		t.Fatalf("initial exclusions=%v", catalog.calls[0].excluded)
	}

	firstOrder := scheduledFingerprints(items)
	catalog.calls = nil
	repeated, err := scheduler.BuildGuided(
		context.Background(),
		Profile{LearnerRating: 1500, SessionSize: 10},
		time.Unix(100, 0),
		rand.New(rand.NewSource(7)),
	)
	if err != nil {
		t.Fatal(err)
	}
	if repeatedOrder := scheduledFingerprints(repeated); !slices.Equal(firstOrder, repeatedOrder) {
		t.Fatalf("orders differ: %v != %v", firstOrder, repeatedOrder)
	}
}

func scheduledFingerprints(items []ScheduledPuzzle) []string {
	result := make([]string, len(items))
	for index, item := range items {
		result[index] = item.Puzzle.Core.Fingerprint
	}
	return result
}
