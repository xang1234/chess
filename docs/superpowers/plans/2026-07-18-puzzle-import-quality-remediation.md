# Puzzle Import Quality Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Centralize puzzle-format policy, make import provenance and frontend state authoritative, reduce the result contract, restore the 1,000-line architecture limit, and remove dead query and metadata work without changing import behavior.

**Architecture:** Registered `PuzzleAdapter` descriptors are the Go-side source of truth for format identity, labels, extensions, and native chooser filters. `CollectionImporter` owns final occurrence provenance immediately before catalogue insertion. The frontend stores import lifecycle data in a discriminated union and keeps the confirmed inspection independently of the reduced import-job result wire shape.

**Tech Stack:** Go 1.24, SQLite, Wails v2, TypeScript 4.6, Svelte 3, Vitest, Playwright.

## Global Constraints

- Follow red-green-refactor for every behavior change; observe each stated expected failure before editing production code.
- Use `apply_patch` for source edits. Use `gofmt` and generated-code commands only for mechanical rewrites.
- Preserve streaming, cancellation, checksumming, staging, sealing, activation, source-ID resolution, rating semantics, progress monotonicity, and stale-event rejection.
- Keep supported formats and probe order unchanged: Lichess zstd, canonical JSON, tactical PGN, Lucas FNS, linear FEN/UCI.
- Treat descriptor labels, chooser descriptions, extensions, and `ReplacesExisting` as display/configuration data, not revalidation identity.
- Do not hand-edit Wails bindings. Regenerate them after the Go wire contracts are final.
- Do not add implementation-shape tests. Keep every owned `.go`, `.ts`, and `.svelte` file at or below 1,000 physical lines.
- Preserve unrelated user changes and do not stage them with task commits.

---

### Task 1: Centralize adapter descriptors and occurrence provenance

**Files:**

- Modify: `internal/puzzles/collection_importer.go`
- Modify: `internal/puzzles/collection_importer_test.go`
- Modify: `internal/puzzles/lichess_importer.go`
- Modify: `internal/puzzles/lichess_importer_test.go`
- Modify: `internal/puzzles/tactical_pgn.go`
- Modify: `internal/puzzles/tactical_pgn_test.go`
- Modify: `internal/puzzles/canonical_json.go`
- Modify: `internal/puzzles/canonical_json_test.go`
- Modify: `internal/puzzles/lucas_fns.go`
- Modify: `internal/puzzles/lucas_fns_test.go`
- Modify: `internal/puzzles/linear_fen_uci.go`
- Modify: `internal/puzzles/linear_fen_uci_test.go`
- Modify: `internal/puzzles/multi_format_import_test.go`
- Modify: `normal_controller.go`
- Modify: `app_test.go`

**Interfaces:**

```go
type ImportFormatDescriptor struct {
    Format                ImportFormat
    Label                 string
    CanonicalExtension    string
    FileFilterDescription string
}

type PuzzleAdapter interface {
    Descriptor() ImportFormatDescriptor
    Inspect(context.Context, string) (ImportInspection, bool, error)
    NewDecoder(io.Reader, ImportInspection) (PuzzleDecoder, error)
}

func (i *CollectionImporter) FormatDescriptors() ([]ImportFormatDescriptor, error)
```

Extend `ImportInspection` with `FormatLabel string \`json:"formatLabel"\``. Remove `PuzzleAdapter.Format()` and the format/extension switch. `CollectionImporter` becomes the only layer that writes final occurrence `SourceID` and `SourceKind`.

- [ ] **Step 1: Write failing descriptor tests**

Change `fakePuzzleAdapter` to carry and return an `ImportFormatDescriptor`. Add `TestCollectionImporterFormatDescriptorsPreserveRegistrationOrder` and assert exact descriptor copies. Add table cases for empty format, label, canonical extension, and filter description and require field-specific configuration errors.

