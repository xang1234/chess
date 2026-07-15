package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"chess-trainer/internal/domain"
)

type PuzzleFileState struct {
	Exists bool
	Legacy bool
	Format int
}

type PuzzleSchemaVersionError struct {
	Path      string
	Found     int
	Supported int
}

func (e *PuzzleSchemaVersionError) Error() string {
	return fmt.Sprintf(
		"puzzle database %s uses schema version %d, but this application supports version %d",
		e.Path,
		e.Found,
		e.Supported,
	)
}

func ProbePuzzleStore(path string) (PuzzleFileState, error) {
	state, db, err := probePuzzleStoreOpen(path)
	return state, closePuzzleProbe(path, db, err)
}

func OpenLegacyPuzzleReadOnly(path string) (*sql.DB, error) {
	state, db, err := probePuzzleStoreOpen(path)
	if err != nil {
		return nil, closePuzzleProbe(path, db, err)
	}
	if !state.Exists || !state.Legacy {
		return nil, closePuzzleProbe(
			path,
			db,
			fmt.Errorf("puzzle database %s is not a recognized legacy catalogue", path),
		)
	}
	return db, nil
}

func RemoveRecognizedLegacyPuzzleStore(path string) error {
	removal, err := prepareRecognizedLegacyRemoval(path)
	if err != nil {
		return err
	}
	return removal.remove()
}

func probePuzzleStoreOpen(path string) (PuzzleFileState, *sql.DB, error) {
	state := PuzzleFileState{}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return state, nil, nil
	} else if err != nil {
		return state, nil, fmt.Errorf("inspect puzzle database %s: %w", path, err)
	}
	state.Exists = true

	db, err := openPuzzleReadOnly(path)
	if err != nil {
		return state, nil, err
	}

	versions, maximum, err := puzzleMigrationVersions(db)
	state.Format = maximum
	if err != nil {
		return state, db, fmt.Errorf("probe puzzle database %s migrations: %w", path, err)
	}
	if maximum > CurrentPuzzleSchemaVersion {
		return state, db, &PuzzleSchemaVersionError{
			Path:      path,
			Found:     maximum,
			Supported: CurrentPuzzleSchemaVersion,
		}
	}

	wantSchema, legacy, ok := recognizedPuzzleSchema(versions)
	if !ok {
		return state, db, fmt.Errorf("puzzle database %s has an unrecognized migration set %v", path, versions)
	}
	actualSchema, err := readPuzzleSchema(db)
	if err != nil {
		return state, db, fmt.Errorf("probe puzzle database %s schema: %w", path, err)
	}
	if !equalPuzzleSchemas(actualSchema, wantSchema) {
		return state, db, fmt.Errorf("puzzle database %s does not match the exact schema for format %d", path, maximum)
	}

	state.Legacy = legacy
	return state, db, nil
}

func closePuzzleProbe(path string, db *sql.DB, probeErr error) error {
	if db == nil {
		return probeErr
	}
	if err := db.Close(); err != nil {
		return errors.Join(probeErr, fmt.Errorf("close puzzle database probe %s: %w", path, err))
	}
	return probeErr
}

type recognizedLegacyRemoval struct {
	path string
}

func prepareRecognizedLegacyRemoval(path string) (*recognizedLegacyRemoval, error) {
	state, db, err := probePuzzleStoreOpen(path)
	err = closePuzzleProbe(path, db, err)
	if err != nil {
		return nil, err
	}
	if !state.Exists || !state.Legacy {
		return nil, fmt.Errorf("refuse to remove puzzle database %s: not a recognized legacy catalogue", path)
	}
	return &recognizedLegacyRemoval{path: path}, nil
}

func (r *recognizedLegacyRemoval) remove() error {
	quarantine, err := quarantinePuzzleStore(r.path)
	if err != nil {
		return err
	}

	state, probeErr := ProbePuzzleStore(quarantine.path)
	if probeErr != nil || !state.Exists || !state.Legacy {
		if probeErr == nil {
			probeErr = fmt.Errorf("quarantined database is not a recognized legacy catalogue")
		}
		restoreErr := quarantine.restore()
		return errors.Join(
			fmt.Errorf("refuse to remove puzzle database %s after path replacement: %w", r.path, probeErr),
			restoreErr,
		)
	}
	return quarantine.discard()
}

type quarantinedPuzzleStore struct {
	originalPath string
	directory    string
	path         string
	moved        []string
}

