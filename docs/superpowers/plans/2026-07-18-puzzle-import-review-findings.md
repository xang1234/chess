# Puzzle Import Review Findings Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Preserve the confirmed puzzle-source identity, make text/PGN detection recoverable, require terminal progress, and remove obsolete Lichess import APIs.

**Architecture:** The UI sends the complete `ImportInspection` it displayed. The asynchronous runner resolves that inspection's adapter by format, re-inspects only that adapter, compares stable identity fields, and then streams the file through the shared generation lifecycle. Adapter inspection recognizes structural signatures; decoders alone decide whether individual records are semantically valid.

**Tech Stack:** Go 1.24, Wails v2, TypeScript, Svelte 5, Vitest.

## Global Constraints

- Preserve bounded-memory streaming, cancellation, checksumming, staging, sealing, and atomic activation.
- Do not trust a stale inspection; compare it against the selected adapter immediately before staging.
- Do not query active-source summaries during execution-time revalidation.
- Treat malformed linear and recoverably framed PGN records as decoder-level rejections.
- Keep `ImportResult.progress` required across Go and TypeScript.

---

### Task 1: Carry and revalidate the confirmed inspection

**Files:**
- Modify: `internal/puzzles/collection_importer.go`
- Modify: `internal/puzzles/collection_importer_test.go`
- Modify: `internal/importjob/service.go`
- Modify: `internal/importjob/service_test.go`
- Modify: `normal_controller.go`
- Modify: `app_test.go`
- Modify: `frontend/src/lib/api.ts`
- Modify: `frontend/src/lib/import-session.ts`
- Modify: `frontend/src/lib/import-session.test.ts`

**Interfaces:**
- Produces: `CollectionImporter.Import(context.Context, ImportInspection, ProgressSink) (ImportReport, error)`.
- Produces: `importjob.Importer.Import(context.Context, puzzles.ImportInspection, puzzles.ProgressSink) (puzzles.ImportReport, error)`.
- Produces: `Service.Start(context.Context, puzzles.ImportInspection) (string, error)` and `Result.Inspection puzzles.ImportInspection`.
- Produces: `NormalController.StartPuzzleImport(puzzles.ImportInspection) (string, error)`.
- Produces: `NormalAPI.startPuzzleImport(inspection: ImportInspection): Promise<string>`.

- [ ] **Step 1: Write failing backend tests for the new identity boundary**

Add a collection-importer test that passes an expected inspection and configures a matching adapter plus an unrelated adapter whose `Inspect` fails the test if called. Assert the matching adapter is inspected exactly once and the generation activates. Change the controller binding test so its fake importer receives the complete `ImportInspection` supplied to `StartPuzzleImport` without a controller-side registry scan.

- [ ] **Step 2: Run the backend tests and verify RED**

Run: `go test ./internal/puzzles ./internal/importjob . -run 'Test(CollectionImporterImportRevalidatesOnlyConfirmedAdapter|StartPuzzleImportPassesConfirmedInspection)' -count=1`

Expected: FAIL to compile because the new `Import` and `StartPuzzleImport(ImportInspection)` contracts do not exist.

- [ ] **Step 3: Write the failing frontend session test**

Change the `NormalAPI` fake to record the argument to `startPuzzleImport`, call `session.start()`, and assert it receives the same inspection object returned by `inspectPuzzleImport`.

- [ ] **Step 4: Run the frontend test and verify RED**

Run: `npm test -- --run --threads=false src/lib/import-session.test.ts`

Expected: FAIL because `start()` currently passes `inspection.path`.

- [ ] **Step 5: Implement selected-adapter revalidation**

Replace `ImportFormat(ctx, format, sourceID, path, progress)` with `Import(ctx, expected, progress)`. Resolve one configured adapter by `expected.Format`, normalize `expected.Path`, call only that adapter's `Inspect`, normalize the adapter result without replacement-status enrichment, and compare `Path`, `Format`, `SourceID`, `SourceIDOrigin`, `SourceName`, `URL`, and `Attribution`. Return a precise `puzzle import <field> changed after inspection` error before `BeginImport` on mismatch. Use the revalidated inspection for decoder construction.

- [ ] **Step 6: Carry the inspection through jobs, controller, and frontend**

Store `puzzles.ImportInspection` directly in the job result, validate its format/source/path in `Service.Start`, and pass it to the importer in `run`. Change the Wails controller and `NormalAPI` to accept the inspection object, and make the session pass its confirmed inspection.

- [ ] **Step 7: Run focused tests and verify GREEN**

Run: `go test ./internal/puzzles ./internal/importjob . -count=1`

Run: `npm test -- --run --threads=false src/lib/import-session.test.ts src/lib/api.test.ts`

Expected: PASS.

### Task 2: Separate structural probes from semantic decoding

**Files:**
- Modify: `internal/puzzles/linear_fen_uci.go`
- Modify: `internal/puzzles/linear_fen_uci_test.go`
- Modify: `internal/puzzles/tactical_pgn.go`
- Modify: `internal/puzzles/tactical_pgn_test.go`

**Interfaces:**
- Produces: internal `looksLikeLinearFENRecord(string) bool` structural probe.
- Produces: tactical PGN inspection that scans bounded game frames for a non-empty `FEN` tag without parsing movetext.

