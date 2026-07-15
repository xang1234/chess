package puzzles

import (
	"context"
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

type GenerationSource struct {
	ID        string
	Kind      string
	Path      string
	StartedAt time.Time
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
	ActiveThemes(context.Context) ([]string, error)
}

type CatalogWriter interface {
	BeginImport(context.Context, GenerationSource) (GenerationImport, error)
}

type CatalogMaintenance interface {
	RecoverStartup(context.Context) error
	CleanupBatch(context.Context, int) (bool, error)
}

type GenerationCatalog interface {
	CatalogReader
	CatalogWriter
	CatalogMaintenance
}
