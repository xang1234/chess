# Multi-Format Puzzle Import Design

**Date:** 2026-07-18

## Purpose

Extend the local puzzle catalogue from its Lichess-only import path to a format-neutral import experience that accepts tactical PGN, a versioned canonical JSON format, Lucas Chess `.fns` files, and line-oriented FEN/UCI datasets. Every importer must preserve the catalogue's streaming, bounded-memory, cancellable, generational activation semantics.

The tactical PGN importer is the first implementation slice. Canonical JSON follows, then Lucas and linear adapters. All formats are first-class imports in the desktop UI; users do not have to run an external converter or create intermediate files.

## Scope

This work will:

- Generalize the puzzle import panel and backend entry point.
- Detect supported formats from content, using the extension only as a hint.
- Preserve the specialized Lichess zstd/CSV decoding rules while routing normalized records through the common import lifecycle.
- Add a tactical PGN decoder with an explicit solver convention.
- Define and import `chess-trainer-puzzles/v1` canonical JSON.
- Add a Lucas Chess `.fns` decoder that converts SAN/PGN variations into the canonical solution tree.
- Add a line-oriented six-field FEN plus UCI decoder.
- Resolve a stable logical source ID before activation.
- Permit explicitly rated canonical JSON puzzles to participate in guided adaptive sessions.
- Keep source difficulty metadata separate from trainer ratings.
- Report record-level rejections without sacrificing independently valid records.

This work will not:

- Add arbitrary EPD dialects.
- Add remote download or source-subscription behavior.
- Add a CLI-only conversion workflow.
- Import ordinary played games into the game library.
- Infer a trainer rating from PGN annotations, Lucas difficulty, filenames, or linear-dataset difficulty.
- Interpret source-supplied HTML as trusted UI markup.

## Approaches Considered

### Shared normalized import pipeline

Each format has a small streaming decoder that emits normalized `TrainingPuzzle` records or bounded record-level rejections. One runner owns checksum calculation, progress, generation staging, sealing, activation, cancellation, and the final report. Lichess keeps its specialized zstd/CSV decoder and existing semantic rules but participates in the same lifecycle.

This is the selected approach. It keeps format knowledge at the boundary and gives every format the same atomic catalogue behavior.

### Independent importer implementations

Each format could own its complete staging and activation sequence. This would minimize the initial refactor but duplicate cancellation, progress, checksumming, rejection accounting, and activation safeguards. Behavioral drift would be likely as formats evolve.

### Convert every input to a temporary canonical JSON file

An intermediate-file conversion layer would make the catalogue writer uniform, but it would double local I/O, require temporary-file cleanup, delay first progress, and weaken the existing streaming guarantee. Canonical normalization remains an in-memory boundary instead.

## Import Contract and UI Flow

The existing Lichess-specific panel becomes **Import puzzle collection**.

The file chooser accepts `.zst`, `.pgn`, `.json`, `.fns`, and `.txt`. After selection, the backend inspects the content and returns:

- Normalized absolute file path.
- Filename.
- Detected format.
- Resolved source ID.
- Whether the ID was embedded, fixed by the format, or derived from the path.
- Whether an active generation for that source will be replaced.

The confirmation view presents the source ID as the primary identity. If the format has no embedded or fixed ID, it explicitly labels the normalized absolute path as the fallback source ID. Format and file details remain visible before the user starts the import.

Detection errors disable the start action. Starting an import re-runs detection rather than trusting a stale inspection result. Existing progress, cancellation, completion counts, and bounded error summaries remain available. The backend's generic entry point is `StartPuzzleImport`; the old Lichess-specific entry point delegates to it for compatibility.

Supported-content detection follows unambiguous signatures:

- Zstandard magic plus the required Lichess CSV header identifies Lichess.
- A JSON object carrying the supported schema identifier identifies canonical JSON.
- PGN tag pairs and a tactical `FEN` tag identify tactical PGN.
- A valid six-field FEN before the first `|`, followed by the Lucas three-field structure, identifies `.fns`.
- A valid six-field FEN followed by one or more UCI tokens identifies the linear format.

An extension narrows ambiguous text inspection but does not override contradictory content. Unsupported zstd payloads, JSON schemas, and text grammars produce an inspection error.

## Source Identity

Source identity resolution is deterministic:

1. Lichess uses the fixed source ID `lichess`.
2. Canonical JSON uses `source.id` when it is non-empty.
3. Tactical PGN uses a non-empty `SourceId` tag on the first game. Every later explicit `SourceId` must match; later games may omit it.
4. All other inputs, and JSON or PGN without an embedded ID, use the normalized absolute file path.

The normalized path is cleaned, made absolute, and symlinks are resolved when the operating system can resolve them. The chosen path string is stored in the generation metadata and used verbatim as the fallback source ID.

A conflicting embedded PGN ID discovered after staging is a fatal import error. The staging generation is abandoned and the previous active generation remains unchanged. Reusing a logical source ID with a different immutable source kind remains an error. Reimporting the same valid source ID replaces only that source's active generation.

