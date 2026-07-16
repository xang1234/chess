package puzzles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

var (
	ErrCatalogCorrupt = errors.New("puzzle catalog is corrupt")
	ErrHeadChanged    = errors.New("puzzle source head changed")
)

type SourceKindMismatchError struct {
	SourceID      string
	ExistingKind  string
	RequestedKind string
}

func (e *SourceKindMismatchError) Error() string {
	return fmt.Sprintf(
		"puzzle source %q has kind %q, cannot use kind %q",
		e.SourceID,
		e.ExistingKind,
		e.RequestedKind,
	)
}

type Source struct {
	ID        string
	Kind      string
	Path      string
	StartedAt time.Time
}

type Rejection struct {
	Ordinal int64  `json:"ordinal"`
	Reason  string `json:"reason"`
}

type ImportReport struct {
	Accepted   int64       `json:"accepted"`
	Duplicates int64       `json:"duplicates"`
	Rejected   int64       `json:"rejected"`
	Examples   []Rejection `json:"examples"`
}

type SQLiteCatalog struct {
	readDB  *sql.DB
	writeDB *sql.DB
}

var _ Catalog = (*SQLiteCatalog)(nil)

func NewSQLiteCatalog(readDB, writeDB *sql.DB) *SQLiteCatalog {
	return &SQLiteCatalog{readDB: readDB, writeDB: writeDB}
}

type GenerationImport interface {
	Add(context.Context, TrainingPuzzle) error
	Reject(Rejection)
	Seal(context.Context, string) (ImportReport, error)
	Activate(context.Context) error
	Abandon(context.Context) error
}

type CatalogReader interface {
	Get(context.Context, PuzzleKey) (TrainingPuzzle, error)
	Resolve(context.Context, string, string) (TrainingPuzzle, error)
	RatedCandidates(context.Context, int, int, []string, int) ([]TrainingPuzzle, error)
	FreePracticeCandidates(context.Context, string, *int, *int, []string, *int, int) ([]TrainingPuzzle, error)
	ActiveSourceSummaries(context.Context) ([]SourceSummary, error)
	LearnerRatingBounds(context.Context) (RatingBounds, error)
	ActiveThemes(context.Context) ([]string, error)
}

type CatalogWriter interface {
	BeginImport(context.Context, Source) (GenerationImport, error)
}

type CatalogMaintenance interface {
	RecoverStartup(context.Context) error
	CleanupBatch(context.Context, int) (bool, error)
}

type Catalog interface {
	CatalogReader
	CatalogWriter
	CatalogMaintenance
}
