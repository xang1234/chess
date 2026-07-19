package openings

import (
	"context"
	"slices"
	"testing"

	"chess-trainer/internal/chessrules"
)

func TestExploreFiltersMovesPreservesSourceOrderAndDoesNotWriteLearnerState(t *testing.T) {
	ctx := context.Background()
	fixture := newOpeningServiceFixture(t)
	installExplorerTransposition(t, fixture)

	before := openingLearningRowCounts(t, fixture)
	quick, err := fixture.service.Explore(ctx, fixture.compiled.Pack.CourseID, "after-bc5", DepthQuick)
	if err != nil {
		t.Fatal(err)
	}
	standard, err := fixture.service.Explore(ctx, fixture.compiled.Pack.CourseID, "after-bc5", DepthStandard)
	if err != nil {
		t.Fatal(err)
	}
	reference, err := fixture.service.Explore(ctx, fixture.compiled.Pack.CourseID, "after-bc5", DepthReference)
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(moveIDs(quick.Moves), []string{"white-c3"}) {
		t.Fatalf("quick moves = %v", moveIDs(quick.Moves))
	}
	if !slices.Equal(moveIDs(standard.Moves), []string{"white-c3"}) {
		t.Fatalf("standard moves = %v", moveIDs(standard.Moves))
	}
	if !slices.Equal(moveIDs(reference.Moves), []string{"white-c3", "white-b4"}) {
		t.Fatalf("reference moves = %v", moveIDs(reference.Moves))
	}
	if reference.PositionID != "after-bc5" || reference.FEN == "" ||
		reference.Label != "Giuoco Piano" || reference.Evaluation.Code != EvaluationEqual {
		t.Fatalf("reference position = %+v", reference)
	}
	if len(reference.Notes) != 1 || reference.Notes[0].Kind != "overview" ||
		reference.Notes[0].Text == "" || reference.Notes[0].SourceRef.PrintedPage != 1 ||
		reference.Notes[0].SourceRef.NoteLabel != "overview" {
		t.Fatalf("reference notes = %+v", reference.Notes)
	}
	if reference.Moves[0].SAN != "c3" || reference.Moves[0].Role != RoleRepertoire ||
		reference.Moves[0].VariationName != "Giuoco Piano" ||
		reference.Moves[0].Evaluation.Code != EvaluationEqual ||
		reference.Moves[0].SourceRef.CoverageID != "p1-c3" {
		t.Fatalf("reference first move = %+v", reference.Moves[0])
	}

	transposition, err := fixture.service.Explore(
		ctx, fixture.compiled.Pack.CourseID, "trans-knights", DepthReference,
	)
	if err != nil {
		t.Fatal(err)
	}
	if transposition.IncomingPaths != 2 {
		t.Fatalf("transposition incoming paths = %d, want 2", transposition.IncomingPaths)
	}
	if _, err := fixture.service.Explore(
		ctx, fixture.compiled.Pack.CourseID, "after-bc5", DepthReference,
	); err != nil {
		t.Fatal(err)
	}
	after := openingLearningRowCounts(t, fixture)
	if before != after {
		t.Fatalf("exploration wrote learner state: before=%v after=%v", before, after)
	}
}

func TestExploreRejectsUnknownDepthAndPosition(t *testing.T) {
	ctx := context.Background()
	fixture := newOpeningServiceFixture(t)
	if _, err := fixture.service.Explore(
		ctx, fixture.compiled.Pack.CourseID, "initial", Depth("novel"),
	); err == nil {
		t.Fatal("invalid explorer depth was accepted")
	}
	if _, err := fixture.service.Explore(
		ctx, fixture.compiled.Pack.CourseID, "missing", DepthReference,
	); err == nil {
		t.Fatal("missing explorer position was accepted")
	}
}

