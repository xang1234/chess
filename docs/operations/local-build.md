# Chess Trainer local build and acceptance

Chess Trainer is a local macOS Wails application. It does not need a hosted web
server and does not expose an HTTP listener in a production build. User data is
stored under `~/Library/Application Support/Chess Trainer/`.

## Prerequisites and version policy

- macOS with Xcode Command Line Tools
- Go 1.26.4 for verified local parity and every distributable build
- Node.js 20 or newer
- Wails v2.12.0 (the commands below run the pinned CLI through `go run`)

The `go 1.25.0` directive in `go.mod` is the application's language and module
compatibility level; it is not the distributable-build toolchain version. The
release verifier requires the exact Go 1.26.4 toolchain, checks its installed
`LICENSE` and `PATENTS` against the reviewed copies under
`third_party/legal/go1.26.4/`, and rejects automatic toolchain switching.

From the repository root, populate dependencies from their lockfiles and verify
the downloaded Go modules:

```bash
npm --prefix frontend ci
go mod download all
go mod verify
npm --prefix frontend run build
npm --prefix frontend run verify:licenses
```

## Automated verification

Run the ordinary backend suite, the full race suite, and the frontend checks:

```bash
go test ./... -count=1
go test -race ./... -count=1
go mod verify
node frontend/scripts/generate-sounds.mjs --check
node scripts/generate-third-party-notices.mjs --check
npm --prefix frontend run build
npm --prefix frontend run verify:licenses
node --test scripts/verify-legal-assets.test.mjs
node --test scripts/generate-third-party-notices.test.mjs
npm --prefix frontend test -- --run --single-thread
npm --prefix frontend run check
npm --prefix frontend run test:e2e
node --test scripts/verify-release.test.mjs
node --test scripts/build-corresponding-source.test.mjs
node --test scripts/build-release.test.mjs
bash -n scripts/build-release.sh
```

Expected results:

- Go reports `ok` for every package. Performance-tagged tests, including the
  real Lichess import, are excluded from ordinary and race runs.
- Vitest reports all component and library tests passing.
- `svelte-check` reports zero errors and zero warnings.
- Playwright passes the trainer flow in Chromium and WebKit.
- Vite creates `frontend/dist` without TypeScript or bundling errors.
- Legal verification proves the public documents, complete GPL text, exact
  Chessground/Svelte preferred-source archives, Go legal files, dependency
  closure, and generated notices still match the committed locks.
- Release-fixture tests prove every supported release subprocess sees fresh
  release-scoped Go and npm caches and that cleanup runs after success or
  failure.

## Puzzle catalogue storage and startup

`user.sqlite` is durable learner history. It contains settings, sessions,
attempts, review state, and rating history and is never recreated as part of
puzzle-catalogue recovery. `library.sqlite` is durable when the optional game
library is used. `puzzles.sqlite` is different: it is a disposable, versioned
cache that can be rebuilt by reimporting its source data.

Startup still treats replacement conservatively:

1. Acquire the data-root instance lock, then preflight every existing durable
   store and the puzzle catalogue before changing any of them. Current durable
   stores are checked in place without a copy; only stores with pending
   migrations are snapshotted to a temporary directory and rehearsed there.
2. After every store passes preflight, migrate `user.sqlite`.
3. Recognize a legacy puzzle store only when its migration set and complete
   table/column signatures exactly match a supported legacy format.
4. Backfill null attempt and queued-session provenance from matching legacy
   fingerprint/source rows, then close the legacy handle.
5. Only after that backfill succeeds, remove the recognized `puzzles.sqlite`
   and its exact WAL/SHM sidecars and create the current catalogue.

An empty, unversioned, modified, corrupt, or newer puzzle store is not assumed
to be disposable. The app preserves the database and sidecars and reports the
incompatibility so an operator can inspect or move them aside deliberately.
It never applies this recreation path to `user.sqlite` or `library.sqlite`.

Normal puzzle startup probes schema version and required tables; it does not
run `PRAGMA quick_check` or otherwise scan the full catalogue. Full integrity
checks remain explicit diagnostic/restore operations. Startup integrity checks
for the durable user and library stores are unchanged.

## Import, cleanup, and shutdown behavior

Only one puzzle import runs at a time. Import writes and bounded inactive-
generation cleanup share one writer gate, so they never overlap. A newly
reserved import prevents further cleanup batches after the current bounded
batch finishes. A completed or cancelled import remains queryable by job ID,
while the prior active generation stays readable until a sealed replacement is
activated.

Choose the compressed source with the native **Choose puzzle database** dialog;
the production app does not require an absolute path to be typed or a Terminal
to be opened. The current job and its event subscription belong to the root app,
so progress, cancellation, and the terminal report remain available after
navigating away from the import screen and returning during the same app run.

Application shutdown first rejects new jobs, cancels active import and cleanup
contexts, and waits for their registered goroutines. SQLite handles close only
after that wait, and the data-root instance lock is released last. Do not
force-remove catalogue files while the app is still shutting down.

