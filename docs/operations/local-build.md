# Chess Trainer local build and acceptance

Chess Trainer is a local macOS Wails application. It does not need a hosted web
server and does not expose an HTTP listener in a production build. User data is
stored under `~/Library/Application Support/Chess Trainer/`.

## Prerequisites

- macOS with Xcode Command Line Tools
- Go 1.25 or newer
- Node.js 20 or newer
- Wails v2 (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`)

From the repository root, install the frontend packages once:

```bash
npm --prefix frontend ci
```

## Automated verification

Run the unit, race, browser, and production-frontend checks:

```bash
go test -race ./...
npm --prefix frontend test -- --run
npm --prefix frontend run check
npm --prefix frontend run test:e2e
npm --prefix frontend run build
```

Expected results:

- Go reports `ok` for every package. The opt-in full-import test explicitly
  skips when `CHESS_TRAINER_LICHESS_PATH` is unset.
- Vitest reports all component and library tests passing.
- `svelte-check` reports zero errors and zero warnings.
- Playwright passes the trainer flow in Chromium and WebKit.
- Vite creates `frontend/dist` without TypeScript or bundling errors.

## Full Lichess catalogue acceptance

Point the opt-in test at the downloaded compressed database. No decompressed
CSV is written; the importer streams the `.zst` directly into a temporary
SQLite catalogue. A current multi-million-row database can take tens of
minutes because every solution line is legality-checked. Keep at least 13 GiB
free for the current 288 MiB download; the app checks its conservative staging
reserve before starting.

```bash
CHESS_TRAINER_LICHESS_PATH="/absolute/path/Lichess Puzzle Database.csv.zst" \
  go test ./internal/puzzles \
  -run TestFullLichessImport \
  -count=1 \
  -timeout=45m \
  -v
```

Expected result: more than one million puzzles accepted, less than 0.1% of
rows rejected, bounded heap and RSS growth, a rated-candidate query below
250 ms, and `PASS`. The source directory must not gain a decompressed `.csv`.

## Build the macOS application

```bash
"$(go env GOPATH)/bin/wails" build -clean
codesign --verify --deep --strict --verbose=2 "build/bin/Chess Trainer.app"
open "build/bin/Chess Trainer.app"
```

Expected result: Wails creates `build/bin/Chess Trainer.app`, `codesign`
reports that it is valid on disk and satisfies its designated requirement,
and Finder launches it by double-click without a terminal or browser window.

## Manual acceptance checklist

Use a backup from Parent view before testing restore against valuable data.

1. Complete first-run setup and import a small `.csv.zst` fixture. Cancel a
   second import and confirm the previous catalogue remains available.
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

At startup the app runs SQLite integrity checks before opening repositories.
If a database is corrupt, the normal interface is replaced by a recovery
screen with only **Restore backup**, **Open data folder**, and **Quit**.

- Restore a validated `.zip` created by Chess Trainer, then relaunch.
- `user.sqlite` is always in a backup; `library.sqlite` is included only when
  selected during backup.
- `puzzles.sqlite` is replaceable and intentionally excluded. If it is the
  corrupt file, quit, move it out of the data folder, relaunch, and reimport
  the compressed Lichess source.
- Restore keeps timestamped pre-restore copies in the `backups` directory.
