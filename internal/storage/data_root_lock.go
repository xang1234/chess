package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"sync"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

var ErrDataRootLocked = errors.New("data root is locked by another running instance")

type DataRootLockError struct {
	Root  string
	Cause error
}

func (e *DataRootLockError) Error() string {
	return fmt.Sprintf("Chess Trainer is already running for data root %s", e.Root)
}

func (e *DataRootLockError) Unwrap() error {
	return e.Cause
}

func (e *DataRootLockError) Is(target error) bool {
	return target == ErrDataRootLocked
}

type DataRootLock struct {
	mu         sync.Mutex
	db         *sql.DB
	connection *sql.Conn
}

func AcquireDataRootLock(root string) (*DataRootLock, error) {
	path := filepath.Join(root, ".chess-trainer-instance.sqlite")
	dsn, err := dataRootLockDSN(path)
	if err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open data-root lock %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	closeDB := true
	defer func() {
		if closeDB {
			db.Close()
		}
	}()

	ctx := context.Background()
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS instance_lock (
		id INTEGER PRIMARY KEY CHECK (id = 1)
	)`); err != nil {
		if isSQLiteLockConflict(err) {
			return nil, &DataRootLockError{Root: root, Cause: err}
		}
		return nil, fmt.Errorf("initialize data-root lock %s: %w", path, err)
	}
	connection, err := db.Conn(ctx)
	if err != nil {
		return nil, fmt.Errorf("reserve data-root lock connection %s: %w", path, err)
	}
	closeConnection := true
	defer func() {
		if closeConnection {
			connection.Close()
		}
	}()
	if _, err := connection.ExecContext(ctx, `PRAGMA busy_timeout=0`); err != nil {
		return nil, fmt.Errorf("configure data-root lock %s: %w", path, err)
	}
	if _, err := connection.ExecContext(ctx, `BEGIN IMMEDIATE`); err != nil {
		if isSQLiteLockConflict(err) {
			return nil, &DataRootLockError{Root: root, Cause: err}
		}
		return nil, fmt.Errorf("acquire data-root lock %s: %w", path, err)
	}

	closeConnection = false
	closeDB = false
	return &DataRootLock{db: db, connection: connection}, nil
}

func (l *DataRootLock) Close() error {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.connection == nil && l.db == nil {
		return nil
	}

	var rollbackErr, connectionErr, databaseErr error
	if l.connection != nil {
		if _, err := l.connection.ExecContext(context.Background(), `ROLLBACK`); err != nil {
			rollbackErr = fmt.Errorf("release data-root lock transaction: %w", err)
		}
		if err := l.connection.Close(); err != nil {
			connectionErr = fmt.Errorf("close data-root lock connection: %w", err)
		}
		l.connection = nil
	}
	if l.db != nil {
		if err := l.db.Close(); err != nil {
			databaseErr = fmt.Errorf("close data-root lock database: %w", err)
		}
		l.db = nil
	}
	return errors.Join(rollbackErr, connectionErr, databaseErr)
}

func dataRootLockDSN(path string) (string, error) {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve data-root lock path: %w", err)
	}
	uri := url.URL{Scheme: "file", Path: filepath.ToSlash(absolutePath)}
	query := uri.Query()
	query.Set("mode", "rwc")
	query.Add("_pragma", "busy_timeout(0)")
	uri.RawQuery = query.Encode()
	return uri.String(), nil
}

func isSQLiteLockConflict(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	code := sqliteErr.Code() & 0xff
	return code == sqlite3.SQLITE_BUSY || code == sqlite3.SQLITE_LOCKED
}
