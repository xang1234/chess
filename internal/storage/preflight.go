package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// PreflightMigrations verifies an existing durable store without changing it.
// Pending migrations are rehearsed against a bounded temporary snapshot.
func PreflightMigrations(path, schema string) error {
	names, err := migrationNames(schema)
	if err != nil {
		return err
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return fmt.Errorf("inspect %s database: %w", schema, err)
	}

	db, err := openReadOnly(path)
	if err != nil {
		return fmt.Errorf("open %s database read-only: %w", schema, err)
	}
	defer db.Close()

	if err := checkIntegrity(db); err != nil {
		return &IntegrityError{Path: path, Detail: err.Error()}
	}
	pending, err := hasPendingMigrations(db, names)
	if err != nil {
		return fmt.Errorf("inspect %s migrations: %w", schema, err)
	}
	if !pending {
		return nil
	}

	scratch, err := os.MkdirTemp(filepath.Dir(path), ".startup-preflight-")
	if err != nil {
		return fmt.Errorf("create %s migration preflight: %w", schema, err)
	}
	defer os.RemoveAll(scratch)
	snapshot := filepath.Join(scratch, filepath.Base(path))
	if _, err := db.Exec(`VACUUM INTO ?`, snapshot); err != nil {
		return fmt.Errorf("snapshot %s database for migration preflight: %w", schema, err)
	}
	rehearsal, err := Open(snapshot)
	if err != nil {
		return fmt.Errorf("open %s migration preflight snapshot: %w", schema, err)
	}
	migrateErr := Migrate(rehearsal, schema)
	closeErr := rehearsal.Close()
	if migrateErr != nil {
		return fmt.Errorf("rehearse %s migrations: %w", schema, migrateErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close %s migration preflight snapshot: %w", schema, closeErr)
	}
	return nil
}

func openReadOnly(path string) (*sql.DB, error) {
	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	uri := url.URL{Scheme: "file", Path: filepath.ToSlash(absolute)}
	query := uri.Query()
	query.Set("mode", "ro")
	uri.RawQuery = query.Encode()
	db, err := sql.Open("sqlite", uri.String())
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func checkIntegrity(db *sql.DB) error {
	var detail string
	if err := db.QueryRow(`PRAGMA quick_check`).Scan(&detail); err != nil {
		return err
	}
	if detail != "ok" {
		return errors.New(detail)
	}
	return nil
}

func hasPendingMigrations(db *sql.DB, names []string) (bool, error) {
	known := make(map[int]struct{}, len(names))
	for _, name := range names {
		version, err := strconv.Atoi(strings.TrimSuffix(name, ".sql"))
		if err != nil {
			return false, fmt.Errorf("invalid migration filename %q: %w", name, err)
		}
		known[version] = struct{}{}
	}

	var migrationTable int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'table' AND name = 'schema_migrations'`).Scan(&migrationTable); err != nil {
		return false, err
	}
	if migrationTable == 0 {
		return len(names) != 0, nil
	}
	rows, err := db.Query(`SELECT version FROM schema_migrations`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	applied := make(map[int]struct{}, len(names))
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return false, err
		}
		if _, ok := known[version]; !ok {
			return false, fmt.Errorf("database uses unsupported migration version %d", version)
		}
		applied[version] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	return len(applied) != len(known), nil
}
