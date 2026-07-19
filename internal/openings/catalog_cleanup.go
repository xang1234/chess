package openings

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
)

func (c *SQLiteCatalog) CleanupBatch(
	ctx context.Context,
	protected map[string]struct{},
	limit int,
) (bool, error) {
	if c == nil || c.db == nil {
		return false, errors.New("course catalog is required")
	}
	if limit <= 0 {
		return false, errors.New("course cleanup limit must be positive")
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}

	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin course cleanup: %w", err)
	}
	defer tx.Rollback()
	query, args := eligibleCourseGenerationsSQL(protected, limit)
	rows, err := tx.QueryContext(ctx, query, args...)
	if err != nil {
		return false, fmt.Errorf("query inactive course generations: %w", err)
	}
	generationIDs := make([]string, 0, limit)
	for rows.Next() {
		var generationID string
		if err := rows.Scan(&generationID); err != nil {
			rows.Close()
			return false, fmt.Errorf("scan inactive course generation: %w", err)
		}
		generationIDs = append(generationIDs, generationID)
	}
	if err := rows.Close(); err != nil {
		return false, fmt.Errorf("close inactive course generations: %w", err)
	}
	if err := rows.Err(); err != nil {
		return false, fmt.Errorf("iterate inactive course generations: %w", err)
	}
	for _, generationID := range generationIDs {
		if _, err := tx.ExecContext(
			ctx,
			`DELETE FROM course_generations
			 WHERE generation_id = ? AND status = 'sealed'
			   AND NOT EXISTS (
			     SELECT 1 FROM course_heads WHERE generation_id = ?
			   )`,
			generationID,
			generationID,
		); err != nil {
			return false, fmt.Errorf("delete inactive course generation %q: %w", generationID, err)
		}
	}

	moreQuery, moreArgs := eligibleCourseGenerationsSQL(protected, 1)
	var next string
	more := true
	if err := tx.QueryRowContext(ctx, moreQuery, moreArgs...).Scan(&next); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return false, fmt.Errorf("check remaining inactive course generations: %w", err)
		}
		more = false
	}
	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit course cleanup: %w", err)
	}
	return more, nil
}

func eligibleCourseGenerationsSQL(
	protected map[string]struct{},
	limit int,
) (string, []any) {
	query := strings.Builder{}
	query.WriteString(`SELECT generation.generation_id
	 FROM course_generations generation
	 WHERE generation.status = 'sealed'
	   AND NOT EXISTS (
	     SELECT 1 FROM course_heads head
	     WHERE head.generation_id = generation.generation_id
	   )`)
	args := make([]any, 0, len(protected)+1)
	if len(protected) != 0 {
		query.WriteString(` AND generation.generation_id NOT IN (`)
		generationIDs := make([]string, 0, len(protected))
		for generationID := range protected {
			generationIDs = append(generationIDs, generationID)
		}
		sort.Strings(generationIDs)
		for index, generationID := range generationIDs {
			if index != 0 {
				query.WriteString(",")
			}
			query.WriteString("?")
			args = append(args, generationID)
		}
		query.WriteString(")")
	}
	query.WriteString(` ORDER BY generation.generation_id LIMIT ?`)
	args = append(args, limit)
	return query.String(), args
}
