package chessrules

import "testing"

func TestRules(t *testing.T) {
	r := Rules{}
	fen := "rnbqkbnr/pppp1ppp/8/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 1 2"

	next, err := r.ApplyUCI(fen, "f1b5")
	if err != nil || next == fen {
		t.Fatalf("ApplyUCI() next=%q err=%v", next, err)
	}
	san, err := r.SAN(fen, "f1b5")
	if err != nil || san != "Bb5" {
		t.Fatalf("SAN()=%q err=%v", san, err)
	}
	if !r.IsCheckmateMove("7k/5Q2/6K1/8/8/8/8/8 w - - 0 1", "f7f8") {
		t.Fatal("expected Qf8 to checkmate")
	}
}
