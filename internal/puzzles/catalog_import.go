package puzzles

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"chess-trainer/internal/domain"

	"github.com/google/uuid"
)

const generationImportBatchSize = 1_000

type generationImportState uint8

const (
	generationImportBuilding generationImportState = iota
	generationImportFinalizing
	generationImportSealed
	generationImportActivated
	generationImportAbandoned
)

type generationRow struct {
	fingerprint   string
	displayedFEN  string
	solver        string
	solutionJSON  string
	solutionPlies int
	externalID    string
	sourceFEN     string
	preludeUCI    string
	rating        *int
	popularity    *int
	playCount     *int
	sourceURL     string
	attribution   string
	metadataJSON  string
	themes        []string
	ordinal       int64
}

type sqliteGenerationImport struct {
	catalog              *SQLiteCatalog
	stage                *generationStage
	source               Source
	generationID         string
	hadExpectedHead      bool
	expectedHead         string
	buffer               []generationRow
	report               ImportReport
	maximumSolutionPlies int
	state                generationImportState
}

var _ GenerationImport = (*sqliteGenerationImport)(nil)

func (c *SQLiteCatalog) BeginImport(
	ctx context.Context,
	source Source,
) (GenerationImport, error) {
	if err := validateSource(source); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	generationID := uuid.NewString()
	tx, err := c.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin generation import: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO sources(source_id, kind)
		 VALUES (?, ?)
		 ON CONFLICT(source_id) DO NOTHING`,
		source.ID,
		source.Kind,
	); err != nil {
		return nil, fmt.Errorf("ensure puzzle source: %w", err)
	}
	var existingKind string
	if err := tx.QueryRowContext(
		ctx,
		`SELECT kind FROM sources WHERE source_id = ?`,
		source.ID,
	).Scan(&existingKind); err != nil {
		return nil, fmt.Errorf("read puzzle source kind: %w", err)
	}
	if existingKind != source.Kind {
		return nil, &SourceKindMismatchError{
			SourceID:      source.ID,
			ExistingKind:  existingKind,
			RequestedKind: source.Kind,
		}
	}

	var expectedHead string
	hadExpectedHead := true
	if err := tx.QueryRowContext(
		ctx,
		`SELECT generation_id FROM source_heads WHERE source_id = ?`,
		source.ID,
	).Scan(&expectedHead); err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("capture puzzle source head: %w", err)
		}
		hadExpectedHead = false
	}
	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO source_generations(
		   generation_id, source_id, status, source_path, started_at
		 ) VALUES (?, ?, 'building', ?, ?)`,
		generationID,
		source.ID,
		source.Path,
		source.StartedAt.Unix(),
	); err != nil {
		return nil, fmt.Errorf("create puzzle source generation: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit generation import start: %w", err)
	}
	stage, err := createGenerationStage(ctx, c.writeDB, generationID)
	if err != nil {
		if _, abandonErr := c.writeDB.ExecContext(
			context.WithoutCancel(ctx),
			`UPDATE source_generations
			 SET status = 'abandoned'
			 WHERE generation_id = ? AND status = 'building'`,
			generationID,
		); abandonErr != nil {
			err = errors.Join(err, fmt.Errorf("abandon generation without stage: %w", abandonErr))
		}
		return nil, err
	}

	return &sqliteGenerationImport{
		catalog:         c,
		stage:           stage,
		source:          source,
		generationID:    generationID,
		hadExpectedHead: hadExpectedHead,
		expectedHead:    expectedHead,
		buffer:          make([]generationRow, 0, generationImportBatchSize),
		state:           generationImportBuilding,
	}, nil
}

func validateSource(source Source) error {
	switch {
	case strings.TrimSpace(source.ID) == "":
		return errors.New("source ID is required")
	case strings.TrimSpace(source.Kind) == "":
		return errors.New("source kind is required")
	case strings.TrimSpace(source.Path) == "":
		return errors.New("source path is required")
	case source.StartedAt.IsZero() || source.StartedAt.Unix() <= 0:
		return errors.New("source start time must be positive")
	default:
		return nil
	}
}

