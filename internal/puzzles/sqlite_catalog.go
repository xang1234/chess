package puzzles

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"chess-trainer/internal/domain"

	"github.com/google/uuid"
)

const stagingBatchSize = 1000

type SQLiteCatalog struct {
	db *sql.DB
}

var _ Catalog = (*SQLiteCatalog)(nil)

type stagedRow struct {
	ordinal int64
	puzzle  []byte
}

type sqliteStagedImport struct {
	catalog  *SQLiteCatalog
	source   Source
	importID string
	checksum string
	next     int64
	buffer   []stagedRow
	report   ImportReport
	finished bool
}

func NewSQLiteCatalog(db *sql.DB) *SQLiteCatalog {
	return &SQLiteCatalog{db: db}
}

func (c *SQLiteCatalog) BeginImport(ctx context.Context, source Source) (StagedImport, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(source.ID) == "" {
		return nil, errors.New("source ID is required")
	}
	return &sqliteStagedImport{
		catalog:  c,
		source:   source,
		importID: uuid.NewString(),
		buffer:   make([]stagedRow, 0, stagingBatchSize),
	}, nil
}

func (s *sqliteStagedImport) Add(ctx context.Context, puzzle domain.Puzzle) error {
	if s.finished {
		return errors.New("import is already finished")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	encoded, err := json.Marshal(puzzle)
	if err != nil {
		return err
	}
	s.next++
	s.buffer = append(s.buffer, stagedRow{ordinal: s.next, puzzle: encoded})
	if len(s.buffer) == stagingBatchSize {
		return s.flush(ctx)
	}
	return nil
}

func (s *sqliteStagedImport) Reject(rejection Rejection) {
	s.report.Rejected++
	if len(s.report.Examples) < 100 {
		s.report.Examples = append(s.report.Examples, rejection)
	}
}

func (s *sqliteStagedImport) SetChecksum(checksum string) {
	s.checksum = strings.ToLower(strings.TrimSpace(checksum))
}

func (s *sqliteStagedImport) flush(ctx context.Context) error {
	if len(s.buffer) == 0 {
		return nil
	}
	tx, err := s.catalog.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, row := range s.buffer {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO import_staging(import_id, ordinal, puzzle_json) VALUES (?, ?, ?)`,
			s.importID,
			row.ordinal,
			string(row.puzzle),
		); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.buffer = s.buffer[:0]
	return nil
}

func (s *sqliteStagedImport) Abort(ctx context.Context) error {
	if s.finished {
		return nil
	}
	if _, err := s.catalog.db.ExecContext(
		ctx,
		`DELETE FROM import_staging WHERE import_id = ?`,
		s.importID,
	); err != nil {
		return err
	}
	s.buffer = nil
	s.finished = true
	return nil
}

func (s *sqliteStagedImport) Commit(ctx context.Context) (ImportReport, error) {
	if s.finished {
		return ImportReport{}, errors.New("import is already finished")
	}
	if s.checksum == "" {
		return ImportReport{}, errors.New("source checksum is required before commit")
	}
	if err := s.flush(ctx); err != nil {
		return ImportReport{}, err
	}

	tx, err := s.catalog.db.BeginTx(ctx, nil)
	if err != nil {
		return ImportReport{}, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, `DELETE FROM sources WHERE source_id = ?`, s.source.ID); err != nil {
		return ImportReport{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM puzzles
        WHERE NOT EXISTS (
          SELECT 1 FROM puzzle_sources ps WHERE ps.fingerprint = puzzles.fingerprint
        )`); err != nil {
		return ImportReport{}, err
	}
	importedAt := s.source.ImportedAt
	if importedAt.IsZero() {
		importedAt = time.Now()
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO sources(source_id, kind, imported_at, source_path, checksum)
         VALUES (?, ?, ?, ?, ?)`,
		s.source.ID,
		s.source.Kind,
		importedAt.Unix(),
		s.source.Path,
		s.checksum,
	); err != nil {
		return ImportReport{}, err
	}

	rows, err := tx.QueryContext(
		ctx,
		`SELECT puzzle_json FROM import_staging WHERE import_id = ? ORDER BY ordinal`,
		s.importID,
	)
	if err != nil {
		return ImportReport{}, err
	}
	report := s.report
	for rows.Next() {
		var encoded string
		if err := rows.Scan(&encoded); err != nil {
			rows.Close()
			return ImportReport{}, err
		}
		var puzzle domain.Puzzle
		if err := json.Unmarshal([]byte(encoded), &puzzle); err != nil {
			rows.Close()
			return ImportReport{}, err
		}
		if err := insertStagedPuzzle(ctx, tx, s.source.ID, puzzle, &report); err != nil {
			rows.Close()
			return ImportReport{}, err
		}
	}
	if err := rows.Close(); err != nil {
		return ImportReport{}, err
	}
	if err := rows.Err(); err != nil {
		return ImportReport{}, err
	}
	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM import_staging WHERE import_id = ?`,
		s.importID,
	); err != nil {
		return ImportReport{}, err
	}
	if err := tx.Commit(); err != nil {
		return ImportReport{}, err
	}

	s.report = report
	s.finished = true
	return report, nil
}

func insertStagedPuzzle(
	ctx context.Context,
	tx *sql.Tx,
	sourceID string,
	puzzle domain.Puzzle,
	report *ImportReport,
) error {
	if puzzle.Fingerprint == "" {
		return errors.New("puzzle fingerprint is required")
	}
	plies := solutionDepth(puzzle.Solution)
	if plies == 0 {
		return fmt.Errorf("puzzle %s has an empty solution", puzzle.Fingerprint)
	}
	solution, err := json.Marshal(puzzle.Solution)
	if err != nil {
		return err
	}
	result, err := tx.ExecContext(
		ctx,
		`INSERT OR IGNORE INTO puzzles(
           fingerprint, source_fen, prelude_uci, displayed_fen, solver, solution_json, solution_plies
         ) VALUES (?, ?, ?, ?, ?, ?, ?)`,
		puzzle.Fingerprint,
		nullIfEmpty(puzzle.SourceFEN),
		nullIfEmpty(puzzle.PreludeUCI),
		puzzle.DisplayedFEN,
		string(puzzle.Solver),
		string(solution),
		plies,
	)
	if err != nil {
		return err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if inserted == 0 {
		report.Duplicates++
	} else {
		report.Accepted++
	}

	ref := sourceReference(puzzle.Sources, sourceID)
	metadata, err := sourceMetadataJSON(ref)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO puzzle_sources(
           fingerprint, source_id, external_id, rating, popularity, play_count, source_url, metadata_json
         ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
         ON CONFLICT(fingerprint, source_id) DO UPDATE SET
           external_id=excluded.external_id,
           rating=excluded.rating,
           popularity=excluded.popularity,
           play_count=excluded.play_count,
           source_url=excluded.source_url,
           metadata_json=excluded.metadata_json`,
		puzzle.Fingerprint,
		sourceID,
		nullIfEmpty(ref.ExternalID),
		puzzle.Rating,
		puzzle.Popularity,
		puzzle.PlayCount,
		nullIfEmpty(ref.URL),
		metadata,
	); err != nil {
		return err
	}
	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM puzzle_themes WHERE fingerprint = ? AND source_id = ?`,
		puzzle.Fingerprint,
		sourceID,
	); err != nil {
		return err
	}
	for _, theme := range puzzle.Themes {
		theme = strings.TrimSpace(theme)
		if theme == "" {
			continue
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT OR IGNORE INTO puzzle_themes(fingerprint, source_id, theme) VALUES (?, ?, ?)`,
			puzzle.Fingerprint,
			sourceID,
			theme,
		); err != nil {
			return err
		}
	}
	return nil
}

func solutionDepth(nodes []domain.MoveNode) int {
	maxDepth := 0
	for _, node := range nodes {
		depth := 1 + solutionDepth(node.Children)
		if depth > maxDepth {
			maxDepth = depth
		}
	}
	return maxDepth
}

func sourceReference(refs []domain.SourceRef, sourceID string) domain.SourceRef {
	for _, ref := range refs {
		if ref.SourceID == sourceID {
			return ref
		}
	}
	return domain.SourceRef{SourceID: sourceID}
}

func sourceMetadataJSON(ref domain.SourceRef) (string, error) {
	metadata := make(map[string]any, len(ref.Metadata)+1)
	for key, value := range ref.Metadata {
		metadata[key] = value
	}
	if ref.Attribution != "" {
		metadata["attribution"] = ref.Attribution
	}
	encoded, err := json.Marshal(metadata)
	return string(encoded), err
}

func nullIfEmpty(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func (c *SQLiteCatalog) Get(ctx context.Context, fingerprint string) (domain.Puzzle, error) {
	var puzzle domain.Puzzle
	var sourceFEN sql.NullString
	var preludeUCI sql.NullString
	var solver string
	var solution string
	err := c.db.QueryRowContext(
		ctx,
		`SELECT source_fen, prelude_uci, displayed_fen, solver, solution_json
         FROM puzzles WHERE fingerprint = ?`,
		fingerprint,
	).Scan(&sourceFEN, &preludeUCI, &puzzle.DisplayedFEN, &solver, &solution)
	if err != nil {
		return domain.Puzzle{}, err
	}
	puzzle.Fingerprint = fingerprint
	puzzle.SourceFEN = sourceFEN.String
	puzzle.PreludeUCI = preludeUCI.String
	puzzle.Solver = domain.Color(solver)
	if err := json.Unmarshal([]byte(solution), &puzzle.Solution); err != nil {
		return domain.Puzzle{}, err
	}

	rows, err := c.db.QueryContext(
		ctx,
		`SELECT source_id, external_id, rating, popularity, play_count, source_url, metadata_json
         FROM puzzle_sources WHERE fingerprint = ? ORDER BY source_id`,
		fingerprint,
	)
	if err != nil {
		return domain.Puzzle{}, err
	}
	for rows.Next() {
		var ref domain.SourceRef
		var externalID, sourceURL sql.NullString
		var rating, popularity, playCount sql.NullInt64
		var metadata string
		if err := rows.Scan(
			&ref.SourceID,
			&externalID,
			&rating,
			&popularity,
			&playCount,
			&sourceURL,
			&metadata,
		); err != nil {
			rows.Close()
			return domain.Puzzle{}, err
		}
		ref.ExternalID = externalID.String
		ref.URL = sourceURL.String
		if err := json.Unmarshal([]byte(metadata), &ref.Metadata); err != nil {
			rows.Close()
			return domain.Puzzle{}, err
		}
		if attribution, ok := ref.Metadata["attribution"].(string); ok {
			ref.Attribution = attribution
			delete(ref.Metadata, "attribution")
		}
		puzzle.Sources = append(puzzle.Sources, ref)
		if puzzle.Rating == nil && rating.Valid {
			value := int(rating.Int64)
			puzzle.Rating = &value
		}
		if puzzle.Popularity == nil && popularity.Valid {
			value := int(popularity.Int64)
			puzzle.Popularity = &value
		}
		if puzzle.PlayCount == nil && playCount.Valid {
			value := int(playCount.Int64)
			puzzle.PlayCount = &value
		}
	}
	if err := rows.Close(); err != nil {
		return domain.Puzzle{}, err
	}
	if err := rows.Err(); err != nil {
		return domain.Puzzle{}, err
	}

	themeRows, err := c.db.QueryContext(
		ctx,
		`SELECT DISTINCT theme FROM puzzle_themes WHERE fingerprint = ? ORDER BY theme`,
		fingerprint,
	)
	if err != nil {
		return domain.Puzzle{}, err
	}
	defer themeRows.Close()
	for themeRows.Next() {
		var theme string
		if err := themeRows.Scan(&theme); err != nil {
			return domain.Puzzle{}, err
		}
		puzzle.Themes = append(puzzle.Themes, theme)
	}
	if err := themeRows.Err(); err != nil {
		return domain.Puzzle{}, err
	}
	return puzzle, nil
}

type candidateKey struct {
	fingerprint string
	sourceID    string
}

func (c *SQLiteCatalog) RatedCandidates(
	ctx context.Context,
	minimum int,
	maximum int,
	excluded []string,
	limit int,
) ([]domain.Puzzle, error) {
	if limit <= 0 {
		return []domain.Puzzle{}, nil
	}
	query := strings.Builder{}
	query.WriteString(`SELECT ps.fingerprint, ps.source_id
        FROM puzzle_sources ps
        WHERE ps.rating BETWEEN ? AND ?`)
	args := []any{minimum, maximum}
	if len(excluded) > 0 {
		query.WriteString(` AND ps.fingerprint NOT IN (`)
		query.WriteString(sqlPlaceholders(len(excluded)))
		query.WriteString(`)`)
		for _, fingerprint := range excluded {
			args = append(args, fingerprint)
		}
	}
	query.WriteString(` ORDER BY ps.rating, ps.fingerprint LIMIT 500`)
	keys, err := c.candidateKeys(ctx, query.String(), args...)
	if err != nil {
		return nil, err
	}
	return c.hydrateCandidates(ctx, keys, limit)
}

func (c *SQLiteCatalog) FreePracticeCandidates(
	ctx context.Context,
	sourceID string,
	minimum *int,
	maximum *int,
	themes []string,
	limit int,
) ([]domain.Puzzle, error) {
	if limit <= 0 {
		return []domain.Puzzle{}, nil
	}
	query := strings.Builder{}
	query.WriteString(`SELECT ps.fingerprint, ps.source_id
        FROM puzzle_sources ps
        WHERE ps.source_id = ?`)
	args := []any{sourceID}
	if minimum != nil {
		query.WriteString(` AND ps.rating >= ?`)
		args = append(args, *minimum)
	}
	if maximum != nil {
		query.WriteString(` AND ps.rating <= ?`)
		args = append(args, *maximum)
	}
	if len(themes) > 0 {
		query.WriteString(` AND EXISTS (
          SELECT 1 FROM puzzle_themes pt
          WHERE pt.fingerprint = ps.fingerprint
            AND pt.source_id = ps.source_id
            AND pt.theme IN (`)
		query.WriteString(sqlPlaceholders(len(themes)))
		query.WriteString(`))`)
		for _, theme := range themes {
			args = append(args, theme)
		}
	}
	query.WriteString(` ORDER BY ps.fingerprint LIMIT 500`)
	keys, err := c.candidateKeys(ctx, query.String(), args...)
	if err != nil {
		return nil, err
	}
	return c.hydrateCandidates(ctx, keys, limit)
}

func sqlPlaceholders(count int) string {
	if count <= 0 {
		return ""
	}
	return strings.TrimSuffix(strings.Repeat("?,", count), ",")
}

func (c *SQLiteCatalog) candidateKeys(ctx context.Context, query string, args ...any) ([]candidateKey, error) {
	rows, err := c.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	keys := make([]candidateKey, 0, 500)
	seen := make(map[string]struct{}, 500)
	for rows.Next() {
		var key candidateKey
		if err := rows.Scan(&key.fingerprint, &key.sourceID); err != nil {
			return nil, err
		}
		if _, exists := seen[key.fingerprint]; exists {
			continue
		}
		seen[key.fingerprint] = struct{}{}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

func (c *SQLiteCatalog) hydrateCandidates(
	ctx context.Context,
	keys []candidateKey,
	limit int,
) ([]domain.Puzzle, error) {
	rand.Shuffle(len(keys), func(i, j int) {
		keys[i], keys[j] = keys[j], keys[i]
	})
	if len(keys) > limit {
		keys = keys[:limit]
	}
	puzzles := make([]domain.Puzzle, 0, len(keys))
	for _, key := range keys {
		puzzle, err := c.getForSource(ctx, key.fingerprint, key.sourceID)
		if err != nil {
			return nil, err
		}
		puzzles = append(puzzles, puzzle)
	}
	return puzzles, nil
}

func (c *SQLiteCatalog) getForSource(
	ctx context.Context,
	fingerprint string,
	sourceID string,
) (domain.Puzzle, error) {
	puzzle, err := c.Get(ctx, fingerprint)
	if err != nil {
		return domain.Puzzle{}, err
	}
	var ref domain.SourceRef
	var externalID, sourceURL sql.NullString
	var rating, popularity, playCount sql.NullInt64
	var metadata string
	err = c.db.QueryRowContext(
		ctx,
		`SELECT source_id, external_id, rating, popularity, play_count, source_url, metadata_json
         FROM puzzle_sources WHERE fingerprint = ? AND source_id = ?`,
		fingerprint,
		sourceID,
	).Scan(
		&ref.SourceID,
		&externalID,
		&rating,
		&popularity,
		&playCount,
		&sourceURL,
		&metadata,
	)
	if err != nil {
		return domain.Puzzle{}, err
	}
	ref.ExternalID = externalID.String
	ref.URL = sourceURL.String
	if err := json.Unmarshal([]byte(metadata), &ref.Metadata); err != nil {
		return domain.Puzzle{}, err
	}
	if attribution, ok := ref.Metadata["attribution"].(string); ok {
		ref.Attribution = attribution
		delete(ref.Metadata, "attribution")
	}
	puzzle.Sources = []domain.SourceRef{ref}
	puzzle.Rating = intPointer(rating)
	puzzle.Popularity = intPointer(popularity)
	puzzle.PlayCount = intPointer(playCount)
	puzzle.Themes = nil

	rows, err := c.db.QueryContext(
		ctx,
		`SELECT theme FROM puzzle_themes
         WHERE fingerprint = ? AND source_id = ? ORDER BY theme`,
		fingerprint,
		sourceID,
	)
	if err != nil {
		return domain.Puzzle{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var theme string
		if err := rows.Scan(&theme); err != nil {
			return domain.Puzzle{}, err
		}
		puzzle.Themes = append(puzzle.Themes, theme)
	}
	if err := rows.Err(); err != nil {
		return domain.Puzzle{}, err
	}
	return puzzle, nil
}

func intPointer(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	converted := int(value.Int64)
	return &converted
}
