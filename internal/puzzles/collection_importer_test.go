package puzzles

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"chess-trainer/internal/chessrules"
)

type fakePuzzleAdapter struct {
	format     ImportFormat
	signature  string
	inspection ImportInspection
	inspected  *[]string
	decoder    func() PuzzleDecoder
	preInspect func()
	inspectErr error
}

func (a fakePuzzleAdapter) Format() ImportFormat { return a.format }

func (a fakePuzzleAdapter) Inspect(_ context.Context, path string) (ImportInspection, bool, error) {
	if a.inspected != nil {
		*a.inspected = append(*a.inspected, path)
	}
	if a.preInspect != nil {
		a.preInspect()
	}
	if a.inspectErr != nil {
		return ImportInspection{}, false, a.inspectErr
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return ImportInspection{}, false, err
	}
	if string(contents) != a.signature {
		return ImportInspection{}, false, nil
	}
	inspection := a.inspection
	inspection.Format = a.format
	if inspection.SourceIDOrigin == "" {
		inspection.SourceIDOrigin = SourceIDPath
	}
	return inspection, true, nil
}

func (a fakePuzzleAdapter) NewDecoder(io.Reader, ImportInspection) (PuzzleDecoder, error) {
	if a.decoder == nil {
		return nil, errors.New("fake decoder is not configured")
	}
	return a.decoder(), nil
}

type fakePuzzleDecoder struct {
	records        []DecodedRecord
	next           int
	terminal       error
	beforeTerminal func()
	onClose        func()
	closed         int
}

func (d *fakePuzzleDecoder) Next(ctx context.Context) (DecodedRecord, error) {
	if err := ctx.Err(); err != nil {
		return DecodedRecord{}, err
	}
	if d.next < len(d.records) {
		record := d.records[d.next]
		d.next++
		return record, nil
	}
	if d.beforeTerminal != nil {
		d.beforeTerminal()
		d.beforeTerminal = nil
	}
	if d.terminal != nil {
		err := d.terminal
		d.terminal = nil
		return DecodedRecord{}, err
	}
	return DecodedRecord{}, io.EOF
}

func (d *fakePuzzleDecoder) Close() error {
	d.closed++
	if d.onClose != nil {
		d.onClose()
	}
	return nil
}

type fakeInspectionReader struct {
	CatalogReader
	summaries []SourceSummary
	err       error
}

func (r fakeInspectionReader) ActiveSourceSummaries(context.Context) ([]SourceSummary, error) {
	return r.summaries, r.err
}

func TestCollectionImporterInspectUsesContentAndPathFallback(t *testing.T) {
	directory := t.TempDir()
	target := filepath.Join(directory, "misleading.pgn")
	if err := os.WriteFile(target, []byte("linear-signature"), 0o600); err != nil {
		t.Fatal(err)
	}
	symlink := filepath.Join(directory, "selected.pgn")
	if err := os.Symlink(target, symlink); err != nil {
		t.Fatal(err)
	}
	var pgnPaths, linearPaths []string
	importer := CollectionImporter{Adapters: []PuzzleAdapter{
		fakePuzzleAdapter{
			format: FormatTacticalPGN, signature: "pgn-signature", inspected: &pgnPaths,
		},
		fakePuzzleAdapter{
			format: FormatLinearFENUCI, signature: "linear-signature", inspected: &linearPaths,
		},
	}}

	got, err := importer.Inspect(context.Background(), symlink)
	if err != nil {
		t.Fatal(err)
	}
	wantPath, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}
	wantPath, err = filepath.Abs(wantPath)
	if err != nil {
		t.Fatal(err)
	}
	if got.Format != FormatLinearFENUCI || got.Path != wantPath || got.Filename != filepath.Base(wantPath) ||
		got.SourceID != got.Path || got.SourceIDOrigin != SourceIDPath {
		t.Fatalf("inspection = %+v", got)
	}
	if !reflect.DeepEqual(pgnPaths, []string{wantPath}) || !reflect.DeepEqual(linearPaths, []string{wantPath}) {
		t.Fatalf("adapter paths = pgn %q, linear %q, want %q", pgnPaths, linearPaths, wantPath)
	}
}

