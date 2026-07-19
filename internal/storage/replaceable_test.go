package storage

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestOpenReplaceableQuarantinesCorruptStoreAndRecreatesSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "courses.sqlite")
	if err := os.WriteFile(path, []byte("not a sqlite database"), 0o600); err != nil {
		t.Fatal(err)
	}
	fixed := time.Date(2026, time.July, 18, 5, 0, 0, 0, time.UTC)

	db, notice, err := OpenReplaceable(path, "courses", func() time.Time { return fixed })
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	wantQuarantine := path + ".quarantine-20260718T050000Z"
	if notice.QuarantinedPath != wantQuarantine {
		t.Fatalf("quarantine path = %q, want %q", notice.QuarantinedPath, wantQuarantine)
	}
	if !strings.Contains(notice.Detail, "integrity") && !strings.Contains(notice.Detail, "database") {
		t.Fatalf("notice detail = %q", notice.Detail)
	}
	quarantined, err := os.ReadFile(wantQuarantine)
	if err != nil {
		t.Fatal(err)
	}
	if string(quarantined) != "not a sqlite database" {
		t.Fatalf("quarantined contents = %q", quarantined)
	}
	var migrations int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&migrations); err != nil {
		t.Fatal(err)
	}
	if migrations != 1 {
		t.Fatalf("migrations = %d, want 1", migrations)
	}
	var table string
	if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE name='course_generations'`).Scan(&table); err != nil {
		t.Fatal(err)
	}
}

func TestOpenReplaceableMovesWALAndSHMWithCorruptStore(t *testing.T) {
	path := filepath.Join(t.TempDir(), "courses.sqlite")
	for suffix, contents := range map[string]string{
		"":     "not sqlite",
		"-wal": "wal bytes",
		"-shm": "shm bytes",
	} {
		if err := os.WriteFile(path+suffix, []byte(contents), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	fixed := time.Date(2026, time.July, 18, 6, 0, 0, 0, time.UTC)

	db, notice, err := OpenReplaceable(path, "courses", func() time.Time { return fixed })
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for suffix, contents := range map[string]string{
		"":     "not sqlite",
		"-wal": "wal bytes",
		"-shm": "shm bytes",
	} {
		got, err := os.ReadFile(notice.QuarantinedPath + suffix)
		if err != nil {
			t.Fatalf("read quarantined %s: %v", suffix, err)
		}
		if string(got) != contents {
			t.Fatalf("quarantined %s = %q, want %q", suffix, got, contents)
		}
	}
}

func TestOpenReplaceableCreatesMissingStoreWithoutNotice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "courses.sqlite")
	db, notice, err := OpenReplaceable(path, "courses", time.Now)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if notice != (QuarantineNotice{}) {
		t.Fatalf("notice = %+v, want zero", notice)
	}
}