func quarantinePuzzleStore(path string) (*quarantinedPuzzleStore, error) {
	directory, err := os.MkdirTemp(
		filepath.Dir(path),
		"."+filepath.Base(path)+".removing-",
	)
	if err != nil {
		return nil, fmt.Errorf("create puzzle removal quarantine for %s: %w", path, err)
	}
	quarantine := &quarantinedPuzzleStore{
		originalPath: path,
		directory:    directory,
		path:         filepath.Join(directory, filepath.Base(path)),
	}
	for _, suffix := range []string{"", "-wal", "-shm"} {
		from := path + suffix
		to := quarantine.path + suffix
		if err := os.Rename(from, to); errors.Is(err, os.ErrNotExist) && suffix != "" {
			continue
		} else if err != nil {
			restoreErr := quarantine.restore()
			return nil, errors.Join(
				fmt.Errorf("quarantine puzzle database file %s: %w", from, err),
				restoreErr,
			)
		}
		quarantine.moved = append(quarantine.moved, suffix)
	}
	return quarantine, nil
}

func (q *quarantinedPuzzleStore) restore() error {
	var restoreErrors []error
	for index := len(q.moved) - 1; index >= 0; index-- {
		suffix := q.moved[index]
		from := q.path + suffix
		to := q.originalPath + suffix
		if _, err := os.Lstat(to); err == nil {
			restoreErrors = append(
				restoreErrors,
				fmt.Errorf("restore quarantined puzzle file %s: destination reappeared", to),
			)
			continue
		} else if !errors.Is(err, os.ErrNotExist) {
			restoreErrors = append(restoreErrors, fmt.Errorf("inspect puzzle restore destination %s: %w", to, err))
			continue
		}
		if err := os.Rename(from, to); err != nil {
			restoreErrors = append(restoreErrors, fmt.Errorf("restore quarantined puzzle file %s: %w", to, err))
		}
	}
	if len(restoreErrors) != 0 {
		return errors.Join(restoreErrors...)
	}
	if err := os.Remove(q.directory); err != nil {
		return fmt.Errorf("remove puzzle quarantine directory %s: %w", q.directory, err)
	}
	return nil
}

func (q *quarantinedPuzzleStore) discard() error {
	if err := os.Remove(q.path); err != nil {
		return fmt.Errorf("remove recognized legacy puzzle database %s: %w", q.originalPath, err)
	}
	for _, suffix := range []string{"-wal", "-shm"} {
		if err := os.Remove(q.path + suffix); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove recognized legacy puzzle sidecar %s: %w", q.originalPath+suffix, err)
		}
	}
	if err := os.Remove(q.directory); err != nil {
		return fmt.Errorf("remove puzzle quarantine directory %s: %w", q.directory, err)
	}
	return nil
}

const legacySnapshotBatchSize = 128

type legacySnapshotKey struct {
	fingerprint string
	sourceID    string
}

type legacyPuzzleSnapshot struct {
	key        legacySnapshotKey
	sourceKind string
	rating     sql.NullInt64
	themesJSON string
	sourceFEN  sql.NullString
	preludeUCI sql.NullString
}

func BackfillLegacyPuzzleSnapshots(
	ctx context.Context,
	userDB *sql.DB,
	legacyPuzzles *sql.DB,
) error {
	var cursor legacySnapshotKey
	hasCursor := false
	for {
		keys, err := readLegacySnapshotKeys(ctx, userDB, cursor, hasCursor)
		if err != nil {
			return err
		}
		if len(keys) == 0 {
			return nil
		}
		cursor = keys[len(keys)-1]
		hasCursor = true

		snapshots, err := readLegacyPuzzleSnapshots(ctx, legacyPuzzles, keys)
		if err != nil {
			return err
		}
		if len(snapshots) != 0 {
			if err := writeLegacyPuzzleSnapshots(ctx, userDB, keys, snapshots); err != nil {
				return err
			}
		}
		if len(keys) < legacySnapshotBatchSize {
			return nil
		}
	}
}

