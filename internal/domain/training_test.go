package domain

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPuzzleViewJSONIncludesLegalMoves(t *testing.T) {
	payload, err := json.Marshal(PuzzleView{LegalMoves: []string{"e2e3", "e2e4"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(payload), `"legalMoves":["e2e3","e2e4"]`) {
		t.Fatalf("PuzzleView JSON=%s, want legalMoves", payload)
	}
}

func TestMoveResultJSONIncludesAppliedMovesAndFinalFEN(t *testing.T) {
	payload, err := json.Marshal(MoveResult{
		AppliedMoves: []AppliedMove{{UCI: "e2e4", ResultingFEN: "after"}},
		FinalFEN:     "after",
	})
	if err != nil {
		t.Fatal(err)
	}
	encoded := string(payload)
	if !strings.Contains(
		encoded,
		`"appliedMoves":[{"uci":"e2e4","resultingFen":"after"}]`,
	) || !strings.Contains(encoded, `"finalFen":"after"`) {
		t.Fatalf("MoveResult JSON=%s, want appliedMoves and finalFen", payload)
	}
}

func TestMoveResultJSONOmitsEmptyAnimationFields(t *testing.T) {
	payload, err := json.Marshal(MoveResult{})
	if err != nil {
		t.Fatal(err)
	}
	encoded := string(payload)
	if strings.Contains(encoded, `"appliedMoves"`) || strings.Contains(encoded, `"finalFen"`) {
		t.Fatalf("MoveResult JSON=%s, want empty animation fields omitted", payload)
	}
}
