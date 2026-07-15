package profile

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"chess-trainer/internal/puzzles"
	"chess-trainer/internal/training"
)

const (
	defaultMinimumRating = 400
	defaultMaximumRating = 3000
	recentSessionLimit   = 20
)

type RatingPoint struct {
	Rating     float64 `json:"rating"`
	RecordedAt int64   `json:"recordedAt"`
}

type ThemePerformance struct {
	Theme    string  `json:"theme"`
	Attempts int     `json:"attempts"`
	Accuracy float64 `json:"accuracy"`
}

type RecentSession struct {
	SessionID string `json:"sessionId"`
	Mode      string `json:"mode"`
	Status    string `json:"status"`
	UpdatedAt int64  `json:"updatedAt"`
	Total     int    `json:"total"`
	Completed int    `json:"completed"`
	FirstTry  int    `json:"firstTry"`
	UsedHint  int    `json:"usedHint"`
	Revealed  int    `json:"revealed"`
}

type Summary struct {
	LearnerRating        float64            `json:"learnerRating"`
	RatingTrend          []RatingPoint      `json:"ratingTrend"`
	FirstAttemptAccuracy float64            `json:"firstAttemptAccuracy"`
	HintRate             float64            `json:"hintRate"`
	ThemePerformance     []ThemePerformance `json:"themePerformance"`
	DueReviews           int                `json:"dueReviews"`
	RecentSessions       []RecentSession    `json:"recentSessions"`
}

type PracticeSource struct {
	ID             string `json:"id"`
	Kind           string `json:"kind"`
	MinimumRating  int    `json:"minimumRating"`
	MaximumRating  int    `json:"maximumRating"`
	HasRatingRange bool   `json:"hasRatingRange"`
	MaximumPlies   int    `json:"maximumPlies"`
}

type PracticeFilters struct {
	Sources              []PracticeSource `json:"sources"`
	Themes               []string         `json:"themes"`
	MaximumSolutionPlies int              `json:"maximumSolutionPlies"`
}

type ProfileCatalogPort interface {
	ActiveSourceSummaries(context.Context) ([]puzzles.SourceSummary, error)
	ActiveThemes(context.Context) ([]string, error)
}

type Service struct {
	userDB  *sql.DB
	catalog ProfileCatalogPort
	store   *training.UserStore
	now     func() time.Time
}

func NewService(userDB *sql.DB, catalog ProfileCatalogPort, store *training.UserStore) *Service {
	return &Service{userDB: userDB, catalog: catalog, store: store, now: time.Now}
}

