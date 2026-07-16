package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const CurrentPuzzleSchemaVersion = 3

type PuzzleStore struct {
	Reader *sql.DB
	Writer *sql.DB
}

func OpenPuzzleStore(path string) (*PuzzleStore, error) {
	existed, err := pathExists(path)
	if err != nil {
		return nil, err
	}
	if existed {
		state, err := ProbePuzzleStore(path)
		if err != nil {
			return nil, err
		}
		if !state.Exists || state.Legacy || state.Format != CurrentPuzzleSchemaVersion {
			return nil, fmt.Errorf(
				"refuse to open puzzle database %s: expected current schema version %d",
				path,
				CurrentPuzzleSchemaVersion,
			)
		}
	}

	writerDSN, err := puzzleStoreDSN(path, false, !existed)
	if err != nil {
		return nil, err
	}
	writer, err := sql.Open("sqlite", writerDSN)
	if err != nil {
		return nil, fmt.Errorf("open generation puzzle writer: %w", err)
	}
	writer.SetMaxOpenConns(1)
	writer.SetMaxIdleConns(1)
	if err := writer.Ping(); err != nil {
		writer.Close()
		return nil, fmt.Errorf("connect generation puzzle writer: %w", err)
	}

	if err := Migrate(writer, "puzzles"); err != nil {
		writer.Close()
		return nil, fmt.Errorf("migrate puzzle database: %w", err)
	}
	if err := validatePuzzleSchema(writer); err != nil {
		writer.Close()
		return nil, err
	}
	if err := enableWAL(writer); err != nil {
		writer.Close()
		return nil, err
	}

	readerDSN, err := puzzleStoreDSN(path, true, false)
	if err != nil {
		writer.Close()
		return nil, err
	}
	reader, err := sql.Open("sqlite", readerDSN)
	if err != nil {
		writer.Close()
		return nil, fmt.Errorf("open generation puzzle reader: %w", err)
	}
	reader.SetMaxOpenConns(4)
	reader.SetMaxIdleConns(4)
	if err := reader.Ping(); err != nil {
		reader.Close()
		writer.Close()
		return nil, fmt.Errorf("connect generation puzzle reader: %w", err)
	}

	return &PuzzleStore{Reader: reader, Writer: writer}, nil
}

func (s *PuzzleStore) Close() error {
	return errors.Join(s.Reader.Close(), s.Writer.Close())
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, os.ErrNotExist):
		return false, nil
	default:
		return false, fmt.Errorf("inspect generation puzzle store: %w", err)
	}
}

func puzzleStoreDSN(path string, readOnly, create bool) (string, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve generation puzzle store path: %w", err)
	}
	uri := url.URL{
		Scheme: "file",
		Path:   filepath.ToSlash(absolutePath),
	}
	query := uri.Query()
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "foreign_keys(ON)")
	if !readOnly {
		query.Add("_pragma", "cache_size(-65536)")
		query.Add("_pragma", "wal_autocheckpoint(16384)")
		query.Add("_pragma", "synchronous(NORMAL)")
	}
	switch {
	case readOnly:
		query.Set("mode", "ro")
	case create:
		query.Set("mode", "rwc")
	default:
		query.Set("mode", "rw")
	}
	uri.RawQuery = query.Encode()
	return uri.String(), nil
}

func validatePuzzleSchema(db *sql.DB) error {
	var count, minimum, maximum int
	if err := db.QueryRow(
		`SELECT COUNT(*), COALESCE(MIN(version), 0), COALESCE(MAX(version), 0)
		 FROM schema_migrations`,
	).Scan(&count, &minimum, &maximum); err != nil {
		return fmt.Errorf("validate generation puzzle schema: %w", err)
	}
	if count != 1 || minimum != CurrentPuzzleSchemaVersion || maximum != CurrentPuzzleSchemaVersion {
		return fmt.Errorf(
			"validate generation puzzle schema: migrations are count=%d range=%d..%d, want only version %d",
			count,
			minimum,
			maximum,
			CurrentPuzzleSchemaVersion,
		)
	}
	actualSchema, err := readPuzzleSchema(db)
	if err != nil {
		return fmt.Errorf("validate generation puzzle logical schema: %w", err)
	}
	if !equalPuzzleSchemas(actualSchema, currentPuzzleSchema) {
		return errors.New("validate generation puzzle logical schema: schema is not exact")
	}
	if err := validateCurrentPuzzlePhysicalSchema(db); err != nil {
		return fmt.Errorf("validate generation puzzle physical schema: %w", err)
	}
	return nil
}

func enableWAL(db *sql.DB) error {
	var mode string
	if err := db.QueryRow(`PRAGMA journal_mode=WAL`).Scan(&mode); err != nil {
		return fmt.Errorf("enable generation puzzle WAL: %w", err)
	}
	if mode != "wal" {
		return fmt.Errorf("enable generation puzzle WAL: journal mode is %q", mode)
	}
	return nil
}