func (s *sqliteGenerationImport) Add(ctx context.Context, puzzle TrainingPuzzle) error {
	if s.state != generationImportBuilding {
		return errors.New("generation import is not building")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if puzzle.Occurrence.SourceID != s.source.ID {
		return fmt.Errorf(
			"puzzle occurrence source ID %q does not match import source %q",
			puzzle.Occurrence.SourceID,
			s.source.ID,
		)
	}
	if puzzle.Occurrence.SourceKind != s.source.Kind {
		return fmt.Errorf(
			"puzzle occurrence source kind %q does not match import kind %q",
			puzzle.Occurrence.SourceKind,
			s.source.Kind,
		)
	}
	if strings.TrimSpace(puzzle.Core.Fingerprint) == "" {
		return errors.New("puzzle fingerprint is required")
	}
	if puzzle.Core.SolutionPlies <= 0 {
		return errors.New("puzzle solution plies must be positive")
	}
	if puzzle.Occurrence.Ordinal <= 0 {
		return errors.New("puzzle occurrence ordinal must be positive")
	}

	normalizedSolution := normalizeNodes(puzzle.Core.Solution)
	solutionJSON, err := json.Marshal(normalizedSolution)
	if err != nil {
		return fmt.Errorf("marshal puzzle solution: %w", err)
	}
	metadata := puzzle.Occurrence.Metadata
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal puzzle occurrence metadata: %w", err)
	}

	s.buffer = append(s.buffer, generationRow{
		fingerprint:   puzzle.Core.Fingerprint,
		displayedFEN:  strings.TrimSpace(puzzle.Core.DisplayedFEN),
		solver:        string(puzzle.Core.Solver),
		solutionJSON:  string(solutionJSON),
		solutionPlies: puzzle.Core.SolutionPlies,
		externalID:    puzzle.Occurrence.ExternalID,
		sourceFEN:     puzzle.Occurrence.SourceFEN,
		preludeUCI:    puzzle.Occurrence.PreludeUCI,
		rating:        cloneGenerationInt(puzzle.Occurrence.Rating),
		popularity:    cloneGenerationInt(puzzle.Occurrence.Popularity),
		playCount:     cloneGenerationInt(puzzle.Occurrence.PlayCount),
		sourceURL:     puzzle.Occurrence.URL,
		attribution:   puzzle.Occurrence.Attribution,
		metadataJSON:  string(metadataJSON),
		themes:        domain.NormalizeThemes(puzzle.Occurrence.Themes),
		ordinal:       puzzle.Occurrence.Ordinal,
	})
	if len(s.buffer) >= generationImportBatchSize {
		return s.flush(ctx)
	}
	return nil
}

