package storage

import "testing"

func TestRequiredImportBytes(t *testing.T) {
	const compressed = int64(100 * 1024 * 1024)
	want := uint64(compressed*10 + 512*1024*1024)
	if got := RequiredImportBytes(compressed); got != want {
		t.Fatalf("RequiredImportBytes()=%d, want %d", got, want)
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
