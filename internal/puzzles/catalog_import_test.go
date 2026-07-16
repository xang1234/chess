package puzzles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"chess-trainer/internal/domain"
	"chess-trainer/internal/storage"
)

func openTestGenerationalCatalog(t *testing.T) (*SQLiteCatalog, *storage.PuzzleStore) {
	t.Helper()
	store, err := storage.OpenPuzzleStore(filepath.Join(t.TempDir(), "puzzles.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close generation puzzle store: %v", err)
		}
	})
	return NewSQLiteCatalog(store.Reader, store.Writer), store
}

func testSource(id, kind, path string) Source {
	return Source{
		ID:        id,
		Kind:      kind,
		Path:      path,
		StartedAt: time.Unix(1_700_000_000, 0),
	}
}

func testTrainingPuzzle(source Source, fingerprint string, rating int, themes ...string) TrainingPuzzle {
	popularity, playCount := rating/10, rating*10
	return TrainingPuzzle{
		Core: PuzzleCore{
			Fingerprint:   fingerprint,
			DisplayedFEN:  "7k/5Q2/6K1/8/8/8/8/8 w - - 0 1",
			Solver:        domain.White,
			Solution:      []domain.MoveNode{{UCI: "f7f8"}},
			SolutionPlies: 1,
		},
		Occurrence: PuzzleOccurrence{
			SourceID:    source.ID,
			SourceKind:  source.Kind,
			ExternalID:  fingerprint,
			SourceFEN:   "source-" + fingerprint,
			PreludeUCI:  "e2e4",
			Rating:      &rating,
			Popularity:  &popularity,
			PlayCount:   &playCount,
			URL:         "https://example.test/" + fingerprint,
			Attribution: "test suite",
			Metadata:    map[string]any{"fingerprint": fingerprint},
			Themes:      themes,
			Ordinal:     1,
		},
	}
}

func beginGenerationImport(t *testing.T, catalog *SQLiteCatalog, source Source) GenerationImport {
	t.Helper()
	importing, err := catalog.BeginImport(context.Background(), source)
	if err != nil {
		t.Fatal(err)
	}
	return importing
}

func sealAndActivate(t *testing.T, importing GenerationImport, puzzles ...TrainingPuzzle) ImportReport {
	t.Helper()
	ctx := context.Background()
	for _, puzzle := range puzzles {
		if err := importing.Add(ctx, puzzle); err != nil {
			t.Fatal(err)
		}
	}
	report, err := importing.Seal(ctx, " ABCDEF0123456789 ")
	if err != nil {
		t.Fatal(err)
	}
	if err := importing.Activate(ctx); err != nil {
		t.Fatal(err)
	}
	return report
}

func generationIDForPath(t *testing.T, db *sql.DB, path string) string {
	t.Helper()
	var generationID string
	if err := db.QueryRow(
		`SELECT generation_id FROM source_generations WHERE source_path = ?`,
		path,
	).Scan(&generationID); err != nil {
		t.Fatal(err)
	}
	return generationID
}

func TestBuildingGenerationIsInvisible(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	source := testSource("building", "test", "/building")
	importing := beginGenerationImport(t, catalog, source)
	staged := importing.(*sqliteGenerationImport)
	defer func() {
		if err := importing.Abandon(context.Background()); err != nil {
			t.Errorf("abandon building-stage fixture: %v", err)
		}
	}()

	for index := range 1_000 {
		puzzle := testTrainingPuzzle(source, fmt.Sprintf("building-%04d", index), 1200)
		puzzle.Occurrence.Ordinal = int64(index + 1)
		if err := importing.Add(context.Background(), puzzle); err != nil {
			t.Fatal(err)
		}
	}

	var persisted int
	if err := store.Reader.QueryRow(
		`SELECT COUNT(*) FROM puzzle_occurrences WHERE fingerprint = 'building-0000'`,
	).Scan(&persisted); err != nil {
		t.Fatal(err)
	}
	if persisted != 0 {
		t.Fatalf("persisted building occurrences = %d, want 0 before seal", persisted)
	}
	var stagedRows int
	if err := staged.stage.db.QueryRow(`SELECT COUNT(*) FROM staged_rows`).Scan(&stagedRows); err != nil {
		t.Fatal(err)
	}
	if stagedRows != generationImportBatchSize || len(staged.buffer) != 0 {
		t.Fatalf(
			"stage after one batch: rows=%d buffer=%d, want %d and 0",
			stagedRows,
			len(staged.buffer),
			generationImportBatchSize,
		)
	}
	_, err := catalog.Get(context.Background(), PuzzleKey{
		Fingerprint: "building-0000",
		SourceID:    source.ID,
	})
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("Get(building generation) err = %v, want sql.ErrNoRows", err)
	}
}