func TestCollectionImporterInspectRejectsNoContentMatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unsupported.pgn")
	if err := os.WriteFile(path, []byte("unknown"), 0o600); err != nil {
		t.Fatal(err)
	}
	importer := CollectionImporter{Adapters: []PuzzleAdapter{
		fakePuzzleAdapter{format: FormatTacticalPGN, signature: "pgn-signature"},
	}}

	if _, err := importer.Inspect(context.Background(), path); err == nil {
		t.Fatal("Inspect() unexpectedly selected an adapter from the extension alone")
	}
}

func TestCollectionImporterInspectUsesExtensionOnlyToNarrowContentMatches(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ambiguous.pgn")
	if err := os.WriteFile(path, []byte("shared-signature"), 0o600); err != nil {
		t.Fatal(err)
	}
	importer := CollectionImporter{Adapters: []PuzzleAdapter{
		fakePuzzleAdapter{format: FormatLinearFENUCI, signature: "shared-signature"},
		fakePuzzleAdapter{format: FormatTacticalPGN, signature: "shared-signature"},
	}}

	got, err := importer.Inspect(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Format != FormatTacticalPGN {
		t.Fatalf("format = %q, want %q", got.Format, FormatTacticalPGN)
	}
}

func TestCollectionImporterInspectUniqueContentMatchWinsOverOtherProbeErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "collection.data")
	if err := os.WriteFile(path, []byte("linear-signature"), 0o600); err != nil {
		t.Fatal(err)
	}
	importer := CollectionImporter{Adapters: []PuzzleAdapter{
		fakePuzzleAdapter{format: FormatCanonicalJSON, inspectErr: errors.New("not supported JSON")},
		fakePuzzleAdapter{format: FormatLinearFENUCI, signature: "linear-signature"},
	}}

	got, err := importer.Inspect(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Format != FormatLinearFENUCI {
		t.Fatalf("format = %q, want %q", got.Format, FormatLinearFENUCI)
	}
}

func TestCollectionImporterInspectDoesNotHideCancellationBehindContentMatch(t *testing.T) {
	path := filepath.Join(t.TempDir(), "collection.data")
	if err := os.WriteFile(path, []byte("linear-signature"), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	importer := CollectionImporter{Adapters: []PuzzleAdapter{
		fakePuzzleAdapter{format: FormatLinearFENUCI, signature: "linear-signature"},
		fakePuzzleAdapter{
			format: FormatCanonicalJSON, preInspect: cancel, inspectErr: context.Canceled,
		},
	}}

	_, err := importer.Inspect(ctx, path)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Inspect() error = %v, want context.Canceled", err)
	}
}

func TestCollectionImporterInspectRejectsAmbiguousContentWithoutExtensionAgreement(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ambiguous.data")
	if err := os.WriteFile(path, []byte("shared-signature"), 0o600); err != nil {
		t.Fatal(err)
	}
	importer := CollectionImporter{Adapters: []PuzzleAdapter{
		fakePuzzleAdapter{format: FormatLinearFENUCI, signature: "shared-signature"},
		fakePuzzleAdapter{format: FormatTacticalPGN, signature: "shared-signature"},
	}}

	_, err := importer.Inspect(context.Background(), path)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("Inspect() error = %v, want ambiguity", err)
	}
}

func TestCollectionImporterInspectMarksExistingSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "collection.json")
	if err := os.WriteFile(path, []byte("json-signature"), 0o600); err != nil {
		t.Fatal(err)
	}
	importer := CollectionImporter{
		Reader: fakeInspectionReader{summaries: []SourceSummary{{SourceID: "club"}}},
		Adapters: []PuzzleAdapter{fakePuzzleAdapter{
			format:    FormatCanonicalJSON,
			signature: "json-signature",
			inspection: ImportInspection{
				SourceID: "club", SourceIDOrigin: SourceIDEmbedded,
			},
		}},
	}

	got, err := importer.Inspect(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if !got.ReplacesExisting || got.SourceID != "club" || got.SourceIDOrigin != SourceIDEmbedded {
		t.Fatalf("inspection = %+v", got)
	}
}

