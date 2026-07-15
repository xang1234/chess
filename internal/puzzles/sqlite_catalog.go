package puzzles

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"chess-trainer/internal/domain"

	"github.com/google/uuid"
)

const stagingBatchSize = 1000

var secondaryCatalogIndexes = []struct {
	name string
	sql  string
}{
	{
		name: "idx_puzzle_sources_rating",
		sql:  `CREATE INDEX idx_puzzle_sources_rating ON puzzle_sources(source_id, rating)`,
	},
	{
		name: "idx_puzzle_sources_rating_global",
		sql:  `CREATE INDEX idx_puzzle_sources_rating_global ON puzzle_sources(rating, fingerprint)`,
	},
	{
		name: "idx_puzzle_themes_theme",
		sql:  `CREATE INDEX idx_puzzle_themes_theme ON puzzle_themes(source_id, theme, fingerprint)`,
	},
}

type SQLiteCatalog struct {
	db *sql.DB
}

var _ Catalog = (*SQLiteCatalog)(nil)

type stagedRow struct {
	ordinal       int64
	fingerprint   string
	sourceFEN     string
	preludeUCI    string
	displayedFEN  string
	solver        string
	solution      string
	solutionPlies int
	externalID    string
	rating        *int
	popularity    *int
	playCount     *int
	sourceURL     string
	metadata      string
	themes        string
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
	solution, err := json.Marshal(puzzle.Solution)
	if err != nil {
		return err
	}
	themes, err := json.Marshal(puzzle.Themes)
	if err != nil {
		return err
	}
	ref := sourceReference(puzzle.Sources, s.source.ID)
	metadata, err := sourceMetadataJSON(ref)
	if err != nil {
		return err
	}
	s.next++
	s.buffer = append(s.buffer, stagedRow{
		ordinal:       s.next,
		fingerprint:   puzzle.Fingerprint,
		sourceFEN:     puzzle.SourceFEN,
		preludeUCI:    puzzle.PreludeUCI,
		displayedFEN:  puzzle.DisplayedFEN,
		solver:        string(puzzle.Solver),
		solution:      string(solution),
		solutionPlies: solutionDepth(puzzle.Solution),
		externalID:    ref.ExternalID,
		rating:        puzzle.Rating,
		popularity:    puzzle.Popularity,
		playCount:     puzzle.PlayCount,
		sourceURL:     ref.URL,
		metadata:      metadata,
		themes:        string(themes),
	})
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
	statement, err := tx.PrepareContext(
		ctx,
		`INSERT INTO import_staging(
           import_id, ordinal, puzzle_json, fingerprint, source_fen, prelude_uci,
           displayed_fen, solver, solution_json, solution_plies, external_id,
           rating, popularity, play_count, source_url, metadata_json, themes_json
         ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return err
	}
	defer statement.Close()
	for _, row := range s.buffer {
		if _, err := statement.ExecContext(
			ctx,
			s.importID,
			row.ordinal,
			"{}",
			row.fingerprint,
			nullIfEmpty(row.sourceFEN),
			nullIfEmpty(row.preludeUCI),
			row.displayedFEN,
			row.solver,
			row.solution,
			row.solutionPlies,
			nullIfEmpty(row.externalID),
			nullableInt(row.rating),
			nullableInt(row.popularity),
			nullableInt(row.playCount),
			nullIfEmpty(row.sourceURL),
			row.metadata,
			row.themes,
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
	if err := dropSecondaryCatalogIndexes(ctx, tx); err != nil {
		return ImportReport{}, err
	}
	if _, err := tx.ExecContext(ctx, `CREATE INDEX idx_import_staging_fingerprint
        ON import_staging(import_id, fingerprint, ordinal DESC)`); err != nil {
		return ImportReport{}, err
	}

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

	report := s.report
	var stagedCount int64
	if err := tx.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM import_staging WHERE import_id = ?`,
		s.importID,
	).Scan(&stagedCount); err != nil {
		return ImportReport{}, err
	}
	var acceptedCount int64
	if err := tx.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
         FROM (
           SELECT DISTINCT staged.fingerprint
           FROM import_staging staged
           WHERE staged.import_id = ?
             AND NOT EXISTS (
               SELECT 1 FROM puzzles existing
               WHERE existing.fingerprint = staged.fingerprint
             )
         )`,
		s.importID,
	).Scan(&acceptedCount); err != nil {
		return ImportReport{}, err
	}
	report.Accepted += acceptedCount
	report.Duplicates += stagedCount - acceptedCount

	if _, err := tx.ExecContext(
		ctx,
		`INSERT OR IGNORE INTO puzzles(
           fingerprint, source_fen, prelude_uci, displayed_fen, solver,
           solution_json, solution_plies
         )
         SELECT staged.fingerprint, staged.source_fen, staged.prelude_uci,
                staged.displayed_fen, staged.solver, staged.solution_json,
                staged.solution_plies
         FROM import_staging staged
         WHERE staged.import_id = ?
           AND NOT EXISTS (
             SELECT 1 FROM import_staging newer
             WHERE newer.import_id = staged.import_id
               AND newer.fingerprint = staged.fingerprint
               AND newer.ordinal > staged.ordinal
           )`,
		s.importID,
	); err != nil {
		return ImportReport{}, err
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO puzzle_sources(
           fingerprint, source_id, external_id, rating, popularity, play_count,
           source_url, metadata_json
         )
         SELECT staged.fingerprint, ?, staged.external_id, staged.rating,
                staged.popularity, staged.play_count, staged.source_url,
                staged.metadata_json
         FROM import_staging staged
         WHERE staged.import_id = ?
           AND NOT EXISTS (
             SELECT 1 FROM import_staging newer
             WHERE newer.import_id = staged.import_id
               AND newer.fingerprint = staged.fingerprint
               AND newer.ordinal > staged.ordinal
           )
         ON CONFLICT(fingerprint, source_id) DO UPDATE SET
           external_id=excluded.external_id,
           rating=excluded.rating,
           popularity=excluded.popularity,
           play_count=excluded.play_count,
           source_url=excluded.source_url,
           metadata_json=excluded.metadata_json`,
		s.source.ID,
		s.importID,
	); err != nil {
		return ImportReport{}, err
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT OR IGNORE INTO puzzle_themes(fingerprint, source_id, theme)
         SELECT staged.fingerprint, ?, trim(CAST(theme.value AS TEXT))
         FROM import_staging staged,
              json_each(COALESCE(staged.themes_json, '[]')) theme
         WHERE staged.import_id = ?
           AND trim(CAST(theme.value AS TEXT)) <> ''
           AND NOT EXISTS (
             SELECT 1 FROM import_staging newer
             WHERE newer.import_id = staged.import_id
               AND newer.fingerprint = staged.fingerprint
               AND newer.ordinal > staged.ordinal
           )`,
		s.source.ID,
		s.importID,
	); err != nil {
		return ImportReport{}, err
	}
	if _, err := tx.ExecContext(
		ctx,
		`DELETE FROM import_staging WHERE import_id = ?`,
		s.importID,
	); err != nil {
		return ImportReport{}, err
	}
	if _, err := tx.ExecContext(ctx, `DROP INDEX idx_import_staging_fingerprint`); err != nil {
		return ImportReport{}, err
	}
	if err := createSecondaryCatalogIndexes(ctx, tx); err != nil {
		return ImportReport{}, err
	}
	if err := tx.Commit(); err != nil {
		return ImportReport{}, err
	}

	s.report = report
	s.finished = true
	return report, nil
}

func dropSecondaryCatalogIndexes(ctx context.Context, tx *sql.Tx) error {
	for _, index := range secondaryCatalogIndexes {
		if _, err := tx.ExecContext(ctx, `DROP INDEX IF EXISTS `+index.name); err != nil {
			return err
		}
	}
	return nil
}

func createSecondaryCatalogIndexes(ctx context.Context, tx *sql.Tx) error {
	for _, index := range secondaryCatalogIndexes {
		if _, err := tx.ExecContext(ctx, index.sql); err != nil {
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

func nullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
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
	maximumSolutionPlies *int,
	limit int,
) ([]domain.Puzzle, error) {
	if limit <= 0 {
		return []domain.Puzzle{}, nil
	}
	query := strings.Builder{}
	query.WriteString(`SELECT ps.fingerprint, ps.source_id
		FROM puzzle_sources ps
		JOIN puzzles p ON p.fingerprint = ps.fingerprint
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
	if maximumSolutionPlies != nil {
		query.WriteString(` AND p.solution_plies <= ?`)
		args = append(args, *maximumSolutionPlies)
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
