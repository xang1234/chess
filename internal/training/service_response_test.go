package training

import (
	"context"
	"math/rand"
	"slices"
	"testing"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/domain"
	"chess-trainer/internal/puzzles"
)

func responseTestService(
	t *testing.T,
	values ...puzzles.TrainingPuzzle,
) (*Service, *UserStore) {
	t.Helper()
	puzzleStore, userDB, catalog := openTrainingStores(t, t.TempDir())
	t.Cleanup(func() {
		_ = userDB.Close()
		_ = puzzleStore.Close()
	})
	importTrainingPuzzles(t, catalog, values...)
	store := NewUserStore(userDB)
	if err := store.UpdateProfile(
		context.Background(),
		Profile{LearnerRating: 1500, SessionSize: 5},
	); err != nil {
		t.Fatal(err)
	}
	return NewService(
		catalog,
		store,
		chessrules.Rules{},
		rand.New(rand.NewSource(1)),
	), store
}

func expectedAppliedMoves(
	t *testing.T,
	fen string,
	moves ...string,
) ([]domain.AppliedMove, string) {
	t.Helper()
	rules := chessrules.Rules{}
	result := make([]domain.AppliedMove, 0, len(moves))
	for _, uci := range moves {
		next, err := rules.ApplyUCI(fen, uci)
		if err != nil {
			t.Fatalf("apply expected move %q: %v", uci, err)
		}
		result = append(result, domain.AppliedMove{UCI: uci, ResultingFEN: next})
		fen = next
	}
	return result, fen
}

func puzzleFingerprint(t *testing.T, puzzle puzzles.TrainingPuzzle) string {
	t.Helper()
	fingerprint, err := puzzles.CoreFingerprint(puzzle.Core)
	if err != nil {
		t.Fatal(err)
	}
	return fingerprint
}

