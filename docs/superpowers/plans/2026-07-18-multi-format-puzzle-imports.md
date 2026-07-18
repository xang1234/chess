# Multi-Format Puzzle Imports Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Import tactical PGN, canonical JSON, Lucas `.fns`, and linear FEN/UCI puzzle collections directly from the desktop UI while preserving the existing atomic generational catalogue behavior.

**Architecture:** A `CollectionImporter` owns content inspection and the shared checksum/stage/seal/activate lifecycle. Small format adapters expose a probe plus a streaming decoder and emit normalized `TrainingPuzzle` records. The import-job service continues to serialize jobs, while the controller resolves source identity from inspection before starting a typed job and the Svelte session displays that inspection.

**Tech Stack:** Go 1.25 language level, `github.com/corentings/chess/v2`, zstd/CSV, `encoding/json`, SQLite generational catalogue, Wails v2.12.0, Svelte 3, TypeScript, Vitest, Testing Library, Playwright.

## Global Constraints

- Implement in this order: shared foundation, tactical PGN, canonical JSON, Lucas `.fns`, linear FEN/UCI, then UI integration.
- Supported formats are `lichess`, `tactical-pgn`, `canonical-json`, `lucas-fns`, and `linear-fen-uci`; extensions are hints, never sole authority.
- Canonical JSON schema is exactly `chess-trainer-puzzles/v1`; explicit ratings are integers from 100 through 4000.
- Linear grammar is exactly `<six-field FEN> <uci1> [uci2 ...] [difficulty]`; blank lines and lines beginning with `#` are ignored.
- Source ID priority is fixed Lichess ID, embedded JSON/first-game PGN ID, then normalized absolute path with resolvable symlinks evaluated.
- Reimport replaces only the resolved source ID; cancellation, fatal errors, and zero-valid imports leave the prior head untouched.
- Lucas and linear difficulty values are metadata only and never populate `PuzzleOccurrence.Rating`.
- Any non-null validated occurrence rating participates in guided scheduling and learner-rating bounds.
- Limits are: 64 KiB per PGN game, 1 MiB per line record, 2 MiB per canonical JSON puzzle, 64 KiB metadata, 128 PGN tags, solution depth 256, and total solution nodes 512.
- Source descriptions are plain text. Never render imported HTML with Svelte `{@html}`.
- Keep every task green before committing. Preserve unrelated existing edits in `app_test.go` and `normal_controller.go` while integrating the broader file filter.

---

## File Structure

### Backend foundation

- Create `internal/puzzles/solution_tree.go`: FEN normalization, solver derivation, linear-tree construction, recursive legality/limit validation, and fingerprint finalization.
- Create `internal/puzzles/collection_importer.go`: public inspection types, adapter/decoder contracts, path normalization, adapter registry, shared import runner, checksum/progress, and format-specific importer wrapper.
- Modify `internal/puzzles/lichess_importer.go`: retain Lichess parsing semantics behind the adapter contract and keep `LichessImporter` as a compatibility wrapper.
- Create one focused adapter file and test file for each new format:
  - `internal/puzzles/tactical_pgn.go`
  - `internal/puzzles/canonical_json.go`
  - `internal/puzzles/lucas_fns.go`
  - `internal/puzzles/linear_fen_uci.go`
- Modify `internal/puzzles/catalog_models.go` and tests: use all rated source summaries for learner bounds.
- Modify `internal/importjob/service.go` and tests: register new kinds and preserve monotonic counters plus monotonic phases.

### Application/UI integration

- Modify `internal/app/services.go` and tests: compose the default adapter set and register one format wrapper per job kind.
- Modify `normal_controller.go` and `app_test.go`: inspect selected files, start by path, keep the legacy Lichess method, and offer all supported extensions.
- Regenerate `frontend/wailsjs/go/main/NormalController.{js,d.ts}` and `frontend/wailsjs/go/models.ts` with Wails; do not hand-edit generated bindings.
- Modify `frontend/src/lib/api-contract.ts`, `api.ts`, `test-fakes.ts`, and tests: add inspection/progress/report contracts.
- Modify `frontend/src/lib/import-session.ts` and tests: inspect on selection and start the generic import.
- Modify `frontend/src/components/import/ImportPanel.svelte` and tests: source-first confirmation, format/path details, phase-aware progress, and rejection examples.
- Modify `frontend/tests/test-backend.ts` and `frontend/tests/trainer.spec.ts`: exercise the generic bindings and visible confirmation.
- Create `docs/operations/puzzle-import-formats.md`; update `README.md` and `docs/operations/local-build.md`.

---

### Task 1: Canonical solution normalization and recursive validation

**Files:**
- Create: `internal/puzzles/solution_tree.go`
- Create: `internal/puzzles/solution_tree_test.go`
- Modify: `internal/puzzles/lichess_importer.go:332-355`

**Interfaces:**
- Consumes: `chessrules.Rules.ApplyUCI`, `domain.MoveNode`, `PuzzleCore`, and `CoreFingerprint`.
- Produces:
  - `normalizeFEN(rules chessrules.Rules, fen string) (string, error)`
  - `solverFromFEN(fen string) (domain.Color, error)`
  - `linearSolution(moves []string) []domain.MoveNode`
  - `normalizeSolutionTree(rules chessrules.Rules, displayedFEN string, nodes []domain.MoveNode) ([]domain.MoveNode, int, error)`
  - `finalizeCore(rules chessrules.Rules, displayedFEN string, solver domain.Color, nodes []domain.MoveNode) (PuzzleCore, error)`

- [ ] **Step 1: Write failing tests for a legal branched tree, normalization, and maximum depth**

