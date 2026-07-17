package chessrules

import (
	"slices"
	"strings"
	"testing"
)

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

func TestApplyUCILineReturnsPositionAfterFirstMoveAndValidatesRemainder(t *testing.T) {
	rules := Rules{}
	fen := "rnbqkbnr/pppp1ppp/8/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 1 2"

	displayed, err := rules.ApplyUCILine(fen, []string{"f1b5", "a7a6", "b5a4"})
	if err != nil {
		t.Fatal(err)
	}
	want, err := rules.ApplyUCI(fen, "f1b5")
	if err != nil {
		t.Fatal(err)
	}
	if displayed != want {
		t.Fatalf("ApplyUCILine()=%q, want %q", displayed, want)
	}

	_, err = rules.ApplyUCILine(fen, []string{"f1b5", "a7a5", "b5b6"})
	if err == nil || !strings.Contains(err.Error(), `move 2 "b5b6"`) {
		t.Fatalf("ApplyUCILine() err=%v, want indexed illegal-move error", err)
	}
	if _, err := rules.ApplyUCILine(fen, nil); err == nil {
		t.Fatal("ApplyUCILine() unexpectedly accepted an empty line")
	}
}

func TestLegalMovesReturnsSortedStartingPosition(t *testing.T) {
	want := []string{
		"a2a3", "a2a4", "b1a3", "b1c3", "b2b3", "b2b4",
		"c2c3", "c2c4", "d2d3", "d2d4", "e2e3", "e2e4",
		"f2f3", "f2f4", "g1f3", "g1h3", "g2g3", "g2g4",
		"h2h3", "h2h4",
	}

	got, err := (Rules{}).LegalMoves(
		"rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
	)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(got, want) {
		t.Fatalf("LegalMoves()=%v, want %v", got, want)
	}
}

func TestLegalMovesIncludesCastlingEnPassantAndPromotions(t *testing.T) {
	tests := []struct {
		name string
		fen  string
		want []string
	}{
		{
			name: "castling",
			fen:  "r3k2r/8/8/8/8/8/8/R3K2R w KQkq - 0 1",
			want: []string{"e1c1", "e1g1"},
		},
		{
			name: "en passant",
			fen:  "8/8/8/3pP3/8/8/8/K6k w - d6 0 1",
			want: []string{"e5d6"},
		},
		{
			name: "promotions",
			fen:  "7k/P7/8/8/8/8/8/K7 w - - 0 1",
			want: []string{"a7a8b", "a7a8n", "a7a8q", "a7a8r"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			moves, err := (Rules{}).LegalMoves(test.fen)
			if err != nil {
				t.Fatal(err)
			}
			for _, want := range test.want {
				if !slices.Contains(moves, want) {
					t.Fatalf("LegalMoves()=%v, missing %q", moves, want)
				}
			}
		})
	}
}

func TestLegalMovesRejectsInvalidFEN(t *testing.T) {
	_, err := (Rules{}).LegalMoves("not a FEN")
	if err == nil || !strings.Contains(err.Error(), "list legal moves") {
		t.Fatalf("LegalMoves() error=%v, want contextual invalid-FEN error", err)
	}
}
