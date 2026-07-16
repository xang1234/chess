//go:build darwin || linux

package puzzles

import (
	"runtime"
	"syscall"
	"testing"
)

func maximumRSS(t *testing.T) uint64 {
	t.Helper()
	var usage syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &usage); err != nil {
		t.Fatal(err)
	}
	if usage.Maxrss < 0 {
		t.Fatalf("getrusage returned negative maximum RSS: %d", usage.Maxrss)
	}
	return maximumRSSBytes(runtime.GOOS, uint64(usage.Maxrss))
}
