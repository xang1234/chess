package profile

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"chess-trainer/internal/training"
)

const (
	defaultMinimumRating = 400
	defaultMaximumRating = 3000
	recentSessionLimit   = 20
	themeQueryBatchSize  = 300
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

type Service struct {
	userDB   *sql.DB
	puzzleDB *sql.DB
	store    *training.UserStore
	now      func() time.Time
}

func NewService(userDB, puzzleDB *sql.DB, store *training.UserStore) *Service {
	return &Service{userDB: userDB, puzzleDB: puzzleDB, store: store, now: time.Now}
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
	rows, err := s.puzzleDB.QueryContext(ctx, `SELECT
		s.source_id,
		s.kind,
		MIN(ps.rating),
		MAX(ps.rating),
		MAX(p.solution_plies)
		FROM sources s
		JOIN puzzle_sources ps ON ps.source_id = s.source_id
		JOIN puzzles p ON p.fingerprint = ps.fingerprint
		GROUP BY s.source_id, s.kind
		ORDER BY s.source_id`)
	if err != nil {
		return PracticeFilters{}, err
	}
	defer rows.Close()
	result := PracticeFilters{Sources: []PracticeSource{}, Themes: []string{}}
	for rows.Next() {
		var source PracticeSource
		var minimum, maximum sql.NullInt64
		if err := rows.Scan(&source.ID, &source.Kind, &minimum, &maximum, &source.MaximumPlies); err != nil {
			return PracticeFilters{}, err
		}
		if minimum.Valid && maximum.Valid {
			source.MinimumRating = int(minimum.Int64)
			source.MaximumRating = int(maximum.Int64)
			source.HasRatingRange = true
		}
		result.MaximumSolutionPlies = max(result.MaximumSolutionPlies, source.MaximumPlies)
		result.Sources = append(result.Sources, source)
	}
	if err := rows.Err(); err != nil {
		return PracticeFilters{}, err
	}
	themeRows, err := s.puzzleDB.QueryContext(ctx, `SELECT DISTINCT theme FROM puzzle_themes ORDER BY theme`)
	if err != nil {
		return PracticeFilters{}, err
	}
	defer themeRows.Close()
	for themeRows.Next() {
		var theme string
		if err := themeRows.Scan(&theme); err != nil {
			return PracticeFilters{}, err
		}
		result.Themes = append(result.Themes, theme)
	}
	return result, themeRows.Err()
}

func (s *Service) lichessRatingRange(ctx context.Context) (int, int, bool, error) {
	var minimum, maximum sql.NullInt64
	err := s.puzzleDB.QueryRowContext(ctx, `SELECT MIN(ps.rating), MAX(ps.rating)
		FROM puzzle_sources ps
		JOIN sources s ON s.source_id = ps.source_id
		WHERE s.kind = 'lichess' AND ps.rating IS NOT NULL`).Scan(&minimum, &maximum)
	if err != nil {
		return 0, 0, false, err
	}
	if !minimum.Valid || !maximum.Valid {
		return 0, 0, false, nil
	}
	return int(minimum.Int64), int(maximum.Int64), true, nil
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

type puzzleAttemptStats struct {
	fingerprint string
	sourceID    string
	attempts    int
	firstTry    int
}

func (s *Service) themePerformance(ctx context.Context) ([]ThemePerformance, error) {
	rows, err := s.userDB.QueryContext(ctx, `SELECT
		fingerprint, source_id, COUNT(*), COALESCE(SUM(first_try), 0)
		FROM attempts WHERE completed_at IS NOT NULL
		GROUP BY fingerprint, source_id`)
	if err != nil {
		return nil, err
	}
	stats := []puzzleAttemptStats{}
	for rows.Next() {
		var value puzzleAttemptStats
		if err := rows.Scan(&value.fingerprint, &value.sourceID, &value.attempts, &value.firstTry); err != nil {
			rows.Close()
			return nil, err
		}
		stats = append(stats, value)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	totals := map[string][2]int{}
	for start := 0; start < len(stats); start += themeQueryBatchSize {
		end := min(start+themeQueryBatchSize, len(stats))
		query := strings.Builder{}
		query.WriteString(`SELECT fingerprint, source_id, theme FROM puzzle_themes WHERE `)
		args := make([]any, 0, (end-start)*2)
		for index, stat := range stats[start:end] {
			if index > 0 {
				query.WriteString(" OR ")
			}
			query.WriteString("(fingerprint = ? AND source_id = ?)")
			args = append(args, stat.fingerprint, stat.sourceID)
		}
		themeRows, err := s.puzzleDB.QueryContext(ctx, query.String(), args...)
		if err != nil {
			return nil, err
		}
		statsByKey := make(map[string]puzzleAttemptStats, end-start)
		for _, stat := range stats[start:end] {
			statsByKey[stat.fingerprint+"\x00"+stat.sourceID] = stat
		}
		for themeRows.Next() {
			var fingerprint, sourceID, theme string
			if err := themeRows.Scan(&fingerprint, &sourceID, &theme); err != nil {
				themeRows.Close()
				return nil, err
			}
			stat := statsByKey[fingerprint+"\x00"+sourceID]
			total := totals[theme]
			total[0] += stat.attempts
			total[1] += stat.firstTry
			totals[theme] = total
		}
		if err := themeRows.Close(); err != nil {
			return nil, err
		}
		if err := themeRows.Err(); err != nil {
			return nil, err
		}
	}
	values := make([]ThemePerformance, 0, len(totals))
	for theme, total := range totals {
		values = append(values, ThemePerformance{
			Theme: theme, Attempts: total[0], Accuracy: percentage(total[1], total[0]),
		})
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Theme < values[j].Theme })
	return values, nil
}

func percentage(numerator, denominator int) float64 {
	if denominator == 0 {
		return 0
	}
	return math.Round((float64(numerator)/float64(denominator))*1000) / 10
}
