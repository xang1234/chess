package training

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"math/rand"
	"sort"
	"time"

	"chess-trainer/internal/puzzles"
)

const maximumReviewsPerSession = 4

type TrainingCatalogPort interface {
	Get(context.Context, puzzles.PuzzleKey) (puzzles.TrainingPuzzle, error)
	Resolve(context.Context, string, string) (puzzles.TrainingPuzzle, error)
	RatedCandidates(context.Context, int, int, []string, int) ([]puzzles.TrainingPuzzle, error)
	FreePracticeCandidates(context.Context, string, *int, *int, []string, *int, int) ([]puzzles.TrainingPuzzle, error)
}

type UserPort interface {
	DueReviews(context.Context, time.Time, *ReviewCursor, int) ([]ReviewState, error)
	RecentFingerprints(context.Context, int) ([]string, error)
}

type ReviewCursor struct {
	DueAt       time.Time
	Fingerprint string
}

type Profile struct {
	LearnerRating float64 `json:"learnerRating"`
	SessionSize   int     `json:"sessionSize"`
}

type ScheduledKind string

const (
	ScheduledReview ScheduledKind = "review"
	ScheduledNew    ScheduledKind = "new"
)

type ScheduledPuzzle struct {
	Puzzle        puzzles.TrainingPuzzle `json:"puzzle"`
	Kind          ScheduledKind          `json:"kind"`
	UpdatesRating bool                   `json:"updatesRating"`
}

type Scheduler struct {
	Catalog TrainingCatalogPort
	User    UserPort
}

func (s Scheduler) BuildGuided(
	ctx context.Context,
	profile Profile,
	now time.Time,
	random *rand.Rand,
) ([]ScheduledPuzzle, error) {
	if profile.SessionSize <= 0 {
		return []ScheduledPuzzle{}, nil
	}
	due, err := s.User.DueReviews(ctx, now, nil, maximumReviewsPerSession)
	if err != nil {
		return nil, err
	}

	recent, err := s.User.RecentFingerprints(ctx, 200)
	if err != nil {
		return nil, err
	}
	excluded := make(map[string]struct{}, len(recent)+profile.SessionSize)
	for _, fingerprint := range recent {
		excluded[fingerprint] = struct{}{}
	}
	items := make([]ScheduledPuzzle, 0, profile.SessionSize)
	reviewTarget := min(profile.SessionSize, maximumReviewsPerSession)
	var cursor *ReviewCursor
	for len(items) < reviewTarget && len(due) > 0 {
		sort.SliceStable(due, func(i, j int) bool {
			if due[i].DueAt.Equal(due[j].DueAt) {
				return due[i].Fingerprint < due[j].Fingerprint
			}
			return due[i].DueAt.Before(due[j].DueAt)
		})
		if cursor != nil && !reviewCursorLess(*cursor, reviewCursor(due[0])) {
			return nil, errors.New("due review page did not advance")
		}
		for _, review := range due {
			if len(items) == reviewTarget {
				break
			}
			puzzle, err := s.Catalog.Resolve(ctx, review.Fingerprint, review.PreferredSourceID)
			if errors.Is(err, sql.ErrNoRows) {
				continue
			}
			if err != nil {
				return nil, err
			}
			excluded[puzzle.Core.Fingerprint] = struct{}{}
			items = append(items, ScheduledPuzzle{
				Puzzle:        puzzle,
				Kind:          ScheduledReview,
				UpdatesRating: false,
			})
		}
		if len(items) == reviewTarget || len(due) < maximumReviewsPerSession {
			break
		}
		next := reviewCursor(due[len(due)-1])
		cursor = &next
		due, err = s.User.DueReviews(ctx, now, cursor, maximumReviewsPerSession)
		if err != nil {
			return nil, err
		}
	}

	if random == nil {
		random = rand.New(rand.NewSource(now.UnixNano()))
	}
	center := int(math.Round(profile.LearnerRating))
	for _, width := range []int{100, 200, 300, 400} {
		remaining := profile.SessionSize - len(items)
		if remaining == 0 {
			break
		}
		candidates, err := s.Catalog.RatedCandidates(
			ctx,
			center-width,
			center+width,
			fingerprintSet(excluded),
			remaining,
		)
		if err != nil {
			return nil, err
		}
		eligible := make([]puzzles.TrainingPuzzle, 0, len(candidates))
		for _, puzzle := range candidates {
			if _, duplicate := excluded[puzzle.Core.Fingerprint]; duplicate {
				continue
			}
			eligible = append(eligible, puzzle)
		}
		random.Shuffle(len(eligible), func(i, j int) {
			eligible[i], eligible[j] = eligible[j], eligible[i]
		})
		for _, puzzle := range eligible {
			if len(items) == profile.SessionSize {
				break
			}
			excluded[puzzle.Core.Fingerprint] = struct{}{}
			items = append(items, ScheduledPuzzle{
				Puzzle:        puzzle,
				Kind:          ScheduledNew,
				UpdatesRating: true,
			})
		}
	}
	return items, nil
}

func reviewCursor(state ReviewState) ReviewCursor {
	return ReviewCursor{DueAt: state.DueAt, Fingerprint: state.Fingerprint}
}

func reviewCursorLess(left, right ReviewCursor) bool {
	return left.DueAt.Before(right.DueAt) ||
		(left.DueAt.Equal(right.DueAt) && left.Fingerprint < right.Fingerprint)
}

func fingerprintSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
