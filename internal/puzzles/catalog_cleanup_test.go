package puzzles

import (
	"context"
	"fmt"
	"os"
	"testing"

	"chess-trainer/internal/storage"

	"github.com/google/uuid"
)

func TestRecoverStartupRemovesOrphanGenerationStage(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	activeSource := testSource("orphan-stage-active", "test", "/orphan-stage-active")
	sealAndActivate(
		t,
		beginGenerationImport(t, catalog, activeSource),
		testTrainingPuzzle(activeSource, "orphan-stage-active", 1200),
	)
	interruptedSource := testSource("orphan-stage-building", "test", "/orphan-stage-building")
	interrupted := beginGenerationImport(t, catalog, interruptedSource)
	staged, ok := interrupted.(*sqliteGenerationImport)
	if !ok {
		t.Fatalf("generation import type = %T, want *sqliteGenerationImport", interrupted)
	}
	for index := range generationImportBatchSize {
		puzzle := testTrainingPuzzle(
			interruptedSource,
			fmt.Sprintf("orphan-stage-%04d", index),
			1300,
		)
		puzzle.Occurrence.Ordinal = int64(index + 1)
		if err := interrupted.Add(context.Background(), puzzle); err != nil {
			t.Fatal(err)
		}
	}
	stagePath := staged.stage.path
	if err := staged.stage.close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stagePath); err != nil {
		t.Fatalf("stat simulated orphan stage: %v", err)
	}
	stageSidecars := []string{stagePath + "-journal", stagePath + "-wal", stagePath + "-shm"}
	for _, path := range stageSidecars {
		if err := os.WriteFile(path, []byte("orphan"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	winnerPath := generationWinnerPathForStage(stagePath)
	if err := os.WriteFile(winnerPath, []byte("orphan winner"), 0o600); err != nil {
		t.Fatal(err)
	}
	winnerSidecars := []string{winnerPath + "-journal", winnerPath + "-wal", winnerPath + "-shm"}
	for _, path := range winnerSidecars {
		if err := os.WriteFile(path, []byte("orphan winner sidecar"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	mainPath, err := puzzleDatabasePath(context.Background(), store.Writer)
	if err != nil {
		t.Fatal(err)
	}
	decoys := []string{
		mainPath + ".stage-not-a-generation.sqlite",
		mainPath + ".stage-" + uuid.NewString() + ".sqlite.backup",
		mainPath + ".winner-not-a-generation.sqlite",
		mainPath + ".winner-" + uuid.NewString() + ".sqlite.backup",
	}
	for _, path := range decoys {
		if err := os.WriteFile(path, []byte("preserve"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	stageShapedDirectory := mainPath + ".stage-" + uuid.NewString() + ".sqlite"
	if err := os.Mkdir(stageShapedDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	winnerShapedDirectory := mainPath + ".winner-" + uuid.NewString() + ".sqlite"
	if err := os.Mkdir(winnerShapedDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		for _, path := range append(decoys, stageShapedDirectory, winnerShapedDirectory) {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				t.Errorf("remove preserved stage decoy %q: %v", path, err)
			}
		}
	})

	resumed := NewSQLiteCatalog(store.Reader, store.Writer)
	if err := resumed.RecoverStartup(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stagePath); !os.IsNotExist(err) {
		t.Fatalf("orphan stage stat after recovery = %v, want not-exist", err)
	}
	for _, path := range stageSidecars {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("orphan stage sidecar %q stat = %v, want not-exist", path, err)
		}
	}
	if _, err := os.Stat(winnerPath); !os.IsNotExist(err) {
		t.Fatalf("orphan winner stat after recovery = %v, want not-exist", err)
	}
	for _, path := range winnerSidecars {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("orphan winner sidecar %q stat = %v, want not-exist", path, err)
		}
	}
	for _, path := range decoys {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("stage decoy %q after recovery: %v", path, err)
		}
	}
	if info, err := os.Stat(stageShapedDirectory); err != nil || !info.IsDir() {
		t.Fatalf("stage-shaped directory after recovery: info=%v err=%v", info, err)
	}
	if info, err := os.Stat(winnerShapedDirectory); err != nil || !info.IsDir() {
		t.Fatalf("winner-shaped directory after recovery: info=%v err=%v", info, err)
	}
	if _, err := resumed.Get(context.Background(), PuzzleKey{
		Fingerprint: "orphan-stage-active",
		SourceID:    activeSource.ID,
	}); err != nil {
		t.Fatalf("old head after orphan-stage recovery: %v", err)
	}
}

func TestRecoverStartupMarksBuildingAbandoned(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	activeSource := testSource("recovery-active", "test", "/recovery-active")
	sealAndActivate(t, beginGenerationImport(t, catalog, activeSource), testTrainingPuzzle(activeSource, "active", 1200))
	buildingSource := testSource("recovery-building", "test", "/recovery-building")
	beginGenerationImport(t, catalog, buildingSource)
	sealedSource := testSource("recovery-sealed", "test", "/recovery-sealed")
	sealed := beginGenerationImport(t, catalog, sealedSource)
	if err := sealed.Add(context.Background(), testTrainingPuzzle(sealedSource, "sealed", 1300)); err != nil {
		t.Fatal(err)
	}
	if _, err := sealed.Seal(context.Background(), "sealed-checksum"); err != nil {
		t.Fatal(err)
	}

	if err := catalog.RecoverStartup(context.Background()); err != nil {
		t.Fatal(err)
	}
	statuses := map[string]string{}
	rows, err := store.Reader.Query(`SELECT source_path, status FROM source_generations`)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var path, status string
		if err := rows.Scan(&path, &status); err != nil {
			rows.Close()
			t.Fatal(err)
		}
		statuses[path] = status
	}
	if err := rows.Close(); err != nil {
		t.Fatal(err)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if statuses["/recovery-building"] != "abandoned" ||
		statuses["/recovery-active"] != "sealed" ||
		statuses["/recovery-sealed"] != "sealed" {
		t.Fatalf("statuses after recovery = %#v", statuses)
	}
	if _, err := catalog.Get(context.Background(), PuzzleKey{Fingerprint: "active", SourceID: activeSource.ID}); err != nil {
		t.Fatalf("recovery changed active head: %v", err)
	}
}

func TestCleanupNeverTouchesActiveGeneration(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	activeSource := testSource("cleanup-active", "test", "/cleanup-active")
	active := testTrainingPuzzle(activeSource, "active-core", 1400, "fork", "pin")
	sealAndActivate(t, beginGenerationImport(t, catalog, activeSource), active)
	activeGeneration := generationIDForPath(t, store.Reader, activeSource.Path)

	eligibleSource := testSource("cleanup-eligible", "test", "/cleanup-eligible")
	eligible := beginGenerationImport(t, catalog, eligibleSource)
	if err := eligible.Add(context.Background(), testTrainingPuzzle(eligibleSource, "eligible-core", 1500, "skewer")); err != nil {
		t.Fatal(err)
	}
	if _, err := eligible.Seal(context.Background(), "eligible-checksum"); err != nil {
		t.Fatal(err)
	}
	cleanupUntilDone(t, catalog, 2)

	if _, err := catalog.Get(context.Background(), PuzzleKey{Fingerprint: "active-core", SourceID: activeSource.ID}); err != nil {
		t.Fatalf("active puzzle removed by cleanup: %v", err)
	}
	var generationRows, occurrenceRows, ratingRows, themeRows, facetRows, headRows int
	if err := store.Reader.QueryRow(`SELECT COUNT(*) FROM source_generations WHERE generation_id = ?`, activeGeneration).Scan(&generationRows); err != nil {
		t.Fatal(err)
	}
	if err := store.Reader.QueryRow(`SELECT COUNT(*) FROM puzzle_occurrences WHERE generation_id = ?`, activeGeneration).Scan(&occurrenceRows); err != nil {
		t.Fatal(err)
	}
	if err := store.Reader.QueryRow(`SELECT COUNT(*) FROM occurrence_ratings WHERE generation_id = ?`, activeGeneration).Scan(&ratingRows); err != nil {
		t.Fatal(err)
	}
	if err := store.Reader.QueryRow(`SELECT COUNT(*) FROM occurrence_themes WHERE generation_id = ?`, activeGeneration).Scan(&themeRows); err != nil {
		t.Fatal(err)
	}
	if err := store.Reader.QueryRow(`SELECT COUNT(*) FROM generation_themes WHERE generation_id = ?`, activeGeneration).Scan(&facetRows); err != nil {
		t.Fatal(err)
	}
	if err := store.Reader.QueryRow(`SELECT COUNT(*) FROM source_heads WHERE generation_id = ?`, activeGeneration).Scan(&headRows); err != nil {
		t.Fatal(err)
	}
	if generationRows != 1 || occurrenceRows != 1 || ratingRows != 1 || themeRows != 2 || facetRows != 2 || headRows != 1 {
		t.Fatalf(
			"active physical rows after cleanup: generation=%d occurrence=%d ratings=%d themes=%d facet=%d head=%d",
			generationRows,
			occurrenceRows,
			ratingRows,
			themeRows,
			facetRows,
			headRows,
		)
	}
	var eligibleGenerations int
	if err := store.Reader.QueryRow(`SELECT COUNT(*) FROM source_generations WHERE source_id = ?`, eligibleSource.ID).Scan(&eligibleGenerations); err != nil {
		t.Fatal(err)
	}
	if eligibleGenerations != 0 {
		t.Fatalf("eligible generations after cleanup = %d, want 0", eligibleGenerations)
	}
}

func TestCleanupUsesOnePhysicalRowBudget(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	source := testSource("budget", "test", "/budget")
	importing := beginGenerationImport(t, catalog, source)
	for index := range 4 {
		puzzle := testTrainingPuzzle(
			source,
			fmt.Sprintf("budget-%d", index),
			1200+index,
			fmt.Sprintf("theme-%d-a", index),
			fmt.Sprintf("theme-%d-b", index),
		)
		puzzle.Occurrence.Ordinal = int64(index + 1)
		if err := importing.Add(context.Background(), puzzle); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := importing.Seal(context.Background(), "budget-checksum"); err != nil {
		t.Fatal(err)
	}

	const limit = 3
	previous := physicalCatalogRowCount(t, store)
	for call := 1; call <= 20; call++ {
		more, err := catalog.CleanupBatch(context.Background(), limit)
		if err != nil {
			t.Fatal(err)
		}
		current := physicalCatalogRowCount(t, store)
		deleted := previous - current
		if deleted < 0 || deleted > limit {
			t.Fatalf("cleanup call %d deleted %d physical rows, budget %d", call, deleted, limit)
		}
		if more && deleted == 0 {
			t.Fatalf("cleanup call %d reported more without deleting rows", call)
		}
		previous = current
		if !more {
			if current != 0 {
				t.Fatalf("cleanup converged with %d managed rows remaining", current)
			}
			return
		}
	}
	t.Fatal("cleanup did not converge within 20 calls")
}

func TestCleanupOccurrencePlanStreamsOneGeneration(t *testing.T) {
	_, store := openTestGenerationalCatalog(t)
	details := catalogQueryPlanDetails(t, store.Reader, `
		SELECT occurrence.generation_id, occurrence.fingerprint
		FROM puzzle_occurrences occurrence
		WHERE occurrence.generation_id = (
		  SELECT MIN(generation.generation_id)
		  FROM source_generations generation
		  WHERE generation.status IN ('abandoned', 'sealed')
		    AND NOT EXISTS (
		      SELECT 1 FROM source_heads head
		      WHERE head.generation_id = generation.generation_id
		    )
		)
		  AND NOT EXISTS (
		    SELECT 1 FROM occurrence_themes theme
		    WHERE theme.generation_id = occurrence.generation_id
		      AND theme.fingerprint = occurrence.fingerprint
		  )
		  AND NOT EXISTS (
		    SELECT 1 FROM occurrence_ratings rated
		    WHERE rated.generation_id = occurrence.generation_id
		      AND rated.fingerprint = occurrence.fingerprint
		  )
		ORDER BY occurrence.fingerprint
		LIMIT ?`, 1_000)
	assertQueryPlanContains(
		t,
		details,
		"occurrence USING COVERING INDEX idx_occurrences_generation (generation_id=?)",
	)
	assertQueryPlanNotContains(t, details, "SCAN occurrence")
	assertQueryPlanNotContains(t, details, "USE TEMP B-TREE")
}

func TestCleanupResumesAndConverges(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	activeSource := testSource("resume-active", "test", "/resume-active")
	sealAndActivate(t, beginGenerationImport(t, catalog, activeSource), testTrainingPuzzle(activeSource, "keep", 1100, "keep"))
	for index := range 2 {
		source := testSource(
			fmt.Sprintf("resume-%d", index),
			"test",
			fmt.Sprintf("/resume-%d", index),
		)
		importing := beginGenerationImport(t, catalog, source)
		puzzle := testTrainingPuzzle(source, fmt.Sprintf("remove-%d", index), 1200+index, "remove")
		if err := importing.Add(context.Background(), puzzle); err != nil {
			t.Fatal(err)
		}
		if _, err := importing.Seal(context.Background(), fmt.Sprintf("checksum-%d", index)); err != nil {
			t.Fatal(err)
		}
	}

	before := physicalCatalogRowCount(t, store)
	more, err := catalog.CleanupBatch(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if !more {
		t.Fatal("first one-row cleanup unexpectedly converged")
	}
	afterFirst := physicalCatalogRowCount(t, store)
	if before-afterFirst != 1 {
		t.Fatalf("first cleanup deleted %d rows, want exactly 1", before-afterFirst)
	}

	resumed := NewSQLiteCatalog(store.Reader, store.Writer)
	cleanupUntilDone(t, resumed, 1)
	if _, err := resumed.Get(context.Background(), PuzzleKey{Fingerprint: "keep", SourceID: activeSource.ID}); err != nil {
		t.Fatalf("active puzzle missing after resumed cleanup: %v", err)
	}
	var eligible int
	if err := store.Reader.QueryRow(`SELECT COUNT(*)
		FROM source_generations generation
		WHERE generation.status IN ('abandoned', 'sealed')
		  AND NOT EXISTS (
		    SELECT 1 FROM source_heads head
		    WHERE head.generation_id = generation.generation_id
		  )`).Scan(&eligible); err != nil {
		t.Fatal(err)
	}
	if eligible != 0 {
		t.Fatalf("eligible generations after resumed cleanup = %d", eligible)
	}
}

func TestCleanupPreservesSharedCoreAndRemovesOrphanCore(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	activeSource := testSource("shared-active", "test", "/shared-active")
	sealAndActivate(t, beginGenerationImport(t, catalog, activeSource), testTrainingPuzzle(activeSource, "shared", 1200, "active"))
	eligibleSource := testSource("shared-eligible", "test", "/shared-eligible")
	eligible := beginGenerationImport(t, catalog, eligibleSource)
	if err := eligible.Add(context.Background(), testTrainingPuzzle(eligibleSource, "shared", 1400, "eligible")); err != nil {
		t.Fatal(err)
	}
	orphan := testTrainingPuzzle(eligibleSource, "orphan", 1500, "orphan")
	orphan.Occurrence.Ordinal = 2
	if err := eligible.Add(context.Background(), orphan); err != nil {
		t.Fatal(err)
	}
	if _, err := eligible.Seal(context.Background(), "shared-eligible-checksum"); err != nil {
		t.Fatal(err)
	}
	cleanupUntilDone(t, catalog, 2)

	var sharedCores, orphanCores int
	if err := store.Reader.QueryRow(`SELECT COUNT(*) FROM puzzle_cores WHERE fingerprint = 'shared'`).Scan(&sharedCores); err != nil {
		t.Fatal(err)
	}
	if err := store.Reader.QueryRow(`SELECT COUNT(*) FROM puzzle_cores WHERE fingerprint = 'orphan'`).Scan(&orphanCores); err != nil {
		t.Fatal(err)
	}
	if sharedCores != 1 || orphanCores != 0 {
		t.Fatalf("cores after cleanup: shared=%d orphan=%d, want 1 and 0", sharedCores, orphanCores)
	}
	if _, err := catalog.Get(context.Background(), PuzzleKey{Fingerprint: "shared", SourceID: activeSource.ID}); err != nil {
		t.Fatalf("shared active core is unreadable: %v", err)
	}
}

func cleanupUntilDone(t *testing.T, catalog *SQLiteCatalog, limit int) {
	t.Helper()
	for call := 1; call <= 10_000; call++ {
		more, err := catalog.CleanupBatch(context.Background(), limit)
		if err != nil {
			t.Fatal(err)
		}
		if !more {
			return
		}
	}
	t.Fatal("cleanup did not converge")
}

func physicalCatalogRowCount(t *testing.T, store *storage.PuzzleStore) int {
	t.Helper()
	var total int
	if err := store.Reader.QueryRow(`SELECT
		(SELECT COUNT(*) FROM sources) +
		(SELECT COUNT(*) FROM source_generations) +
		(SELECT COUNT(*) FROM source_heads) +
		(SELECT COUNT(*) FROM puzzle_cores) +
		(SELECT COUNT(*) FROM puzzle_occurrences) +
		(SELECT COUNT(*) FROM occurrence_ratings) +
		(SELECT COUNT(*) FROM occurrence_themes) +
		(SELECT COUNT(*) FROM generation_themes)`).Scan(&total); err != nil {
		t.Fatal(err)
	}
	return total
}
