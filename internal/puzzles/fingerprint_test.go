package puzzles

import (
	"testing"

	"chess-trainer/internal/domain"
)

const goldenFingerprint = "3d112c34a05eb2425dad48347cfee3970c32e7d8bffd6868b2541cb6bd0bb196"

func TestCoreFingerprintMatchesFrozenGolden(t *testing.T) {
	core := PuzzleCore{
		DisplayedFEN: " 7k/5Q2/6K1/8/8/8/8/8 w - - 0 1 ",
		Solver:       domain.White,
		Solution:     []domain.MoveNode{{UCI: " F7F8 "}},
	}
	coreFingerprint, err := CoreFingerprint(core)
	if err != nil {
		t.Fatal(err)
	}
	if coreFingerprint != goldenFingerprint {
		t.Fatalf("CoreFingerprint() = %q, want golden %q", coreFingerprint, goldenFingerprint)
	}
}
