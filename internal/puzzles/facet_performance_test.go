//go:build performance

package puzzles

import (
	"context"
	"reflect"
	"testing"
	"time"
)

func TestGenerationThemeFacetSealScanIsNegligibleAtHundredThousandRows(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	ctx := context.Background()
	source := testSource("facet-performance", "synthetic", "/facet-performance")
	importing := beginGenerationImport(t, catalog, source)
	generationID := generationIDForPath(t, store.Reader, source.Path)

	tx, err := store.Writer.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	const rowCount = 100_000
	const chunkSize = 1_000
	for start := 1; start <= rowCount; start += chunkSize {
		end := min(start+chunkSize-1, rowCount)
		if _, err := tx.ExecContext(ctx, `WITH RECURSIVE sequence(value) AS (
			SELECT ?
			UNION ALL
			SELECT value + 1 FROM sequence WHERE value < ?
		)
		INSERT INTO puzzle_cores(
			fingerprint, displayed_fen, solver, solution_json, solution_plies
		)
		SELECT printf('facet-%07d', value),
		       '7k/5Q2/6K1/8/8/8/8/8 w - - 0 1',
		       'white',
		       '[{"uci":"f7f8"}]',
		       1
		FROM sequence`, start, end); err != nil {
			t.Fatal(err)
		}
		if _, err := tx.ExecContext(ctx, `WITH RECURSIVE sequence(value) AS (
			SELECT ?
			UNION ALL
			SELECT value + 1 FROM sequence WHERE value < ?
		)
		INSERT INTO puzzle_occurrences(
			generation_id, fingerprint, metadata_json, ordinal
		)
		SELECT ?, printf('facet-%07d', value), '{}', value
		FROM sequence`, start, end, generationID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO occurrence_themes(
		generation_id, fingerprint, theme
	)
	SELECT occurrence.generation_id, occurrence.fingerprint, themes.theme
	FROM puzzle_occurrences occurrence
	CROSS JOIN (
		SELECT 'fork' AS theme
		UNION ALL SELECT 'middlegame'
		UNION ALL SELECT 'sacrifice'
		UNION ALL SELECT 'short'
	) themes
	WHERE occurrence.generation_id = ?`, generationID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	staged := importing.(*sqliteGenerationImport)
	if err := staged.stage.closeAndRemove(); err != nil {
		t.Fatal(err)
	}
	staged.stage = nil
	staged.state = generationImportFinalizing

	started := time.Now()
	if err := staged.sealMaterializedGeneration(ctx, "facet-performance-checksum"); err != nil {
		t.Fatal(err)
	}
	staged.state = generationImportSealed
	elapsed := time.Since(started)
	want := []string{"fork", "middlegame", "sacrifice", "short"}
	if got := generationThemes(t, store.Reader, generationID); !reflect.DeepEqual(got, want) {
		t.Fatalf("generation themes = %q, want %q", got, want)
	}
	if elapsed >= 2*time.Second {
		t.Fatalf("100k-row generation theme facet seal scan took %s, want less than 2s", elapsed)
	}
	t.Logf("facet_seal_scan_elapsed=%s occurrences=%d occurrence_theme_rows=%d distinct_themes=%d", elapsed, rowCount, rowCount*4, len(want))
}