## Canonical JSON Schema

The canonical file is a JSON object:

```json
{
  "schema": "chess-trainer-puzzles/v1",
  "source": {
    "id": "club-tactics",
    "name": "Club tactics",
    "url": "https://example.com",
    "attribution": "Example author"
  },
  "puzzles": [
    {
      "id": "puzzle-42",
      "displayedFen": "4k3/8/8/8/8/8/4R3/4K3 w - - 0 1",
      "solver": "white",
      "solution": [
        {
          "uci": "e2e7",
          "children": []
        }
      ],
      "rating": 1450,
      "themes": ["back-rank"],
      "metadata": {}
    }
  ]
}
```

### Top-level fields

- `schema` is required and must equal `chess-trainer-puzzles/v1`.
- `source` is optional. `id`, `name`, `url`, and `attribution` are optional bounded strings.
- `puzzles` is required and is streamed with `json.Decoder`; the complete array is never retained in memory.
- Unknown structural fields are rejected. Forward-incompatible changes require a new schema version.

Inspection may make a bounded-memory pass over the JSON token stream to resolve top-level source metadata regardless of object-field order. Import reopens the file and makes the authoritative validation pass.

### Puzzle fields

- `id` is an optional source-local external ID. The import ordinal is used when it is absent.
- `displayedFen` is required and must be a legal six-field FEN.
- `solver` is required and is `white` or `black`. It must equal the displayed position's active color.
- `solution` is a required, non-empty array of canonical move nodes.
- A move node contains required normalized `uci` and optional `children`; unknown fields are rejected.
- Every branch is recursively replayed from its parent position. Illegal moves, invalid promotion syntax, empty UCI, excessive depth, and excessive total nodes reject the puzzle.
- `sourceFen` and `preludeUci` are optional occurrence presentation fields. They must appear together, and applying the legal prelude to `sourceFen` must produce the canonical displayed position.
- `rating` is optional. When present it is an integer from 100 through 4000.
- `themes` is an optional array of normalized, non-empty strings.
- `popularity` and `playCount` are optional integers using the catalogue's existing ranges.
- `url` and `attribution` optionally override corresponding source defaults.
- `metadata` is an optional bounded JSON object and is the only extension point for arbitrary source fields.

Syntactically malformed JSON is fatal because a safe next array boundary cannot be guaranteed. A syntactically valid but semantically invalid puzzle is a record-level rejection, and decoding continues with the next array item.

## Tactical PGN Adapter

Each PGN game is one puzzle and follows this convention:

- `FEN` is required. `SetUp`, when present, must be `1`.
- Exactly one of `White` or `Black` must equal `solver`, compared case-insensitively after trimming.
- The solver color comes from that player tag.
- If the FEN active color is already the solver, the source FEN is also the displayed FEN and there is no prelude.
- If the FEN active color is the opponent, the first legal mainline move becomes `preludeUci`; the resulting position becomes `displayedFen` and must have the solver to move.
- At least one move must remain after any prelude. The remaining mainline moves become one linear canonical solution branch.
- PGN variations, comments, NAGs, clock annotations, and the game result do not create accepted alternatives.
- `PuzzleId` supplies the external ID when present; otherwise the one-based game ordinal is used.
- `SourceId` follows the file-level consistency rule above.
- Other bounded tag pairs may be retained in metadata, excluding redundant full movetext and presentation fields.

A malformed game is rejected when the scanner can recover the next game boundary. An unrecoverable PGN tokenization or read error is fatal. Adjacent tag pairs without intervening blank lines are accepted when the PGN parser can tokenize them legally.

## Lucas `.fns` Adapter

Each non-empty line has three logical fields:

```text
<six-field FEN>|<description>|<SAN/PGN movetext>
```

- The first two `|` delimiters separate the fields; additional characters belong to the movetext field.
- The FEN active color is the solver, and no presentation prelude is created.
- Movetext is parsed from the supplied FEN. Its mainline and legal recursive annotation variations become canonical UCI branches.
- At least one legal solver move is required.
- The description is retained as bounded plain-text metadata and is never rendered as trusted HTML.
- A recognizable normalized filename stem supplies a theme. Generic stems such as `training`, `tactics`, and `puzzles` do not.
- Difficulty text detected in the description is source metadata only.
- The one-based line number is the external ID.

A bad line is independently rejected. Blank lines and lines whose first non-space character is `#` are ignored.

## Linear FEN/UCI Adapter

Each non-comment line follows the approved grammar:

```text
<six-field FEN> <uci1> [uci2 ...] [difficulty]
```

- The first six whitespace-separated tokens form the displayed FEN.
- The displayed FEN active color is the solver.
- One or more legal UCI moves form a single canonical solution branch.
- If the final token is an integer, it is stored as `sourceDifficulty` metadata and is not treated as a move or trainer rating.
- The one-based line number is the external ID.
- Blank lines and lines whose first non-space character is `#` are ignored. Inline comments are not part of the grammar.

