package importjob

import (
	"context"
	"fmt"

	"chess-trainer/internal/importing"
)

type Router struct {
	importers []Importer
}

func NewRouter(importers ...Importer) *Router {
	return &Router{importers: append([]Importer(nil), importers...)}
}

func (r *Router) Supports(format importing.Format) bool {
	return len(r.matching(format)) == 1
}

func (r *Router) Import(
	ctx context.Context,
	inspection importing.Inspection,
	progress importing.ProgressSink,
) (importing.Report, error) {
	matches := r.matching(inspection.Format)
	switch len(matches) {
	case 0:
		return importing.Report{}, fmt.Errorf(
			"importer for kind %q is not configured",
			inspection.Format,
		)
	case 1:
		return matches[0].Import(ctx, inspection, progress)
	default:
		return importing.Report{}, fmt.Errorf(
			"multiple importers are configured for kind %q",
			inspection.Format,
		)
	}
}

func (r *Router) matching(format importing.Format) []Importer {
	if r == nil {
		return nil
	}
	matches := make([]Importer, 0, 1)
	for _, importer := range r.importers {
		if importer != nil && importer.Supports(format) {
			matches = append(matches, importer)
		}
	}
	return matches
}
