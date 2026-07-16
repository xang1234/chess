package storage

import (
	"database/sql"
	"fmt"
	"slices"
	"sort"
	"strings"
)

var currentPuzzleExplicitIndexes = []string{
	"idx_generations_cleanup@source_generations;unique=0;origin=c;partial=0;xinfo=0:2:status:0:BINARY:1,1:0:generation_id:0:BINARY:1,2:-1:<rowid>:0:BINARY:0",
	"idx_occurrences_generation@puzzle_occurrences;unique=0;origin=c;partial=0;xinfo=0:0:generation_id:0:BINARY:1,1:1:fingerprint:0:BINARY:1",
	"idx_source_heads_generation@source_heads;unique=0;origin=c;partial=0;xinfo=0:1:generation_id:0:BINARY:1,1:-1:<rowid>:0:BINARY:0",
}

// Logical signatures identify primary-key columns and ordinals, while the
// table-storage signature identifies rowid versus WITHOUT ROWID. Neither one
// records key collation/direction or the auxiliary columns carried by a
// WITHOUT ROWID primary key. Canonicalizing every implicit PK/UNIQUE index
// closes that boundary without depending on SQLite's unstable autoindex names.
var currentPuzzleImplicitIndexes = []string{
	"generation_themes;unique=1;origin=pk;partial=0;xinfo=0:0:generation_id:0:BINARY:1,1:1:theme:0:BINARY:1",
	"occurrence_ratings;unique=1;origin=pk;partial=0;xinfo=0:0:generation_id:0:BINARY:1,1:1:rating_key:0:BINARY:1,2:2:fingerprint:0:BINARY:1",
	"occurrence_themes;unique=1;origin=pk;partial=0;xinfo=0:0:generation_id:0:BINARY:1,1:2:theme:0:BINARY:1,2:1:fingerprint:0:BINARY:1",
	"puzzle_cores;unique=1;origin=pk;partial=0;xinfo=0:0:fingerprint:0:BINARY:1,1:-1:<rowid>:0:BINARY:0",
	"puzzle_occurrences;unique=1;origin=pk;partial=0;xinfo=0:1:fingerprint:0:BINARY:1,1:0:generation_id:0:BINARY:1,2:2:external_id:0:BINARY:0,3:3:source_fen:0:BINARY:0,4:4:prelude_uci:0:BINARY:0,5:5:rating:0:BINARY:0,6:6:popularity:0:BINARY:0,7:7:play_count:0:BINARY:0,8:8:source_url:0:BINARY:0,9:9:attribution:0:BINARY:0,10:10:metadata_json:0:BINARY:0,11:11:themes_json:0:BINARY:0,12:12:ordinal:0:BINARY:0",
	"source_generations;unique=1;origin=pk;partial=0;xinfo=0:0:generation_id:0:BINARY:1,1:-1:<rowid>:0:BINARY:0",
	"source_generations;unique=1;origin=u;partial=0;xinfo=0:1:source_id:0:BINARY:1,1:0:generation_id:0:BINARY:1,2:-1:<rowid>:0:BINARY:0",
	"source_heads;unique=1;origin=pk;partial=0;xinfo=0:0:source_id:0:BINARY:1,1:-1:<rowid>:0:BINARY:0",
	"sources;unique=1;origin=pk;partial=0;xinfo=0:0:source_id:0:BINARY:1,1:-1:<rowid>:0:BINARY:0",
}

var currentPuzzleForeignKeys = []string{
	"generation_themes(generation_id)->source_generations(generation_id);update=NO ACTION;delete=CASCADE;match=NONE",
	"occurrence_ratings(fingerprint,generation_id)->puzzle_occurrences(fingerprint,generation_id);update=NO ACTION;delete=CASCADE;match=NONE",
	"occurrence_themes(fingerprint,generation_id)->puzzle_occurrences(fingerprint,generation_id);update=NO ACTION;delete=CASCADE;match=NONE",
	"puzzle_occurrences(fingerprint)->puzzle_cores(fingerprint);update=NO ACTION;delete=NO ACTION;match=NONE",
	"puzzle_occurrences(generation_id)->source_generations(generation_id);update=NO ACTION;delete=CASCADE;match=NONE",
	"source_generations(source_id)->sources(source_id);update=NO ACTION;delete=NO ACTION;match=NONE",
	"source_heads(source_id)->sources(source_id);update=NO ACTION;delete=NO ACTION;match=NONE",
	"source_heads(source_id,generation_id)->source_generations(source_id,generation_id);update=NO ACTION;delete=NO ACTION;match=NONE",
}

