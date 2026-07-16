package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"path/filepath"
	"sync"
	"time"

	"chess-trainer/internal/backup"
	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/importjob"
	"chess-trainer/internal/profile"
	"chess-trainer/internal/puzzles"
	"chess-trainer/internal/storage"
	"chess-trainer/internal/training"
)

type Services struct {
	Paths         storage.Paths
	PuzzleStore   *storage.PuzzleStore
	DataRootLock  *storage.DataRootLock
	UserDB        *sql.DB
	LibraryDB     *sql.DB
	Catalog       *puzzles.SQLiteCatalog
	Importer      *puzzles.LichessImporter
	ImportJobs    *importjob.Service
	UserStore     *training.UserStore
	Training      *training.Service
	Profile       *profile.Service
	Backup        *backup.Service
	quiesceOnce   sync.Once
	quiesceResult error
	closeOnce     sync.Once
	closeResult   error
	stateMu       sync.RWMutex
	state         runtimeState
}

var ErrRuntimeUnavailable = errors.New("application services are unavailable until restart")

type runtimeState uint8

const (
	runtimeRunning runtimeState = iota
	runtimeQuiescing
	runtimeQuiesced
	runtimeClosed
)

// RecoveryRequiredError marks startup failures for which the application can
// safely offer only backup, restore, data-folder, and quit operations.
type RecoveryRequiredError struct {
	Path   string
	Detail string
	cause  error
}

func (e *RecoveryRequiredError) Error() string {
	return fmt.Sprintf("application data recovery is required for %s: %s", e.Path, e.Detail)
}

func (e *RecoveryRequiredError) Unwrap() error {
	return e.cause
}

func Open(paths storage.Paths) (*Services, error) {
	services, err := OpenApplication(paths)
	if err != nil {
		if services == nil {
			return nil, err
		}
		return nil, errors.Join(err, services.Close())
	}
	return services, nil
}

// OpenApplication preserves ownership of the data-root lock when startup finds
// application data that requires recovery. The returned lifecycle must be
// closed when the recovery application terminates.
func OpenApplication(paths storage.Paths) (*Services, error) {
	if err := paths.Ensure(); err != nil {
		return nil, err
	}
	services := &Services{Paths: paths}
	recoverFrom := func(path string, err error) (*Services, error) {
		recoveryErr := recoveryRequired(path, err)
		return services, errors.Join(recoveryErr, services.QuiesceForRestore())
	}

	var err error
	services.DataRootLock, err = storage.AcquireDataRootLock(paths.Root)
	if err != nil {
		return nil, err
	}
	services.Backup = backup.NewService(paths, services.QuiesceForRestore)
	if path, err := preflightStores(paths); err != nil {
		return recoverFrom(path, err)
	}
	services.UserDB, err = openStore(paths.UserDB, "user")
	if err != nil {
		return recoverFrom(paths.UserDB, err)
	}
	if err := preparePuzzleStore(context.Background(), paths.PuzzlesDB, services.UserDB); err != nil {
		return recoverFrom(paths.PuzzlesDB, err)
	}
	services.PuzzleStore, err = storage.OpenPuzzleStore(paths.PuzzlesDB)
	if err != nil {
		return recoverFrom(paths.PuzzlesDB, err)
	}
	services.Catalog = puzzles.NewSQLiteCatalog(
		services.PuzzleStore.Reader,
		services.PuzzleStore.Writer,
	)
	if err := services.Catalog.RecoverStartup(context.Background()); err != nil {
		return recoverFrom(paths.PuzzlesDB, err)
	}
	services.LibraryDB, err = openStore(paths.LibraryDB, "library")
	if err != nil {
		return recoverFrom(paths.LibraryDB, err)
	}

	services.Importer = &puzzles.LichessImporter{
		Catalog:          services.Catalog,
		Rules:            chessrules.Rules{},
		CatalogDirectory: filepath.Dir(paths.PuzzlesDB),
		AvailableBytes:   storage.AvailableBytes,
	}
	services.ImportJobs = importjob.NewService(map[importjob.Kind]importjob.Importer{
		importjob.KindLichess: services.Importer,
	}, services.Catalog, nil)
	services.UserStore = training.NewUserStore(services.UserDB)
	services.Training = training.NewService(
		services.Catalog,
		services.UserStore,
		chessrules.Rules{},
		rand.New(rand.NewSource(time.Now().UnixNano())),
	)
	services.Profile = profile.NewService(services.UserDB, services.Catalog, services.UserStore)
	services.ImportJobs.RequestCleanup()
	return services, nil
}

