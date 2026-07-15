package puzzles

import (
	"testing"

	"chess-trainer/internal/domain"
)

const goldenFingerprint = "3d112c34a05eb2425dad48347cfee3970c32e7d8bffd6868b2541cb6bd0bb196"

func TestCoreFingerprintMatchesLegacyFingerprint(t *testing.T) {
	legacy := domain.Puzzle{
		DisplayedFEN: " 7k/5Q2/6K1/8/8/8/8/8 w - - 0 1 ",
		Solver:       domain.White,
		Solution:     []domain.MoveNode{{UCI: " F7F8 "}},
	}
	core := PuzzleCore{
		DisplayedFEN: legacy.DisplayedFEN,
		Solver:       legacy.Solver,
		Solution:     legacy.Solution,
	}

	legacyFingerprint, err := Fingerprint(legacy)
	if err != nil {
		t.Fatal(err)
	}
	coreFingerprint, err := CoreFingerprint(core)
	if err != nil {
		t.Fatal(err)
	}
	if legacyFingerprint != goldenFingerprint {
		t.Fatalf("Fingerprint() = %q, want golden %q", legacyFingerprint, goldenFingerprint)
	}
	if coreFingerprint != goldenFingerprint {
		t.Fatalf("CoreFingerprint() = %q, want golden %q", coreFingerprint, goldenFingerprint)
	}
}

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
