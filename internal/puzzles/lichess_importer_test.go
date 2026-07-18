package puzzles

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"chess-trainer/internal/chessrules"

	"github.com/klauspost/compress/zstd"
)

type captureGenerationImport struct {
	puzzles     []TrainingPuzzle
	report      ImportReport
	checksum    string
	abandoned   int
	seals       int
	activations int
}

func (s *captureGenerationImport) Add(_ context.Context, puzzle TrainingPuzzle) error {
	s.puzzles = append(s.puzzles, puzzle)
	return nil
}

func (s *captureGenerationImport) Reject(rejection Rejection) {
	s.report.Rejected++
	s.report.Examples = append(s.report.Examples, rejection)
}

func (s *captureGenerationImport) Seal(_ context.Context, checksum string) (ImportReport, error) {
	s.seals++
	s.checksum = checksum
	s.report.Accepted = int64(len(s.puzzles))
	return s.report, nil
}

func (s *captureGenerationImport) Activate(context.Context) error {
	s.activations++
	return nil
}

func (s *captureGenerationImport) Abandon(context.Context) error {
	s.abandoned++
	return nil
}

type captureCatalog struct {
	generation  GenerationImport
	beginCalled int
}

func (c *captureCatalog) BeginImport(context.Context, Source) (GenerationImport, error) {
	c.beginCalled++
	return c.generation, nil
}

func writeZstandardFixture(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "puzzles.csv.zst")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	writer, err := zstd.NewWriter(file)
	if err != nil {
		file.Close()
		t.Fatal(err)
	}
	if _, err := writer.Write([]byte(contents)); err != nil {
		writer.Close()
		file.Close()
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path
}

func newLichessCollectionImporter(
	catalog CatalogWriter,
	catalogDirectory string,
) CollectionImporter {
	return CollectionImporter{
		Catalog:          catalog,
		Adapters:         []PuzzleAdapter{NewLichessAdapter(chessrules.Rules{})},
		CatalogDirectory: catalogDirectory,
	}
}

func inspectAndImportLichess(
	ctx context.Context,
	importer CollectionImporter,
	path string,
	progress ProgressSink,
) (ImportReport, error) {
	inspection, err := importer.Inspect(ctx, path)
	if err != nil {
		return ImportReport{}, err
	}
	return importer.Import(ctx, inspection, progress)
}

func TestLichessAdapterInspectsCompressedHeaderRegardlessOfFilename(t *testing.T) {
	path := writeZstandardFixture(t, `PuzzleId,FEN,Moves,Rating,RatingDeviation,Popularity,NbPlays,Themes,GameUrl,OpeningTags
`)
	unrelatedName := filepath.Join(filepath.Dir(path), "download.bin")
	if err := os.Rename(path, unrelatedName); err != nil {
		t.Fatal(err)
	}

	adapter := NewLichessAdapter(chessrules.Rules{})
	inspection, matched, err := adapter.Inspect(context.Background(), unrelatedName)
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Fatal("Inspect() did not match a compressed Lichess header")
	}
	if adapter.Format() != FormatLichess {
		t.Fatalf("Format() = %q, want %q", adapter.Format(), FormatLichess)
	}
	if inspection.SourceID != "lichess" || inspection.SourceIDOrigin != SourceIDFixed {
		t.Fatalf("inspection source identity = %q/%q, want lichess/fixed", inspection.SourceID, inspection.SourceIDOrigin)
	}
}

