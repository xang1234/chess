package openings

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"chess-trainer/internal/chessrules"
)

func TestBuildActivityViewSeparatesTeachingNotesFromReferenceNotes(t *testing.T) {
	course := CompiledCourse{
		Pack: CoursePack{
			CourseID: "course", Perspective: PerspectiveWhite, RootPositionID: "position",
			RootFEN: "reference-fen",
		},
		Positions: map[string]CompiledPosition{
			"position": {
				ID:      "position",
				FEN:     "reference-fen",
				NoteIDs: []string{"reference-note", "teaching-copy"},
			},
		},
		Notes: map[string]Note{
			"teaching-note":  {NoteID: "teaching-note", Text: "Keep this teaching note visible."},
			"teaching-copy":  {NoteID: "teaching-copy", Text: "Keep this teaching note visible."},
			"reference-note": {NoteID: "reference-note", Text: "Keep this detailed note in the reference section."},
		},
	}
	activity := LessonActivity{
		ActivityID: "concept", Kind: ActivityConcept, Required: true, PositionID: "position",
		Title: "Plan", Instruction: "Learn the plan.", NoteIDs: []string{"teaching-note"},
	}
	lesson := Lesson{LessonID: "lesson", Activities: []LessonActivity{activity}}
	db := openOpeningUserTestDB(t)
	store := NewUserStore(db)
	if err := store.RecordActivityProgress(context.Background(), ActivityProgressUpdate{
		CourseID: "course", LessonID: "lesson", CompletedActivityID: "concept",
		RequiredActivityIDs: []string{"concept"}, Now: time.Unix(1, 0).UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	session := StoredSession{
		CourseID: "course", LessonID: "lesson", Depth: DepthQuick,
		State: SessionState{Position: PositionState{PositionID: "position", CurrentFEN: "current-fen"}},
	}

	view, err := (&Service{store: store, rules: chessrules.Rules{}}).lessonActivityView(
		context.Background(), course, lesson, session,
	)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"Keep this teaching note visible."}; !reflect.DeepEqual(view.TeachingNoteTexts, want) {
		t.Fatalf("teaching notes = %v, want %v", view.TeachingNoteTexts, want)
	}

	encoded, err := json.Marshal(view)
	if err != nil {
		t.Fatal(err)
	}
	var contract struct {
		ReferenceNoteTexts []string `json:"referenceNoteTexts"`
	}
	if err := json.Unmarshal(encoded, &contract); err != nil {
		t.Fatal(err)
	}
	if want := []string{"Keep this detailed note in the reference section."}; !reflect.DeepEqual(contract.ReferenceNoteTexts, want) {
		t.Fatalf("reference notes = %v, want %v", contract.ReferenceNoteTexts, want)
	}
}
