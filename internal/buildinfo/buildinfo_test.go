package buildinfo

import "testing"

func TestProductName(t *testing.T) {
	if Name != "Chess Trainer" {
		t.Fatalf("Name = %q, want %q", Name, "Chess Trainer")
	}
}
