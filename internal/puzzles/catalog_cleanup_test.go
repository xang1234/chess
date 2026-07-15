package puzzles

import (
	"context"
	"fmt"
	"testing"

	"chess-trainer/internal/storage"
)

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
	var generationRows, occurrenceRows, themeRows, headRows int
	if err := store.Reader.QueryRow(`SELECT COUNT(*) FROM source_generations WHERE generation_id = ?`, activeGeneration).Scan(&generationRows); err != nil {
		t.Fatal(err)
	}
	if err := store.Reader.QueryRow(`SELECT COUNT(*) FROM puzzle_occurrences WHERE generation_id = ?`, activeGeneration).Scan(&occurrenceRows); err != nil {
		t.Fatal(err)
	}
	if err := store.Reader.QueryRow(`SELECT COUNT(*) FROM occurrence_themes WHERE generation_id = ?`, activeGeneration).Scan(&themeRows); err != nil {
		t.Fatal(err)
	}
	if err := store.Reader.QueryRow(`SELECT COUNT(*) FROM source_heads WHERE generation_id = ?`, activeGeneration).Scan(&headRows); err != nil {
		t.Fatal(err)
	}
	if generationRows != 1 || occurrenceRows != 1 || themeRows != 2 || headRows != 1 {
		t.Fatalf(
			"active physical rows after cleanup: generation=%d occurrence=%d themes=%d head=%d",
			generationRows,
			occurrenceRows,
			themeRows,
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
		(SELECT COUNT(*) FROM occurrence_themes)`).Scan(&total); err != nil {
		t.Fatal(err)
	}
	return total
}
