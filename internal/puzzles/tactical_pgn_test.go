package puzzles

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/domain"

	"github.com/corentings/chess/v2"
)

const tacticalPGNDirectSolver = `[Event "Direct solver turn"]
[SourceId "club-tactics"]
[PuzzleId "white-1"]
[SetUp "1"]
[FEN "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"]
[White "solver"]
[Black "?"]

1. e4 Kf7 *
`

const tacticalPGNOpponentPrelude = `[Event "Opponent prelude"]
[SetUp "1"][FEN "4k3/8/8/4p3/8/8/4P3/4K3 b - - 0 1"]
[White "solver"][Black "?"]

1... e4 2. Kf2 *
`

func TestTacticalPGNAdapterNormalizesDirectSolverAndOpponentPrelude(t *testing.T) {
	tests := []struct {
		name             string
		pgn              string
		wantSourceID     func(string) string
		wantOrigin       SourceIDOrigin
		wantExternalID   string
		wantSourceFEN    string
		wantDisplayedFEN string
		wantPrelude      string
		wantSolver       domain.Color
		wantMoves        []string
	}{
		{
			name:             "direct solver turn with embedded identity",
			pgn:              tacticalPGNDirectSolver,
			wantSourceID:     func(string) string { return "club-tactics" },
			wantOrigin:       SourceIDEmbedded,
			wantExternalID:   "white-1",
			wantSourceFEN:    "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1",
			wantDisplayedFEN: "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1",
			wantSolver:       domain.White,
			wantMoves:        []string{"e2e4", "e8f7"},
		},
		{
			name:             "opponent prelude with path identity and ordinal ID",
			pgn:              tacticalPGNOpponentPrelude,
			wantSourceID:     func(path string) string { return path },
			wantOrigin:       SourceIDPath,
			wantExternalID:   "1",
			wantSourceFEN:    "4k3/8/8/4p3/8/8/4P3/4K3 b - - 0 1",
			wantDisplayedFEN: "4k3/8/8/8/4p3/8/4P3/4K3 w - - 0 2",
			wantPrelude:      "e5e4",
			wantSolver:       domain.White,
			wantMoves:        []string{"e1f2"},
		},
		{
			name: "black solver tag is case insensitive",
			pgn: `[Event "Black solver"]
[SetUp "1"][FEN "4k3/4p3/8/8/8/8/8/4K3 b - - 0 1"]
[White "?"][Black "  SoLvEr  "]

1... e5 2. Kf2 *
`,
			wantSourceID:     func(path string) string { return path },
			wantOrigin:       SourceIDPath,
			wantExternalID:   "1",
			wantSourceFEN:    "4k3/4p3/8/8/8/8/8/4K3 b - - 0 1",
			wantDisplayedFEN: "4k3/4p3/8/8/8/8/8/4K3 b - - 0 1",
			wantSolver:       domain.Black,
			wantMoves:        []string{"e7e5", "e1f2"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			adapter, path, inspection := inspectTacticalPGN(t, test.pgn)
			if inspection.SourceID != test.wantSourceID(path) || inspection.SourceIDOrigin != test.wantOrigin {
				t.Fatalf("inspection identity = %q/%q, want %q/%q", inspection.SourceID, inspection.SourceIDOrigin, test.wantSourceID(path), test.wantOrigin)
			}
			if inspection.Format != FormatTacticalPGN {
				t.Fatalf("inspection format = %q, want %q", inspection.Format, FormatTacticalPGN)
			}

			file, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			defer file.Close()
			decoder, err := adapter.NewDecoder(file, inspection)
			if err != nil {
				t.Fatal(err)
			}
			defer decoder.Close()

			record, err := decoder.Next(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if record.Puzzle == nil || record.Rejection != nil {
				t.Fatalf("decoded record = %+v, want puzzle", record)
			}
			puzzle := *record.Puzzle
			if puzzle.Occurrence.SourceID != inspection.SourceID ||
				puzzle.Occurrence.SourceKind != string(FormatTacticalPGN) ||
				puzzle.Occurrence.ExternalID != test.wantExternalID ||
				puzzle.Occurrence.SourceFEN != test.wantSourceFEN ||
				puzzle.Occurrence.PreludeUCI != test.wantPrelude ||
				puzzle.Occurrence.Rating != nil ||
				puzzle.Occurrence.Ordinal != 1 {
				t.Fatalf("occurrence = %+v", puzzle.Occurrence)
			}
			if puzzle.Core.DisplayedFEN != test.wantDisplayedFEN ||
				puzzle.Core.Solver != test.wantSolver ||
				puzzle.Core.SolutionPlies != len(test.wantMoves) {
				t.Fatalf("core = %+v", puzzle.Core)
			}
			if got := tacticalPGNLinearMoves(t, puzzle.Core.Solution); !slices.Equal(got, test.wantMoves) {
				t.Fatalf("solution moves = %q, want %q", got, test.wantMoves)
			}
			if _, err := decoder.Next(context.Background()); !errors.Is(err, io.EOF) {
				t.Fatalf("terminal error = %v, want EOF", err)
			}
		})
	}
}

func TestTacticalPGNDecoderRejectsInvalidRecords(t *testing.T) {
	tests := []struct {
		name string
		pgn  string
		want string
	}{
		{
			name: "missing FEN",
			pgn:  `[SetUp "1"][White "solver"][Black "?"] 1. e4 *`,
			want: "FEN is required",
		},
		{
			name: "invalid SetUp",
			pgn:  `[SetUp "0"][FEN "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"][White "solver"][Black "?"] 1. e4 *`,
			want: `SetUp must be "1"`,
		},
		{
			name: "both players solver",
			pgn:  `[FEN "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"][White "solver"][Black "solver"] 1. e4 *`,
			want: "exactly one",
		},
		{
			name: "neither player solver",
			pgn:  `[FEN "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"][White "?"][Black "?"] 1. e4 *`,
			want: "exactly one",
		},
		{
			name: "illegal movetext",
			pgn:  `[FEN "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"][White "solver"][Black "?"] 1. e5 *`,
			want: "parse PGN game",
		},
		{
			name: "prelude consumes only move",
			pgn:  `[FEN "4k3/8/8/4p3/8/8/4P3/4K3 b - - 0 1"][White "solver"][Black "?"] 1... e4 *`,
			want: "at least one solution move",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decoder := newTacticalPGNTestDecoder(t, test.pgn, "club-tactics")
			record, err := decoder.Next(context.Background())
			if err != nil {
				t.Fatalf("Next() fatal error = %v, want rejection", err)
			}
			if record.Rejection == nil || record.Puzzle != nil {
				t.Fatalf("decoded record = %+v, want rejection", record)
			}
			if record.Rejection.Ordinal != 1 || !strings.Contains(record.Rejection.Reason, test.want) {
				t.Fatalf("rejection = %+v, want ordinal 1 and %q", *record.Rejection, test.want)
			}
		})
	}
}

