package openings

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func readMiniCoursePack(t *testing.T) []byte {
	t.Helper()
	contents, err := os.ReadFile("testdata/mini.ctcourse")
	if err != nil {
		t.Fatal(err)
	}
	return contents
}

func TestDecodeCoursePackAcceptsStrictSyntheticFixture(t *testing.T) {
	pack, err := DecodeCoursePack(bytes.NewReader(readMiniCoursePack(t)))
	if err != nil {
		t.Fatal(err)
	}
	if pack.CourseID != "synthetic-italian" || pack.DefaultDepth != DepthReference {
		t.Fatalf("decoded pack = %+v", pack)
	}
	if len(pack.Moves) != 10 || len(pack.Lessons) != 1 || len(pack.Prompts) != 2 {
		t.Fatalf("fixture counts: moves=%d lessons=%d prompts=%d", len(pack.Moves), len(pack.Lessons), len(pack.Prompts))
	}
}

func TestDecodeCoursePackRejectsUnknownFieldsAtEveryLevel(t *testing.T) {
	valid := readMiniCoursePack(t)
	tests := map[string][]byte{
		"root":   bytes.Replace(valid, []byte(`"prompts":`), []byte(`"unexpected":true,"prompts":`), 1),
		"nested": bytes.Replace(valid, []byte(`"edition":"Synthetic"`), []byte(`"edition":"Synthetic","unexpected":true`), 1),
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := DecodeCoursePack(bytes.NewReader(input))
			if err == nil || !strings.Contains(err.Error(), "unknown field") {
				t.Fatalf("DecodeCoursePack() error = %v, want unknown field", err)
			}
		})
	}
}

func TestDecodeCoursePackRejectsTrailingDocument(t *testing.T) {
	input := append(readMiniCoursePack(t), []byte("\n{}\n")...)
	_, err := DecodeCoursePack(bytes.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "second JSON value") {
		t.Fatalf("DecodeCoursePack() error = %v, want second JSON value", err)
	}
}

func TestDecodeCoursePackRejectsInvalidUTF8(t *testing.T) {
	input := bytes.Replace(readMiniCoursePack(t), []byte("Synthetic Italian"), []byte{0xff}, 1)
	_, err := DecodeCoursePack(bytes.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "UTF-8") {
		t.Fatalf("DecodeCoursePack() error = %v, want UTF-8 error", err)
	}
}

func TestDecodeCoursePackRejectsUnsupportedSchemaVersion(t *testing.T) {
	input := bytes.Replace(readMiniCoursePack(t), []byte(`"schemaVersion":1`), []byte(`"schemaVersion":2`), 1)
	_, err := DecodeCoursePack(bytes.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "unsupported course schema version 2") {
		t.Fatalf("DecodeCoursePack() error = %v", err)
	}
}

func TestDecodeCoursePackRejectsMissingRootText(t *testing.T) {
	input := bytes.Replace(readMiniCoursePack(t), []byte(`"courseId":"synthetic-italian"`), []byte(`"courseId":"  "`), 1)
	_, err := DecodeCoursePack(bytes.NewReader(input))
	if err == nil || !strings.Contains(err.Error(), "courseId is required") {
		t.Fatalf("DecodeCoursePack() error = %v", err)
	}
}

func TestDecodeCoursePackReportsMissingRootTextDeterministically(t *testing.T) {
	for iteration := 0; iteration < 100; iteration++ {
		_, err := DecodeCoursePack(strings.NewReader(`{"schemaVersion":1}`))
		if err == nil || err.Error() != "courseId is required" {
			t.Fatalf("iteration %d error = %v, want courseId first", iteration, err)
		}
	}
}