type collectionCaptureGeneration struct {
	puzzles            []TrainingPuzzle
	rejections         []Rejection
	events             []string
	checksum           string
	sealCalls          int
	activateCalls      int
	abandonCalls       int
	abandonContextErr  error
	abandonHasDeadline bool
	report             ImportReport
}

func (g *collectionCaptureGeneration) Add(_ context.Context, puzzle TrainingPuzzle) error {
	g.events = append(g.events, "add")
	g.puzzles = append(g.puzzles, puzzle)
	return nil
}

func (g *collectionCaptureGeneration) Reject(rejection Rejection) {
	g.events = append(g.events, "reject")
	g.rejections = append(g.rejections, rejection)
}

func (g *collectionCaptureGeneration) Seal(_ context.Context, checksum string) (ImportReport, error) {
	g.events = append(g.events, "seal")
	g.sealCalls++
	g.checksum = checksum
	return g.report, nil
}

func (g *collectionCaptureGeneration) Activate(context.Context) error {
	g.events = append(g.events, "activate")
	g.activateCalls++
	return nil
}

func (g *collectionCaptureGeneration) Abandon(ctx context.Context) error {
	g.events = append(g.events, "abandon")
	g.abandonCalls++
	g.abandonContextErr = ctx.Err()
	_, g.abandonHasDeadline = ctx.Deadline()
	return nil
}

type collectionCaptureCatalog struct {
	generation *collectionCaptureGeneration
	source     Source
	beginCalls int
}

func (c *collectionCaptureCatalog) BeginImport(_ context.Context, source Source) (GenerationImport, error) {
	c.beginCalls++
	c.source = source
	return c.generation, nil
}

func newCollectionRunner(
	t *testing.T,
	contents string,
	decoder func() PuzzleDecoder,
) (CollectionImporter, string, *collectionCaptureGeneration) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "collection.txt")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	path = resolvedPath
	generation := &collectionCaptureGeneration{report: ImportReport{Accepted: 1, Rejected: 1}}
	catalog := &collectionCaptureCatalog{generation: generation}
	return CollectionImporter{
		Catalog:          catalog,
		CatalogDirectory: t.TempDir(),
		AvailableBytes:   func(string) (uint64, error) { return math.MaxUint64, nil },
		Adapters: []PuzzleAdapter{fakePuzzleAdapter{
			format: FormatLinearFENUCI, signature: contents, decoder: decoder,
		}},
	}, path, generation
}

