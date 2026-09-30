package openings

import (
	"fmt"
	"strings"
)

const (
	maximumInstructionWords   = 20
	maximumDemonstrationPlies = 4
)

// PedagogyWarnings reports advisory teaching-quality findings for a course
// that already compiled. They never block validation or import.
func PedagogyWarnings(compiled CompiledCourse) []Diagnostic {
	warnings := []Diagnostic{}
	warn := func(code, path, message string) {
		warnings = append(warnings, Diagnostic{Code: code, Path: path, Message: message})
	}
	for lessonIndex, lesson := range compiled.Pack.Lessons {
		lessonPath := fmt.Sprintf("lessons[%d]", lessonIndex)
		name := fmt.Sprintf("%s lesson %q", lesson.MinimumDepth, lesson.LessonID)
		demonstrated := map[string]bool{}
		hasDecision := false
		for activityIndex, activity := range lesson.Activities {
			path := fmt.Sprintf("%s.activities[%d]", lessonPath, activityIndex)
			if activity.Kind != ActivityReference && len(strings.Fields(activity.Instruction)) > maximumInstructionWords {
				warn("long_instruction", path+".instruction", fmt.Sprintf(
					"%s: instruction is longer than %d words", name, maximumInstructionWords))
			}
			switch activity.Kind {
			case ActivityConcept:
				if len(activity.Annotations) == 0 {
					warn("concept_without_annotation", path+".annotations", name+": concept shows no arrow or square on the board")
				}
			case ActivityDemonstration:
				if isLongDemonstration(activity) {
					warn("long_demonstration", path+".moveIds", fmt.Sprintf(
						"%s: demonstration plays more than %d plies", name, maximumDemonstrationPlies))
				}
				for _, moveID := range activity.MoveIDs {
					demonstrated[moveID] = true
				}
			case ActivityDecision:
				hasDecision = true
				if demonstrated[compiled.Prompts[activity.PromptID].PrimaryMoveID] {
					warn("decision_answer_demonstrated", path+".promptId", name+": decision asks for a move an earlier demonstration already showed")
				}
			}
		}
		if !hasDecision {
			warn("lesson_without_decision", lessonPath+".activities", name+": learner never moves a piece")
		}
	}
	return warnings
}

func isLongDemonstration(activity LessonActivity) bool {
	return activity.Kind == ActivityDemonstration && len(activity.MoveIDs) > maximumDemonstrationPlies
}
