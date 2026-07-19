package storage

import (
	"path/filepath"
	"testing"
)

func TestPathsAt(t *testing.T) {
	p := PathsAt("/tmp/support")
	if p.UserDB != filepath.Join("/tmp/support", "user.sqlite") {
		t.Fatalf("UserDB=%q", p.UserDB)
	}
	if p.CoursesDB != filepath.Join("/tmp/support", "courses.sqlite") {
		t.Fatalf("CoursesDB=%q", p.CoursesDB)
	}
	if p.BackupsDir != filepath.Join("/tmp/support", "backups") {
		t.Fatalf("BackupsDir=%q", p.BackupsDir)
	}
}
