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

const canonicalJSONWhitePuzzle = `{
  "id": "json-1",
  "displayedFen": "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1",
  "solver": "white",
  "solution": [{"uci":"e2e4","children":[
    {"uci":"e8e7","children":[]},
    {"uci":"e8f7","children":[]}
  ]}],
  "rating": 1450,
  "themes": ["fork", " fork ", ""],
  "metadata": {"chapter": 3}
}`

const canonicalJSONSource = `{
  "id": "club-json",
  "name": "Club JSON",
  "url": "https://example.test/club",
  "attribution": "Club coach"
}`

func TestCanonicalJSONSolutionRepresentationDoesNotRetainRawSubtrees(t *testing.T) {
	field, present := reflect.TypeOf(canonicalPuzzle{}).FieldByName("Solution")
	if !present {
		t.Fatal("canonicalPuzzle has no Solution field")
	}
	want := reflect.TypeOf([]domain.MoveNode(nil))
	if field.Type != want {
		t.Fatalf("canonicalPuzzle.Solution type = %v, want %v", field.Type, want)
	}
}

func TestCanonicalJSONValueFramingCapsCaptureAndConsumesOversizedValue(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(
		`{"blob":"` + strings.Repeat("x", 256) + `"}, {"next":true}`,
	))
	raw, oversized, err := readBoundedJSONValue(reader, 32)
	if err != nil {
		t.Fatal(err)
	}
	if !oversized || len(raw) != 33 {
		t.Fatalf("first capture oversized/bytes = %v/%d, want true/33", oversized, len(raw))
	}
	separator, err := readBoundedJSONNonSpace(reader)
	if err != nil || separator != ',' {
		t.Fatalf("separator/error = %q/%v, want comma", separator, err)
	}
	second, oversized, err := readBoundedJSONValue(reader, 32)
	if err != nil {
		t.Fatal(err)
	}
	if oversized || string(second) != `{"next":true}` {
		t.Fatalf("second capture oversized/value = %v/%q", oversized, second)
	}
}

func TestCanonicalJSONCaptureStartsSmallAndNeverGrowsPastLimit(t *testing.T) {
	large := newBoundedJSONCapture(maxCanonicalJSONPuzzleBytes)
	if capacity := cap(large.bytes); capacity > 256 {
		t.Fatalf("initial capacity = %d, want at most 256", capacity)
	}

	const limit = 1000
	capture := newBoundedJSONCapture(limit)
	for index := 0; index < limit+100; index++ {
		capture.append('x')
	}
	if !capture.oversized || len(capture.bytes) != limit+1 || cap(capture.bytes) > limit+1 {
		t.Fatalf(
			"grown capture oversized/len/cap = %v/%d/%d, want true/%d/at-most-%d",
			capture.oversized,
			len(capture.bytes),
			cap(capture.bytes),
			limit+1,
			limit+1,
		)
	}
}

func TestCanonicalJSONNestingLimitCountsContainersNotScalarLeaf(t *testing.T) {
	exact := strings.Repeat("[", maxBoundedJSONNestingDepth) + "0" +
		strings.Repeat("]", maxBoundedJSONNestingDepth)
	if _, _, err := readBoundedJSONValue(
		bufio.NewReader(strings.NewReader(exact)),
		len(exact),
	); err != nil {
		t.Fatalf("exact nesting limit rejected: %v", err)
	}

	tooDeep := "[" + exact + "]"
	if _, _, err := readBoundedJSONValue(
		bufio.NewReader(strings.NewReader(tooDeep)),
		len(tooDeep),
	); err == nil {
		t.Fatal("nesting above limit accepted")
	}
}

func TestCanonicalJSONValueFramingPreservesNestedValueExactly(t *testing.T) {
	reader := bufio.NewReader(strings.NewReader(canonicalJSONWhitePuzzle))
	raw, oversized, err := readBoundedJSONValue(reader, maxCanonicalJSONPuzzleBytes)
	if err != nil {
		t.Fatal(err)
	}
	if oversized || string(raw) != canonicalJSONWhitePuzzle {
		t.Fatalf("framed nested value oversized/matches = %v/%v\nraw: %s", oversized, string(raw) == canonicalJSONWhitePuzzle, raw)
	}
}

