package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
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
	if !legacy {
		if err := validateCurrentPuzzlePhysicalSchema(db); err != nil {
			return state, db, fmt.Errorf(
				"puzzle database %s does not match the exact physical schema for format %d: %w",
				path,
				maximum,
				err,
			)
		}
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