- [ ] **Step 1: Change the linear inspection regression test to the desired behavior**

Use a syntactically recognizable but illegal leading FEN/UCI record followed by the valid fixture. Assert `Inspect` matches `FormatLinearFENUCI`; decode the same file and assert the first record is rejected while the later record is accepted.

- [ ] **Step 2: Run the linear test and verify RED**

Run: `go test ./internal/puzzles -run 'TestLinearFENInspectionAllowsMalformedLeadingRecord' -count=1 -v`

Expected: FAIL because inspection stops after semantic validation of the first record.

- [ ] **Step 3: Implement the linear structural probe**

Recognize eight slash-separated placement ranks, `w`/`b`, castling and en-passant field shapes, numeric move counters, and at least one lower-case UCI-shaped move token after the first six fields. Continue scanning past unrelated lines until a structural match; retain scanner-limit errors as fatal framing errors.

- [ ] **Step 4: Add a failing tactical PGN inspection regression test**

Construct a first game with a non-empty `FEN` tag and malformed movetext followed by a valid tactical game. Assert inspection matches and resolves the first tactical game's source ID, then assert decoding rejects the first game and accepts the second.

- [ ] **Step 5: Run the tactical test and verify RED**

Run: `go test ./internal/puzzles -run 'TestTacticalPGNInspectionAllowsMalformedLeadingGame' -count=1 -v`

Expected: FAIL because inspection parses the first game's movetext.

- [ ] **Step 6: Implement tag-only tactical inspection**

Scan games until a bounded frame exposes a non-empty `FEN` tag. Use tokenizer-returned tags even when tokenization reports a recoverable semantic error. Resolve `SourceId` from that game or fall back to the path. Keep scanner/framing errors fatal and let the decoder reject malformed games.

- [ ] **Step 7: Run adapter tests and verify GREEN**

Run: `go test ./internal/puzzles -run 'Test(LinearFEN|TacticalPGN)' -count=1`

Expected: PASS.

### Task 3: Require progress in frontend result decoding

**Files:**
- Modify: `frontend/src/lib/api-contract.ts`
- Modify: `frontend/src/lib/api.test.ts`
- Modify: `frontend/src/lib/import-session.ts`
- Modify: frontend import component/session fixtures that construct `ImportResult`.

**Interfaces:**
- Produces: `ImportResult.progress: ImportProgressSnapshot` as a required property.

- [ ] **Step 1: Write the missing-progress decoder regression test**

Add `expect(() => decodeImportResult({...validResult, progress: undefined})).toThrow('import result.progress must be an object')`.

- [ ] **Step 2: Run the decoder test and verify RED**

Run: `npm test -- --run --threads=false src/lib/api.test.ts`

Expected: FAIL because missing progress is currently accepted.

- [ ] **Step 3: Tighten the contract and remove fallbacks**

Make `progress` required, call `decodeImportProgressSnapshot(raw.progress, ...)` unconditionally, update typed fixtures, and have terminal session state always merge the result snapshot.

- [ ] **Step 4: Run frontend unit tests and verify GREEN**

Run: `npm test -- --run --threads=false src/lib/api.test.ts src/lib/import-session.test.ts src/components/import/ImportPanel.test.ts`

Expected: PASS.

### Task 4: Remove compatibility-only import APIs

**Files:**
- Modify: `internal/puzzles/lichess_importer.go`
- Modify: `internal/puzzles/lichess_importer_test.go`
- Modify: `internal/puzzles/lichess_generation_test.go`
- Modify: `internal/puzzles/full_import_test.go`
- Modify: `normal_controller.go`
- Modify: `app_test.go`
- Regenerate: `frontend/wailsjs/go/main/NormalController.d.ts`
- Regenerate: `frontend/wailsjs/go/main/NormalController.js`
- Regenerate: `frontend/wailsjs/go/models.ts`

**Interfaces:**
- Removes: `puzzles.LichessImporter`.
- Removes: `NormalController.StartLichessImport` and its generated Wails binding.
- Keeps: `NewLichessAdapter` and the shared `CollectionImporter` lifecycle.

- [ ] **Step 1: Migrate Lichess tests to the shared importer**

Construct `CollectionImporter{Adapters: []PuzzleAdapter{NewLichessAdapter(rules)}}`, inspect the fixture, and call `Import(ctx, inspection, progress)` in lifecycle, generation, cancellation, allocation, and full-import tests.

- [ ] **Step 2: Run migrated tests before deletion**

Run: `go test ./internal/puzzles -run 'Test(Lichess|FullLichess)' -count=1`

Expected: PASS through the shared importer.

- [ ] **Step 3: Delete the wrappers and regenerate bindings**

Remove `LichessImporter`, its forwarding `Import` method, `StartLichessImport`, and the obsolete controller tests. Regenerate Wails models/bindings using the repository's Wails generator.

- [ ] **Step 4: Verify removal and the complete branch**

Run: `rg -n 'LichessImporter|StartLichessImport|ImportFormat' --glob '!docs/superpowers/**' .`

Expected: no matches.

Run: `gofmt -w` on changed Go files, then `go test ./... -count=1`, `go test -race ./... -count=1`, `go vet ./...`, `npm test -- --run --threads=false`, `npm run check`, `npm run build`, and the configured Playwright suite.

Expected: every command passes with no warnings attributable to these changes.