Update `TestCollectionImporterInspectUsesExtensionOnlyToNarrowContentMatches`: both fake adapters match the same bytes, only the tactical descriptor declares `.pgn`, and the resulting inspection must have `FormatTacticalPGN` plus `FormatLabel == "Tactical PGN"`.

Update the chooser test to construct a `CollectionImporter` with these exact descriptor values and assert both display name and pattern in registration order:

| Format | Label | Filter description | Extension | Wails display name |
|---|---|---|---|---|
| `lichess` | `Lichess` | `Zstandard archive` | `.zst` | `Zstandard archive (*.zst)` |
| `canonical-json` | `Canonical JSON` | `JSON collection` | `.json` | `JSON collection (*.json)` |
| `tactical-pgn` | `Tactical PGN` | `PGN collection` | `.pgn` | `PGN collection (*.pgn)` |
| `lucas-fns` | `Lucas FNS` | `Lucas collection` | `.fns` | `Lucas collection (*.fns)` |
| `linear-fen-uci` | `Linear FEN/UCI` | `FEN/UCI collection` | `.txt` | `FEN/UCI collection (*.txt)` |

- [ ] **Step 2: Write failing authoritative-provenance tests**

In `TestCollectionImporterImportSealsChecksumAndActivatesInOrder`, return a fake decoded occurrence with `SourceID: "decoder-source"` and `SourceKind: "decoder-kind"`. Assert the captured puzzle instead contains `inspection.SourceID` and `string(inspection.Format)` while retaining its external ID.

Change direct adapter tests to expect decoded occurrences to leave `SourceID` and `SourceKind` empty. Retain all assertions for external ID, source FEN, prelude, rating, URL, attribution, metadata, themes, and ordinal. Keep `multi_format_import_test.go` assertions that catalogued occurrences contain resolved source IDs and formats; this is the end-to-end runner proof.

- [ ] **Step 3: Run focused tests and verify RED**

```bash
go test ./internal/puzzles . -run 'Test(CollectionImporterFormatDescriptors|CollectionImporterInspectUsesExtension|CollectionImporterImportSeals|ChoosePuzzleImportFile|.*AdapterConverts|MultiFormatImportPersists)' -count=1
```

Expected: compile failures for the missing descriptor API and `FormatLabel`, then provenance failures if production is partially migrated.

- [ ] **Step 4: Implement validated descriptors**

Add the descriptor type, `FormatLabel`, and `Descriptor()`. Implement one literal per adapter using the table. Add a shared validator that trims/checks all four fields, requires an extension beginning with `.` and containing no wildcard, and names the adapter index and invalid field in errors.

Implement `FormatDescriptors()` as an ordered defensive copy. A nil adapter is a configuration error. Replace adapter-format uses in `Supports`, `inspectRegistry`, ambiguity errors, `adapterForFormat`, and `normalizeAdapterInspection` with `adapter.Descriptor().Format`. Complete inspections with `Format` and `FormatLabel`; do not compare `FormatLabel` or `ReplacesExisting` in `compareImportInspection`.

Replace the extension switch with:

```go
func descriptorMatchesExtension(descriptor ImportFormatDescriptor, path string) bool {
    return strings.EqualFold(filepath.Ext(path), descriptor.CanonicalExtension)
}
```

- [ ] **Step 5: Derive chooser filters from descriptors**

In `ChoosePuzzleImportFile`, call `c.services.Importer.FormatDescriptors()`, propagate errors, and map each descriptor to:

```go
runtime.FileFilter{
    DisplayName: fmt.Sprintf("%s (*%s)",
        descriptor.FileFilterDescription,
        descriptor.CanonicalExtension,
    ),
    Pattern: "*" + descriptor.CanonicalExtension,
}
```

Retain the title `Choose a puzzle collection`. Update controller fixtures to provide `services.Importer`; cancellation must still return `"", nil`.

- [ ] **Step 6: Stamp final provenance in the shared runner**

Immediately before `ordered.Add`, copy and overwrite only authoritative fields:

