package openings

import (
	"bytes"
	"errors"
	"os"
	"slices"
	"testing"

	"chess-trainer/internal/chessrules"
)

func decodeMiniPack(t *testing.T) CoursePack {
	t.Helper()
	pack, err := DecodeCoursePack(bytes.NewReader(readMiniCoursePack(t)))
	if err != nil {
		t.Fatal(err)
	}
	return pack
}

func TestCompileDerivesLegalGraphDepthsAndPromptFingerprints(t *testing.T) {
	compiled, err := Compile(decodeMiniPack(t), chessrules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	if got := compiled.Moves["white-c3"].SAN; got != "c3" {
		t.Fatalf("SAN = %q, want c3", got)
	}
	if compiled.Prompts["recall-c3"].SemanticFingerprint == "" {
		t.Fatal("prompt fingerprint is empty")
	}
	quick := compiled.VisibleMoves("after-bc5", DepthQuick)
	if len(quick) != 1 || quick[0].MoveID != "white-c3" {
		t.Fatalf("Quick moves = %+v, want white-c3", quick)
	}
	reference := compiled.VisibleMoves("after-bc5", DepthReference)
	if len(reference) != 2 || reference[0].MoveID != "white-c3" || reference[1].MoveID != "white-b4" {
		t.Fatalf("Reference moves = %+v", reference)
	}
	if got := compiled.Positions["after-c3"].FEN; got == "" || got == compiled.Pack.RootFEN {
		t.Fatalf("derived FEN = %q", got)
	}
	if !slices.Equal(compiled.Outgoing["after-bc5"], []string{"white-c3", "white-b4"}) {
		t.Fatalf("outgoing order = %v", compiled.Outgoing["after-bc5"])
	}
}

func TestCompileAllowsAlternativeBranchMoveAsPromptPrimary(t *testing.T) {
	pack := decodeMiniPack(t)
	for index := range pack.Moves {
		if pack.Moves[index].MoveID == "white-b4" {
			pack.Moves[index].MinimumDepth = DepthQuick
		}
	}
	pack.Prompts[0].PrimaryMoveID = "white-b4"
	pack.Prompts[0].AcceptedAlternativeMoveIDs = []string{}

	compiled, err := Compile(pack, chessrules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	if compiled.Prompts["recall-c3"].PrimaryMoveID != "white-b4" {
		t.Fatalf("primary move = %q", compiled.Prompts["recall-c3"].PrimaryMoveID)
	}
}

func TestCompileAcceptsBlackPerspectiveRepertoireMoves(t *testing.T) {
	contents, err := os.ReadFile("testdata/black_tree.ctcourse")
	if err != nil {
		t.Fatal(err)
	}
	pack, err := DecodeCoursePack(bytes.NewReader(contents))
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := Compile(pack, chessrules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	if compiled.Pack.Perspective != PerspectiveBlack {
		t.Fatalf("perspective = %q", compiled.Pack.Perspective)
	}
	if got := compiled.Moves["black-c6"].SAN; got != "c6" {
		t.Fatalf("black-c6 SAN = %q, want c6", got)
	}
	if got := compiled.Prompts["prompt-caro-c6"].PrimaryMoveID; got != "black-c6" {
		t.Fatalf("primary move = %q", got)
	}
}

func TestCompileRejectsInvalidGraphMutations(t *testing.T) {
	tests := []struct {
		name   string
		code   string
		mutate func(*CoursePack)
	}{
		{
			name: "invalid course ID", code: "invalid_id",
			mutate: func(pack *CoursePack) { pack.CourseID = "Italian Course" },
		},
		{
			name: "duplicate position ID", code: "duplicate_id",
			mutate: func(pack *CoursePack) { pack.Positions = append(pack.Positions, pack.Positions[0]) },
		},
		{
			name: "illegal UCI", code: "illegal_move",
			mutate: func(pack *CoursePack) { pack.Moves[0].UCI = "e2e5" },
		},
		{
			name: "unreachable position", code: "unreachable_position",
			mutate: func(pack *CoursePack) {
				pack.Positions = append(pack.Positions, Position{PositionID: "orphan", Evaluation: Evaluation{Code: EvaluationNone}, NoteIDs: []string{}})
			},
		},
		{
			name: "cycle", code: "cycle",
			mutate: func(pack *CoursePack) {
				pack.Moves = append(pack.Moves, Move{
					MoveID: "black-cycle", FromPositionID: "after-c3", ToPositionID: "initial",
					UCI: "g8f6", MinimumDepth: DepthQuick, TrainingRole: RoleOpponent,
					Evaluation: Evaluation{Code: EvaluationNone}, NoteIDs: []string{},
					SourceRef: SourceRef{PrintedPage: 1, CoverageID: "p1-cycle"},
				})
				pack.SourceCoverage.ExpectedReferences = append(pack.SourceCoverage.ExpectedReferences, "p1-cycle")
			},
		},
		{
			name: "shallower child depends on deeper parent", code: "depth_dependency",
			mutate: func(pack *CoursePack) { pack.Moves[5].MinimumDepth = DepthStandard },
		},
		{
			name: "prompt primary leaves another position", code: "prompt_primary_position",
			mutate: func(pack *CoursePack) { pack.Prompts[0].PrimaryMoveID = "white-d3" },
		},
		{
			name: "inconsistent transposition", code: "inconsistent_transposition",
			mutate: func(pack *CoursePack) { pack.Moves[9].ToPositionID = "after-c3" },
		},
		{
			name: "missing coverage", code: "missing_coverage",
			mutate: func(pack *CoursePack) {
				pack.SourceCoverage.ExpectedReferences = append(pack.SourceCoverage.ExpectedReferences, "p1-not-entered")
			},
		},
		{
			name: "unexpected coverage", code: "unexpected_coverage",
			mutate: func(pack *CoursePack) {
				pack.SourceCoverage.ExpectedReferences = slices.Delete(pack.SourceCoverage.ExpectedReferences, 10, 11)
			},
		},
		{
			name: "role disagrees with side to move", code: "role_perspective",
			mutate: func(pack *CoursePack) { pack.Moves[0].TrainingRole = RoleOpponent },
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pack := decodeMiniPack(t)
			test.mutate(&pack)
			_, err := Compile(pack, chessrules.Rules{})
			var validation *ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("Compile() error = %T %v, want ValidationError", err, err)
			}
			if !hasDiagnosticCode(validation.Diagnostics, test.code) {
				t.Fatalf("diagnostics = %+v, missing code %q", validation.Diagnostics, test.code)
			}
		})
	}
}

func hasDiagnosticCode(diagnostics []Diagnostic, code string) bool {
	return slices.ContainsFunc(diagnostics, func(diagnostic Diagnostic) bool {
		return diagnostic.Code == code
	})
}

func TestCanonicalPositionKeepsOnlyPositionIdentityFields(t *testing.T) {
	got, err := CanonicalPosition("8/8/8/8/8/8/8/K6k w - - 14 27")
	if err != nil {
		t.Fatal(err)
	}
	if got != "8/8/8/8/8/8/8/K6k w - -" {
		t.Fatalf("CanonicalPosition() = %q", got)
	}
	if _, err := CanonicalPosition("not-a-fen"); err == nil {
		t.Fatal("CanonicalPosition() accepted a malformed FEN")
	}
}