func TestServiceViewIncludesEveryLegalMove(t *testing.T) {
	service, _ := responseTestService(
		t,
		trainingPuzzle("e2e4", "e7e5", "g1f3"),
	)
	started, err := service.StartGuided(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if started.Current == nil {
		t.Fatal("session has no current puzzle")
	}
	if len(started.Current.LegalMoves) != 20 ||
		!slices.Contains(started.Current.LegalMoves, "e2e3") ||
		!slices.Contains(started.Current.LegalMoves, "e2e4") ||
		!slices.IsSorted(started.Current.LegalMoves) {
		t.Fatalf("legal moves=%v, want all 20 sorted starting moves", started.Current.LegalMoves)
	}
}

func TestServiceCorrectNonFinalMoveRefreshesLegalMoves(t *testing.T) {
	service, _ := responseTestService(
		t,
		trainingPuzzle("e2e4", "e7e5", "g1f3"),
	)
	started, err := service.StartGuided(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.PlayMove(context.Background(), started.SessionID, "e2e4")
	if err != nil {
		t.Fatal(err)
	}
	if result.PuzzleCompleted || result.Session.Current == nil {
		t.Fatalf("result=%+v, want active non-final puzzle", result)
	}
	_, afterReply := expectedAppliedMoves(
		t,
		started.Current.CurrentFEN,
		"e2e4",
		"e7e5",
	)
	want, err := (chessrules.Rules{}).LegalMoves(afterReply)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(result.Session.Current.LegalMoves, want) {
		t.Fatalf("legal moves=%v, want %v", result.Session.Current.LegalMoves, want)
	}
}

func TestServicePlayMoveReturnsSubmittedMoveAndAutomaticReply(t *testing.T) {
	service, _ := responseTestService(
		t,
		trainingPuzzle("e2e4", "e7e5", "g1f3"),
	)
	started, err := service.StartGuided(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want, _ := expectedAppliedMoves(
		t,
		started.Current.CurrentFEN,
		"e2e4",
		"e7e5",
	)
	result, err := service.PlayMove(context.Background(), started.SessionID, "e2e4")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(result.AppliedMoves, want) || result.FinalFEN != "" {
		t.Fatalf("result=%+v, want frames=%+v and no final FEN", result, want)
	}
}

func TestServiceWrongMoveReturnsNoAppliedMovesOrFinalFEN(t *testing.T) {
	service, _ := responseTestService(
		t,
		trainingPuzzle("e2e4", "e7e5", "g1f3"),
	)
	started, err := service.StartGuided(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	initialFEN := started.Current.CurrentFEN
	result, err := service.PlayMove(context.Background(), started.SessionID, "e2e3")
	if err != nil {
		t.Fatal(err)
	}
	if result.Correct || len(result.AppliedMoves) != 0 || result.FinalFEN != "" ||
		result.Session.Current == nil || result.Session.Current.CurrentFEN != initialFEN {
		t.Fatalf("wrong result=%+v", result)
	}
}

func TestServiceCompletionKeepsFinalFENWhilePreparingNextPuzzle(t *testing.T) {
	first := trainingPuzzle("e2e4")
	second := trainingPuzzle("d2d4")
	service, _ := responseTestService(t, first, second)
	started, err := service.StartGuided(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if started.Current == nil {
		t.Fatal("session has no current puzzle")
	}
	move := "d2d4"
	if started.Current.Fingerprint == puzzleFingerprint(t, first) {
		move = "e2e4"
	}
	wantApplied, wantFinal := expectedAppliedMoves(
		t,
		started.Current.CurrentFEN,
		move,
	)

	result, err := service.PlayMove(context.Background(), started.SessionID, move)
	if err != nil {
		t.Fatal(err)
	}
	if !result.PuzzleCompleted || result.Session.Status != "active" ||
		result.Session.CurrentIndex != 1 || result.Session.Current == nil ||
		result.Session.Current.Fingerprint == started.Current.Fingerprint ||
		!slices.Equal(result.AppliedMoves, wantApplied) || result.FinalFEN != wantFinal {
		t.Fatalf("completion result=%+v, want final=%q frames=%+v", result, wantFinal, wantApplied)
	}
}

func TestServiceLastPuzzleCompletionReturnsFinalFENWithSummary(t *testing.T) {
	service, _ := responseTestService(t, trainingPuzzle("e2e4"))
	started, err := service.StartGuided(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	wantApplied, wantFinal := expectedAppliedMoves(
		t,
		started.Current.CurrentFEN,
		"e2e4",
	)

	result, err := service.PlayMove(context.Background(), started.SessionID, "e2e4")
	if err != nil {
		t.Fatal(err)
	}
	if !result.PuzzleCompleted || result.Session.Status != "completed" ||
		result.Session.Current != nil || result.Session.Summary == nil ||
		!slices.Equal(result.AppliedMoves, wantApplied) || result.FinalFEN != wantFinal {
		t.Fatalf("last result=%+v, want final=%q frames=%+v", result, wantFinal, wantApplied)
	}
}

func TestServiceRevealReturnsCompleteRemainingAppliedLine(t *testing.T) {
	service, _ := responseTestService(
		t,
		trainingPuzzle("e2e4", "e7e5", "g1f3"),
	)
	started, err := service.StartGuided(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	for range 3 {
		if _, err := service.UseHint(context.Background(), started.SessionID); err != nil {
			t.Fatal(err)
		}
	}
	wantApplied, wantFinal := expectedAppliedMoves(
		t,
		started.Current.CurrentFEN,
		"e2e4",
		"e7e5",
		"g1f3",
	)

	result, err := service.Reveal(context.Background(), started.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if !result.PuzzleCompleted || !slices.Equal(result.AppliedMoves, wantApplied) ||
		result.FinalFEN != wantFinal {
		t.Fatalf("reveal=%+v, want final=%q frames=%+v", result, wantFinal, wantApplied)
	}
}