```go
puzzle := *record.Puzzle
puzzle.Occurrence.SourceID = sourceID
puzzle.Occurrence.SourceKind = string(format)
if err := ordered.Add(ctx, puzzle); err != nil {
    return abandon(err)
}
```

Remove `sourceID` fields/constructor assignments from Lichess, Lucas, and linear decoders. Remove Lucas/linear decoder source-ID validation. Tactical PGN and canonical JSON may retain inspection identity for embedded-source validation and source-level defaults, but must stop assigning final occurrence `SourceID`/`SourceKind`.

- [ ] **Step 7: Verify GREEN and commit**

```bash
gofmt -w internal/puzzles/collection_importer.go internal/puzzles/collection_importer_test.go internal/puzzles/lichess_importer.go internal/puzzles/lichess_importer_test.go internal/puzzles/tactical_pgn.go internal/puzzles/tactical_pgn_test.go internal/puzzles/canonical_json.go internal/puzzles/canonical_json_test.go internal/puzzles/lucas_fns.go internal/puzzles/lucas_fns_test.go internal/puzzles/linear_fen_uci.go internal/puzzles/linear_fen_uci_test.go internal/puzzles/multi_format_import_test.go normal_controller.go app_test.go
go test ./internal/puzzles . -count=1
git add internal/puzzles/collection_importer.go internal/puzzles/collection_importer_test.go internal/puzzles/lichess_importer.go internal/puzzles/lichess_importer_test.go internal/puzzles/tactical_pgn.go internal/puzzles/tactical_pgn_test.go internal/puzzles/canonical_json.go internal/puzzles/canonical_json_test.go internal/puzzles/lucas_fns.go internal/puzzles/lucas_fns_test.go internal/puzzles/linear_fen_uci.go internal/puzzles/linear_fen_uci_test.go internal/puzzles/multi_format_import_test.go normal_controller.go app_test.go
git commit -m "refactor: centralize puzzle import format policy"
```

Expected: PASS. Ambiguity/error wording continues to use stable format IDs, not display labels.

---

### Task 2: Remove inspection from import-job results and regenerate bindings

**Files:**

- Modify: `internal/importjob/service.go`
- Modify: `internal/importjob/service_test.go` (split in Task 4)
- Modify: `internal/app/services_test.go`
- Regenerate: `frontend/wailsjs/go/models.ts`
- Regenerate: `frontend/wailsjs/go/main/NormalController.d.ts`
- Regenerate: `frontend/wailsjs/go/main/NormalController.js`

**Interface:** `importjob.Result` contains only `JobID`, `Status`, `Progress`, `Report`, and optional `Error`. Confirmed inspection remains the direct argument to `Service.run` and `Importer.Import`.

- [ ] **Step 1: Write and run the failing wire-shape test**

Add `TestResultJSONOmitsInspection`. Marshal a populated `Result`, unmarshal into `map[string]json.RawMessage`, fail if `inspection` exists, and assert required keys `jobId`, `status`, `progress`, and `report` exist.

Run: `go test ./internal/importjob -run TestResultJSONOmitsInspection -count=1`

Expected: FAIL because `Result` currently serializes `inspection`.

- [ ] **Step 2: Remove the field without weakening execution identity**

Delete `Result.Inspection` and the `Inspection: inspection` initializer in `Service.Start`. Update complete-result assertions/fixtures. Do not remove inspection from `Start`, `run`, or the importer call.

- [ ] **Step 3: Regenerate and normalize Wails bindings**

```bash
/Users/admin/go/bin/wails generate module -tags bindings
perl -pi -e 's/[ \t]+$//' frontend/wailsjs/go/models.ts
perl -0pi -e 's/\s+\z/\n/' frontend/wailsjs/go/models.ts
chmod 0644 frontend/wailsjs/runtime/package.json frontend/wailsjs/runtime/runtime.d.ts frontend/wailsjs/runtime/runtime.js
rg -n 'inspection' frontend/wailsjs/go/models.ts internal/importjob/service.go
```

