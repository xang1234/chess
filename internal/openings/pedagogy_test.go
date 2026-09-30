package openings

import (
	"strings"
	"testing"

	"chess-trainer/internal/chessrules"
)

func pedagogyWarningsFor(t *testing.T, mutate func(*CoursePack)) []Diagnostic {
	t.Helper()
	pack := decodeTreePack(t)
	if mutate != nil {
		mutate(&pack)
	}
	compiled, err := Compile(pack, chessrules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	return PedagogyWarnings(compiled)
}

func hasWarning(warnings []Diagnostic, code, path string) bool {
	for _, warning := range warnings {
		if warning.Code == code && warning.Path == path {
			return true
		}
	}
	return false
}

func TestPedagogyWarningsFlagConceptWithoutAnnotation(t *testing.T) {
	warnings := pedagogyWarningsFor(t, nil)
	if !hasWarning(warnings, "concept_without_annotation", "lessons[0].activities[0].annotations") {
		t.Fatalf("warnings = %#v", warnings)
	}

	warnings = pedagogyWarningsFor(t, func(pack *CoursePack) {
		pack.Lessons[0].Activities[0].Annotations = []BoardAnnotation{{Kind: "arrow", From: "c2", To: "c3"}}
	})
	if len(warnings) != 0 {
		t.Fatalf("annotated tree fixture should be clean, got %#v", warnings)
	}
}

func TestPedagogyWarningsFlagLessonWithoutDecision(t *testing.T) {
	warnings := pedagogyWarningsFor(t, func(pack *CoursePack) {
		lesson := &pack.Lessons[0]
		lesson.Activities = append(lesson.Activities[:1], lesson.Activities[2:]...)
	})
	if !hasWarning(warnings, "lesson_without_decision", "lessons[0].activities") {
		t.Fatalf("warnings = %#v", warnings)
	}
}

func TestPedagogyWarningsFlagLongInstructionExceptReference(t *testing.T) {
	long := strings.TrimSpace(strings.Repeat("word ", 21))
	warnings := pedagogyWarningsFor(t, func(pack *CoursePack) {
		pack.Lessons[0].Activities[1].Instruction = long
		pack.Lessons[0].Activities = append(pack.Lessons[0].Activities, LessonActivity{
			ActivityID: "giuoco-reference", Kind: ActivityReference, Title: "Deeper analysis",
			Instruction: long, PositionID: "after-c3", NoteIDs: []string{}, MoveIDs: []string{},
		})
	})
	if !hasWarning(warnings, "long_instruction", "lessons[0].activities[1].instruction") {
		t.Fatalf("warnings = %#v", warnings)
	}
	if hasWarning(warnings, "long_instruction", "lessons[0].activities[3].instruction") {
		t.Fatalf("reference activity must be exempt: %#v", warnings)
	}
}

func TestPedagogyWarningsFlagDemonstrationProblems(t *testing.T) {
	warnings := pedagogyWarningsFor(t, func(pack *CoursePack) {
		lesson := &pack.Lessons[0]
		prompt := lesson.Activities[1].PromptID
		var primary string
		for _, candidate := range pack.Prompts {
			if candidate.PromptID == prompt {
				primary = candidate.PrimaryMoveID
			}
		}
		demonstration := LessonActivity{
			ActivityID: "giuoco-watch", Kind: ActivityDemonstration, Title: "Watch",
			Instruction: "Watch the move.", Required: true,
			PositionID: lesson.Activities[1].PositionID, NoteIDs: []string{}, MoveIDs: []string{primary},
		}
		lesson.Activities = append([]LessonActivity{demonstration}, lesson.Activities...)
	})
	if !hasWarning(warnings, "decision_answer_demonstrated", "lessons[0].activities[2].promptId") {
		t.Fatalf("warnings = %#v", warnings)
	}
	if hasWarning(warnings, "long_demonstration", "lessons[0].activities[0].moveIds") {
		t.Fatalf("one-move demonstration is not long: %#v", warnings)
	}
}

func TestLongDemonstrationThreshold(t *testing.T) {
	if isLongDemonstration(LessonActivity{Kind: ActivityDemonstration, MoveIDs: make([]string, 4)}) {
		t.Fatal("four plies should be allowed")
	}
	if !isLongDemonstration(LessonActivity{Kind: ActivityDemonstration, MoveIDs: make([]string, 5)}) {
		t.Fatal("five plies should be flagged")
	}
}
