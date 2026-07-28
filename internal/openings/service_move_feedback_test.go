package openings

import (
	"context"
	"testing"

	"chess-trainer/internal/chessrules"
)

func TestPlayMoveExplainsVisibleAcceptedAlternative(t *testing.T) {
	fixture := newOpeningServiceFixture(t)
	started := advanceFixtureToDecision(t, &fixture)
	result, err := fixture.service.PlayMove(
		context.Background(), started.SessionID, "b2b4",
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Feedback != FeedbackAlternative || result.FeedbackDetail == nil ||
		result.FeedbackDetail.Coverage != FeedbackCoverageVisible ||
		result.FeedbackDetail.PlayedSAN != "b4" ||
		result.FeedbackDetail.VariationName != "Evans idea" ||
		result.FeedbackDetail.ExpectedSAN != "c3" {
		t.Fatalf("alternative feedback = %+v", result)
	}
	stored, err := fixture.store.LoadSession(context.Background(), started.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.State.Attempt == nil || stored.State.Attempt.AlternativesTried != 1 ||
		stored.State.Attempt.IncorrectMoves != 0 {
		t.Fatalf("alternative attempt = %+v", stored.State.Attempt)
	}
}

func TestPlayMoveExplainsVisibleAuthoredMoveThatIsNotAccepted(t *testing.T) {
	fixture := newOpeningServiceFixture(t)
	pack := decodeMiniPack(t)
	pack.ContentVersion = "1.1.0"
	pack.Positions = append(pack.Positions, Position{
		PositionID: "after-giuoco-d3",
		Label:      "Quiet Giuoco",
		Evaluation: Evaluation{Code: EvaluationEqual},
		NoteIDs:    []string{},
	})
	pack.Notes = append(pack.Notes,
		Note{NoteID: "d3-warning", Kind: "warning", Text: "Do not block the centre too early.", SourceRef: SourceRef{PrintedPage: 1, CoverageID: "p1-d3-warning"}},
		Note{NoteID: "d3-plan", Kind: "plan", Text: "Develop before opening the centre.", SourceRef: SourceRef{PrintedPage: 1, CoverageID: "p1-d3-plan"}},
		Note{NoteID: "d3-explanation", Kind: "explanation", Text: "A quieter way to support e4.", SourceRef: SourceRef{PrintedPage: 1, CoverageID: "p1-d3-explanation"}},
	)
	pack.Moves = append(pack.Moves, Move{
		MoveID:         "white-d3-giuoco",
		FromPositionID: "after-bc5",
		ToPositionID:   "after-giuoco-d3",
		UCI:            "d2d3",
		MinimumDepth:   DepthQuick,
		TrainingRole:   RoleAlternative,
		VariationName:  "Quiet Giuoco",
		Evaluation:     Evaluation{Code: EvaluationEqual},
		NoteIDs:        []string{"d3-warning", "d3-plan", "d3-explanation"},
		SourceRef:      SourceRef{PrintedPage: 1, CoverageID: "p1-d3-giuoco"},
	})
	pack.SourceCoverage.ExpectedReferences = append(
		pack.SourceCoverage.ExpectedReferences,
		"p1-d3-warning", "p1-d3-plan", "p1-d3-explanation", "p1-d3-giuoco",
	)
	compiled, err := Compile(pack, chessrules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.catalog.Replace(
		context.Background(), compiled, "/private/feedback.ctcourse", "sha-feedback",
	); err != nil {
		t.Fatal(err)
	}

	started := advanceFixtureToDecision(t, &fixture)
	result, err := fixture.service.PlayMove(
		context.Background(), started.SessionID, "d2d3",
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.Feedback != FeedbackOffCourse || result.FeedbackDetail == nil ||
		result.FeedbackDetail.Coverage != FeedbackCoverageVisible ||
		result.FeedbackDetail.VariationName != "Quiet Giuoco" ||
		result.FeedbackDetail.NoteText != "A quieter way to support e4." {
		t.Fatalf("visible authored feedback = %+v", result)
	}
	stored, err := fixture.store.LoadSession(context.Background(), started.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.State.Attempt == nil || stored.State.Attempt.IncorrectMoves != 1 ||
		stored.State.Attempt.AlternativesTried != 0 {
		t.Fatalf("visible authored attempt = %+v", stored.State.Attempt)
	}
}

func TestPlayMoveMarksHiddenAuthoredMoveCoveredLater(t *testing.T) {
	fixture := newOpeningServiceFixture(t)
	ctx := context.Background()
	if err := fixture.service.SetDepth(
		ctx, fixture.compiled.Pack.CourseID, DepthQuick,
	); err != nil {
		t.Fatal(err)
	}
	started := advanceFixtureToDecision(t, &fixture)
	result, err := fixture.service.PlayMove(ctx, started.SessionID, "b2b4")
	if err != nil {
		t.Fatal(err)
	}
	if result.Feedback != FeedbackOffCourse || result.FeedbackDetail == nil ||
		result.FeedbackDetail.Coverage != FeedbackCoverageDeeper ||
		result.FeedbackDetail.RequiredDepth != DepthReference {
		t.Fatalf("deeper feedback = %+v", result)
	}
}

func TestPlayMoveExplainsUnknownLegalMoveWithoutInventingMetadata(t *testing.T) {
	fixture := newOpeningServiceFixture(t)
	started := advanceFixtureToDecision(t, &fixture)
	result, err := fixture.service.PlayMove(
		context.Background(), started.SessionID, "d2d3",
	)
	if err != nil {
		t.Fatal(err)
	}
	if result.FeedbackDetail == nil ||
		result.FeedbackDetail.Coverage != FeedbackCoverageUnknown ||
		result.FeedbackDetail.PlayedSAN != "d3" ||
		result.FeedbackDetail.VariationName != "" ||
		result.FeedbackDetail.NoteText != "" ||
		result.FeedbackDetail.Evaluation.Code != EvaluationNone {
		t.Fatalf("unknown feedback = %+v", result.FeedbackDetail)
	}
}

func advanceFixtureToDecision(
	t *testing.T,
	fixture *openingServiceFixture,
) OpeningSessionView {
	t.Helper()
	ctx := context.Background()
	started, err := fixture.service.StartLesson(
		ctx, fixture.compiled.Pack.CourseID, "giuoco-c3",
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.service.Advance(ctx, started.SessionID); err != nil {
		t.Fatal(err)
	}
	result, err := fixture.service.Advance(ctx, started.SessionID)
	if err != nil {
		t.Fatal(err)
	}
	return result.Session
}
