package openings

import "strings"

func lessonActivityByID(lesson Lesson, activityID string) (int, LessonActivity, bool) {
	for index, activity := range lesson.Activities {
		if activity.ActivityID == activityID {
			return index, activity, true
		}
	}
	return 0, LessonActivity{}, false
}

func activityCompatible(
	oldCourse CompiledCourse,
	oldActivity LessonActivity,
	newCourse CompiledCourse,
	newActivity LessonActivity,
) bool {
	if oldActivity.Kind != newActivity.Kind ||
		!sameActivityPosition(oldCourse, oldActivity, newCourse, newActivity) {
		return false
	}
	if oldActivity.Kind == ActivityDecision {
		oldPrompt, oldExists := oldCourse.Prompts[oldActivity.PromptID]
		newPrompt, newExists := newCourse.Prompts[newActivity.PromptID]
		if !oldExists || !newExists || oldPrompt.SemanticFingerprint != newPrompt.SemanticFingerprint {
			return false
		}
	}
	if oldActivity.Kind == ActivityDemonstration || oldActivity.Kind == ActivityDecision {
		return sameActivityMoveIDs(oldCourse, oldActivity.MoveIDs, newCourse, newActivity.MoveIDs)
	}
	return true
}

func sameActivityPosition(
	oldCourse CompiledCourse,
	oldActivity LessonActivity,
	newCourse CompiledCourse,
	newActivity LessonActivity,
) bool {
	if oldActivity.PositionID == "" || newActivity.PositionID == "" {
		return oldActivity.PositionID == newActivity.PositionID
	}
	return sameMovePosition(
		oldCourse, oldActivity.PositionID,
		newCourse, newActivity.PositionID,
	)
}

func activityStatePositionCompatible(
	oldCourse CompiledCourse,
	oldActivity LessonActivity,
	newCourse CompiledCourse,
	newActivity LessonActivity,
	statePositionID string,
) bool {
	if oldActivity.PositionID != "" || newActivity.PositionID != "" {
		return true
	}
	return sameMovePosition(oldCourse, statePositionID, newCourse, statePositionID)
}

func sameActivityMoveIDs(
	oldCourse CompiledCourse,
	oldMoveIDs []string,
	newCourse CompiledCourse,
	newMoveIDs []string,
) bool {
	if len(oldMoveIDs) != len(newMoveIDs) {
		return false
	}
	for index, oldMoveID := range oldMoveIDs {
		newMoveID := newMoveIDs[index]
		if oldMoveID != newMoveID || !playedMovesCompatible(oldCourse, newCourse, []string{oldMoveID}) {
			return false
		}
	}
	return true
}

func playedMovesCompatible(oldCourse CompiledCourse, newCourse CompiledCourse, moveIDs []string) bool {
	for _, moveID := range moveIDs {
		oldMove, oldExists := oldCourse.Moves[moveID]
		newMove, newExists := newCourse.Moves[moveID]
		if !oldExists || !newExists || oldMove.UCI != newMove.UCI ||
			!sameMovePosition(oldCourse, oldMove.FromPositionID, newCourse, newMove.FromPositionID) ||
			!sameMovePosition(oldCourse, oldMove.ToPositionID, newCourse, newMove.ToPositionID) {
			return false
		}
	}
	return true
}

func sameMovePosition(oldCourse CompiledCourse, oldID string, newCourse CompiledCourse, newID string) bool {
	oldPosition, oldExists := oldCourse.Positions[oldID]
	newPosition, newExists := newCourse.Positions[newID]
	if !oldExists || !newExists {
		return false
	}
	oldCanonical, oldErr := CanonicalPosition(oldPosition.FEN)
	newCanonical, newErr := CanonicalPosition(newPosition.FEN)
	return oldErr == nil && newErr == nil && oldCanonical == newCanonical
}

func compatibleActivityCheckpoint(
	oldCourse CompiledCourse,
	oldLesson Lesson,
	newCourse CompiledCourse,
	newLesson Lesson,
	currentIndex int,
) *RestartCheckpoint {
	if strings.TrimSpace(oldLesson.LessonID) == "" || strings.TrimSpace(newLesson.LessonID) == "" {
		return nil
	}
	if currentIndex >= len(oldLesson.Activities) {
		currentIndex = len(oldLesson.Activities) - 1
	}
	for index := currentIndex; index >= 0; index-- {
		oldActivity := oldLesson.Activities[index]
		if !oldActivity.Required {
			continue
		}
		newIndex, newActivity, exists := lessonActivityByID(newLesson, oldActivity.ActivityID)
		if exists && newActivity.Required && activityCompatible(oldCourse, oldActivity, newCourse, newActivity) {
			return &RestartCheckpoint{ActivityIndex: newIndex}
		}
	}
	return nil
}

func firstVisibleLesson(course CompiledCourse, depth Depth) (Lesson, bool) {
	for _, lesson := range course.Pack.Lessons {
		if visibleAtDepth(lesson.MinimumDepth, depth) && len(RequiredActivityIDs(lesson)) > 0 {
			return lesson, true
		}
	}
	return Lesson{}, false
}