func TestTacticalPGNDecoderContinuesAfterRecoverablyFramedParseError(t *testing.T) {
	contents := `[Event "Broken move"]
[SourceId "club-tactics"]
[FEN "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"]
[White "solver"][Black "?"]

1. e5 *

[Event "Recovered"]
[FEN "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"]
[White "solver"][Black "?"]

1. e4 *
`
	decoder := newTacticalPGNTestDecoder(t, contents, "club-tactics")

	first, err := decoder.Next(context.Background())
	if err != nil || first.Rejection == nil || first.Rejection.Ordinal != 1 {
		t.Fatalf("first record/error = %+v/%v, want ordinal-1 rejection", first, err)
	}
	second, err := decoder.Next(context.Background())
	if err != nil || second.Puzzle == nil || second.Puzzle.Occurrence.Ordinal != 2 || second.Puzzle.Occurrence.ExternalID != "2" {
		t.Fatalf("second record/error = %+v/%v, want recovered ordinal-2 puzzle", second, err)
	}
}

func TestTacticalPGNDecoderTreatsAnyLaterExplicitSourceConflictAsFatal(t *testing.T) {
	tests := []struct {
		name     string
		sourceID string
		movetext string
	}{
		{
			name:     "conflict wins over illegal movetext rejection",
			sourceID: "other-club",
			movetext: "1. e5 *",
		},
		{
			name:     "explicit empty identity is not an omission",
			sourceID: "   ",
			movetext: "1. e4 *",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			contents := tacticalPGNDirectSolver + `
[Event "Conflicting identity"]
[SourceId "` + test.sourceID + `"]
[FEN "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"]
[White "solver"][Black "?"]

` + test.movetext + "\n"
			decoder := newTacticalPGNTestDecoder(t, contents, "club-tactics")
			if first, err := decoder.Next(context.Background()); err != nil || first.Puzzle == nil {
				t.Fatalf("first record/error = %+v/%v, want puzzle", first, err)
			}

			second, err := decoder.Next(context.Background())
			if err == nil || !strings.Contains(err.Error(), "SourceId") || second.Rejection != nil || second.Puzzle != nil {
				t.Fatalf("second record/error = %+v/%v, want fatal SourceId conflict", second, err)
			}
		})
	}
}

