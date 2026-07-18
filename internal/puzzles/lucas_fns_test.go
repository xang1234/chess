package puzzles

import (
	"bufio"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/domain"
)

const lucasFNSBranchedPuzzle = `4k3/8/8/8/8/8/4P3/4K3 w - - 0 1|Difficulty **|1. e4 Kf7 (1... Kd7) 2. Kf2 *`

func TestLucasFNSAdapterConvertsMainlineAndVariations(t *testing.T) {
	adapter, path, inspection := inspectLucasFNS(t, "Pin.fns", lucasFNSBranchedPuzzle)
	if inspection.Format != FormatLucasFNS ||
		inspection.SourceID != path ||
		inspection.SourceIDOrigin != SourceIDPath ||
		inspection.Filename != "Pin.fns" {
		t.Fatalf("inspection = %+v", inspection)
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
	if puzzle.Core.DisplayedFEN != "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1" ||
		puzzle.Core.Solver != domain.White ||
		puzzle.Core.SolutionPlies != 3 {
		t.Fatalf("core = %+v", puzzle.Core)
	}
	if puzzle.Occurrence.SourceID != path ||
		puzzle.Occurrence.SourceKind != string(FormatLucasFNS) ||
		puzzle.Occurrence.ExternalID != "1" ||
		puzzle.Occurrence.SourceFEN != puzzle.Core.DisplayedFEN ||
		puzzle.Occurrence.PreludeUCI != "" ||
		puzzle.Occurrence.Rating != nil ||
		puzzle.Occurrence.Ordinal != 1 {
		t.Fatalf("occurrence = %+v", puzzle.Occurrence)
	}
	if !reflect.DeepEqual(puzzle.Occurrence.Metadata, map[string]any{
		"description":      "Difficulty **",
		"sourceDifficulty": "**",
	}) {
		t.Fatalf("metadata = %#v", puzzle.Occurrence.Metadata)
	}
	if !reflect.DeepEqual(puzzle.Occurrence.Themes, []string{"pin"}) {
		t.Fatalf("themes = %q, want pin", puzzle.Occurrence.Themes)
	}

	root := puzzle.Core.Solution
	if len(root) != 1 || root[0].UCI != "e2e4" {
		t.Fatalf("root = %+v, want e2e4", root)
	}
	opponent := root[0].Children
	if len(opponent) != 2 || opponent[0].UCI != "e8f7" || opponent[1].UCI != "e8d7" {
		t.Fatalf("opponent branches = %+v, want e8f7 and e8d7", opponent)
	}
	if len(opponent[0].Children) != 1 || opponent[0].Children[0].UCI != "e1f2" {
		t.Fatalf("mainline continuation = %+v, want e1f2", opponent[0].Children)
	}
	if len(opponent[1].Children) != 0 {
		t.Fatalf("variation continuation = %+v, want none", opponent[1].Children)
	}
}

func TestLucasFNSDecoderIgnoresBlankCommentsAndRecoversAfterRejectedLines(t *testing.T) {
	validFEN := "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"
	contents := "\n" +
		"  # Lucas collection comment\n" +
		validFEN + "|missing movetext field\n" +
		"not a FEN|description|1. e4 *\n" +
		validFEN + "|bad SAN|1. e5 *\n" +
		validFEN + "|empty movetext|   \n" +
		validFEN + "|recovered|1. e4 *"
	decoder := newLucasFNSTestDecoder(t, contents, ImportInspection{
		SourceID:       "/collections/recovery.fns",
		SourceIDOrigin: SourceIDPath,
		Filename:       "recovery.fns",
	})

	for _, wantOrdinal := range []int64{3, 4, 5, 6} {
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
	if recovered.Puzzle.Occurrence.Ordinal != 7 ||
		recovered.Puzzle.Occurrence.ExternalID != "7" ||
		recovered.Puzzle.Core.Solution[0].UCI != "e2e4" {
		t.Fatalf("recovered puzzle = %+v", *recovered.Puzzle)
	}
	if _, err := decoder.Next(context.Background()); !errors.Is(err, io.EOF) {
		t.Fatalf("terminal error = %v, want EOF", err)
	}
}

func TestLucasFNSDecoderTreatsOverlongLineAsFatalFramingError(t *testing.T) {
	decoder := newLucasFNSTestDecoder(t, strings.Repeat("x", (1<<20)+1), ImportInspection{
		SourceID:       "/collections/oversized.fns",
		SourceIDOrigin: SourceIDPath,
		Filename:       "oversized.fns",
	})

	record, err := decoder.Next(context.Background())
	if !errors.Is(err, bufio.ErrTooLong) || record.Puzzle != nil || record.Rejection != nil {
		t.Fatalf("record/error = %+v/%v, want fatal bufio.ErrTooLong", record, err)
	}
}

func TestLucasFNSAdapterNormalizesFilenameThemeAndExcludesGenericStems(t *testing.T) {
	for _, test := range []struct {
		filename string
		want     []string
	}{
		{filename: "Pin.fns", want: []string{"pin"}},
		{filename: "Back-RankMate!.fns", want: []string{"back-rank-mate"}},
		{filename: "Tactics.fns"},
	} {
		t.Run(test.filename, func(t *testing.T) {
			adapter, path, inspection := inspectLucasFNS(
				t,
				test.filename,
				"4k3/8/8/8/8/8/4P3/4K3 w - - 0 1|plain text|1. e4 *",
			)
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
			if err != nil || record.Puzzle == nil {
				t.Fatalf("record/error = %+v/%v, want puzzle", record, err)
			}
			if !reflect.DeepEqual(record.Puzzle.Occurrence.Themes, test.want) {
				t.Fatalf("themes = %q, want %q", record.Puzzle.Occurrence.Themes, test.want)
			}
		})
	}
}

func TestLucasFNSDecoderHonorsCancellationAndClose(t *testing.T) {
	decoder := newLucasFNSTestDecoder(t, lucasFNSBranchedPuzzle, ImportInspection{
		SourceID:       "/collections/lifecycle.fns",
		SourceIDOrigin: SourceIDPath,
		Filename:       "lifecycle.fns",
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

func inspectLucasFNS(
	t *testing.T,
	filename string,
	contents string,
) (PuzzleAdapter, string, ImportInspection) {
	t.Helper()
	path := filepath.Join(t.TempDir(), filename)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	adapter := NewLucasFNSAdapter(chessrules.Rules{})
	inspection, err := (CollectionImporter{Adapters: []PuzzleAdapter{adapter}}).Inspect(
		context.Background(),
		path,
	)
	if err != nil {
		t.Fatal(err)
	}
	return adapter, inspection.Path, inspection
}

func newLucasFNSTestDecoder(
	t *testing.T,
	contents string,
	inspection ImportInspection,
) PuzzleDecoder {
	t.Helper()
	decoder, err := NewLucasFNSAdapter(chessrules.Rules{}).NewDecoder(
		strings.NewReader(contents),
		inspection,
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
