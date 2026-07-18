package puzzles

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/domain"
)

const linearFixture = `# Larion-style sample
4k3/8/8/8/8/8/4P3/4K3 w - - 0 1 e2e4 e8f7 1375
7k/P7/8/8/8/8/8/K7 w - - 0 1 a7a8Q
`

func TestLinearFENAdapterConvertsNormalizedMoveLinesAndDifficulty(t *testing.T) {
	adapter, path, inspection := inspectLinearFEN(t, "larion.txt", linearFixture)
	if inspection.Format != FormatLinearFENUCI ||
		inspection.SourceID != path ||
		inspection.SourceIDOrigin != SourceIDPath ||
		inspection.Filename != "larion.txt" {
		t.Fatalf("inspection = %+v", inspection)
	}

	records := decodeLinearFENFile(t, adapter, path, inspection)
	if len(records) != 2 {
		t.Fatalf("records = %d, want 2", len(records))
	}
	first := requireLinearFENPuzzle(t, records[0])
	if first.Core.DisplayedFEN != "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1" ||
		first.Core.Solver != domain.White || first.Core.SolutionPlies != 2 {
		t.Fatalf("first core = %+v", first.Core)
	}
	if got := linearFENMoves(first.Core.Solution); !slices.Equal(got, []string{"e2e4", "e8f7"}) {
		t.Fatalf("first solution = %q, want [e2e4 e8f7]", got)
	}
	if first.Occurrence.SourceID != path ||
		first.Occurrence.SourceKind != string(FormatLinearFENUCI) ||
		first.Occurrence.ExternalID != "2" ||
		first.Occurrence.SourceFEN != first.Core.DisplayedFEN ||
		first.Occurrence.PreludeUCI != "" || first.Occurrence.Rating != nil ||
		first.Occurrence.Ordinal != 2 ||
		!reflect.DeepEqual(first.Occurrence.Metadata, map[string]any{"sourceDifficulty": 1375}) {
		t.Fatalf("first occurrence = %+v", first.Occurrence)
	}

	second := requireLinearFENPuzzle(t, records[1])
	if second.Core.DisplayedFEN != "7k/P7/8/8/8/8/8/K7 w - - 0 1" ||
		second.Core.Solver != domain.White || second.Core.SolutionPlies != 1 {
		t.Fatalf("second core = %+v", second.Core)
	}
	if got := linearFENMoves(second.Core.Solution); !slices.Equal(got, []string{"a7a8q"}) {
		t.Fatalf("second solution = %q, want [a7a8q]", got)
	}
	if second.Occurrence.SourceID != path || second.Occurrence.ExternalID != "3" ||
		second.Occurrence.Rating != nil || second.Occurrence.Ordinal != 3 ||
		len(second.Occurrence.Metadata) != 0 {
		t.Fatalf("second occurrence = %+v", second.Occurrence)
	}
}

func TestLinearFENDecoderRejectsMalformedRecordsAndRecovers(t *testing.T) {
	const fen = "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"
	contents := "\n" +
		"  # full-line comment\n" +
		fen + "\n" +
		fen + " 1375\n" +
		"not-a-board w - - 0 1 e2e4\n" +
		fen + " e2e5\n" +
		"7k/P7/8/8/8/8/8/K7 w - - 0 1 a7a8x\n" +
		fen + " e2e4 1375 e8f7\n" +
		fen + " e2e4 # inline comments are data\n" +
		fen + " e2e4 Rating 1375\n" +
		fen + " e2e4\n"
	decoder := newLinearFENTestDecoder(t, contents, ImportInspection{
		SourceID:       "/collections/recovery.txt",
		SourceIDOrigin: SourceIDPath,
		Filename:       "recovery.txt",
	})

	for _, wantOrdinal := range []int64{3, 4, 5, 6, 7, 8, 9, 10} {
		record, err := decoder.Next(context.Background())
		if err != nil {
			t.Fatalf("line %d fatal error = %v, want rejection", wantOrdinal, err)
		}
		if record.Rejection == nil || record.Puzzle != nil ||
			record.Rejection.Ordinal != wantOrdinal || record.Rejection.Reason == "" {
			t.Fatalf("line %d record = %+v, want rejection", wantOrdinal, record)
		}
	}

	recovered, err := decoder.Next(context.Background())
	if err != nil || recovered.Puzzle == nil || recovered.Rejection != nil {
		t.Fatalf("recovered record/error = %+v/%v, want puzzle", recovered, err)
	}
	if recovered.Puzzle.Occurrence.Ordinal != 11 ||
		recovered.Puzzle.Occurrence.ExternalID != "11" ||
		!reflect.DeepEqual(linearFENMoves(recovered.Puzzle.Core.Solution), []string{"e2e4"}) {
		t.Fatalf("recovered puzzle = %+v", *recovered.Puzzle)
	}
	if _, err := decoder.Next(context.Background()); !errors.Is(err, io.EOF) {
		t.Fatalf("terminal error = %v, want EOF", err)
	}
}