func TestTacticalPGNDecoderChecksConflictingSourceBeforeGenuineLexerError(t *testing.T) {
	conflicting := `[Event "Conflicting identity"]
[SourceId "other-club"]
[FEN "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"]
[White "solver"][Black "?"]

1. e *
`
	tokens, err := chess.TokenizeGame(&chess.GameScanned{Raw: conflicting})
	if err != nil {
		t.Fatal(err)
	}
	var lexerError error
	for _, token := range tokens {
		if token.Error != nil {
			lexerError = token.Error
			break
		}
	}
	if lexerError == nil || !strings.Contains(lexerError.Error(), "invalid square") {
		t.Fatalf("fixture lexer error = %v, want genuine invalid-square token error", lexerError)
	}

	decoder := newTacticalPGNTestDecoder(
		t,
		tacticalPGNDirectSolver+"\n"+conflicting,
		"club-tactics",
	)
	if first, err := decoder.Next(context.Background()); err != nil || first.Puzzle == nil {
		t.Fatalf("first record/error = %+v/%v, want puzzle", first, err)
	}
	second, err := decoder.Next(context.Background())
	if err == nil || !strings.Contains(err.Error(), "SourceId") || second.Rejection != nil || second.Puzzle != nil {
		t.Fatalf("second record/error = %+v/%v, want fatal SourceId conflict before lexer rejection", second, err)
	}
}

func TestTacticalPGNDecoderUsesMainlineOnlyAndAcceptsAdjacentTags(t *testing.T) {
	contents := `[Event "Annotated mainline"][SetUp "1"][FEN "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"][White "solver"][Black "?"][Untrusted "discard me"]

1. e4 {mainline comment [%clk 0:01:00]} $1 (1. e3 Kf7) Kf7 *
`
	decoder := newTacticalPGNTestDecoder(t, contents, "club-tactics")

	record, err := decoder.Next(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if record.Puzzle == nil {
		t.Fatalf("record = %+v, want puzzle", record)
	}
	puzzle := *record.Puzzle
	if got := tacticalPGNLinearMoves(t, puzzle.Core.Solution); !slices.Equal(got, []string{"e2e4", "e8f7"}) {
		t.Fatalf("solution moves = %q, want mainline only", got)
	}
	if puzzle.Occurrence.Metadata["Event"] != "Annotated mainline" {
		t.Fatalf("metadata = %#v, want Event provenance", puzzle.Occurrence.Metadata)
	}
	if _, retained := puzzle.Occurrence.Metadata["Untrusted"]; retained {
		t.Fatalf("metadata retained unknown tag: %#v", puzzle.Occurrence.Metadata)
	}
}

func TestTacticalPGNDecoderRejectsOverDepthSolution(t *testing.T) {
	var movetext strings.Builder
	cycle := [...]string{"Nf3", "Nf6", "Ng1", "Ng8"}
	for ply := 0; ply < maxSolutionDepth+1; ply++ {
		if ply%2 == 0 {
			movetext.WriteString(strconv.Itoa(ply/2 + 1))
			movetext.WriteString(". ")
		}
		movetext.WriteString(cycle[ply%len(cycle)])
		movetext.WriteByte(' ')
	}
	movetext.WriteByte('*')
	contents := `[SourceId "club-tactics"]
[SetUp "1"]
[FEN "` + standardStartingFEN + `"]
[White "solver"]
[Black "?"]

` + movetext.String()
	decoder := newTacticalPGNTestDecoder(t, contents, "club-tactics")

	record, err := decoder.Next(context.Background())
	if err != nil {
		t.Fatalf("Next() fatal error = %v, want record rejection", err)
	}
	if record.Rejection == nil || record.Puzzle != nil || !strings.Contains(
		record.Rejection.Reason,
		"solution depth 257 exceeds maximum of 256",
	) {
		t.Fatalf("record = %+v, want bounded depth rejection", record)
	}
}

func TestTacticalPGNDecoderEnforcesExactTagLimit(t *testing.T) {
	for _, test := range []struct {
		name       string
		extraTags  int
		wantReject bool
	}{
		{name: "128 tags accepted", extraTags: 124},
		{name: "129 tags rejected", extraTags: 125, wantReject: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var tags strings.Builder
			for index := 0; index < test.extraTags; index++ {
				tags.WriteString(`[Extra` + strconv.Itoa(index) + ` "value"]`)
			}
			contents := `[FEN "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"][SetUp "1"][White "solver"][Black "?"]` + tags.String() + ` 1. e4 *`
			decoder := newTacticalPGNTestDecoder(t, contents, "club-tactics")

			record, err := decoder.Next(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if test.wantReject {
				if record.Rejection == nil || !strings.Contains(record.Rejection.Reason, "128") {
					t.Fatalf("record = %+v, want tag-limit rejection", record)
				}
			} else if record.Puzzle == nil {
				t.Fatalf("record = %+v, want accepted puzzle", record)
			}
		})
	}
}

func TestTacticalPGNDecoderBoundsRetainedProvenanceMetadata(t *testing.T) {
	contents := `[Event "` + strings.Repeat("<", 12*1024) + `"]
[FEN "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"]
[White "solver"][Black "?"]

1. e4 *`
	decoder := newTacticalPGNTestDecoder(t, contents, "club-tactics")

	record, err := decoder.Next(context.Background())
	if err != nil || record.Puzzle == nil {
		t.Fatalf("record/error = %+v/%v, want accepted puzzle", record, err)
	}
	encoded, err := json.Marshal(record.Puzzle.Occurrence.Metadata)
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) > 64*1024 {
		t.Fatalf("serialized metadata is %d bytes, want at most %d", len(encoded), 64*1024)
	}
}