func (s *Service) Get(ctx context.Context) (*training.Profile, error) {
	value, err := s.store.Profile(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &value, nil
}

func (s *Service) UpdateSettings(ctx context.Context, value training.Profile) error {
	if value.SessionSize != 5 && value.SessionSize != 10 && value.SessionSize != 15 {
		return fmt.Errorf("session size must be 5, 10, or 15")
	}
	minimum, maximum, available, err := s.lichessRatingRange(ctx)
	if err != nil {
		return err
	}
	if !available {
		minimum, maximum = defaultMinimumRating, defaultMaximumRating
	}
	if value.LearnerRating < float64(minimum) || value.LearnerRating > float64(maximum) {
		return fmt.Errorf("learner rating must be between %d and %d", minimum, maximum)
	}
	return s.store.UpdateProfile(ctx, value)
}

func (s *Service) Summary(ctx context.Context) (Summary, error) {
	profile, err := s.store.Profile(ctx)
	if err != nil {
		return Summary{}, err
	}
	result := Summary{LearnerRating: profile.LearnerRating}
	var completed, firstTry, usedHint int
	if err := s.userDB.QueryRowContext(ctx, `SELECT
		COUNT(*),
		COALESCE(SUM(first_try), 0),
		COALESCE(SUM(CASE WHEN hints_used > 0 THEN 1 ELSE 0 END), 0)
		FROM attempts WHERE completed_at IS NOT NULL`).Scan(&completed, &firstTry, &usedHint); err != nil {
		return Summary{}, err
	}
	result.FirstAttemptAccuracy = percentage(firstTry, completed)
	result.HintRate = percentage(usedHint, completed)
	if err := s.userDB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM review_state WHERE due_at <= ?`, s.now().Unix(),
	).Scan(&result.DueReviews); err != nil {
		return Summary{}, err
	}
	if result.RatingTrend, err = s.ratingTrend(ctx); err != nil {
		return Summary{}, err
	}
	if result.RecentSessions, err = s.recentSessions(ctx); err != nil {
		return Summary{}, err
	}
	if result.ThemePerformance, err = s.themePerformance(ctx); err != nil {
		return Summary{}, err
	}
	return result, nil
}

func (s *Service) PracticeFilters(ctx context.Context) (PracticeFilters, error) {
	summaries, err := s.catalog.ActiveSourceSummaries(ctx)
	if err != nil {
		return PracticeFilters{}, err
	}
	themes, err := s.catalog.ActiveThemes(ctx)
	if err != nil {
		return PracticeFilters{}, err
	}
	result := PracticeFilters{
		Sources: make([]PracticeSource, 0, len(summaries)),
		Themes:  append([]string{}, themes...),
	}
	for _, summary := range summaries {
		source := PracticeSource{
			ID:           summary.SourceID,
			Kind:         summary.Kind,
			MaximumPlies: summary.MaximumSolutionPlies,
		}
		if summary.MinimumRating != nil && summary.MaximumRating != nil {
			source.MinimumRating = *summary.MinimumRating
			source.MaximumRating = *summary.MaximumRating
			source.HasRatingRange = true
		}
		result.MaximumSolutionPlies = max(result.MaximumSolutionPlies, source.MaximumPlies)
		result.Sources = append(result.Sources, source)
	}
	return result, nil
}

func (s *Service) lichessRatingRange(ctx context.Context) (int, int, bool, error) {
	summaries, err := s.catalog.ActiveSourceSummaries(ctx)
	if err != nil {
		return 0, 0, false, err
	}
	var minimum, maximum int
	available := false
	for _, summary := range summaries {
		if summary.Kind != "lichess" || summary.MinimumRating == nil || summary.MaximumRating == nil {
			continue
		}
		if !available || *summary.MinimumRating < minimum {
			minimum = *summary.MinimumRating
		}
		if !available || *summary.MaximumRating > maximum {
			maximum = *summary.MaximumRating
		}
		available = true
	}
	if !available {
		return 0, 0, false, nil
	}
	return minimum, maximum, true, nil
}

func (s *Service) ratingTrend(ctx context.Context) ([]RatingPoint, error) {
	rows, err := s.userDB.QueryContext(ctx, `SELECT rating, recorded_at
		FROM rating_history ORDER BY recorded_at, rating_history_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	points := []RatingPoint{}
	for rows.Next() {
		var point RatingPoint
		if err := rows.Scan(&point.Rating, &point.RecordedAt); err != nil {
			return nil, err
		}
		points = append(points, point)
	}
	return points, rows.Err()
}

func (s *Service) recentSessions(ctx context.Context) ([]RecentSession, error) {
	rows, err := s.userDB.QueryContext(ctx, `SELECT
		s.session_id,
		s.mode,
		s.status,
		s.updated_at,
		(SELECT COUNT(*) FROM session_items si WHERE si.session_id = s.session_id),
		COALESCE(SUM(CASE WHEN a.completed_at IS NOT NULL THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN a.completed_at IS NOT NULL THEN a.first_try ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN a.completed_at IS NOT NULL AND a.hints_used > 0 THEN 1 ELSE 0 END), 0),
		COALESCE(SUM(CASE WHEN a.completed_at IS NOT NULL THEN a.solution_revealed ELSE 0 END), 0)
		FROM sessions s
		LEFT JOIN attempts a ON a.session_id = s.session_id
		GROUP BY s.session_id, s.mode, s.status, s.updated_at
		ORDER BY s.updated_at DESC, s.session_id DESC
		LIMIT ?`, recentSessionLimit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []RecentSession{}
	for rows.Next() {
		var value RecentSession
		if err := rows.Scan(
			&value.SessionID, &value.Mode, &value.Status, &value.UpdatedAt,
			&value.Total, &value.Completed, &value.FirstTry, &value.UsedHint, &value.Revealed,
		); err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, rows.Err()
}

func (s *Service) themePerformance(ctx context.Context) ([]ThemePerformance, error) {
	rows, err := s.userDB.QueryContext(ctx, `SELECT
		CAST(theme.value AS TEXT),
		COUNT(*),
		COALESCE(SUM(a.first_try), 0)
		FROM attempts a
		JOIN json_each(a.themes_json) AS theme
		WHERE a.completed_at IS NOT NULL
		  AND theme.type = 'text'
		GROUP BY CAST(theme.value AS TEXT)
		ORDER BY CAST(theme.value AS TEXT)`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []ThemePerformance{}
	for rows.Next() {
		var value ThemePerformance
		var firstTry int
		if err := rows.Scan(&value.Theme, &value.Attempts, &firstTry); err != nil {
			return nil, err
		}
		value.Accuracy = percentage(firstTry, value.Attempts)
		values = append(values, value)
	}
	return values, rows.Err()
}

func percentage(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return math.Round((float64(numerator)/float64(denominator))*1000) / 10
}
