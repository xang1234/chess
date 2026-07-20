package openings

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type ReplaceResult struct {
	CourseID     string
	GenerationID string
	PreviousHead string
}

type CourseSummary struct {
	CourseID       string      `json:"courseId"`
	GenerationID   string      `json:"generationId"`
	ContentVersion string      `json:"contentVersion"`
	Title          string      `json:"title"`
	RootPositionID string      `json:"rootPositionId"`
	Perspective    Perspective `json:"perspective"`
	DefaultDepth   Depth       `json:"defaultDepth"`
}

type SQLiteCatalog struct {
	db          *sql.DB
	now         func() time.Time
	newID       func() string
	afterInsert func(string, int)
}

func NewSQLiteCatalog(db *sql.DB) *SQLiteCatalog {
	return &SQLiteCatalog{db: db, now: time.Now, newID: uuid.NewString}
}

type storedCoverage struct {
	Source SourceCoverage `json:"source"`
	Report CoverageReport `json:"report"`
}

func (c *SQLiteCatalog) Replace(
	ctx context.Context,
	compiled CompiledCourse,
	sourcePath string,
	checksum string,
) (ReplaceResult, error) {
	if c == nil || c.db == nil {
		return ReplaceResult{}, errors.New("course catalog is required")
	}
	if err := ctx.Err(); err != nil {
		return ReplaceResult{}, err
	}
	if strings.TrimSpace(compiled.Pack.CourseID) == "" {
		return ReplaceResult{}, errors.New("course ID is required")
	}
	if strings.TrimSpace(sourcePath) == "" {
		return ReplaceResult{}, errors.New("course source path is required")
	}
	if strings.TrimSpace(checksum) == "" {
		return ReplaceResult{}, errors.New("course checksum is required")
	}

	generationID := c.newID()
	startedAt := c.now().UTC()
	if generationID == "" || startedAt.Unix() <= 0 {
		return ReplaceResult{}, errors.New("course generation identity is invalid")
	}
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return ReplaceResult{}, fmt.Errorf("begin course replacement: %w", err)
	}
	defer tx.Rollback()

	result := ReplaceResult{CourseID: compiled.Pack.CourseID, GenerationID: generationID}
	err = tx.QueryRowContext(
		ctx,
		`SELECT generation_id FROM course_heads WHERE course_id = ?`,
		compiled.Pack.CourseID,
	).Scan(&result.PreviousHead)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return ReplaceResult{}, fmt.Errorf("read previous course head: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO course_generations(
		   generation_id, course_id, status, source_path, schema_version,
		   content_version, started_at
		 ) VALUES (?, ?, 'building', ?, ?, ?, ?)`,
		generationID,
		compiled.Pack.CourseID,
		sourcePath,
		compiled.Pack.SchemaVersion,
		compiled.Pack.ContentVersion,
		startedAt.Unix(),
	); err != nil {
		return ReplaceResult{}, fmt.Errorf("insert course generation: %w", err)
	}

	if err := c.insertCompiledCourse(ctx, tx, generationID, compiled); err != nil {
		return ReplaceResult{}, err
	}
	if _, err := tx.ExecContext(
		ctx,
		`UPDATE course_generations
		 SET status = 'sealed', checksum = ?, sealed_at = ?
		 WHERE generation_id = ? AND status = 'building'`,
		checksum,
		c.now().UTC().Unix(),
		generationID,
	); err != nil {
		return ReplaceResult{}, fmt.Errorf("seal course generation: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO course_heads(course_id, generation_id) VALUES (?, ?)
		 ON CONFLICT(course_id) DO UPDATE SET generation_id = excluded.generation_id`,
		compiled.Pack.CourseID,
		generationID,
	); err != nil {
		return ReplaceResult{}, fmt.Errorf("activate course generation: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return ReplaceResult{}, fmt.Errorf("commit course replacement: %w", err)
	}
	return result, nil
}

func (c *SQLiteCatalog) insertCompiledCourse(
	ctx context.Context,
	tx *sql.Tx,
	generationID string,
	compiled CompiledCourse,
) error {
	sourceJSON, err := marshalStoredJSON(compiled.Pack.Source)
	if err != nil {
		return fmt.Errorf("marshal course source: %w", err)
	}
	coverageJSON, err := marshalStoredJSON(storedCoverage{
		Source: compiled.Pack.SourceCoverage,
		Report: compiled.Coverage,
	})
	if err != nil {
		return fmt.Errorf("marshal course coverage: %w", err)
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO courses(
		   generation_id, course_id, title, description, perspective,
		   default_depth, root_position_id, source_json, coverage_json
		 ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		generationID,
		compiled.Pack.CourseID,
		compiled.Pack.Title,
		compiled.Pack.Description,
		compiled.Pack.Perspective,
		compiled.Pack.DefaultDepth,
		compiled.Pack.RootPositionID,
		sourceJSON,
		coverageJSON,
	); err != nil {
		return fmt.Errorf("insert course: %w", err)
	}
	if err := c.afterCourseInsert(ctx, "courses", 1); err != nil {
		return err
	}

	for index, chapter := range compiled.Pack.Chapters {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO course_chapters(
			   generation_id, chapter_id, ordinal, title, overview, minimum_depth
			 ) VALUES (?, ?, ?, ?, ?, ?)`,
			generationID,
			chapter.ChapterID,
			chapter.Ordinal,
			chapter.Title,
			chapter.Overview,
			chapter.MinimumDepth,
		); err != nil {
			return fmt.Errorf("insert course chapter %q: %w", chapter.ChapterID, err)
		}
		if err := c.afterCourseInsert(ctx, "course_chapters", index+1); err != nil {
			return err
		}
	}
	if err := c.insertCoursePositions(ctx, tx, generationID, compiled); err != nil {
		return err
	}
	if err := c.insertCourseMoves(ctx, tx, generationID, compiled); err != nil {
		return err
	}
	if err := c.insertCourseNotes(ctx, tx, generationID, compiled); err != nil {
		return err
	}
	if err := c.insertCourseLessons(ctx, tx, generationID, compiled); err != nil {
		return err
	}
	if err := c.insertCourseLessonEdges(ctx, tx, generationID, compiled); err != nil {
		return err
	}
	return c.insertCoursePrompts(ctx, tx, generationID, compiled)
}

func (c *SQLiteCatalog) afterCourseInsert(ctx context.Context, table string, ordinal int) error {
	if c.afterInsert != nil {
		c.afterInsert(table, ordinal)
	}
	return ctx.Err()
}

func marshalStoredJSON(value any) (string, error) {
	encoded, err := json.Marshal(value)
	return string(encoded), err
}
