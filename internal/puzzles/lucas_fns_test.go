package puzzles

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/domain"
	"chess-trainer/internal/importing"
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
	if puzzle.Occurrence.SourceID != "" ||
		puzzle.Occurrence.SourceKind != "" ||
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
	decoder := newLucasFNSTestDecoder(t, contents, importing.Inspection{
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

func TestLucasFNSDecoderRejectsMeaningfulMovetextAfterFirstResult(t *testing.T) {
	const fen = "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"
	for _, test := range []struct {
		name     string
		movetext string
	}{
		{name: "moves after result", movetext: "1. e4 * 1... Kf7 *"},
		{name: "tag pair after result", movetext: `1. e4 * [Event "smuggled"]`},
	} {
		t.Run(test.name, func(t *testing.T) {
			decoder := newLucasFNSTestDecoder(
				t,
				fen+"|trailing content|"+test.movetext,
				importing.Inspection{
					SourceID:       "/collections/trailing.fns",
					SourceIDOrigin: SourceIDPath,
					Filename:       "trailing.fns",
				},
			)

			record, err := decoder.Next(context.Background())
			if err != nil {
				t.Fatalf("Next() fatal error = %v, want record rejection", err)
			}
			if record.Rejection == nil || record.Puzzle != nil ||
				!strings.Contains(record.Rejection.Reason, "after result") {
				t.Fatalf("record = %+v, want trailing-movetext rejection", record)
			}
		})
	}
}

func TestLucasFNSDecoderAcceptsCommentsNAGsAndVariationsBeforeResult(t *testing.T) {
	contents := "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1|annotated|" +
		"1. e4 $1 {solver note} Kf7 (1... Kd7 $2 {variation note}) *"
	decoder := newLucasFNSTestDecoder(t, contents, importing.Inspection{
		SourceID:       "/collections/annotated.fns",
		SourceIDOrigin: SourceIDPath,
		Filename:       "annotated.fns",
	})

	record, err := decoder.Next(context.Background())
	if err != nil || record.Puzzle == nil || record.Rejection != nil {
		t.Fatalf("record/error = %+v/%v, want accepted annotated puzzle", record, err)
	}
	root := record.Puzzle.Core.Solution
	if len(root) != 1 || root[0].UCI != "e2e4" || len(root[0].Children) != 2 ||
		root[0].Children[0].UCI != "e8f7" || root[0].Children[1].UCI != "e8d7" {
		t.Fatalf("annotated solution tree = %+v", root)
	}
}

func TestLucasFNSDecoderHonorsLineContractForLargeValidPGNComments(t *testing.T) {
	for _, targetBytes := range []int{96 * 1024, maxLucasFNSLineBytes} {
		t.Run(strconv.Itoa(targetBytes)+" bytes", func(t *testing.T) {
			const fen = "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"
			prefix := fen + "|comment-heavy|1. e4 {"
			suffix := "} Kf7 (1... Kd7 {variation note}) 2. Kf2 *"
			padding := targetBytes - len(prefix) - len(suffix)
			if padding < 1 {
				t.Fatalf("target %d is too small for fixture", targetBytes)
			}
			line := prefix + strings.Repeat("x", padding) + suffix
			if len(line) != targetBytes {
				t.Fatalf("fixture is %d bytes, want %d", len(line), targetBytes)
			}
			decoder := newLucasFNSTestDecoder(t, line, importing.Inspection{
				SourceID:       "/collections/comment-heavy.fns",
				SourceIDOrigin: SourceIDPath,
				Filename:       "comment-heavy.fns",
			})

			record, err := decoder.Next(context.Background())
			if err != nil || record.Puzzle == nil || record.Rejection != nil {
				reason := ""
				if record.Rejection != nil {
					reason = record.Rejection.Reason
				}
				t.Fatalf(
					"record/rejection/error = %+v/%q/%v, want accepted %d-byte puzzle",
					record,
					reason,
					err,
					targetBytes,
				)
			}
			root := record.Puzzle.Core.Solution
			if len(root) != 1 || root[0].UCI != "e2e4" || len(root[0].Children) != 2 ||
				root[0].Children[0].UCI != "e8f7" ||
				root[0].Children[1].UCI != "e8d7" ||
				len(root[0].Children[0].Children) != 1 ||
				root[0].Children[0].Children[0].UCI != "e1f2" {
				t.Fatalf("solution tree = %+v, want preserved mainline and variation", root)
			}
		})
	}
}

func TestLucasFNSDecoderTreatsOverlongLineAsFatalFramingError(t *testing.T) {
	decoder := newLucasFNSTestDecoder(t, strings.Repeat("x", (1<<20)+1), importing.Inspection{
		SourceID:       "/collections/oversized.fns",
		SourceIDOrigin: SourceIDPath,
		Filename:       "oversized.fns",
	})

	record, err := decoder.Next(context.Background())
	if !errors.Is(err, bufio.ErrTooLong) || record.Puzzle != nil || record.Rejection != nil {
		t.Fatalf("record/error = %+v/%v, want fatal bufio.ErrTooLong", record, err)
	}
}

func TestLucasFNSDecoderRejectsOversizedMetadataAndRecoversNextLine(t *testing.T) {
	const fen = "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"
	description := strings.Repeat("x", 64*1024) + " Difficulty **"
	encoded, err := json.Marshal(map[string]any{
		"description":      description,
		"sourceDifficulty": "**",
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(encoded) <= 64*1024 {
		t.Fatalf("oversized metadata fixture serialized to only %d bytes", len(encoded))
	}
	contents := fen + "|" + description + "|1. e4 *\n" +
		fen + "|recovered|1. e4 *"
	decoder := newLucasFNSTestDecoder(t, contents, importing.Inspection{
		SourceID:       "/collections/metadata.fns",
		SourceIDOrigin: SourceIDPath,
		Filename:       "metadata.fns",
	})

	first, err := decoder.Next(context.Background())
	if err != nil {
		t.Fatalf("first Next() fatal error = %v, want record rejection", err)
	}
	if first.Rejection == nil || first.Puzzle != nil || first.Rejection.Ordinal != 1 ||
		!strings.Contains(first.Rejection.Reason, "metadata exceeds maximum of 65536 bytes") {
		t.Fatalf("first record = %+v, want metadata-limit rejection", first)
	}
	second, err := decoder.Next(context.Background())
	if err != nil || second.Puzzle == nil || second.Rejection != nil {
		t.Fatalf("second record/error = %+v/%v, want recovered puzzle", second, err)
	}
	if second.Puzzle.Occurrence.Ordinal != 2 ||
		second.Puzzle.Occurrence.ExternalID != "2" ||
		second.Puzzle.Occurrence.Metadata["description"] != "recovered" {
		t.Fatalf("recovered puzzle = %+v", *second.Puzzle)
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
	decoder := newLucasFNSTestDecoder(t, lucasFNSBranchedPuzzle, importing.Inspection{
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
) (PuzzleAdapter, string, importing.Inspection) {
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
	inspection importing.Inspection,
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
