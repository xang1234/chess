package openings

import (
	"fmt"
	"regexp"
	"strings"
)

var boardSquarePattern = regexp.MustCompile(`^[a-h][1-8]$`)

func RequiredActivityIDs(lesson Lesson) []string {
	ids := make([]string, 0, len(lesson.Activities))
	for _, activity := range lesson.Activities {
		if activity.Required {
			ids = append(ids, activity.ActivityID)
		}
	}
	return ids
}

func (c *courseCompiler) validateActivities() {
	for lessonIndex, lesson := range c.pack.Lessons {
		lessonPath := fmt.Sprintf("lessons[%d]", lessonIndex)
		lessonRank, lessonDepthOK := depthRank(lesson.MinimumDepth)
		chapter, chapterOK := c.compiled.Chapters[lesson.ChapterID]
		if chapterOK && lessonDepthOK {
			chapterRank, chapterDepthOK := depthRank(chapter.MinimumDepth)
			if chapterDepthOK && lessonRank < chapterRank {
				c.addDiagnostic("lesson_depth_before_chapter", lessonPath+".minimumDepth", fmt.Sprintf("lesson depth %q is shallower than chapter depth %q", lesson.MinimumDepth, chapter.MinimumDepth))
			}
		}
		reachable := c.reachableByDepth[lesson.MinimumDepth]
		if lessonDepthOK && reachable != nil && !reachable[lesson.StartPositionID] {
			c.addDiagnostic("lesson_position_hidden", lessonPath+".startPositionId", fmt.Sprintf("start position %q is hidden at %s depth", lesson.StartPositionID, lesson.MinimumDepth))
		}

		required := 0
		decisionKeys := map[string]string{}
		for activityIndex, activity := range lesson.Activities {
			activityPath := fmt.Sprintf("%s.activities[%d]", lessonPath, activityIndex)
			c.registerID(c.activityPaths, activity.ActivityID, activityPath+".activityId")
			c.validateText(activityPath+".title", activity.Title, true)
			c.validateText(activityPath+".instruction", activity.Instruction, true)
			if activity.Required {
				required++
			}
			c.validateActivityPosition(activity, activityPath, reachable, lessonDepthOK, lesson.MinimumDepth)
			c.validateActivityNotes(activity, activityPath)
			c.validateActivityAnnotations(activity, activityPath)
			c.validateActivityKind(lesson, activity, activityPath, lessonRank, lessonDepthOK)

			if c.pack.SchemaVersion == 2 && activity.Required && activity.Kind == ActivityDecision {
				prompt, ok := c.compiled.Prompts[activity.PromptID]
				if !ok {
					continue
				}
				key := prompt.PositionID + "\x00" + prompt.PrimaryMoveID
				if first, duplicate := decisionKeys[key]; duplicate {
					c.addDiagnostic("duplicate_lesson_decision", activityPath+".promptId", fmt.Sprintf("decision repeats %s", first))
				} else {
					decisionKeys[key] = activityPath
				}
			}
		}
		if required == 0 {
			c.addDiagnostic("missing_required_activity", lessonPath+".activities", "lesson requires at least one required activity")
		}
	}
}

func (c *courseCompiler) validateActivityPosition(
	activity LessonActivity,
	path string,
	reachable map[string]bool,
	lessonDepthOK bool,
	lessonDepth Depth,
) {
	requiresPosition := activity.Kind == ActivityDemonstration || activity.Kind == ActivityDecision || activity.Kind == ActivityComparison
	if strings.TrimSpace(activity.PositionID) == "" {
		if requiresPosition {
			c.addDiagnostic("missing_position", path+".positionId", fmt.Sprintf("%s activity requires a position", activity.Kind))
		}
		return
	}
	if _, ok := c.compiled.Positions[activity.PositionID]; !ok {
		c.addDiagnostic("missing_reference", path+".positionId", fmt.Sprintf("position %q does not exist", activity.PositionID))
	} else if lessonDepthOK && reachable != nil && !reachable[activity.PositionID] {
		c.addDiagnostic("lesson_position_hidden", path+".positionId", fmt.Sprintf("position %q is hidden at %s depth", activity.PositionID, lessonDepth))
	}
}