func TestCollectionImporterImportFormatSealsChecksumAndActivatesInOrder(t *testing.T) {
	puzzle := TrainingPuzzle{Occurrence: PuzzleOccurrence{ExternalID: "one"}}
	rejection := Rejection{Ordinal: 2, Reason: "bad row"}
	var decoder *fakePuzzleDecoder
	importer, path, generation := newCollectionRunner(t, "exact raw file bytes", func() PuzzleDecoder {
		decoder = &fakePuzzleDecoder{records: []DecodedRecord{
			{Puzzle: &puzzle},
			{Rejection: &rejection},
		}}
		return decoder
	})
	var progress []Progress

	report, err := importer.ImportFormat(context.Background(), FormatLinearFENUCI, path, path, func(got Progress) {
		progress = append(progress, got)
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(report, generation.report) {
		t.Fatalf("report = %+v, want %+v", report, generation.report)
	}
	wantChecksum := sha256.Sum256([]byte("exact raw file bytes"))
	if generation.checksum != hex.EncodeToString(wantChecksum[:]) {
		t.Fatalf("checksum = %q, want %q", generation.checksum, hex.EncodeToString(wantChecksum[:]))
	}
	if generation.sealCalls != 1 || generation.activateCalls != 1 || generation.abandonCalls != 0 {
		t.Fatalf("lifecycle calls = seal %d, activate %d, abandon %d", generation.sealCalls, generation.activateCalls, generation.abandonCalls)
	}
	if !reflect.DeepEqual(generation.events, []string{"add", "reject", "seal", "activate"}) {
		t.Fatalf("events = %q", generation.events)
	}
	if !reflect.DeepEqual(generation.rejections, []Rejection{rejection}) {
		t.Fatalf("rejections = %+v, want %+v", generation.rejections, rejection)
	}
	if decoder.closed != 1 {
		t.Fatalf("decoder Close calls = %d, want 1", decoder.closed)
	}
	wantPhases := []ImportPhase{ImportDetecting, ImportParsing, ImportSealing, ImportActivating}
	var phases []ImportPhase
	for _, snapshot := range progress {
		if len(phases) == 0 || phases[len(phases)-1] != snapshot.Phase {
			phases = append(phases, snapshot.Phase)
		}
	}
	if !reflect.DeepEqual(phases, wantPhases) {
		t.Fatalf("progress phases = %q, want %q; snapshots = %+v", phases, wantPhases, progress)
	}
	last := progress[len(progress)-1]
	if last.BytesRead != int64(len("exact raw file bytes")) || last.TotalBytes != int64(len("exact raw file bytes")) {
		t.Fatalf("final progress = %+v", last)
	}
}

func TestCollectionImporterImportFormatAbandonsWhenNoValidPuzzles(t *testing.T) {
	rejection := Rejection{Ordinal: 1, Reason: "invalid"}
	importer, path, generation := newCollectionRunner(t, "only rejected", func() PuzzleDecoder {
		return &fakePuzzleDecoder{records: []DecodedRecord{{Rejection: &rejection}}}
	})

	_, err := importer.ImportFormat(context.Background(), FormatLinearFENUCI, path, path, nil)
	if !errors.Is(err, ErrNoValidPuzzles) {
		t.Fatalf("ImportFormat() error = %v, want ErrNoValidPuzzles", err)
	}
	if generation.abandonCalls != 1 || generation.sealCalls != 0 || generation.activateCalls != 0 {
		t.Fatalf("lifecycle calls = abandon %d, seal %d, activate %d", generation.abandonCalls, generation.sealCalls, generation.activateCalls)
	}
}

func TestCollectionImporterImportFormatAbandonsAfterLateFatalError(t *testing.T) {
	puzzle := TrainingPuzzle{Occurrence: PuzzleOccurrence{ExternalID: "one"}}
	importer, path, generation := newCollectionRunner(t, "truncated", func() PuzzleDecoder {
		return &fakePuzzleDecoder{
			records:  []DecodedRecord{{Puzzle: &puzzle}},
			terminal: errors.New("truncated source"),
		}
	})

	_, err := importer.ImportFormat(context.Background(), FormatLinearFENUCI, path, path, nil)
	if err == nil || !strings.Contains(err.Error(), "truncated source") {
		t.Fatalf("ImportFormat() error = %v", err)
	}
	if generation.abandonCalls != 1 || generation.sealCalls != 0 || generation.activateCalls != 0 {
		t.Fatalf("lifecycle calls = abandon %d, seal %d, activate %d", generation.abandonCalls, generation.sealCalls, generation.activateCalls)
	}
}

func TestCollectionImporterCanonicalJSONAbandonsLateSyntaxError(t *testing.T) {
	valid := `{
  "schema":"chess-trainer-puzzles/v1",
  "source":` + canonicalJSONSource + `,
  "puzzles":[` + canonicalJSONWhitePuzzle + `,` + canonicalJSONWhitePuzzle + `]
}`
	broken := `{
  "schema":"chess-trainer-puzzles/v1",
  "source":` + canonicalJSONSource + `,
  "puzzles":[` + canonicalJSONWhitePuzzle + `,{"id":"broken","solver":]
}`
	path := filepath.Join(t.TempDir(), "late-broken.json")
	if err := os.WriteFile(path, []byte(valid), 0o600); err != nil {
		t.Fatal(err)
	}
	generation := &collectionCaptureGeneration{report: ImportReport{Accepted: 2}}
	catalog := &collectionCaptureCatalog{generation: generation}
	mutated := false
	importer := CollectionImporter{
		Catalog:          catalog,
		CatalogDirectory: t.TempDir(),
		AvailableBytes: func(string) (uint64, error) {
			mutated = true
			return math.MaxUint64, os.WriteFile(path, []byte(broken), 0o600)
		},
		Adapters: []PuzzleAdapter{NewCanonicalJSONAdapter(chessrules.Rules{})},
	}

	_, err := importer.ImportFormat(
		context.Background(), FormatCanonicalJSON, "club-json", path, nil,
	)
	if err == nil {
		t.Fatal("ImportFormat() error = nil, want fatal JSON syntax error")
	}
	if !mutated || catalog.beginCalls != 1 {
		t.Fatalf("mutation/begin calls = %v/%d, want true/1", mutated, catalog.beginCalls)
	}
	if len(generation.puzzles) != 1 {
		t.Fatalf("staged puzzles = %d, want first record staged before fatal error", len(generation.puzzles))
	}
	if generation.abandonCalls != 1 || generation.sealCalls != 0 || generation.activateCalls != 0 {
		t.Fatalf("lifecycle = abandon %d, seal %d, activate %d", generation.abandonCalls, generation.sealCalls, generation.activateCalls)
	}
}

func TestCollectionImporterImportFormatCancellationUsesCleanupContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	importer, path, generation := newCollectionRunner(t, "cancelled", func() PuzzleDecoder {
		return &fakePuzzleDecoder{
			terminal:       context.Canceled,
			beforeTerminal: cancel,
		}
	})

	_, err := importer.ImportFormat(ctx, FormatLinearFENUCI, path, path, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ImportFormat() error = %v, want context.Canceled", err)
	}
	if generation.abandonCalls != 1 || generation.sealCalls != 0 || generation.activateCalls != 0 {
		t.Fatalf("lifecycle calls = abandon %d, seal %d, activate %d", generation.abandonCalls, generation.sealCalls, generation.activateCalls)
	}
	if generation.abandonContextErr != nil || !generation.abandonHasDeadline {
		t.Fatalf("abandon context error/deadline = %v/%v", generation.abandonContextErr, generation.abandonHasDeadline)
	}
}

func TestCollectionImporterImportFormatRejectsStaleInspectionIdentity(t *testing.T) {
	importer, path, generation := newCollectionRunner(t, "identity", func() PuzzleDecoder {
		return &fakePuzzleDecoder{}
	})

	_, err := importer.ImportFormat(context.Background(), FormatLinearFENUCI, "stale-source", path, nil)
	if err == nil || !strings.Contains(err.Error(), "source ID") {
		t.Fatalf("ImportFormat() error = %v, want source ID mismatch", err)
	}
	if generation.abandonCalls != 0 || generation.sealCalls != 0 || generation.activateCalls != 0 {
		t.Fatalf("stale inspection entered generation lifecycle: %+v", generation)
	}
}

func TestCollectionImporterImportFormatReinspectionUsesRegistrySelection(t *testing.T) {
	path := filepath.Join(t.TempDir(), "collection.pgn")
	if err := os.WriteFile(path, []byte("shared-signature"), 0o600); err != nil {
		t.Fatal(err)
	}
	normalizedPath, err := normalizeImportPath(path)
	if err != nil {
		t.Fatal(err)
	}
	puzzle := TrainingPuzzle{Occurrence: PuzzleOccurrence{ExternalID: "one"}}
	generation := &collectionCaptureGeneration{report: ImportReport{Accepted: 1}}
	catalog := &collectionCaptureCatalog{generation: generation}
	importer := CollectionImporter{
		Catalog:          catalog,
		CatalogDirectory: t.TempDir(),
		AvailableBytes:   func(string) (uint64, error) { return math.MaxUint64, nil },
		Adapters: []PuzzleAdapter{
			fakePuzzleAdapter{
				format: FormatLinearFENUCI, signature: "shared-signature",
				decoder: func() PuzzleDecoder {
					return &fakePuzzleDecoder{records: []DecodedRecord{{Puzzle: &puzzle}}}
				},
			},
			fakePuzzleAdapter{format: FormatTacticalPGN, signature: "shared-signature"},
		},
	}

	_, err = importer.ImportFormat(
		context.Background(), FormatLinearFENUCI, normalizedPath, normalizedPath, nil,
	)
	if err == nil || !strings.Contains(err.Error(), "format") {
		t.Fatalf("ImportFormat() error = %v, want authoritative format mismatch", err)
	}
	if catalog.beginCalls != 0 {
		t.Fatalf("BeginImport calls = %d, want 0", catalog.beginCalls)
	}
}

func TestCollectionImporterTacticalPGNAbandonsConflictingSourceIdentity(t *testing.T) {
	tests := []struct {
		name     string
		sourceID string
		movetext string
	}{
		{name: "valid conflicting record", sourceID: "other-club", movetext: "1. e4 *"},
		{name: "conflict before illegal movetext", sourceID: "other-club", movetext: "1. e5 *"},
		{name: "conflict before genuine lexer error", sourceID: "other-club", movetext: "1. e *"},
		{name: "explicit empty identity", sourceID: "   ", movetext: "1. e4 *"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			contents := tacticalPGNDirectSolver + `
[Event "Conflicting identity"]
[SourceId "` + test.sourceID + `"]
[FEN "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"]
[White "solver"][Black "?"]
` + "\n" + test.movetext + "\n"
			path := filepath.Join(t.TempDir(), "conflict.pgn")
			if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
				t.Fatal(err)
			}
			generation := &collectionCaptureGeneration{report: ImportReport{Accepted: 1}}
			importer := CollectionImporter{
				Catalog:          &collectionCaptureCatalog{generation: generation},
				CatalogDirectory: t.TempDir(),
				AvailableBytes:   func(string) (uint64, error) { return math.MaxUint64, nil },
				Adapters:         []PuzzleAdapter{NewTacticalPGNAdapter(chessrules.Rules{})},
			}

			_, err := importer.ImportFormat(
				context.Background(),
				FormatTacticalPGN,
				"club-tactics",
				path,
				nil,
			)
			if err == nil || !strings.Contains(err.Error(), "SourceId") {
				t.Errorf("ImportFormat() error = %v, want fatal SourceId conflict", err)
			}
			if generation.activateCalls != 0 || generation.sealCalls != 0 || generation.abandonCalls != 1 {
				t.Errorf("lifecycle = activate %d, seal %d, abandon %d", generation.activateCalls, generation.sealCalls, generation.abandonCalls)
			}
		})
	}
}

