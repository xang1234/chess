package openings

import (
	"slices"
	"testing"

	"chess-trainer/internal/chessrules"
)

func TestCoverageGroupsRecordsThatShareOneSourceUnit(t *testing.T) {
	pack := decodeMiniPack(t)
	pack.Moves[0].SourceRef.CoverageID = "p1-opening-column"
	pack.Moves[1].SourceRef.CoverageID = "p1-opening-column"
	pack.Moves[0].SourceRef.TableColumn = "A"
	pack.Moves[1].SourceRef.TableColumn = "A"
	pack.SourceCoverage.ExpectedReferences = slices.Delete(pack.SourceCoverage.ExpectedReferences, 1, 3)
	pack.SourceCoverage.ExpectedReferences = append(pack.SourceCoverage.ExpectedReferences, "p1-opening-column")

	compiled, err := Compile(pack, chessrules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	var grouped *CoverageItem
	for index := range compiled.Coverage.Captured {
		if compiled.Coverage.Captured[index].CoverageID == "p1-opening-column" {
			grouped = &compiled.Coverage.Captured[index]
			break
		}
	}
	if grouped == nil {
		t.Fatalf("coverage = %+v", compiled.Coverage)
	}
	want := []string{"move:black-e5", "move:white-e4"}
	if !slices.Equal(grouped.RecordIDs, want) || grouped.PrintedPage != 1 || grouped.TableColumn != "A" {
		t.Fatalf("grouped coverage = %+v, want records %v", *grouped, want)
	}
}

func TestCoverageRejectsConflictingCoordinatesForSharedID(t *testing.T) {
	pack := decodeMiniPack(t)
	pack.Moves[1].SourceRef.CoverageID = pack.Moves[0].SourceRef.CoverageID
	pack.Moves[1].SourceRef.TableColumn = "B"
	pack.SourceCoverage.ExpectedReferences = slices.Delete(pack.SourceCoverage.ExpectedReferences, 2, 3)

	_, err := Compile(pack, chessrules.Rules{})
	validation, ok := err.(*ValidationError)
	if !ok || !hasDiagnosticCode(validation.Diagnostics, "coverage_coordinate_conflict") {
		t.Fatalf("Compile() error = %v", err)
	}
}