func TestCanonicalJSONValueFramingRejectsMalformedAndTruncatedValues(t *testing.T) {
	tests := []struct {
		name  string
		value string
	}{
		{name: "invalid escape after capture limit", value: `{"blob":"` + strings.Repeat("x", 64) + `\q"}`},
		{name: "truncated oversized string", value: `{"blob":"` + strings.Repeat("x", 64)},
		{name: "leading zero", value: `{"value":01}`},
		{name: "missing fraction digit", value: `{"value":1.}`},
		{name: "missing exponent digit", value: `{"value":1e}`},
		{name: "invalid literal", value: `{"value":truX}`},
		{name: "object trailing comma", value: `{"value":1,}`},
		{name: "array trailing comma", value: `[1,]`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			reader := bufio.NewReader(strings.NewReader(test.value))
			if _, _, err := readBoundedJSONValue(reader, 16); err == nil {
				t.Fatalf("readBoundedJSONValue(%q) error = nil", test.value)
			}
		})
	}
}

func TestCanonicalJSONDecoderContinuesAfterOversizedPuzzle(t *testing.T) {
	oversized := `{"id":"` + strings.Repeat("x", 2*1024*1024) + `"}`
	contents := `{
  "schema":"chess-trainer-puzzles/v1",
  "puzzles":[` + oversized + `,` + canonicalJSONWhitePuzzle + `],
  "source":` + canonicalJSONSource + `
}`
	adapter, path, inspection := inspectCanonicalJSON(t, contents)
	records := decodeCanonicalJSONFile(t, adapter, path, inspection)
	if len(records) != 2 || records[0].Rejection == nil || records[1].Puzzle == nil {
		t.Fatalf("records = %+v, want oversized rejection then puzzle", records)
	}
	if records[0].Rejection.Ordinal != 1 || !strings.Contains(records[0].Rejection.Reason, "maximum") {
		t.Fatalf("first rejection = %+v", records[0].Rejection)
	}
}

func TestCanonicalJSONAdapterStreamsPuzzlesBeforeSourceAndAppliesDefaults(t *testing.T) {
	contents := `{
  "schema": "chess-trainer-puzzles/v1",
  "puzzles": [` + canonicalJSONWhitePuzzle + `, {
    "displayedFen": "4k3/4p3/8/8/8/8/8/4K3 b - - 0 1",
    "solver": "black",
    "solution": [{"uci":"e7e5","children":[{"uci":"e1f2"}]}],
    "themes": [" pin ", "fork", "pin"],
    "url": "https://example.test/puzzle-2",
    "attribution": "Guest analyst"
  }],
  "source": ` + canonicalJSONSource + `
}`

	adapter, path, inspection := inspectCanonicalJSON(t, contents)
	if inspection.Format != FormatCanonicalJSON ||
		inspection.SourceID != "club-json" ||
		inspection.SourceIDOrigin != SourceIDEmbedded ||
		inspection.SourceName != "Club JSON" ||
		inspection.URL != "https://example.test/club" ||
		inspection.Attribution != "Club coach" {
		t.Fatalf("inspection = %+v", inspection)
	}

	records := decodeCanonicalJSONFile(t, adapter, path, inspection)
	if len(records) != 2 {
		t.Fatalf("records = %d, want 2", len(records))
	}
	first := requireCanonicalPuzzle(t, records[0])
	if first.Occurrence.SourceID != "" ||
		first.Occurrence.SourceKind != "" ||
		first.Occurrence.ExternalID != "json-1" ||
		first.Occurrence.URL != inspection.URL ||
		first.Occurrence.Attribution != inspection.Attribution ||
		first.Occurrence.Rating == nil || *first.Occurrence.Rating != 1450 ||
		first.Occurrence.Ordinal != 1 {
		t.Fatalf("first occurrence = %+v", first.Occurrence)
	}
	if first.Core.Solver != domain.White || first.Core.SolutionPlies != 2 {
		t.Fatalf("first core = %+v", first.Core)
	}
	if !slices.Equal(first.Occurrence.Themes, []string{"fork"}) {
		t.Fatalf("first themes = %q, want [fork]", first.Occurrence.Themes)
	}
	if !reflect.DeepEqual(first.Occurrence.Metadata, map[string]any{"chapter": float64(3)}) {
		t.Fatalf("first metadata = %#v", first.Occurrence.Metadata)
	}

	second := requireCanonicalPuzzle(t, records[1])
	if second.Occurrence.ExternalID != "2" ||
		second.Occurrence.URL != "https://example.test/puzzle-2" ||
		second.Occurrence.Attribution != "Guest analyst" ||
		second.Occurrence.Rating != nil ||
		second.Occurrence.Ordinal != 2 {
		t.Fatalf("second occurrence = %+v", second.Occurrence)
	}
	if second.Core.Solver != domain.Black || second.Core.SolutionPlies != 2 {
		t.Fatalf("second core = %+v", second.Core)
	}
	if !slices.Equal(second.Occurrence.Themes, []string{"fork", "pin"}) {
		t.Fatalf("second themes = %q, want [fork pin]", second.Occurrence.Themes)
	}
}

