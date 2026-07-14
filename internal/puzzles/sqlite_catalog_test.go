package puzzles

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"chess-trainer/internal/domain"
	"chess-trainer/internal/storage"
)

func openTestCatalog(t *testing.T) (*SQLiteCatalog, *sql.DB) {
	t.Helper()
	db, err := storage.Open(filepath.Join(t.TempDir(), "puzzles.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Migrate(db, "puzzles"); err != nil {
		db.Close()
		t.Fatal(err)
	}
	return NewSQLiteCatalog(db), db
}

func testPuzzle(fingerprint string, rating int, themes ...string) domain.Puzzle {
	return domain.Puzzle{
		Fingerprint:  fingerprint,
		DisplayedFEN: "7k/5Q2/6K1/8/8/8/8/8 w - - 0 1",
		Solver:       domain.White,
		Solution:     []domain.MoveNode{{UCI: "f7f8"}},
		Rating:       &rating,
		Themes:       themes,
		Sources: []domain.SourceRef{{
			SourceID:   "lichess",
			ExternalID: fingerprint,
		}},
	}
}

func stagePuzzle(t *testing.T, catalog *SQLiteCatalog, source Source, puzzle domain.Puzzle) (ImportReport, error) {
	t.Helper()
	staged, err := catalog.BeginImport(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	if err := staged.Add(context.Background(), puzzle); err != nil {
		t.Fatal(err)
	}
	staged.SetChecksum("0123456789abcdef")
	return staged.Commit(context.Background())
}

func stagePuzzles(t *testing.T, catalog *SQLiteCatalog, source Source, puzzles ...domain.Puzzle) ImportReport {
	t.Helper()
	staged, err := catalog.BeginImport(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	for _, puzzle := range puzzles {
		if err := staged.Add(context.Background(), puzzle); err != nil {
			t.Fatal(err)
		}
	}
	staged.SetChecksum("0123456789abcdef")
	report, err := staged.Commit(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return report
}

func TestSQLiteCatalogAtomicSourceReplacement(t *testing.T) {
	catalog, db := openTestCatalog(t)
	defer db.Close()
	ctx := context.Background()
	source := Source{ID: "lichess", Kind: "lichess", Path: "/a.csv.zst", ImportedAt: time.Unix(100, 0)}

	if report, err := stagePuzzle(t, catalog, source, testPuzzle("A", 1200)); err != nil || report.Accepted != 1 {
		t.Fatalf("first import report=%+v err=%v", report, err)
	}

	staged, err := catalog.BeginImport(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	if err := staged.Add(ctx, testPuzzle("B", 1300)); err != nil {
		t.Fatal(err)
	}
	if err := staged.Abort(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := catalog.Get(ctx, "A"); err != nil {
		t.Fatalf("A disappeared after abort: %v", err)
	}
	if _, err := catalog.Get(ctx, "B"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("Get(B) err=%v, want sql.ErrNoRows", err)
	}

	report, err := stagePuzzle(t, catalog, source, testPuzzle("B", 1300))
	if err != nil || report.Accepted != 1 {
		t.Fatalf("replacement report=%+v err=%v", report, err)
	}
	if _, err := catalog.Get(ctx, "A"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("Get(A) err=%v, want sql.ErrNoRows", err)
	}
	if _, err := catalog.Get(ctx, "B"); err != nil {
		t.Fatalf("B missing after commit: %v", err)
	}
}

func TestSQLiteCatalogCandidateFilters(t *testing.T) {
	catalog, db := openTestCatalog(t)
	defer db.Close()
	ctx := context.Background()
	source := Source{ID: "lichess", Kind: "lichess", Path: "/a.csv.zst", ImportedAt: time.Unix(100, 0)}
	stagePuzzles(
		t,
		catalog,
		source,
		testPuzzle("P1200", 1200, "fork"),
		testPuzzle("P1500", 1500, "pin"),
		testPuzzle("P1800", 1800, "fork"),
	)

	candidates, err := catalog.RatedCandidates(ctx, 1400, 1600, nil, 10)
	if err != nil || len(candidates) != 1 || candidates[0].Rating == nil || *candidates[0].Rating != 1500 {
		t.Fatalf("candidates=%v err=%v", candidates, err)
	}
	excluded, err := catalog.RatedCandidates(ctx, 1400, 1600, []string{"P1500"}, 10)
	if err != nil || len(excluded) != 0 {
		t.Fatalf("excluded=%v err=%v", excluded, err)
	}

	minimum, maximum := 1400, 1600
	practice, err := catalog.FreePracticeCandidates(ctx, "lichess", &minimum, &maximum, []string{"pin"}, nil, 10)
	if err != nil || len(practice) != 1 || practice[0].Fingerprint != "P1500" {
		t.Fatalf("practice=%v err=%v", practice, err)
	}
	filtered, err := catalog.FreePracticeCandidates(ctx, "lichess", &minimum, &maximum, []string{"fork"}, nil, 10)
	if err != nil || len(filtered) != 0 {
		t.Fatalf("filtered=%v err=%v", filtered, err)
	}
}