func cloneGenerationInt(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func (s *sqliteGenerationImport) Reject(rejection Rejection) {
	if s.state != generationImportBuilding {
		return
	}
	s.report.Rejected++
	if len(s.report.Examples) < 100 {
		s.report.Examples = append(s.report.Examples, rejection)
	}
}

func (s *sqliteGenerationImport) flush(ctx context.Context) error {
	if len(s.buffer) == 0 {
		return nil
	}
	if s.state != generationImportBuilding {
		return errors.New("generation import is not building")
	}
	if err := s.stage.append(ctx, s.buffer); err != nil {
		return err
	}
	s.buffer = s.buffer[:0]
	return nil
}

func generationNullString(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func generationNullableInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func (s *sqliteGenerationImport) Seal(ctx context.Context, checksum string) (ImportReport, error) {
	if s.state != generationImportBuilding {
		return ImportReport{}, errors.New("generation import is not building")
	}
	normalizedChecksum := strings.ToLower(strings.TrimSpace(checksum))
	if normalizedChecksum == "" {
		return ImportReport{}, errors.New("source checksum is required before sealing")
	}
	if err := s.flush(ctx); err != nil {
		return ImportReport{}, err
	}
	s.state = generationImportFinalizing
	if err := s.materialize(ctx); err != nil {
		return ImportReport{}, err
	}
	if err := s.sealMaterializedGeneration(ctx, normalizedChecksum); err != nil {
		return ImportReport{}, err
	}
	s.state = generationImportSealed
	return cloneImportReport(s.report), nil
}

func (s *sqliteGenerationImport) sealMaterializedGeneration(
	ctx context.Context,
	normalizedChecksum string,
) error {
	tx, err := s.catalog.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin puzzle source generation seal: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(
		ctx,
		`INSERT INTO generation_themes(generation_id, theme)
		 SELECT ?, theme
		 FROM (
		   SELECT DISTINCT theme
		   FROM occurrence_themes
		   WHERE generation_id = ?
		 )`,
		s.generationID,
		s.generationID,
	); err != nil {
		return fmt.Errorf("build sealed generation theme facet: %w", err)
	}
	result, err := tx.ExecContext(
		ctx,
		`UPDATE source_generations
		 SET status = 'sealed', checksum = ?, sealed_at = ?, maximum_solution_plies = ?
		 WHERE generation_id = ? AND source_id = ? AND status = 'building'`,
		normalizedChecksum,
		time.Now().Unix(),
		s.maximumSolutionPlies,
		s.generationID,
		s.source.ID,
	)
	if err != nil {
		return fmt.Errorf("seal puzzle source generation: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count sealed puzzle source generations: %w", err)
	}
	if affected != 1 {
		return errors.New("generation import is not building")
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit puzzle source generation seal: %w", err)
	}
	return nil
}

func cloneImportReport(report ImportReport) ImportReport {
	copy := report
	copy.Examples = append([]Rejection(nil), report.Examples...)
	return copy
}

func (s *sqliteGenerationImport) Activate(ctx context.Context) error {
	tx, err := s.catalog.writeDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin puzzle generation activation: %w", err)
	}
	defer tx.Rollback()

	var status string
	var checksum sql.NullString
	if err := tx.QueryRowContext(
		ctx,
		`SELECT status, checksum
		 FROM source_generations
		 WHERE generation_id = ? AND source_id = ?`,
		s.generationID,
		s.source.ID,
	).Scan(&status, &checksum); err != nil {
		return fmt.Errorf("read puzzle generation before activation: %w", err)
	}
	if status != "sealed" || !checksum.Valid || strings.TrimSpace(checksum.String) == "" {
		return errors.New("puzzle generation must be sealed with a checksum before activation")
	}

	var result sql.Result
	if s.hadExpectedHead {
		result, err = tx.ExecContext(
			ctx,
			`UPDATE source_heads
			 SET generation_id = ?
			 WHERE source_id = ? AND generation_id = ?`,
			s.generationID,
			s.source.ID,
			s.expectedHead,
		)
	} else {
		result, err = tx.ExecContext(
			ctx,
			`INSERT INTO source_heads(source_id, generation_id)
			 VALUES (?, ?)
			 ON CONFLICT(source_id) DO NOTHING`,
			s.source.ID,
			s.generationID,
		)
	}
	if err != nil {
		return fmt.Errorf("activate puzzle source generation: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("count activated puzzle source generations: %w", err)
	}
	if affected != 1 {
		return fmt.Errorf("%w: source %q", ErrHeadChanged, s.source.ID)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit puzzle generation activation: %w", err)
	}
	s.state = generationImportActivated
	return nil
}

func (s *sqliteGenerationImport) Abandon(ctx context.Context) (returnErr error) {
	defer func() {
		stageErr := s.stage.closeAndRemove()
		s.stage = nil
		s.buffer = nil
		s.state = generationImportAbandoned
		returnErr = errors.Join(returnErr, stageErr)
	}()

	result, err := s.catalog.writeDB.ExecContext(
		ctx,
		`UPDATE source_generations
		 SET status = 'abandoned'
		 WHERE generation_id = ? AND source_id = ? AND status = 'building'`,
		s.generationID,
		s.source.ID,
	)
	if err != nil {
		return fmt.Errorf("abandon puzzle source generation: %w", err)
	}
	if _, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("count abandoned puzzle source generations: %w", err)
	}
	return nil
}
