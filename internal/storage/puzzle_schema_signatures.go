package storage

import (
	"context"
	"database/sql"
	"slices"
)

type puzzleColumnSignature struct {
	Name       string
	Type       string
	NotNull    int
	Default    sql.NullString
	PrimaryKey int
	Hidden     int
}

type puzzleSchemaSignature map[string][]puzzleColumnSignature

func readPuzzleSchema(db *sql.DB) (puzzleSchemaSignature, error) {
	rows, err := db.Query(
		`SELECT name
		 FROM sqlite_master
		 WHERE type = 'table' AND lower(substr(name, 1, 7)) <> 'sqlite_'
		 ORDER BY name`,
	)
	if err != nil {
		return nil, err
	}
	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			rows.Close()
			return nil, err
		}
		tables = append(tables, table)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	schema := make(puzzleSchemaSignature, len(tables))
	for _, table := range tables {
		columns, err := db.QueryContext(
			context.Background(),
			`SELECT name, type, "notnull", dflt_value, pk, hidden
			 FROM pragma_table_xinfo(?) ORDER BY cid`,
			table,
		)
		if err != nil {
			return nil, err
		}
		for columns.Next() {
			var column puzzleColumnSignature
			if err := columns.Scan(
				&column.Name,
				&column.Type,
				&column.NotNull,
				&column.Default,
				&column.PrimaryKey,
				&column.Hidden,
			); err != nil {
				columns.Close()
				return nil, err
			}
			schema[table] = append(schema[table], column)
		}
		if err := columns.Close(); err != nil {
			return nil, err
		}
		if err := columns.Err(); err != nil {
			return nil, err
		}
	}
	return schema, nil
}

func equalPuzzleSchemas(got, want puzzleSchemaSignature) bool {
	if len(got) != len(want) {
		return false
	}
	for table, wantColumns := range want {
		gotColumns, ok := got[table]
		if !ok || !slices.Equal(gotColumns, wantColumns) {
			return false
		}
	}
	return true
}

func recognizedPuzzleSchema(versions []int) (puzzleSchemaSignature, bool, bool, bool) {
	switch {
	case slices.Equal(versions, []int{1}):
		return legacyPuzzleSchemaV1, true, false, true
	case slices.Equal(versions, []int{1, 2}):
		return legacyPuzzleSchemaV2, true, false, true
	case slices.Equal(versions, []int{3}):
		return puzzleSchemaV3, false, true, true
	case slices.Equal(versions, []int{CurrentPuzzleSchemaVersion}):
		return currentPuzzleSchema, false, false, true
	default:
		return nil, false, false, false
	}
}

func puzzleColumn(name, columnType string, notNull, primaryKey int) puzzleColumnSignature {
	return puzzleColumnSignature{
		Name:       name,
		Type:       columnType,
		NotNull:    notNull,
		PrimaryKey: primaryKey,
	}
}

func puzzleColumnWithDefault(
	name,
	columnType string,
	notNull,
	primaryKey int,
	defaultValue string,
) puzzleColumnSignature {
	column := puzzleColumn(name, columnType, notNull, primaryKey)
	column.Default = sql.NullString{String: defaultValue, Valid: true}
	return column
}

var legacyPuzzleCommonSchema = puzzleSchemaSignature{
	"schema_migrations": {
		puzzleColumn("version", "INTEGER", 0, 1),
	},
	"sources": {
		puzzleColumn("source_id", "TEXT", 0, 1),
		puzzleColumn("kind", "TEXT", 1, 0),
		puzzleColumn("imported_at", "INTEGER", 1, 0),
		puzzleColumn("source_path", "TEXT", 1, 0),
		puzzleColumn("checksum", "TEXT", 1, 0),
	},
	"puzzles": {
		puzzleColumn("fingerprint", "TEXT", 0, 1),
		puzzleColumn("source_fen", "TEXT", 0, 0),
		puzzleColumn("prelude_uci", "TEXT", 0, 0),
		puzzleColumn("displayed_fen", "TEXT", 1, 0),
		puzzleColumn("solver", "TEXT", 1, 0),
		puzzleColumn("solution_json", "TEXT", 1, 0),
		puzzleColumn("solution_plies", "INTEGER", 1, 0),
	},
	"puzzle_sources": {
		puzzleColumn("fingerprint", "TEXT", 1, 1),
		puzzleColumn("source_id", "TEXT", 1, 2),
		puzzleColumn("external_id", "TEXT", 0, 0),
		puzzleColumn("rating", "INTEGER", 0, 0),
		puzzleColumn("popularity", "INTEGER", 0, 0),
		puzzleColumn("play_count", "INTEGER", 0, 0),
		puzzleColumn("source_url", "TEXT", 0, 0),
		puzzleColumnWithDefault("metadata_json", "TEXT", 1, 0, "'{}'"),
	},
	"puzzle_themes": {
		puzzleColumn("fingerprint", "TEXT", 1, 1),
		puzzleColumn("source_id", "TEXT", 1, 2),
		puzzleColumn("theme", "TEXT", 1, 3),
	},
}

var legacyPuzzleSchemaV1 = withImportStagingSignature([]puzzleColumnSignature{
	puzzleColumn("import_id", "TEXT", 1, 1),
	puzzleColumn("ordinal", "INTEGER", 1, 2),
	puzzleColumn("puzzle_json", "TEXT", 1, 0),
})

