package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"chess-trainer/internal/openings"
)

func miniCoursePath(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs("../../internal/openings/testdata/mini.ctcourse")
	if err != nil {
		t.Fatal(err)
	}
	return path
}

func TestRunValidateWritesDeterministicCourseSummary(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"validate", miniCoursePath(t)}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("run() code = %d stderr = %s", code, stderr.String())
	}
	var result struct {
		CourseID       string                  `json:"courseId"`
		ContentVersion string                  `json:"contentVersion"`
		Counts         map[string]int64        `json:"counts"`
		Coverage       openings.CoverageReport `json:"coverage"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode stdout %q: %v", stdout.String(), err)
	}
	if result.CourseID != "synthetic-italian" || result.ContentVersion != "1.0.0" {
		t.Fatalf("validation result = %#v", result)
	}
	wantCounts := map[string]int64{
		"chapters": 3, "positions": 11, "moves": 10, "variations": 7,
		"notes": 1, "lessons": 1, "prompts": 2, "warnings": 0,
	}
	if !reflect.DeepEqual(result.Counts, wantCounts) {
		t.Fatalf("counts = %#v, want %#v", result.Counts, wantCounts)
	}
	if len(result.Coverage.Missing) != 0 || len(result.Coverage.Unexpected) != 0 {
		t.Fatalf("coverage = %#v", result.Coverage)
	}
	if result.Coverage.Missing == nil || result.Coverage.Unexpected == nil {
		t.Fatalf("empty coverage lists must encode as JSON arrays: %#v", result.Coverage)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q", stderr.String())
	}
	var repeated bytes.Buffer
	if code := run([]string{"validate", miniCoursePath(t)}, &repeated, &bytes.Buffer{}); code != 0 {
		t.Fatalf("repeated run code = %d", code)
	}
	if repeated.String() != stdout.String() {
		t.Fatalf("validation output changed between runs")
	}
}

func TestRunValidateWritesStableDiagnostics(t *testing.T) {
	valid, err := os.ReadFile(miniCoursePath(t))
	if err != nil {
		t.Fatal(err)
	}
	invalid := strings.Replace(string(valid), `"uci":"e2e4"`, `"uci":"e2e5"`, 1)
	path := filepath.Join(t.TempDir(), "invalid.ctcourse")
	if err := os.WriteFile(path, []byte(invalid), 0o600); err != nil {
		t.Fatal(err)
	}

	var first, second bytes.Buffer
	if code := run([]string{"validate", path}, &bytes.Buffer{}, &first); code != 1 {
		t.Fatalf("first run code = %d stderr = %s", code, first.String())
	}
	if code := run([]string{"validate", path}, &bytes.Buffer{}, &second); code != 1 {
		t.Fatalf("second run code = %d stderr = %s", code, second.String())
	}
	if first.String() != second.String() || !strings.Contains(first.String(), "illegal_move") {
		t.Fatalf("diagnostics are not stable:\nfirst=%s\nsecond=%s", first.String(), second.String())
	}
}

func TestRunRejectsUnsupportedCommand(t *testing.T) {
	var stderr bytes.Buffer
	if code := run([]string{"inspect"}, &bytes.Buffer{}, &stderr); code != 2 {
		t.Fatalf("run() code = %d, want 2", code)
	}
	if stderr.String() != "usage: coursepack validate <path>\n" {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
