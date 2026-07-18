package profile

import (
	"context"
	"database/sql"
	"math"
	"path/filepath"
	"testing"
	"time"

	"chess-trainer/internal/puzzles"
	"chess-trainer/internal/storage"
	"chess-trainer/internal/training"
)

func openProfileStore(t *testing.T) *sql.DB {
	t.Helper()
	root := t.TempDir()
	userDB, err := storage.Open(filepath.Join(root, "user.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Migrate(userDB, "user"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		userDB.Close()
	})
	return userDB
}

type profileTestCatalog struct {
	summaries []puzzles.SourceSummary
	themes    []string
	bounds    puzzles.RatingBounds
}

func (c *profileTestCatalog) ActiveSourceSummaries(context.Context) ([]puzzles.SourceSummary, error) {
	return append([]puzzles.SourceSummary(nil), c.summaries...), nil
}

func (c *profileTestCatalog) ActiveThemes(context.Context) ([]string, error) {
	return append([]string(nil), c.themes...), nil
}

func (c *profileTestCatalog) LearnerRatingBounds(context.Context) (puzzles.RatingBounds, error) {
	return c.bounds, nil
}

func seedProfileCatalogue() *profileTestCatalog {
	minimum, maximum := 1000, 2000
	return &profileTestCatalog{
		bounds: puzzles.RatingBounds{Minimum: minimum, Maximum: maximum},
		summaries: []puzzles.SourceSummary{{
			SourceID: "lichess", Kind: "lichess", MinimumRating: &minimum,
			MaximumRating: &maximum, MaximumSolutionPlies: 5,
		}},
		themes: []string{"fork", "pin"},
	}
}

