package openings

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
)

type courseQuerier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (c *SQLiteCatalog) ListActive(ctx context.Context) ([]CourseSummary, error) {
	if c == nil || c.db == nil {
		return nil, fmt.Errorf("course catalog is required")
	}
	rows, err := c.db.QueryContext(
		ctx,
		`SELECT course.course_id, head.generation_id, generation.content_version,
		        course.title, course.root_position_id, course.perspective,
		        course.default_depth
		 FROM course_heads head
		 JOIN course_generations generation
		   ON generation.course_id = head.course_id
		  AND generation.generation_id = head.generation_id
		 JOIN courses course ON course.generation_id = head.generation_id
		 WHERE generation.status = 'sealed'
		 ORDER BY course.title, course.course_id`,
	)
	if err != nil {
		return nil, fmt.Errorf("query active courses: %w", err)
	}
	defer rows.Close()
	summaries := make([]CourseSummary, 0)
	for rows.Next() {
		var summary CourseSummary
		if err := rows.Scan(
			&summary.CourseID,
			&summary.GenerationID,
			&summary.ContentVersion,
			&summary.Title,
			&summary.RootPositionID,
			&summary.Perspective,
			&summary.DefaultDepth,
		); err != nil {
			return nil, fmt.Errorf("scan active course: %w", err)
		}
		summaries = append(summaries, summary)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate active courses: %w", err)
	}
	return summaries, nil
}

func (c *SQLiteCatalog) ActiveGenerationID(ctx context.Context, courseID string) (string, error) {
	if c == nil || c.db == nil {
		return "", fmt.Errorf("course catalog is required")
	}
	var generationID string
	err := c.db.QueryRowContext(
		ctx,
		`SELECT head.generation_id
		 FROM course_heads head
		 JOIN course_generations generation
		   ON generation.course_id = head.course_id
		  AND generation.generation_id = head.generation_id
		 WHERE head.course_id = ? AND generation.status = 'sealed'`,
		courseID,
	).Scan(&generationID)
	return generationID, err
}

func (c *SQLiteCatalog) LoadActive(ctx context.Context, courseID string) (CompiledCourse, error) {
	if c == nil || c.db == nil {
		return CompiledCourse{}, fmt.Errorf("course catalog is required")
	}
	tx, err := c.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return CompiledCourse{}, fmt.Errorf("begin active course read: %w", err)
	}
	defer tx.Rollback()
	var generationID string
	if err := tx.QueryRowContext(
		ctx,
		`SELECT head.generation_id
		 FROM course_heads head
		 JOIN course_generations generation
		   ON generation.course_id = head.course_id
		  AND generation.generation_id = head.generation_id
		 WHERE head.course_id = ? AND generation.status = 'sealed'`,
		courseID,
	).Scan(&generationID); err != nil {
		return CompiledCourse{}, err
	}
	compiled, err := loadCourseGeneration(ctx, tx, generationID)
	if err != nil {
		return CompiledCourse{}, err
	}
	if err := tx.Commit(); err != nil {
		return CompiledCourse{}, fmt.Errorf("finish active course read: %w", err)
	}
	return compiled, nil
}

func (c *SQLiteCatalog) LoadGeneration(ctx context.Context, generationID string) (CompiledCourse, error) {
	if c == nil || c.db == nil {
		return CompiledCourse{}, fmt.Errorf("course catalog is required")
	}
	tx, err := c.db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
	if err != nil {
		return CompiledCourse{}, fmt.Errorf("begin course generation read: %w", err)
	}
	defer tx.Rollback()
	compiled, err := loadCourseGeneration(ctx, tx, generationID)
	if err != nil {
		return CompiledCourse{}, err
	}
	if err := tx.Commit(); err != nil {
		return CompiledCourse{}, fmt.Errorf("finish course generation read: %w", err)
	}
	return compiled, nil
}

