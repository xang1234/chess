package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"chess-trainer/internal/domain"
)

const legacySnapshotBatchSize = 128

type legacySnapshotKey struct {
	fingerprint string
	sourceID    string
}

type legacyPuzzleSnapshot struct {
	key        legacySnapshotKey
	sourceKind string
	rating     sql.NullInt64
	themesJSON string
	sourceFEN  sql.NullString
	preludeUCI sql.NullString
}

func BackfillLegacyPuzzleSnapshots(
	ctx context.Context,
	userDB *sql.DB,
	legacyPuzzles *sql.DB,
) error {
	var cursor legacySnapshotKey
	hasCursor := false
	for {
		keys, err := readLegacySnapshotKeys(ctx, userDB, cursor, hasCursor)
		if err != nil {
			return err
		}
		if len(keys) == 0 {
			return nil
		}
		cursor = keys[len(keys)-1]
		hasCursor = true

		snapshots, err := readLegacyPuzzleSnapshots(ctx, legacyPuzzles, keys)
		if err != nil {
			return err
		}
		if len(snapshots) != 0 {
			if err := writeLegacyPuzzleSnapshots(ctx, userDB, keys, snapshots); err != nil {
				return err
			}
		}
		if len(keys) < legacySnapshotBatchSize {
			return nil
		}
	}
}

func readLegacySnapshotKeys(
	ctx context.Context,
	userDB *sql.DB,
	cursor legacySnapshotKey,
	hasCursor bool,
) ([]legacySnapshotKey, error) {
	rows, err := userDB.QueryContext(
		ctx,
		`SELECT fingerprint, source_id
		 FROM (
		   SELECT fingerprint, source_id
		   FROM session_items
		   WHERE source_kind IS NULL
		      OR rating_snapshot IS NULL
		      OR themes_json IS NULL
		      OR source_fen_snapshot IS NULL
		      OR prelude_uci_snapshot IS NULL
		   UNION
		   SELECT fingerprint, source_id
		   FROM attempts
		   WHERE source_kind IS NULL
		      OR rating_snapshot IS NULL
		      OR themes_json IS NULL
		 ) AS history_keys
		 WHERE ? = 0
		    OR fingerprint > ?
		    OR (fingerprint = ? AND source_id > ?)
		 ORDER BY fingerprint, source_id
		 LIMIT ?`,
		boolAsInteger(hasCursor),
		cursor.fingerprint,
		cursor.fingerprint,
		cursor.sourceID,
		legacySnapshotBatchSize,
	)
	if err != nil {
		return nil, fmt.Errorf("read legacy snapshot keys: %w", err)
	}
	keys := make([]legacySnapshotKey, 0, legacySnapshotBatchSize)
	for rows.Next() {
		var key legacySnapshotKey
		if err := rows.Scan(&key.fingerprint, &key.sourceID); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan legacy snapshot key: %w", err)
		}
		keys = append(keys, key)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close legacy snapshot keys: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate legacy snapshot keys: %w", err)
	}
	return keys, nil
}

func legacySnapshotLookupQuery(keys []legacySnapshotKey) (string, []any) {
	var query strings.Builder
	query.WriteString(`WITH requested(fingerprint, source_id) AS (VALUES `)
	args := make([]any, 0, len(keys)*2)
	for index, key := range keys {
		if index != 0 {
			query.WriteByte(',')
		}
		query.WriteString(`(?, ?)`)
		args = append(args, key.fingerprint, key.sourceID)
	}
	query.WriteString(`)
		SELECT requested.fingerprint, requested.source_id, sources.kind,
		       puzzle_sources.rating, puzzles.source_fen, puzzles.prelude_uci,
		       puzzle_themes.theme
		FROM requested
		JOIN puzzle_sources
		  ON puzzle_sources.fingerprint = requested.fingerprint
		 AND puzzle_sources.source_id = requested.source_id
		JOIN sources ON sources.source_id = puzzle_sources.source_id
		JOIN puzzles ON puzzles.fingerprint = puzzle_sources.fingerprint
		LEFT JOIN puzzle_themes
		  ON puzzle_themes.fingerprint = puzzle_sources.fingerprint
		 AND puzzle_themes.source_id = puzzle_sources.source_id
		ORDER BY requested.fingerprint, requested.source_id, puzzle_themes.theme`)
	return query.String(), args
}

