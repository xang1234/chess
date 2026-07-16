package profile

import (
	"context"
	"database/sql"
	"path/filepath"
	"reflect"
	"testing"

	"chess-trainer/internal/puzzles"
	"chess-trainer/internal/storage"
	"chess-trainer/internal/training"
)

type task5ProfileCatalog struct {
	summaries        []puzzles.SourceSummary
	themes           []string
	summariesErr     error
	themesErr        error
	bounds           puzzles.RatingBounds
	boundsErr        error
	summariesCalls   int
	themesCalls      int
	boundsCalls      int
	panicIfConsulted bool
}

func (c *task5ProfileCatalog) ActiveSourceSummaries(context.Context) ([]puzzles.SourceSummary, error) {
	if c.panicIfConsulted {
		panic("historical profile reporting consulted the active puzzle catalogue")
	}
	c.summariesCalls++
	return append([]puzzles.SourceSummary(nil), c.summaries...), c.summariesErr
}

func (c *task5ProfileCatalog) ActiveThemes(context.Context) ([]string, error) {
	if c.panicIfConsulted {
		panic("historical profile reporting consulted the active puzzle catalogue")
	}
	c.themesCalls++
	return append([]string(nil), c.themes...), c.themesErr
}

func (c *task5ProfileCatalog) LearnerRatingBounds(context.Context) (puzzles.RatingBounds, error) {
	if c.panicIfConsulted {
		panic("historical profile reporting consulted the active puzzle catalogue")
	}
	c.boundsCalls++
	return c.bounds, c.boundsErr
}

