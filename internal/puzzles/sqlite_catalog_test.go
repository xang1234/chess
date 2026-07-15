package puzzles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

func TestSQLiteCatalogFailedCommitRestoresPreviousDataAndIndexes(t *testing.T) {
	catalog, db := openTestCatalog(t)
	defer db.Close()
	ctx := context.Background()
	source := Source{ID: "lichess", Kind: "lichess", Path: "/a.csv.zst", ImportedAt: time.Unix(100, 0)}
	stagePuzzles(t, catalog, source, testPuzzle("A", 1200, "fork"))

	staged, err := catalog.BeginImport(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	invalid := testPuzzle("B", 1300, "pin")
	invalid.Solution = nil
	if err := staged.Add(ctx, invalid); err != nil {
		t.Fatal(err)
	}
	staged.SetChecksum("fedcba9876543210")
	if _, err := staged.Commit(ctx); err == nil {
		t.Fatal("Commit() unexpectedly accepted an empty solution")
	}

	if _, err := catalog.Get(ctx, "A"); err != nil {
		t.Fatalf("previous puzzle disappeared after rollback: %v", err)
	}
	if _, err := catalog.Get(ctx, "B"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("invalid puzzle survived rollback: %v", err)
	}
	for _, index := range secondaryCatalogIndexes {
		var count int
		if err := db.QueryRowContext(
			ctx,
			`SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?`,
			index.name,
		).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("index %s count=%d after rollback", index.name, count)
		}
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

func TestSQLiteCatalogSetCommitCountsDuplicatesAndKeepsLastSourceRecord(t *testing.T) {
	catalog, db := openTestCatalog(t)
	defer db.Close()
	source := Source{ID: "lichess", Kind: "lichess", Path: "/a.csv.zst", ImportedAt: time.Unix(100, 0)}
	first := testPuzzle("same", 1200, "fork")
	last := testPuzzle("same", 1450, "pin")
	last.Sources[0].ExternalID = "last-record"

	report := stagePuzzles(t, catalog, source, first, last)
	if report.Accepted != 1 || report.Duplicates != 1 {
		t.Fatalf("report=%+v, want one accepted and one duplicate", report)
	}
	puzzle, err := catalog.Get(context.Background(), "same")
	if err != nil {
		t.Fatal(err)
	}
	if puzzle.Rating == nil || *puzzle.Rating != 1450 {
		t.Fatalf("rating=%v, want last record rating 1450", puzzle.Rating)
	}
	if len(puzzle.Themes) != 1 || puzzle.Themes[0] != "pin" {
		t.Fatalf("themes=%v, want last record themes", puzzle.Themes)
	}
	if len(puzzle.Sources) != 1 || puzzle.Sources[0].ExternalID != "last-record" {
		t.Fatalf("sources=%+v, want last record metadata", puzzle.Sources)
	}
}

func TestSQLiteCatalogSetCommitHandlesTwentyThousandRowsWithinFiveSeconds(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping bulk catalogue commit probe in short mode")
	}
	catalog, db := openTestCatalog(t)
	defer db.Close()
	ctx := context.Background()
	staged, err := catalog.BeginImport(ctx, Source{
		ID: "lichess", Kind: "lichess", Path: "/bulk.csv.zst", ImportedAt: time.Unix(100, 0),
	})
	if err != nil {
		t.Fatal(err)
	}
	const count = 20_000
	for index := 0; index < count; index++ {
		if err := staged.Add(ctx, testPuzzle(fmt.Sprintf("bulk-%05d", index), 1200, "fork")); err != nil {
			t.Fatal(err)
		}
	}
	staged.SetChecksum("0123456789abcdef")
	started := time.Now()
	report, err := staged.Commit(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if report.Accepted != count {
		t.Fatalf("accepted=%d, want %d", report.Accepted, count)
	}
	if elapsed := time.Since(started); elapsed > 5*time.Second {
		t.Fatalf("set-based commit took %v, want <= 5s", elapsed)
	}
}
