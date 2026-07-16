//go:build performance

package puzzles

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/storage"
)

const expectedFullLichessSHA256 = "5503bfaf5534518ffe3c4c3bb0ac1ae82350d117ad1a52947796096b75e6247e"

func TestFullLichessImport(t *testing.T) {
	path := os.Getenv("CHESS_TRAINER_LICHESS_PATH")
	if path == "" {
		t.Skip("CHESS_TRAINER_LICHESS_PATH is unset; skipping full local Lichess import")
	}
	if !strings.HasSuffix(path, ".zst") {
		t.Fatalf("CHESS_TRAINER_LICHESS_PATH=%q, want a .zst file", path)
	}
	decompressedPath := strings.TrimSuffix(path, ".zst")
	if _, err := os.Stat(decompressedPath); err == nil {
		t.Fatalf("decompressed CSV already exists before test: %s", decompressedPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	inputInfo, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	databasePath := filepath.Join(root, "puzzles.sqlite")
	store, err := storage.OpenPuzzleStore(databasePath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close full-import puzzle store: %v", err)
		}
	})
	catalog := NewSQLiteCatalog(store.Reader, store.Writer)
	priorSource := testSource("lichess", "lichess", path+"#prior")
	prior := testTrainingPuzzle(priorSource, "performance-prior-head", 1250, "old-theme")
	prior.Occurrence.ExternalID = "old-external-id"
	prior.Occurrence.SourceFEN = "old source FEN"
	prior.Occurrence.PreludeUCI = "a2a4"
	prior.Occurrence.URL = "https://example.test/old"
	prior.Occurrence.Attribution = "old attribution"
	prior.Occurrence.Metadata = map[string]any{"generation": "old"}
	sealAndActivate(t, beginGenerationImport(t, catalog, priorSource), prior)
	expectedOld, err := catalog.Get(context.Background(), prior.Key())
	if err != nil {
		t.Fatal(err)
	}

	availableBefore, err := storage.AvailableBytes(root)
	if err != nil {
		t.Fatal(err)
	}
	requiredBytes := storage.RequiredImportBytes(inputInfo.Size())
	if availableBefore < requiredBytes {
		t.Fatalf(
			"insufficient disk before full import: available=%d MiB required=%d MiB",
			availableBefore>>20,
			requiredBytes>>20,
		)
	}
	t.Logf(
		"dataset_bytes=%d available_before=%d_MiB required_reserve=%d_MiB database=%s",
		inputInfo.Size(),
		availableBefore>>20,
		requiredBytes>>20,
		databasePath,
	)

	timedCatalog := &activationTimingCatalog{CatalogWriter: catalog}
	importer := LichessImporter{
		Catalog:          timedCatalog,
		Rules:            chessrules.Rules{},
		CatalogDirectory: root,
	}
	var before runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	beforeRSS := maximumRSS(t)
	stopHeapSampler := startPeakHeapSampler(before.Alloc)
	started := time.Now()
	var lastLogged int64
	var visibilityChecks int
	var visibilityErr error
	importContext, cancelImport := context.WithCancel(context.Background())
	defer cancelImport()

	report, importErr := importer.Import(importContext, "lichess", path, func(progress Progress) {
		visibilityChecks++
		old, readErr := catalog.Get(importContext, prior.Key())
		if readErr != nil && visibilityErr == nil {
			visibilityErr = fmt.Errorf("read old head at row %d: %w", progress.RowsRead, readErr)
			cancelImport()
		} else if readErr == nil && !reflect.DeepEqual(old, expectedOld) && visibilityErr == nil {
			visibilityErr = fmt.Errorf(
				"old head changed at row %d: got=%+v want=%+v",
				progress.RowsRead,
				old,
				expectedOld,
			)
			cancelImport()
		}
		if progress.RowsRead-lastLogged >= 500_000 {
			lastLogged = progress.RowsRead
			t.Logf(
				"imported %d rows in %s (compressed bytes read: %d MiB)",
				progress.RowsRead,
				time.Since(started).Round(time.Second),
				progress.BytesRead>>20,
			)
		}
	})
	totalElapsed := time.Since(started)
	peakAlloc := stopHeapSampler()
	if visibilityErr != nil {
		t.Fatal(visibilityErr)
	}
	if importErr != nil {
		t.Fatal(importErr)
	}
	if visibilityChecks < 100 {
		t.Fatalf("old-head visibility checks = %d, want at least 100", visibilityChecks)
	}
	if timedCatalog.activationCalls != 1 {
		t.Fatalf("activation calls = %d, want 1", timedCatalog.activationCalls)
	}
	if timedCatalog.activationElapsed >= 5*time.Second {
		t.Fatalf("exact activation took %s, want less than 5s", timedCatalog.activationElapsed)
	}
	if timedCatalog.sealChecksum != expectedFullLichessSHA256 {
		t.Fatalf("dataset SHA-256 = %s, want %s", timedCatalog.sealChecksum, expectedFullLichessSHA256)
	}
	if totalElapsed >= time.Hour {
		t.Fatalf("full Lichess import took %s, want less than 1h", totalElapsed)
	}
	if report.Accepted <= 1_000_000 {
		t.Fatalf("accepted=%d, want more than one million", report.Accepted)
	}
	total := report.Accepted + report.Duplicates + report.Rejected
	if total == 0 || float64(report.Rejected)/float64(total) >= 0.001 {
		t.Fatalf("report=%+v, rejected ratio must be below 0.1%%", report)
	}
	const maximumHeapGrowth = 256 << 20
	heapGrowth := growthSince(peakAlloc, before.Alloc)
	if heapGrowth > maximumHeapGrowth {
		t.Fatalf("peak heap growth=%d MiB, want <= %d MiB", heapGrowth>>20, maximumHeapGrowth>>20)
	}
	const maximumRSSGrowth = 384 << 20
	rssGrowth := growthSince(maximumRSS(t), beforeRSS)
	if rssGrowth > maximumRSSGrowth {
		t.Fatalf("maximum RSS growth=%d MiB, want <= %d MiB", rssGrowth>>20, maximumRSSGrowth>>20)
	}

	candidates, err := catalog.RatedCandidates(context.Background(), 1400, 1600, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) == 0 {
		t.Fatal("rated candidate query returned no puzzles")
	}
	for index, candidate := range candidates {
		if candidate.Core.Fingerprint == "" || candidate.Occurrence.SourceID != "lichess" ||
			candidate.Occurrence.SourceKind != "lichess" {
			t.Fatalf("post-activation candidate %d is not source-aware: %+v", index, candidate)
		}
	}
	if _, err := catalog.Get(context.Background(), prior.Key()); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("prior head remains active after replacement: %v", err)
	}
	matches, err := filepath.Glob(filepath.Join(root, "*.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("import created decompressed CSV files: %v", matches)
	}
	if _, err := os.Stat(decompressedPath); err == nil {
		t.Fatalf("import created decompressed CSV: %s", decompressedPath)
	} else if !errors.Is(err, os.ErrNotExist) {
		t.Fatal(err)
	}
	availableAfter, err := storage.AvailableBytes(root)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf(
		"full_import_elapsed=%s activation_elapsed=%s accepted=%d duplicates=%d rejected=%d rejection_ratio=%.6f visibility_checks=%d peak_heap_growth=%d_MiB max_rss_growth=%d_MiB catalog_bytes=%d available_after=%d_MiB",
		totalElapsed,
		timedCatalog.activationElapsed,
		report.Accepted,
		report.Duplicates,
		report.Rejected,
		float64(report.Rejected)/float64(total),
		visibilityChecks,
		heapGrowth>>20,
		rssGrowth>>20,
		catalogBytes(t, databasePath),
		availableAfter>>20,
	)
}