func installExplorerTransposition(t *testing.T, fixture openingServiceFixture) {
	t.Helper()
	pack := decodeMiniPack(t)
	pack.ContentVersion = "1.1.0"
	pack.Positions = append(pack.Positions,
		Position{PositionID: "trans-a-nf3", Label: "King knight first", Evaluation: Evaluation{Code: EvaluationEqual}, NoteIDs: []string{}},
		Position{PositionID: "trans-a-nf6", Label: "Both king knights developed", Evaluation: Evaluation{Code: EvaluationEqual}, NoteIDs: []string{}},
		Position{PositionID: "trans-a-nc3", Label: "White knights developed", Evaluation: Evaluation{Code: EvaluationEqual}, NoteIDs: []string{}},
		Position{PositionID: "trans-b-nc3", Label: "Queen knight first", Evaluation: Evaluation{Code: EvaluationEqual}, NoteIDs: []string{}},
		Position{PositionID: "trans-b-nc6", Label: "Both queen knights developed", Evaluation: Evaluation{Code: EvaluationEqual}, NoteIDs: []string{}},
		Position{PositionID: "trans-b-nf3", Label: "White knights developed", Evaluation: Evaluation{Code: EvaluationEqual}, NoteIDs: []string{}},
		Position{PositionID: "trans-knights", Label: "Four Knights transposition", Evaluation: Evaluation{Code: EvaluationEqual}, NoteIDs: []string{}},
	)
	pack.Moves = append(pack.Moves,
		Move{
			MoveID: "trans-a-nf3", FromPositionID: "initial", ToPositionID: "trans-a-nf3",
			UCI: "g1f3", MinimumDepth: DepthReference, TrainingRole: RoleAlternative,
			VariationName: "King knight first", Evaluation: Evaluation{Code: EvaluationEqual}, NoteIDs: []string{},
			SourceRef: SourceRef{PrintedPage: 1, CoverageID: "p1-trans-a-nf3"},
		},
		Move{
			MoveID: "trans-a-nf6", FromPositionID: "trans-a-nf3", ToPositionID: "trans-a-nf6",
			UCI: "g8f6", MinimumDepth: DepthReference, TrainingRole: RoleOpponent,
			VariationName: "King knight first", Evaluation: Evaluation{Code: EvaluationEqual}, NoteIDs: []string{},
			SourceRef: SourceRef{PrintedPage: 1, CoverageID: "p1-trans-a-nf6"},
		},
		Move{
			MoveID: "trans-a-nc3", FromPositionID: "trans-a-nf6", ToPositionID: "trans-a-nc3",
			UCI: "b1c3", MinimumDepth: DepthReference, TrainingRole: RoleAlternative,
			VariationName: "King knight first", Evaluation: Evaluation{Code: EvaluationEqual}, NoteIDs: []string{},
			SourceRef: SourceRef{PrintedPage: 1, CoverageID: "p1-trans-a-nc3"},
		},
		Move{
			MoveID: "trans-a-nc6", FromPositionID: "trans-a-nc3", ToPositionID: "trans-knights",
			UCI: "b8c6", MinimumDepth: DepthReference, TrainingRole: RoleOpponent,
			VariationName: "Four Knights transposition", Evaluation: Evaluation{Code: EvaluationEqual}, NoteIDs: []string{},
			SourceRef: SourceRef{PrintedPage: 1, CoverageID: "p1-trans-a-nc6"},
		},
		Move{
			MoveID: "trans-b-nc3", FromPositionID: "initial", ToPositionID: "trans-b-nc3",
			UCI: "b1c3", MinimumDepth: DepthReference, TrainingRole: RoleAlternative,
			VariationName: "Queen knight first", Evaluation: Evaluation{Code: EvaluationEqual}, NoteIDs: []string{},
			SourceRef: SourceRef{PrintedPage: 1, CoverageID: "p1-trans-b-nc3"},
		},
		Move{
			MoveID: "trans-b-nc6", FromPositionID: "trans-b-nc3", ToPositionID: "trans-b-nc6",
			UCI: "b8c6", MinimumDepth: DepthReference, TrainingRole: RoleOpponent,
			VariationName: "Queen knight first", Evaluation: Evaluation{Code: EvaluationEqual}, NoteIDs: []string{},
			SourceRef: SourceRef{PrintedPage: 1, CoverageID: "p1-trans-b-nc6"},
		},
		Move{
			MoveID: "trans-b-nf3", FromPositionID: "trans-b-nc6", ToPositionID: "trans-b-nf3",
			UCI: "g1f3", MinimumDepth: DepthReference, TrainingRole: RoleAlternative,
			VariationName: "Queen knight first", Evaluation: Evaluation{Code: EvaluationEqual}, NoteIDs: []string{},
			SourceRef: SourceRef{PrintedPage: 1, CoverageID: "p1-trans-b-nf3"},
		},
		Move{
			MoveID: "trans-b-nf6", FromPositionID: "trans-b-nf3", ToPositionID: "trans-knights",
			UCI: "g8f6", MinimumDepth: DepthReference, TrainingRole: RoleOpponent,
			VariationName: "Four Knights transposition", Evaluation: Evaluation{Code: EvaluationEqual}, NoteIDs: []string{},
			SourceRef: SourceRef{PrintedPage: 1, CoverageID: "p1-trans-b-nf6"},
		},
	)
	pack.SourceCoverage.ExpectedReferences = append(
		pack.SourceCoverage.ExpectedReferences,
		"p1-trans-a-nf3", "p1-trans-a-nf6", "p1-trans-a-nc3", "p1-trans-a-nc6",
		"p1-trans-b-nc3", "p1-trans-b-nc6", "p1-trans-b-nf3", "p1-trans-b-nf6",
	)
	compiled, err := Compile(pack, chessrules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.catalog.Replace(context.Background(), compiled, "/private/explorer.ctcourse", "sha-explorer"); err != nil {
		t.Fatal(err)
	}
}

func moveIDs(moves []ExplorerMove) []string {
	ids := make([]string, len(moves))
	for index, move := range moves {
		ids[index] = move.MoveID
	}
	return ids
}

type learningRowCounts struct {
	attempts int
	progress int
	reviews  int
}

func openingLearningRowCounts(t *testing.T, fixture openingServiceFixture) learningRowCounts {
	t.Helper()
	var counts learningRowCounts
	for _, item := range []struct {
		table string
		value *int
	}{
		{table: "opening_attempts", value: &counts.attempts},
		{table: "opening_prompt_progress", value: &counts.progress},
		{table: "opening_review_state", value: &counts.reviews},
	} {
		if err := fixture.userDB.QueryRow(`SELECT COUNT(*) FROM ` + item.table).Scan(item.value); err != nil {
			t.Fatal(err)
		}
	}
	return counts
}
