package puzzles

import (
	"testing"

	"chess-trainer/internal/domain"
)

func TestFingerprintIgnoresSourceMetadata(t *testing.T) {
	base := domain.Puzzle{
		DisplayedFEN: "7k/5Q2/6K1/8/8/8/8/8 w - - 0 1",
		Solver:       domain.White,
		Solution:     []domain.MoveNode{{UCI: "f7f8"}},
	}
	a := base
	a.Sources = []domain.SourceRef{{SourceID: "a", ExternalID: "1"}}
	b := base
	b.Sources = []domain.SourceRef{{SourceID: "b", ExternalID: "9"}}

	fa, err := Fingerprint(a)
	if err != nil {
		t.Fatal(err)
	}
	fb, err := Fingerprint(b)
	if err != nil {
		t.Fatal(err)
	}
	if fa != fb {
		t.Fatalf("fingerprints differ: %s != %s", fa, fb)
	}
}