func readLegacyPuzzleSnapshots(
	ctx context.Context,
	legacyPuzzles *sql.DB,
	keys []legacySnapshotKey,
) (map[legacySnapshotKey]legacyPuzzleSnapshot, error) {
	query, args := legacySnapshotLookupQuery(keys)
	rows, err := legacyPuzzles.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("read legacy puzzle snapshots: %w", err)
	}

	type snapshotBuilder struct {
		snapshot legacyPuzzleSnapshot
		themes   []string
	}
	builders := make(map[legacySnapshotKey]*snapshotBuilder, len(keys))
	for rows.Next() {
		var snapshot legacyPuzzleSnapshot
		var theme sql.NullString
		if err := rows.Scan(
			&snapshot.key.fingerprint,
			&snapshot.key.sourceID,
			&snapshot.sourceKind,
			&snapshot.rating,
			&snapshot.sourceFEN,
			&snapshot.preludeUCI,
			&theme,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan legacy puzzle snapshot: %w", err)
		}
		builder := builders[snapshot.key]
		if builder == nil {
			builder = &snapshotBuilder{snapshot: snapshot, themes: []string{}}
			builders[snapshot.key] = builder
		}
		if theme.Valid {
			builder.themes = append(builder.themes, theme.String)
		}
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close legacy puzzle snapshots: %w", err)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate legacy puzzle snapshots: %w", err)
	}

	snapshots := make(map[legacySnapshotKey]legacyPuzzleSnapshot, len(builders))
	for key, builder := range builders {
		themesJSON, err := json.Marshal(domain.NormalizeThemes(builder.themes))
		if err != nil {
			return nil, fmt.Errorf("encode legacy themes for %s/%s: %w", key.fingerprint, key.sourceID, err)
		}
		builder.snapshot.themesJSON = string(themesJSON)
		snapshots[key] = builder.snapshot
	}
	return snapshots, nil
}

func writeLegacyPuzzleSnapshots(
	ctx context.Context,
	userDB *sql.DB,
	keys []legacySnapshotKey,
	snapshots map[legacySnapshotKey]legacyPuzzleSnapshot,
) error {
	tx, err := userDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin legacy puzzle snapshot backfill: %w", err)
	}
	defer tx.Rollback()

	updateSessionItem, err := tx.PrepareContext(ctx, `UPDATE session_items SET
		source_kind = COALESCE(source_kind, ?1),
		rating_snapshot = COALESCE(rating_snapshot, ?2),
		themes_json = COALESCE(themes_json, ?3),
		source_fen_snapshot = COALESCE(source_fen_snapshot, ?4),
		prelude_uci_snapshot = COALESCE(prelude_uci_snapshot, ?5)
		WHERE fingerprint = ?6 AND source_id = ?7
		  AND ((source_kind IS NULL AND ?1 IS NOT NULL)
		    OR (rating_snapshot IS NULL AND ?2 IS NOT NULL)
		    OR (themes_json IS NULL AND ?3 IS NOT NULL)
		    OR (source_fen_snapshot IS NULL AND ?4 IS NOT NULL)
		    OR (prelude_uci_snapshot IS NULL AND ?5 IS NOT NULL))`)
	if err != nil {
		return fmt.Errorf("prepare session snapshot backfill: %w", err)
	}
	defer updateSessionItem.Close()

	updateAttempt, err := tx.PrepareContext(ctx, `UPDATE attempts SET
		source_kind = COALESCE(source_kind, ?1),
		rating_snapshot = COALESCE(rating_snapshot, ?2),
		themes_json = COALESCE(themes_json, ?3)
		WHERE fingerprint = ?4 AND source_id = ?5
		  AND ((source_kind IS NULL AND ?1 IS NOT NULL)
		    OR (rating_snapshot IS NULL AND ?2 IS NOT NULL)
		    OR (themes_json IS NULL AND ?3 IS NOT NULL))`)
	if err != nil {
		return fmt.Errorf("prepare attempt snapshot backfill: %w", err)
	}
	defer updateAttempt.Close()

	for _, key := range keys {
		snapshot, ok := snapshots[key]
		if !ok {
			continue
		}
		if _, err := updateSessionItem.ExecContext(
			ctx,
			snapshot.sourceKind,
			snapshot.rating,
			snapshot.themesJSON,
			snapshot.sourceFEN,
			snapshot.preludeUCI,
			key.fingerprint,
			key.sourceID,
		); err != nil {
			return fmt.Errorf("backfill session snapshot for %s/%s: %w", key.fingerprint, key.sourceID, err)
		}
		if _, err := updateAttempt.ExecContext(
			ctx,
			snapshot.sourceKind,
			snapshot.rating,
			snapshot.themesJSON,
			key.fingerprint,
			key.sourceID,
		); err != nil {
			return fmt.Errorf("backfill attempt snapshot for %s/%s: %w", key.fingerprint, key.sourceID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit legacy puzzle snapshot backfill: %w", err)
	}
	return nil
}

func boolAsInteger(value bool) int {
	if value {
		return 1
	}
	return 0
}