Each malformed or illegal line is independently rejected.

## Shared Import Pipeline

Format decoders implement one internal pull contract. Each pull returns one of:

- A normalized `TrainingPuzzle`.
- A record-level rejection containing ordinal and bounded diagnostic context.
- End of input.
- A fatal error.

The shared runner owns the following lifecycle:

1. Detect the format and resolve the source identity.
2. Create a hidden `building` generation.
3. Stream and validate normalized records while calculating the raw-file checksum and reporting progress.
4. Add accepted records through the existing bounded generational writer.
5. Drain writes, seal the generation, and run existing integrity audits.
6. Atomically activate the sealed generation only after successful end-of-input.
7. Emit the terminal immutable job snapshot and trigger bounded cleanup.

Existing fingerprint behavior remains authoritative: stable displayed FEN, solver, and recursively normalized solution tree identify a core. Equal cores from different sources share canonical content while retaining distinct source occurrences and metadata. Existing deterministic duplicate handling within one generation remains unchanged.

Progress exposes detection, parsing/validation, sealing, and activation phases. Byte progress uses the raw file size and a counting reader where available. Record counts are always reported. Checksums cover the exact selected source bytes, including compressed bytes for Lichess.

## Rating and Scheduling Behavior

Any active occurrence with a non-null, validated rating is eligible for guided adaptive sessions. Candidate queries remain based on occurrence rating, but learner rating bounds are derived from all active rated source summaries instead of filtering to source kind `lichess`.

Consequences:

- Lichess behavior is preserved.
- Canonical JSON `rating` participates in guided sessions on the same scale.
- Tactical PGN is unrated.
- Lucas description difficulty is metadata only.
- Linear final-token difficulty is metadata only.
- Unrated puzzles remain available through free practice and source/theme filters.

No numeric field is silently converted into a rating. A future adapter must explicitly define rating semantics before it can populate `PuzzleOccurrence.Rating`.

## Failure, Cancellation, and Resource Limits

- Record-local semantic failures increment rejection counts and retain a bounded sample of diagnostics.
- Conflicting source identity, unsupported schema, unrecoverable syntax, interrupted reads, checksum failures, and catalogue failures are fatal.
- Cancellation abandons the building generation and leaves the previous head unchanged.
- A run with zero accepted puzzles is not sealed or activated, even if every rejection was record-local.
- A fatal error after accepted staging rows still abandons the generation; partial success is never activated.
- Activation remains a short source-head pointer transaction.
- Imports remain globally serialized through the existing puzzle import job coordinator.

Decoders enforce documented constants for maximum record bytes, metadata bytes, tag count, solution depth, and total solution nodes. Limits are deliberately generous for real tactical collections but finite. A record that exceeds a record-local limit is rejected when the decoder can resume safely; loss of framing is fatal.

## Code Boundaries

The implementation should preserve these responsibilities:

- Format detection and source inspection are independent of Wails controllers.
- Each decoder owns only syntax, format-specific conventions, and conversion into canonical domain values.
- Recursive chess-tree validation is shared by JSON and Lucas rather than duplicated.
- The import runner owns staging, checksum, progress, rejection accounting, cancellation, sealing, and activation.
- The import-job service continues to own global exclusion and terminal snapshots.
- The controller owns native file selection and delegates inspection/start/cancel operations.
- The Svelte import session owns view state but does not infer formats or source IDs.

The domain and catalogue layers must not depend on PGN, JSON, Lucas, Wails, or frontend types.

## Verification Strategy

Implementation proceeds test-first in this order:

1. Shared decoder contract, recursive tree validator, detection, and runner behavior.
2. Tactical PGN fixtures covering solver side, no-prelude and prelude positions, adjacent tags, malformed records, illegal moves, empty remaining solutions, IDs, and cancellation.
3. Canonical JSON fixtures covering linear and branched solutions, optional presentation fields, field strictness, streaming behavior, metadata limits, ratings, and fatal syntax.
4. Lucas fixtures covering SAN conversion, variations, comments/results, malformed lines, filename themes, and plain-text metadata.
5. Linear fixtures covering multi-move solutions, promotions, optional difficulty, comments, malformed FEN, illegal UCI, and line-number IDs.
6. Catalogue integration tests covering cross-source deduplication, same-source replacement, zero-valid imports, conflicting IDs, fatal late failure, and preservation of the prior active generation.
7. Import-job and controller tests covering inspection, generic dispatch, compatibility delegation, progress, busy state, cancellation, and file filters.
8. Frontend tests covering format/source confirmation, fallback-path labeling, progress, cancellation, failure summaries, and successful completion.

Final verification runs the repository's Go test suite, frontend unit/type/build checks, and relevant browser interaction tests. Small synthetic fixtures are committed for all formats; third-party puzzle collections are not copied into the repository.
