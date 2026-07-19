package importing

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReportKeepsPuzzleFieldsAndAddsOptionalCounts(t *testing.T) {
	encoded, err := json.Marshal(Report{
		Accepted: 2,
		Rejected: 1,
		Counts:   map[string]int64{"chapters": 3, "moves": 40},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, fragment := range []string{
		`"accepted":2`,
		`"duplicates":0`,
		`"rejected":1`,
		`"examples":null`,
		`"counts":{"chapters":3,"moves":40}`,
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("JSON %s is missing %s", text, fragment)
		}
	}
}

func TestInspectionAndProgressKeepCurrentJSONNames(t *testing.T) {
	encoded, err := json.Marshal(struct {
		Inspection Inspection `json:"inspection"`
		Progress   Progress   `json:"progress"`
	}{
		Inspection: Inspection{
			Path:             "/tmp/course.ctcourse",
			Filename:         "course.ctcourse",
			Format:           Format("coursepack"),
			FormatLabel:      "Opening course",
			SourceID:         "italian-white",
			SourceIDOrigin:   SourceIDEmbedded,
			ReplacesExisting: true,
		},
		Progress: Progress{
			Phase: PhaseParsing, RowsRead: 7, BytesRead: 11, TotalBytes: 13,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, fragment := range []string{
		`"format":"coursepack"`,
		`"sourceIdOrigin":"embedded"`,
		`"replacesExisting":true`,
		`"phase":"parsing"`,
		`"totalBytes":13`,
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("JSON %s is missing %s", text, fragment)
		}
	}
}
