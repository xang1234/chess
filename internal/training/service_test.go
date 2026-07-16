package training

import (
	"context"
	"database/sql"
	"math/rand"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/domain"
	"chess-trainer/internal/puzzles"
	"chess-trainer/internal/storage"
)

func openTrainingStores(t *testing.T, root string) (*storage.PuzzleStore, *sql.DB, *puzzles.SQLiteCatalog) {
	t.Helper()
	puzzleStore, err := storage.OpenPuzzleStore(filepath.Join(root, "puzzles.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	userDB, err := storage.Open(filepath.Join(root, "user.sqlite"))
	if err != nil {
		puzzleStore.Close()
		t.Fatal(err)
	}
	if err := storage.Migrate(userDB, "user"); err != nil {
		puzzleStore.Close()
		userDB.Close()
		t.Fatal(err)
	}
	return puzzleStore, userDB, puzzles.NewSQLiteCatalog(puzzleStore.Reader, puzzleStore.Writer)
}

func importTrainingPuzzles(t *testing.T, catalog *puzzles.SQLiteCatalog, values ...puzzles.TrainingPuzzle) {
	t.Helper()
	importing, err := catalog.BeginImport(context.Background(), puzzles.Source{
		ID: "lichess", Kind: "lichess", Path: "/fixture.csv.zst", StartedAt: time.Unix(1, 0),
	})
	if err != nil {
		t.Fatal(err)
	}
	for index := range values {
		values[index].Occurrence.SourceID = "lichess"
		values[index].Occurrence.SourceKind = "lichess"
		values[index].Occurrence.Themes = []string{"development"}
		values[index].Occurrence.Ordinal = int64(index + 1)
		values[index].Core.Fingerprint, err = puzzles.CoreFingerprint(values[index].Core)
		if err != nil {
			t.Fatal(err)
		}
		if err := importing.Add(context.Background(), values[index]); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := importing.Seal(context.Background(), "abc123"); err != nil {
		t.Fatal(err)
	}
	if err := importing.Activate(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func trainingPuzzle(line ...string) puzzles.TrainingPuzzle {
	rating := 1500
	return puzzles.TrainingPuzzle{
		Core: puzzles.PuzzleCore{
			DisplayedFEN:  "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
			Solver:        domain.White,
			Solution:      testMoveLine(line),
			SolutionPlies: len(line),
		},
		Occurrence: puzzles.PuzzleOccurrence{Rating: &rating},
	}
}

func testMoveLine(moves []string) []domain.MoveNode {
	if len(moves) == 0 {
		return nil
	}
	return []domain.MoveNode{{UCI: moves[0], Children: testMoveLine(moves[1:])}}
}

func TestServiceResumesAfterEveryMove(t *testing.T) {
	root := t.TempDir()
	puzzleStore, userDB, catalog := openTrainingStores(t, root)
	first := trainingPuzzle("e2e4", "e7e5", "g1f3")
	second := trainingPuzzle("d2d4", "d7d5", "c1f4")
	importTrainingPuzzles(t, catalog, first, second)
	store := NewUserStore(userDB)
	if err := store.UpdateProfile(context.Background(), Profile{LearnerRating: 1500, SessionSize: 5}); err != nil {
		t.Fatal(err)
	}
	service := NewService(catalog, store, chessrules.Rules{}, rand.New(rand.NewSource(4)))

	started, err := service.StartGuided(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if started.Current == nil {
		t.Fatal("session has no current puzzle")
	}
	firstFingerprint, _ := puzzles.CoreFingerprint(first.Core)
	move := "d2d4"
	if started.Current.Fingerprint == firstFingerprint {
		move = "e2e4"
	}
	result, err := service.PlayMove(context.Background(), started.SessionID, move)
	if err != nil {
		t.Fatal(err)
	}
	want := result.Session.Current
	if want == nil || !slices.Equal(want.CurrentPath, []int{0, 0}) {
		t.Fatalf("current=%+v", want)
	}
	if err := userDB.Close(); err != nil {
		t.Fatal(err)
	}
	if err := puzzleStore.Close(); err != nil {
		t.Fatal(err)
	}

	puzzleStore, userDB, catalog = openTrainingStores(t, root)
	defer puzzleStore.Close()
	defer userDB.Close()
	resumedService := NewService(
		catalog,
		NewUserStore(userDB),
		chessrules.Rules{},
		rand.New(rand.NewSource(4)),
	)
	resumed, err := resumedService.Resume(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if resumed == nil || resumed.Current == nil {
		t.Fatalf("resumed=%+v", resumed)
	}
	got := resumed.Current
	if resumed.CurrentIndex != 0 || got.CurrentFEN != want.CurrentFEN ||
		!slices.Equal(got.CurrentPath, want.CurrentPath) || got.IncorrectMoves != 0 || got.HintLevel != 0 {
		t.Fatalf("resumed=%+v, want current=%+v", resumed, want)
	}
}

func TestServicePersistsWrongMoveHintsRevealAndSummary(t *testing.T) {
	root := t.TempDir()
	puzzleStore, userDB, catalog := openTrainingStores(t, root)
	puzzle := trainingPuzzle("e2e4", "e7e5", "g1f3")
	importTrainingPuzzles(t, catalog, puzzle)
	store := NewUserStore(userDB)
	if err := store.UpdateProfile(context.Background(), Profile{LearnerRating: 1500, SessionSize: 5}); err != nil {
		t.Fatal(err)
	}
	service := NewService(catalog, store, chessrules.Rules{}, rand.New(rand.NewSource(1)))
	started, err := service.StartGuided(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	initialFEN := started.Current.CurrentFEN
	wrong, err := service.PlayMove(context.Background(), started.SessionID, "e2e3")
	if err != nil {
		t.Fatal(err)
	}
	if wrong.Correct || wrong.Session.Current.CurrentFEN != initialFEN {
		t.Fatalf("wrong=%+v", wrong)
	}
	if err := userDB.Close(); err != nil {
		t.Fatal(err)
	}
	userDB, err = storage.Open(filepath.Join(root, "user.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	store = NewUserStore(userDB)
	service = NewService(catalog, store, chessrules.Rules{}, rand.New(rand.NewSource(1)))
	resumed, err := service.Resume(context.Background())
	if err != nil || resumed.Current.IncorrectMoves != 1 {
		t.Fatalf("resumed=%+v err=%v", resumed, err)
	}

	for level := 1; level <= 3; level++ {
		hint, err := service.UseHint(context.Background(), started.SessionID)
		if err != nil {
			t.Fatal(err)
		}
		if hint.Level != level {
			t.Fatalf("hint=%+v", hint)
		}
		if level == 2 && hint.SourceSquare != "e2" {
			t.Fatalf("hint=%+v", hint)
		}
		if level == 3 && (hint.TargetSquare != "e4" || !hint.CanReveal) {
			t.Fatalf("hint=%+v", hint)
		}
	}
	revealed, err := service.Reveal(context.Background(), started.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if !revealed.PuzzleCompleted || revealed.Session.Status != "completed" {
		t.Fatalf("revealed=%+v", revealed)
	}
	if err := userDB.Close(); err != nil {
		t.Fatal(err)
	}
	userDB, err = storage.Open(filepath.Join(root, "user.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer userDB.Close()
	service = NewService(catalog, NewUserStore(userDB), chessrules.Rules{}, rand.New(rand.NewSource(1)))
	summary, err := service.Summary(context.Background(), started.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Total != 1 || summary.Retried != 1 || summary.UsedHint != 1 || summary.Revealed != 1 {
		t.Fatalf("summary=%+v", summary)
	}
	profile, err := NewUserStore(userDB).Profile(context.Background())
	if err != nil || profile.LearnerRating != 1500 {
		t.Fatalf("profile=%+v err=%v", profile, err)
	}
	var reviews int
	if err := userDB.QueryRow(`SELECT COUNT(*) FROM review_state`).Scan(&reviews); err != nil || reviews != 1 {
		t.Fatalf("reviews=%d err=%v", reviews, err)
	}
	puzzleStore.Close()
}

func TestServiceAcceptsAlternativeMateAndPauseResume(t *testing.T) {
	root := t.TempDir()
	puzzleStore, userDB, catalog := openTrainingStores(t, root)
	defer puzzleStore.Close()
	defer userDB.Close()
	rating := 1500
	puzzle := puzzles.TrainingPuzzle{
		Core: puzzles.PuzzleCore{
			DisplayedFEN:  "7k/5Q2/6K1/8/8/8/8/8 w - - 0 1",
			Solver:        domain.White,
			Solution:      []domain.MoveNode{{UCI: "f7f8"}},
			SolutionPlies: 1,
		},
		Occurrence: puzzles.PuzzleOccurrence{Rating: &rating},
	}
	importTrainingPuzzles(t, catalog, puzzle)
	store := NewUserStore(userDB)
	if err := store.UpdateProfile(context.Background(), Profile{LearnerRating: 1500, SessionSize: 5}); err != nil {
		t.Fatal(err)
	}
	service := NewService(catalog, store, chessrules.Rules{}, rand.New(rand.NewSource(1)))
	started, err := service.StartGuided(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Pause(context.Background(), started.SessionID); err != nil {
		t.Fatal(err)
	}
	resumed, err := service.Resume(context.Background())
	if err != nil || resumed.Status != "active" {
		t.Fatalf("resumed=%+v err=%v", resumed, err)
	}
	result, err := service.PlayMove(context.Background(), started.SessionID, "f7g7")
	if err != nil {
		t.Fatal(err)
	}
	if !result.Correct || !result.PuzzleCompleted || result.Session.Status != "completed" {
		t.Fatalf("result=%+v", result)
	}
	summary, err := service.Summary(context.Background(), started.SessionID)
	if err != nil || summary.FirstTry != 1 {
		t.Fatalf("summary=%+v err=%v", summary, err)
	}
	profile, err := store.Profile(context.Background())
	if err != nil || profile.LearnerRating != 1500 {
		t.Fatalf("profile=%+v err=%v", profile, err)
	}
	var reviews int
	if err := userDB.QueryRow(`SELECT COUNT(*) FROM review_state`).Scan(&reviews); err != nil || reviews != 0 {
		t.Fatalf("reviews=%d err=%v", reviews, err)
	}
}

func TestResumeSkipsPuzzleRemovedBySourceReimport(t *testing.T) {
	root := t.TempDir()
	puzzleStore, userDB, catalog := openTrainingStores(t, root)
	defer puzzleStore.Close()
	defer userDB.Close()
	first := trainingPuzzle("e2e4")
	second := trainingPuzzle("d2d4")
	importTrainingPuzzles(t, catalog, first, second)
	store := NewUserStore(userDB)
	if err := store.UpdateProfile(context.Background(), Profile{LearnerRating: 1500, SessionSize: 5}); err != nil {
		t.Fatal(err)
	}
	service := NewService(catalog, store, chessrules.Rules{}, rand.New(rand.NewSource(2)))
	started, err := service.StartGuided(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	firstFingerprint, _ := puzzles.CoreFingerprint(first.Core)
	move := "d2d4"
	if started.Current.Fingerprint == firstFingerprint {
		move = "e2e4"
	}
	completed, err := service.PlayMove(context.Background(), started.SessionID, move)
	if err != nil || !completed.PuzzleCompleted || completed.Session.Status != "active" {
		t.Fatalf("completed=%+v err=%v", completed, err)
	}

	replacement := trainingPuzzle("g1f3")
	importTrainingPuzzles(t, catalog, replacement)
	resumed, err := service.Resume(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Status != "completed" || resumed.Current != nil {
		t.Fatalf("resumed=%+v", resumed)
	}
	summary, err := service.Summary(context.Background(), started.SessionID)
	if err != nil || summary.Unavailable != 1 {
		t.Fatalf("summary=%+v err=%v", summary, err)
	}
	var attempts int
	if err := userDB.QueryRow(`SELECT COUNT(*) FROM attempts`).Scan(&attempts); err != nil {
		t.Fatal(err)
	}
	if attempts != 1 {
		t.Fatalf("attempts=%d, want only the completed historical attempt", attempts)
	}
}

func TestFreePracticeFiltersPuzzlesAndNeverChangesLearnerRating(t *testing.T) {
	root := t.TempDir()
	puzzleStore, userDB, catalog := openTrainingStores(t, root)
	defer puzzleStore.Close()
	defer userDB.Close()
	short := trainingPuzzle("e2e4")
	long := trainingPuzzle("d2d4", "d7d5", "c1f4")
	importTrainingPuzzles(t, catalog, short, long)
	store := NewUserStore(userDB)
	if err := store.UpdateProfile(context.Background(), Profile{LearnerRating: 1500, SessionSize: 5}); err != nil {
		t.Fatal(err)
	}
	service := NewService(catalog, store, chessrules.Rules{}, rand.New(rand.NewSource(1)))
	minimum, maximum, maximumPlies := 1400, 1600, 1

	started, err := service.StartFreePractice(context.Background(), PracticeRequest{
		SourceID:             "lichess",
		MinimumRating:        &minimum,
		MaximumRating:        &maximum,
		Themes:               []string{"development"},
		MaximumSolutionPlies: &maximumPlies,
	})
	if err != nil {
		t.Fatal(err)
	}
	if started.Mode != "practice" || started.Total != 1 || started.Current == nil {
		t.Fatalf("started=%+v", started)
	}
	wrong, err := service.PlayMove(context.Background(), started.SessionID, "e2e3")
	if err != nil || wrong.Correct {
		t.Fatalf("wrong=%+v err=%v", wrong, err)
	}
	completed, err := service.PlayMove(context.Background(), started.SessionID, "e2e4")
	if err != nil || !completed.PuzzleCompleted || completed.Session.Status != "completed" {
		t.Fatalf("completed=%+v err=%v", completed, err)
	}
	profile, err := store.Profile(context.Background())
	if err != nil || profile.LearnerRating != 1500 {
		t.Fatalf("profile=%+v err=%v", profile, err)
	}
	var reviews int
	if err := userDB.QueryRow(`SELECT COUNT(*) FROM review_state`).Scan(&reviews); err != nil || reviews != 1 {
		t.Fatalf("reviews=%d err=%v", reviews, err)
	}
}
