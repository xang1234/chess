package puzzles

import "testing"

func TestMaximumRSSBytesNormalizesDarwinAndLinuxUnits(t *testing.T) {
	const want = uint64(384 << 20)
	tests := []struct {
		name string
		goos string
		raw  uint64
	}{
		{name: "darwin reports bytes", goos: "darwin", raw: want},
		{name: "linux reports kibibytes", goos: "linux", raw: want >> 10},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := maximumRSSBytes(test.goos, test.raw); got != want {
				t.Fatalf("maximumRSSBytes(%q, %d) = %d, want %d", test.goos, test.raw, got, want)
			}
		})
	}
}