func loadCourseGeneration(
	ctx context.Context,
	query courseQuerier,
	generationID string,
) (CompiledCourse, error) {
	compiled := CompiledCourse{
		Positions:      map[string]CompiledPosition{},
		Moves:          map[string]CompiledMove{},
		Notes:          map[string]Note{},
		Chapters:       map[string]Chapter{},
		Lessons:        map[string]Lesson{},
		Prompts:        map[string]CompiledPrompt{},
		Outgoing:       map[string][]string{},
		Incoming:       map[string][]string{},
		LessonChildren: map[string][]LessonEdge{},
		LessonParent:   map[string]LessonEdge{},
	}
	var sourceJSON, coverageJSON string
	if err := query.QueryRowContext(
		ctx,
		`SELECT generation.schema_version, generation.course_id,
		        generation.content_version, course.title, course.description,
		        course.perspective, course.default_depth, course.root_position_id,
		        course.source_json, course.coverage_json
		 FROM course_generations generation
		 JOIN courses course ON course.generation_id = generation.generation_id
		 WHERE generation.generation_id = ? AND generation.status = 'sealed'`,
		generationID,
	).Scan(
		&compiled.Pack.SchemaVersion,
		&compiled.Pack.CourseID,
		&compiled.Pack.ContentVersion,
		&compiled.Pack.Title,
		&compiled.Pack.Description,
		&compiled.Pack.Perspective,
		&compiled.Pack.DefaultDepth,
		&compiled.Pack.RootPositionID,
		&sourceJSON,
		&coverageJSON,
	); err != nil {
		return CompiledCourse{}, err
	}
	if err := json.Unmarshal([]byte(sourceJSON), &compiled.Pack.Source); err != nil {
		return CompiledCourse{}, fmt.Errorf("decode stored course source: %w", err)
	}
	var coverage storedCoverage
	if err := json.Unmarshal([]byte(coverageJSON), &coverage); err != nil {
		return CompiledCourse{}, fmt.Errorf("decode stored course coverage: %w", err)
	}
	compiled.Pack.SourceCoverage = coverage.Source
	compiled.Coverage = coverage.Report

	if err := loadCoursePositions(ctx, query, generationID, &compiled); err != nil {
		return CompiledCourse{}, err
	}
	root, exists := compiled.Positions[compiled.Pack.RootPositionID]
	if !exists {
		return CompiledCourse{}, fmt.Errorf("stored course root position %q is missing", compiled.Pack.RootPositionID)
	}
	compiled.Pack.RootFEN = root.FEN
	if err := loadCourseMoves(ctx, query, generationID, &compiled); err != nil {
		return CompiledCourse{}, err
	}
	if err := loadCourseNotes(ctx, query, generationID, &compiled); err != nil {
		return CompiledCourse{}, err
	}
	if err := loadCourseChapters(ctx, query, generationID, &compiled); err != nil {
		return CompiledCourse{}, err
	}
	if err := loadCourseLessons(ctx, query, generationID, &compiled); err != nil {
		return CompiledCourse{}, err
	}
	if err := loadCourseLessonEdges(ctx, query, generationID, &compiled); err != nil {
		return CompiledCourse{}, err
	}
	if err := loadCoursePrompts(ctx, query, generationID, &compiled); err != nil {
		return CompiledCourse{}, err
	}
	normalized, err := NormalizeCoursePack(compiled.Pack)
	if err != nil {
		return CompiledCourse{}, fmt.Errorf("normalize stored course: %w", err)
	}
	compiled.Pack = normalized
	compiled.Lessons = make(map[string]Lesson, len(normalized.Lessons))
	for _, lesson := range normalized.Lessons {
		compiled.Lessons[lesson.LessonID] = lesson
	}
	indexCompiledTeachingTree(&compiled)
	return compiled, nil
}

func decodeStoredJSON(text string, destination any, label string) error {
	if err := json.Unmarshal([]byte(text), destination); err != nil {
		return fmt.Errorf("decode stored %s: %w", label, err)
	}
	return nil
}