Expected: generated `importjob.Result` has no inspection property. Remaining matches belong only to input/model types and execution parameters.

- [ ] **Step 4: Verify GREEN and commit**

```bash
gofmt -w internal/importjob/service.go internal/importjob/service_test.go
go test ./internal/importjob ./internal/app . -count=1
git add internal/importjob/service.go internal/importjob/service_test.go internal/app/services_test.go frontend/wailsjs/go/models.ts frontend/wailsjs/go/main/NormalController.d.ts frontend/wailsjs/go/main/NormalController.js frontend/wailsjs/runtime/package.json frontend/wailsjs/runtime/runtime.d.ts frontend/wailsjs/runtime/runtime.js
git diff --cached --name-only
git commit -m "refactor: reduce puzzle import result contract"
```

Expected: PASS. Unstage generated runtime files if their content/mode did not actually change.

---

### Task 3: Make frontend import contracts and lifecycle states explicit

**Files:**

- Modify: `frontend/src/lib/api-contract.ts`
- Modify: `frontend/src/lib/api.test.ts`
- Modify: `frontend/src/lib/import-session.ts`
- Modify: `frontend/src/lib/import-session.test.ts`
- Modify: `frontend/src/components/import/ImportPanel.svelte`
- Modify: `frontend/src/components/import/ImportPanel.test.ts`
- Modify: `frontend/src/test-fakes.ts`
- Modify: `frontend/tests/test-backend.ts`

**Interfaces:** Define one `importFormats` tuple and derive `ImportFormat`; add required `ImportInspection.formatLabel`; replace the nullable state bag with a discriminated union; make guards accept/narrow the full state.

- [ ] **Step 1: Write failing contract and state tests**

Add `formatLabel` to valid inspection fixtures, assert it decodes, and add a missing-label case expecting `import inspection.formatLabel must be a string`. Keep unknown-format rejection.

Replace nullable-bag assertions with phase-owned shapes. Cover: exact idle; selecting with prior state and exact cancellation restoration; inspecting with path; inspection failure to idle; required inspection in ready/starting; job/progress in running; terminal result in finished; stale IDs ignored; running polls cannot overwrite terminal state; guards accept states rather than strings. Retain pending-chooser and pending-inspection concurrency tests, narrowing before reading state-owned fields.

- [ ] **Step 2: Run tests and verify RED**

```bash
npm --prefix frontend test -- --run --single-thread src/lib/api.test.ts src/lib/import-session.test.ts src/components/import/ImportPanel.test.ts
```

Expected: compile/test failures because the label is not decoded, guards accept phases, and the session remains a nullable bag.

- [ ] **Step 3: Implement the closed format contract**

```ts
const importFormats = [
  'lichess',
  'tactical-pgn',
  'canonical-json',
  'lucas-fns',
  'linear-fen-uci'
] as const

export type ImportFormat = typeof importFormats[number]
```

Pass the tuple to `enumeration`, add `formatLabel: string`, and decode it with `string(raw.formatLabel, 'import inspection.formatLabel')`. Delete the duplicate literal allowlist.

- [ ] **Step 4: Implement the discriminated union and guards**

```ts
type IdleImportState = { phase: 'idle'; error: string }
type ReadyImportState = { phase: 'ready'; inspection: ImportInspection; error: string }
type TerminalImportResult = Omit<ImportResult, 'status'> & {
  status: 'succeeded' | 'failed' | 'cancelled'
}
type FinishedImportState = {
  phase: 'finished'
  inspection: ImportInspection
  jobId: string
  progress: ImportProgress
  result: TerminalImportResult
  error: string
}
export type SelectableImportState = IdleImportState | ReadyImportState | FinishedImportState
export type ImportSessionState =
  | SelectableImportState
  | { phase: 'selecting'; previous: SelectableImportState; error: string }
  | { phase: 'inspecting'; path: string; error: string }
  | { phase: 'starting'; inspection: ImportInspection; error: string }
  | { phase: 'running'; inspection: ImportInspection; jobId: string; progress: ImportProgress; error: string }
```

