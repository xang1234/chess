package storage

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

type QuarantineNotice struct {
	QuarantinedPath string
	Detail          string
}

func OpenReplaceable(
	path string,
	schema string,
	now func() time.Time,
) (*sql.DB, QuarantineNotice, error) {
	if now == nil {
		return nil, QuarantineNotice{}, errors.New("replaceable store clock is required")
	}
	if _, err := migrationNames(schema); err != nil {
		return nil, QuarantineNotice{}, err
	}
	exists, err := pathExists(path)
	if err != nil {
		return nil, QuarantineNotice{}, err
	}
	if !exists {
		db, openErr := openMigratedReplaceable(path, schema)
		return db, QuarantineNotice{}, openErr
	}

	headerOK, headerErr := hasSQLiteHeader(path)
	var preflightErr error
	switch {
	case headerErr != nil:
		preflightErr = fmt.Errorf("inspect %s database header: %w", schema, headerErr)
	case !headerOK:
		preflightErr = fmt.Errorf("preflight %s database integrity: invalid SQLite header", schema)
	default:
		preflightErr = PreflightMigrations(path, schema)
	}
	if preflightErr == nil {
		db, openErr := openMigratedReplaceable(path, schema)
		if openErr == nil {
			return db, QuarantineNotice{}, nil
		}
		preflightErr = openErr
	}

	quarantinedPath, quarantineErr := quarantineReplaceableStore(path, now().UTC())
	notice := QuarantineNotice{Detail: preflightErr.Error()}
	if quarantineErr != nil {
		return nil, notice, errors.Join(preflightErr, quarantineErr)
	}
	notice.QuarantinedPath = quarantinedPath
	db, recreateErr := openMigratedReplaceable(path, schema)
	if recreateErr != nil {
		return nil, notice, fmt.Errorf(
			"recreate %s database after quarantine at %s: %w",
			schema,
			quarantinedPath,
			recreateErr,
		)
	}
	return db, notice, nil
}

func hasSQLiteHeader(path string) (bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer file.Close()
	header := make([]byte, len("SQLite format 3\x00"))
	if _, err := io.ReadFull(file, header); err != nil {
		if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
			return false, nil
		}
		return false, err
	}
	return bytes.Equal(header, []byte("SQLite format 3\x00")), nil
}

func openMigratedReplaceable(path, schema string) (*sql.DB, error) {
	db, err := Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s database: %w", schema, err)
	}
	if err := Migrate(db, schema); err != nil {
		return nil, errors.Join(
			fmt.Errorf("migrate %s database: %w", schema, err),
			db.Close(),
		)
	}
	return db, nil
}

func quarantineReplaceableStore(path string, at time.Time) (string, error) {
	quarantinedPath := path + ".quarantine-" + at.Format("20060102T150405Z")
	suffixes := []string{"", "-wal", "-shm"}
	existing := make([]string, 0, len(suffixes))
	for _, suffix := range suffixes {
		if _, err := os.Stat(path + suffix); err == nil {
			existing = append(existing, suffix)
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect replaceable database file %s: %w", path+suffix, err)
		}
	}
	if len(existing) == 0 || existing[0] != "" {
		return "", fmt.Errorf("replaceable database %s disappeared before quarantine", path)
	}
	for _, suffix := range existing {
		if _, err := os.Stat(quarantinedPath + suffix); err == nil {
			return "", fmt.Errorf("replaceable database quarantine already exists: %s", quarantinedPath+suffix)
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("inspect replaceable database quarantine %s: %w", quarantinedPath+suffix, err)
		}
	}

	moved := make([]string, 0, len(existing))
	for _, suffix := range existing {
		if err := os.Rename(path+suffix, quarantinedPath+suffix); err != nil {
			rollbackErr := restoreReplaceableFiles(path, quarantinedPath, moved)
			return "", errors.Join(
				fmt.Errorf("quarantine replaceable database file %s: %w", path+suffix, err),
				rollbackErr,
			)
		}
		moved = append(moved, suffix)
	}
	return quarantinedPath, nil
}

func restoreReplaceableFiles(path, quarantinedPath string, moved []string) error {
	var restoreErrors []error
	for index := len(moved) - 1; index >= 0; index-- {
		suffix := moved[index]
		if err := os.Rename(quarantinedPath+suffix, path+suffix); err != nil {
			restoreErrors = append(restoreErrors, fmt.Errorf(
				"restore replaceable database file %s: %w",
				path+suffix,
				err,
			))
		}
	}
	return errors.Join(restoreErrors...)
}
