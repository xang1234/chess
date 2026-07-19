package storage

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestOpenGenerationPuzzleStoreCreatesExactSchema(t *testing.T) {
	store := openTestGenerationPuzzleStore(t, filepath.Join(t.TempDir(), "puzzles.sqlite"))

	wantTables := []string{
		"generation_themes",
		"occurrence_ratings",
		"occurrence_themes",
		"puzzle_cores",
		"puzzle_occurrences",
		"schema_migrations",
		"source_generations",
		"source_heads",
		"sources",
	}
	if got := schemaObjectNames(t, store.Reader, "table"); !slices.Equal(got, wantTables) {
		t.Fatalf("tables = %q, want %q", got, wantTables)
	}

	wantIndexes := []string{
		"idx_generations_cleanup",
		"idx_occurrences_generation",
		"idx_source_heads_generation",
	}
	if got := schemaObjectNames(t, store.Reader, "index"); !slices.Equal(got, wantIndexes) {
		t.Fatalf("indexes = %q, want %q", got, wantIndexes)
	}

	indexColumns := map[string][]string{
		"idx_generations_cleanup":     {"status", "generation_id"},
		"idx_occurrences_generation":  {"generation_id", "fingerprint"},
		"idx_source_heads_generation": {"generation_id"},
	}
	for name, want := range indexColumns {
		if got := pragmaNames(t, store.Reader, `PRAGMA index_info("`+name+`")`, 2); !slices.Equal(got, want) {
			t.Fatalf("index %s columns = %q, want %q", name, got, want)
		}
	}

	wantColumns := map[string][]string{
		"generation_themes":  {"generation_id", "theme"},
		"occurrence_ratings": {"generation_id", "rating_key", "fingerprint"},
		"sources":            {"source_id", "kind"},
		"source_generations": {"generation_id", "source_id", "status", "source_path", "checksum", "started_at", "sealed_at", "maximum_solution_plies"},
		"source_heads":       {"source_id", "generation_id"},
		"puzzle_cores":       {"fingerprint", "displayed_fen", "solver", "solution_json", "solution_plies"},
		"puzzle_occurrences": {"generation_id", "fingerprint", "external_id", "source_fen", "prelude_uci", "rating", "popularity", "play_count", "source_url", "attribution", "metadata_json", "themes_json", "ordinal"},
		"occurrence_themes":  {"generation_id", "fingerprint", "theme"},
		"schema_migrations":  {"version"},
	}
	for _, table := range []string{"puzzle_occurrences", "occurrence_ratings", "occurrence_themes", "generation_themes"} {
		assertPuzzleTableWithoutRowID(t, store.Reader, table, true)
	}
	assertPuzzleTableWithoutRowID(t, store.Reader, "puzzle_cores", false)
	if got, want := puzzlePrimaryKeyColumns(t, store.Reader, "puzzle_occurrences"), []string{"fingerprint", "generation_id"}; !slices.Equal(got, want) {
		t.Fatalf("puzzle_occurrences primary key = %q, want %q", got, want)
	}
	if got, want := puzzleForeignKeyColumns(t, store.Reader, "occurrence_themes", "puzzle_occurrences"), []string{"fingerprint:fingerprint", "generation_id:generation_id"}; !slices.Equal(got, want) {
		t.Fatalf("occurrence_themes parent key = %q, want %q", got, want)
	}
	if got, want := puzzlePrimaryKeyColumns(t, store.Reader, "occurrence_ratings"), []string{"generation_id", "rating_key", "fingerprint"}; !slices.Equal(got, want) {
		t.Fatalf("occurrence_ratings primary key = %q, want %q", got, want)
	}
	if got, want := puzzleForeignKeyColumns(t, store.Reader, "occurrence_ratings", "puzzle_occurrences"), []string{"fingerprint:fingerprint", "generation_id:generation_id"}; !slices.Equal(got, want) {
		t.Fatalf("occurrence_ratings parent key = %q, want %q", got, want)
	}
	if got, want := puzzlePrimaryKeyColumns(t, store.Reader, "occurrence_themes"), []string{"generation_id", "theme", "fingerprint"}; !slices.Equal(got, want) {
		t.Fatalf("occurrence_themes primary key = %q, want %q", got, want)
	}
	for table, want := range wantColumns {
		if got := pragmaNames(t, store.Reader, `PRAGMA table_info("`+table+`")`, 1); !slices.Equal(got, want) {
			t.Fatalf("table %s columns = %q, want %q", table, got, want)
		}
	}

	var versions []int
	rows, err := store.Reader.Query(`SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			t.Fatal(err)
		}
		versions = append(versions, version)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(versions, []int{4}) {
		t.Fatalf("schema versions = %v, want [4]", versions)
	}

	var journalMode string
	if err := store.Reader.QueryRow(`PRAGMA journal_mode`).Scan(&journalMode); err != nil {
		t.Fatal(err)
	}
	if journalMode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", journalMode)
	}

	assertGenerationSchemaConstraints(t, store.Writer)
}

func TestProbeRecognizesExactCurrentV4(t *testing.T) {
	path := filepath.Join(t.TempDir(), "puzzles.sqlite")
	openTestGenerationPuzzleStore(t, path)

	state, err := ProbePuzzleStore(path)
	if err != nil {
		t.Fatal(err)
	}
	if !state.Exists || state.Legacy || state.Format != CurrentPuzzleSchemaVersion {
		t.Fatalf("ProbePuzzleStore() = %+v, want existing current v%d", state, CurrentPuzzleSchemaVersion)
	}

	if legacy, err := OpenLegacyPuzzleReadOnly(path); err == nil {
		legacy.Close()
		t.Fatal("OpenLegacyPuzzleReadOnly() accepted the current catalogue")
	}
}

func TestOpenGenerationPuzzleStoreRejectsExistingEmptyFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "puzzles.sqlite")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}

	store, err := OpenPuzzleStore(path)
	if err == nil {
		store.Close()
		t.Fatal("OpenGenerationPuzzleStore() accepted an existing empty file")
	}
	info, statErr := os.Stat(path)
	if statErr != nil {
		t.Fatal(statErr)
	}
	if info.Size() != 0 {
		t.Fatalf("existing empty file grew to %d bytes", info.Size())
	}
}

func TestPuzzleStoreEscapesSpecialPathCharacters(t *testing.T) {
	path := filepath.Join(t.TempDir(), "puzzles #?.sqlite")
	store := openTestGenerationPuzzleStore(t, path)

	if _, err := store.Writer.Exec(`INSERT INTO sources(source_id, kind) VALUES ('special', 'test')`); err != nil {
		t.Fatal(err)
	}
	var kind string
	if err := store.Reader.QueryRow(`SELECT kind FROM sources WHERE source_id = 'special'`).Scan(&kind); err != nil {
		t.Fatal(err)
	}
	if kind != "test" {
		t.Fatalf("kind = %q, want test", kind)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("escaped database path was not created: %v", err)
	}
}

func TestPuzzleStoreAppliesPragmasToEveryConnection(t *testing.T) {
	store := openTestGenerationPuzzleStore(t, filepath.Join(t.TempDir(), "puzzles.sqlite"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	type connectionResult struct {
		connection *sql.Conn
		err        error
	}
	results := make(chan connectionResult, 4)
	var wait sync.WaitGroup
	for range 4 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			connection, err := store.Reader.Conn(ctx)
			results <- connectionResult{connection: connection, err: err}
		}()
	}
	wait.Wait()
	close(results)

	connections := make([]*sql.Conn, 0, 4)
	for result := range results {
		if result.err != nil {
			t.Fatal(result.err)
		}
		connections = append(connections, result.connection)
	}
	defer func() {
		for _, connection := range connections {
			connection.Close()
		}
	}()
	if len(connections) != 4 {
		t.Fatalf("connections = %d, want 4", len(connections))
	}

	for index, connection := range connections {
		assertConnectionPragmas(t, connection, index)
	}
	writer, err := store.Writer.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	assertConnectionPragmas(t, writer, -1)
}

func TestPuzzleStoreWriterUsesBoundedImportPragmas(t *testing.T) {
	store := openTestGenerationPuzzleStore(t, filepath.Join(t.TempDir(), "puzzles.sqlite"))
	var cacheSize, walAutoCheckpoint, synchronous int
	if err := store.Writer.QueryRow(`PRAGMA cache_size`).Scan(&cacheSize); err != nil {
		t.Fatal(err)
	}
	if err := store.Writer.QueryRow(`PRAGMA wal_autocheckpoint`).Scan(&walAutoCheckpoint); err != nil {
		t.Fatal(err)
	}
	if err := store.Writer.QueryRow(`PRAGMA synchronous`).Scan(&synchronous); err != nil {
		t.Fatal(err)
	}
	if cacheSize != -65_536 {
		t.Fatalf("writer cache_size = %d, want -65536 (64 MiB)", cacheSize)
	}
	if walAutoCheckpoint != 16_384 {
		t.Fatalf("writer wal_autocheckpoint = %d, want 16384 pages", walAutoCheckpoint)
	}
	if synchronous != 1 {
		t.Fatalf("writer synchronous = %d, want 1 (NORMAL)", synchronous)
	}
}

func TestPuzzleStoreReaderRejectsWrites(t *testing.T) {
	store := openTestGenerationPuzzleStore(t, filepath.Join(t.TempDir(), "puzzles.sqlite"))

	if _, err := store.Reader.Exec(`INSERT INTO sources(source_id, kind) VALUES ('reader', 'test')`); err == nil {
		t.Fatal("read-only handle accepted a write")
	}
	var count int
	if err := store.Writer.QueryRow(`SELECT COUNT(*) FROM sources WHERE source_id = 'reader'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("reader write created %d rows", count)
	}
}

