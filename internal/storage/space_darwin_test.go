package storage

import (
	"math"
	"testing"
)

func TestRequiredImportBytes(t *testing.T) {
	const compressed = int64(100 * 1024 * 1024)
	want := uint64(compressed*40 + 1024*1024*1024)
	if got := RequiredImportBytes(compressed); got != want {
		t.Fatalf("RequiredImportBytes()=%d, want %d", got, want)
	}
	if got := RequiredImportBytes(-1); got != importSafetyMargin {
		t.Fatalf("negative-size reserve=%d, want %d", got, importSafetyMargin)
	}
	if got := RequiredImportBytes(math.MaxInt64); got != math.MaxUint64 {
		t.Fatalf("overflow reserve=%d, want MaxUint64", got)
	}
}

func TestAvailableBytes(t *testing.T) {
	available, err := AvailableBytes(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if available == 0 {
		t.Fatal("AvailableBytes() returned zero")
	}
}