Implement `canSelectImportFile(state)` as a `SelectableImportState` type guard, `canStartImport(state)` as a ready/finished guard, and `selectedImportInspection(state)`. For selecting, the accessor recursively reads `previous`; ready/starting/running/finished return their inspection; other states return null.

- [ ] **Step 5: Separate chooser and inspection error boundaries**

Implement `selectFile` with two `try/catch` blocks and no `let inspecting`: capture the selectable `previous`; enter selecting; restore it on cancellation; restore it with error on chooser failure; enter inspecting with a chosen path; enter ready on success; enter idle with error on inspection failure.

Implement start/cancel/refresh/listeners only against narrowed states. `applyResult` may transition only a matching running job. Merge progress monotonically before storing running or terminal snapshots; ignore stale IDs.

- [ ] **Step 6: Narrow the Svelte template**

Delete `formatLabels`. Derive display inspection through `selectedImportInspection($session)` and render `inspection.formatLabel`. Pass `$session` to guards. Read progress only in the running branch and result only in the finished branch. Keep the import button disabled unless `canStartImport($session)`.

- [ ] **Step 7: Verify GREEN and commit**

```bash
npm --prefix frontend test -- --run --single-thread src/lib/api.test.ts src/lib/import-session.test.ts src/components/import/ImportPanel.test.ts
npm --prefix frontend run check
git add frontend/src/lib/api-contract.ts frontend/src/lib/api.test.ts frontend/src/lib/import-session.ts frontend/src/lib/import-session.test.ts frontend/src/components/import/ImportPanel.svelte frontend/src/components/import/ImportPanel.test.ts frontend/src/test-fakes.ts frontend/tests/test-backend.ts
git commit -m "refactor: model puzzle import states explicitly"
```

---

### Task 4: Enforce repository-wide source size and split import-job tests

**Files:**

- Create: `source_size_test.go`
- Delete: `internal/puzzles/architecture_test.go`
- Delete after redistribution: `internal/importjob/service_test.go`
- Create: `internal/importjob/service_test_support_test.go`
- Create: `internal/importjob/service_lifecycle_test.go`
- Create: `internal/importjob/service_cleanup_test.go`

**Interfaces:** Add root test `TestOwnedSourceFilesStayUnderLimit`; preserve every import-job behavior test; remove the reflection-based bool ban.

- [ ] **Step 1: Add the failing root architecture test**

Walk from `os.Getwd()` with `filepath.WalkDir`. Skip directory basenames `.git`, `.worktrees`, `node_modules`, `vendor`, `build`, `dist`, `coverage`, and `wailsjs`. Inspect regular `.go`, `.ts`, and `.svelte` files, count physical lines with `bufio.Scanner`, and report every file above 1,000 lines using slash-normalized relative paths.

Give the scanner at least a 1 MiB buffer so a long source line produces an explicit scanner error rather than a false count. Include `_test.go` files and the guard itself.

- [ ] **Step 2: Run the guard and verify RED**

Run: `go test . -run TestOwnedSourceFilesStayUnderLimit -count=1`

Expected: FAIL naming `internal/importjob/service_test.go` above the 1,000-line limit.

- [ ] **Step 3: Split shared support and lifecycle cases**

Move all shared structs, doubles, channel payloads, and their methods from the top of `service_test.go` into `service_test_support_test.go` under `package importjob`.

Move these tests into `service_lifecycle_test.go`:

- `TestStartPassesCanonicalFormatToImporter`
- `TestStartPassesConfirmedInspectionToImporter`
- `TestStartSeedsDetectingProgressBeforePublishingJob`
- `TestStartValidatesInspection`
- `TestStartRejectsSecondActivePuzzleImport`
- `TestResultKeepsMonotonicProgress`
- `TestImportProgressPhaseAndTotalsRemainMonotonic`
- `TestCompletedResultRemainsQueryableAfterLaterJob`
- `TestJobReachesExactlyOneTerminalState`
- `TestSuccessfulImportRemainsSucceededWhenContextCancelledAfterActivation`
- `TestTerminalJobAllowsNextImport`
- `TestResultJSONOmitsInspection`

