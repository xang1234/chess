package puzzles

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/domain"

	"github.com/klauspost/compress/zstd"
)

const lichessGenerationCSV = `PuzzleId,FEN,Moves,Rating,RatingDeviation,Popularity,NbPlays,Themes,GameUrl,OpeningTags
mate1,8/5Q1k/6K1/8/8/8/8/8 b - - 0 1,h7h8 f7f8,1200,60,95,200,"mateIn1 mate mate",https://lichess.org/example,"kingsideAttack sicilian"
`

type lichessGenerationResult struct {
	report ImportReport
	err    error
}

type lichessGenerationCatalog struct {
	importing    *lichessGenerationImport
	beginCalls   int
	beginSource  Source
	beginErr     error
	events       []string
	head         string
	cleanupCalls int
}

func (c *lichessGenerationCatalog) BeginImport(
	_ context.Context,
	source Source,
) (GenerationImport, error) {
	c.beginCalls++
	c.beginSource = source
	c.events = append(c.events, "begin")
	if c.beginErr != nil {
		return nil, c.beginErr
	}
	c.importing.catalog = c
	c.importing.state = "building"
	return c.importing, nil
}

// CleanupBatch deliberately models physical deletion separately from Abandon.
// The Lichess adapter only owns the generation lifecycle and must never call it.
func (c *lichessGenerationCatalog) CleanupBatch(context.Context, int) (bool, error) {
	c.cleanupCalls++
	c.importing.physicalRows = 0
	return false, nil
}

type lichessGenerationImport struct {
	catalog               *lichessGenerationCatalog
	puzzles               []TrainingPuzzle
	rejections            []Rejection
	report                ImportReport
	checksum              string
	state                 string
	physicalRows          int
	addErr                error
	sealErr               error
	activateErr           error
	abandonErr            error
	waitForCancellation   bool
	addEntered            chan struct{}
	onAdd                 func()
	returnAddContextError bool
	sealCalls             int
	activateCalls         int
	abandonCalls          int
	abandonContextErr     error
	abandonHasDeadline    bool
}

func (i *lichessGenerationImport) Add(ctx context.Context, puzzle TrainingPuzzle) error {
	i.catalog.events = append(i.catalog.events, "add")
	i.puzzles = append(i.puzzles, puzzle)
	i.physicalRows++
	if i.addEntered != nil {
		select {
		case i.addEntered <- struct{}{}:
		default:
		}
	}
	if i.onAdd != nil {
		i.onAdd()
	}
	if i.waitForCancellation {
		<-ctx.Done()
		return ctx.Err()
	}
	if i.returnAddContextError {
		return ctx.Err()
	}
	return i.addErr
}

func (i *lichessGenerationImport) Reject(rejection Rejection) {
	i.rejections = append(i.rejections, rejection)
}

func (i *lichessGenerationImport) Seal(_ context.Context, checksum string) (ImportReport, error) {
	i.catalog.events = append(i.catalog.events, "seal")
	i.sealCalls++
	i.checksum = checksum
	if i.sealErr != nil {
		return ImportReport{}, i.sealErr
	}
	i.state = "sealed"
	return i.report, nil
}

func (i *lichessGenerationImport) Activate(context.Context) error {
	i.catalog.events = append(i.catalog.events, "activate")
	i.activateCalls++
	if i.activateErr != nil {
		return i.activateErr
	}
	i.state = "active"
	i.catalog.head = "candidate-generation"
	return nil
}

func (i *lichessGenerationImport) Abandon(ctx context.Context) error {
	i.catalog.events = append(i.catalog.events, "abandon")
	i.abandonCalls++
	i.abandonContextErr = ctx.Err()
	_, i.abandonHasDeadline = ctx.Deadline()
	i.state = "abandoned"
	return i.abandonErr
}

func newLichessGenerationHarness(
	catalogDirectory string,
	availableBytes func(string) (uint64, error),
) (CollectionImporter, *lichessGenerationCatalog, *lichessGenerationImport) {
	importing := &lichessGenerationImport{report: ImportReport{Accepted: 1}}
	catalog := &lichessGenerationCatalog{
		importing: importing,
		head:      "previous-generation",
	}
	return CollectionImporter{
		Catalog:          catalog,
		Adapters:         []PuzzleAdapter{NewLichessAdapter(chessrules.Rules{})},
		CatalogDirectory: catalogDirectory,
		AvailableBytes:   availableBytes,
	}, catalog, importing
}