func (c *courseCompiler) validateActivityNotes(activity LessonActivity, path string) {
	for noteIndex, noteID := range activity.NoteIDs {
		if _, ok := c.compiled.Notes[noteID]; !ok {
			c.addDiagnostic("missing_reference", fmt.Sprintf("%s.noteIds[%d]", path, noteIndex), fmt.Sprintf("note %q does not exist", noteID))
		}
	}
}

func (c *courseCompiler) validateActivityAnnotations(activity LessonActivity, path string) {
	for annotationIndex, annotation := range activity.Annotations {
		annotationPath := fmt.Sprintf("%s.annotations[%d]", path, annotationIndex)
		c.validateText(annotationPath+".label", annotation.Label, false)
		switch annotation.Kind {
		case "square":
			if !boardSquarePattern.MatchString(annotation.From) || annotation.To != "" {
				c.addDiagnostic("invalid_annotation", annotationPath, "square annotation requires one valid square")
			}
		case "arrow":
			if !boardSquarePattern.MatchString(annotation.From) || !boardSquarePattern.MatchString(annotation.To) || annotation.From == annotation.To {
				c.addDiagnostic("invalid_annotation", annotationPath, "arrow annotation requires two distinct valid squares")
			}
		default:
			c.addDiagnostic("invalid_annotation", annotationPath+".kind", fmt.Sprintf("unsupported annotation kind %q", annotation.Kind))
		}
	}
}

func (c *courseCompiler) validateActivityKind(
	lesson Lesson,
	activity LessonActivity,
	path string,
	lessonRank int,
	lessonDepthOK bool,
) {
	switch activity.Kind {
	case ActivityConcept, ActivityRecap:
		c.rejectActivityPromptMovesAndComparison(activity, path)
	case ActivityReference:
		if activity.Required {
			c.addDiagnostic("reference_required", path+".required", "reference activity must be optional")
		}
		c.rejectActivityPromptMovesAndComparison(activity, path)
	case ActivityDemonstration:
		if activity.PromptID != "" {
			c.addDiagnostic("unexpected_prompt", path+".promptId", "demonstration activity cannot have a prompt")
		}
		if len(activity.Comparison) != 0 {
			c.addDiagnostic("unexpected_comparison", path+".comparison", "demonstration activity cannot have comparison lines")
		}
		if len(activity.MoveIDs) == 0 {
			c.addDiagnostic("missing_demonstration_move", path+".moveIds", "demonstration activity requires at least one move")
		}
		c.validateConnectedActivityMoves(lesson, activity.PositionID, activity.MoveIDs, path+".moveIds", "disconnected_demonstration_path", lessonRank, lessonDepthOK)
	case ActivityDecision:
		if len(activity.Comparison) != 0 {
			c.addDiagnostic("unexpected_comparison", path+".comparison", "decision activity cannot have comparison lines")
		}
		c.validateDecisionActivity(lesson, activity, path, lessonRank, lessonDepthOK)
	case ActivityComparison:
		if activity.PromptID != "" {
			c.addDiagnostic("unexpected_prompt", path+".promptId", "comparison activity cannot have a prompt")
		}
		if len(activity.MoveIDs) != 0 {
			c.addDiagnostic("unexpected_move", path+".moveIds", "comparison activity stores moves in labeled lines")
		}
		if len(activity.Comparison) < 2 {
			c.addDiagnostic("comparison_lines", path+".comparison", "comparison activity requires at least two labeled lines")
		}
		for lineIndex, line := range activity.Comparison {
			linePath := fmt.Sprintf("%s.comparison[%d]", path, lineIndex)
			c.validateText(linePath+".label", line.Label, true)
			if len(line.MoveIDs) == 0 {
				c.addDiagnostic("comparison_lines", linePath+".moveIds", "comparison line requires at least one move")
			}
			c.validateConnectedActivityMoves(lesson, activity.PositionID, line.MoveIDs, linePath+".moveIds", "disconnected_comparison_path", lessonRank, lessonDepthOK)
		}
	default:
		c.addDiagnostic("invalid_activity_kind", path+".kind", fmt.Sprintf("unsupported activity kind %q", activity.Kind))
	}
}

