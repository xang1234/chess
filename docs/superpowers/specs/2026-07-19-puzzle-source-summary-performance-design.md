# Puzzle Source Summary Performance Design

**Date:** 2026-07-19

## Purpose

Remove the long pause before unrated Free Practice opens. The delay is not in
puzzle selection: it is caused by `ActiveSourceSummaries` recomputing the
maximum solution length across every active occurrence whenever the practice
filters are requested.

On the current catalogue, that query scans about 7.5 million occurrences and
performs a primary-key lookup into `puzzle_cores` for each one. It takes about
51.79 seconds. The same source-summary query without that aggregation takes
about 0.19 seconds; `ActiveThemes` takes about 0.01 seconds; and selecting and
hydrating 200 candidates from the 1.42-million-puzzle unrated MATE source takes
about 35.9 milliseconds.

The design therefore persists the maximum solution length once per source
generation and makes source-summary reads proportional to the number of active
sources, not the number of puzzles.

## Goals

- Make practice-filter loading independent of catalogue row count.
- Preserve exact `MaximumSolutionPlies` behavior for existing and future
  imports.
- Upgrade existing v3 catalogues in place without deleting, recreating, or
  reimporting them.
- Keep the replaceable catalogue's exact logical and physical schema
  validation.
- Add a regression gate that catches a return to occurrence-wide aggregation.

## Non-Goals

- Change the 200-puzzle candidate pool or its hydration path.
- Change puzzle selection, randomness, rating filters, theme filters, source
  labels, or training semantics.
- Add a general-purpose statistics table or cache framework.
- Optimize import compaction or materialization beyond collecting the statistic
  already available in the compact winner database.

## Chosen Approach

Add `maximum_solution_plies` directly to `source_generations`:

```sql
maximum_solution_plies INTEGER NOT NULL DEFAULT 0
  CHECK (maximum_solution_plies >= 0)
```

The value is a property of an immutable generation. Storing it on that row
keeps ownership explicit and avoids a second table whose lifetime and foreign
key behavior would duplicate the generation lifecycle. A cache or lazy
calculation was rejected because every cold start would retain the current
multi-second failure mode.

Zero is valid for a building, abandoned, or empty generation. A sealed
non-empty generation stores the exact maximum of its winning puzzle rows.
`ActiveSourceSummaries` only reads sealed head generations.

## Safe v3 to v4 Upgrade

Puzzle catalogues use an exact schema signature and contain a single schema
marker rather than a cumulative migration history. The upgrade must preserve
that convention.

Schema v4 will be the new current format. The probe will recognize two distinct
non-legacy states:

- exact v3: supported predecessor eligible for in-place upgrade;
- exact v4: current catalogue eligible for normal opening.

Legacy v1 and v2 behavior remains unchanged. A v3 file with changed columns,
tables, indexes, constraints, or migration markers remains unrecognized and is
never upgraded or removed.

After preflight and while the application owns the data-root lock, startup will
route an exact v3 catalogue through a dedicated upgrade operation before
`OpenPuzzleStore`. That operation opens the existing file read-write,
revalidates the exact v3 schema on that handle, and applies migration 004 in one
transaction:

1. append the new non-null column with a default of zero;
2. backfill every existing generation with the exact maximum from
   `puzzle_occurrences` joined to `puzzle_cores`;
3. delete the v3 schema marker;
4. let the migration runner insert the v4 marker;
5. validate the exact v4 logical and physical schema before normal startup.

The backfill is intentionally expensive once, rather than approximately correct
forever. On the current catalogue its source query is expected to take roughly
the same order of time as the measured 51.79-second summary query. The
transaction changes only the generation rows plus schema metadata, so it does
not rewrite millions of occurrence or core rows. If the process exits or the
backfill fails, SQLite rolls the transaction back to exact v3 and startup can
retry. Failed or unknown catalogues enter the existing recovery path; they are
not classified as legacy and are not deleted.

For a new catalogue, the normal migration runner creates v3, immediately
applies v4, replaces the marker, and validates the same final schema. This keeps
one canonical creation path and one final schema.

## Future Import Lifecycle

During finalization, the importer already owns a compact
`generation_winner.winner_rows` database containing one row per accepted
fingerprint and its `solution_plies`. It will query
`COALESCE(MAX(solution_plies), 0)` there while the winner database is available
and retain that scalar in the generation import state.

The existing seal transaction will write `maximum_solution_plies` in the same
conditional update that changes the generation from `building` to `sealed`.
This makes the statistic and sealed status atomic: an active sealed generation
cannot expose an uncommitted summary. Failed and abandoned imports never become
heads and may retain zero.

Recovery keeps the same rule. Startup may abandon interrupted building
generations; it never synthesizes a head from an incomplete generation.

## Source Summary Read

`ActiveSourceSummaries` will continue to obtain minimum and maximum non-null
ratings through the indexed `occurrence_ratings` subqueries. It will select
`generation.maximum_solution_plies` directly for solution length.

The query will no longer join `puzzle_occurrences` or `puzzle_cores`, and it will
no longer require `GROUP BY`. Its work is bounded by active source heads plus
two indexed rating endpoints per source. Response models and frontend contracts
do not change.

## Tests and Performance Gates

Implementation follows red-green-refactor cycles for:

1. exact v3 recognition as an upgradable, non-legacy predecessor;
2. successful v3-to-v4 migration with exact maxima, unchanged generation IDs,
   source heads, occurrences, cores, ratings, and themes;
3. rollback or refusal for tampered v3 catalogues and validation of exact v4;
4. winner-derived maximum persistence when sealing new imports, including an
   unrated source and an empty generation;
5. source-summary equivalence for mixed rated and unrated active sources;
6. an `EXPLAIN QUERY PLAN` assertion that the summary read does not scan or
   search `puzzle_occurrences` or `puzzle_cores` and does not build a temporary
   grouping tree;
7. a performance-tagged large-catalogue test that keeps a warmed
   `ActiveSourceSummaries` call below 250 milliseconds.

The timing gate is deliberately aimed at the primary fix. Candidate hydration
performance remains covered by its existing tests and is not changed in this
work.

## Delivery and Operational Behavior

The first launch after installing schema v4 may spend about a minute upgrading
a catalogue as large as the current 9.8 GB file. Subsequent launches and every
Free Practice filter load use the persisted statistic and should avoid the
observed 50-second scan. No puzzle sources need to be reimported.

Final verification will run the targeted storage, catalogue, profile, and app
startup tests; the performance-tagged catalogue tests; the full Go test suite;
frontend type and unit checks where generated contracts are touched; and a
production rebuild. The unrelated uncommitted Italian-lesson changes remain
outside this change's staging and commit scope.
