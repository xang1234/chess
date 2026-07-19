package storage

import (
	"os"
	"path/filepath"
)

type Paths struct {
	Root       string
	PuzzlesDB  string
	CoursesDB  string
	LibraryDB  string
	UserDB     string
	BackupsDir string
}

func PathsAt(root string) Paths {
	return Paths{
		Root:       root,
		PuzzlesDB:  filepath.Join(root, "puzzles.sqlite"),
		CoursesDB:  filepath.Join(root, "courses.sqlite"),
		LibraryDB:  filepath.Join(root, "library.sqlite"),
		UserDB:     filepath.Join(root, "user.sqlite"),
		BackupsDir: filepath.Join(root, "backups"),
	}
}

func DefaultPaths() (Paths, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, err
	}
	return PathsAt(filepath.Join(base, "Chess Trainer")), nil
}

func (p Paths) Ensure() error {
	if err := os.MkdirAll(p.Root, 0o700); err != nil {
		return err
	}
	return os.MkdirAll(p.BackupsDir, 0o700)
}
