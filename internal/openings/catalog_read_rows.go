package openings

import (
	"context"
	"fmt"
)

func loadCoursePositions(
	ctx context.Context,
	query courseQuerier,
	generationID string,
	compiled *CompiledCourse,
) error {
	rows, err := query.QueryContext(
		ctx,
		`SELECT position_id, fen, label, evaluation_json, note_ids_json
		 FROM course_positions WHERE generation_id = ? ORDER BY rowid`,
		generationID,
	)
	if err != nil {
		return fmt.Errorf("query course positions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var position CompiledPosition
		var evaluationJSON, noteIDsJSON string
		if err := rows.Scan(
			&position.ID,
			&position.FEN,
			&position.Label,
			&evaluationJSON,
			&noteIDsJSON,
		); err != nil {
			return fmt.Errorf("scan course position: %w", err)
		}
		if err := decodeStoredJSON(evaluationJSON, &position.Evaluation, "position evaluation"); err != nil {
			return err
		}
		if err := decodeStoredJSON(noteIDsJSON, &position.NoteIDs, "position notes"); err != nil {
			return err
		}
		compiled.Positions[position.ID] = position
		compiled.Pack.Positions = append(compiled.Pack.Positions, Position{
			PositionID: position.ID,
			Label:      position.Label,
			Evaluation: position.Evaluation,
			NoteIDs:    position.NoteIDs,
		})
	}
	return rows.Err()
}

func loadCourseMoves(
	ctx context.Context,
	query courseQuerier,
	generationID string,
	compiled *CompiledCourse,
) error {
	rows, err := query.QueryContext(
		ctx,
		`SELECT move_id, from_position_id, to_position_id, uci, san,
		        minimum_depth, training_role, variation_name, evaluation_json,
		        note_ids_json, source_ref_json
		 FROM course_moves WHERE generation_id = ? ORDER BY rowid`,
		generationID,
	)
	if err != nil {
		return fmt.Errorf("query course moves: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var move CompiledMove
		var evaluationJSON, noteIDsJSON, sourceRefJSON string
		if err := rows.Scan(
			&move.MoveID,
			&move.FromPositionID,
			&move.ToPositionID,
			&move.UCI,
			&move.SAN,
			&move.MinimumDepth,
			&move.TrainingRole,
			&move.VariationName,
			&evaluationJSON,
			&noteIDsJSON,
			&sourceRefJSON,
		); err != nil {
			return fmt.Errorf("scan course move: %w", err)
		}
		if err := decodeStoredJSON(evaluationJSON, &move.Evaluation, "move evaluation"); err != nil {
			return err
		}
		if err := decodeStoredJSON(noteIDsJSON, &move.NoteIDs, "move notes"); err != nil {
			return err
		}
		if err := decodeStoredJSON(sourceRefJSON, &move.SourceRef, "move source"); err != nil {
			return err
		}
		compiled.Moves[move.MoveID] = move
		compiled.Outgoing[move.FromPositionID] = append(compiled.Outgoing[move.FromPositionID], move.MoveID)
		compiled.Incoming[move.ToPositionID] = append(compiled.Incoming[move.ToPositionID], move.MoveID)
		compiled.Pack.Moves = append(compiled.Pack.Moves, move.Move)
	}
	return rows.Err()
}

func loadCourseNotes(
	ctx context.Context,
	query courseQuerier,
	generationID string,
	compiled *CompiledCourse,
) error {
	rows, err := query.QueryContext(
		ctx,
		`SELECT note_id, kind, text, source_ref_json
		 FROM course_notes WHERE generation_id = ? ORDER BY rowid`,
		generationID,
	)
	if err != nil {
		return fmt.Errorf("query course notes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var note Note
		var sourceRefJSON string
		if err := rows.Scan(&note.NoteID, &note.Kind, &note.Text, &sourceRefJSON); err != nil {
			return fmt.Errorf("scan course note: %w", err)
		}
		if err := decodeStoredJSON(sourceRefJSON, &note.SourceRef, "note source"); err != nil {
			return err
		}
		compiled.Notes[note.NoteID] = note
		compiled.Pack.Notes = append(compiled.Pack.Notes, note)
	}
	return rows.Err()
}

func loadCourseChapters(
	ctx context.Context,
	query courseQuerier,
	generationID string,
	compiled *CompiledCourse,
) error {
	rows, err := query.QueryContext(
		ctx,
		`SELECT chapter_id, ordinal, title, overview, minimum_depth
		 FROM course_chapters WHERE generation_id = ? ORDER BY ordinal`,
		generationID,
	)
	if err != nil {
		return fmt.Errorf("query course chapters: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var chapter Chapter
		if err := rows.Scan(
			&chapter.ChapterID,
			&chapter.Ordinal,
			&chapter.Title,
			&chapter.Overview,
			&chapter.MinimumDepth,
		); err != nil {
			return fmt.Errorf("scan course chapter: %w", err)
		}
		compiled.Chapters[chapter.ChapterID] = chapter
		compiled.Pack.Chapters = append(compiled.Pack.Chapters, chapter)
	}
	return rows.Err()
}

func loadCourseLessons(
	ctx context.Context,
	query courseQuerier,
	generationID string,
	compiled *CompiledCourse,
) error {
	rows, err := query.QueryContext(
		ctx,
		`SELECT lesson.lesson_id, lesson.chapter_id, lesson.ordinal, lesson.title,
		        lesson.objectives_json, lesson.minimum_depth, lesson.start_position_id
		 FROM course_lessons lesson
		 JOIN course_chapters chapter
		   ON chapter.generation_id = lesson.generation_id
		  AND chapter.chapter_id = lesson.chapter_id
		 WHERE lesson.generation_id = ?
		 ORDER BY chapter.ordinal, lesson.ordinal, lesson.rowid`,
		generationID,
	)
	if err != nil {
		return fmt.Errorf("query course lessons: %w", err)
	}
	for rows.Next() {
		var lesson Lesson
		var objectivesJSON string
		if err := rows.Scan(
			&lesson.LessonID,
			&lesson.ChapterID,
			&lesson.Ordinal,
			&lesson.Title,
			&objectivesJSON,
			&lesson.MinimumDepth,
			&lesson.StartPositionID,
		); err != nil {
			rows.Close()
			return fmt.Errorf("scan course lesson: %w", err)
		}
		if err := decodeStoredJSON(objectivesJSON, &lesson.Objectives, "lesson objectives"); err != nil {
			rows.Close()
			return err
		}
		compiled.Lessons[lesson.LessonID] = lesson
		compiled.Pack.Lessons = append(compiled.Pack.Lessons, lesson)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close course lessons: %w", err)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	stepRows, err := query.QueryContext(
		ctx,
		`SELECT lesson_id, data_json
		 FROM course_lesson_steps
		 WHERE generation_id = ? ORDER BY lesson_id, ordinal`,
		generationID,
	)
	if err != nil {
		return fmt.Errorf("query course lesson steps: %w", err)
	}
	defer stepRows.Close()
	for stepRows.Next() {
		var lessonID, dataJSON string
		if err := stepRows.Scan(&lessonID, &dataJSON); err != nil {
			return fmt.Errorf("scan course lesson step: %w", err)
		}
		var step LessonStep
		if err := decodeStoredJSON(dataJSON, &step, "lesson step"); err != nil {
			return err
		}
		lesson, exists := compiled.Lessons[lessonID]
		if !exists {
			return fmt.Errorf("stored lesson step references missing lesson %q", lessonID)
		}
		lesson.Steps = append(lesson.Steps, step)
		compiled.Lessons[lessonID] = lesson
	}
	if err := stepRows.Err(); err != nil {
		return err
	}
	for index := range compiled.Pack.Lessons {
		compiled.Pack.Lessons[index] = compiled.Lessons[compiled.Pack.Lessons[index].LessonID]
	}
	return nil
}

func loadCoursePrompts(
	ctx context.Context,
	query courseQuerier,
	generationID string,
	compiled *CompiledCourse,
) error {
	rows, err := query.QueryContext(
		ctx,
		`SELECT prompt_id, position_id, primary_move_id, accepted_move_ids_json,
		        semantic_fingerprint
		 FROM course_prompts WHERE generation_id = ? ORDER BY rowid`,
		generationID,
	)
	if err != nil {
		return fmt.Errorf("query course prompts: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var prompt CompiledPrompt
		var acceptedJSON string
		if err := rows.Scan(
			&prompt.PromptID,
			&prompt.PositionID,
			&prompt.PrimaryMoveID,
			&acceptedJSON,
			&prompt.SemanticFingerprint,
		); err != nil {
			return fmt.Errorf("scan course prompt: %w", err)
		}
		if err := decodeStoredJSON(
			acceptedJSON,
			&prompt.AcceptedAlternativeMoveIDs,
			"prompt alternatives",
		); err != nil {
			return err
		}
		compiled.Prompts[prompt.PromptID] = prompt
		compiled.Pack.Prompts = append(compiled.Pack.Prompts, prompt.Prompt)
	}
	return rows.Err()
}