```go
func TestNormalizeSolutionTreeValidatesEveryBranch(t *testing.T) {
    fen := "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"
    nodes := []domain.MoveNode{{
        UCI: " E2E4 ",
        Children: []domain.MoveNode{
            {UCI: "e8e7"},
            {UCI: "e8f7"},
        },
    }}

    got, plies, err := normalizeSolutionTree(chessrules.Rules{}, fen, nodes)
    if err != nil { t.Fatal(err) }
    if got[0].UCI != "e2e4" || len(got[0].Children) != 2 || plies != 2 {
        t.Fatalf("normalized = %+v, plies = %d", got, plies)
    }
}

func TestNormalizeSolutionTreeRejectsIllegalSibling(t *testing.T) {
    fen := "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"
    _, _, err := normalizeSolutionTree(chessrules.Rules{}, fen, []domain.MoveNode{{
        UCI: "e2e4",
        Children: []domain.MoveNode{{UCI: "e8e7"}, {UCI: "a8a7"}},
    }})
    if err == nil || !strings.Contains(err.Error(), "a8a7") {
        t.Fatalf("error = %v, want illegal branch context", err)
    }
}
```

Add table cases for empty roots, duplicate sibling UCI, 257-ply depth, 513 total nodes, malformed FEN, solver/active-color mismatch in `finalizeCore`, and canonical lowercase fingerprint stability.

- [ ] **Step 2: Run the focused test and verify RED**

Run: `go test ./internal/puzzles -run 'TestNormalizeSolutionTree|TestFinalizeCore' -count=1`

Expected: build failure because the normalization functions do not exist.

- [ ] **Step 3: Implement the bounded recursive validator**

Use these constants and recursion contract:

```go
const (
    maxSolutionDepth = 256
    maxSolutionNodes = 512
)

func normalizeSolutionTree(
    rules chessrules.Rules,
    displayedFEN string,
    nodes []domain.MoveNode,
) ([]domain.MoveNode, int, error) {
    if len(nodes) == 0 { return nil, 0, errors.New("solution is empty") }
    total := 0
    normalized, depth, err := normalizeSolutionLevel(rules, displayedFEN, nodes, 1, &total)
    if err != nil { return nil, 0, err }
    return normalized, depth, nil
}
```

At each level, trim/lowercase UCI, reject empty or duplicate siblings, increment the shared node count before comparing with 512, compare depth with 256, call `rules.ApplyUCI(parentFEN, uci)`, and recurse from that resulting FEN. Return the maximum root-to-leaf depth as `SolutionPlies`. `finalizeCore` must canonicalize the FEN, derive and compare active color, validate the tree, then call `CoreFingerprint`.

Move the existing `solverFromFEN` and `moveLine` behavior into this file under the new names; keep thin aliases only until all current call sites are migrated in Task 3.

- [ ] **Step 4: Run focused and package tests and verify GREEN**

Run: `go test ./internal/puzzles -run 'TestNormalizeSolutionTree|TestFinalizeCore|TestLichessImporterNormalizesSetupMove' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the shared chess normalization boundary**

```bash
git add internal/puzzles/solution_tree.go internal/puzzles/solution_tree_test.go internal/puzzles/lichess_importer.go
git commit -m "refactor: centralize puzzle solution validation"
```

---

### Task 2: Adapter contract, inspection registry, and shared atomic runner

**Files:**
- Create: `internal/puzzles/collection_importer.go`
- Create: `internal/puzzles/collection_importer_test.go`
- Modify: `internal/puzzles/lichess_importer.go:36-61`
- Modify: `internal/importjob/service.go:13-52,175-195`
- Modify: `internal/importjob/service_test.go`

**Interfaces:**
- Consumes: `CatalogWriter`, optional `CatalogReader.ActiveSourceSummaries`, `GenerationImport`, `storage.RequiredImportBytes`, and `ProgressSink`.
- Produces:

```go
type ImportFormat string
const (
    FormatLichess       ImportFormat = "lichess"
    FormatTacticalPGN   ImportFormat = "tactical-pgn"
    FormatCanonicalJSON ImportFormat = "canonical-json"
    FormatLucasFNS      ImportFormat = "lucas-fns"
    FormatLinearFENUCI  ImportFormat = "linear-fen-uci"
)

type SourceIDOrigin string
const (
    SourceIDFixed    SourceIDOrigin = "fixed"
    SourceIDEmbedded SourceIDOrigin = "embedded"
    SourceIDPath     SourceIDOrigin = "path"
)

type ImportInspection struct {
    Path             string         `json:"path"`
    Filename         string         `json:"filename"`
    Format           ImportFormat   `json:"format"`
    SourceID         string         `json:"sourceId"`
    SourceIDOrigin   SourceIDOrigin `json:"sourceIdOrigin"`
    SourceName       string         `json:"sourceName,omitempty"`
    URL              string         `json:"url,omitempty"`
    Attribution      string         `json:"attribution,omitempty"`
    ReplacesExisting bool           `json:"replacesExisting"`
}

type DecodedRecord struct {
    Puzzle    *TrainingPuzzle
    Rejection *Rejection
}

type PuzzleDecoder interface {
    Next(context.Context) (DecodedRecord, error)
    Close() error
}