func writeLichessGenerationFixture(t *testing.T, directory, contents string) string {
	t.Helper()
	path := filepath.Join(directory, "lichess.csv.zst")
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

func waitForLichessGenerationEvent(t *testing.T, event <-chan struct{}, description string) {
	t.Helper()
	timer := time.NewTimer(2 * time.Second)
	defer timer.Stop()
	select {
	case <-event:
	case <-timer.C:
		t.Fatalf("timed out waiting for %s", description)
	}
}

func TestLichessProducesCoreAndOccurrence(t *testing.T) {
	sourceDirectory := t.TempDir()
	catalogDirectory := t.TempDir()
	path := writeLichessGenerationFixture(t, sourceDirectory, lichessGenerationCSV)
	importer, catalog, importing := newLichessGenerationHarness(
		catalogDirectory,
		func(string) (uint64, error) { return math.MaxUint64, nil },
	)

	report, err := inspectAndImportLichess(context.Background(), importer, path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(report, ImportReport{Accepted: 1}) {
		t.Fatalf("report = %+v, want one accepted occurrence", report)
	}
	if catalog.beginCalls != 1 {
		t.Fatalf("BeginImport calls = %d, want 1", catalog.beginCalls)
	}
	normalizedPath, err := normalizeImportPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if catalog.beginSource.ID != "lichess" ||
		catalog.beginSource.Kind != "lichess" ||
		catalog.beginSource.Path != normalizedPath ||
		catalog.beginSource.StartedAt.IsZero() {
		t.Fatalf("generation source = %+v", catalog.beginSource)
	}
	if len(importing.puzzles) != 1 {
		t.Fatalf("Add calls = %d, want 1", len(importing.puzzles))
	}

	wantCore := PuzzleCore{
		DisplayedFEN:  "7k/5Q2/6K1/8/8/8/8/8 w - - 1 2",
		Solver:        domain.White,
		Solution:      []domain.MoveNode{{UCI: "f7f8"}},
		SolutionPlies: 1,
	}
	wantCore.Fingerprint, err = CoreFingerprint(wantCore)
	if err != nil {
		t.Fatal(err)
	}
	wantRating, wantPopularity, wantPlayCount := 1200, 95, 200
	wantOccurrence := PuzzleOccurrence{
		SourceID:    "lichess",
		SourceKind:  "lichess",
		ExternalID:  "mate1",
		SourceFEN:   "8/5Q1k/6K1/8/8/8/8/8 b - - 0 1",
		PreludeUCI:  "h7h8",
		Rating:      &wantRating,
		Popularity:  &wantPopularity,
		PlayCount:   &wantPlayCount,
		URL:         "https://lichess.org/example",
		Attribution: "Lichess puzzle database (CC0)",
		Metadata: map[string]any{
			"ratingDeviation": 60,
			"openingTags":     []string{"kingsideAttack", "sicilian"},
		},
		Themes:  []string{"mate", "mateIn1"},
		Ordinal: 2,
	}
	got := importing.puzzles[0]
	if !reflect.DeepEqual(got.Core, wantCore) {
		t.Fatalf("core = %#v, want %#v", got.Core, wantCore)
	}
	if !reflect.DeepEqual(got.Occurrence, wantOccurrence) {
		t.Fatalf("occurrence = %#v, want %#v", got.Occurrence, wantOccurrence)
	}
}

func TestLichessPreflightUsesCatalogueVolume(t *testing.T) {
	sourceDirectory := t.TempDir()
	catalogDirectory := t.TempDir()
	path := writeLichessGenerationFixture(t, sourceDirectory, lichessGenerationCSV)
	var checkedPaths []string
	var importer CollectionImporter
	var catalog *lichessGenerationCatalog
	importer, catalog, _ = newLichessGenerationHarness(
		catalogDirectory,
		func(path string) (uint64, error) {
			checkedPaths = append(checkedPaths, path)
			catalog.events = append(catalog.events, "preflight")
			if filepath.Clean(path) == filepath.Clean(catalogDirectory) {
				return math.MaxUint64, nil
			}
			return 0, nil
		},
	)

	if _, err := inspectAndImportLichess(context.Background(), importer, path, nil); err != nil {
		t.Fatalf("Import() using ample catalogue volume: %v", err)
	}
	if !reflect.DeepEqual(checkedPaths, []string{catalogDirectory}) {
		t.Fatalf("AvailableBytes paths = %q, want catalogue directory %q", checkedPaths, catalogDirectory)
	}
	if len(catalog.events) < 2 || catalog.events[0] != "preflight" || catalog.events[1] != "begin" {
		t.Fatalf("events = %q, want preflight before generation creation", catalog.events)
	}
	if filepath.Clean(sourceDirectory) == filepath.Clean(catalogDirectory) {
		t.Fatal("test requires distinct source and catalogue volumes")
	}
}

func TestLichessPreflightFailureCreatesNoGeneration(t *testing.T) {
	sourceDirectory := t.TempDir()
	catalogDirectory := t.TempDir()
	path := writeLichessGenerationFixture(t, sourceDirectory, lichessGenerationCSV)
	var checkedPath string
	importer, catalog, importing := newLichessGenerationHarness(
		catalogDirectory,
		func(path string) (uint64, error) {
			checkedPath = path
			return 0, nil
		},
	)

	if _, err := inspectAndImportLichess(context.Background(), importer, path, nil); err == nil {
		t.Fatal("Import() unexpectedly passed an insufficient-space preflight")
	}
	if checkedPath != catalogDirectory {
		t.Fatalf("AvailableBytes path = %q, want %q", checkedPath, catalogDirectory)
	}
	if catalog.beginCalls != 0 {
		t.Fatalf("BeginImport calls = %d, want 0", catalog.beginCalls)
	}
	if importing.state != "" || importing.physicalRows != 0 || importing.abandonCalls != 0 {
		t.Fatalf("preflight failure created generation state: %+v", importing)
	}
}

func TestLichessCancellationReturnsPromptlyAndOnlyAbandons(t *testing.T) {
	sourceDirectory := t.TempDir()
	catalogDirectory := t.TempDir()
	path := writeLichessGenerationFixture(t, sourceDirectory, lichessGenerationCSV)
	importer, catalog, importing := newLichessGenerationHarness(
		catalogDirectory,
		func(string) (uint64, error) { return math.MaxUint64, nil },
	)
	importing.addEntered = make(chan struct{}, 1)
	importing.waitForCancellation = true
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan lichessGenerationResult, 1)
	go func() {
		report, err := inspectAndImportLichess(ctx, importer, path, nil)
		result <- lichessGenerationResult{report: report, err: err}
	}()

	waitForLichessGenerationEvent(t, importing.addEntered, "the first persisted occurrence")
	cancel()
	done := make(chan struct{}, 1)
	var got lichessGenerationResult
	go func() {
		got = <-result
		done <- struct{}{}
	}()
	waitForLichessGenerationEvent(t, done, "cancelled import to return")

	if !errors.Is(got.err, context.Canceled) {
		t.Fatalf("Import() error = %v, want context.Canceled", got.err)
	}
	if importing.state != "abandoned" || importing.abandonCalls != 1 {
		t.Fatalf("state = %q, abandon calls = %d", importing.state, importing.abandonCalls)
	}
	if importing.sealCalls != 0 || importing.activateCalls != 0 {
		t.Fatalf("seal calls = %d, activate calls = %d", importing.sealCalls, importing.activateCalls)
	}
	if importing.abandonContextErr != nil || !importing.abandonHasDeadline {
		t.Fatalf(
			"abandon context error = %v, has deadline = %v; want bounded non-cancelled context",
			importing.abandonContextErr,
			importing.abandonHasDeadline,
		)
	}
	if catalog.cleanupCalls != 0 || importing.physicalRows != 1 {
		t.Fatalf(
			"cleanup calls = %d, physical rows = %d; cancellation must only mark abandoned",
			catalog.cleanupCalls,
			importing.physicalRows,
		)
	}
	if !reflect.DeepEqual(catalog.events, []string{"begin", "add", "abandon"}) {
		t.Fatalf("events = %q", catalog.events)
	}
}

func TestLichessFailureAndCancellationPreservePreviousHead(t *testing.T) {
	tests := []struct {
		name      string
		configure func(context.CancelFunc, *lichessGenerationImport)
		wantError error
	}{
		{
			name: "write failure",
			configure: func(_ context.CancelFunc, importing *lichessGenerationImport) {
				importing.addErr = errors.New("write occurrence")
			},
			wantError: errors.New("write occurrence"),
		},
		{
			name: "cancellation",
			configure: func(cancel context.CancelFunc, importing *lichessGenerationImport) {
				importing.onAdd = cancel
				importing.returnAddContextError = true
			},
			wantError: context.Canceled,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sourceDirectory := t.TempDir()
			catalogDirectory := t.TempDir()
			path := writeLichessGenerationFixture(t, sourceDirectory, lichessGenerationCSV)
			importer, catalog, importing := newLichessGenerationHarness(
				catalogDirectory,
				func(string) (uint64, error) { return math.MaxUint64, nil },
			)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			test.configure(cancel, importing)

			_, err := inspectAndImportLichess(ctx, importer, path, nil)
			if test.wantError == context.Canceled {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("Import() error = %v, want context.Canceled", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), test.wantError.Error()) {
				t.Fatalf("Import() error = %v, want %q", err, test.wantError)
			}
			if catalog.head != "previous-generation" {
				t.Fatalf("source head = %q, want previous-generation", catalog.head)
			}
			if importing.state != "abandoned" || importing.abandonCalls != 1 {
				t.Fatalf("state = %q, abandon calls = %d", importing.state, importing.abandonCalls)
			}
			if importing.activateCalls != 0 {
				t.Fatalf("Activate calls = %d, want 0", importing.activateCalls)
			}
			if importing.abandonContextErr != nil || !importing.abandonHasDeadline {
				t.Fatalf(
					"abandon context error = %v, has deadline = %v",
					importing.abandonContextErr,
					importing.abandonHasDeadline,
				)
			}
		})
	}
}

