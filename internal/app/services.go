package app

import (
	"database/sql"
	"errors"
	"fmt"
	"sync"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/importjob"
	"chess-trainer/internal/puzzles"
	"chess-trainer/internal/storage"
)

type Services struct {
	Paths       storage.Paths
	PuzzlesDB   *sql.DB
	UserDB      *sql.DB
	LibraryDB   *sql.DB
	Catalog     *puzzles.SQLiteCatalog
	Importer    *puzzles.LichessImporter
	ImportJobs  *importjob.Service
	closeOnce   sync.Once
	closeResult error
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
	services.PuzzlesDB, err = openStore(paths.PuzzlesDB, "puzzles")
	if err != nil {
		return closeOnError(err)
	}
	services.UserDB, err = openStore(paths.UserDB, "user")
	if err != nil {
		return closeOnError(err)
	}
	services.LibraryDB, err = openStore(paths.LibraryDB, "library")
	if err != nil {
		return closeOnError(err)
	}

	services.Catalog = puzzles.NewSQLiteCatalog(services.PuzzlesDB)
	services.Importer = &puzzles.LichessImporter{
		Catalog: services.Catalog,
		Rules:   chessrules.Rules{},
		AvailableBytes: func(string) (uint64, error) {
			return storage.AvailableBytes(paths.Root)
		},
	}
	services.ImportJobs = importjob.NewService(services.Importer, nil)
	return services, nil
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
		var closeErrors []error
		for _, db := range []*sql.DB{s.LibraryDB, s.UserDB, s.PuzzlesDB} {
			if db != nil {
				closeErrors = append(closeErrors, db.Close())
			}
		}
		s.closeResult = errors.Join(closeErrors...)
	})
	return s.closeResult
}