func readLegacySnapshotKeys(
	ctx context.Context,
	userDB *sql.DB,
	cursor legacySnapshotKey,
	hasCursor bool,
) ([]legacySnapshotKey, error) {
	rows, err := userDB.QueryContext(
		ctx,
		`SELECT fingerprint, source_id
		 FROM (
		   SELECT fingerprint, source_id
		   FROM session_items
		   WHERE source_kind IS NULL
		      OR rating_snapshot IS NULL
		      OR themes_json IS NULL
		      OR source_fen_snapshot IS NULL
		      OR prelude_uci_snapshot IS NULL
		   UNION
		   SELECT fingerprint, source_id
		   FROM attempts
		   WHERE source_kind IS NULL
		      OR rating_snapshot IS NULL
		      OR themes_json IS NULL
		 ) AS history_keys
		 WHERE ? = 0
		    OR fingerprint > ?
		    OR (fingerprint = ? AND source_id > ?)
		 ORDER BY fingerprint, source_id
		 LIMIT ?`,
		boolAsInteger(hasCursor),
		cursor.fingerprint,
		cursor.fingerprint,
		cursor.sourceID,
		legacySnapshotBatchSize,
	)
	if err != nil {
		return nil, fmt.Errorf("read legacy snapshot keys: %w", err)
	}
	keys := make([]legacySnapshotKey, 0, legacySnapshotBatchSize)
	for rows.Next() {
		var key legacySnapshotKey
		if err := rows.Scan(&key.fingerprint, &key.sourceID); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan legacy snapshot key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close legacy snapshot keys: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate legacy snapshot keys: %w", err)
	}
	return keys, nil
}

func legacySnapshotLookupQuery(keys []legacySnapshotKey) (string, []any) {
	var query strings.Builder
	query.WriteString(`WITH requested(fingerprint, source_id) AS (VALUES `)
	args := make([]any, 0, len(keys)*2)
	for index, key := range keys {
		if index != 0 {
			query.WriteByte(',')
		}
		query.WriteString(`(?, ?)`)
		args = append(args, key.fingerprint, key.sourceID)
	}
	query.WriteString(`)
		SELECT requested.fingerprint, requested.source_id, sources.kind,
		       puzzle_sources.rating, puzzles.source_fen, puzzles.prelude_uci,
		       puzzle_themes.theme
		FROM requested
		JOIN puzzle_sources
		  ON puzzle_sources.fingerprint = requested.fingerprint
		 AND puzzle_sources.source_id = requested.source_id
		JOIN sources ON sources.source_id = puzzle_sources.source_id
		JOIN puzzles ON puzzles.fingerprint = puzzle_sources.fingerprint
		LEFT JOIN puzzle_themes
		  ON puzzle_themes.fingerprint = puzzle_sources.fingerprint
		 AND puzzle_themes.source_id = puzzle_sources.source_id
		ORDER BY requested.fingerprint, requested.source_id, puzzle_themes.theme`)
	return query.String(), args
}

func readLegacyPuzzleSnapshots(
	ctx context.Context,
	legacyPuzzles *sql.DB,
	keys []legacySnapshotKey,
) (map[legacySnapshotKey]legacyPuzzleSnapshot, error) {
	query, args := legacySnapshotLookupQuery(keys)
	rows, err := legacyPuzzles.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read legacy puzzle snapshots: %w", err)
	}

	type snapshotBuilder struct {
		snapshot legacyPuzzleSnapshot
		themes   []string
	}
	builders := make(map[legacySnapshotKey]*snapshotBuilder, len(keys))
	for rows.Next() {
		var snapshot legacyPuzzleSnapshot
		var theme sql.NullString
		if err := rows.Scan(
			&snapshot.key.fingerprint,
			&snapshot.key.sourceID,
			&snapshot.sourceKind,
			&snapshot.rating,
			&snapshot.sourceFEN,
			&snapshot.preludeUCI,
			&theme,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan legacy puzzle snapshot: %w", err)
		}
		builder := builders[snapshot.key]
		if builder == nil {
			builder = &snapshotBuilder{snapshot: snapshot, themes: []string{}}
			builders[snapshot.key] = builder
		}
		if theme.Valid {
			builder.themes = append(builder.themes, theme.String)
		}
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close legacy puzzle snapshots: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate legacy puzzle snapshots: %w", err)
	}

	snapshots := make(map[legacySnapshotKey]legacyPuzzleSnapshot, len(builders))
	for key, builder := range builders {
		themesJSON, err := json.Marshal(domain.NormalizeThemes(builder.themes))
		if err != nil {
			return nil, fmt.Errorf("encode legacy themes for %s/%s: %w", key.fingerprint, key.sourceID, err)
		}
		builder.snapshot.themesJSON = string(themesJSON)
		snapshots[key] = builder.snapshot
	}
	return snapshots, nil
}