func seedProgress(t *testing.T, db *sql.DB, now time.Time) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO profile(
		id, learner_rating, session_size, created_at, updated_at
	) VALUES (1, 1320, 10, 1, 3)`); err != nil {
		t.Fatal(err)
	}
	for index, rating := range []float64{1200, 1260, 1320} {
		if _, err := db.Exec(`INSERT INTO rating_history(rating, recorded_at) VALUES (?, ?)`,
			rating, int64(index+1)); err != nil {
			t.Fatal(err)
		}
	}
	for index := 0; index < 22; index++ {
		id := "session-" + string(rune('a'+index))
		if _, err := db.Exec(`INSERT INTO sessions(
			session_id, mode, status, created_at, updated_at, current_index
		) VALUES (?, 'guided', 'completed', ?, ?, 1)`, id, index+1, index+1); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO session_items(
			session_id, ordinal, fingerprint, source_id, state_json
		) VALUES (?, 0, 'fork-puzzle', 'lichess', '{}')`, id); err != nil {
			t.Fatal(err)
		}
	}
	attempts := []struct {
		id          string
		session     string
		fingerprint string
		incorrect   int
		hints       int
		firstTry    int
		themesJSON  string
	}{
		{"attempt-1", "session-a", "fork-puzzle", 0, 0, 1, `["fork"]`},
		{"attempt-2", "session-b", "fork-puzzle", 1, 0, 0, `["fork"]`},
		{"attempt-3", "session-c", "pin-puzzle", 0, 1, 0, `["pin"]`},
	}
	for index, attempt := range attempts {
		if _, err := db.Exec(`INSERT INTO attempts(
			attempt_id, session_id, fingerprint, source_id, started_at, completed_at,
			incorrect_moves, hints_used, solution_revealed, first_try, duration_ms, themes_json
		) VALUES (?, ?, ?, 'lichess', ?, ?, ?, ?, 0, ?, 1000, ?)`,
			attempt.id, attempt.session, attempt.fingerprint, index+1, index+1,
			attempt.incorrect, attempt.hints, attempt.firstTry, attempt.themesJSON); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO review_state(
		fingerprint, due_at, interval_index, successful_reviews, last_outcome
	) VALUES
		('fork-puzzle', ?, 0, 0, 'missed'),
		('pin-puzzle', ?, 1, 1, 'clean')`, now.Add(-time.Hour).Unix(), now.Add(time.Hour).Unix()); err != nil {
		t.Fatal(err)
	}
}

func TestServiceSummarisesProgressWithSQLAggregates(t *testing.T) {
	userDB := openProfileStore(t)
	catalog := seedProfileCatalogue()
	now := time.Unix(1000, 0)
	seedProgress(t, userDB, now)
	service := NewService(userDB, catalog, training.NewUserStore(userDB))
	service.now = func() time.Time { return now }

	summary, err := service.Summary(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if summary.LearnerRating != 1320 || len(summary.RatingTrend) != 3 {
		t.Fatalf("rating summary=%+v", summary)
	}
	if math.Abs(summary.FirstAttemptAccuracy-33.3) > 0.01 || math.Abs(summary.HintRate-33.3) > 0.01 {
		t.Fatalf("accuracy=%v hintRate=%v", summary.FirstAttemptAccuracy, summary.HintRate)
	}
	if summary.DueReviews != 1 {
		t.Fatalf("due reviews=%d", summary.DueReviews)
	}
	if len(summary.ThemePerformance) != 2 || summary.ThemePerformance[0].Theme != "fork" ||
		summary.ThemePerformance[0].Accuracy != 50 || summary.ThemePerformance[1].Theme != "pin" ||
		summary.ThemePerformance[1].Accuracy != 0 {
		t.Fatalf("themes=%+v", summary.ThemePerformance)
	}
	if len(summary.RecentSessions) != 20 || summary.RecentSessions[0].SessionID != "session-v" ||
		summary.RecentSessions[19].SessionID != "session-c" {
		t.Fatalf("recent sessions=%+v", summary.RecentSessions)
	}
}

func TestServiceValidatesSettingsAgainstActiveLichessRange(t *testing.T) {
	userDB := openProfileStore(t)
	catalog := seedProfileCatalogue()
	service := NewService(userDB, catalog, training.NewUserStore(userDB))

	if err := service.UpdateSettings(context.Background(), training.Profile{
		LearnerRating: 1500,
		SessionSize:   5,
	}); err != nil {
		t.Fatal(err)
	}
	for _, invalid := range []training.Profile{
		{LearnerRating: 1500, SessionSize: 7},
		{LearnerRating: 999, SessionSize: 5},
		{LearnerRating: 2001, SessionSize: 5},
	} {
		if err := service.UpdateSettings(context.Background(), invalid); err == nil {
			t.Fatalf("UpdateSettings(%+v) succeeded", invalid)
		}
	}

	filters, err := service.PracticeFilters(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(filters.Sources) != 1 || filters.Sources[0].ID != "lichess" ||
		filters.Sources[0].MinimumRating != 1000 || filters.Sources[0].MaximumRating != 2000 ||
		filters.MaximumSolutionPlies != 5 || len(filters.Themes) != 2 ||
		filters.LearnerRatingBounds != (puzzles.RatingBounds{Minimum: 1000, Maximum: 2000}) {
		t.Fatalf("filters=%+v", filters)
	}
}

func TestPracticeFiltersUseDefaultLearnerBoundsWithoutCataloguedRatings(t *testing.T) {
	userDB := openProfileStore(t)
	filters, err := NewService(
		userDB,
		&profileTestCatalog{},
		training.NewUserStore(userDB),
	).PracticeFilters(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if filters.LearnerRatingBounds != puzzles.DefaultLearnerRatingBounds() {
		t.Fatalf("learner rating bounds = %+v, want fallback %+v",
			filters.LearnerRatingBounds, puzzles.DefaultLearnerRatingBounds())
	}
}

func TestProfilePracticeFiltersIncludeEveryRatedSourceInLearnerRatingBounds(t *testing.T) {
	userDB := openProfileStore(t)
	jsonMin, jsonMax := 900, 2100
	lichessMin, lichessMax := 1100, 1800
	filters, err := NewService(
		userDB,
		&profileTestCatalog{summaries: []puzzles.SourceSummary{
			{SourceID: "canonical", Kind: "canonical-json", MinimumRating: &jsonMin, MaximumRating: &jsonMax},
			{SourceID: "lichess", Kind: "lichess", MinimumRating: &lichessMin, MaximumRating: &lichessMax},
			{SourceID: "pgn", Kind: "tactical-pgn"},
		}},
		training.NewUserStore(userDB),
	).PracticeFilters(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if want := (puzzles.RatingBounds{Minimum: 900, Maximum: 2100}); filters.LearnerRatingBounds != want {
		t.Fatalf("learner rating bounds = %+v, want %+v", filters.LearnerRatingBounds, want)
	}
}