func preflightStores(paths storage.Paths) (string, error) {
	if err := storage.PreflightMigrations(paths.UserDB, "user"); err != nil {
		return paths.UserDB, fmt.Errorf("preflight user database: %w", err)
	}
	if _, err := storage.ProbePuzzleStore(paths.PuzzlesDB); err != nil {
		return paths.PuzzlesDB, fmt.Errorf("preflight puzzle database: %w", err)
	}
	if err := storage.PreflightMigrations(paths.LibraryDB, "library"); err != nil {
		return paths.LibraryDB, fmt.Errorf("preflight library database: %w", err)
	}
	return "", nil
}

func recoveryRequired(path string, err error) *RecoveryRequiredError {
	var integrityErr *storage.IntegrityError
	if errors.As(err, &integrityErr) {
		path = integrityErr.Path
		return &RecoveryRequiredError{Path: path, Detail: integrityErr.Detail, cause: err}
	}
	return &RecoveryRequiredError{Path: path, Detail: err.Error(), cause: err}
}

func preparePuzzleStore(ctx context.Context, path string, userDB *sql.DB) error {
	state, err := storage.ProbePuzzleStore(path)
	if err != nil {
		return err
	}
	if !state.Exists || !state.Legacy {
		return nil
	}
	legacy, err := storage.OpenLegacyPuzzleRecreation(path)
	if err != nil {
		return err
	}
	if err := storage.BackfillLegacyPuzzleSnapshots(ctx, userDB, legacy.ReadOnlyDB()); err != nil {
		return errors.Join(err, legacy.Close())
	}
	return legacy.CloseAndRemove()
}

func openStore(path string, schema string) (*sql.DB, error) {
	db, err := storage.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s database: %w", schema, err)
	}
	if err := storage.Migrate(db, schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate %s database: %w", schema, err)
	}
	return db, nil
}

func (s *Services) Close() error {
	s.closeOnce.Do(func() {
		s.closeResult = errors.Join(s.QuiesceForRestore(), s.closeDataRootLock())
		s.setRuntimeState(runtimeClosed)
	})
	return s.closeResult
}

// QuiesceForRestore stops background work and closes every live database handle
// without releasing the process-wide data-root lock. Close releases that lock
// only when the application itself shuts down.
func (s *Services) QuiesceForRestore() error {
	s.quiesceOnce.Do(func() {
		s.setRuntimeState(runtimeQuiescing)
		if s.ImportJobs != nil {
			s.ImportJobs.Close()
		}
		var closeErrors []error
		if s.PuzzleStore != nil {
			closeErrors = append(closeErrors, s.PuzzleStore.Close())
		}
		for _, db := range []*sql.DB{s.LibraryDB, s.UserDB} {
			if db != nil {
				closeErrors = append(closeErrors, db.Close())
			}
		}
		s.quiesceResult = errors.Join(closeErrors...)
		s.setRuntimeState(runtimeQuiesced)
	})
	return s.quiesceResult
}

func (s *Services) EnsureRunning() error {
	s.stateMu.RLock()
	defer s.stateMu.RUnlock()
	if s.state != runtimeRunning {
		return ErrRuntimeUnavailable
	}
	return nil
}

func (s *Services) setRuntimeState(state runtimeState) {
	s.stateMu.Lock()
	s.state = state
	s.stateMu.Unlock()
}

func (s *Services) closeDataRootLock() error {
	if s.DataRootLock == nil {
		return nil
	}
	return s.DataRootLock.Close()
}
