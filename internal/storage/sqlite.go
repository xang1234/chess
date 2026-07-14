package storage

import (
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"strings"

	_ "modernc.org/sqlite"
)

//go:embed migrations/*/*.sql
var migrationFiles embed.FS

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	for _, pragma := range []string{
		"PRAGMA foreign_keys=ON",
		"PRAGMA journal_mode=WAL",
		"PRAGMA busy_timeout=5000",
	} {
		if _, err := db.Exec(pragma); err != nil {
			db.Close()
			return nil, fmt.Errorf("apply %s: %w", pragma, err)
		}
	}
	return db, nil
}

func Migrate(db *sql.DB, schema string) error {
	switch schema {
	case "puzzles", "user", "library":
	default:
		return fmt.Errorf("unknown database schema %q", schema)
	}

	directory := "migrations/" + schema
	entries, err := migrationFiles.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("read %s migrations: %w", schema, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
        version INTEGER PRIMARY KEY
    )`); err != nil {
		return err
	}

	for _, name := range names {
		version, err := strconv.Atoi(strings.TrimSuffix(name, ".sql"))
		if err != nil {
			return fmt.Errorf("invalid migration filename %q: %w", name, err)
		}
		var applied int
		if err := tx.QueryRow(
			`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`,
			version,
		).Scan(&applied); err != nil {
			return err
		}
		if applied != 0 {
			continue
		}

		body, err := migrationFiles.ReadFile(directory + "/" + name)
		if err != nil {
			return err
		}
		if _, err := tx.Exec(string(body)); err != nil {
			return fmt.Errorf("apply %s migration %s: %w", schema, name, err)
		}
		if _, err := tx.Exec(
			`INSERT INTO schema_migrations(version) VALUES (?)`,
			version,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}
