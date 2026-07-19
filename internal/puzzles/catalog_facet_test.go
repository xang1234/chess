package puzzles

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"
	"testing"
)

func TestSealBuildsThemesFromFinalCrossBatchOccurrences(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	ctx := context.Background()
	source := testSource("facet-last-wins", "test", "/facet-last-wins")
	importing := beginGenerationImport(t, catalog, source)

	first := testTrainingPuzzle(source, "repeated", 1400, "removed-theme")
	first.Occurrence.Ordinal = 1
	if err := importing.Add(ctx, first); err != nil {
		t.Fatal(err)
	}
	for index := 1; index < generationImportBatchSize; index++ {
		puzzle := testTrainingPuzzle(source, fmt.Sprintf("retained-%04d", index), 1400, "retained-theme")
		puzzle.Occurrence.Ordinal = int64(index + 1)
		if err := importing.Add(ctx, puzzle); err != nil {
			t.Fatal(err)
		}
	}

	last := testTrainingPuzzle(source, "repeated", 1400)
	last.Occurrence.Ordinal = int64(generationImportBatchSize + 1)
	if err := importing.Add(ctx, last); err != nil {
		t.Fatal(err)
	}
	if _, err := importing.Seal(ctx, "facet-last-wins-checksum"); err != nil {
		t.Fatal(err)
	}

	generationID := generationIDForPath(t, store.Reader, source.Path)
	if got := generationThemes(t, store.Reader, generationID); !reflect.DeepEqual(got, []string{"retained-theme"}) {
		t.Fatalf("sealed generation themes = %q, want only retained-theme", got)
	}
	if err := importing.Activate(ctx); err != nil {
		t.Fatal(err)
	}

	// ActiveThemes is a sealed-generation facet read, not a rescan of occurrence themes.
	if _, err := store.Writer.ExecContext(ctx, `INSERT INTO occurrence_themes(
		generation_id, fingerprint, theme
	) VALUES (?, 'repeated', 'late-raw-theme')`, generationID); err != nil {
		t.Fatal(err)
	}
	if got, err := catalog.ActiveThemes(ctx); err != nil {
		t.Fatal(err)
	} else if !reflect.DeepEqual(got, []string{"retained-theme"}) {
		t.Fatalf("active themes after raw occurrence mutation = %q, want sealed facet", got)
	}
}

func TestSealBuildsEmptyThemeFacetForEmptyGeneration(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	ctx := context.Background()
	source := testSource("facet-empty", "test", "/facet-empty")
	importing := beginGenerationImport(t, catalog, source)

	if _, err := importing.Seal(ctx, "facet-empty-checksum"); err != nil {
		t.Fatal(err)
	}
	generationID := generationIDForPath(t, store.Reader, source.Path)
	if got := generationThemes(t, store.Reader, generationID); len(got) != 0 {
		t.Fatalf("empty generation themes = %q, want none", got)
	}
	if err := importing.Activate(ctx); err != nil {
		t.Fatal(err)
	}
	if got, err := catalog.ActiveThemes(ctx); err != nil {
		t.Fatal(err)
	} else if got == nil || len(got) != 0 {
		t.Fatalf("active themes for empty generation = %#v, want non-nil empty", got)
	}
}

func TestSealPersistsMaximumSolutionPliesFromGenerationWinners(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	source := testSource("summary-max", "test", "/summary-max")
	short := testTrainingPuzzle(source, "short", 0)
	short.Occurrence.Rating = nil
	short.Core.Solution = linearTestSolution([]string{"f7f8", "h8h7"})
	short.Core.SolutionPlies = 2
	long := testTrainingPuzzle(source, "long", 0)
	long.Occurrence.Rating = nil
	long.Occurrence.Ordinal = 2
	long.Core.Solution = linearTestSolution([]string{
		"f7f8", "h8h7", "f8f7", "h7h8", "f7f8", "h8h7", "f8f7",
	})
	long.Core.SolutionPlies = 7

	importing := beginGenerationImport(t, catalog, source)
	if err := importing.Add(context.Background(), short); err != nil {
		t.Fatal(err)
	}
	if err := importing.Add(context.Background(), long); err != nil {
		t.Fatal(err)
	}
	if _, err := importing.Seal(context.Background(), "summary-max-checksum"); err != nil {
		t.Fatal(err)
	}
	assertGenerationMaximumSolutionPlies(t, store.Reader, source.Path, 7)
}

func TestSealPersistsZeroMaximumForEmptyGeneration(t *testing.T) {
	catalog, store := openTestGenerationalCatalog(t)
	source := testSource("summary-empty", "test", "/summary-empty")
	importing := beginGenerationImport(t, catalog, source)
	if _, err := importing.Seal(context.Background(), "summary-empty-checksum"); err != nil {
		t.Fatal(err)
	}
	assertGenerationMaximumSolutionPlies(t, store.Reader, source.Path, 0)
}

func assertGenerationMaximumSolutionPlies(
	t *testing.T,
	db generationSummaryQueryer,
	sourcePath string,
	want int,
) {
	t.Helper()
	var got int
	if err := db.QueryRowContext(
		context.Background(),
		`SELECT maximum_solution_plies
		 FROM source_generations
		 WHERE source_path = ?`,
		sourcePath,
	).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("generation maximum solution plies = %d, want %d", got, want)
	}
}

type generationSummaryQueryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type generationThemeQueryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func generationThemes(t *testing.T, db generationThemeQueryer, generationID string) []string {
	t.Helper()
	rows, err := db.QueryContext(
		context.Background(),
		`SELECT theme FROM generation_themes WHERE generation_id = ? ORDER BY theme`,
		generationID,
	)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	themes := make([]string, 0)
	for rows.Next() {
		var theme string
		if err := rows.Scan(&theme); err != nil {
			t.Fatal(err)
		}
		themes = append(themes, theme)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return themes
}