type activationTimingCatalog struct {
	CatalogWriter
	activationElapsed time.Duration
	activationCalls   int
	sealChecksum      string
}

func (c *activationTimingCatalog) BeginImport(ctx context.Context, source Source) (GenerationImport, error) {
	importing, err := c.CatalogWriter.BeginImport(ctx, source)
	if err != nil {
		return nil, err
	}
	return &activationTimingImport{GenerationImport: importing, observer: c}, nil
}

type activationTimingImport struct {
	GenerationImport
	observer *activationTimingCatalog
}

func (i *activationTimingImport) Seal(ctx context.Context, checksum string) (ImportReport, error) {
	i.observer.sealChecksum = strings.ToLower(strings.TrimSpace(checksum))
	return i.GenerationImport.Seal(ctx, checksum)
}

func (i *activationTimingImport) Activate(ctx context.Context) error {
	started := time.Now()
	err := i.GenerationImport.Activate(ctx)
	i.observer.activationElapsed = time.Since(started)
	i.observer.activationCalls++
	return err
}

func startPeakHeapSampler(initial uint64) func() uint64 {
	stop := make(chan struct{})
	result := make(chan uint64, 1)
	go func() {
		ticker := time.NewTicker(250 * time.Millisecond)
		defer ticker.Stop()
		peak := initial
		for {
			select {
			case <-ticker.C:
				var current runtime.MemStats
				runtime.ReadMemStats(&current)
				if current.Alloc > peak {
					peak = current.Alloc
				}
			case <-stop:
				var current runtime.MemStats
				runtime.ReadMemStats(&current)
				if current.Alloc > peak {
					peak = current.Alloc
				}
				result <- peak
				return
			}
		}
	}()
	return func() uint64 {
		close(stop)
		return <-result
	}
}

func growthSince(peak, baseline uint64) uint64 {
	if peak <= baseline {
		return 0
	}
	return peak - baseline
}

func catalogBytes(t *testing.T, databasePath string) int64 {
	t.Helper()
	var total int64
	for _, path := range []string{databasePath, databasePath + "-wal", databasePath + "-shm"} {
		info, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			t.Fatal(err)
		}
		total += info.Size()
	}
	return total
}