func writeLegacyPuzzleSnapshots(
	ctx context.Context,
	userDB *sql.DB,
	keys []legacySnapshotKey,
	snapshots map[legacySnapshotKey]legacyPuzzleSnapshot,
) error {
	tx, err := userDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin legacy puzzle snapshot backfill: %w", err)
	}
	defer tx.Rollback()

	updateSessionItem, err := tx.PrepareContext(ctx, `UPDATE session_items SET
		source_kind = COALESCE(source_kind, ?1),
		rating_snapshot = COALESCE(rating_snapshot, ?2),
		themes_json = COALESCE(themes_json, ?3),
		source_fen_snapshot = COALESCE(source_fen_snapshot, ?4),
		prelude_uci_snapshot = COALESCE(prelude_uci_snapshot, ?5)
		WHERE fingerprint = ?6 AND source_id = ?7
		  AND ((source_kind IS NULL AND ?1 IS NOT NULL)
		    OR (rating_snapshot IS NULL AND ?2 IS NOT NULL)
		    OR (themes_json IS NULL AND ?3 IS NOT NULL)
		    OR (source_fen_snapshot IS NULL AND ?4 IS NOT NULL)
		    OR (prelude_uci_snapshot IS NULL AND ?5 IS NOT NULL))`)
	if err != nil {
		return fmt.Errorf("prepare session snapshot backfill: %w", err)
	}
	defer updateSessionItem.Close()

	updateAttempt, err := tx.PrepareContext(ctx, `UPDATE attempts SET
		source_kind = COALESCE(source_kind, ?1),
		rating_snapshot = COALESCE(rating_snapshot, ?2),
		themes_json = COALESCE(themes_json, ?3)
		WHERE fingerprint = ?4 AND source_id = ?5
		  AND ((source_kind IS NULL AND ?1 IS NOT NULL)
		    OR (rating_snapshot IS NULL AND ?2 IS NOT NULL)
		    OR (themes_json IS NULL AND ?3 IS NOT NULL))`)
	if err != nil {
		return fmt.Errorf("prepare attempt snapshot backfill: %w", err)
	}
	defer updateAttempt.Close()

	for _, key := range keys {
		snapshot, ok := snapshots[key]
		if !ok {
			continue
		}
		if _, err := updateSessionItem.ExecContext(
			ctx,
			snapshot.sourceKind,
			snapshot.rating,
			snapshot.themesJSON,
			snapshot.sourceFEN,
			snapshot.preludeUCI,
			key.fingerprint,
			key.sourceID,
		); err != nil {
			return fmt.Errorf("backfill session snapshot for %s/%s: %w", key.fingerprint, key.sourceID, err)
		}
		if _, err := updateAttempt.ExecContext(
			ctx,
			snapshot.sourceKind,
			snapshot.rating,
			snapshot.themesJSON,
			key.fingerprint,
			key.sourceID,
		); err != nil {
			return fmt.Errorf("backfill attempt snapshot for %s/%s: %w", key.fingerprint, key.sourceID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit legacy puzzle snapshot backfill: %w", err)
	}
	return nil
}

func boolAsInteger(value bool) int {
	if value {
		return 1
	}
	return 0
}

func openPuzzleReadOnly(path string) (*sql.DB, error) {
	dsn, err := puzzleStoreDSN(path, true, false)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open puzzle database %s read-only: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return db, nil
}

func puzzleMigrationVersions(db *sql.DB) ([]int, int, error) {
	var count, maximum int
	if err := db.QueryRow(
		`SELECT COUNT(*), COALESCE(MAX(version), 0) FROM schema_migrations`,
	).Scan(&count, &maximum); err != nil {
		return nil, 0, err
	}
	rows, err := db.Query(`SELECT version FROM schema_migrations ORDER BY version LIMIT 4`)
	if err != nil {
		return nil, maximum, err
	}
	capacity := count
	if capacity > 4 {
		capacity = 4
	}
	versions := make([]int, 0, capacity)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			rows.Close()
			return nil, maximum, err
		}
		versions = append(versions, version)
	}
	if err := rows.Close(); err != nil {
		return nil, maximum, err
	}
	if err := rows.Err(); err != nil {
		return nil, maximum, err
	}
	return versions, maximum, nil
}

type puzzleColumnSignature struct {
	Name       string
	Type       string
	NotNull    int
	Default    sql.NullString
	PrimaryKey int
	Hidden     int
}

type puzzleSchemaSignature map[string][]puzzleColumnSignature

