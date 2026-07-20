package openings

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/importing"
)

func copyMiniCoursePack(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mini.ctcourse")
	if err := os.WriteFile(path, readMiniCoursePack(t), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestCourseImporterInspectsAndImportsExactCounts(t *testing.T) {
	ctx := context.Background()
	catalog := NewSQLiteCatalog(openCourseCatalogTestDB(t))
	importer := NewImporter(catalog, chessrules.Rules{})
	path := copyMiniCoursePack(t)

	inspection, err := importer.Inspect(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if inspection.Format != FormatCoursePack || inspection.FormatLabel != "Opening course" {
		t.Fatalf("inspection format = %q label = %q", inspection.Format, inspection.FormatLabel)
	}
	if inspection.SourceID != "synthetic-italian" || inspection.SourceIDOrigin != importing.SourceIDEmbedded {
		t.Fatalf("inspection source = %q origin = %q", inspection.SourceID, inspection.SourceIDOrigin)
	}
	if inspection.ReplacesExisting {
		t.Fatal("new course inspection unexpectedly reports replacement")
	}

	var progress []importing.Progress
	report, err := importer.Import(ctx, inspection, func(update importing.Progress) {
		progress = append(progress, update)
	})
	if err != nil {
		t.Fatal(err)
	}
	wantCounts := map[string]int64{
		"chapters":    3,
		"positions":   11,
		"moves":       10,
		"variations":  7,
		"notes":       1,
		"lessons":     1,
		"lessonEdges": 0,
		"activities":  5,
		"prompts":     2,
		"warnings":    0,
	}
	if report.Accepted != 1 || !reflect.DeepEqual(report.Counts, wantCounts) {
		t.Fatalf("report = %#v, want counts %#v", report, wantCounts)
	}
	if len(progress) < 4 || progress[0].Phase != importing.PhaseDetecting ||
		progress[len(progress)-1].Phase != importing.PhaseActivating {
		t.Fatalf("progress = %#v", progress)
	}
	for index := 1; index < len(progress); index++ {
		if progress[index].RowsRead < progress[index-1].RowsRead ||
			progress[index].BytesRead < progress[index-1].BytesRead {
			t.Fatalf("progress moved backwards at %d: %#v", index, progress)
		}
	}

	replacement, err := importer.Inspect(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if !replacement.ReplacesExisting {
		t.Fatal("active course inspection did not report replacement")
	}
}

func TestCourseImporterRejectsFileLargerThanLimit(t *testing.T) {
	path := filepath.Join(t.TempDir(), "large.ctcourse")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate(MaxCoursePackBytes + 1); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	importer := NewImporter(NewSQLiteCatalog(openCourseCatalogTestDB(t)), chessrules.Rules{})
	_, err = importer.Inspect(context.Background(), path)
	if err == nil || !strings.Contains(err.Error(), "32 MiB") {
		t.Fatalf("Inspect() error = %v, want size limit", err)
	}
}

func TestCourseImporterInvalidReplacementLeavesOldHeadActive(t *testing.T) {
	ctx := context.Background()
	catalog := NewSQLiteCatalog(openCourseCatalogTestDB(t))
	importer := NewImporter(catalog, chessrules.Rules{})
	path := copyMiniCoursePack(t)
	inspection, err := importer.Inspect(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := importer.Import(ctx, inspection, nil); err != nil {
		t.Fatal(err)
	}
	oldHead, err := catalog.ActiveGenerationID(ctx, inspection.SourceID)
	if err != nil {
		t.Fatal(err)
	}

	invalid := strings.Replace(string(readMiniCoursePack(t)), `"uci":"e2e4"`, `"uci":"e2e5"`, 1)
	if err := os.WriteFile(path, []byte(invalid), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err = importer.Import(ctx, inspection, nil)
	if err == nil || !strings.Contains(err.Error(), "illegal_move") {
		t.Fatalf("Import() error = %v, want illegal move", err)
	}
	active, err := catalog.ActiveGenerationID(ctx, inspection.SourceID)
	if err != nil || active != oldHead {
		t.Fatalf("head after invalid import = %q err=%v, want %q", active, err, oldHead)
	}
}

func TestCourseImporterCancellationRollsBackReplacement(t *testing.T) {
	ctx := context.Background()
	catalog := NewSQLiteCatalog(openCourseCatalogTestDB(t))
	importer := NewImporter(catalog, chessrules.Rules{})
	path := copyMiniCoursePack(t)
	inspection, err := importer.Inspect(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := importer.Import(ctx, inspection, nil); err != nil {
		t.Fatal(err)
	}
	oldHead, err := catalog.ActiveGenerationID(ctx, inspection.SourceID)
	if err != nil {
		t.Fatal(err)
	}

	cancelled, cancel := context.WithCancel(ctx)
	catalog.afterInsert = func(table string, ordinal int) {
		if table == "course_positions" && ordinal == 2 {
			cancel()
		}
	}
	_, err = importer.Import(cancelled, inspection, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Import() error = %v, want context cancellation", err)
	}
	active, err := catalog.ActiveGenerationID(ctx, inspection.SourceID)
	if err != nil || active != oldHead {
		t.Fatalf("head after cancelled import = %q err=%v, want %q", active, err, oldHead)
	}
}