func TestLichessSealsThenActivates(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		sourceDirectory := t.TempDir()
		catalogDirectory := t.TempDir()
		path := writeLichessGenerationFixture(t, sourceDirectory, lichessGenerationCSV)
		importer, catalog, importing := newLichessGenerationHarness(
			catalogDirectory,
			func(string) (uint64, error) { return math.MaxUint64, nil },
		)

		report, err := inspectAndImportLichess(context.Background(), importer, path, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(report, importing.report) {
			t.Fatalf("report = %+v, want sealed report %+v", report, importing.report)
		}
		if !reflect.DeepEqual(catalog.events, []string{"begin", "add", "seal", "activate"}) {
			t.Fatalf("events = %q", catalog.events)
		}
		compressed, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		wantChecksum := sha256.Sum256(compressed)
		if importing.checksum != hex.EncodeToString(wantChecksum[:]) {
			t.Fatalf("sealed checksum = %q, want %x", importing.checksum, wantChecksum)
		}
		if importing.state != "active" || catalog.head != "candidate-generation" {
			t.Fatalf("state = %q, head = %q", importing.state, catalog.head)
		}
	})

	t.Run("activation failure preserves sealed generation and prior head", func(t *testing.T) {
		sourceDirectory := t.TempDir()
		catalogDirectory := t.TempDir()
		path := writeLichessGenerationFixture(t, sourceDirectory, lichessGenerationCSV)
		importer, catalog, importing := newLichessGenerationHarness(
			catalogDirectory,
			func(string) (uint64, error) { return math.MaxUint64, nil },
		)
		activateErr := errors.New("activate generation")
		importing.activateErr = activateErr

		_, err := inspectAndImportLichess(context.Background(), importer, path, nil)
		if !errors.Is(err, activateErr) {
			t.Fatalf("Import() error = %v, want activation error", err)
		}
		if !reflect.DeepEqual(catalog.events, []string{"begin", "add", "seal", "activate"}) {
			t.Fatalf("events = %q", catalog.events)
		}
		if importing.state != "sealed" || importing.abandonCalls != 0 || importing.physicalRows != 1 {
			t.Fatalf(
				"state = %q, abandon calls = %d, physical rows = %d",
				importing.state,
				importing.abandonCalls,
				importing.physicalRows,
			)
		}
		if catalog.head != "previous-generation" || catalog.cleanupCalls != 0 {
			t.Fatalf("head = %q, cleanup calls = %d", catalog.head, catalog.cleanupCalls)
		}
	})
}
