//go:build performance

package puzzles

import (
	"context"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/storage"
)

func TestFullLichessImport(t *testing.T) {
	path := os.Getenv("CHESS_TRAINER_LICHESS_PATH")
	if path == "" {
		t.Skip("CHESS_TRAINER_LICHESS_PATH is unset; skipping full local Lichess import")
	}
	root := t.TempDir()
	db, err := storage.Open(filepath.Join(root, "puzzles.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := storage.Migrate(db, "puzzles"); err != nil {
		t.Fatal(err)
	}
	catalog := NewSQLiteCatalog(db)
	importer := LichessImporter{
		Catalog: catalog,
		Rules:   chessrules.Rules{},
		AvailableBytes: func(string) (uint64, error) {
			return math.MaxUint64, nil
		},
	}
	var before runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	beforeRSS := maximumRSS(t)
	peakAlloc := before.Alloc
	started := time.Now()
	var lastLogged int64

	report, err := importer.Import(context.Background(), "lichess", path, func(progress Progress) {
		var current runtime.MemStats
		runtime.ReadMemStats(&current)
		if current.Alloc > peakAlloc {
			peakAlloc = current.Alloc
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
	if err != nil {
		t.Fatal(err)
	}
	if report.Accepted <= 1_000_000 {
		t.Fatalf("accepted=%d, want more than one million", report.Accepted)
	}
	total := report.Accepted + report.Duplicates + report.Rejected
	if total == 0 || float64(report.Rejected)/float64(total) >= 0.001 {
		t.Fatalf("report=%+v, rejected ratio must be below 0.1%%", report)
	}
	const maximumHeapGrowth = 256 << 20
	if growth := peakAlloc - before.Alloc; growth > maximumHeapGrowth {
		t.Fatalf("peak heap growth=%d MiB, want <= %d MiB", growth>>20, maximumHeapGrowth>>20)
	}
	const maximumRSSGrowth = 384 << 20
	if growth := maximumRSS(t) - beforeRSS; growth > maximumRSSGrowth {
		t.Fatalf("maximum RSS growth=%d MiB, want <= %d MiB", growth>>20, maximumRSSGrowth>>20)
	}

	started = time.Now()
	candidates, err := catalog.RatedCandidates(context.Background(), 1400, 1600, nil, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) == 0 {
		t.Fatal("rated candidate query returned no puzzles")
	}
	if elapsed := time.Since(started); elapsed > 250*time.Millisecond {
		t.Fatalf("rated candidate query took %v, want <= 250ms", elapsed)
	}
	matches, err := filepath.Glob(filepath.Join(root, "*.csv"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("import created decompressed CSV files: %v", matches)
	}
}

func maximumRSS(t *testing.T) uint64 {
	t.Helper()
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		t.Fatal(err)
	}
	return uint64(usage.Maxrss)
}
