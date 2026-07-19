package openings

import (
	"fmt"
	"sort"
	"strings"
)

type coverageGroup struct {
	item CoverageItem
	path string
}

func compileCoverage(pack CoursePack, diagnostics []Diagnostic) (CoverageReport, []Diagnostic) {
	report := CoverageReport{
		Expected:   append([]string{}, pack.SourceCoverage.ExpectedReferences...),
		Captured:   []CoverageItem{},
		Missing:    []string{},
		Unexpected: []string{},
	}
	pages := make(map[int]struct{}, len(pack.SourceCoverage.PrintedPages))
	for index, page := range pack.SourceCoverage.PrintedPages {
		path := fmt.Sprintf("sourceCoverage.printedPages[%d]", index)
		if page <= 0 {
			diagnostics = append(diagnostics, Diagnostic{Code: "invalid_source_page", Path: path, Message: "printed page must be positive"})
		}
		if _, duplicate := pages[page]; duplicate {
			diagnostics = append(diagnostics, Diagnostic{Code: "duplicate_source_page", Path: path, Message: fmt.Sprintf("printed page %d is duplicated", page)})
		}
		pages[page] = struct{}{}
	}

	expected := make(map[string]struct{}, len(pack.SourceCoverage.ExpectedReferences))
	for index, coverageID := range pack.SourceCoverage.ExpectedReferences {
		path := fmt.Sprintf("sourceCoverage.expectedReferences[%d]", index)
		if strings.TrimSpace(coverageID) == "" {
			diagnostics = append(diagnostics, Diagnostic{Code: "invalid_coverage_id", Path: path, Message: "coverage ID is required"})
			continue
		}
		if _, duplicate := expected[coverageID]; duplicate {
			diagnostics = append(diagnostics, Diagnostic{Code: "duplicate_expected_coverage", Path: path, Message: fmt.Sprintf("coverage ID %q is duplicated", coverageID)})
		}
		expected[coverageID] = struct{}{}
	}

	groups := map[string]*coverageGroup{}
	add := func(recordID, path string, ref SourceRef) {
		if ref.PrintedPage <= 0 {
			diagnostics = append(diagnostics, Diagnostic{Code: "invalid_source_page", Path: path + ".printedPage", Message: "printed page must be positive"})
		} else if _, declared := pages[ref.PrintedPage]; !declared {
			diagnostics = append(diagnostics, Diagnostic{Code: "undeclared_source_page", Path: path + ".printedPage", Message: fmt.Sprintf("printed page %d is not declared", ref.PrintedPage)})
		}
		coverageID := strings.TrimSpace(ref.CoverageID)
		if coverageID == "" {
			diagnostics = append(diagnostics, Diagnostic{Code: "invalid_coverage_id", Path: path + ".coverageId", Message: "coverage ID is required"})
			return
		}
		group, exists := groups[coverageID]
		if !exists {
			groups[coverageID] = &coverageGroup{
				item: CoverageItem{
					CoverageID: coverageID, PrintedPage: ref.PrintedPage,
					TableColumn: ref.TableColumn, NoteLabel: ref.NoteLabel,
					RecordIDs: []string{recordID},
				},
				path: path,
			}
			return
		}
		if group.item.PrintedPage != ref.PrintedPage || group.item.TableColumn != ref.TableColumn || group.item.NoteLabel != ref.NoteLabel {
			diagnostics = append(diagnostics, Diagnostic{
				Code: "coverage_coordinate_conflict", Path: path,
				Message: fmt.Sprintf("coverage ID %q uses coordinates that differ from %s", coverageID, group.path),
			})
		}
		group.item.RecordIDs = append(group.item.RecordIDs, recordID)
	}

	for index, move := range pack.Moves {
		add("move:"+move.MoveID, fmt.Sprintf("moves[%d].sourceRef", index), move.SourceRef)
	}
	for index, note := range pack.Notes {
		add("note:"+note.NoteID, fmt.Sprintf("notes[%d].sourceRef", index), note.SourceRef)
	}

	capturedIDs := make([]string, 0, len(groups))
	for coverageID, group := range groups {
		capturedIDs = append(capturedIDs, coverageID)
		sort.Strings(group.item.RecordIDs)
		report.Captured = append(report.Captured, group.item)
	}
	sort.Slice(report.Captured, func(left, right int) bool {
		a, b := report.Captured[left], report.Captured[right]
		if a.PrintedPage != b.PrintedPage {
			return a.PrintedPage < b.PrintedPage
		}
		if a.TableColumn != b.TableColumn {
			return a.TableColumn < b.TableColumn
		}
		if a.NoteLabel != b.NoteLabel {
			return a.NoteLabel < b.NoteLabel
		}
		return a.CoverageID < b.CoverageID
	})
	for coverageID := range expected {
		if _, captured := groups[coverageID]; !captured {
			report.Missing = append(report.Missing, coverageID)
		}
	}
	for _, coverageID := range capturedIDs {
		if _, declared := expected[coverageID]; !declared {
			report.Unexpected = append(report.Unexpected, coverageID)
		}
	}
	sort.Strings(report.Missing)
	sort.Strings(report.Unexpected)
	for _, coverageID := range report.Missing {
		diagnostics = append(diagnostics, Diagnostic{Code: "missing_coverage", Path: "sourceCoverage.expectedReferences", Message: fmt.Sprintf("coverage ID %q was not captured", coverageID)})
	}
	for _, coverageID := range report.Unexpected {
		diagnostics = append(diagnostics, Diagnostic{Code: "unexpected_coverage", Path: "sourceCoverage.expectedReferences", Message: fmt.Sprintf("captured coverage ID %q was not expected", coverageID)})
	}
	return report, diagnostics
}
