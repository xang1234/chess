package puzzles

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type generationStage struct {
	db   *sql.DB
	path string
}

const generationStageSchemaSQL = `CREATE TABLE staged_rows (
	row_id INTEGER PRIMARY KEY,
	ordinal INTEGER NOT NULL,
	fingerprint TEXT NOT NULL,
	displayed_fen TEXT NOT NULL,
	solver TEXT NOT NULL,
	solution_json TEXT NOT NULL,
	solution_plies INTEGER NOT NULL,
	external_id TEXT,
	source_fen TEXT,
	prelude_uci TEXT,
	rating INTEGER,
	popularity INTEGER,
	play_count INTEGER,
	source_url TEXT,
	attribution TEXT,
	metadata_json TEXT NOT NULL,
	themes_json TEXT NOT NULL
)`

func createGenerationStage(
	ctx context.Context,
	mainDB *sql.DB,
	generationID string,
) (*generationStage, error) {
	db, stagePath, err := createGenerationArtifact(
		ctx,
		mainDB,
		"stage",
		generationID,
		generationStageSchemaSQL,
	)
	if err != nil {
		return nil, err
	}
	return &generationStage{db: db, path: stagePath}, nil
}

func createGenerationArtifact(
	ctx context.Context,
	mainDB *sql.DB,
	kind string,
	generationID string,
	schemaSQL string,
) (db *sql.DB, artifactPath string, returnErr error) {
	mainPath, err := puzzleDatabasePath(ctx, mainDB)
	if err != nil {
		return nil, "", err
	}
	artifactPath = mainPath + "." + kind + "-" + generationID + ".sqlite"
	absolutePath, err := filepath.Abs(artifactPath)
	if err != nil {
		return nil, "", fmt.Errorf("resolve generation %s path: %w", kind, err)
	}
	uri := url.URL{Scheme: "file", Path: filepath.ToSlash(absolutePath)}
	query := uri.Query()
	query.Set("mode", "rwc")
	query.Add("_pragma", "busy_timeout(5000)")
	query.Add("_pragma", "cache_size(-65536)")
	uri.RawQuery = query.Encode()
	db, err = sql.Open("sqlite", uri.String())
	if err != nil {
		return nil, "", fmt.Errorf("open generation %s: %w", kind, err)
	}
	defer func() {
		if returnErr == nil {
			return
		}
		returnErr = errors.Join(
			returnErr,
			closeAndRemoveGenerationArtifact(db, artifactPath),
		)
	}()
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.PingContext(ctx); err != nil {
		return nil, "", fmt.Errorf("connect generation %s: %w", kind, err)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA temp_store=FILE`); err != nil {
		return nil, "", fmt.Errorf("set generation %s temp store: %w", kind, err)
	}
	var lockingMode, journalMode string
	if err := db.QueryRowContext(ctx, `PRAGMA locking_mode=EXCLUSIVE`).Scan(&lockingMode); err != nil {
		return nil, "", fmt.Errorf("set generation %s locking mode: %w", kind, err)
	}
	if err := db.QueryRowContext(ctx, `PRAGMA journal_mode=OFF`).Scan(&journalMode); err != nil {
		return nil, "", fmt.Errorf("set generation %s journal mode: %w", kind, err)
	}
	if _, err := db.ExecContext(ctx, `PRAGMA synchronous=OFF`); err != nil {
		return nil, "", fmt.Errorf("set generation %s synchronous mode: %w", kind, err)
	}
	if lockingMode != "exclusive" || journalMode != "off" {
		return nil, "", fmt.Errorf(
			"generation %s modes locking=%q journal=%q, want exclusive/off",
			kind,
			lockingMode,
			journalMode,
		)
	}
	if _, err := db.ExecContext(ctx, schemaSQL); err != nil {
		return nil, "", fmt.Errorf("create generation %s schema: %w", kind, err)
	}
	return db, artifactPath, nil
}

func puzzleDatabasePath(ctx context.Context, db *sql.DB) (string, error) {
	var mainPath string
	if err := db.QueryRowContext(
		ctx,
		`SELECT file FROM pragma_database_list WHERE name = 'main'`,
	).Scan(&mainPath); err != nil {
		return "", fmt.Errorf("locate puzzle catalogue database: %w", err)
	}
	if strings.TrimSpace(mainPath) == "" {
		return "", errors.New("puzzle catalogue database path is empty")
	}
	return mainPath, nil
}

func removeOrphanGenerationStages(ctx context.Context, db *sql.DB) error {
	mainPath, err := puzzleDatabasePath(ctx, db)
	if err != nil {
		return err
	}
	directory := filepath.Dir(mainPath)
	prefixes := []string{
		filepath.Base(mainPath) + ".stage-",
		filepath.Base(mainPath) + ".winner-",
	}
	entries, err := os.ReadDir(directory)
	if err != nil {
		return fmt.Errorf("list orphan generation artifacts: %w", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !isGenerationArtifact(name, prefixes) {
			continue
		}
		if err := os.Remove(filepath.Join(directory, name)); err != nil &&
			!errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove orphan generation artifact %q: %w", name, err)
		}
	}
	return nil
}

func isGenerationArtifact(name string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		remainder := strings.TrimPrefix(name, prefix)
		for _, suffix := range []string{".sqlite", ".sqlite-journal", ".sqlite-wal", ".sqlite-shm"} {
			if !strings.HasSuffix(remainder, suffix) {
				continue
			}
			generationID := strings.TrimSuffix(remainder, suffix)
			parsed, err := uuid.Parse(generationID)
			return err == nil && parsed.String() == generationID
		}
	}
	return false
}

func (s *generationStage) append(ctx context.Context, rows []generationRow) error {
	if len(rows) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin generation stage batch: %w", err)
	}
	defer tx.Rollback()
	insert, err := tx.PrepareContext(ctx, `INSERT INTO staged_rows(
		ordinal, fingerprint, displayed_fen, solver, solution_json, solution_plies,
		external_id, source_fen, prelude_uci, rating, popularity, play_count,
		source_url, attribution, metadata_json, themes_json
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare generation stage insert: %w", err)
	}
	defer insert.Close()
	for _, row := range rows {
		themesJSON, err := json.Marshal(row.themes)
		if err != nil {
			return fmt.Errorf("marshal generation stage themes: %w", err)
		}
		if _, err := insert.ExecContext(
			ctx,
			row.ordinal,
			row.fingerprint,
			row.displayedFEN,
			row.solver,
			row.solutionJSON,
			row.solutionPlies,
			generationNullString(row.externalID),
			generationNullString(row.sourceFEN),
			generationNullString(row.preludeUCI),
			generationNullableInt(row.rating),
			generationNullableInt(row.popularity),
			generationNullableInt(row.playCount),
			generationNullString(row.sourceURL),
			generationNullString(row.attribution),
			row.metadataJSON,
			string(themesJSON),
		); err != nil {
			return fmt.Errorf("append generation stage row %d: %w", row.ordinal, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit generation stage batch: %w", err)
	}
	return nil
}

func (s *generationStage) close() error {
	if s == nil || s.db == nil {
		return nil
	}
	err := s.db.Close()
	s.db = nil
	return err
}

func (s *generationStage) remove() error {
	if s == nil {
		return nil
	}
	return removeGenerationArtifact(s.path)
}

func (s *generationStage) closeAndRemove() error {
	if s == nil {
		return nil
	}
	return errors.Join(s.close(), s.remove())
}

func closeAndRemoveGenerationArtifact(db *sql.DB, path string) error {
	var closeErr error
	if db != nil {
		closeErr = db.Close()
	}
	return errors.Join(closeErr, removeGenerationArtifact(path))
}

func removeGenerationArtifact(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	var returnErr error
	for _, artifactPath := range []string{
		path,
		path + "-journal",
		path + "-wal",
		path + "-shm",
	} {
		if err := os.Remove(artifactPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			returnErr = errors.Join(returnErr, err)
		}
	}
	return returnErr
}