- [ ] **Step 4: Split cleanup and ordering cases**

Move these tests into `service_cleanup_test.go`:

- `TestConcurrentProgressEmitsMonotonicSnapshots`
- `TestTerminalEventPrecedesLaterJobProgress`
- `TestStaleCleanupWaitsForLaterTerminalEvent`
- `TestCleanupStartsAfterTerminalAndNeverOverlapsImport`
- `TestNewImportPreemptsFurtherCleanupBatches`
- `TestCloseCancelsAndWaitsForImporterAndCleanup`
- `TestConcurrentStartAndCloseWaitsForEveryRegisteredJob`

Each file owns only imports it uses. Delete `service_test.go` after all declarations move. Compare `rg '^func Test'` output before and after so every original test plus the new JSON test exists exactly once.

- [ ] **Step 5: Remove the implementation-shape guard**

Delete `internal/puzzles/architecture_test.go` in full. Its size rule is superseded by the root guard; do not recreate its reflection-based parser-field rule.

- [ ] **Step 6: Verify GREEN and commit**

```bash
gofmt -w source_size_test.go internal/importjob/service_test_support_test.go internal/importjob/service_lifecycle_test.go internal/importjob/service_cleanup_test.go
go test . ./internal/importjob ./internal/puzzles -count=1
find . -type f \( -name '*.go' -o -name '*.ts' -o -name '*.svelte' \) -not -path './.git/*' -not -path './frontend/node_modules/*' -not -path './frontend/wailsjs/*' -not -path './build/*' -not -path './vendor/*' -print0 | xargs -0 wc -l | sort -nr | head -20
git add source_size_test.go internal/importjob/service_test_support_test.go internal/importjob/service_lifecycle_test.go internal/importjob/service_cleanup_test.go internal/importjob/service_test.go internal/puzzles/architecture_test.go
git commit -m "test: enforce repository source size limits"
```

Expected: tests PASS and every displayed owned file is at or below 1,000 lines.

---

### Task 5: Remove dead rating-query and linear-metadata work

**Files:**

- Modify: `internal/puzzles/catalog_reader.go`
- Modify: `internal/puzzles/catalog_reader_test.go`
- Modify: `internal/puzzles/linear_fen_uci.go`

**Interfaces:** Keep `LearnerRatingBounds` and linear accepted syntax unchanged. Remove rating-bounds dependence on `sources.kind`/`sources`; remove the impossible linear 64 KiB metadata path.

- [ ] **Step 1: Add and run a failing query-dependency test**

Add `TestLearnerRatingBoundsDoesNotRequireSourcesTable`. Seed one active generation with ratings 900 and 2200, set `PRAGMA foreign_keys=OFF`, drop `sources`, call `LearnerRatingBounds`, and assert `{Minimum: 900, Maximum: 2200}`.

Run: `go test ./internal/puzzles -run TestLearnerRatingBoundsDoesNotRequireSourcesTable -count=1`

Expected: FAIL with a missing `sources` table because the current query joins it for an unused kind.

- [ ] **Step 2: Simplify the rating-bounds query**

Remove `source.kind` from `SELECT`, remove `JOIN sources AS source`, and scan only `minimum` and `maximum`. Remove the local `kind` variable and `Kind` assignment in `SourceSummary`. Preserve both per-generation ordered `occurrence_ratings` subqueries and the sealed-generation filter exactly.

- [ ] **Step 3: Establish the linear behavior baseline**

```bash
go test ./internal/puzzles -run 'TestLinearFEN(AdapterConvertsNormalizedMoveLinesAndDifficulty|DecoderRejectsMalformedRecordsAndRecovers)' -count=1
```

