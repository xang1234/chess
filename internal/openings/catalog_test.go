package openings

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"testing"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/storage"
)

func openCourseCatalogTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := storage.Open(t.TempDir() + "/courses.sqlite")
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Migrate(db, "courses"); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func compileMiniCourse(t *testing.T) CompiledCourse {
	t.Helper()
	compiled, err := Compile(decodeMiniPack(t), chessrules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	return compiled
}

func TestSQLiteCatalogRoundTripsTeachingTree(t *testing.T) {
	ctx := context.Background()
	catalog := NewSQLiteCatalog(openCourseCatalogTestDB(t))
	want := compileTreeCourse(t)
	result, err := catalog.Replace(ctx, want, "/private/tree.ctcourse", "sha-tree")
	if err != nil {
		t.Fatal(err)
	}
	got, err := catalog.LoadGeneration(ctx, result.GenerationID)
	if err != nil {
		t.Fatal(err)
	}
	if got.RootLessonID != "giuoco-plan" || len(got.LessonChildren["giuoco-plan"]) != 1 {
		t.Fatalf("tree=%+v", got)
	}
	if !reflect.DeepEqual(got.Lessons["giuoco-plan"].Activities, want.Lessons["giuoco-plan"].Activities) {
		t.Fatalf("activities=%#v", got.Lessons["giuoco-plan"].Activities)
	}
}

func TestSQLiteCatalogReplacesHeadAtomicallyAndRetainsOldGeneration(t *testing.T) {
	ctx := context.Background()
	catalog := NewSQLiteCatalog(openCourseCatalogTestDB(t))
	compiledV1 := compileMiniCourse(t)
	compiledV2 := compiledV1
	compiledV2.Pack.ContentVersion = "2.0.0"
	compiledV2.Pack.Title = "Synthetic Italian, revised"

	first, err := catalog.Replace(ctx, compiledV1, "/private/v1.ctcourse", "sha-v1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := catalog.Replace(ctx, compiledV2, "/private/v2.ctcourse", "sha-v2")
	if err != nil {
		t.Fatal(err)
	}
	if first.GenerationID == second.GenerationID {
		t.Fatal("replacement reused generation ID")
	}
	if second.PreviousHead != first.GenerationID {
		t.Fatalf("previous head = %q, want %q", second.PreviousHead, first.GenerationID)
	}

	active, err := catalog.ActiveGenerationID(ctx, compiledV1.Pack.CourseID)
	if err != nil || active != second.GenerationID {
		t.Fatalf("active = %q err=%v", active, err)
	}
	old, err := catalog.LoadGeneration(ctx, first.GenerationID)
	if err != nil {
		t.Fatalf("old generation disappeared before cleanup: %v", err)
	}
	if old.Pack.ContentVersion != "1.0.0" {
		t.Fatalf("old content version = %q", old.Pack.ContentVersion)
	}
	loaded, err := catalog.LoadActive(ctx, compiledV1.Pack.CourseID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Pack.Title != compiledV2.Pack.Title || len(loaded.Moves) != len(compiledV2.Moves) {
		t.Fatalf("loaded active course = title %q, moves %d", loaded.Pack.Title, len(loaded.Moves))
	}
	if loaded.Pack.RootFEN != compiledV2.Pack.RootFEN {
		t.Fatalf("loaded root FEN = %q, want %q", loaded.Pack.RootFEN, compiledV2.Pack.RootFEN)
	}
	if loaded.Moves["white-c3"].SAN != compiledV2.Moves["white-c3"].SAN {
		t.Fatalf("loaded SAN = %q, want %q", loaded.Moves["white-c3"].SAN, compiledV2.Moves["white-c3"].SAN)
	}
	if loaded.Prompts["recall-c3"].SemanticFingerprint != compiledV2.Prompts["recall-c3"].SemanticFingerprint {
		t.Fatal("loaded prompt fingerprint changed")
	}
	if len(loaded.Lessons["giuoco-c3"].Steps) != 5 {
		t.Fatalf("loaded lesson steps = %#v", loaded.Lessons["giuoco-c3"].Steps)
	}
	if len(loaded.Lessons["giuoco-c3"].Activities) != 5 || loaded.RootLessonID != "giuoco-c3" {
		t.Fatalf("loaded legacy teaching model = lesson %#v root %q", loaded.Lessons["giuoco-c3"], loaded.RootLessonID)
	}
	summaries, err := catalog.ListActive(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 1 || summaries[0].GenerationID != second.GenerationID {
		t.Fatalf("active summaries = %#v", summaries)
	}
}

func TestSQLiteCatalogCancellationRollsBackReplacement(t *testing.T) {
	ctx := context.Background()
	catalog := NewSQLiteCatalog(openCourseCatalogTestDB(t))
	compiled := compileMiniCourse(t)
	first, err := catalog.Replace(ctx, compiled, "/private/v1.ctcourse", "sha-v1")
	if err != nil {
		t.Fatal(err)
	}

	cancelled, cancel := context.WithCancel(ctx)
	catalog.afterInsert = func(table string, ordinal int) {
		if table == "course_moves" && ordinal == 2 {
			cancel()
		}
	}
	_, err = catalog.Replace(cancelled, compiled, "/private/v2.ctcourse", "sha-v2")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Replace() error = %v, want context cancellation", err)
	}
	active, err := catalog.ActiveGenerationID(ctx, compiled.Pack.CourseID)
	if err != nil || active != first.GenerationID {
		t.Fatalf("active after rollback = %q err=%v, want %q", active, err, first.GenerationID)
	}
}

func TestSQLiteCatalogCleanupSkipsHeadsAndProtectedGenerations(t *testing.T) {
	ctx := context.Background()
	catalog := NewSQLiteCatalog(openCourseCatalogTestDB(t))
	compiled := compileMiniCourse(t)
	first, err := catalog.Replace(ctx, compiled, "/private/v1.ctcourse", "sha-v1")
	if err != nil {
		t.Fatal(err)
	}
	compiled.Pack.ContentVersion = "2.0.0"
	second, err := catalog.Replace(ctx, compiled, "/private/v2.ctcourse", "sha-v2")
	if err != nil {
		t.Fatal(err)
	}
	compiled.Pack.ContentVersion = "3.0.0"
	third, err := catalog.Replace(ctx, compiled, "/private/v3.ctcourse", "sha-v3")
	if err != nil {
		t.Fatal(err)
	}

	more, err := catalog.CleanupBatch(ctx, map[string]struct{}{first.GenerationID: {}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if more {
		t.Fatal("cleanup reports more eligible generations after deleting the only unprotected inactive generation")
	}
	if _, err := catalog.LoadGeneration(ctx, second.GenerationID); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("unprotected old generation error = %v, want sql.ErrNoRows", err)
	}
	if _, err := catalog.LoadGeneration(ctx, first.GenerationID); err != nil {
		t.Fatalf("protected generation was removed: %v", err)
	}
	active, err := catalog.ActiveGenerationID(ctx, compiled.Pack.CourseID)
	if err != nil || active != third.GenerationID {
		t.Fatalf("head after cleanup = %q err=%v", active, err)
	}
}
