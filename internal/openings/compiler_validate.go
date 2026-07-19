package openings

import (
	"fmt"
	"strings"
)

const maximumCourseTextBytes = 20_000

var validNoteKinds = map[string]struct{}{
	"overview": {}, "history": {}, "plan": {}, "warning": {},
	"explanation": {}, "evaluation": {}, "transposition": {}, "illustrative_game": {},
}

func (c *courseCompiler) indexAndValidateValues() {
	if c.rules == nil {
		c.addDiagnostic("missing_rules", "$", "chess rules are required")
	}
	if !stableIDPattern.MatchString(c.pack.CourseID) {
		c.addDiagnostic("invalid_id", "courseId", fmt.Sprintf("%q is not a stable ID", c.pack.CourseID))
	}
	if c.pack.Perspective != PerspectiveWhite && c.pack.Perspective != PerspectiveBlack {
		c.addDiagnostic("invalid_perspective", "perspective", fmt.Sprintf("unsupported perspective %q", c.pack.Perspective))
	}
	if _, ok := depthRank(c.pack.DefaultDepth); !ok {
		c.addDiagnostic("invalid_depth", "defaultDepth", fmt.Sprintf("unsupported depth %q", c.pack.DefaultDepth))
	}
	c.validateText("title", c.pack.Title, true)
	c.validateText("description", c.pack.Description, true)

	for index, position := range c.pack.Positions {
		path := fmt.Sprintf("positions[%d]", index)
		if !c.registerID(c.positionPaths, position.PositionID, path+".positionId") {
			continue
		}
		c.validateEvaluation(path+".evaluation", position.Evaluation)
		c.validateText(path+".label", position.Label, false)
		c.compiled.Positions[position.PositionID] = CompiledPosition{
			ID: position.PositionID, Label: position.Label, Evaluation: position.Evaluation,
			NoteIDs: append([]string(nil), position.NoteIDs...),
		}
	}

	for index, move := range c.pack.Moves {
		path := fmt.Sprintf("moves[%d]", index)
		if !c.registerID(c.movePaths, move.MoveID, path+".moveId") {
			continue
		}
		if _, ok := depthRank(move.MinimumDepth); !ok {
			c.addDiagnostic("invalid_depth", path+".minimumDepth", fmt.Sprintf("unsupported depth %q", move.MinimumDepth))
		}
		switch move.TrainingRole {
		case RoleRepertoire, RoleOpponent, RoleAlternative:
		default:
			c.addDiagnostic("invalid_training_role", path+".trainingRole", fmt.Sprintf("unsupported role %q", move.TrainingRole))
		}
		if strings.TrimSpace(move.UCI) == "" {
			c.addDiagnostic("missing_move", path+".uci", "UCI move is required")
		}
		c.validateText(path+".variationName", move.VariationName, false)
		c.validateEvaluation(path+".evaluation", move.Evaluation)
		c.compiled.Moves[move.MoveID] = CompiledMove{Move: move}
	}

	for index, note := range c.pack.Notes {
		path := fmt.Sprintf("notes[%d]", index)
		if !c.registerID(c.notePaths, note.NoteID, path+".noteId") {
			continue
		}
		if _, ok := validNoteKinds[note.Kind]; !ok {
			c.addDiagnostic("invalid_note_kind", path+".kind", fmt.Sprintf("unsupported note kind %q", note.Kind))
		}
		c.validateText(path+".text", note.Text, true)
		c.compiled.Notes[note.NoteID] = note
	}

	chapterOrdinals := map[int]string{}
	for index, chapter := range c.pack.Chapters {
		path := fmt.Sprintf("chapters[%d]", index)
		if !c.registerID(c.chapterPaths, chapter.ChapterID, path+".chapterId") {
			continue
		}
		if chapter.Ordinal <= 0 {
			c.addDiagnostic("invalid_ordinal", path+".ordinal", "chapter ordinal must be positive")
		} else if first, duplicate := chapterOrdinals[chapter.Ordinal]; duplicate {
			c.addDiagnostic("duplicate_ordinal", path+".ordinal", fmt.Sprintf("chapter ordinal duplicates %s", first))
		} else {
			chapterOrdinals[chapter.Ordinal] = path
		}
		if _, ok := depthRank(chapter.MinimumDepth); !ok {
			c.addDiagnostic("invalid_depth", path+".minimumDepth", fmt.Sprintf("unsupported depth %q", chapter.MinimumDepth))
		}
		c.validateText(path+".title", chapter.Title, true)
		c.validateText(path+".overview", chapter.Overview, true)
		c.compiled.Chapters[chapter.ChapterID] = chapter
	}

	lessonOrdinals := map[string]map[int]string{}
	for index, lesson := range c.pack.Lessons {
		path := fmt.Sprintf("lessons[%d]", index)
		if !c.registerID(c.lessonPaths, lesson.LessonID, path+".lessonId") {
			continue
		}
		if lesson.Ordinal <= 0 {
			c.addDiagnostic("invalid_ordinal", path+".ordinal", "lesson ordinal must be positive")
		} else {
			byOrdinal := lessonOrdinals[lesson.ChapterID]
			if byOrdinal == nil {
				byOrdinal = map[int]string{}
				lessonOrdinals[lesson.ChapterID] = byOrdinal
			}
			if first, duplicate := byOrdinal[lesson.Ordinal]; duplicate {
				c.addDiagnostic("duplicate_ordinal", path+".ordinal", fmt.Sprintf("lesson ordinal duplicates %s", first))
			} else {
				byOrdinal[lesson.Ordinal] = path
			}
		}
		if _, ok := depthRank(lesson.MinimumDepth); !ok {
			c.addDiagnostic("invalid_depth", path+".minimumDepth", fmt.Sprintf("unsupported depth %q", lesson.MinimumDepth))
		}
		c.validateText(path+".title", lesson.Title, true)
		if len(lesson.Objectives) == 0 {
			c.addDiagnostic("missing_objective", path+".objectives", "lesson requires at least one objective")
		}
		for objectiveIndex, objective := range lesson.Objectives {
			c.validateText(fmt.Sprintf("%s.objectives[%d]", path, objectiveIndex), objective, true)
		}
		c.compiled.Lessons[lesson.LessonID] = lesson
	}

	for index, prompt := range c.pack.Prompts {
		path := fmt.Sprintf("prompts[%d]", index)
		if !c.registerID(c.promptPaths, prompt.PromptID, path+".promptId") {
			continue
		}
		c.compiled.Prompts[prompt.PromptID] = CompiledPrompt{Prompt: prompt}
	}
}

