package importjob

import (
	"context"
	"strings"
	"testing"

	"chess-trainer/internal/importing"
)

type routerImporter struct {
	format importing.Format
	called chan importing.Inspection
}

func (i routerImporter) Supports(format importing.Format) bool {
	return format == i.format
}

func (i routerImporter) Import(
	_ context.Context,
	inspection importing.Inspection,
	_ importing.ProgressSink,
) (importing.Report, error) {
	if i.called != nil {
		i.called <- inspection
	}
	return importing.Report{Accepted: 1}, nil
}

func TestRouterDispatchesInspectionToMatchingImporter(t *testing.T) {
	puzzle := routerImporter{format: "canonical-json", called: make(chan importing.Inspection, 1)}
	course := routerImporter{format: "coursepack", called: make(chan importing.Inspection, 1)}
	router := NewRouter(puzzle, course)
	inspection := importing.Inspection{Format: "coursepack", SourceID: "italian", Path: "/course"}

	if !router.Supports(inspection.Format) || router.Supports("unknown") {
		t.Fatalf("router support mismatch")
	}
	result, err := router.Import(context.Background(), inspection, nil)
	if err != nil || result.Accepted != 1 {
		t.Fatalf("Import() = %#v, %v", result, err)
	}
	if got := <-course.called; got != inspection {
		t.Fatalf("course inspection = %#v, want %#v", got, inspection)
	}
	select {
	case got := <-puzzle.called:
		t.Fatalf("puzzle importer was called with %#v", got)
	default:
	}
}

func TestRouterRejectsMissingAndAmbiguousFormats(t *testing.T) {
	router := NewRouter(
		routerImporter{format: "coursepack"},
		routerImporter{format: "coursepack"},
	)
	if router.Supports("coursepack") {
		t.Fatal("ambiguous format unexpectedly reported as supported")
	}
	_, err := router.Import(context.Background(), importing.Inspection{Format: "coursepack"}, nil)
	if err == nil || !strings.Contains(err.Error(), "multiple") {
		t.Fatalf("ambiguous Import() error = %v", err)
	}
	_, err = router.Import(context.Background(), importing.Inspection{Format: "unknown"}, nil)
	if err == nil || !strings.Contains(err.Error(), "not configured") {
		t.Fatalf("missing Import() error = %v", err)
	}
}
