package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"chess-trainer/internal/chessrules"
	"chess-trainer/internal/openings"
)

type validationOutput struct {
	CourseID       string                  `json:"courseId"`
	ContentVersion string                  `json:"contentVersion"`
	Counts         map[string]int64        `json:"counts"`
	Coverage       openings.CoverageReport `json:"coverage"`
}

type validationFailure struct {
	Diagnostics []openings.Diagnostic `json:"diagnostics"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 || args[0] != "validate" {
		_, _ = io.WriteString(stderr, "usage: coursepack validate <path>\n")
		return 2
	}
	compiled, err := openings.ValidateCoursePackFile(
		context.Background(),
		args[1],
		chessrules.Rules{},
	)
	if err != nil {
		var validationErr *openings.ValidationError
		if errors.As(err, &validationErr) {
			if encodeErr := writeIndentedJSON(stderr, validationFailure{
				Diagnostics: validationErr.Diagnostics,
			}); encodeErr != nil {
				_, _ = fmt.Fprintf(stderr, "write diagnostics: %v\n", encodeErr)
			}
		} else {
			_, _ = fmt.Fprintln(stderr, err)
		}
		return 1
	}
	if err := writeIndentedJSON(stdout, validationOutput{
		CourseID:       compiled.Pack.CourseID,
		ContentVersion: compiled.Pack.ContentVersion,
		Counts:         openings.StructuralCounts(compiled),
		Coverage:       compiled.Coverage,
	}); err != nil {
		_, _ = fmt.Fprintf(stderr, "write validation result: %v\n", err)
		return 1
	}
	return 0
}

func writeIndentedJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
