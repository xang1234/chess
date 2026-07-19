package storage

import (
	"database/sql"
	"path/filepath"
	"testing"
)

func TestProbeRecognizesExactV3AsUpgradableNonLegacy(t *testing.T) {
	path := populatedPuzzleSchemaV3Fixture(t)

	state, err := ProbePuzzleStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if !state.Exists || state.Legacy || !state.Upgradable || state.Format != 3 {
		t.Fatalf("ProbePuzzleStore() = %+v, want exact upgradable v3", state)
	}
}

func TestUpgradePuzzleStoreBackfillsExactGenerationMaximums(t *testing.T) {
	path := populatedPuzzleSchemaV3Fixture(t)

	if err := UpgradePuzzleStore(path); err != nil {
		t.Fatal(err)
	}
	state, err := ProbePuzzleStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if state.Format != 4 || state.Legacy || state.Upgradable {
		t.Fatalf("upgraded state = %+v, want current v4", state)
	}

	db := openPuzzleValidationFixture(t, path)
	for generationID, want := range map[string]int{
		"populated-generation": 5,
		"empty-generation":     0,
	} {
		var got int
		if err := db.QueryRow(`SELECT maximum_solution_plies
			FROM source_generations WHERE generation_id = ?`, generationID).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("generation %q maximum = %d, want %d", generationID, got, want)
		}
	}
	assertV3FixtureContentUnchangedExceptSummary(t, db)
}

func TestUpgradePuzzleStoreRejectsTamperedV3WithoutChangingIt(t *testing.T) {
	path := populatedPuzzleSchemaV3Fixture(t)
	db := createSQLiteFixture(t, path, "")
	if _, err := db.Exec(`CREATE TABLE unexpected_local_state(value TEXT)`); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	wantHash := hashFile(t, path)

	if err := UpgradePuzzleStore(path); err == nil {
		t.Fatal("UpgradePuzzleStore() accepted a tampered v3 catalogue")
	}
	if gotHash := hashFile(t, path); gotHash != wantHash {
		t.Fatal("rejected v3 catalogue changed on disk")
	}
}

func populatedPuzzleSchemaV3Fixture(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "puzzles.sqlite")
	db := createSQLiteFixture(t, path, currentPuzzleSchemaFixture(t))
	statements := []string{
		`INSERT INTO sources(source_id, kind) VALUES
			('populated', 'test'),
			('empty', 'test')`,
		`INSERT INTO source_generations(
			generation_id, source_id, status, source_path, checksum, started_at, sealed_at
		) VALUES
			('populated-generation', 'populated', 'sealed', '/populated', 'populated-checksum', 1, 2),
			('empty-generation', 'empty', 'sealed', '/empty', 'empty-checksum', 1, 2)`,
		`INSERT INTO source_heads(source_id, generation_id) VALUES
			('populated', 'populated-generation'),
			('empty', 'empty-generation')`,
		`INSERT INTO puzzle_cores(
			fingerprint, displayed_fen, solver, solution_json, solution_plies
		) VALUES
			('two-plies', 'fen-two', 'white', '[{"uci":"a1a2"}]', 2),
			('five-plies', 'fen-five', 'black', '[{"uci":"h8h7"}]', 5)`,
		`INSERT INTO puzzle_occurrences(
			generation_id, fingerprint, rating, metadata_json, themes_json, ordinal
		) VALUES
			('populated-generation', 'two-plies', 1200, '{}', '["fork"]', 1),
			('populated-generation', 'five-plies', NULL, '{}', '[]', 2)`,
		`INSERT INTO occurrence_ratings(generation_id, rating_key, fingerprint) VALUES
			('populated-generation', 1200, 'two-plies'),
			('populated-generation', -9223372036854775808, 'five-plies')`,
		`INSERT INTO occurrence_themes(generation_id, theme, fingerprint)
			VALUES ('populated-generation', 'fork', 'two-plies')`,
		`INSERT INTO generation_themes(generation_id, theme)
			VALUES ('populated-generation', 'fork')`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			db.Close()
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertV3FixtureContentUnchangedExceptSummary(t *testing.T, db puzzleTestQueryer) {
	t.Helper()
	for table, want := range map[string]int{
		"sources":            2,
		"source_generations": 2,
		"source_heads":       2,
		"puzzle_cores":       2,
		"puzzle_occurrences": 2,
		"occurrence_ratings": 2,
		"occurrence_themes":  1,
		"generation_themes":  1,
	} {
		var got int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("%s rows = %d, want %d", table, got, want)
		}
	}
	var versions int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = 4`).Scan(&versions); err != nil {
		t.Fatal(err)
	}
	if versions != 1 {
		t.Fatalf("v4 migration markers = %d, want 1", versions)
	}
}

type puzzleTestQueryer interface {
	QueryRow(string, ...any) *sql.Row
}