func TestLinearFENInspectionValidatesOnlyTheFirstMeaningfulLine(t *testing.T) {
	const fen = "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"
	path := filepath.Join(t.TempDir(), "misleading.txt")
	contents := "\n# comment\n" + fen + " 1375\n" + fen + " e2e4\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	inspection, matched, err := NewLinearFENAdapter(chessrules.Rules{}).Inspect(
		context.Background(), path,
	)
	if err != nil {
		t.Fatal(err)
	}
	if matched || inspection != (ImportInspection{}) {
		t.Fatalf("inspection/matched = %+v/%v, want no content match", inspection, matched)
	}
}

func TestLinearFENDecoderTreatsOverlongLineAsFatalFramingError(t *testing.T) {
	decoder := newLinearFENTestDecoder(t, strings.Repeat("x", (1<<20)+1), ImportInspection{
		SourceID:       "/collections/oversized.txt",
		SourceIDOrigin: SourceIDPath,
		Filename:       "oversized.txt",
	})

	record, err := decoder.Next(context.Background())
	if !errors.Is(err, bufio.ErrTooLong) || record.Puzzle != nil || record.Rejection != nil {
		t.Fatalf("record/error = %+v/%v, want fatal bufio.ErrTooLong", record, err)
	}
}

func TestLinearFENInspectionReportsOverlongFirstMeaningfulLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "oversized.txt")
	if err := os.WriteFile(path, []byte(strings.Repeat("x", (1<<20)+1)), 0o600); err != nil {
		t.Fatal(err)
	}

	_, matched, err := NewLinearFENAdapter(chessrules.Rules{}).Inspect(context.Background(), path)
	if matched || !errors.Is(err, bufio.ErrTooLong) {
		t.Fatalf("matched/error = %v/%v, want false/bufio.ErrTooLong", matched, err)
	}
}

func TestLinearFENDecoderHonorsCancellationAndClose(t *testing.T) {
	decoder := newLinearFENTestDecoder(t, linearFixture, ImportInspection{
		SourceID:       "/collections/lifecycle.txt",
		SourceIDOrigin: SourceIDPath,
		Filename:       "lifecycle.txt",
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := decoder.Next(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled Next() error = %v, want context.Canceled", err)
	}
	if err := decoder.Close(); err != nil {
		t.Fatal(err)
	}
	if err := decoder.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := decoder.Next(context.Background()); !errors.Is(err, io.EOF) {
		t.Fatalf("Next() after Close error = %v, want EOF", err)
	}
}

func inspectLinearFEN(
	t *testing.T,
	filename string,
	contents string,
) (PuzzleAdapter, string, ImportInspection) {
	t.Helper()
	path := filepath.Join(t.TempDir(), filename)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	adapter := NewLinearFENAdapter(chessrules.Rules{})
	inspection, err := (CollectionImporter{Adapters: []PuzzleAdapter{adapter}}).Inspect(
		context.Background(), path,
	)
	if err != nil {
		t.Fatal(err)
	}
	return adapter, inspection.Path, inspection
}

func newLinearFENTestDecoder(
	t *testing.T,
	contents string,
	inspection ImportInspection,
) PuzzleDecoder {
	t.Helper()
	decoder, err := NewLinearFENAdapter(chessrules.Rules{}).NewDecoder(
		strings.NewReader(contents), inspection,
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

func decodeLinearFENFile(
	t *testing.T,
	adapter PuzzleAdapter,
	path string,
	inspection ImportInspection,
) []DecodedRecord {
	t.Helper()
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
	var records []DecodedRecord
	for {
		record, err := decoder.Next(context.Background())
		if errors.Is(err, io.EOF) {
			return records
		}
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, record)
	}
}

func requireLinearFENPuzzle(t *testing.T, record DecodedRecord) TrainingPuzzle {
	t.Helper()
	if record.Puzzle == nil || record.Rejection != nil {
		t.Fatalf("record = %+v, want puzzle", record)
	}
	return *record.Puzzle
}

func linearFENMoves(nodes []domain.MoveNode) []string {
	var moves []string
	for len(nodes) == 1 {
		moves = append(moves, nodes[0].UCI)
		nodes = nodes[0].Children
	}
	return moves
}