type PuzzleAdapter interface {
    Format() ImportFormat
    Inspect(context.Context, string) (ImportInspection, bool, error)
    NewDecoder(io.Reader, ImportInspection) (PuzzleDecoder, error)
}
```

`CollectionImporter.Inspect(ctx, path)` returns a public inspection. `CollectionImporter.ImportFormat(ctx, format, sourceID, path, sink)` performs the authoritative second inspection and import. `FormatImporter{Collection, Format}` implements the existing import-job `Importer` shape without importing `internal/importjob`.

- [ ] **Step 1: Write failing registry tests with fake adapters**

Test that inspection resolves an absolute symlink target, rejects no-match, uses a matching extension to narrow two content matches, rejects ambiguity when neither match agrees with the extension, marks an existing source, and does not choose an adapter based only on extension. Use a fake adapter whose `Inspect` records the normalized path and returns `matched` only when file content equals its signature.

```go
func TestCollectionImporterInspectUsesContentAndPathFallback(t *testing.T) {
    path := filepath.Join(t.TempDir(), "misleading.pgn")
    if err := os.WriteFile(path, []byte("linear-signature"), 0o600); err != nil { t.Fatal(err) }
    importer := CollectionImporter{Adapters: []PuzzleAdapter{
        fakeAdapter{format: FormatTacticalPGN, signature: "pgn-signature"},
        fakeAdapter{format: FormatLinearFENUCI, signature: "linear-signature"},
    }}

    got, err := importer.Inspect(context.Background(), path)
    if err != nil { t.Fatal(err) }
    if got.Format != FormatLinearFENUCI || got.SourceID != got.Path || got.SourceIDOrigin != SourceIDPath {
        t.Fatalf("inspection = %+v", got)
    }
}
```

- [ ] **Step 2: Write failing runner tests for success, rejection, zero-valid, cancellation, checksum, and late fatal error**

Use the existing capture generation pattern. The fake decoder must emit one puzzle, one rejection, and `io.EOF`; a second fake must emit only rejections; a third must emit a valid record and then `errors.New("truncated source")`.

Assert:

- Success calls `Seal` once and `Activate` once with SHA-256 of exact file bytes.
- Record rejection reaches `GenerationImport.Reject` in order.
- Zero-valid and late-fatal cases call `Abandon`, never `Seal` or `Activate`.
- Cancellation returns `context.Canceled` and abandons with a non-cancelled cleanup context.
- Progress advances through `detecting`, `parsing`, `sealing`, and `activating` with final byte count equal to file size.

- [ ] **Step 3: Run the new tests and verify RED**

Run: `go test ./internal/puzzles ./internal/importjob -run 'TestCollectionImporter|TestImportProgressPhase' -count=1`

Expected: build failures for missing contracts and phase fields.

- [ ] **Step 4: Implement path normalization and adapter selection**

`normalizeImportPath` must trim, reject empty, call `filepath.Abs`, `filepath.Clean`, and replace the result with `filepath.EvalSymlinks` only when that succeeds. `Inspect` calls every adapter. One content match wins regardless of extension; with multiple matches, use the adapter whose documented extension matches, otherwise return ambiguity. For path-origin matches, overwrite `SourceID` with the normalized path. Compare `SourceID` against `ActiveSourceSummaries` when a reader is configured.

- [ ] **Step 5: Implement the shared runner with one cleanup path**

Move `countingReader` and `abandonImportTimeout` into `collection_importer.go`. Add:

```go
type ImportPhase string
const (
    ImportDetecting  ImportPhase = "detecting"
    ImportParsing    ImportPhase = "parsing"
    ImportSealing    ImportPhase = "sealing"
    ImportActivating ImportPhase = "activating"
)

type Progress struct {
    Phase      ImportPhase `json:"phase"`
    RowsRead   int64       `json:"rowsRead"`
    BytesRead  int64       `json:"bytesRead"`
    TotalBytes int64       `json:"totalBytes"`
}
```

The runner must re-inspect using the requested format, compare the normalized path and source ID with the job request, perform the existing disk-space check, begin and wrap the generation with `newOrderedGenerationImport`, tee raw bytes into SHA-256, and pull until `io.EOF`. Reject a `DecodedRecord` with both/neither fields. Track normalized puzzle count before sealing so zero valid records abandon with `ErrNoValidPuzzles`. Close the decoder, drain the tee, emit sealing/activating phases, and preserve the existing seal-before-activate rule.

- [ ] **Step 6: Make job progress phase-monotonic**

Add kind constants for all five formats. In `recordProgress`, keep `RowsRead`, `BytesRead`, and `TotalBytes` monotonic with `max`; advance `Phase` only when the new phase's rank is equal or later. Extend `cloneResult` tests so an older parsing callback cannot overwrite `sealing`.

- [ ] **Step 7: Run backend tests and verify GREEN**

Run: `go test ./internal/puzzles ./internal/importjob -count=1`

Expected: PASS, including existing import-job ordering/cancellation tests.

- [ ] **Step 8: Commit the shared import lifecycle**

```bash
git add internal/puzzles/collection_importer.go internal/puzzles/collection_importer_test.go internal/puzzles/lichess_importer.go internal/importjob/service.go internal/importjob/service_test.go
git commit -m "feat: add shared puzzle import pipeline"
```

---

### Task 3: Move Lichess behind the adapter without changing behavior

**Files:**
- Modify: `internal/puzzles/lichess_importer.go`
- Modify: `internal/puzzles/lichess_importer_test.go`
- Modify: `internal/puzzles/full_import_test.go`
- Modify: `internal/puzzles/lichess_generation_test.go`

**Interfaces:**
- Consumes: `PuzzleAdapter`, `PuzzleDecoder`, `CollectionImporter.ImportFormat`, and Task 1 solution helpers.
- Produces: `NewLichessAdapter(rules chessrules.Rules) PuzzleAdapter`; keeps `LichessImporter.Import(...)` source-compatible.

- [ ] **Step 1: Add failing adapter inspection tests**

Add tests proving zstd magic plus the exact required CSV columns matches regardless of filename, plain CSV does not match, a zstd file with an unrelated header returns an unsupported-content error, and inspection always returns source ID `lichess` with origin `fixed`.

- [ ] **Step 2: Add a failing compatibility test for shared-runner semantics**

Extend `TestLichessImporterNormalizesSetupMove` to assert the emitted progress phases, raw compressed checksum, and zero-valid protection while preserving its existing core/occurrence assertions.

- [ ] **Step 3: Run the focused Lichess tests and verify RED**

Run: `go test ./internal/puzzles -run 'TestLichess' -count=1`

Expected: FAIL because `NewLichessAdapter` and content inspection do not exist.

- [ ] **Step 4: Extract the zstd/CSV decoder**

The adapter probe opens the file, verifies the four-byte zstd magic, creates a low-memory single-concurrency decoder, reads one CSV header, and calls `lichessColumnIndexes`. `NewDecoder` owns the zstd reader and CSV reader. Its `Next` preserves the existing ordinal convention, recoverable row rejection, required columns, rating/popularity/play-count parsing, attribution, themes, URL, opening tags, and first-move prelude semantics.

Replace direct core construction with:

```go
displayedFEN, err := rules.ApplyUCI(sourceFEN, moves[0])
solver, err := solverFromFEN(displayedFEN)
core, err := finalizeCore(rules, displayedFEN, solver, linearSolution(moves[1:]))
```

The decoder still validates every remaining move through `finalizeCore`.

- [ ] **Step 5: Make `LichessImporter` a compatibility wrapper**

Construct a `CollectionImporter` from its existing fields and the one Lichess adapter, then call `ImportFormat(ctx, FormatLichess, sourceID, path, progress)`. This keeps all existing tests and full-import entry points source-compatible while removing duplicated staging/checksum code.

- [ ] **Step 6: Run Lichess and full puzzle package tests and verify GREEN**

Run: `go test ./internal/puzzles -run 'TestLichess|TestOrderedGeneration|TestGeneration' -count=1`

Expected: PASS.

- [ ] **Step 7: Commit the Lichess adapter migration**

```bash
git add internal/puzzles/lichess_importer.go internal/puzzles/lichess_importer_test.go internal/puzzles/full_import_test.go internal/puzzles/lichess_generation_test.go
git commit -m "refactor: route Lichess through shared importer"
```

---

### Task 4: Tactical PGN importer — first new working format

**Files:**
- Create: `internal/puzzles/tactical_pgn.go`
- Create: `internal/puzzles/tactical_pgn_test.go`
- Modify: `internal/puzzles/collection_importer_test.go`

**Interfaces:**
- Consumes: `chess.NewScanner`, `chess.TokenizeGame`, `chess.NewParser`, `Game.MoveHistory`, `chess.UCINotation`, the adapter contract, and Task 1 helpers.
- Produces: `NewTacticalPGNAdapter(rules chessrules.Rules) PuzzleAdapter`.

- [ ] **Step 1: Write failing white-solver and black-solver tests**

Use the real PGN parser with these fixture shapes:

```pgn
[Event "Direct solver turn"]
[SourceId "club-tactics"]
[PuzzleId "white-1"]
[SetUp "1"]
[FEN "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1"]
[White "solver"]
[Black "?"]