func TestThemePerformanceUsesAttemptSnapshotsOnly(t *testing.T) {
	db := openTask5ProfileUserDB(t)
	seedTask5Profile(t, db)
	mustTask5ProfileExec(t, db, `INSERT INTO attempts(
		attempt_id, fingerprint, source_id, source_kind, rating_snapshot, themes_json,
		started_at, completed_at, first_try
	) VALUES
		('attempt-1', 'same-core', 'old-source', 'lichess', 1400, '["fork","pin"]', 1, 2, 1),
		('attempt-2', 'same-core', 'old-source', 'lichess', 1400, '["fork"]', 3, 4, 0),
		('attempt-open', 'other-core', 'old-source', 'lichess', 1500, '["ignored"]', 5, NULL, 1)`)

	catalog := &task5ProfileCatalog{panicIfConsulted: true}
	summary, err := NewService(db, catalog, training.NewUserStore(db)).Summary(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []ThemePerformance{
		{Theme: "fork", Attempts: 2, Accuracy: 50},
		{Theme: "pin", Attempts: 1, Accuracy: 100},
	}
	if !reflect.DeepEqual(summary.ThemePerformance, want) {
		t.Fatalf("theme performance = %+v, want %+v", summary.ThemePerformance, want)
	}
}

func TestThemePerformanceOmitsNullLegacyThemes(t *testing.T) {
	db := openTask5ProfileUserDB(t)
	seedTask5Profile(t, db)
	mustTask5ProfileExec(t, db, `INSERT INTO attempts(
		attempt_id, fingerprint, source_id, source_kind, rating_snapshot, themes_json,
		started_at, completed_at, first_try
	) VALUES
		('known', 'known-core', 'legacy', 'lichess', 1200, '["skewer"]', 1, 2, 1),
		('unknown', 'unknown-core', 'legacy', NULL, NULL, NULL, 3, 4, 0)`)

	summary, err := NewService(
		db,
		&task5ProfileCatalog{panicIfConsulted: true},
		training.NewUserStore(db),
	).Summary(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []ThemePerformance{{Theme: "skewer", Attempts: 1, Accuracy: 100}}
	if !reflect.DeepEqual(summary.ThemePerformance, want) {
		t.Fatalf("theme performance = %+v, want %+v", summary.ThemePerformance, want)
	}
}

func TestThemePerformanceSurvivesCatalogueRecreation(t *testing.T) {
	db := openTask5ProfileUserDB(t)
	seedTask5Profile(t, db)
	mustTask5ProfileExec(t, db, `INSERT INTO attempts(
		attempt_id, fingerprint, source_id, source_kind, rating_snapshot, themes_json,
		started_at, completed_at, first_try
	) VALUES
		('historical-1', 'gone-core', 'gone-source', 'lichess', 1750, '["clearance","fork"]', 1, 2, 1),
		('historical-2', 'gone-core', 'gone-source', 'lichess', 1750, '["clearance"]', 3, 4, 0)`)

	oldMinimum, oldMaximum := 1700, 1800
	oldCatalogue := &task5ProfileCatalog{
		summaries: []puzzles.SourceSummary{{
			SourceID: "gone-source", Kind: "lichess",
			MinimumRating: &oldMinimum, MaximumRating: &oldMaximum,
		}},
		themes: []string{"clearance", "fork"},
	}
	before, err := NewService(db, oldCatalogue, training.NewUserStore(db)).Summary(context.Background())
	if err != nil {
		t.Fatal(err)
	}

	// A replacement catalogue can contain unrelated sources and themes. Historical
	// reporting must remain a projection of the durable attempt snapshots.
	newMinimum, newMaximum := 900, 1100
	recreatedCatalogue := &task5ProfileCatalog{
		summaries: []puzzles.SourceSummary{{
			SourceID: "replacement-source", Kind: "pgn",
			MinimumRating: &newMinimum, MaximumRating: &newMaximum,
		}},
		themes: []string{"completely-new-theme"},
	}
	after, err := NewService(db, recreatedCatalogue, training.NewUserStore(db)).Summary(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(after.ThemePerformance, before.ThemePerformance) {
		t.Fatalf("theme performance changed across catalogue recreation: before=%+v after=%+v",
			before.ThemePerformance, after.ThemePerformance)
	}
	if oldCatalogue.summariesCalls != 0 || oldCatalogue.themesCalls != 0 ||
		recreatedCatalogue.summariesCalls != 0 || recreatedCatalogue.themesCalls != 0 {
		t.Fatalf("historical summary consulted old/recreated catalogues: old=%d/%d new=%d/%d",
			oldCatalogue.summariesCalls, oldCatalogue.themesCalls,
			recreatedCatalogue.summariesCalls, recreatedCatalogue.themesCalls)
	}
}

func TestPracticeFiltersUseActiveCataloguePort(t *testing.T) {
	db := openTask5ProfileUserDB(t)
	minimum, maximum := 900, 2100
	catalog := &task5ProfileCatalog{
		summaries: []puzzles.SourceSummary{
			{SourceID: "club", Kind: "pgn", MaximumSolutionPlies: 7},
			{
				SourceID: "lichess", Kind: "lichess", MinimumRating: &minimum,
				MaximumRating: &maximum, MaximumSolutionPlies: 5,
			},
		},
		themes: []string{"fork", "pin"},
	}

	filters, err := NewService(db, catalog, training.NewUserStore(db)).PracticeFilters(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := PracticeFilters{
		Sources: []PracticeSource{
			{ID: "club", Kind: "pgn", MaximumPlies: 7},
			{
				ID: "lichess", Kind: "lichess", MinimumRating: 900,
				MaximumRating: 2100, HasRatingRange: true, MaximumPlies: 5,
			},
		},
		Themes: []string{"fork", "pin"}, MaximumSolutionPlies: 7,
	}
	if !reflect.DeepEqual(filters, want) {
		t.Fatalf("PracticeFilters() = %+v, want %+v", filters, want)
	}
	if catalog.summariesCalls != 1 || catalog.themesCalls != 1 {
		t.Fatalf("catalogue calls = summaries %d themes %d, want one each",
			catalog.summariesCalls, catalog.themesCalls)
	}
}

func TestRatingBoundsUseCanonicalCatalogBoundary(t *testing.T) {
	db := openTask5ProfileUserDB(t)
	catalog := &task5ProfileCatalog{bounds: puzzles.RatingBounds{Minimum: 800, Maximum: 2400}}
	service := NewService(db, catalog, training.NewUserStore(db))

	if err := service.UpdateSettings(context.Background(), training.Profile{
		LearnerRating: 800, SessionSize: 5,
	}); err != nil {
		t.Fatalf("minimum active Lichess rating was rejected: %v", err)
	}
	for _, rating := range []float64{799, 2401} {
		if err := service.UpdateSettings(context.Background(), training.Profile{
			LearnerRating: rating, SessionSize: 5,
		}); err == nil {
			t.Fatalf("out-of-range rating %.0f was accepted", rating)
		}
	}
	if catalog.boundsCalls != 3 || catalog.summariesCalls != 0 || catalog.themesCalls != 0 {
		t.Fatalf("catalogue calls = bounds %d summaries %d themes %d", catalog.boundsCalls, catalog.summariesCalls, catalog.themesCalls)
	}
}

func openTask5ProfileUserDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "user.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Migrate(db, "user"); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close user database: %v", err)
		}
	})
	return db
}

func seedTask5Profile(t *testing.T, db *sql.DB) {
	t.Helper()
	mustTask5ProfileExec(t, db, `INSERT INTO profile(
		id, learner_rating, session_size, created_at, updated_at
	) VALUES (1, 1500, 10, 1, 1)`)
}

func mustTask5ProfileExec(t *testing.T, db *sql.DB, statement string, args ...any) {
	t.Helper()
	if _, err := db.Exec(statement, args...); err != nil {
		t.Fatal(err)
	}
}
