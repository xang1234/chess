package importing

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// NormalizePath returns the stable absolute identity used between inspection and import.
func NormalizePath(importPath, subject string) (string, error) {
	importPath = strings.TrimSpace(importPath)
	subject = strings.TrimSpace(subject)
	if importPath == "" {
		if subject == "" {
			return "", errors.New("import path is required")
		}
		return "", fmt.Errorf("%s path is required", subject)
	}
	absolute, err := filepath.Abs(importPath)
	if err != nil {
		return "", err
	}
	normalized := filepath.Clean(absolute)
	if resolved, err := filepath.EvalSymlinks(normalized); err == nil {
		normalized = filepath.Clean(resolved)
	}
	return normalized, nil
}

// CompareInspection ensures the confirmed import identity still matches the source.
func CompareInspection(current, expected Inspection, subject string) error {
	fields := []struct {
		name          string
		current, want string
	}{
		{name: "path", current: current.Path, want: expected.Path},
		{name: "filename", current: current.Filename, want: expected.Filename},
		{name: "format", current: string(current.Format), want: string(expected.Format)},
		{name: "source ID", current: current.SourceID, want: expected.SourceID},
		{name: "source ID origin", current: string(current.SourceIDOrigin), want: string(expected.SourceIDOrigin)},
		{name: "source name", current: current.SourceName, want: expected.SourceName},
		{name: "source URL", current: current.URL, want: expected.URL},
		{name: "source attribution", current: current.Attribution, want: expected.Attribution},
	}
	for _, field := range fields {
		if field.current != field.want {
			return fmt.Errorf(
				"%s %s changed after inspection: got %q, want %q",
				strings.TrimSpace(subject), field.name, field.current, field.want,
			)
		}
	}
	return nil
}
