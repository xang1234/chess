package training

import (
	"context"
	"math/rand"
	"slices"
	"testing"
	"time"

	"chess-trainer/internal/domain"
)

type candidateCall struct {
	minimum  int
	maximum  int
	excluded []string
	limit    int
}

type schedulerCatalogFake struct {
	puzzles map[string]domain.Puzzle
	calls   []candidateCall
}

func (f *schedulerCatalogFake) Get(_ context.Context, fingerprint string) (domain.Puzzle, error) {
	return f.puzzles[fingerprint], nil
}

func (f *schedulerCatalogFake) RatedCandidates(
	_ context.Context,
	minimum int,
	maximum int,
	excluded []string,
	limit int,
) ([]domain.Puzzle, error) {
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
	result := make([]domain.Puzzle, 0, count)
	for index := 1; index <= count; index++ {
		fingerprint := "new" + string(rune('0'+index))
		if !slices.Contains(excluded, fingerprint) {
			result = append(result, f.puzzles[fingerprint])
		}
	}
	return result, nil
}

type schedulerUserFake struct {
	due         []ReviewState
	recent      []string
	reviewLimit int
}

func (f *schedulerUserFake) DueReviews(_ context.Context, _ time.Time, limit int) ([]ReviewState, error) {
	f.reviewLimit = limit
	return slices.Clone(f.due), nil
}

func (f *schedulerUserFake) RecentFingerprints(context.Context, int) ([]string, error) {
	return slices.Clone(f.recent), nil
}

func scheduledTestPuzzle(fingerprint string, rating int) domain.Puzzle {
	return domain.Puzzle{
		Fingerprint:  fingerprint,
		DisplayedFEN: "7k/5Q2/6K1/8/8/8/8/8 w - - 0 1",
		Solver:       domain.White,
		Solution:     []domain.MoveNode{{UCI: "f7f8"}},
		Rating:       &rating,
		Sources:      []domain.SourceRef{{SourceID: "lichess"}},
	}
}

func TestSchedulerBuildsFourReviewsThenWidensForNewPuzzles(t *testing.T) {
	puzzles := make(map[string]domain.Puzzle)
	due := make([]ReviewState, 0, 6)
	for index := 1; index <= 6; index++ {
		fingerprint := "review" + string(rune('0'+index))
		puzzles[fingerprint] = scheduledTestPuzzle(fingerprint, 1400+index)
		due = append(due, ReviewState{
			Fingerprint: fingerprint,
			DueAt:       time.Unix(int64(index), 0),
		})
	}
	for index := 1; index <= 6; index++ {
		fingerprint := "new" + string(rune('0'+index))
		puzzles[fingerprint] = scheduledTestPuzzle(fingerprint, 1500)
	}
	catalog := &schedulerCatalogFake{puzzles: puzzles}
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
		result[index] = item.Puzzle.Fingerprint
	}
	return result
}
