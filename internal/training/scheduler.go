package training

import (
	"context"
	"database/sql"
	"errors"
	"math"
	"math/rand"
	"sort"
	"time"

	"chess-trainer/internal/domain"
)

const maximumReviewsPerSession = 4

type CatalogPort interface {
	Get(context.Context, string) (domain.Puzzle, error)
	RatedCandidates(context.Context, int, int, []string, int) ([]domain.Puzzle, error)
}

type UserPort interface {
	DueReviews(context.Context, time.Time, int) ([]ReviewState, error)
	RecentFingerprints(context.Context, int) ([]string, error)
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
	Puzzle        domain.Puzzle `json:"puzzle"`
	SourceID      string        `json:"sourceId"`
	Kind          ScheduledKind `json:"kind"`
	UpdatesRating bool          `json:"updatesRating"`
}

type Scheduler struct {
	Catalog CatalogPort
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
	due, err := s.User.DueReviews(ctx, now, maximumReviewsPerSession)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(due, func(i, j int) bool {
		return due[i].DueAt.Before(due[j].DueAt)
	})
	if len(due) > maximumReviewsPerSession {
		due = due[:maximumReviewsPerSession]
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
	for _, review := range due {
		if len(items) == profile.SessionSize {
			break
		}
		puzzle, err := s.Catalog.Get(ctx, review.Fingerprint)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		excluded[puzzle.Fingerprint] = struct{}{}
		items = append(items, ScheduledPuzzle{
			Puzzle:        puzzle,
			SourceID:      firstSourceID(puzzle),
			Kind:          ScheduledReview,
			UpdatesRating: false,
		})
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
		eligible := make([]domain.Puzzle, 0, len(candidates))
		for _, puzzle := range candidates {
			if _, duplicate := excluded[puzzle.Fingerprint]; duplicate {
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
			excluded[puzzle.Fingerprint] = struct{}{}
			items = append(items, ScheduledPuzzle{
				Puzzle:        puzzle,
				SourceID:      firstSourceID(puzzle),
				Kind:          ScheduledNew,
				UpdatesRating: true,
			})
		}
	}
	return items, nil
}

func fingerprintSet(values map[string]struct{}) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func firstSourceID(puzzle domain.Puzzle) string {
	if len(puzzle.Sources) == 0 {
		return ""
	}
	return puzzle.Sources[0].SourceID
}