func readPuzzleSchema(db *sql.DB) (puzzleSchemaSignature, error) {
	rows, err := db.Query(
		`SELECT name
		 FROM sqlite_master
		 WHERE type = 'table' AND lower(substr(name, 1, 7)) <> 'sqlite_'
		 ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			rows.Close()
			return nil, err
		}
		tables = append(tables, table)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	schema := make(puzzleSchemaSignature, len(tables))
	for _, table := range tables {
		columns, err := db.QueryContext(
			context.Background(),
			`SELECT name, type, "notnull", dflt_value, pk, hidden
			 FROM pragma_table_xinfo(?) ORDER BY cid`,
			table,
		)
		if err != nil {
			return nil, err
		}
		for columns.Next() {
			var column puzzleColumnSignature
			if err := columns.Scan(
				&column.Name,
				&column.Type,
				&column.NotNull,
				&column.Default,
				&column.PrimaryKey,
				&column.Hidden,
			); err != nil {
				columns.Close()
				return nil, err
			}
			schema[table] = append(schema[table], column)
		}
		if err := columns.Close(); err != nil {
			return nil, err
		}
		if err := columns.Err(); err != nil {
			return nil, err
		}
	}
	return schema, nil
}

func equalPuzzleSchemas(got, want puzzleSchemaSignature) bool {
	if len(got) != len(want) {
		return false
	}
	for table, wantColumns := range want {
		gotColumns, ok := got[table]
		if !ok || !slices.Equal(gotColumns, wantColumns) {
			return false
		}
	}
	return true
}

func recognizedPuzzleSchema(versions []int) (puzzleSchemaSignature, bool, bool) {
	switch {
	case slices.Equal(versions, []int{1}):
		return legacyPuzzleSchemaV1, true, true
	case slices.Equal(versions, []int{1, 2}):
		return legacyPuzzleSchemaV2, true, true
	case slices.Equal(versions, []int{CurrentPuzzleSchemaVersion}):
		return currentPuzzleSchema, false, true
	default:
		return nil, false, false
	}
}

func puzzleColumn(name, columnType string, notNull, primaryKey int) puzzleColumnSignature {
	return puzzleColumnSignature{
		Name:       name,
		Type:       columnType,
		NotNull:    notNull,
		PrimaryKey: primaryKey,
	}
}

func puzzleColumnWithDefault(
	name,
	columnType string,
	notNull,
	primaryKey int,
	defaultValue string,
) puzzleColumnSignature {
	column := puzzleColumn(name, columnType, notNull, primaryKey)
	column.Default = sql.NullString{String: defaultValue, Valid: true}
	return column
}

var legacyPuzzleCommonSchema = puzzleSchemaSignature{
	"schema_migrations": {
		puzzleColumn("version", "INTEGER", 0, 1),
	},
	"sources": {
		puzzleColumn("source_id", "TEXT", 0, 1),
		puzzleColumn("kind", "TEXT", 1, 0),
		puzzleColumn("imported_at", "INTEGER", 1, 0),
		puzzleColumn("source_path", "TEXT", 1, 0),
		puzzleColumn("checksum", "TEXT", 1, 0),
	},
	"puzzles": {
		puzzleColumn("fingerprint", "TEXT", 0, 1),
		puzzleColumn("source_fen", "TEXT", 0, 0),
		puzzleColumn("prelude_uci", "TEXT", 0, 0),
		puzzleColumn("displayed_fen", "TEXT", 1, 0),
		puzzleColumn("solver", "TEXT", 1, 0),
		puzzleColumn("solution_json", "TEXT", 1, 0),
		puzzleColumn("solution_plies", "INTEGER", 1, 0),
	},
	"puzzle_sources": {
		puzzleColumn("fingerprint", "TEXT", 1, 1),
		puzzleColumn("source_id", "TEXT", 1, 2),
		puzzleColumn("external_id", "TEXT", 0, 0),
		puzzleColumn("rating", "INTEGER", 0, 0),
		puzzleColumn("popularity", "INTEGER", 0, 0),
		puzzleColumn("play_count", "INTEGER", 0, 0),
		puzzleColumn("source_url", "TEXT", 0, 0),
		puzzleColumnWithDefault("metadata_json", "TEXT", 1, 0, "'{}'"),
	},
	"puzzle_themes": {
		puzzleColumn("fingerprint", "TEXT", 1, 1),
		puzzleColumn("source_id", "TEXT", 1, 2),
		puzzleColumn("theme", "TEXT", 1, 3),
	},
}

var legacyPuzzleSchemaV1 = withImportStagingSignature([]puzzleColumnSignature{
	puzzleColumn("import_id", "TEXT", 1, 1),
	puzzleColumn("ordinal", "INTEGER", 1, 2),
	puzzleColumn("puzzle_json", "TEXT", 1, 0),
})

var legacyPuzzleSchemaV2 = withImportStagingSignature([]puzzleColumnSignature{
	puzzleColumn("import_id", "TEXT", 1, 1),
	puzzleColumn("ordinal", "INTEGER", 1, 2),
	puzzleColumn("puzzle_json", "TEXT", 1, 0),
	puzzleColumn("fingerprint", "TEXT", 0, 0),
	puzzleColumn("source_fen", "TEXT", 0, 0),
	puzzleColumn("prelude_uci", "TEXT", 0, 0),
	puzzleColumn("displayed_fen", "TEXT", 0, 0),
	puzzleColumn("solver", "TEXT", 0, 0),
	puzzleColumn("solution_json", "TEXT", 0, 0),
	puzzleColumn("solution_plies", "INTEGER", 0, 0),
	puzzleColumn("external_id", "TEXT", 0, 0),
	puzzleColumn("rating", "INTEGER", 0, 0),
	puzzleColumn("popularity", "INTEGER", 0, 0),
	puzzleColumn("play_count", "INTEGER", 0, 0),
	puzzleColumn("source_url", "TEXT", 0, 0),
	puzzleColumn("metadata_json", "TEXT", 0, 0),
	puzzleColumn("themes_json", "TEXT", 0, 0),
})

func withImportStagingSignature(importStaging []puzzleColumnSignature) puzzleSchemaSignature {
	schema := make(puzzleSchemaSignature, len(legacyPuzzleCommonSchema)+1)
	for table, columns := range legacyPuzzleCommonSchema {
		schema[table] = columns
	}
	schema["import_staging"] = importStaging
	return schema
}

var currentPuzzleSchema = puzzleSchemaSignature{
	"schema_migrations": {
		puzzleColumn("version", "INTEGER", 0, 1),
	},
	"sources": {
		puzzleColumn("source_id", "TEXT", 0, 1),
		puzzleColumn("kind", "TEXT", 1, 0),
	},
	"source_generations": {
		puzzleColumn("generation_id", "TEXT", 0, 1),
		puzzleColumn("source_id", "TEXT", 1, 0),
		puzzleColumn("status", "TEXT", 1, 0),
		puzzleColumn("source_path", "TEXT", 1, 0),
		puzzleColumn("checksum", "TEXT", 0, 0),
		puzzleColumn("started_at", "INTEGER", 1, 0),
		puzzleColumn("sealed_at", "INTEGER", 0, 0),
	},
	"source_heads": {
		puzzleColumn("source_id", "TEXT", 0, 1),
		puzzleColumn("generation_id", "TEXT", 1, 0),
	},
	"puzzle_cores": {
		puzzleColumn("fingerprint", "TEXT", 0, 1),
		puzzleColumn("displayed_fen", "TEXT", 1, 0),
		puzzleColumn("solver", "TEXT", 1, 0),
		puzzleColumn("solution_json", "TEXT", 1, 0),
		puzzleColumn("solution_plies", "INTEGER", 1, 0),
	},
	"puzzle_occurrences": {
		puzzleColumn("generation_id", "TEXT", 1, 1),
		puzzleColumn("fingerprint", "TEXT", 1, 2),
		puzzleColumn("external_id", "TEXT", 0, 0),
		puzzleColumn("source_fen", "TEXT", 0, 0),
		puzzleColumn("prelude_uci", "TEXT", 0, 0),
		puzzleColumn("rating", "INTEGER", 0, 0),
		puzzleColumn("popularity", "INTEGER", 0, 0),
		puzzleColumn("play_count", "INTEGER", 0, 0),
		puzzleColumn("source_url", "TEXT", 0, 0),
		puzzleColumn("attribution", "TEXT", 0, 0),
		puzzleColumnWithDefault("metadata_json", "TEXT", 1, 0, "'{}'"),
		puzzleColumn("ordinal", "INTEGER", 1, 0),
	},
	"occurrence_themes": {
		puzzleColumn("generation_id", "TEXT", 1, 1),
		puzzleColumn("fingerprint", "TEXT", 1, 2),
		puzzleColumn("theme", "TEXT", 1, 3),
	},
}
