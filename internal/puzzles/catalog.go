package puzzles

import (
	"context"
	"time"

	"chess-trainer/internal/domain"
)

type Source struct {
	ID         string
	Kind       string
	Path       string
	Checksum   string
	ImportedAt time.Time
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

type StagedImport interface {
	Add(context.Context, domain.Puzzle) error
	Reject(Rejection)
	SetChecksum(string)
	Commit(context.Context) (ImportReport, error)
	Abort(context.Context) error
}

type Catalog interface {
	BeginImport(context.Context, Source) (StagedImport, error)
	Get(context.Context, string) (domain.Puzzle, error)
	RatedCandidates(context.Context, int, int, []string, int) ([]domain.Puzzle, error)
	FreePracticeCandidates(context.Context, string, *int, *int, []string, int) ([]domain.Puzzle, error)
}