func TestGenerationLocalDuplicateLastRecordWins(t *testing.T) {
	catalog, _ := openTestGenerationalCatalog(t)
	source := testSource("duplicates", "test", "/duplicates")
	importing := beginGenerationImport(t, catalog, source)
	first := testTrainingPuzzle(source, "shared", 1200, " fork ", "fork")
	last := testTrainingPuzzle(source, "shared", 1700, " pin ", "", "skewer", "pin")
	last.Occurrence.Ordinal = 2
	last.Occurrence.ExternalID = "last-record"
	last.Occurrence.SourceFEN = "last source fen"
	last.Occurrence.PreludeUCI = "d2d4"
	last.Occurrence.URL = "https://example.test/last"
	last.Occurrence.Attribution = "last attribution"
	last.Occurrence.Metadata = map[string]any{"record": "last"}
	if err := importing.Add(context.Background(), first); err != nil {
		t.Fatal(err)
	}
	if err := importing.Add(context.Background(), last); err != nil {
		t.Fatal(err)
	}
	report, err := importing.Seal(context.Background(), "checksum")
	if err != nil {
		t.Fatal(err)
	}
	if report.Accepted != 1 || report.Duplicates != 1 {
		t.Fatalf("report = %+v, want accepted=1 duplicates=1", report)
	}
	if err := importing.Activate(context.Background()); err != nil {
		t.Fatal(err)
	}

	got, err := catalog.Get(context.Background(), PuzzleKey{Fingerprint: "shared", SourceID: source.ID})
	if err != nil {
		t.Fatal(err)
	}
	if got.Occurrence.ExternalID != "last-record" ||
		got.Occurrence.SourceFEN != "last source fen" ||
		got.Occurrence.PreludeUCI != "d2d4" ||
		got.Occurrence.Rating == nil || *got.Occurrence.Rating != 1700 ||
		got.Occurrence.URL != "https://example.test/last" ||
		got.Occurrence.Attribution != "last attribution" ||
		got.Occurrence.Ordinal != 2 {
		t.Fatalf("last occurrence was not retained: %+v", got.Occurrence)
	}
	if !reflect.DeepEqual(got.Occurrence.Metadata, map[string]any{"record": "last"}) {
		t.Fatalf("metadata = %#v, want last record", got.Occurrence.Metadata)
	}
	if !reflect.DeepEqual(got.Occurrence.Themes, []string{"pin", "skewer"}) {
		t.Fatalf("themes = %q, want normalized last themes", got.Occurrence.Themes)
	}
}

