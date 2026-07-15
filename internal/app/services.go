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
	Paths        storage.Paths
	PuzzleStore  *storage.PuzzleStore
	DataRootLock *storage.DataRootLock
	UserDB       *sql.DB
	LibraryDB    *sql.DB
	Catalog      *puzzles.SQLiteCatalog
	Importer     *puzzles.LichessImporter
	ImportJobs   *importjob.Service
	UserStore    *training.UserStore
	Training     *training.Service
	Profile      *profile.Service
	Backup       *backup.Service
	closeOnce    sync.Once
	closeResult  error
}

func Open(paths storage.Paths) (*Services, error) {
	if err := paths.Ensure(); err != nil {
		return nil, err
	}
	services := &Services{Paths: paths}
	closeOnError := func(err error) (*Services, error) {
		return nil, errors.Join(err, services.Close())
	}

	var err error
	services.DataRootLock, err = storage.AcquireDataRootLock(paths.Root)
	if err != nil {
		return nil, err
	}
	if err := storage.CheckDurableIntegrity(paths); err != nil {
		return closeOnError(err)
	}
	services.UserDB, err = openStore(paths.UserDB, "user")
	if err != nil {
		return closeOnError(err)
	}
	if err := preparePuzzleStore(context.Background(), paths.PuzzlesDB, services.UserDB); err != nil {
		return closeOnError(err)
	}
	services.PuzzleStore, err = storage.OpenPuzzleStore(paths.PuzzlesDB)
	if err != nil {
		return closeOnError(err)
	}
	services.Catalog = puzzles.NewSQLiteCatalog(
		services.PuzzleStore.Reader,
		services.PuzzleStore.Writer,
	)
	if err := services.Catalog.RecoverStartup(context.Background()); err != nil {
		return closeOnError(err)
	}
	services.LibraryDB, err = openStore(paths.LibraryDB, "library")
	if err != nil {
		return closeOnError(err)
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
	services.Backup = backup.NewService(paths, services.Close)
	services.ImportJobs.RequestCleanup()
	return services, nil
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
		if s.DataRootLock != nil {
			closeErrors = append(closeErrors, s.DataRootLock.Close())
		}
		s.closeResult = errors.Join(closeErrors...)
	})
	return s.closeResult
}