Restore uses the same ordering but deliberately keeps the data-root instance
lock after jobs and database handles are quiesced. The lock remains held while
validated files are replaced or rolled back and is released only when the app
terminates. A failed restore therefore leaves normal services quiesced; retry
the restore or quit and relaunch rather than continuing training in that run.

## Performance-tagged catalogue gates

Performance assertions are intentionally separate from ordinary correctness
and race tests. Run the synthetic activation and candidate-query gates with:

```bash
go test -tags=performance ./internal/puzzles \
  -run 'Test(GenerationActivation|ActiveCandidateQuery)' \
  -count=1
```

The synthetic gates require activation to finish within five seconds and a
warmed representative candidate query within 250 ms.

### Full Lichess catalogue acceptance

Point the opt-in test at the downloaded compressed database. No decompressed
CSV is written; the importer streams the `.zst` through disposable
catalogue-local staging databases before sealing a generation. A current
multi-million-row database can take tens of minutes because every solution
line is legality-checked. Keep at least 13 GiB free for the current 288 MiB
download; the app checks its conservative staging reserve before starting.

```bash
: "${CHESS_TRAINER_LICHESS_PATH:?set CHESS_TRAINER_LICHESS_PATH to the downloaded .zst file}"
CHESS_TRAINER_LICHESS_PATH="$CHESS_TRAINER_LICHESS_PATH" \
  go test -tags=performance ./internal/puzzles \
  -run '^TestFullLichessImport$' -count=1 -timeout=65m
```

The test itself requires total import below one hour, activation below five
seconds, more than one million accepted occurrences, less than 0.1% rejected,
bounded heap/RSS growth, source-aware candidates, the prior head remaining
readable during import, and no decompressed `.csv` beside the source.

## Build the macOS application

For a local, unpublished test build, use the pinned Wails source and keep the
default `development` build identity:

```bash
GOWORK=off GOTOOLCHAIN=local \
  go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build -clean -trimpath
perl -pi -e 's/[ \t]+$//' frontend/wailsjs/go/models.ts
perl -0pi -e 's/\s+\z/\n/' frontend/wailsjs/go/models.ts
chmod 0644 frontend/wailsjs/runtime/package.json \
  frontend/wailsjs/runtime/runtime.d.ts \
  frontend/wailsjs/runtime/runtime.js
git diff --check
codesign --verify --deep --strict --verbose=2 "build/bin/Chess Trainer.app"
open "build/bin/Chess Trainer.app"
```

Expected result: the pinned Wails build creates
`build/bin/Chess Trainer.app`, generated bindings remain normalized,
`codesign` reports that the app is valid on disk and satisfies its designated
requirement, and Finder launches it by double-click without a terminal or
browser window.

The local build is not a supported public distributable: it reports source
identity `development` and does not prove that a matching tag is public. For a
binary you intend to share, follow `docs/operations/release.md` and use only
`scripts/build-release.sh <public-tag>`. That wrapper creates empty disposable
`GOMODCACHE`, `GOCACHE`, and npm cache directories beneath a temporary release
root, extracts the exact commit into an isolated build tree, fixes
`GOWORK=off`, `GOTOOLCHAIN=local`, and `GOENV=off`, and clears inherited Go,
Node, npm, and Git settings before installing from the locks and embedding the
exact public commit. Ignored checkout files, overlays, shared module
extractions, and user npm configuration are never release inputs.

## Manual acceptance checklist

Use a backup from Parent view before testing restore against valuable data.

1. Complete first-run setup and choose a small `.csv.zst` fixture with the
   native file dialog. During a second import, navigate home and back to Parent
   view, confirm progress is still present, then cancel it and confirm the
   previous catalogue remains available.
2. Start guided training, make one move, force-quit the app, relaunch it, and
   confirm **Continue today's training** restores the same board and progress.
3. Finish or pause the session. Confirm the home hub is interactive within
   three seconds after returning from import.
4. In Parent view, create a backup. Change profile data, restore that backup,
   relaunch, and confirm the earlier data returns.
5. While the production app is running, verify it has no listening TCP socket:

   ```bash
   pgrep -fl "Chess Trainer"
   lsof -nP -iTCP -sTCP:LISTEN | rg "Chess Trainer"
   ```

   The first command should show the application process. The second should
   print nothing and return status 1, meaning no matching listener exists.

## Recovery mode

At startup the app runs SQLite integrity checks on durable user/library stores
before opening repositories. Corruption, migration/open failures, and preserved
incompatible puzzle schemas replace the normal interface with a recovery screen
containing only **Restore backup**, **Open data folder**, and **Quit**. The
backend binds only those recovery capabilities—normal training/import methods
are not exposed—and the single-instance lock remains owned by that recovery
process. Puzzle startup uses the cheap conservative probe described above
instead of scanning the complete catalogue.

- Restore a validated `.zip` created by Chess Trainer, then relaunch.
- `user.sqlite` is always in a backup; `library.sqlite` is included only when
  selected during backup.
- `puzzles.sqlite` is replaceable and intentionally excluded. If its preserved
  file is corrupt or incompatible, quit, retain or move it for diagnosis,
  relaunch with the path absent, and reimport the compressed Lichess source.
- Restore keeps timestamped pre-restore copies in the `backups` directory.
