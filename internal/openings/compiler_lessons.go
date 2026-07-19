package openings

import "fmt"

func (c *courseCompiler) validateLessons() {
	phaseRank := map[StepKind]int{
		StepExplain: 0,
		StepWatch:   1,
		StepTry:     2,
		StepBranch:  3,
		StepRecall:  4,
	}
	for lessonIndex, lesson := range c.pack.Lessons {
		path := fmt.Sprintf("lessons[%d]", lessonIndex)
		lessonRank, lessonDepthOK := depthRank(lesson.MinimumDepth)
		chapter, chapterOK := c.compiled.Chapters[lesson.ChapterID]
		if chapterOK && lessonDepthOK {
			chapterRank, chapterDepthOK := depthRank(chapter.MinimumDepth)
			if chapterDepthOK && lessonRank < chapterRank {
				c.addDiagnostic("lesson_depth_before_chapter", path+".minimumDepth", fmt.Sprintf("lesson depth %q is shallower than chapter depth %q", lesson.MinimumDepth, chapter.MinimumDepth))
			}
		}
		reachable := c.reachableByDepth[lesson.MinimumDepth]
		if lessonDepthOK && reachable != nil && !reachable[lesson.StartPositionID] {
			c.addDiagnostic("lesson_position_hidden", path+".startPositionId", fmt.Sprintf("start position %q is hidden at %s depth", lesson.StartPositionID, lesson.MinimumDepth))
		}

		seenSteps := map[string]string{}
		seenPhases := map[StepKind]bool{}
		lastPhase := -1
		for stepIndex, step := range lesson.Steps {
			stepPath := fmt.Sprintf("%s.steps[%d]", path, stepIndex)
			if !stableIDPattern.MatchString(step.StepID) {
				c.addDiagnostic("invalid_id", stepPath+".stepId", fmt.Sprintf("%q is not a stable ID", step.StepID))
			} else if first, duplicate := seenSteps[step.StepID]; duplicate {
				c.addDiagnostic("duplicate_id", stepPath+".stepId", fmt.Sprintf("step ID %q duplicates %s", step.StepID, first))
			} else {
				seenSteps[step.StepID] = stepPath
			}
			rank, kindOK := phaseRank[step.Kind]
			if !kindOK {
				c.addDiagnostic("invalid_step_kind", stepPath+".kind", fmt.Sprintf("unsupported step kind %q", step.Kind))
			} else {
				seenPhases[step.Kind] = true
				if rank < lastPhase {
					c.addDiagnostic("step_phase_order", stepPath+".kind", "teaching phases must stay in explain, watch, try, branch, recall order")
				}
				lastPhase = rank
			}
			c.validateText(stepPath+".title", step.Title, true)
			c.validateText(stepPath+".instruction", step.Instruction, true)
			if _, ok := c.compiled.Positions[step.PositionID]; !ok {
				c.addDiagnostic("missing_reference", stepPath+".positionId", fmt.Sprintf("position %q does not exist", step.PositionID))
			} else if lessonDepthOK && reachable != nil && !reachable[step.PositionID] {
				c.addDiagnostic("lesson_position_hidden", stepPath+".positionId", fmt.Sprintf("position %q is hidden at %s depth", step.PositionID, lesson.MinimumDepth))
			}
			for noteIndex, noteID := range step.NoteIDs {
				if _, ok := c.compiled.Notes[noteID]; !ok {
					c.addDiagnostic("missing_reference", fmt.Sprintf("%s.noteIds[%d]", stepPath, noteIndex), fmt.Sprintf("note %q does not exist", noteID))
				}
			}
			c.validateLessonStep(lesson, step, stepPath, lessonRank, lessonDepthOK)
		}
		for _, required := range []StepKind{StepExplain, StepWatch, StepTry, StepBranch, StepRecall} {
			if !seenPhases[required] {
				c.addDiagnostic("missing_teaching_phase", path+".steps", fmt.Sprintf("lesson is missing %s phase", required))
			}
		}
		if len(lesson.Steps) > 0 {
			if lesson.Steps[0].Kind != StepExplain {
				c.addDiagnostic("step_phase_order", path+".steps[0].kind", "lesson must begin with explain")
			}
			lastIndex := len(lesson.Steps) - 1
			if lesson.Steps[lastIndex].Kind != StepRecall {
				c.addDiagnostic("step_phase_order", fmt.Sprintf("%s.steps[%d].kind", path, lastIndex), "lesson must end with recall")
			}
		}
	}
}

func (c *courseCompiler) validateLessonStep(
	lesson Lesson,
	step LessonStep,
	path string,
	lessonRank int,
	lessonDepthOK bool,
) {
	switch step.Kind {
	case StepExplain:
		if step.PromptID != "" {
			c.addDiagnostic("unexpected_prompt", path+".promptId", "explain step cannot have a prompt")
		}
		if len(step.MoveIDs) != 0 {
			c.addDiagnostic("unexpected_move", path+".moveIds", "explain step cannot contain moves")
		}
	case StepWatch:
		if step.PromptID != "" {
			c.addDiagnostic("unexpected_prompt", path+".promptId", "watch step cannot have a prompt")
		}
		if len(step.MoveIDs) == 0 {
			c.addDiagnostic("missing_watch_move", path+".moveIds", "watch step requires at least one move")
		}
		positionID := step.PositionID
		for moveIndex, moveID := range step.MoveIDs {
			movePath := fmt.Sprintf("%s.moveIds[%d]", path, moveIndex)
			move, ok := c.compiled.Moves[moveID]
			if !ok {
				c.addDiagnostic("missing_reference", movePath, fmt.Sprintf("move %q does not exist", moveID))
				continue
			}
			if move.FromPositionID != positionID {
				c.addDiagnostic("disconnected_watch_path", movePath, fmt.Sprintf("move %q leaves %q, want %q", moveID, move.FromPositionID, positionID))
			}
			positionID = move.ToPositionID
			c.validateLessonMoveDepth(move, movePath, lesson.MinimumDepth, lessonRank, lessonDepthOK)
		}
	case StepTry, StepBranch, StepRecall:
		if step.PromptID == "" {
			c.addDiagnostic("missing_prompt", path+".promptId", fmt.Sprintf("%s step requires a prompt", step.Kind))
			return
		}
		prompt, ok := c.compiled.Prompts[step.PromptID]
		if !ok {
			c.addDiagnostic("missing_reference", path+".promptId", fmt.Sprintf("prompt %q does not exist", step.PromptID))
			return
		}
		if prompt.PositionID != step.PositionID {
			c.addDiagnostic("prompt_step_position", path+".positionId", fmt.Sprintf("step position %q differs from prompt position %q", step.PositionID, prompt.PositionID))
		}
		primary, primaryOK := c.compiled.Moves[prompt.PrimaryMoveID]
		if primaryOK {
			c.validateLessonMoveDepth(primary, path+".promptId", lesson.MinimumDepth, lessonRank, lessonDepthOK)
		}
		if len(step.MoveIDs) > 1 {
			c.addDiagnostic("automatic_continuation_length", path+".moveIds", "prompt step allows at most one automatic opponent move")
		}
		if len(step.MoveIDs) == 1 {
			continuation, continuationOK := c.compiled.Moves[step.MoveIDs[0]]
			movePath := path + ".moveIds[0]"
			if !continuationOK {
				c.addDiagnostic("missing_reference", movePath, fmt.Sprintf("move %q does not exist", step.MoveIDs[0]))
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
	default:
		// The caller reports the invalid kind.
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