var currentPuzzleTableStorage = []struct {
	table        string
	withoutRowID bool
}{
	{table: "puzzle_cores", withoutRowID: false},
	{table: "puzzle_occurrences", withoutRowID: true},
	{table: "occurrence_ratings", withoutRowID: true},
	{table: "occurrence_themes", withoutRowID: true},
	{table: "generation_themes", withoutRowID: true},
}

func validateCurrentPuzzlePhysicalSchema(db *sql.DB) error {
	explicitIndexes, implicitIndexes, err := readCurrentPuzzleIndexes(db)
	if err != nil {
		return fmt.Errorf("read indexes: %w", err)
	}
	if !equalSortedStrings(explicitIndexes, currentPuzzleExplicitIndexes) {
		return fmt.Errorf(
			"explicit indexes are %q, want %q",
			explicitIndexes,
			currentPuzzleExplicitIndexes,
		)
	}
	if !equalSortedStrings(implicitIndexes, currentPuzzleImplicitIndexes) {
		return fmt.Errorf(
			"implicit indexes are %q, want %q",
			implicitIndexes,
			currentPuzzleImplicitIndexes,
		)
	}

	for _, expected := range currentPuzzleTableStorage {
		var withoutRowID int
		if err := db.QueryRow(
			`SELECT wr
			 FROM pragma_table_list
			 WHERE "schema" = 'main' AND name = ?`,
			expected.table,
		).Scan(&withoutRowID); err != nil {
			return fmt.Errorf("read table storage for %s: %w", expected.table, err)
		}
		if got := withoutRowID != 0; got != expected.withoutRowID {
			return fmt.Errorf(
				"table %s WITHOUT ROWID is %t, want %t",
				expected.table,
				got,
				expected.withoutRowID,
			)
		}
	}

	foreignKeys, err := readCurrentPuzzleForeignKeys(db)
	if err != nil {
		return fmt.Errorf("read foreign keys: %w", err)
	}
	if !slices.Equal(foreignKeys, currentPuzzleForeignKeys) {
		return fmt.Errorf(
			"foreign keys are %q, want %q",
			foreignKeys,
			currentPuzzleForeignKeys,
		)
	}
	return nil
}

