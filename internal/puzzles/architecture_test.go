package puzzles

import (
	"bufio"
	"os"
	"reflect"
	"strings"
	"testing"
)

func TestPuzzleProductionFilesRemainFocused(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		file, err := os.Open(entry.Name())
		if err != nil {
			t.Fatal(err)
		}
		scanner := bufio.NewScanner(file)
		lines := 0
		for scanner.Scan() {
			lines++
		}
		closeErr := file.Close()
		if err := scanner.Err(); err != nil {
			t.Fatal(err)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		if lines > 1_000 {
			t.Errorf("production file %s has %d lines; split files before they exceed 1,000 lines", entry.Name(), lines)
		}
	}
}

func TestCanonicalJSONDocumentStreamUsesExplicitStates(t *testing.T) {
	typeOfStream := reflect.TypeOf(canonicalJSONDocumentStream{})
	for index := 0; index < typeOfStream.NumField(); index++ {
		field := typeOfStream.Field(index)
		if field.Type.Kind() == reflect.Bool {
			t.Errorf("canonicalJSONDocumentStream field %s is bool; model parser states explicitly", field.Name)
		}
	}
}