func (c *courseCompiler) registerID(paths map[string]string, id, path string) bool {
	if !stableIDPattern.MatchString(id) {
		c.addDiagnostic("invalid_id", path, fmt.Sprintf("%q is not a stable ID", id))
		return false
	}
	if first, duplicate := paths[id]; duplicate {
		c.addDiagnostic("duplicate_id", path, fmt.Sprintf("ID %q duplicates %s", id, first))
		return false
	}
	paths[id] = path
	return true
}

func (c *courseCompiler) validateText(path, value string, required bool) {
	if required && strings.TrimSpace(value) == "" {
		c.addDiagnostic("missing_text", path, "text is required")
	}
	if len(value) > maximumCourseTextBytes {
		c.addDiagnostic("text_too_large", path, fmt.Sprintf("text exceeds %d UTF-8 bytes", maximumCourseTextBytes))
	}
}

func (c *courseCompiler) validateEvaluation(path string, evaluation Evaluation) {
	switch evaluation.Code {
	case EvaluationNone, EvaluationEqual, EvaluationUnclear,
		EvaluationWhiteSlight, EvaluationBlackSlight,
		EvaluationWhiteClear, EvaluationBlackClear,
		EvaluationWhiteWinning, EvaluationBlackWinning:
	default:
		c.addDiagnostic("invalid_evaluation", path+".code", fmt.Sprintf("unsupported evaluation %q", evaluation.Code))
	}
	c.validateText(path+".sourceSymbol", evaluation.SourceSymbol, false)
}

