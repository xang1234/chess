package importjob

import (
	"testing"

	"chess-trainer/internal/puzzles"
)

func TestCloneReportOwnsExamplesAndCounts(t *testing.T) {
	original := puzzles.ImportReport{
		Examples: []puzzles.Rejection{{Ordinal: 1, Reason: "first"}},
		Counts:   map[string]int64{"chapters": 3},
	}

	cloned := cloneReport(original)
	if cloned.Counts["chapters"] != 3 {
		t.Fatalf("cloned counts = %+v, want chapters preserved", cloned.Counts)
	}
	cloned.Examples[0].Reason = "changed"
	cloned.Counts["chapters"] = 99

	if original.Examples[0].Reason != "first" {
		t.Fatalf("original examples changed: %+v", original.Examples)
	}
	if original.Counts["chapters"] != 3 {
		t.Fatalf("original counts changed: %+v", original.Counts)
	}
}
