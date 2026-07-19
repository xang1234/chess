package openings

import "testing"

func TestPromptFingerprintIgnoresProseAndSourceLabels(t *testing.T) {
	position := "8/8/8/8/8/8/8/K6k w - -"
	primary := fingerprintMove{ID: "primary", UCI: "a1a2", Destination: "8/8/8/8/8/8/K7/7k b - -"}
	alternatives := []fingerprintMove{{ID: "alternative", UCI: "a1b1", Destination: "8/8/8/8/8/8/8/1K5k b - -"}}

	first := promptFingerprint("prompt", position, primary, alternatives)
	second := promptFingerprint("prompt", position, primary, append([]fingerprintMove(nil), alternatives...))
	if first == "" || first != second {
		t.Fatalf("fingerprints = %q and %q", first, second)
	}

	changed := primary
	changed.Destination = "8/8/8/8/8/8/1K6/7k b - -"
	if promptFingerprint("prompt", position, changed, alternatives) == first {
		t.Fatal("fingerprint ignored a semantic destination change")
	}
}
