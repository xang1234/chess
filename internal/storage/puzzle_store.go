package storage

import (
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

//go:embed puzzle_schema_v3.sql
var puzzleSchemaV3 string

const CurrentPuzzleSchemaVersion = 3

type PuzzleStore struct {
	Reader *sql.DB
	Writer *sql.DB
}

func OpenGenerationPuzzleStore(path string) (*PuzzleStore, error) {
	existed, err := pathExists(path)
	if err != nil {
		return nil, err
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

	if !existed {
		if err := bootstrapGenerationPuzzleSchema(writer); err != nil {
			writer.Close()
			return nil, err
		}
	}
	if err := validateGenerationPuzzleSchema(writer); err != nil {
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

func bootstrapGenerationPuzzleSchema(db *sql.DB) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin generation puzzle bootstrap: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(puzzleSchemaV3); err != nil {
		return fmt.Errorf("bootstrap generation puzzle schema: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit generation puzzle bootstrap: %w", err)
	}
	return nil
}

func validateGenerationPuzzleSchema(db *sql.DB) error {
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
