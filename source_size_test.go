package main

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

const maximumOwnedSourceLines = 1_000

func TestOwnedSourceFilesStayUnderLimit(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	excludedDirectories := map[string]struct{}{
		".git": {}, ".worktrees": {}, "node_modules": {}, "vendor": {},
		"build": {}, "dist": {}, "coverage": {}, "wailsjs": {},
	}
	ownedExtensions := map[string]struct{}{`.go`: {}, `.ts`: {}, `.svelte`: {}}

	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root {
				if _, excluded := excludedDirectories[entry.Name()]; excluded {
					return filepath.SkipDir
				}
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		if _, owned := ownedExtensions[filepath.Ext(entry.Name())]; !owned {
			return nil
		}

		lines, err := sourceLineCount(path)
		if err != nil {
			return err
		}
		if lines <= maximumOwnedSourceLines {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		t.Errorf(
			"owned source file %s has %d lines; split it before it exceeds %d lines",
			filepath.ToSlash(relative),
			lines,
			maximumOwnedSourceLines,
		)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func sourceLineCount(path string) (int, error) {
	file, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 1<<20)
	lines := 0
	for scanner.Scan() {
		lines++
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("count source lines in %s: %w", path, err)
	}
	return lines, nil
}