func TestPuzzleStoreReadsDuringRealUncommittedWrite(t *testing.T) {
	store := openTestGenerationPuzzleStore(t, filepath.Join(t.TempDir(), "puzzles.sqlite"))
	if _, err := store.Writer.Exec(`INSERT INTO sources(source_id, kind) VALUES ('committed', 'test')`); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	tx, err := store.Writer.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO sources(source_id, kind) VALUES ('pending', 'test')`); err != nil {
		t.Fatal(err)
	}

	var committed, pending int
	if err := store.Reader.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FILTER (WHERE source_id = 'committed'),
		        COUNT(*) FILTER (WHERE source_id = 'pending')
		 FROM sources`,
	).Scan(&committed, &pending); err != nil {
		t.Fatalf("read during uncommitted write: %v", err)
	}
	if committed != 1 || pending != 0 {
		t.Fatalf("reader saw committed=%d pending=%d, want 1 and 0", committed, pending)
	}
}

func openTestGenerationPuzzleStore(t *testing.T, path string) *PuzzleStore {
	t.Helper()
	store, err := OpenPuzzleStore(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close puzzle store: %v", err)
		}
	})
	return store
}

func schemaObjectNames(t *testing.T, db *sql.DB, objectType string) []string {
	t.Helper()
	rows, err := db.Query(
		`SELECT name
		 FROM sqlite_master
		 WHERE type = ? AND name NOT LIKE 'sqlite_autoindex_%'
		 ORDER BY name`,
		objectType,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return names
}

func pragmaNames(t *testing.T, db *sql.DB, query string, nameColumn int) []string {
	t.Helper()
	rows, err := db.Query(query)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		t.Fatal(err)
	}
	values := make([]any, len(columns))
	destinations := make([]any, len(columns))
	for index := range values {
		destinations[index] = &values[index]
	}
	var names []string
	for rows.Next() {
		if err := rows.Scan(destinations...); err != nil {
			t.Fatal(err)
		}
		name, ok := values[nameColumn].(string)
		if !ok {
			t.Fatalf("column %d from %q has type %T", nameColumn, query, values[nameColumn])
		}
		names = append(names, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return names
}

func puzzlePrimaryKeyColumns(t *testing.T, db *sql.DB, table string) []string {
	t.Helper()
	rows, err := db.Query(`SELECT name FROM pragma_table_info(?) WHERE pk > 0 ORDER BY pk`, table)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var columns []string
	for rows.Next() {
		var column string
		if err := rows.Scan(&column); err != nil {
			t.Fatal(err)
		}
		columns = append(columns, column)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return columns
}

func puzzleForeignKeyColumns(t *testing.T, db *sql.DB, table, parent string) []string {
	t.Helper()
	rows, err := db.Query(`SELECT "from", "to"
		FROM pragma_foreign_key_list(?) WHERE "table" = ? ORDER BY seq`, table, parent)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var columns []string
	for rows.Next() {
		var from, to string
		if err := rows.Scan(&from, &to); err != nil {
			t.Fatal(err)
		}
		columns = append(columns, from+":"+to)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return columns
}

func assertPuzzleTableWithoutRowID(t *testing.T, db *sql.DB, table string, want bool) {
	t.Helper()
	var statement string
	if err := db.QueryRow(
		`SELECT sql FROM sqlite_master WHERE type = 'table' AND name = ?`,
		table,
	).Scan(&statement); err != nil {
		t.Fatal(err)
	}
	if got := strings.Contains(strings.ToUpper(statement), "WITHOUT ROWID"); got != want {
		t.Fatalf("table %s WITHOUT ROWID = %t, want %t; sql=%s", table, got, want, statement)
	}
}

func assertConnectionPragmas(t *testing.T, connection *sql.Conn, index int) {
	t.Helper()
	ctx := context.Background()
	var foreignKeys, busyTimeout int
	if err := connection.QueryRowContext(ctx, `PRAGMA foreign_keys`).Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if err := connection.QueryRowContext(ctx, `PRAGMA busy_timeout`).Scan(&busyTimeout); err != nil {
		t.Fatal(err)
	}
	if foreignKeys != 1 || busyTimeout != 5000 {
		t.Fatalf("connection %d pragmas: foreign_keys=%d busy_timeout=%d", index, foreignKeys, busyTimeout)
	}
}

func assertGenerationSchemaConstraints(t *testing.T, db *sql.DB) {
	t.Helper()
	mustPuzzleStoreExec(t, db, `INSERT INTO sources(source_id, kind) VALUES ('one', 'test'), ('two', 'test')`)
	expectPuzzleStoreExecError(t, db, `INSERT INTO source_generations(
		generation_id, source_id, status, source_path, started_at
	) VALUES ('bad-status', 'one', 'active', '/tmp/input', 1)`)
	expectPuzzleStoreExecError(t, db, `INSERT INTO source_generations(
		generation_id, source_id, status, source_path, started_at
	) VALUES ('bad-start', 'one', 'building', '/tmp/input', 0)`)
	expectPuzzleStoreExecError(t, db, `INSERT INTO source_generations(
		generation_id, source_id, status, source_path, started_at, sealed_at
	) VALUES ('bad-seal', 'one', 'sealed', '/tmp/input', 1, 0)`)
	expectPuzzleStoreExecError(t, db, `INSERT INTO source_generations(
		generation_id, source_id, status, source_path, started_at, maximum_solution_plies
	) VALUES ('bad-maximum', 'one', 'building', '/tmp/input', 1, -1)`)
	mustPuzzleStoreExec(t, db, `INSERT INTO source_generations(
		generation_id, source_id, status, source_path, started_at
	) VALUES ('generation', 'one', 'building', '/tmp/input', 1)`)
	expectPuzzleStoreExecError(t, db, `INSERT INTO source_heads(source_id, generation_id)
		VALUES ('two', 'generation')`)

	expectPuzzleStoreExecError(t, db, `INSERT INTO puzzle_cores(
		fingerprint, displayed_fen, solver, solution_json, solution_plies
	) VALUES ('bad-solver', 'fen', 'green', '[]', 1)`)
	expectPuzzleStoreExecError(t, db, `INSERT INTO puzzle_cores(
		fingerprint, displayed_fen, solver, solution_json, solution_plies
	) VALUES ('bad-plies', 'fen', 'white', '[]', 0)`)
	mustPuzzleStoreExec(t, db, `INSERT INTO puzzle_cores(
		fingerprint, displayed_fen, solver, solution_json, solution_plies
	) VALUES ('core', 'fen', 'white', '[]', 1)`)
	expectPuzzleStoreExecError(t, db, `INSERT INTO puzzle_occurrences(
		generation_id, fingerprint, metadata_json, ordinal
	) VALUES ('generation', 'core', '{}', 0)`)
	mustPuzzleStoreExec(t, db, `INSERT INTO puzzle_occurrences(
		generation_id, fingerprint, metadata_json, ordinal
	) VALUES ('generation', 'core', '{}', 1)`)
	var themesJSON string
	if err := db.QueryRow(`SELECT themes_json FROM puzzle_occurrences
		WHERE generation_id = 'generation' AND fingerprint = 'core'`).Scan(&themesJSON); err != nil {
		t.Fatal(err)
	}
	if themesJSON != "[]" {
		t.Fatalf("default themes_json = %q, want []", themesJSON)
	}
	expectPuzzleStoreExecError(t, db, `INSERT INTO occurrence_ratings(generation_id, rating_key, fingerprint)
		VALUES ('generation', 1500, 'missing')`)
	mustPuzzleStoreExec(t, db, `INSERT INTO occurrence_ratings(generation_id, rating_key, fingerprint)
		VALUES ('generation', -9223372036854775808, 'core')`)
	var sourceRating sql.NullInt64
	if err := db.QueryRow(`SELECT rating FROM puzzle_occurrences
		WHERE generation_id = 'generation' AND fingerprint = 'core'`).Scan(&sourceRating); err != nil {
		t.Fatal(err)
	}
	if sourceRating.Valid {
		t.Fatalf("source rating = %d, want NULL while rating_key stores the NULL sentinel", sourceRating.Int64)
	}
	expectPuzzleStoreExecError(t, db, `INSERT INTO occurrence_themes(generation_id, fingerprint, theme)
		VALUES ('generation', 'missing', 'fork')`)
	mustPuzzleStoreExec(t, db, `INSERT INTO occurrence_themes(generation_id, fingerprint, theme)
		VALUES ('generation', 'core', 'fork')`)
	expectPuzzleStoreExecError(t, db, `INSERT INTO generation_themes(generation_id, theme)
		VALUES ('missing', 'fork')`)
	mustPuzzleStoreExec(t, db, `INSERT INTO generation_themes(generation_id, theme)
		VALUES ('generation', 'fork')`)

	mustPuzzleStoreExec(t, db, `DELETE FROM source_generations WHERE generation_id = 'generation'`)
	for _, table := range []string{"puzzle_occurrences", "occurrence_ratings", "occurrence_themes", "generation_themes"} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s rows after generation cascade = %d, want 0", table, count)
		}
	}
}

func mustPuzzleStoreExec(t *testing.T, db *sql.DB, statement string) {
	t.Helper()
	if _, err := db.Exec(statement); err != nil {
		t.Fatal(err)
	}
}

func expectPuzzleStoreExecError(t *testing.T, db *sql.DB, statement string) {
	t.Helper()
	if _, err := db.Exec(statement); err == nil {
		t.Fatalf("statement unexpectedly succeeded: %s", statement)
	}
}
