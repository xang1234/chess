//go:build !darwin && !linux

package puzzles

import "testing"

func maximumRSS(t *testing.T) uint64 {
	t.Helper()
	t.Skip("maximum RSS performance gate is supported only on Darwin and Linux")
	return 0
}