func TestTacticalPGNDecoderTreatsScannerLimitFailureAsFatal(t *testing.T) {
	contents := `[Event "Oversized"]
[FEN "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"]
[White "solver"][Black "?"]
{` + strings.Repeat("x", 65*1024) + `}
1. e4 *`
	decoder := newTacticalPGNTestDecoder(t, contents, "club-tactics")

	record, err := decoder.Next(context.Background())
	if err == nil || record.Puzzle != nil || record.Rejection != nil {
		t.Fatalf("record/error = %+v/%v, want fatal scanner framing error", record, err)
	}
}

func TestTacticalPGNAdapterDoesNotMatchPGNWithoutTacticalFEN(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ordinary.pgn")
	if err := os.WriteFile(path, []byte(`[Event "Ordinary"] 1. e4 e5 *`), 0o600); err != nil {
		t.Fatal(err)
	}

	_, matched, err := NewTacticalPGNAdapter(chessrules.Rules{}).Inspect(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if matched {
		t.Fatal("Inspect() matched PGN without a tactical FEN")
	}
}

func inspectTacticalPGN(t *testing.T, contents string) (PuzzleAdapter, string, ImportInspection) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "collection.pgn")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	adapter := NewTacticalPGNAdapter(chessrules.Rules{})
	inspection, err := (CollectionImporter{Adapters: []PuzzleAdapter{adapter}}).Inspect(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	return adapter, inspection.Path, inspection
}

func newTacticalPGNTestDecoder(t *testing.T, contents, sourceID string) PuzzleDecoder {
	t.Helper()
	origin := SourceIDPath
	scanner := chess.NewScanner(strings.NewReader(contents))
	if firstGame, scanErr := scanner.ScanGame(); scanErr == nil {
		_, tags, tokenizeErr := tokenizeTacticalPGNGame(firstGame)
		if tokenizeErr == nil && len(tags["SourceId"]) > 0 {
			origin = SourceIDEmbedded
		}
	}
	decoder, err := NewTacticalPGNAdapter(chessrules.Rules{}).NewDecoder(
		strings.NewReader(contents),
		ImportInspection{SourceID: sourceID, SourceIDOrigin: origin},
	)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := decoder.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
	return decoder
}

func tacticalPGNLinearMoves(t *testing.T, nodes []domain.MoveNode) []string {
	t.Helper()
	var moves []string
	for len(nodes) > 0 {
		if len(nodes) != 1 {
			t.Fatalf("solution level has %d branches, want one", len(nodes))
		}
		moves = append(moves, nodes[0].UCI)
		nodes = nodes[0].Children
	}
	return moves
}
