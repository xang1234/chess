package puzzles

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"chess-trainer/internal/domain"

	"github.com/klauspost/compress/zstd"
)

type captureStagedImport struct {
	puzzles  []domain.Puzzle
	report   ImportReport
	checksum string
	aborted  int
	commits  int
}

func (s *captureStagedImport) Add(_ context.Context, puzzle domain.Puzzle) error {
	s.puzzles = append(s.puzzles, puzzle)
	return nil
}

func (s *captureStagedImport) Reject(rejection Rejection) {
	s.report.Rejected++
	s.report.Examples = append(s.report.Examples, rejection)
}

func (s *captureStagedImport) SetChecksum(checksum string) {
	s.checksum = checksum
}

func (s *captureStagedImport) Commit(context.Context) (ImportReport, error) {
	s.commits++
	s.report.Accepted = int64(len(s.puzzles))
	return s.report, nil
}

func (s *captureStagedImport) Abort(context.Context) error {
	s.aborted++
	return nil
}

type captureCatalog struct {
	staged      StagedImport
	beginCalled int
}

func (c *captureCatalog) BeginImport(context.Context, Source) (StagedImport, error) {
	c.beginCalled++
	return c.staged, nil
}

func (*captureCatalog) Get(context.Context, string) (domain.Puzzle, error) {
	return domain.Puzzle{}, nil
}

func (*captureCatalog) RatedCandidates(context.Context, int, int, []string, int) ([]domain.Puzzle, error) {
	return nil, nil
}

func (*captureCatalog) FreePracticeCandidates(context.Context, string, *int, *int, []string, int) ([]domain.Puzzle, error) {
	return nil, nil
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

func TestLichessImporterNormalizesSetupMove(t *testing.T) {
	path := writeZstandardFixture(t, `PuzzleId,FEN,Moves,Rating,RatingDeviation,Popularity,NbPlays,Themes,GameUrl,OpeningTags
mate1,8/5Q1k/6K1/8/8/8/8/8 b - - 0 1,h7h8 f7f8,1200,60,95,200,mate mateIn1,https://lichess.org/example,
bad,not-a-fen,a1a2,1500,60,10,2,short,,
`)
	staged := &captureStagedImport{}
	catalog := &captureCatalog{staged: staged}
	importer := LichessImporter{Catalog: catalog}

	report, err := importer.Import(context.Background(), "lichess", path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if report.Accepted != 1 || report.Rejected != 1 {
		t.Fatalf("report=%+v", report)
	}
	if len(staged.puzzles) != 1 {
		t.Fatalf("captured %d puzzles", len(staged.puzzles))
	}
	puzzle := staged.puzzles[0]
	fields := strings.Fields(puzzle.DisplayedFEN)
	if len(fields) < 2 || fields[1] != "w" {
		t.Fatalf("DisplayedFEN=%q", puzzle.DisplayedFEN)
	}
	if puzzle.PreludeUCI != "h7h8" {
		t.Fatalf("PreludeUCI=%q", puzzle.PreludeUCI)
	}
	if len(puzzle.Solution) != 1 || puzzle.Solution[0].UCI != "f7f8" {
		t.Fatalf("Solution=%+v", puzzle.Solution)
	}
	if puzzle.Rating == nil || *puzzle.Rating != 1200 {
		t.Fatalf("Rating=%v", puzzle.Rating)
	}
	if len(puzzle.Themes) != 2 || puzzle.Themes[0] != "mate" || puzzle.Themes[1] != "mateIn1" {
		t.Fatalf("Themes=%v", puzzle.Themes)
	}
	if len(puzzle.Fingerprint) != 64 {
		t.Fatalf("Fingerprint=%q", puzzle.Fingerprint)
	}
	if len(staged.checksum) != 64 {
		t.Fatalf("checksum=%q", staged.checksum)
	}
}

func TestLichessImporterCancellationAbortsStaging(t *testing.T) {
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
	staged := &captureStagedImport{}
	catalog := &captureCatalog{staged: staged}
	importer := LichessImporter{Catalog: catalog}
	ctx, cancel := context.WithCancel(context.Background())

	_, err := importer.Import(ctx, "lichess", path, func(progress Progress) {
		if progress.RowsRead >= 10_000 {
			cancel()
		}
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Import() err=%v, want context.Canceled", err)
	}
	if staged.aborted != 1 || staged.commits != 0 {
		t.Fatalf("aborted=%d commits=%d", staged.aborted, staged.commits)
	}
}

func TestLichessImporterRejectsMissingHeader(t *testing.T) {
	path := writeZstandardFixture(t, "PuzzleId,Moves\nmate1,h7h8 f7f8\n")
	staged := &captureStagedImport{}
	catalog := &captureCatalog{staged: staged}
	importer := LichessImporter{Catalog: catalog}

	_, err := importer.Import(context.Background(), "lichess", path, nil)
	if err == nil || !strings.Contains(err.Error(), "missing required Lichess column") {
		t.Fatalf("Import() err=%v", err)
	}
	if staged.aborted != 1 || staged.commits != 0 {
		t.Fatalf("aborted=%d commits=%d", staged.aborted, staged.commits)
	}
}

func TestLichessImporterRejectsTruncatedZstandard(t *testing.T) {
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
	staged := &captureStagedImport{}
	catalog := &captureCatalog{staged: staged}
	importer := LichessImporter{Catalog: catalog}

	if _, err := importer.Import(context.Background(), "lichess", path, nil); err == nil {
		t.Fatal("Import() unexpectedly succeeded")
	}
	if staged.aborted != 1 || staged.commits != 0 {
		t.Fatalf("aborted=%d commits=%d", staged.aborted, staged.commits)
	}
}

func TestLichessImporterChecksDiskBeforeStaging(t *testing.T) {
	path := writeZstandardFixture(t, "unused")
	staged := &captureStagedImport{}
	catalog := &captureCatalog{staged: staged}
	importer := LichessImporter{
		Catalog: catalog,
		AvailableBytes: func(string) (uint64, error) {
			return 0, nil
		},
	}

	if _, err := importer.Import(context.Background(), "lichess", path, nil); err == nil {
		t.Fatal("Import() unexpectedly succeeded")
	}
	if catalog.beginCalled != 0 {
		t.Fatalf("BeginImport called %d times", catalog.beginCalled)
	}
}

type discardStagedImport struct {
	accepted int64
}

func (s *discardStagedImport) Add(context.Context, domain.Puzzle) error {
	s.accepted++
	return nil
}

func (*discardStagedImport) Reject(Rejection)   {}
func (*discardStagedImport) SetChecksum(string) {}
func (s *discardStagedImport) Commit(context.Context) (ImportReport, error) {
	return ImportReport{Accepted: s.accepted}, nil
}
func (*discardStagedImport) Abort(context.Context) error { return nil }

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

func TestLichessImporterStreamingAllocations(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 100,000-row streaming allocation probe in short mode")
	}
	path := writeGeneratedLichessFixture(t, 100_000)
	staged := &discardStagedImport{}
	catalog := &captureCatalog{staged: staged}
	importer := LichessImporter{Catalog: catalog}

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

	report, err := importer.Import(context.Background(), "lichess", path, nil)
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