func (c *courseCompiler) validateReferences() {
	if _, ok := c.compiled.Positions[c.pack.RootPositionID]; !ok {
		c.addDiagnostic("missing_reference", "rootPositionId", fmt.Sprintf("position %q does not exist", c.pack.RootPositionID))
	}

	for index, position := range c.pack.Positions {
		path := fmt.Sprintf("positions[%d]", index)
		for noteIndex, noteID := range position.NoteIDs {
			if _, ok := c.compiled.Notes[noteID]; !ok {
				c.addDiagnostic("missing_reference", fmt.Sprintf("%s.noteIds[%d]", path, noteIndex), fmt.Sprintf("note %q does not exist", noteID))
			}
		}
	}

	outgoingUCI := map[string]map[string]string{}
	for index, move := range c.pack.Moves {
		path := fmt.Sprintf("moves[%d]", index)
		_, moveIndexed := c.compiled.Moves[move.MoveID]
		_, fromOK := c.compiled.Positions[move.FromPositionID]
		_, toOK := c.compiled.Positions[move.ToPositionID]
		if !fromOK {
			c.addDiagnostic("missing_reference", path+".fromPositionId", fmt.Sprintf("position %q does not exist", move.FromPositionID))
		}
		if !toOK {
			c.addDiagnostic("missing_reference", path+".toPositionId", fmt.Sprintf("position %q does not exist", move.ToPositionID))
		}
		for noteIndex, noteID := range move.NoteIDs {
			if _, ok := c.compiled.Notes[noteID]; !ok {
				c.addDiagnostic("missing_reference", fmt.Sprintf("%s.noteIds[%d]", path, noteIndex), fmt.Sprintf("note %q does not exist", noteID))
			}
		}
		if !moveIndexed || !fromOK || !toOK {
			continue
		}
		seen := outgoingUCI[move.FromPositionID]
		if seen == nil {
			seen = map[string]string{}
			outgoingUCI[move.FromPositionID] = seen
		}
		if first, duplicate := seen[move.UCI]; duplicate {
			c.addDiagnostic("duplicate_outgoing_move", path+".uci", fmt.Sprintf("move %q duplicates %s", move.UCI, first))
		} else {
			seen[move.UCI] = path
		}
		c.compiled.Outgoing[move.FromPositionID] = append(c.compiled.Outgoing[move.FromPositionID], move.MoveID)
		c.compiled.Incoming[move.ToPositionID] = append(c.compiled.Incoming[move.ToPositionID], move.MoveID)
	}

	for index, lesson := range c.pack.Lessons {
		path := fmt.Sprintf("lessons[%d]", index)
		if _, ok := c.compiled.Chapters[lesson.ChapterID]; !ok {
			c.addDiagnostic("missing_reference", path+".chapterId", fmt.Sprintf("chapter %q does not exist", lesson.ChapterID))
		}
		if _, ok := c.compiled.Positions[lesson.StartPositionID]; !ok {
			c.addDiagnostic("missing_reference", path+".startPositionId", fmt.Sprintf("position %q does not exist", lesson.StartPositionID))
		}
	}

	for index, prompt := range c.pack.Prompts {
		path := fmt.Sprintf("prompts[%d]", index)
		if _, ok := c.compiled.Positions[prompt.PositionID]; !ok {
			c.addDiagnostic("missing_reference", path+".positionId", fmt.Sprintf("position %q does not exist", prompt.PositionID))
		}
		primary, primaryOK := c.compiled.Moves[prompt.PrimaryMoveID]
		if !primaryOK {
			c.addDiagnostic("missing_reference", path+".primaryMoveId", fmt.Sprintf("move %q does not exist", prompt.PrimaryMoveID))
		} else {
			if primary.FromPositionID != prompt.PositionID {
				c.addDiagnostic("prompt_primary_position", path+".primaryMoveId", fmt.Sprintf("primary move leaves %q, not prompt position %q", primary.FromPositionID, prompt.PositionID))
			}
			if primary.TrainingRole != RoleRepertoire {
				c.addDiagnostic("prompt_primary_role", path+".primaryMoveId", "primary move must have repertoire role")
			}
		}
		accepted := map[string]struct{}{}
		for alternativeIndex, moveID := range prompt.AcceptedAlternativeMoveIDs {
			alternativePath := fmt.Sprintf("%s.acceptedAlternativeMoveIds[%d]", path, alternativeIndex)
			if moveID == prompt.PrimaryMoveID {
				c.addDiagnostic("prompt_alternative_primary", alternativePath, "primary move cannot also be an alternative")
			}
			if _, duplicate := accepted[moveID]; duplicate {
				c.addDiagnostic("duplicate_reference", alternativePath, fmt.Sprintf("move %q is duplicated", moveID))
			}
			accepted[moveID] = struct{}{}
			move, ok := c.compiled.Moves[moveID]
			if !ok {
				c.addDiagnostic("missing_reference", alternativePath, fmt.Sprintf("move %q does not exist", moveID))
				continue
			}
			if move.FromPositionID != prompt.PositionID {
				c.addDiagnostic("prompt_alternative_position", alternativePath, fmt.Sprintf("alternative leaves %q, not prompt position %q", move.FromPositionID, prompt.PositionID))
			}
			if move.TrainingRole != RoleAlternative {
				c.addDiagnostic("prompt_alternative_role", alternativePath, "accepted alternative must have alternative role")
			}
		}
	}
}