func TestLichessAdapterDoesNotMatchPlainCSV(t *testing.T) {
	path := filepath.Join(t.TempDir(), "looks-like-lichess.csv.zst")
	if err := os.WriteFile(
		path,
		[]byte("PuzzleId,FEN,Moves,Rating,RatingDeviation,Popularity,NbPlays,Themes,GameUrl,OpeningTags\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	_, matched, err := NewLichessAdapter(chessrules.Rules{}).Inspect(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if matched {
		t.Fatal("Inspect() matched plain CSV without zstd magic")
	}
}

func TestLichessAdapterRejectsUnsupportedCompressedContent(t *testing.T) {
	path := writeZstandardFixture(t, "Name,Value\nexample,one\n")

	_, matched, err := NewLichessAdapter(chessrules.Rules{}).Inspect(context.Background(), path)
	if err == nil || !strings.Contains(err.Error(), "unsupported Lichess content") {
		t.Fatalf("Inspect() error = %v, want unsupported Lichess content", err)
	}
	if matched {
		t.Fatal("Inspect() matched an unrelated compressed CSV header")
	}
}

func TestCollectionImporterNormalizesLichessSetupMove(t *testing.T) {
	path := writeZstandardFixture(t, `PuzzleId,FEN,Moves,Rating,RatingDeviation,Popularity,NbPlays,Themes,GameUrl,OpeningTags
mate1,8/5Q1k/6K1/8/8/8/8/8 b - - 0 1,h7h8 f7f8,1200,60,95,200,mate mateIn1,https://lichess.org/example,
bad,not-a-fen,a1a2,1500,60,10,2,short,,
`)
	generation := &captureGenerationImport{}
	catalog := &captureCatalog{generation: generation}
	importer := newLichessCollectionImporter(catalog, filepath.Dir(path))
	var progress []Progress

	report, err := inspectAndImportLichess(context.Background(), importer, path, func(snapshot Progress) {
		progress = append(progress, snapshot)
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Accepted != 1 || report.Rejected != 1 {
		t.Fatalf("report=%+v", report)
	}
	if len(generation.puzzles) != 1 {
		t.Fatalf("captured %d puzzles", len(generation.puzzles))
	}
	puzzle := generation.puzzles[0]
	fields := strings.Fields(puzzle.Core.DisplayedFEN)
	if len(fields) < 2 || fields[1] != "w" {
		t.Fatalf("DisplayedFEN=%q", puzzle.Core.DisplayedFEN)
	}
	if puzzle.Occurrence.PreludeUCI != "h7h8" {
		t.Fatalf("PreludeUCI=%q", puzzle.Occurrence.PreludeUCI)
	}
	if len(puzzle.Core.Solution) != 1 || puzzle.Core.Solution[0].UCI != "f7f8" {
		t.Fatalf("Solution=%+v", puzzle.Core.Solution)
	}
	if puzzle.Occurrence.Rating == nil || *puzzle.Occurrence.Rating != 1200 {
		t.Fatalf("Rating=%v", puzzle.Occurrence.Rating)
	}
	if len(puzzle.Occurrence.Themes) != 2 || puzzle.Occurrence.Themes[0] != "mate" || puzzle.Occurrence.Themes[1] != "mateIn1" {
		t.Fatalf("Themes=%v", puzzle.Occurrence.Themes)
	}
	if len(puzzle.Core.Fingerprint) != 64 {
		t.Fatalf("Fingerprint=%q", puzzle.Core.Fingerprint)
	}
	compressed, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	wantChecksum := sha256.Sum256(compressed)
	if generation.checksum != hex.EncodeToString(wantChecksum[:]) {
		t.Fatalf("checksum=%q, want raw compressed checksum %x", generation.checksum, wantChecksum)
	}
	wantPhases := []ImportPhase{ImportDetecting, ImportParsing, ImportSealing, ImportActivating}
	var phases []ImportPhase
	for _, snapshot := range progress {
		if len(phases) == 0 || phases[len(phases)-1] != snapshot.Phase {
			phases = append(phases, snapshot.Phase)
		}
	}
	if !reflect.DeepEqual(phases, wantPhases) {
		t.Fatalf("progress phases=%q, want %q; snapshots=%+v", phases, wantPhases, progress)
	}
	last := progress[len(progress)-1]
	if last.BytesRead != int64(len(compressed)) || last.TotalBytes != int64(len(compressed)) {
		t.Fatalf("final progress=%+v, want %d raw bytes", last, len(compressed))
	}
}

func TestCollectionImporterRejectsZeroValidLichessPuzzles(t *testing.T) {
	path := writeZstandardFixture(t, `PuzzleId,FEN,Moves,Rating,RatingDeviation,Popularity,NbPlays,Themes,GameUrl,OpeningTags
bad,not-a-fen,a1a2,1500,60,10,2,short,,
`)
	generation := &captureGenerationImport{}
	catalog := &captureCatalog{generation: generation}
	importer := newLichessCollectionImporter(catalog, filepath.Dir(path))

	_, err := inspectAndImportLichess(context.Background(), importer, path, nil)
	if !errors.Is(err, ErrNoValidPuzzles) {
		t.Fatalf("Import() err=%v, want ErrNoValidPuzzles", err)
	}
	if generation.abandoned != 1 || generation.seals != 0 || generation.activations != 0 {
		t.Fatalf("abandoned=%d seals=%d activations=%d", generation.abandoned, generation.seals, generation.activations)
	}
}

func TestCollectionImporterRejectsOverDepthLichessBeforeMoveValidation(t *testing.T) {
	moves := "e2e4 " + strings.TrimSpace(strings.Repeat("not-a-move ", maxSolutionDepth+1))
	path := writeZstandardFixture(t, `PuzzleId,FEN,Moves,Rating,RatingDeviation,Popularity,NbPlays,Themes,GameUrl,OpeningTags
over-depth,`+standardStartingFEN+`,`+moves+`,1200,60,95,200,,,`+"\n")
	generation := &captureGenerationImport{}
	importer := newLichessCollectionImporter(
		&captureCatalog{generation: generation}, filepath.Dir(path),
	)

	_, err := inspectAndImportLichess(context.Background(), importer, path, nil)
	if !errors.Is(err, ErrNoValidPuzzles) {
		t.Fatalf("Import() error = %v, want ErrNoValidPuzzles", err)
	}
	if len(generation.report.Examples) != 1 || !strings.Contains(
		generation.report.Examples[0].Reason,
		"solution depth 257 exceeds maximum of 256",
	) {
		t.Fatalf("rejections = %+v, want bounded depth error before invalid-move handling", generation.report.Examples)
	}
}

func TestCollectionImporterCancellationAbortsLichessStaging(t *testing.T) {
	var fixture strings.Builder
	fixture.WriteString("PuzzleId,FEN,Moves,Rating,RatingDeviation,Popularity,NbPlays,Themes,GameUrl,OpeningTags\n")
	for index := 0; index < 20_000; index++ {
		fmt.Fprintf(
			&fixture,
			"mate%d,8/5Q1k/6K1/8/8/8/8/8 b - - 0 1,h7h8 f7f8,1200,60,95,200,mate mateIn1,https://lichess.org/example,\n",
			index,
		)
	}
	path := writeZstandardFixture(t, fixture.String())
	generation := &captureGenerationImport{}
	catalog := &captureCatalog{generation: generation}
	importer := newLichessCollectionImporter(catalog, filepath.Dir(path))
	ctx, cancel := context.WithCancel(context.Background())

	_, err := inspectAndImportLichess(ctx, importer, path, func(progress Progress) {
		if progress.RowsRead >= 10_000 {
			cancel()
		}
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Import() err=%v, want context.Canceled", err)
	}
	if generation.abandoned != 1 || generation.seals != 0 || generation.activations != 0 {
		t.Fatalf("abandoned=%d seals=%d activations=%d", generation.abandoned, generation.seals, generation.activations)
	}
}

func TestCollectionImporterRejectsMissingLichessHeader(t *testing.T) {
	path := writeZstandardFixture(t, "PuzzleId,Moves\nmate1,h7h8 f7f8\n")
	generation := &captureGenerationImport{}
	catalog := &captureCatalog{generation: generation}
	importer := newLichessCollectionImporter(catalog, filepath.Dir(path))

	_, err := inspectAndImportLichess(context.Background(), importer, path, nil)
	if err == nil || !strings.Contains(err.Error(), "missing required Lichess column") {
		t.Fatalf("Import() err=%v", err)
	}
	if catalog.beginCalled != 0 || generation.abandoned != 0 || generation.seals != 0 || generation.activations != 0 {
		t.Fatalf(
			"begin=%d abandoned=%d seals=%d activations=%d",
			catalog.beginCalled,
			generation.abandoned,
			generation.seals,
			generation.activations,
		)
	}
}

func TestCollectionImporterRejectsTruncatedLichessZstandard(t *testing.T) {
	path := writeZstandardFixture(t, `PuzzleId,FEN,Moves,Rating,RatingDeviation,Popularity,NbPlays,Themes,GameUrl,OpeningTags
mate1,8/5Q1k/6K1/8/8/8/8/8 b - - 0 1,h7h8 f7f8,1200,60,95,200,mate mateIn1,https://lichess.org/example,
`)
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, contents[:len(contents)/2], 0o600); err != nil {
		t.Fatal(err)
	}
	generation := &captureGenerationImport{}
	catalog := &captureCatalog{generation: generation}
	importer := newLichessCollectionImporter(catalog, filepath.Dir(path))

	if _, err := inspectAndImportLichess(context.Background(), importer, path, nil); err == nil {
		t.Fatal("Import() unexpectedly succeeded")
	}
	if catalog.beginCalled != 0 || generation.abandoned != 0 || generation.seals != 0 || generation.activations != 0 {
		t.Fatalf(
			"begin=%d abandoned=%d seals=%d activations=%d",
			catalog.beginCalled,
			generation.abandoned,
			generation.seals,
			generation.activations,
		)
	}
}

func TestCollectionImporterChecksDiskBeforeLichessStaging(t *testing.T) {
	path := writeZstandardFixture(t, `PuzzleId,FEN,Moves,Rating,RatingDeviation,Popularity,NbPlays,Themes,GameUrl,OpeningTags
`)
	generation := &captureGenerationImport{}
	catalog := &captureCatalog{generation: generation}
	importer := newLichessCollectionImporter(catalog, filepath.Dir(path))
	importer.AvailableBytes = func(string) (uint64, error) { return 0, nil }

	if _, err := inspectAndImportLichess(context.Background(), importer, path, nil); err == nil {
		t.Fatal("Import() unexpectedly succeeded")
	}
	if catalog.beginCalled != 0 {
		t.Fatalf("BeginImport called %d times", catalog.beginCalled)
	}
}

type discardGenerationImport struct {
	accepted int64
}

func (s *discardGenerationImport) Add(context.Context, TrainingPuzzle) error {
	s.accepted++
	return nil
}

func (*discardGenerationImport) Reject(Rejection) {}
func (s *discardGenerationImport) Seal(context.Context, string) (ImportReport, error) {
	return ImportReport{Accepted: s.accepted}, nil
}
func (*discardGenerationImport) Activate(context.Context) error { return nil }
func (*discardGenerationImport) Abandon(context.Context) error  { return nil }

func writeGeneratedLichessFixture(t *testing.T, count int) string {
	t.Helper()
	var fixture strings.Builder
	fixture.WriteString("PuzzleId,FEN,Moves,Rating,RatingDeviation,Popularity,NbPlays,Themes,GameUrl,OpeningTags\n")
	for index := 0; index < count; index++ {
		fmt.Fprintf(
			&fixture,
			"mate%d,8/5Q1k/6K1/8/8/8/8/8 b - - 0 1,h7h8 f7f8,1200,60,95,200,mate mateIn1,https://lichess.org/example,\n",
			index,
		)
	}
	return writeZstandardFixture(t, fixture.String())
}

func TestCollectionImporterLichessStreamingAllocations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 100,000-row streaming allocation probe in short mode")
	}
	path := writeGeneratedLichessFixture(t, 100_000)
	generation := &discardGenerationImport{}
	catalog := &captureCatalog{generation: generation}
	importer := newLichessCollectionImporter(catalog, filepath.Dir(path))

	runtime.GC()
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	baseline := memory.HeapAlloc
	var maximum atomic.Uint64
	maximum.Store(baseline)
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				runtime.ReadMemStats(&memory)
				for current := maximum.Load(); memory.HeapAlloc > current; current = maximum.Load() {
					if maximum.CompareAndSwap(current, memory.HeapAlloc) {
						break
					}
				}
			case <-done:
				return
			}
		}
	}()

	report, err := inspectAndImportLichess(context.Background(), importer, path, nil)
	close(done)
	if err != nil {
		t.Fatal(err)
	}
	if report.Accepted != 100_000 {
		t.Fatalf("Accepted=%d", report.Accepted)
	}
	const maximumGrowth = 128 * 1024 * 1024
	if growth := maximum.Load() - baseline; growth >= maximumGrowth {
		t.Fatalf("peak heap growth=%d bytes, want below %d", growth, maximumGrowth)
	}
}