1. e4 Kf7 *
```

```pgn
[Event "Opponent prelude"]
[SetUp "1"][FEN "4k3/8/8/4p3/8/8/4P3/4K3 b - - 0 1"]
[White "solver"][Black "?"]

1... e4 2. Kf2 *
```

Assert source ID/origin, external ID fallback, solver, source/displayed FEN, prelude presence or absence, normalized UCI solution, solution plies, source kind `tactical-pgn`, and nil rating.

- [ ] **Step 2: Write failing record-rejection and fatal-identity tests**

Add table cases for missing FEN, invalid `SetUp`, both/neither solver tags, illegal movetext, prelude consuming the only move, and a later explicit `SourceId` that differs from the first game. Parse errors with a recoverable next PGN game must reject one record and continue; conflicting IDs must return fatal error and prevent activation.

- [ ] **Step 3: Write failing annotation behavior tests**

Use a game with comments, NAGs, and a RAV. Assert only `Game.MoveHistory()` mainline moves enter the solution and adjacent `][` tag pairs are accepted.

- [ ] **Step 4: Run the PGN tests and verify RED**

Run: `go test ./internal/puzzles -run 'TestTacticalPGN' -count=1`

Expected: build failure because the PGN adapter does not exist.

- [ ] **Step 5: Implement PGN probing and streaming**

Probe the first scanned game, reject records over 64 KiB, tokenize/parse it, require `FEN`, and resolve trimmed first-game `SourceId` or path fallback. The decoder calls `ScanGame` to retain record framing, classifies scanner/I/O failures as fatal, and converts tokenize/parse/semantic failures into `Rejection{Ordinal: gameNumber, Reason: ...}`.

For a parsed game:

1. Normalize FEN and solver tags.
2. Encode mainline history moves with `chess.UCINotation{}.Encode(history.PrePosition, history.Move)`.
3. Consume one prelude only when initial active color differs from solver.
4. Require at least one remaining move.
5. Finalize a linear solution core from the displayed FEN.
6. Store `SourceFEN` for PGN; store `PreludeUCI` only when consumed.
7. Use trimmed `PuzzleId`, otherwise `strconv.FormatInt(ordinal, 10)`.
8. Reject more than 128 parsed tag pairs; retain only bounded known provenance tags in metadata.

- [ ] **Step 6: Run PGN plus catalogue integration tests and verify GREEN**

Run: `go test ./internal/puzzles -run 'TestTacticalPGN|TestCollectionImporter' -count=1`

Expected: PASS. At this checkpoint tactical PGN is fully importable through the backend runner before any JSON/Lucas/linear code exists.

- [ ] **Step 7: Commit the tactical PGN milestone**

```bash
git add internal/puzzles/tactical_pgn.go internal/puzzles/tactical_pgn_test.go internal/puzzles/collection_importer_test.go
git commit -m "feat: import tactical PGN puzzles"
```

---

### Task 5: Versioned canonical JSON importer

**Files:**
- Create: `internal/puzzles/canonical_json.go`
- Create: `internal/puzzles/canonical_json_test.go`
- Modify: `internal/puzzles/collection_importer_test.go`

**Interfaces:**
- Consumes: `encoding/json`, the adapter contract, `finalizeCore`, and normalized source metadata from inspection.
- Produces: `NewCanonicalJSONAdapter(rules chessrules.Rules) PuzzleAdapter` and the exact `chess-trainer-puzzles/v1` parser.

- [ ] **Step 1: Write a failing streaming happy-path test**

Use an object where `puzzles` appears before `source` to prove inspection is field-order independent. Include one linear puzzle and one branched puzzle. Assert embedded source ID, inherited attribution/URL, explicit per-puzzle override, external ID fallback, normalized themes, nil versus non-nil rating, and maximum solution depth.

```json
{
  "schema": "chess-trainer-puzzles/v1",
  "puzzles": [{
    "id": "json-1",
    "displayedFen": "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1",
    "solver": "white",
    "solution": [{"uci":"e2e4","children":[
      {"uci":"e8e7","children":[]},
      {"uci":"e8f7","children":[]}
    ]}],
    "rating": 1450,
    "themes": ["fork", "fork"],
    "metadata": {"chapter": 3}
  }],
  "source": {
    "id": "club-json",
    "name": "Club JSON",
    "url": "https://example.test/club",
    "attribution": "Club coach"
  }
}
```

- [ ] **Step 2: Write failing strictness and presentation tests**

Add separate cases for unsupported/missing schema, duplicate top-level keys, unknown top-level/source/puzzle/move fields, invalid solver, rating 99/4001/fraction, non-object metadata, puzzle larger than 2 MiB, metadata larger than 64 KiB, and illegal tree branches.

Add one valid presentation case with `sourceFen` plus `preludeUci` and assert applying the prelude exactly matches normalized `displayedFen`. Add rejection cases when only one field is present or the resulting FEN differs.

- [ ] **Step 3: Write a failing syntax-versus-semantic recovery test**

One semantic-invalid item between two valid array items must produce two puzzles and one rejection. Broken JSON syntax in the second item must return a fatal decoder error and leave the previous active generation untouched in the runner test.

- [ ] **Step 4: Run canonical JSON tests and verify RED**

Run: `go test ./internal/puzzles -run 'TestCanonicalJSON' -count=1`

Expected: build failure because the adapter is absent.

- [ ] **Step 5: Implement the strict token-stream inspector**

Define narrow structs with `json.RawMessage` for puzzle and metadata fields. Walk the top-level object with `Decoder.Token`, track seen keys, and stream over `puzzles` without retaining the array. During inspection, validate the schema and source object and skip each puzzle as one bounded raw message. Reject trailing JSON tokens.

Use a helper that wraps a decoder with `DisallowUnknownFields` for source, puzzle, and move structs. Explicitly reject duplicate structural keys because `encoding/json` otherwise keeps the last value.

- [ ] **Step 6: Implement the authoritative puzzle decoder**

The decoder revalidates top-level structure but uses the source defaults supplied by the authoritative inspection even when `source` follows `puzzles`. For each raw puzzle:

1. Reject raw bytes over 2 MiB.
2. Strictly decode fields and reject metadata bytes over 64 KiB.
3. Parse solver only from `white` or `black`.
4. Validate rating range 100–4000 and non-negative popularity/play count.
5. Normalize themes with `domain.NormalizeThemes`.
6. Validate optional presentation fields together with `Rules.ApplyUCI`.
7. Call `finalizeCore` for FEN, solver, legal tree, plies, and fingerprint.
8. Populate source kind `canonical-json`, source defaults, override fields, metadata, and ordinal fallback ID.

Semantic errors return `DecodedRecord{Rejection: ...}`; JSON framing errors return fatal errors.

When the authoritative pass reaches `schema` or `source`, compare them with the inspection descriptor. A changed schema or source ID is fatal; do not silently import under the previously inspected identity.

- [ ] **Step 7: Run JSON and shared runner tests and verify GREEN**

Run: `go test ./internal/puzzles -run 'TestCanonicalJSON|TestCollectionImporter' -count=1`

Expected: PASS.

- [ ] **Step 8: Commit canonical JSON import**

```bash
git add internal/puzzles/canonical_json.go internal/puzzles/canonical_json_test.go internal/puzzles/collection_importer_test.go
git commit -m "feat: import canonical puzzle JSON"
```

---

### Task 6: Make all explicit ratings guide adaptive scheduling

**Files:**
- Modify: `internal/puzzles/catalog_models.go:63-82`
- Modify: `internal/puzzles/catalog_models_test.go`
- Modify: `internal/puzzles/catalog_reader.go:561-620`
- Modify: `internal/puzzles/catalog_reader_test.go`
- Modify: `internal/profile/service_test.go`

**Interfaces:**
- Consumes: `SourceSummary.MinimumRating/MaximumRating` and existing rating membership queries.
- Produces: learner bounds spanning every active rated source kind; no SQL schema change.

- [ ] **Step 1: Write failing model and SQLite tests**

```go
func TestLearnerRatingBoundsIncludeEveryRatedSource(t *testing.T) {
    jsonMin, jsonMax := 900, 2100
    lichessMin, lichessMax := 1100, 1800
    got := LearnerRatingBoundsFromSourceSummaries([]SourceSummary{
        {Kind: "canonical-json", MinimumRating: &jsonMin, MaximumRating: &jsonMax},
        {Kind: "lichess", MinimumRating: &lichessMin, MaximumRating: &lichessMax},
        {Kind: "tactical-pgn"},
    })
    if got != (RatingBounds{Minimum: 900, Maximum: 2100}) { t.Fatalf("bounds = %+v", got) }
}
```

The SQLite test must activate one rated canonical generation, one rated Lichess generation, and one unrated PGN generation, then assert `LearnerRatingBounds` spans the two rated sources and `RatedCandidates` can return the canonical occurrence.

- [ ] **Step 2: Run rating tests and verify RED**

Run: `go test ./internal/puzzles ./internal/profile -run 'TestLearnerRatingBounds|TestProfile.*RatingBounds' -count=1`

Expected: FAIL because model and SQL filter to Lichess.

- [ ] **Step 3: Remove source-kind filtering without changing defaults**

In the model helper, skip only summaries with nil min/max. In `LearnerRatingBounds`, remove `source.kind = 'lichess'`, scan the source kind from SQL or call `ActiveSourceSummaries` within one read transaction, and retain `DefaultLearnerRatingBounds()` when no active rated occurrence exists. Update the comment to describe all explicitly rated sources.

- [ ] **Step 4: Run puzzles, profile, and training tests and verify GREEN**

Run: `go test ./internal/puzzles ./internal/profile ./internal/training -count=1`

Expected: PASS.

- [ ] **Step 5: Commit rating generalization**

```bash
git add internal/puzzles/catalog_models.go internal/puzzles/catalog_models_test.go internal/puzzles/catalog_reader.go internal/puzzles/catalog_reader_test.go internal/profile/service_test.go
git commit -m "feat: guide sessions from all rated puzzle sources"
```

---

### Task 7: Lucas `.fns` converter/importer

**Files:**
- Create: `internal/puzzles/lucas_fns.go`
- Create: `internal/puzzles/lucas_fns_test.go`
- Modify: `internal/puzzles/collection_importer_test.go`

**Interfaces:**
- Consumes: `chess.NewScanner`, parsed move-tree `Game.GetRootMove().Children()`, `Move.Children()`, UCI encoding, and the adapter contract.
- Produces: `NewLucasFNSAdapter(rules chessrules.Rules) PuzzleAdapter`.

- [ ] **Step 1: Write failing mainline and variation conversion tests**

Use:

```text
4k3/8/8/8/8/8/4P3/4K3 w - - 0 1|Difficulty **|1. e4 Kf7 (1... Kd7) 2. Kf2 *
```

Assert solver white, no source prelude, source kind `lucas-fns`, line-number external ID, root UCI `e2e4`, opponent children `e8f7` and `e8d7`, correct continuation below the mainline child, and nil rating. The metadata must contain the plain description plus detected `sourceDifficulty`, not trusted HTML.

- [ ] **Step 2: Write failing line framing, theme, and rejection tests**

Cover blank/comment lines, fewer than two `|`, invalid FEN, invalid SAN, empty movetext, a line over 1 MiB, and continuation after a rejected line. Import a file named `Pin.fns` and assert theme `pin`; assert generic `Tactics.fns` supplies no filename theme.

- [ ] **Step 3: Run Lucas tests and verify RED**

Run: `go test ./internal/puzzles -run 'TestLucasFNS' -count=1`

Expected: build failure because the adapter is absent.

- [ ] **Step 4: Implement line scanning and synthetic PGN parsing**

Use `bufio.Scanner.Buffer(make([]byte, 64*1024), 1<<20)`. Split each non-comment line with `strings.SplitN(line, "|", 3)`. Wrap movetext for the chess parser:

```go
rawPGN := fmt.Sprintf("[SetUp \"1\"]\n[FEN %q]\n\n%s", fen, movetext)
```

Parse one game, reject a second. Convert every root child recursively to `domain.MoveNode` using lowercase `chess.UCINotation{}.Encode(nil, move)`, enforcing the shared depth/node limits while converting. Pass the resulting tree through `finalizeCore` as an independent legality/canonicalization check.

Normalize the filename stem by splitting camel case and punctuation to lowercase hyphenated words. Exclude exactly `training`, `tactics`, and `puzzles`. Detect `Difficulty` followed by stars or a number with a compiled case-insensitive regexp and store the captured value as `sourceDifficulty`.

- [ ] **Step 5: Run Lucas and shared adapter tests and verify GREEN**

Run: `go test ./internal/puzzles -run 'TestLucasFNS|TestCollectionImporter' -count=1`

Expected: PASS.

- [ ] **Step 6: Commit Lucas support**

```bash
git add internal/puzzles/lucas_fns.go internal/puzzles/lucas_fns_test.go internal/puzzles/collection_importer_test.go
git commit -m "feat: import Lucas FNS puzzles"
```

---

### Task 8: Linear FEN/UCI converter/importer

**Files:**
- Create: `internal/puzzles/linear_fen_uci.go`
- Create: `internal/puzzles/linear_fen_uci_test.go`
- Modify: `internal/puzzles/collection_importer_test.go`

**Interfaces:**
- Consumes: the adapter contract, `linearSolution`, and `finalizeCore`.
- Produces: `NewLinearFENAdapter(rules chessrules.Rules) PuzzleAdapter`.

- [ ] **Step 1: Write failing grammar and normalization tests**

```go
const linearFixture = `# Larion-style sample
4k3/8/8/8/8/8/4P3/4K3 w - - 0 1 e2e4 e8f7 1375
4k3/4P3/8/8/8/8/8/4K3 w - - 0 1 e7e8q
`
```

Assert two puzzles, line-number IDs `2` and `3`, white solver, linear solution depths 2 and 1, promotion normalization, first record metadata `sourceDifficulty: 1375`, nil ratings, and path source identity.

- [ ] **Step 2: Write failing rejection and framing tests**

Cover zero UCI moves, invalid six-field FEN, illegal UCI, invalid promotion, integer before the final token, inline `#` text, a line over 1 MiB, blank/comment lines, and continuation after malformed lines. Add a misleading `.pgn` filename containing valid linear content and assert content detection selects linear.

- [ ] **Step 3: Run linear tests and verify RED**

Run: `go test ./internal/puzzles -run 'TestLinearFEN' -count=1`

Expected: build failure because the adapter is absent.

- [ ] **Step 4: Implement the exact line grammar**

Scan with the 1 MiB buffer. For each non-comment line, require at least seven fields, join fields 0–5 as FEN, inspect only the final token with `strconv.Atoi`, and require at least one token to remain as UCI. Do not recognize inline comments. Build a linear tree, finalize the core, store `sourceDifficulty` only when the final integer exists, and leave rating nil.

The probe must validate the first meaningful line with the same parser without requiring the file extension.

- [ ] **Step 5: Run all adapter tests and verify GREEN**

Run: `go test ./internal/puzzles -run 'Test(Lichess|TacticalPGN|CanonicalJSON|LucasFNS|LinearFEN|CollectionImporter)' -count=1`

Expected: PASS.

- [ ] **Step 6: Commit linear support**

```bash
git add internal/puzzles/linear_fen_uci.go internal/puzzles/linear_fen_uci_test.go internal/puzzles/collection_importer_test.go
git commit -m "feat: import linear FEN UCI puzzles"
```

---

### Task 9: Compose all formats and expose generic Wails operations

**Files:**
- Modify: `internal/app/services.go:24-31,130-139`
- Modify: `internal/app/services_test.go`
- Modify: `normal_controller.go:66-99`
- Modify: `app_test.go:70-145`
- Regenerate: `frontend/wailsjs/go/main/NormalController.js`
- Regenerate: `frontend/wailsjs/go/main/NormalController.d.ts`
- Regenerate: `frontend/wailsjs/go/models.ts`

**Interfaces:**
- Consumes: all five adapters, `CollectionImporter.Inspect`, `FormatImporter`, `importjob.Service.Start`, and Wails runtime dialogs.
- Produces:
  - `NormalController.InspectPuzzleImport(path string) (puzzles.ImportInspection, error)`
  - `NormalController.StartPuzzleImport(path string) (string, error)`
  - compatibility `NormalController.StartLichessImport(path string) (string, error)` delegating to the generic method.

- [ ] **Step 1: Write failing service composition tests**

Open real application services and inspect small temporary fixtures for each format. Assert the detected format has a configured import-job route. Keep the Lichess fixture zstd-compressed and use minimal valid new-format fixtures from adapter tests.

- [ ] **Step 2: Update controller tests before production code**

Change the dirty file-filter test without discarding its macOS-safe `*.zst` correction. Assert the dialog title is `Choose a puzzle collection` and filters contain separate safe patterns for `*.zst`, `*.pgn`, `*.json`, `*.fns`, and `*.txt`.

Add tests that `InspectPuzzleImport` delegates and that `StartPuzzleImport(path)` starts a request whose kind/source/path match authoritative inspection. Assert empty paths fail and `StartLichessImport` delegates to the same path flow.

- [ ] **Step 3: Run application/controller tests and verify RED**

Run: `go test ./internal/app . -run 'Test.*(Import|OpenCreates)' -count=1`

Expected: compile/test failures for missing generic signatures and composition.

- [ ] **Step 4: Compose the default importer**

Change `Services.Importer` to `*puzzles.CollectionImporter`. Construct it with catalog reader/writer, rules, catalogue directory, disk-space callback, and adapters in deterministic probe order: zstd, JSON, PGN, Lucas, linear. Register `FormatImporter` under the matching five `importjob.Kind` values.

- [ ] **Step 5: Implement controller inspection and start-by-path**

Both methods run under `runNormalOperation`. `StartPuzzleImport` calls authoritative inspection, converts `inspection.Format` to `importjob.Kind`, and starts:

```go
importjob.ImportRequest{
    Kind: importjob.Kind(inspection.Format),
    SourceID: inspection.SourceID,
    Path: inspection.Path,
}
```

Keep `StartLichessImport(path)` as `return c.StartPuzzleImport(path)`.

- [ ] **Step 6: Regenerate Wails bindings**

Run:

```bash
/Users/admin/go/bin/wails generate module -tags bindings
perl -pi -e 's/[ \t]+$//' frontend/wailsjs/go/models.ts
perl -0pi -e 's/\s+\z/\n/' frontend/wailsjs/go/models.ts
chmod 0644 frontend/wailsjs/runtime/package.json frontend/wailsjs/runtime/runtime.d.ts frontend/wailsjs/runtime/runtime.js
```

Expected: generated `NormalController` accepts a path string for `StartPuzzleImport` and exposes `InspectPuzzleImport`; models include `ImportInspection` with string-valued format/origin properties and expanded `Progress`.

- [ ] **Step 7: Run Go tests and verify GREEN**

Run: `go test ./internal/app . ./internal/importjob ./internal/puzzles -count=1`

Expected: PASS.

- [ ] **Step 8: Commit backend composition and bindings**

```bash
git add internal/app/services.go internal/app/services_test.go normal_controller.go app_test.go frontend/wailsjs/go/main/NormalController.js frontend/wailsjs/go/main/NormalController.d.ts frontend/wailsjs/go/models.ts
git commit -m "feat: expose generic puzzle collection imports"
```

---

### Task 10: Source-first import UI and frontend contracts

**Files:**
- Modify: `frontend/src/lib/api-contract.ts:128-147,411-440`
- Modify: `frontend/src/lib/api.ts:62-145`
- Modify: `frontend/src/lib/api.test.ts`
- Modify: `frontend/src/test-fakes.ts:48-104`
- Modify: `frontend/src/lib/import-session.ts`
- Modify: `frontend/src/lib/import-session.test.ts`
- Modify: `frontend/src/components/import/ImportPanel.svelte`
- Modify: `frontend/src/components/import/ImportPanel.test.ts`

**Interfaces:**
- Consumes: generated `Normal.InspectPuzzleImport`, `Normal.StartPuzzleImport`, existing import events/results.
- Produces:

```ts
export type ImportInspection = {
  path: string
  filename: string
  format: 'lichess' | 'tactical-pgn' | 'canonical-json' | 'lucas-fns' | 'linear-fen-uci'
  sourceId: string
  sourceIdOrigin: 'fixed' | 'embedded' | 'path'
  replacesExisting: boolean
}

export type ImportProgress = {
  jobId: string
  phase: 'detecting' | 'parsing' | 'sealing' | 'activating'
  rowsRead: number
  bytesRead: number
  totalBytes: number
}
```

`ImportReport` gains decoded `examples: Array<{ordinal:number; reason:string}>`. `ImportSessionState` gains `inspection: ImportInspection | null` and starts only when inspection exists.

- [ ] **Step 1: Write failing API decoder and production-wiring tests**

Test strict enum decoding for formats/origins/phases, malformed inspection rejection, report examples, and that `productionNormalAPI.inspectPuzzleImport/startPuzzleImport` call the generated functions. Remove the frontend dependency on `startLichessImport` while leaving the generated compatibility method untouched.

- [ ] **Step 2: Write failing import-session tests**

Assert `selectFile()` calls choose then inspect, stores only authoritative inspection, clears stale result/inspection on a new selection, treats chooser cancellation as no error, surfaces inspection errors, and calls `startPuzzleImport(inspection.path)` exactly once. Preserve existing stale-progress and terminal-event ordering tests with phase/total-byte fields.

- [ ] **Step 3: Write failing component tests for source-first confirmation**

Render an embedded-ID inspection and assert visible text order/content includes:

- `club-tactics` as the selected card's strong primary text.
- `Tactical PGN` format label.
- Filename and full path.
- `This import will replace the active club-tactics collection` when applicable.

Render a path-origin inspection and assert `Fallback source ID (file path)` is visible. Assert the import button stays disabled until inspection succeeds. Add phase text assertions (`Reading puzzles`, `Finalizing collection`, `Activating collection`) and bounded rejection example rendering.

- [ ] **Step 4: Run frontend unit tests and verify RED**

Run: `npm --prefix frontend test -- --run --single-thread src/lib/api.test.ts src/lib/import-session.test.ts src/components/import/ImportPanel.test.ts`

Expected: FAIL for missing contracts and UI.

- [ ] **Step 5: Implement strict frontend contracts and API wiring**

Add `decodeImportInspection`, expand progress/report decoding, export the new types through `api.ts`, wire generated methods, and update preview/fake APIs with a deterministic Lichess inspection. Keep runtime decoding strict rather than casting generated values.

- [ ] **Step 6: Implement inspection-aware session state**

During `selectFile`, set `busy`, clear prior errors, call chooser, return unchanged on empty path, then call inspection and atomically store `{path: inspection.path, inspection, result: null}`. During `start`, require `current.inspection`, call generic start with its normalized path, and preserve the existing immediate result refresh.

Phase merging uses the same ordered rank as Go; counters remain independent maxima and a terminal result cannot be replaced by a running snapshot.

- [ ] **Step 7: Implement the format-neutral panel**

Use heading `Import puzzle collection`, chooser label `Choose puzzle collection`, and copy explaining direct local streaming for all formats. Display source ID first, then a human-readable format map, source-ID origin, filename/path, and replacement warning. Progress uses ordinary `bytes`, not `compressed bytes`. Render at most the backend-provided rejection examples as plain text in a list.

- [ ] **Step 8: Run frontend unit, type, and build checks and verify GREEN**

Run:

```bash
npm --prefix frontend test -- --run --single-thread src/lib/api.test.ts src/lib/import-session.test.ts src/components/import/ImportPanel.test.ts
npm --prefix frontend run check
npm --prefix frontend run build
```

Expected: all commands PASS with no Svelte accessibility warnings.

- [ ] **Step 9: Commit frontend integration**

```bash
git add frontend/src/lib/api-contract.ts frontend/src/lib/api.ts frontend/src/lib/api.test.ts frontend/src/test-fakes.ts frontend/src/lib/import-session.ts frontend/src/lib/import-session.test.ts frontend/src/components/import/ImportPanel.svelte frontend/src/components/import/ImportPanel.test.ts
git commit -m "feat: show source-first puzzle import UI"
```

---

### Task 11: Cross-format integration, browser flow, and operator documentation

**Files:**
- Create: `internal/puzzles/multi_format_import_test.go`
- Modify: `frontend/tests/test-backend.ts`
- Modify: `frontend/tests/trainer.spec.ts`
- Create: `docs/operations/puzzle-import-formats.md`
- Modify: `docs/operations/local-build.md:101-115,141-159,235-238`
- Modify: `README.md:3-7,36-39`

**Interfaces:**
- Consumes: complete backend importer, generated/frontend API, and existing Playwright test backend.
- Produces: durable examples and end-to-end acceptance evidence for every supported format.

- [ ] **Step 1: Write failing catalogue integration tests**

For each new format, import a temporary file through `CollectionImporter` into a real SQLite catalogue and assert the active source summary, source kind, puzzle occurrence, and solution tree. Then:

- Import the same core from PGN and JSON and assert one canonical fingerprint with two active occurrences.
- Reimport the JSON source ID with a replacement file and assert only JSON head changes.
- Attempt zero-valid, cancelled, and late-conflicting PGN imports and assert old heads remain queryable.
- Query guided candidates and learner bounds to prove canonical rating eligibility and Lucas/linear difficulty ineligibility.

- [ ] **Step 2: Run integration tests and verify RED if any cross-layer gap remains**

Run: `go test ./internal/puzzles -run 'TestMultiFormatImport' -count=1`

Expected: PASS only after every adapter, runner, and rating integration is complete; fix production code through an additional red/green cycle for any exposed gap.

- [ ] **Step 3: Update the browser test backend before UI code**

Replace `StartLichessImport` usage with `InspectPuzzleImport` and `StartPuzzleImport`. Return a deterministic embedded source ID `club-tactics`, `tactical-pgn` format, normalized path, and expanded progress. Record the path passed to generic start so `selectedImportPath` remains authoritative.

Update `trainer.spec.ts` to assert source ID appears before import, format label is visible, progress survives navigation, and cancellation still reaches the terminal state.

- [ ] **Step 4: Run the focused browser test**

Run: `npm --prefix frontend run test:e2e -- trainer.spec.ts --project=chromium`

Expected: PASS.

- [ ] **Step 5: Document exact supported grammars and examples**

`docs/operations/puzzle-import-formats.md` must include:

- Source-ID resolution table.
- Tactical PGN solver/prelude convention and a complete valid game.
- Full `chess-trainer-puzzles/v1` object with branched solution and optional rating.
- Lucas three-field syntax and variation example.
- Linear six-field FEN/UCI syntax with optional difficulty.
- Rating versus difficulty behavior.
- Atomic replacement, cancellation, zero-valid, and rejection-report behavior.
- The exact resource limits from Global Constraints.

Update README to say the app imports multiple local puzzle formats. Update the local build guide's chooser copy and retain the gated full-Lichess acceptance procedure.

- [ ] **Step 6: Run the complete verification matrix**

Use the `superpowers:verification-before-completion` skill before claiming success. Run fresh:

```bash
git diff --check
go test ./... -count=1
go test -race ./... -count=1
npm --prefix frontend test -- --run --single-thread
npm --prefix frontend run check
npm --prefix frontend run build
npm --prefix frontend run test:e2e -- trainer.spec.ts --project=chromium
```

Expected: every command exits 0. If the full race suite exceeds the local checkpoint budget, rerun the affected packages with `-race` and report the exact timeout rather than claiming the full suite passed.

- [ ] **Step 7: Review generated and unrelated changes**

Run:

```bash
git status --short
git diff --stat
git diff -- app_test.go normal_controller.go
```

Confirm the user's existing macOS-safe `*.zst` change is incorporated into the broader filter rather than discarded, no board-results files changed, and no third-party puzzle data was committed.

- [ ] **Step 8: Commit integration tests and documentation**

```bash
git add internal/puzzles/multi_format_import_test.go frontend/tests/test-backend.ts frontend/tests/trainer.spec.ts docs/operations/puzzle-import-formats.md docs/operations/local-build.md README.md
git commit -m "test: verify multi-format puzzle imports"
```
