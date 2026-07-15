package domain

import (
	"slices"
	"testing"
)

func TestNormalizeThemesTrimsDeduplicatesAndSorts(t *testing.T) {
	got := NormalizeThemes([]string{
		" fork ",
		"",
		"pin",
		"fork",
		"   ",
		"Pin",
		"\tzugzwang\n",
	})
	want := []string{"Pin", "fork", "pin", "zugzwang"}

	if !slices.Equal(got, want) {
		t.Fatalf("NormalizeThemes() = %q, want %q", got, want)
	}
}