func TestGenerationDuplicateAccountingAndLastRecordWinsAcrossBatches(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	source := testSource("cross-batch-duplicates", "test", "/cross-batch-duplicates")
	importing := beginGenerationImport(t, catalog, source)

	first := testTrainingPuzzle(source, "shared-across-batches", 1200, "fork", "pin")
	first.Occurrence.Ordinal = 1
	withinBatch := testTrainingPuzzle(source, "shared-across-batches", 1400, "skewer")
	withinBatch.Occurrence.Ordinal = 2
	for _, puzzle := range []TrainingPuzzle{first, withinBatch} {
		if err := importing.Add(context.Background(), puzzle); err != nil {
			t.Fatal(err)
		}
	}
	for index := range 998 {
		puzzle := testTrainingPuzzle(source, fmt.Sprintf("unique-%04d", index), 1300+(index%100))
		puzzle.Occurrence.Ordinal = int64(index + 3)
		if err := importing.Add(context.Background(), puzzle); err != nil {
			t.Fatal(err)
		}
	}

	acrossBatch := testTrainingPuzzle(source, "shared-across-batches", 1700)
	acrossBatch.Occurrence.Ordinal = 1001
	acrossBatch.Occurrence.ExternalID = "last-across-batches"
	if err := importing.Add(context.Background(), acrossBatch); err != nil {
		t.Fatal(err)
	}
	tail := testTrainingPuzzle(source, "tail-after-flush", 1500, "mate")
	tail.Occurrence.Ordinal = 1002
	if err := importing.Add(context.Background(), tail); err != nil {
		t.Fatal(err)
	}

	report, err := importing.Seal(context.Background(), "cross-batch-checksum")
	if err != nil {
		t.Fatal(err)
	}
	if report.Accepted != 1000 || report.Duplicates != 2 || report.Rejected != 0 {
		t.Fatalf("report = %+v, want accepted=1000 duplicates=2 rejected=0", report)
	}
	if err := importing.Activate(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, err := catalog.Get(context.Background(), PuzzleKey{
		Fingerprint: "shared-across-batches",
		SourceID:    source.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Occurrence.ExternalID != "last-across-batches" ||
		got.Occurrence.Rating == nil || *got.Occurrence.Rating != 1700 ||
		got.Occurrence.Ordinal != 1001 || len(got.Occurrence.Themes) != 0 {
		t.Fatalf("last cross-batch occurrence was not retained: %+v", got.Occurrence)
	}
	generationID := generationIDForPath(t, store.Reader, source.Path)
	var occurrences, sharedThemes int
	if err := store.Reader.QueryRow(
		`SELECT
		   (SELECT COUNT(*) FROM puzzle_occurrences WHERE generation_id = ?),
		   (SELECT COUNT(*) FROM occurrence_themes
		    WHERE generation_id = ? AND fingerprint = 'shared-across-batches')`,
		generationID,
		generationID,
	).Scan(&occurrences, &sharedThemes); err != nil {
		t.Fatal(err)
	}
	if occurrences != 1000 || sharedThemes != 0 {
		t.Fatalf("physical last-record state: occurrences=%d shared_themes=%d, want 1000 and 0", occurrences, sharedThemes)
	}
}

func TestIntraBatchCoreConflictRollsBackWholeBatch(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	source := testSource("intra-batch-conflict", "test", "/intra-batch-conflict")
	importing := beginGenerationImport(t, catalog, source)
	valid := testTrainingPuzzle(source, "must-roll-back", 1200, "fork")
	conflict := testTrainingPuzzle(source, "intra-batch-shared", 1300, "pin")
	mismatch := testTrainingPuzzle(source, "intra-batch-shared", 1400, "skewer")
	mismatch.Core.DisplayedFEN = "8/8/8/8/8/8/5Q2/7k w - - 0 1"
	for index, puzzle := range []TrainingPuzzle{valid, conflict, mismatch} {
		puzzle.Occurrence.Ordinal = int64(index + 1)
		if err := importing.Add(context.Background(), puzzle); err != nil {
			t.Fatal(err)
		}
	}
	_, err := importing.Seal(context.Background(), "intra-batch-conflict-checksum")
	if !errors.Is(err, ErrCatalogCorrupt) {
		t.Fatalf("Seal(intra-batch core conflict) err = %v, want ErrCatalogCorrupt", err)
	}
	generationID := generationIDForPath(t, store.Reader, source.Path)
	assertFailedGenerationHasNoPuzzleWrites(t, store.Reader, generationID, "must-roll-back")
	assertFailedGenerationHasNoPuzzleWrites(t, store.Reader, generationID, "intra-batch-shared")
}

func TestExistingCoreConflictRollsBackWholeBatch(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	existingSource := testSource("existing-core-owner", "test", "/existing-core-owner")
	sealAndActivate(
		t,
		beginGenerationImport(t, catalog, existingSource),
		testTrainingPuzzle(existingSource, "existing-conflict-core", 1200, "fork"),
	)

	conflictingSource := testSource("existing-core-conflict", "test", "/existing-core-conflict")
	importing := beginGenerationImport(t, catalog, conflictingSource)
	valid := testTrainingPuzzle(conflictingSource, "new-core-must-roll-back", 1300, "pin")
	mismatch := testTrainingPuzzle(conflictingSource, "existing-conflict-core", 1400, "skewer")
	mismatch.Core.DisplayedFEN = "8/8/8/8/8/8/5Q2/7k w - - 0 1"
	for index, puzzle := range []TrainingPuzzle{valid, mismatch} {
		puzzle.Occurrence.Ordinal = int64(index + 1)
		if err := importing.Add(context.Background(), puzzle); err != nil {
			t.Fatal(err)
		}
	}
	_, err := importing.Seal(context.Background(), "existing-core-conflict-checksum")
	if !errors.Is(err, ErrCatalogCorrupt) {
		t.Fatalf("Seal(existing core conflict) err = %v, want ErrCatalogCorrupt", err)
	}
	generationID := generationIDForPath(t, store.Reader, conflictingSource.Path)
	assertFailedGenerationHasNoPuzzleWrites(t, store.Reader, generationID, "new-core-must-roll-back")
	assertFailedGenerationHasNoPuzzleWrites(t, store.Reader, generationID, "existing-conflict-core")
	if _, err := catalog.Get(context.Background(), PuzzleKey{
		Fingerprint: "existing-conflict-core",
		SourceID:    existingSource.ID,
	}); err != nil {
		t.Fatalf("existing active puzzle changed after conflict rollback: %v", err)
	}
}

func assertFailedGenerationHasNoPuzzleWrites(
	t *testing.T,
	db *sql.DB,
	generationID string,
	fingerprint string,
) {
	t.Helper()
	var occurrences, themes, orphanCore int
	if err := db.QueryRow(
		`SELECT
		   (SELECT COUNT(*) FROM puzzle_occurrences
		    WHERE generation_id = ? AND fingerprint = ?),
		   (SELECT COUNT(*) FROM occurrence_themes
		    WHERE generation_id = ? AND fingerprint = ?),
		   (SELECT COUNT(*) FROM puzzle_cores AS core
		    WHERE core.fingerprint = ?
		      AND NOT EXISTS (
		        SELECT 1 FROM puzzle_occurrences AS occurrence
		        WHERE occurrence.fingerprint = core.fingerprint
		      ))`,
		generationID,
		fingerprint,
		generationID,
		fingerprint,
		fingerprint,
	).Scan(&occurrences, &themes, &orphanCore); err != nil {
		t.Fatal(err)
	}
	if occurrences != 0 || themes != 0 || orphanCore != 0 {
		t.Fatalf(
			"failed generation retained writes for %q: occurrences=%d themes=%d orphan_cores=%d",
			fingerprint,
			occurrences,
			themes,
			orphanCore,
		)
	}
}

func TestSharedCoreRetainsIndependentOccurrences(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	firstSource := testSource("alpha", "csv", "/alpha")
	secondSource := testSource("beta", "lichess", "/beta")
	first := testTrainingPuzzle(firstSource, "common-core", 1100, "fork")
	second := testTrainingPuzzle(secondSource, "common-core", 1900, "pin")
	second.Occurrence.ExternalID = "beta-external"
	second.Occurrence.SourceFEN = "beta source fen"
	second.Occurrence.PreludeUCI = "d2d4"
	second.Occurrence.Attribution = "beta attribution"

	sealAndActivate(t, beginGenerationImport(t, catalog, firstSource), first)
	report := sealAndActivate(t, beginGenerationImport(t, catalog, secondSource), second)
	if report.Accepted != 1 || report.Duplicates != 0 {
		t.Fatalf("second source report = %+v, want one accepted occurrence", report)
	}

	firstGot, err := catalog.Get(context.Background(), PuzzleKey{Fingerprint: "common-core", SourceID: "alpha"})
	if err != nil {
		t.Fatal(err)
	}
	secondGot, err := catalog.Get(context.Background(), PuzzleKey{Fingerprint: "common-core", SourceID: "beta"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(firstGot.Core, secondGot.Core) {
		t.Fatalf("shared cores differ: first=%+v second=%+v", firstGot.Core, secondGot.Core)
	}
	if firstGot.Occurrence.SourceFEN == secondGot.Occurrence.SourceFEN ||
		*firstGot.Occurrence.Rating == *secondGot.Occurrence.Rating ||
		reflect.DeepEqual(firstGot.Occurrence.Themes, secondGot.Occurrence.Themes) {
		t.Fatalf("source occurrences were conflated: first=%+v second=%+v", firstGot.Occurrence, secondGot.Occurrence)
	}
	var cores, occurrences int
	if err := store.Reader.QueryRow(`SELECT COUNT(*) FROM puzzle_cores WHERE fingerprint = 'common-core'`).Scan(&cores); err != nil {
		t.Fatal(err)
	}
	if err := store.Reader.QueryRow(`SELECT COUNT(*) FROM puzzle_occurrences WHERE fingerprint = 'common-core'`).Scan(&occurrences); err != nil {
		t.Fatal(err)
	}
	if cores != 1 || occurrences != 2 {
		t.Fatalf("physical rows: cores=%d occurrences=%d, want 1 and 2", cores, occurrences)
	}
}

func TestCoreContentMismatchReturnsCatalogCorrupt(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	firstSource := testSource("core-one", "test", "/core-one")
	secondSource := testSource("core-two", "test", "/core-two")
	sealAndActivate(t, beginGenerationImport(t, catalog, firstSource), testTrainingPuzzle(firstSource, "same-fingerprint", 1200))

	importing := beginGenerationImport(t, catalog, secondSource)
	mismatch := testTrainingPuzzle(secondSource, "same-fingerprint", 1300)
	mismatch.Core.DisplayedFEN = "8/8/8/8/8/8/5Q2/7k w - - 0 1"
	if err := importing.Add(context.Background(), mismatch); err != nil {
		t.Fatal(err)
	}
	_, err := importing.Seal(context.Background(), "checksum")
	if !errors.Is(err, ErrCatalogCorrupt) {
		t.Fatalf("Seal(core mismatch) err = %v, want ErrCatalogCorrupt", err)
	}
	var secondHead int
	if err := store.Reader.QueryRow(`SELECT COUNT(*) FROM source_heads WHERE source_id = 'core-two'`).Scan(&secondHead); err != nil {
		t.Fatal(err)
	}
	if secondHead != 0 {
		t.Fatalf("mismatched source head count = %d, want 0", secondHead)
	}
	if _, err := catalog.Get(context.Background(), PuzzleKey{Fingerprint: "same-fingerprint", SourceID: "core-one"}); err != nil {
		t.Fatalf("existing active core disappeared: %v", err)
	}
}

func TestExistingSourceKindMismatchPreservesHead(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	source := testSource("immutable-source", "lichess", "/initial")
	sealAndActivate(t, beginGenerationImport(t, catalog, source), testTrainingPuzzle(source, "initial", 1200))
	var headBefore string
	if err := store.Reader.QueryRow(`SELECT generation_id FROM source_heads WHERE source_id = ?`, source.ID).Scan(&headBefore); err != nil {
		t.Fatal(err)
	}

	mismatched := testSource(source.ID, "csv", "/mismatch")
	_, err := catalog.BeginImport(context.Background(), mismatched)
	var kindErr *SourceKindMismatchError
	if !errors.As(err, &kindErr) {
		t.Fatalf("BeginImport(kind mismatch) err = %v, want SourceKindMismatchError", err)
	}
	if kindErr.SourceID != source.ID || kindErr.ExistingKind != "lichess" || kindErr.RequestedKind != "csv" {
		t.Fatalf("kind mismatch details = %+v", kindErr)
	}
	var headAfter string
	if err := store.Reader.QueryRow(`SELECT generation_id FROM source_heads WHERE source_id = ?`, source.ID).Scan(&headAfter); err != nil {
		t.Fatal(err)
	}
	if headAfter != headBefore {
		t.Fatalf("head changed from %q to %q", headBefore, headAfter)
	}
	var mismatchedGenerations int
	if err := store.Reader.QueryRow(`SELECT COUNT(*) FROM source_generations WHERE source_path = '/mismatch'`).Scan(&mismatchedGenerations); err != nil {
		t.Fatal(err)
	}
	if mismatchedGenerations != 0 {
		t.Fatalf("kind mismatch created %d generations", mismatchedGenerations)
	}
}

func TestSealRequiresBuildingGenerationAndChecksum(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	source := testSource("seal-state", "test", "/seal-state")
	importing := beginGenerationImport(t, catalog, source)
	if err := importing.Add(context.Background(), testTrainingPuzzle(source, "seal-state", 1200)); err != nil {
		t.Fatal(err)
	}
	if _, err := importing.Seal(context.Background(), " \t\n "); err == nil {
		t.Fatal("Seal() accepted an empty normalized checksum")
	}
	generationID := generationIDForPath(t, store.Reader, source.Path)
	var status string
	if err := store.Reader.QueryRow(`SELECT status FROM source_generations WHERE generation_id = ?`, generationID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if status != "building" {
		t.Fatalf("status after empty checksum = %q, want building", status)
	}
	if err := importing.Abandon(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := importing.Seal(context.Background(), "valid"); err == nil {
		t.Fatal("Seal() accepted an abandoned generation")
	}
}

func TestActivateRequiresOwnSealedGeneration(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	source := testSource("activate-state", "test", "/activate-state")
	importing := beginGenerationImport(t, catalog, source)
	if err := importing.Add(context.Background(), testTrainingPuzzle(source, "activate-state", 1200)); err != nil {
		t.Fatal(err)
	}
	if err := importing.Activate(context.Background()); err == nil {
		t.Fatal("Activate() accepted a building generation")
	}
	if _, err := importing.Seal(context.Background(), "checksum"); err != nil {
		t.Fatal(err)
	}
	generationID := generationIDForPath(t, store.Reader, source.Path)
	if _, err := store.Writer.Exec(`UPDATE source_generations SET checksum = NULL WHERE generation_id = ?`, generationID); err != nil {
		t.Fatal(err)
	}
	if err := importing.Activate(context.Background()); err == nil {
		t.Fatal("Activate() accepted a sealed generation without a checksum")
	}
	var heads int
	if err := store.Reader.QueryRow(`SELECT COUNT(*) FROM source_heads WHERE source_id = ?`, source.ID).Scan(&heads); err != nil {
		t.Fatal(err)
	}
	if heads != 0 {
		t.Fatalf("invalid activation created %d heads", heads)
	}
}

func TestActivateCASWithExpectedHead(t *testing.T) {
	catalog, _ := openTestGenerationalCatalog(t)
	source := testSource("cas-existing", "test", "/baseline")
	sealAndActivate(t, beginGenerationImport(t, catalog, source), testTrainingPuzzle(source, "baseline", 1000))

	oldSource := source
	oldSource.Path = "/old-writer"
	newSource := source
	newSource.Path = "/new-writer"
	oldWriter := beginGenerationImport(t, catalog, oldSource)
	newWriter := beginGenerationImport(t, catalog, newSource)
	oldPuzzle := testTrainingPuzzle(source, "old-writer", 1200)
	newPuzzle := testTrainingPuzzle(source, "new-writer", 1300)
	if err := oldWriter.Add(context.Background(), oldPuzzle); err != nil {
		t.Fatal(err)
	}
	if err := newWriter.Add(context.Background(), newPuzzle); err != nil {
		t.Fatal(err)
	}
	if _, err := oldWriter.Seal(context.Background(), "old-checksum"); err != nil {
		t.Fatal(err)
	}
	if _, err := newWriter.Seal(context.Background(), "new-checksum"); err != nil {
		t.Fatal(err)
	}
	if err := newWriter.Activate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := oldWriter.Activate(context.Background()); !errors.Is(err, ErrHeadChanged) {
		t.Fatalf("stale Activate() err = %v, want ErrHeadChanged", err)
	}
	if _, err := catalog.Get(context.Background(), PuzzleKey{Fingerprint: "new-writer", SourceID: source.ID}); err != nil {
		t.Fatalf("new writer is not active: %v", err)
	}
	if _, err := catalog.Get(context.Background(), PuzzleKey{Fingerprint: "old-writer", SourceID: source.ID}); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("stale writer became active: %v", err)
	}
}

func TestActivateCASWithNoExpectedHead(t *testing.T) {
	catalog, _ := openTestGenerationalCatalog(t)
	source := testSource("cas-empty", "test", "/first-writer")
	secondSource := source
	secondSource.Path = "/second-writer"
	first := beginGenerationImport(t, catalog, source)
	second := beginGenerationImport(t, catalog, secondSource)
	if err := first.Add(context.Background(), testTrainingPuzzle(source, "first", 1200)); err != nil {
		t.Fatal(err)
	}
	if err := second.Add(context.Background(), testTrainingPuzzle(source, "second", 1300)); err != nil {
		t.Fatal(err)
	}
	if _, err := first.Seal(context.Background(), "first-checksum"); err != nil {
		t.Fatal(err)
	}
	if _, err := second.Seal(context.Background(), "second-checksum"); err != nil {
		t.Fatal(err)
	}
	if err := first.Activate(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := second.Activate(context.Background()); !errors.Is(err, ErrHeadChanged) {
		t.Fatalf("second first-head Activate() err = %v, want ErrHeadChanged", err)
	}
	if _, err := catalog.Get(context.Background(), PuzzleKey{Fingerprint: "first", SourceID: source.ID}); err != nil {
		t.Fatalf("first writer is not active: %v", err)
	}
}

func TestAbandonDoesNotDeleteOrChangeHead(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	source := testSource("abandon", "test", "/active")
	sealAndActivate(t, beginGenerationImport(t, catalog, source), testTrainingPuzzle(source, "active", 1000))
	var headBefore string
	if err := store.Reader.QueryRow(`SELECT generation_id FROM source_heads WHERE source_id = ?`, source.ID).Scan(&headBefore); err != nil {
		t.Fatal(err)
	}

	replacementSource := source
	replacementSource.Path = "/abandoned"
	replacement := beginGenerationImport(t, catalog, replacementSource)
	stagePath := replacement.(*sqliteGenerationImport).stage.path
	for index := range 1_000 {
		puzzle := testTrainingPuzzle(source, fmt.Sprintf("abandoned-%04d", index), 1200)
		puzzle.Occurrence.Ordinal = int64(index + 1)
		if err := replacement.Add(context.Background(), puzzle); err != nil {
			t.Fatal(err)
		}
	}
	abandonedID := generationIDForPath(t, store.Reader, replacementSource.Path)
	if err := replacement.Abandon(context.Background()); err != nil {
		t.Fatal(err)
	}
	var headAfter, status string
	if err := store.Reader.QueryRow(`SELECT generation_id FROM source_heads WHERE source_id = ?`, source.ID).Scan(&headAfter); err != nil {
		t.Fatal(err)
	}
	if err := store.Reader.QueryRow(`SELECT status FROM source_generations WHERE generation_id = ?`, abandonedID).Scan(&status); err != nil {
		t.Fatal(err)
	}
	if headAfter != headBefore || status != "abandoned" {
		t.Fatalf("after abandon: head=%q (want %q), status=%q", headAfter, headBefore, status)
	}
	var retained int
	if err := store.Reader.QueryRow(`SELECT COUNT(*) FROM puzzle_occurrences WHERE generation_id = ?`, abandonedID).Scan(&retained); err != nil {
		t.Fatal(err)
	}
	if retained != 0 {
		t.Fatalf("abandon retained %d final occurrences, want 0", retained)
	}
	if _, err := os.Stat(stagePath); !os.IsNotExist(err) {
		t.Fatalf("abandoned generation stage stat = %v, want not-exist", err)
	}
	if _, err := catalog.Get(context.Background(), PuzzleKey{Fingerprint: "active", SourceID: source.ID}); err != nil {
		t.Fatalf("old head is unreadable after abandon: %v", err)
	}
}
