package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// LegacyPuzzleRecreation binds the read-only database used for snapshot
// backfill to the exact database and WAL files that may later be removed.
// Close is the non-destructive path; CloseAndRemove is valid only while this
// handle still owns that binding.
type LegacyPuzzleRecreation struct {
	mu      sync.Mutex
	db      *sql.DB
	removal *recognizedLegacyRemoval
	closed  bool
}

func OpenLegacyPuzzleRecreation(path string) (*LegacyPuzzleRecreation, error) {
	db, removal, err := openRecognizedLegacyRemoval(path)
	if err != nil {
		return nil, err
	}
	return &LegacyPuzzleRecreation{db: db, removal: removal}, nil
}

func (r *LegacyPuzzleRecreation) ReadOnlyDB() *sql.DB {
	return r.db
}

func (r *LegacyPuzzleRecreation) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil
	}
	r.closed = true
	return r.db.Close()
}

func (r *LegacyPuzzleRecreation) CloseAndRemove() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return errors.New("legacy puzzle recreation handle is already closed")
	}
	r.closed = true
	closeErr := r.db.Close()
	removal := r.removal
	r.mu.Unlock()
	if closeErr != nil {
		return closeErr
	}
	return removal.remove()
}

func RemoveRecognizedLegacyPuzzleStore(path string) error {
	removal, err := prepareRecognizedLegacyRemoval(path)
	if err != nil {
		return err
	}
	return removal.remove()
}

type recognizedLegacyRemoval struct {
	path     string
	identity puzzleStoreFileIdentity
}

func prepareRecognizedLegacyRemoval(path string) (*recognizedLegacyRemoval, error) {
	db, removal, err := openRecognizedLegacyRemoval(path)
	if err != nil {
		return nil, err
	}
	if err := db.Close(); err != nil {
		return nil, fmt.Errorf("close puzzle database probe %s: %w", path, err)
	}
	return removal, nil
}

func openRecognizedLegacyRemoval(path string) (*sql.DB, *recognizedLegacyRemoval, error) {
	beforeValidation, err := capturePuzzleStoreFileIdentity(path)
	if err != nil {
		return nil, nil, err
	}
	db, err := OpenLegacyPuzzleReadOnly(path)
	if err != nil {
		return nil, nil, err
	}
	afterValidation, err := capturePuzzleStoreFileIdentity(path)
	if err != nil {
		return nil, nil, closePuzzleProbe(path, db, err)
	}
	if !samePuzzleStoreFileIdentity(beforeValidation, afterValidation) {
		return nil, nil, closePuzzleProbe(
			path,
			db,
			fmt.Errorf("refuse to remove puzzle database %s: file identity changed during validation", path),
		)
	}
	return db, &recognizedLegacyRemoval{path: path, identity: afterValidation}, nil
}

func (r *recognizedLegacyRemoval) remove() error {
	quarantine, err := quarantinePuzzleStore(r.path)
	if err != nil {
		return err
	}
	quarantinedIdentity, identityErr := capturePuzzleStoreFileIdentity(quarantine.path)
	if identityErr == nil && !samePuzzleStoreFileIdentity(r.identity, quarantinedIdentity) {
		identityErr = fmt.Errorf("quarantined database files do not match the validated files")
	}
	if identityErr != nil {
		restoreErr := quarantine.restore()
		return errors.Join(
			fmt.Errorf("refuse to remove puzzle database %s after path replacement: %w", r.path, identityErr),
			restoreErr,
		)
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

// puzzleStoreFileIdentity excludes the SHM file because it is transient
// coordination state that SQLite may rebuild during a read-only probe. The
// database and WAL are the data-bearing files bound to the validation result.
type puzzleStoreFileIdentity struct {
	database os.FileInfo
	wal      os.FileInfo
}

func capturePuzzleStoreFileIdentity(path string) (puzzleStoreFileIdentity, error) {
	database, err := os.Stat(path)
	if err != nil {
		return puzzleStoreFileIdentity{}, fmt.Errorf("capture puzzle database identity %s: %w", path, err)
	}
	wal, err := os.Stat(path + "-wal")
	if errors.Is(err, os.ErrNotExist) {
		wal = nil
	} else if err != nil {
		return puzzleStoreFileIdentity{}, fmt.Errorf("capture puzzle WAL identity %s: %w", path+"-wal", err)
	}
	return puzzleStoreFileIdentity{database: database, wal: wal}, nil
}

func samePuzzleStoreFileIdentity(left, right puzzleStoreFileIdentity) bool {
	return sameFileIdentity(left.database, right.database) &&
		sameFileIdentity(left.wal, right.wal)
}

func sameFileIdentity(left, right os.FileInfo) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return os.SameFile(left, right) &&
		left.Size() == right.Size() &&
		left.Mode() == right.Mode() &&
		left.ModTime().Equal(right.ModTime())
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
