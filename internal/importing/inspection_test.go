package importing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCompareInspectionReportsSubjectAndChangedField(t *testing.T) {
	base := Inspection{
		Path: "/tmp/a.ctcourse", Filename: "a.ctcourse",
		Format: "coursepack", SourceID: "italian-white",
		SourceIDOrigin: SourceIDEmbedded, SourceName: "Italian",
		URL: "https://example.invalid/source", Attribution: "Reference",
	}
	changed := base
	changed.SourceID = "other"
	err := CompareInspection(changed, base, "course import")
	if err == nil || !strings.Contains(err.Error(), "course import source ID changed after inspection") {
		t.Fatalf("CompareInspection() error = %v", err)
	}
}

func TestNormalizePathRequiresSubjectAndResolvesSymlink(t *testing.T) {
	if _, err := NormalizePath(" ", "puzzle import"); err == nil ||
		err.Error() != "puzzle import path is required" {
		t.Fatalf("empty path error = %v", err)
	}
	directory := t.TempDir()
	target := filepath.Join(directory, "target.txt")
	if err := os.WriteFile(target, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(directory, "link.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	resolvedTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}
	got, err := NormalizePath(link, "puzzle import")
	if err != nil || got != resolvedTarget {
		t.Fatalf("NormalizePath() = %q, %v; want %q", got, err, resolvedTarget)
	}
}