var legacyPuzzleSchemaV2 = withImportStagingSignature([]puzzleColumnSignature{
	puzzleColumn("import_id", "TEXT", 1, 1),
	puzzleColumn("ordinal", "INTEGER", 1, 2),
	puzzleColumn("puzzle_json", "TEXT", 1, 0),
	puzzleColumn("fingerprint", "TEXT", 0, 0),
	puzzleColumn("source_fen", "TEXT", 0, 0),
	puzzleColumn("prelude_uci", "TEXT", 0, 0),
	puzzleColumn("displayed_fen", "TEXT", 0, 0),
	puzzleColumn("solver", "TEXT", 0, 0),
	puzzleColumn("solution_json", "TEXT", 0, 0),
	puzzleColumn("solution_plies", "INTEGER", 0, 0),
	puzzleColumn("external_id", "TEXT", 0, 0),
	puzzleColumn("rating", "INTEGER", 0, 0),
	puzzleColumn("popularity", "INTEGER", 0, 0),
	puzzleColumn("play_count", "INTEGER", 0, 0),
	puzzleColumn("source_url", "TEXT", 0, 0),
	puzzleColumn("metadata_json", "TEXT", 0, 0),
	puzzleColumn("themes_json", "TEXT", 0, 0),
})

func withImportStagingSignature(importStaging []puzzleColumnSignature) puzzleSchemaSignature {
	schema := make(puzzleSchemaSignature, len(legacyPuzzleCommonSchema)+1)
	for table, columns := range legacyPuzzleCommonSchema {
		schema[table] = columns
	}
	schema["import_staging"] = importStaging
	return schema
}

var puzzleSchemaV3 = puzzleSchemaSignature{
	"schema_migrations": {
		puzzleColumn("version", "INTEGER", 0, 1),
	},
	"sources": {
		puzzleColumn("source_id", "TEXT", 0, 1),
		puzzleColumn("kind", "TEXT", 1, 0),
	},
	"source_generations": {
		puzzleColumn("generation_id", "TEXT", 0, 1),
		puzzleColumn("source_id", "TEXT", 1, 0),
		puzzleColumn("status", "TEXT", 1, 0),
		puzzleColumn("source_path", "TEXT", 1, 0),
		puzzleColumn("checksum", "TEXT", 0, 0),
		puzzleColumn("started_at", "INTEGER", 1, 0),
		puzzleColumn("sealed_at", "INTEGER", 0, 0),
	},
	"source_heads": {
		puzzleColumn("source_id", "TEXT", 0, 1),
		puzzleColumn("generation_id", "TEXT", 1, 0),
	},
	"puzzle_cores": {
		puzzleColumn("fingerprint", "TEXT", 0, 1),
		puzzleColumn("displayed_fen", "TEXT", 1, 0),
		puzzleColumn("solver", "TEXT", 1, 0),
		puzzleColumn("solution_json", "TEXT", 1, 0),
		puzzleColumn("solution_plies", "INTEGER", 1, 0),
	},
	"puzzle_occurrences": {
		puzzleColumn("generation_id", "TEXT", 1, 2),
		puzzleColumn("fingerprint", "TEXT", 1, 1),
		puzzleColumn("external_id", "TEXT", 0, 0),
		puzzleColumn("source_fen", "TEXT", 0, 0),
		puzzleColumn("prelude_uci", "TEXT", 0, 0),
		puzzleColumn("rating", "INTEGER", 0, 0),
		puzzleColumn("popularity", "INTEGER", 0, 0),
		puzzleColumn("play_count", "INTEGER", 0, 0),
		puzzleColumn("source_url", "TEXT", 0, 0),
		puzzleColumn("attribution", "TEXT", 0, 0),
		puzzleColumnWithDefault("metadata_json", "TEXT", 1, 0, "'{}'"),
		puzzleColumnWithDefault("themes_json", "TEXT", 1, 0, "'[]'"),
		puzzleColumn("ordinal", "INTEGER", 1, 0),
	},
	"occurrence_ratings": {
		puzzleColumn("generation_id", "TEXT", 1, 1),
		puzzleColumn("rating_key", "INTEGER", 1, 2),
		puzzleColumn("fingerprint", "TEXT", 1, 3),
	},
	"occurrence_themes": {
		puzzleColumn("generation_id", "TEXT", 1, 1),
		puzzleColumn("fingerprint", "TEXT", 1, 3),
		puzzleColumn("theme", "TEXT", 1, 2),
	},
	"generation_themes": {
		puzzleColumn("generation_id", "TEXT", 1, 1),
		puzzleColumn("theme", "TEXT", 1, 2),
	},
}

var currentPuzzleSchema = puzzleSchemaWithGenerationMaximum(puzzleSchemaV3)

func puzzleSchemaWithGenerationMaximum(base puzzleSchemaSignature) puzzleSchemaSignature {
	schema := make(puzzleSchemaSignature, len(base))
	for table, columns := range base {
		schema[table] = append([]puzzleColumnSignature(nil), columns...)
	}
	schema["source_generations"] = append(
		schema["source_generations"],
		puzzleColumnWithDefault("maximum_solution_plies", "INTEGER", 1, 0, "0"),
	)
	return schema
}