Expected: PASS before deletion. These tests characterize the only metadata shapes: empty or one machine-sized integer.

- [ ] **Step 4: Delete the impossible metadata-size path**

Remove `encoding/json`, `maxLinearFENMetadataBytes`, `json.Marshal(metadata)`, serialization error wrapping, byte-length comparison, and metadata-limit error. Return `core, metadata, nil` after `finalizeCore`. Do not change Lucas FNS's separate description-size guard; Lucas metadata contains unbounded text.

- [ ] **Step 5: Verify GREEN and commit**

```bash
gofmt -w internal/puzzles/catalog_reader.go internal/puzzles/catalog_reader_test.go internal/puzzles/linear_fen_uci.go
go test ./internal/puzzles -count=1
rg -n 'maxLinearFENMetadataBytes|serialize linear FEN/UCI metadata|JOIN sources AS source' internal/puzzles/linear_fen_uci.go internal/puzzles/catalog_reader.go
git add internal/puzzles/catalog_reader.go internal/puzzles/catalog_reader_test.go internal/puzzles/linear_fen_uci.go
git commit -m "refactor: remove dead puzzle import plumbing"
```

Expected: tests PASS and `rg` has no matches in the named files. Leave unrelated catalogue `sources` joins untouched.

---

### Task 6: Verify the complete remediation

**Files:** Review every file changed by Tasks 1-5. Update `docs/operations/puzzle-import-formats.md` or `docs/operations/local-build.md` only if contract/chooser prose became stale.

**Interfaces:** None. This task proves all six approved findings are addressed.

- [ ] **Step 1: Run static contract searches**

```bash
rg -n 'func \([^)]*Adapter\) Format\(\)|adapter\.Format\(\)|formatLabels|inspection: ImportInspection \| null|maxLinearFENMetadataBytes' --glob '!docs/superpowers/**' internal frontend normal_controller.go app_test.go
rg -n 'SourceID:.*d\.|SourceKind:' internal/puzzles/lichess_importer.go internal/puzzles/tactical_pgn.go internal/puzzles/canonical_json.go internal/puzzles/lucas_fns.go internal/puzzles/linear_fen_uci.go
rg -n '^[[:space:]]*Inspection[[:space:]]+puzzles\.ImportInspection|json:"inspection"' internal/importjob frontend/wailsjs/go/models.ts
```

Expected: no obsolete adapter method, frontend label map, nullable inspection bag, linear metadata limit, decoder-owned final provenance, or result inspection. Manually inspect matches: tactical/JSON identity-validation fields are allowed; final occurrence assignments are not.

- [ ] **Step 2: Run fresh Go verification**

Use `superpowers:verification-before-completion`, then run:

```bash
git diff --check
go test ./... -count=1
go test -race ./... -count=1
go vet ./...
go test -tags=performance ./internal/puzzles -run '^$' -count=1
```

Expected: every command exits 0. The performance command compiles tagged tests without running gated benchmarks/full-dataset imports.

- [ ] **Step 3: Run fresh frontend verification**

```bash
npm --prefix frontend test -- --run --single-thread
npm --prefix frontend run check
npm --prefix frontend run build
npm --prefix frontend run verify:licenses
npm --prefix frontend run test:e2e -- trainer.spec.ts --project=chromium
```

Expected: every command exits 0.

- [ ] **Step 4: Review scope and architecture evidence**

```bash
git status --short
git diff --stat 3057981..HEAD
git diff --name-status 3057981..HEAD
git log --oneline --decorate -8
```

Confirm all six findings map to changes, only intended source/tests/generated bindings/optional prose changed, no data/build/browser/dependency artifacts are tracked, every owned source file satisfies the guard, and the worktree is clean.

- [ ] **Step 5: Commit only necessary documentation correction**

If operation prose required correction:

```bash
git add docs/operations/puzzle-import-formats.md docs/operations/local-build.md
git commit -m "docs: align puzzle import contracts"
```

Otherwise, do not create an empty final commit.