func TestCanonicalJSONInspectionRejectsInvalidTopLevelStructure(t *testing.T) {
	tests := []struct {
		name     string
		contents string
		want     string
	}{
		{
			name:     "unsupported schema",
			contents: `{"schema":"chess-trainer-puzzles/v2","puzzles":[]}`,
			want:     "schema",
		},
		{
			name:     "missing schema",
			contents: `{"puzzles":[]}`,
			want:     "schema",
		},
		{
			name:     "missing puzzles",
			contents: `{"schema":"chess-trainer-puzzles/v1"}`,
			want:     "puzzles",
		},
		{
			name:     "duplicate top-level key",
			contents: `{"schema":"chess-trainer-puzzles/v1","schema":"chess-trainer-puzzles/v1","puzzles":[]}`,
			want:     "duplicate",
		},
		{
			name:     "unknown top-level field",
			contents: `{"schema":"chess-trainer-puzzles/v1","puzzles":[],"future":true}`,
			want:     "unknown",
		},
		{
			name:     "unknown source field",
			contents: `{"schema":"chess-trainer-puzzles/v1","source":{"id":"club-json","future":true},"puzzles":[]}`,
			want:     "unknown",
		},
		{
			name:     "duplicate source field",
			contents: `{"schema":"chess-trainer-puzzles/v1","source":{"id":"club-json","id":"club-json"},"puzzles":[]}`,
			want:     "duplicate",
		},
		{
			name:     "trailing JSON value",
			contents: `{"schema":"chess-trainer-puzzles/v1","puzzles":[]} true`,
			want:     "trailing",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := writeCanonicalJSON(t, test.contents)
			_, _, err := NewCanonicalJSONAdapter(chessrules.Rules{}).Inspect(context.Background(), path)
			if err == nil || !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(test.want)) {
				t.Fatalf("Inspect() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestCanonicalJSONDecoderRejectsInvalidPuzzleRecords(t *testing.T) {
	const base = `{
  "id":"base",
  "displayedFen":"4k3/8/8/8/8/8/4P3/4K3 w - - 0 1",
  "solver":"white",
  "solution":[{"uci":"e2e4"}]
}`
	tests := []struct {
		name   string
		puzzle string
		want   string
	}{
		{
			name:   "unknown puzzle field",
			puzzle: strings.Replace(base, `"id":"base"`, `"id":"base","future":true`, 1),
			want:   "unknown",
		},
		{
			name:   "duplicate puzzle field",
			puzzle: strings.Replace(base, `"id":"base"`, `"id":"base","id":"again"`, 1),
			want:   "duplicate",
		},
		{
			name:   "unknown move field",
			puzzle: strings.Replace(base, `"uci":"e2e4"`, `"uci":"e2e4","san":"e4"`, 1),
			want:   "unknown",
		},
		{
			name:   "duplicate move field",
			puzzle: strings.Replace(base, `"uci":"e2e4"`, `"uci":"e2e4","uci":"e2e4"`, 1),
			want:   "duplicate",
		},
		{
			name:   "invalid solver",
			puzzle: strings.Replace(base, `"solver":"white"`, `"solver":"green"`, 1),
			want:   "solver",
		},
		{
			name:   "rating below range",
			puzzle: strings.Replace(base, `"solver":"white"`, `"solver":"white","rating":99`, 1),
			want:   "rating",
		},
		{
			name:   "rating above range",
			puzzle: strings.Replace(base, `"solver":"white"`, `"solver":"white","rating":4001`, 1),
			want:   "rating",
		},
		{
			name:   "fractional rating",
			puzzle: strings.Replace(base, `"solver":"white"`, `"solver":"white","rating":1450.5`, 1),
			want:   "rating",
		},
		{
			name:   "negative popularity",
			puzzle: strings.Replace(base, `"solver":"white"`, `"solver":"white","popularity":-1`, 1),
			want:   "popularity",
		},
		{
			name:   "negative play count",
			puzzle: strings.Replace(base, `"solver":"white"`, `"solver":"white","playCount":-1`, 1),
			want:   "playCount",
		},
		{
			name:   "non-object metadata",
			puzzle: strings.Replace(base, `"solver":"white"`, `"solver":"white","metadata":[]`, 1),
			want:   "metadata",
		},
		{
			name:   "null metadata",
			puzzle: strings.Replace(base, `"solver":"white"`, `"solver":"white","metadata":null`, 1),
			want:   "metadata",
		},
		{
			name:   "metadata above byte limit",
			puzzle: strings.Replace(base, `"solver":"white"`, `"solver":"white","metadata":{"blob":"`+strings.Repeat("x", 64*1024)+`"}`, 1),
			want:   "metadata",
		},
		{
			name:   "puzzle above byte limit",
			puzzle: strings.Replace(base, `"id":"base"`, `"id":"`+strings.Repeat("x", 2*1024*1024)+`"`, 1),
			want:   "maximum",
		},
		{
			name:   "illegal tree move",
			puzzle: strings.Replace(base, `"uci":"e2e4"`, `"uci":"e2e5"`, 1),
			want:   "e2e5",
		},
		{
			name:   "duplicate sibling branch",
			puzzle: strings.Replace(base, `{"uci":"e2e4"}`, `{"uci":"e2e4"},{"uci":"e2e4"}`, 1),
			want:   "duplicate sibling",
		},
		{
			name:   "empty solution",
			puzzle: strings.Replace(base, `[{"uci":"e2e4"}]`, `[]`, 1),
			want:   "solution is empty",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record := decodeOneCanonicalJSONPuzzle(t, test.puzzle)
			if record.Rejection == nil || record.Puzzle != nil {
				t.Fatalf("record = %+v, want rejection", record)
			}
			if record.Rejection.Ordinal != 1 || !strings.Contains(strings.ToLower(record.Rejection.Reason), strings.ToLower(test.want)) {
				t.Fatalf("rejection = %+v, want ordinal 1 and %q", *record.Rejection, test.want)
			}
		})
	}
}

func TestCanonicalJSONDecoderValidatesPresentationFields(t *testing.T) {
	valid := `{
  "id":"presentation",
  "sourceFen":"4k3/4p3/8/8/8/8/8/4K3 b - - 0 1",
  "preludeUci":"e7e5",
  "displayedFen":"4k3/8/8/4p3/8/8/8/4K3 w - e6 0 2",
  "solver":"white",
  "solution":[{"uci":"e1f2"}]
}`
	record := decodeOneCanonicalJSONPuzzle(t, valid)
	puzzle := requireCanonicalPuzzle(t, record)
	if puzzle.Occurrence.SourceFEN != "4k3/4p3/8/8/8/8/8/4K3 b - - 0 1" ||
		puzzle.Occurrence.PreludeUCI != "e7e5" ||
		puzzle.Core.DisplayedFEN != "4k3/8/8/4p3/8/8/8/4K3 w - e6 0 2" {
		t.Fatalf("presentation/core = %+v/%+v", puzzle.Occurrence, puzzle.Core)
	}

	tests := []struct {
		name   string
		puzzle string
		want   string
	}{
		{
			name:   "source FEN only",
			puzzle: strings.Replace(valid, `  "preludeUci":"e7e5",`+"\n", "", 1),
			want:   "together",
		},
		{
			name:   "prelude only",
			puzzle: strings.Replace(valid, `  "sourceFen":"4k3/4p3/8/8/8/8/8/4K3 b - - 0 1",`+"\n", "", 1),
			want:   "together",
		},
		{
			name:   "prelude result differs",
			puzzle: strings.Replace(valid, `"displayedFen":"4k3/8/8/4p3/8/8/8/4K3 w - e6 0 2"`, `"displayedFen":"4k3/8/8/8/4p3/8/8/4K3 w - - 0 2"`, 1),
			want:   "displayedFen",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			record := decodeOneCanonicalJSONPuzzle(t, test.puzzle)
			if record.Rejection == nil || !strings.Contains(strings.ToLower(record.Rejection.Reason), strings.ToLower(test.want)) {
				t.Fatalf("record = %+v, want rejection containing %q", record, test.want)
			}
		})
	}
}

func TestCanonicalJSONDecoderContinuesAfterSemanticRejection(t *testing.T) {
	invalid := strings.Replace(canonicalJSONWhitePuzzle, `"solver": "white"`, `"solver": "green"`, 1)
	contents := `{
  "schema":"chess-trainer-puzzles/v1",
  "source":` + canonicalJSONSource + `,
  "puzzles":[` + canonicalJSONWhitePuzzle + `,` + invalid + `,` + canonicalJSONWhitePuzzle + `]
}`
	adapter, path, inspection := inspectCanonicalJSON(t, contents)
	records := decodeCanonicalJSONFile(t, adapter, path, inspection)
	if len(records) != 3 || records[0].Puzzle == nil || records[1].Rejection == nil || records[2].Puzzle == nil {
		t.Fatalf("records = %+v, want puzzle/rejection/puzzle", records)
	}
	if records[1].Rejection.Ordinal != 2 {
		t.Fatalf("middle rejection = %+v, want ordinal 2", *records[1].Rejection)
	}
}

func TestCanonicalJSONDecoderTreatsBrokenFramingAsFatal(t *testing.T) {
	contents := `{
  "schema":"chess-trainer-puzzles/v1",
  "source":` + canonicalJSONSource + `,
  "puzzles":[` + canonicalJSONWhitePuzzle + `,{"id":"broken","solver":]
}`
	decoder, err := NewCanonicalJSONAdapter(chessrules.Rules{}).NewDecoder(
		strings.NewReader(contents),
		puzzleInspection{
			SourceID:       "club-json",
			SourceIDOrigin: SourceIDEmbedded,
			SourceName:     "Club JSON",
			URL:            "https://example.test/club",
			Attribution:    "Club coach",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	defer decoder.Close()
	first, err := decoder.Next(context.Background())
	if err != nil || first.Puzzle == nil {
		t.Fatalf("first record/error = %+v/%v, want puzzle", first, err)
	}
	if _, err := decoder.Next(context.Background()); err == nil {
		t.Fatal("second Next() error = nil, want fatal JSON framing error")
	}
}

func TestCanonicalJSONDecoderTreatsInvalidRecordDelimiterAsFatal(t *testing.T) {
	contents := `{
  "schema":"chess-trainer-puzzles/v1",
  "source":` + canonicalJSONSource + `,
  "puzzles":[` + canonicalJSONWhitePuzzle + `,` + canonicalJSONWhitePuzzle + `x]
}`
	decoder, err := NewCanonicalJSONAdapter(chessrules.Rules{}).NewDecoder(
		strings.NewReader(contents),
		puzzleInspection{
			SourceID:       "club-json",
			SourceIDOrigin: SourceIDEmbedded,
			SourceName:     "Club JSON",
			URL:            "https://example.test/club",
			Attribution:    "Club coach",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	defer decoder.Close()
	first, err := decoder.Next(context.Background())
	if err != nil || first.Puzzle == nil {
		t.Fatalf("first record/error = %+v/%v, want puzzle", first, err)
	}
	if record, err := decoder.Next(context.Background()); err == nil || record.Puzzle != nil || record.Rejection != nil {
		t.Fatalf("second record/error = %+v/%v, want immediate fatal delimiter error", record, err)
	}
}

func TestCanonicalJSONDecoderFatallyRevalidatesInspectionIdentity(t *testing.T) {
	tests := []struct {
		name       string
		contents   string
		inspection puzzleInspection
		want       string
	}{
		{
			name:       "schema changed",
			contents:   `{"schema":"chess-trainer-puzzles/v2","source":{"id":"club-json"},"puzzles":[]}`,
			inspection: puzzleInspection{SourceID: "club-json", SourceIDOrigin: SourceIDEmbedded},
			want:       "schema",
		},
		{
			name:       "source changed",
			contents:   `{"schema":"chess-trainer-puzzles/v1","source":{"id":"other-json"},"puzzles":[]}`,
			inspection: puzzleInspection{SourceID: "club-json", SourceIDOrigin: SourceIDEmbedded},
			want:       "source",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decoder, err := NewCanonicalJSONAdapter(chessrules.Rules{}).NewDecoder(strings.NewReader(test.contents), test.inspection)
			if err != nil {
				t.Fatal(err)
			}
			defer decoder.Close()
			if _, err := decoder.Next(context.Background()); err == nil || !strings.Contains(strings.ToLower(err.Error()), test.want) {
				t.Fatalf("Next() error = %v, want fatal %q mismatch", err, test.want)
			}
		})
	}
}

func inspectCanonicalJSON(t *testing.T, contents string) (PuzzleAdapter, string, puzzleInspection) {
	t.Helper()
	path := writeCanonicalJSON(t, contents)
	adapter := NewCanonicalJSONAdapter(chessrules.Rules{})
	inspection, err := (CollectionImporter{Adapters: []PuzzleAdapter{adapter}}).Inspect(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	return adapter, path, inspection
}

func writeCanonicalJSON(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "puzzles.json")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func decodeCanonicalJSONFile(
	t *testing.T,
	adapter PuzzleAdapter,
	path string,
	inspection puzzleInspection,
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

func decodeOneCanonicalJSONPuzzle(t *testing.T, puzzle string) DecodedRecord {
	t.Helper()
	contents := `{
  "schema":"chess-trainer-puzzles/v1",
  "source":` + canonicalJSONSource + `,
  "puzzles":[` + puzzle + `]
}`
	adapter, path, inspection := inspectCanonicalJSON(t, contents)
	records := decodeCanonicalJSONFile(t, adapter, path, inspection)
	if len(records) != 1 {
		t.Fatalf("records = %d, want 1", len(records))
	}
	return records[0]
}

func requireCanonicalPuzzle(t *testing.T, record DecodedRecord) TrainingPuzzle {
	t.Helper()
	if record.Puzzle == nil || record.Rejection != nil {
		if record.Rejection != nil {
			t.Fatalf("record rejection = %+v, want puzzle", *record.Rejection)
		}
		t.Fatalf("record = %+v, want puzzle", record)
	}
	return *record.Puzzle
}