func readCurrentPuzzleIndexes(db *sql.DB) ([]string, []string, error) {
	rows, err := db.Query(
		`SELECT schema_table.name,
		        index_list.name,
		        index_list."unique",
		        index_list.origin,
		        index_list.partial
		 FROM sqlite_master schema_table
		 JOIN pragma_index_list(schema_table.name) index_list
		 WHERE schema_table.type = 'table'
		 ORDER BY schema_table.name, index_list.origin, index_list.name`,
	)
	if err != nil {
		return nil, nil, err
	}
	type puzzleIndex struct {
		name            string
		table, origin   string
		unique, partial int
	}
	var found []puzzleIndex
	for rows.Next() {
		var index puzzleIndex
		if err := rows.Scan(
			&index.table,
			&index.name,
			&index.unique,
			&index.origin,
			&index.partial,
		); err != nil {
			rows.Close()
			return nil, nil, err
		}
		found = append(found, index)
	}
	if err := rows.Close(); err != nil {
		return nil, nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	var explicit, implicit []string
	for _, index := range found {
		xinfo, err := readPuzzleIndexXInfo(db, index.name)
		if err != nil {
			return nil, nil, err
		}
		name := ""
		if index.origin == "c" {
			name = index.name + "@"
		}
		signature := fmt.Sprintf(
			"%s%s;unique=%d;origin=%s;partial=%d;xinfo=%s",
			name,
			index.table,
			index.unique,
			index.origin,
			index.partial,
			strings.Join(xinfo, ","),
		)
		if index.origin == "c" {
			explicit = append(explicit, signature)
		} else {
			implicit = append(implicit, signature)
		}
	}
	return explicit, implicit, nil
}

func readPuzzleIndexXInfo(db *sql.DB, index string) ([]string, error) {
	rows, err := db.Query(
		`SELECT seqno, cid, name, "desc", coll, key
		 FROM pragma_index_xinfo(?) ORDER BY seqno`,
		index,
	)
	if err != nil {
		return nil, fmt.Errorf("read index %s xinfo: %w", index, err)
	}
	defer rows.Close()
	var signatures []string
	for rows.Next() {
		var sequence, columnID, descending, key int
		var column, collation sql.NullString
		if err := rows.Scan(
			&sequence,
			&columnID,
			&column,
			&descending,
			&collation,
			&key,
		); err != nil {
			return nil, fmt.Errorf("scan index %s xinfo: %w", index, err)
		}
		columnName := "<rowid>"
		if column.Valid {
			columnName = column.String
		} else if columnID != -1 {
			columnName = "<expression>"
		}
		collationName := "<none>"
		if collation.Valid {
			collationName = collation.String
		}
		signatures = append(signatures, fmt.Sprintf(
			"%d:%d:%s:%d:%s:%d",
			sequence,
			columnID,
			columnName,
			descending,
			collationName,
			key,
		))
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate index %s xinfo: %w", index, err)
	}
	return signatures, nil
}

func equalSortedStrings(got, want []string) bool {
	got = append([]string(nil), got...)
	want = append([]string(nil), want...)
	sort.Strings(got)
	sort.Strings(want)
	return slices.Equal(got, want)
}

func readCurrentPuzzleForeignKeys(db *sql.DB) ([]string, error) {
	tables := make([]string, 0, len(currentPuzzleSchema))
	for table := range currentPuzzleSchema {
		tables = append(tables, table)
	}
	sort.Strings(tables)

	var signatures []string
	for _, table := range tables {
		tableSignatures, err := readPuzzleTableForeignKeys(db, table)
		if err != nil {
			return nil, err
		}
		signatures = append(signatures, tableSignatures...)
	}
	sort.Strings(signatures)
	return signatures, nil
}

func readPuzzleTableForeignKeys(db *sql.DB, table string) ([]string, error) {
	rows, err := db.Query(
		`SELECT id, seq, "table", "from", "to", on_update, on_delete, match
		 FROM pragma_foreign_key_list(?)
		 ORDER BY id, seq`,
		table,
	)
	if err != nil {
		return nil, fmt.Errorf("read table %s foreign keys: %w", table, err)
	}
	type foreignKey struct {
		parent  string
		from    []string
		to      []string
		update  string
		delete  string
		match   string
		lastSeq int
	}
	foreignKeys := make(map[int]*foreignKey)
	for rows.Next() {
		var id, sequence int
		var parent, from, to, update, deleteAction, match string
		if err := rows.Scan(
			&id,
			&sequence,
			&parent,
			&from,
			&to,
			&update,
			&deleteAction,
			&match,
		); err != nil {
			rows.Close()
			return nil, fmt.Errorf("scan table %s foreign key: %w", table, err)
		}
		key := foreignKeys[id]
		if key == nil {
			key = &foreignKey{
				parent:  parent,
				update:  update,
				delete:  deleteAction,
				match:   match,
				lastSeq: -1,
			}
			foreignKeys[id] = key
		}
		if key.parent != parent || key.update != update || key.delete != deleteAction || key.match != match {
			rows.Close()
			return nil, fmt.Errorf("table %s foreign key %d has inconsistent rows", table, id)
		}
		if sequence != key.lastSeq+1 {
			rows.Close()
			return nil, fmt.Errorf(
				"table %s foreign key %d sequence is %d after %d",
				table,
				id,
				sequence,
				key.lastSeq,
			)
		}
		key.from = append(key.from, from)
		key.to = append(key.to, to)
		key.lastSeq = sequence
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	signatures := make([]string, 0, len(foreignKeys))
	for _, key := range foreignKeys {
		signatures = append(signatures, fmt.Sprintf(
			"%s(%s)->%s(%s);update=%s;delete=%s;match=%s",
			table,
			strings.Join(key.from, ","),
			key.parent,
			strings.Join(key.to, ","),
			key.update,
			key.delete,
			key.match,
		))
	}
	sort.Strings(signatures)
	return signatures, nil
}
