package puzzles

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/domain"
)

const solutionTreeTestFEN = "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"

const standardStartingFEN = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"

func TestNormalizeSolutionTreeValidatesEveryBranch(t *testing.T) {
	nodes := []domain.MoveNode{{
		UCI: " E2E4 ",
		Children: []domain.MoveNode{
			{UCI: "e8e7"},
			{UCI: "e8f7"},
		},
	}}

	got, plies, err := normalizeSolutionTree(chessrules.Rules{}, solutionTreeTestFEN, nodes)
	if err != nil {
		t.Fatal(err)
	}
	if got[0].UCI != "e2e4" || len(got[0].Children) != 2 || plies != 2 {
		t.Fatalf("normalized = %+v, plies = %d", got, plies)
	}
}

func TestNormalizeSolutionTreeRejectsIllegalSibling(t *testing.T) {
	_, _, err := normalizeSolutionTree(chessrules.Rules{}, solutionTreeTestFEN, []domain.MoveNode{{
		UCI: "e2e4",
		Children: []domain.MoveNode{
			{UCI: "e8e7"},
			{UCI: "a8a7"},
		},
	}})
	if err == nil || !strings.Contains(err.Error(), "a8a7") {
		t.Fatalf("error = %v, want illegal branch context", err)
	}
}

func TestNormalizeSolutionTreeRejectsInvalidStructureAndLimits(t *testing.T) {
	tests := []struct {
		name  string
		fen   string
		nodes []domain.MoveNode
		want  string
	}{
		{name: "empty roots", want: "solution is empty"},
		{
			name:  "empty UCI",
			nodes: []domain.MoveNode{{UCI: " \t "}},
			want:  "empty UCI",
		},
		{
			name:  "duplicate normalized sibling UCI",
			nodes: []domain.MoveNode{{UCI: "e2e4"}, {UCI: " E2E4 "}},
			want:  "duplicate",
		},
		{
			name:  "257-ply depth",
			fen:   standardStartingFEN,
			nodes: linearTestSolution(repeatingKnightMoves("e2e4", 257)),
			want:  "exceeds maximum of 256",
		},
		{
			name: "513 total nodes",
			fen:  standardStartingFEN,
			nodes: []domain.MoveNode{
				linearTestSolution(repeatingKnightMoves("e2e4", 171))[0],
				linearTestSolution(repeatingKnightMoves("d2d4", 171))[0],
				linearTestSolution(repeatingKnightMoves("c2c4", 171))[0],
			},
			want: "exceeds maximum of 512 nodes",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fen := test.fen
			if fen == "" {
				fen = solutionTreeTestFEN
			}
			_, _, err := normalizeSolutionTree(chessrules.Rules{}, fen, test.nodes)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want text %q", err, test.want)
			}
		})
	}
}

func TestLinearSolutionRejectsOverDepthBeforeConstruction(t *testing.T) {
	for _, moveCount := range []int{maxSolutionDepth + 1, 100_000} {
		t.Run(strconv.Itoa(moveCount)+" moves", func(t *testing.T) {
			moves := make([]string, moveCount)
			for index := range moves {
				moves[index] = "not-a-move"
			}

			solution, err := linearSolution(moves)
			if err == nil || !strings.Contains(
				err.Error(),
				fmt.Sprintf("solution depth %d exceeds maximum of %d", moveCount, maxSolutionDepth),
			) {
				t.Fatalf("linearSolution() error = %v, want precise depth limit", err)
			}
			if solution != nil {
				t.Fatalf("linearSolution() returned %d roots after limit error, want none", len(solution))
			}
		})
	}
}

func TestFinalizeCoreRejectsMalformedFENAndSolverMismatch(t *testing.T) {
	tests := []struct {
		name   string
		fen    string
		solver domain.Color
		want   string
	}{
		{name: "malformed FEN", fen: "not-a-fen", solver: domain.White, want: "FEN"},
		{name: "solver mismatch", fen: solutionTreeTestFEN, solver: domain.Black, want: "solver"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := finalizeCore(
				chessrules.Rules{},
				test.fen,
				test.solver,
				[]domain.MoveNode{{UCI: "e2e4"}},
			)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want text %q", err, test.want)
			}
		})
	}
}

func TestFinalizeCoreCanonicalLowercaseFingerprintStable(t *testing.T) {
	rules := chessrules.Rules{}
	canonical, err := finalizeCore(
		rules,
		solutionTreeTestFEN,
		domain.White,
		[]domain.MoveNode{{UCI: "e2e4"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	normalized, err := finalizeCore(
		rules,
		" \t4k3/8/8/8/8/8/4P3/4K3   w  -  -  0  1\n",
		domain.White,
		[]domain.MoveNode{{UCI: " E2E4 "}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if normalized.DisplayedFEN != solutionTreeTestFEN {
		t.Fatalf("DisplayedFEN = %q, want %q", normalized.DisplayedFEN, solutionTreeTestFEN)
	}
	if len(normalized.Solution) != 1 || normalized.Solution[0].UCI != "e2e4" {
		t.Fatalf("Solution = %+v, want canonical lowercase UCI", normalized.Solution)
	}
	if normalized.SolutionPlies != 1 {
		t.Fatalf("SolutionPlies = %d, want 1", normalized.SolutionPlies)
	}
	if normalized.Fingerprint != canonical.Fingerprint {
		t.Fatalf("fingerprints differ: normalized %q, canonical %q", normalized.Fingerprint, canonical.Fingerprint)
	}
}

func TestFinalizeCoreCanonicalizesCastlingRightsForStableFingerprint(t *testing.T) {
	const canonicalFEN = "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1"
	rules := chessrules.Rules{}
	canonical, err := finalizeCore(
		rules,
		canonicalFEN,
		domain.White,
		[]domain.MoveNode{{UCI: "e1d1"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	shuffled, err := finalizeCore(
		rules,
		"r3k2r/8/8/8/8/8/8/R3K2R w qkQK - 0 1",
		domain.White,
		[]domain.MoveNode{{UCI: "e1d1"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	if shuffled.DisplayedFEN != canonicalFEN {
		t.Fatalf("DisplayedFEN = %q, want canonical %q", shuffled.DisplayedFEN, canonicalFEN)
	}
	if shuffled.Fingerprint != canonical.Fingerprint {
		t.Fatalf("fingerprints differ: shuffled %q, canonical %q", shuffled.Fingerprint, canonical.Fingerprint)
	}
}

func TestFinalizeCoreRejectsMalformedCastlingRights(t *testing.T) {
	_, err := finalizeCore(
		chessrules.Rules{},
		"r3k2r/8/8/8/8/8/8/R3K2R w K- - 0 1",
		domain.White,
		[]domain.MoveNode{{UCI: "e1d1"}},
	)
	if err == nil || !strings.Contains(err.Error(), "castling rights") {
		t.Fatalf("error = %v, want malformed castling-rights rejection", err)
	}
}

func repeatingKnightMoves(root string, count int) []string {
	moves := make([]string, count)
	moves[0] = root
	cycle := [...]string{"g8f6", "g1f3", "f6g8", "f3g1"}
	for index := 1; index < count; index++ {
		moves[index] = cycle[(index-1)%len(cycle)]
	}
	return moves
}

func linearTestSolution(moves []string) []domain.MoveNode {
	if len(moves) == 0 {
		return nil
	}
	return []domain.MoveNode{{
		UCI:      moves[0],
		Children: linearTestSolution(moves[1:]),
	}}
}
