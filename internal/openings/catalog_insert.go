package openings

import (
	"context"
	"database/sql"
	"fmt"
)

func (c *SQLiteCatalog) insertCoursePositions(
	ctx context.Context,
	tx *sql.Tx,
	generationID string,
	compiled CompiledCourse,
) error {
	for index, source := range compiled.Pack.Positions {
		position, exists := compiled.Positions[source.PositionID]
		if !exists {
			return fmt.Errorf("compiled position %q is missing", source.PositionID)
		}
		evaluationJSON, err := marshalStoredJSON(position.Evaluation)
		if err != nil {
			return fmt.Errorf("marshal position %q evaluation: %w", position.ID, err)
		}
		noteIDsJSON, err := marshalStoredJSON(position.NoteIDs)
		if err != nil {
			return fmt.Errorf("marshal position %q notes: %w", position.ID, err)
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO course_positions(
			   generation_id, position_id, fen, label, evaluation_json, note_ids_json
			 ) VALUES (?, ?, ?, ?, ?, ?)`,
			generationID,
			position.ID,
			position.FEN,
			position.Label,
			evaluationJSON,
			noteIDsJSON,
		); err != nil {
			return fmt.Errorf("insert course position %q: %w", position.ID, err)
		}
		if err := c.afterCourseInsert(ctx, "course_positions", index+1); err != nil {
			return err
		}
	}
	return nil
}

func (c *SQLiteCatalog) insertCourseMoves(
	ctx context.Context,
	tx *sql.Tx,
	generationID string,
	compiled CompiledCourse,
) error {
	for index, source := range compiled.Pack.Moves {
		move, exists := compiled.Moves[source.MoveID]
		if !exists {
			return fmt.Errorf("compiled move %q is missing", source.MoveID)
		}
		evaluationJSON, err := marshalStoredJSON(move.Evaluation)
		if err != nil {
			return fmt.Errorf("marshal move %q evaluation: %w", move.MoveID, err)
		}
		noteIDsJSON, err := marshalStoredJSON(move.NoteIDs)
		if err != nil {
			return fmt.Errorf("marshal move %q notes: %w", move.MoveID, err)
		}
		sourceRefJSON, err := marshalStoredJSON(move.SourceRef)
		if err != nil {
			return fmt.Errorf("marshal move %q source: %w", move.MoveID, err)
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO course_moves(
			   generation_id, move_id, from_position_id, to_position_id, uci, san,
			   minimum_depth, training_role, variation_name, evaluation_json,
			   note_ids_json, source_ref_json
			 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			generationID,
			move.MoveID,
			move.FromPositionID,
			move.ToPositionID,
			move.UCI,
			move.SAN,
			move.MinimumDepth,
			move.TrainingRole,
			move.VariationName,
			evaluationJSON,
			noteIDsJSON,
			sourceRefJSON,
		); err != nil {
			return fmt.Errorf("insert course move %q: %w", move.MoveID, err)
		}
		if err := c.afterCourseInsert(ctx, "course_moves", index+1); err != nil {
			return err
		}
	}
	return nil
}

func (c *SQLiteCatalog) insertCourseNotes(
	ctx context.Context,
	tx *sql.Tx,
	generationID string,
	compiled CompiledCourse,
) error {
	for index, note := range compiled.Pack.Notes {
		sourceRefJSON, err := marshalStoredJSON(note.SourceRef)
		if err != nil {
			return fmt.Errorf("marshal note %q source: %w", note.NoteID, err)
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO course_notes(
			   generation_id, note_id, kind, text, source_ref_json
			 ) VALUES (?, ?, ?, ?, ?)`,
			generationID,
			note.NoteID,
			note.Kind,
			note.Text,
			sourceRefJSON,
		); err != nil {
			return fmt.Errorf("insert course note %q: %w", note.NoteID, err)
		}
		if err := c.afterCourseInsert(ctx, "course_notes", index+1); err != nil {
			return err
		}
	}
	return nil
}

func (c *SQLiteCatalog) insertCourseLessons(
	ctx context.Context,
	tx *sql.Tx,
	generationID string,
	compiled CompiledCourse,
) error {
	stepOrdinal := 0
	for index, lesson := range compiled.Pack.Lessons {
		objectivesJSON, err := marshalStoredJSON(lesson.Objectives)
		if err != nil {
			return fmt.Errorf("marshal lesson %q objectives: %w", lesson.LessonID, err)
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO course_lessons(
			   generation_id, lesson_id, chapter_id, ordinal, title, objectives_json,
			   minimum_depth, start_position_id
			 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			generationID,
			lesson.LessonID,
			lesson.ChapterID,
			lesson.Ordinal,
			lesson.Title,
			objectivesJSON,
			lesson.MinimumDepth,
			lesson.StartPositionID,
		); err != nil {
			return fmt.Errorf("insert course lesson %q: %w", lesson.LessonID, err)
		}
		if err := c.afterCourseInsert(ctx, "course_lessons", index+1); err != nil {
			return err
		}
		for lessonStepOrdinal, step := range lesson.Steps {
			dataJSON, err := marshalStoredJSON(step)
			if err != nil {
				return fmt.Errorf("marshal lesson step %q: %w", step.StepID, err)
			}
			if _, err := tx.ExecContext(
				ctx,
				`INSERT INTO course_lesson_steps(
				   generation_id, lesson_id, ordinal, step_id, kind, position_id, data_json
				 ) VALUES (?, ?, ?, ?, ?, ?, ?)`,
				generationID,
				lesson.LessonID,
				lessonStepOrdinal+1,
				step.StepID,
				step.Kind,
				step.PositionID,
				dataJSON,
			); err != nil {
				return fmt.Errorf("insert course lesson step %q: %w", step.StepID, err)
			}
			stepOrdinal++
			if err := c.afterCourseInsert(ctx, "course_lesson_steps", stepOrdinal); err != nil {
				return err
			}
		}
	}
	return nil
}

func (c *SQLiteCatalog) insertCoursePrompts(
	ctx context.Context,
	tx *sql.Tx,
	generationID string,
	compiled CompiledCourse,
) error {
	for index, source := range compiled.Pack.Prompts {
		prompt, exists := compiled.Prompts[source.PromptID]
		if !exists {
			return fmt.Errorf("compiled prompt %q is missing", source.PromptID)
		}
		acceptedJSON, err := marshalStoredJSON(prompt.AcceptedAlternativeMoveIDs)
		if err != nil {
			return fmt.Errorf("marshal prompt %q alternatives: %w", prompt.PromptID, err)
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO course_prompts(
			   generation_id, prompt_id, position_id, primary_move_id,
			   accepted_move_ids_json, semantic_fingerprint
			 ) VALUES (?, ?, ?, ?, ?, ?)`,
			generationID,
			prompt.PromptID,
			prompt.PositionID,
			prompt.PrimaryMoveID,
			acceptedJSON,
			prompt.SemanticFingerprint,
		); err != nil {
			return fmt.Errorf("insert course prompt %q: %w", prompt.PromptID, err)
		}
		if err := c.afterCourseInsert(ctx, "course_prompts", index+1); err != nil {
			return err
		}
	}
	return nil
}