func TestCollectionImporterImportFormatCancelsUnreadRawDrain(t *testing.T) {
	const contents = "unread trailing raw source bytes"
	ctx, cancel := context.WithCancel(context.Background())
	puzzle := TrainingPuzzle{Occurrence: PuzzleOccurrence{ExternalID: "one"}}
	importer, path, generation := newCollectionRunner(t, contents, func() PuzzleDecoder {
		return &fakePuzzleDecoder{
			records: []DecodedRecord{{Puzzle: &puzzle}},
			onClose: cancel,
		}
	})
	var progress []Progress

	_, err := importer.ImportFormat(ctx, FormatLinearFENUCI, path, path, func(got Progress) {
		progress = append(progress, got)
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ImportFormat() error = %v, want context.Canceled", err)
	}
	var maximumBytesRead int64
	for _, snapshot := range progress {
		maximumBytesRead = max(maximumBytesRead, snapshot.BytesRead)
	}
	if maximumBytesRead == int64(len(contents)) {
		t.Fatalf("drain consumed all %d raw bytes after cancellation; progress = %+v", len(contents), progress)
	}
	if generation.abandonCalls != 1 || generation.sealCalls != 0 || generation.activateCalls != 0 {
		t.Fatalf("lifecycle calls = abandon %d, seal %d, activate %d", generation.abandonCalls, generation.sealCalls, generation.activateCalls)
	}
}