func (c *courseCompiler) rejectActivityPromptMovesAndComparison(activity LessonActivity, path string) {
	if activity.PromptID != "" {
		c.addDiagnostic("unexpected_prompt", path+".promptId", fmt.Sprintf("%s activity cannot have a prompt", activity.Kind))
	}
	if len(activity.MoveIDs) != 0 {
		c.addDiagnostic("unexpected_move", path+".moveIds", fmt.Sprintf("%s activity cannot contain moves", activity.Kind))
	}
	if len(activity.Comparison) != 0 {
		c.addDiagnostic("unexpected_comparison", path+".comparison", fmt.Sprintf("%s activity cannot have comparison lines", activity.Kind))
	}
}

func (c *courseCompiler) validateDecisionActivity(
	lesson Lesson,
	activity LessonActivity,
	path string,
	lessonRank int,
	lessonDepthOK bool,
) {
	if activity.PromptID == "" {
		c.addDiagnostic("missing_prompt", path+".promptId", "decision activity requires a prompt")
		return
	}
	prompt, ok := c.compiled.Prompts[activity.PromptID]
	if !ok {
		c.addDiagnostic("missing_reference", path+".promptId", fmt.Sprintf("prompt %q does not exist", activity.PromptID))
		return
	}
	if prompt.PositionID != activity.PositionID {
		c.addDiagnostic("prompt_activity_position", path+".positionId", fmt.Sprintf("activity position %q differs from prompt position %q", activity.PositionID, prompt.PositionID))
	}
	primary, primaryOK := c.compiled.Moves[prompt.PrimaryMoveID]
	if primaryOK {
		c.validateLessonMoveDepth(primary, path+".promptId", lesson.MinimumDepth, lessonRank, lessonDepthOK)
	}
	if len(activity.MoveIDs) > 1 {
		c.addDiagnostic("automatic_continuation_length", path+".moveIds", "decision activity allows at most one automatic opponent move")
	}
	if len(activity.MoveIDs) == 1 {
		continuation, continuationOK := c.compiled.Moves[activity.MoveIDs[0]]
		movePath := path + ".moveIds[0]"
		if !continuationOK {
			c.addDiagnostic("missing_reference", movePath, fmt.Sprintf("move %q does not exist", activity.MoveIDs[0]))
		} else if primaryOK {
			if continuation.FromPositionID != primary.ToPositionID {
				c.addDiagnostic("automatic_continuation_position", movePath, fmt.Sprintf("automatic move leaves %q, want %q", continuation.FromPositionID, primary.ToPositionID))
			}
			if continuation.TrainingRole != RoleOpponent {
				c.addDiagnostic("automatic_continuation_role", movePath, "automatic move must have opponent role")
			}
			c.validateLessonMoveDepth(continuation, movePath, lesson.MinimumDepth, lessonRank, lessonDepthOK)
		}
	}
}

func (c *courseCompiler) validateConnectedActivityMoves(
	lesson Lesson,
	positionID string,
	moveIDs []string,
	path string,
	diagnosticCode string,
	lessonRank int,
	lessonDepthOK bool,
) {
	for moveIndex, moveID := range moveIDs {
		movePath := fmt.Sprintf("%s[%d]", path, moveIndex)
		move, ok := c.compiled.Moves[moveID]
		if !ok {
			c.addDiagnostic("missing_reference", movePath, fmt.Sprintf("move %q does not exist", moveID))
			continue
		}
		if move.FromPositionID != positionID {
			c.addDiagnostic(diagnosticCode, movePath, fmt.Sprintf("move %q leaves %q, want %q", moveID, move.FromPositionID, positionID))
		}
		positionID = move.ToPositionID
		c.validateLessonMoveDepth(move, movePath, lesson.MinimumDepth, lessonRank, lessonDepthOK)
	}
}

func (c *courseCompiler) validateLessonMoveDepth(
	move CompiledMove,
	path string,
	lessonDepth Depth,
	lessonRank int,
	lessonDepthOK bool,
) {
	minimumRank, moveDepthOK := depthRank(move.MinimumDepth)
	if lessonDepthOK && moveDepthOK && minimumRank > lessonRank {
		c.addDiagnostic("lesson_move_hidden", path, fmt.Sprintf("move %q is hidden at lesson depth %s", move.MoveID, lessonDepth))
	}
}
