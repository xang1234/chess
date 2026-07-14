# Chess Trainer Foundation and Lichess Puzzles Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver a double-clickable macOS Wails application that imports the local Lichess puzzle database, runs adaptive guided and free-practice sessions, preserves progress, and needs no host or internet connection.

**Architecture:** Wails v2 provides the macOS shell and generated Go-to-TypeScript bindings. A Go application core owns chess rules, streaming imports, scheduling, and three SQLite stores; Svelte components render a custom board and child/parent screens without embedding GPL chess UI code. This is plan 1 of 3 and produces a useful standalone Lichess trainer before PGN and analysis-boundary work begins.

**Tech Stack:** Go 1.26.x, Wails v2.12.0, Svelte/TypeScript/Vite from the Wails `svelte-ts` template, npm 11/Node 24, `github.com/corentings/chess/v2`, `modernc.org/sqlite`, `github.com/klauspost/compress/zstd`, Vitest, Testing Library, and Playwright.

## Global Constraints

- Target macOS first; the deliverable is a normal `Chess Trainer.app` launched without Terminal use.
- Normal operation is offline and opens no network listener.
- Store application data under `~/Library/Application Support/Chess Trainer/`.
- Support one learner profile; no account, cloud sync, telemetry, or analytics.
- Do not bundle, download, or implement a chess engine.
- Do not use GPL-only frontend UI dependencies unless the whole application is deliberately relicensed under the GPL.
- Stream the `.csv.zst` input; never materialise the complete decompressed CSV or hold it in memory.
- Save an active session after every learner move.
- Use test-first red/green/refactor cycles and commit after every task.

## Planned File Map

- `main.go` — Wails process entry point and dependency composition.
- `app.go` — thin Wails binding facade only.
- `internal/domain/puzzle.go` — canonical puzzle, solution tree, source, attempt, and review value types.
- `internal/domain/training.go` — session commands and view models.
- `internal/chessrules/rules.go` — legal move, FEN, SAN, and checkmate adapter.
- `internal/storage/paths.go` — macOS application-support paths.
- `internal/storage/space_darwin.go` — import disk-space preflight on macOS.
- `internal/storage/sqlite.go` — SQLite connection policy and migration runner.
- `internal/storage/migrations/{puzzles,user,library}/001.sql` — initial schemas.
- `internal/puzzles/fingerprint.go` — stable canonical puzzle identity.
- `internal/puzzles/catalog.go` — catalogue interfaces and queries.
- `internal/puzzles/sqlite_catalog.go` — atomic staged source import and catalogue persistence.
- `internal/puzzles/lichess_importer.go` — streamed Lichess CSV/Zstandard adapter.
- `internal/importjob/service.go` — cancellable background import jobs and progress events.
- `internal/training/{rating,review,scheduler,service}.go` — adaptive selection and persisted solving workflow.
- `internal/profile/service.go` — parent configuration and progress summaries.
- `internal/backup/service.go` — validated user-data backup and restore.
- `frontend/src/lib/api.ts` — typed wrapper over generated Wails bindings.
- `frontend/src/lib/navigation.ts` — small screen-state store.
- `frontend/src/lib/fen.ts` — pure FEN-to-board presentation parser.
- `frontend/src/components/chess/ChessBoard.svelte` — permissively licensed in-project board UI.
- `frontend/src/components/{home,import,puzzle,practice,parent}/*.svelte` — focused feature components.
- `frontend/src/styles/{tokens,app}.css` — visual system selected during brainstorming.
- `frontend/tests/*.spec.ts` — browser-level happy paths with a fake binding adapter.

---

### Task 1: Scaffold the pinned Wails application and test harness

**Files:**
- Create: `main.go`
- Create: `app.go`
- Create: `wails.json`
- Create: `go.mod`
- Create: `go.sum`
- Create: `build/`
- Create: `frontend/`
- Create: `internal/buildinfo/buildinfo.go`
- Test: `internal/buildinfo/buildinfo_test.go`
- Modify: `.gitignore`
- Modify: `frontend/vite.config.ts`
- Modify: `frontend/package.json`
- Test: `frontend/src/App.test.ts`

**Interfaces:**
- Produces: `buildinfo.Name == "Chess Trainer"` and the standard Wails v2 project/build commands used by every later task.
- Produces: frontend test command `npm test -- --run` and backend test command `go test ./...`.

- [ ] **Step 1: Install and verify the stable Wails CLI**

Run:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
wails doctor
```

Expected: `wails doctor` reports Wails `v2.12.0`, Go, npm, Xcode command-line tools, and WebKit as available.

- [ ] **Step 2: Scaffold into the existing repository without touching the design documents or database**

Run from the repository root in one shell:

```bash
SCAFFOLD_DIR="$(mktemp -d)"
wails init -n "Chess Trainer" -t svelte-ts -d "$SCAFFOLD_DIR"
rsync -a --exclude .gitignore "$SCAFFOLD_DIR/" ./
go mod edit -module chess-trainer
```

Expected: root contains `main.go`, `app.go`, `wails.json`, `go.mod`, `build/`, and `frontend/`; `docs/` and `Lichess Puzzle Database.csv.zst` remain unchanged.

- [ ] **Step 3: Write the failing backend identity test**

Create `internal/buildinfo/buildinfo_test.go`:

```go
package buildinfo

import "testing"

func TestProductName(t *testing.T) {
	if Name != "Chess Trainer" {
		t.Fatalf("Name = %q, want %q", Name, "Chess Trainer")
	}
}
```

- [ ] **Step 4: Run the backend test and verify red**

Run: `go test ./internal/buildinfo -run TestProductName -v`

Expected: FAIL because `Name` is undefined.

- [ ] **Step 5: Add the minimal backend identity**

Create `internal/buildinfo/buildinfo.go`:

```go
package buildinfo

const Name = "Chess Trainer"
```

Run: `go test ./internal/buildinfo -v`

Expected: PASS.

- [ ] **Step 6: Install the frontend test stack and define its command**

Run:

```bash
cd frontend
npm install --save-dev vitest jsdom @testing-library/svelte @testing-library/jest-dom @playwright/test
```

Add to `frontend/package.json` scripts:

```json
"test": "vitest",
"test:e2e": "playwright test"
```

Add this `test` member to `frontend/vite.config.ts`:

```ts
test: {
  environment: 'jsdom',
  setupFiles: ['./src/test-setup.ts']
}
```

Create `frontend/src/test-setup.ts`:

```ts
import '@testing-library/jest-dom/vitest'
```

- [ ] **Step 7: Write and run the frontend smoke test**

Replace `frontend/src/App.test.ts` with:

```ts
import { render, screen } from '@testing-library/svelte'
import App from './App.svelte'

test('renders the product name', () => {
  render(App)
  expect(screen.getByText('Chess Trainer')).toBeInTheDocument()
})
```

Run: `npm test -- --run`

Expected: FAIL until `frontend/src/App.svelte` contains an accessible `Chess Trainer` heading. Replace the generated heading with `<h1>Chess Trainer</h1>`, rerun, and expect PASS.

- [ ] **Step 8: Verify scaffold build and ignore generated outputs**

Append to `.gitignore`:

```gitignore
build/bin/
frontend/dist/
frontend/node_modules/
frontend/playwright-report/
frontend/test-results/
```

Run:

```bash
go test ./...
cd frontend
npm test -- --run
cd ..
wails build
```

Expected: all tests PASS and `build/bin/Chess Trainer.app` exists.

- [ ] **Step 9: Commit the independently buildable shell**

```bash
git add .gitignore main.go app.go wails.json go.mod go.sum build frontend internal/buildinfo
git commit -m "build: scaffold Wails chess trainer"
```

### Task 2: Define canonical puzzles and the chess-rules adapter

**Files:**
- Create: `internal/domain/puzzle.go`
- Create: `internal/chessrules/rules.go`
- Test: `internal/chessrules/rules_test.go`
- Create: `internal/puzzles/fingerprint.go`
- Test: `internal/puzzles/fingerprint_test.go`
- Modify: `go.mod`
- Modify: `go.sum`

**Interfaces:**
- Produces: `domain.Puzzle`, `domain.MoveNode`, `domain.Color`, `domain.SourceRef`.
- Produces: `chessrules.Rules.ApplyUCI(fen, uci)`, `SAN(fen, uci)`, and `IsCheckmateMove(fen, uci)`.
- Produces: `puzzles.Fingerprint(domain.Puzzle) (string, error)`.

- [ ] **Step 1: Add the chess-rules dependency**

Run: `go get github.com/corentings/chess/v2`

Expected: `go.mod` and `go.sum` record a v2 release.

- [ ] **Step 2: Define the canonical puzzle types**

Create `internal/domain/puzzle.go`:

```go
package domain

type Color string

const (
	White Color = "white"
	Black Color = "black"
)

type MoveNode struct {
	UCI      string     `json:"uci"`
	Children []MoveNode `json:"children,omitempty"`
}

type SourceRef struct {
	SourceID   string         `json:"sourceId"`
	ExternalID string         `json:"externalId,omitempty"`
	URL        string         `json:"url,omitempty"`
	Attribution string        `json:"attribution,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
}

type Puzzle struct {
	Fingerprint      string      `json:"fingerprint"`
	SourceFEN        string      `json:"sourceFen,omitempty"`
	PreludeUCI       string      `json:"preludeUci,omitempty"`
	DisplayedFEN     string      `json:"displayedFen"`
	Solver           Color       `json:"solver"`
	Solution         []MoveNode  `json:"solution"`
	Rating           *int        `json:"rating,omitempty"`
	Themes           []string    `json:"themes"`
	Popularity       *int        `json:"popularity,omitempty"`
	PlayCount        *int        `json:"playCount,omitempty"`
	Sources          []SourceRef `json:"sources"`
}
```

- [ ] **Step 3: Write failing rules tests for setup moves, SAN, and mate**

Create `internal/chessrules/rules_test.go`:

```go
package chessrules

import "testing"

func TestRules(t *testing.T) {
	r := Rules{}
	fen := "rnbqkbnr/pppp1ppp/8/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 1 2"

	next, err := r.ApplyUCI(fen, "f1b5")
	if err != nil || next == fen {
		t.Fatalf("ApplyUCI() next=%q err=%v", next, err)
	}
	san, err := r.SAN(fen, "f1b5")
	if err != nil || san != "Bb5" {
		t.Fatalf("SAN()=%q err=%v", san, err)
	}
	if !r.IsCheckmateMove("7k/5Q2/6K1/8/8/8/8/8 w - - 0 1", "f7f8") {
		t.Fatal("expected Qf8 to checkmate")
	}
}
```

Run: `go test ./internal/chessrules -v`

Expected: FAIL because `Rules` is undefined.

- [ ] **Step 4: Implement the rules adapter**

Create `internal/chessrules/rules.go`:

```go
package chessrules

import "github.com/corentings/chess/v2"

type Rules struct{}

func gameAt(fen string) (*chess.Game, error) {
	option, err := chess.FEN(fen)
	if err != nil {
		return nil, err
	}
	return chess.NewGame(option), nil
}

func (Rules) ApplyUCI(fen, uci string) (string, error) {
	game, err := gameAt(fen)
	if err != nil {
		return "", err
	}
	if err := game.PushNotationMove(uci, chess.UCINotation{}, nil); err != nil {
		return "", err
	}
	return game.Position().String(), nil
}

func (Rules) SAN(fen, uci string) (string, error) {
	game, err := gameAt(fen)
	if err != nil {
		return "", err
	}
	move, err := (chess.UCINotation{}).Decode(game.Position(), uci)
	if err != nil {
		return "", err
	}
	return (chess.AlgebraicNotation{}).Encode(game.Position(), move), nil
}

func (r Rules) IsCheckmateMove(fen, uci string) bool {
	game, err := gameAt(fen)
	if err != nil {
		return false
	}
	if err := game.PushNotationMove(uci, chess.UCINotation{}, nil); err != nil {
		return false
	}
	return game.Method() == chess.Checkmate
}
```

Run: `go test ./internal/chessrules -v`

Expected: PASS.

- [ ] **Step 5: Write the stable fingerprint test**

Create `internal/puzzles/fingerprint_test.go`:

```go
package puzzles

import (
	"testing"

	"chess-trainer/internal/domain"
)

func TestFingerprintIgnoresSourceMetadata(t *testing.T) {
	base := domain.Puzzle{
		DisplayedFEN: "7k/5Q2/6K1/8/8/8/8/8 w - - 0 1",
		Solver: domain.White,
		Solution: []domain.MoveNode{{UCI: "f7f8"}},
	}
	a := base
	a.Sources = []domain.SourceRef{{SourceID: "a", ExternalID: "1"}}
	b := base
	b.Sources = []domain.SourceRef{{SourceID: "b", ExternalID: "9"}}

	fa, err := Fingerprint(a)
	if err != nil {
		t.Fatal(err)
	}
	fb, err := Fingerprint(b)
	if err != nil {
		t.Fatal(err)
	}
	if fa != fb {
		t.Fatalf("fingerprints differ: %s != %s", fa, fb)
	}
}
```

Run: `go test ./internal/puzzles -run TestFingerprintIgnoresSourceMetadata -v`

Expected: FAIL because `Fingerprint` is undefined.

- [ ] **Step 6: Implement deterministic fingerprinting**

Create `internal/puzzles/fingerprint.go`:

```go
package puzzles

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"

	"chess-trainer/internal/domain"
)

func normalizeNodes(nodes []domain.MoveNode) []domain.MoveNode {
	out := make([]domain.MoveNode, len(nodes))
	for i, node := range nodes {
		out[i] = domain.MoveNode{
			UCI: strings.ToLower(strings.TrimSpace(node.UCI)),
			Children: normalizeNodes(node.Children),
		}
	}
	return out
}

func Fingerprint(p domain.Puzzle) (string, error) {
	payload := struct {
		FEN      string            `json:"fen"`
		Solver   domain.Color      `json:"solver"`
		Solution []domain.MoveNode `json:"solution"`
	}{strings.TrimSpace(p.DisplayedFEN), p.Solver, normalizeNodes(p.Solution)}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}
```

Run: `go test ./internal/puzzles ./internal/chessrules -v`

Expected: PASS.

- [ ] **Step 7: Commit the domain boundary**

```bash
git add go.mod go.sum internal/domain internal/chessrules internal/puzzles/fingerprint.go internal/puzzles/fingerprint_test.go
git commit -m "feat: define canonical chess puzzles"
```

### Task 3: Create application-support paths and versioned SQLite schemas

**Files:**
- Create: `internal/storage/paths.go`
- Test: `internal/storage/paths_test.go`
- Create: `internal/storage/sqlite.go`
- Test: `internal/storage/sqlite_test.go`
- Create: `internal/storage/migrations/puzzles/001.sql`
- Create: `internal/storage/migrations/user/001.sql`
- Create: `internal/storage/migrations/library/001.sql`
- Modify: `go.mod`
- Modify: `go.sum`

**Interfaces:**
- Produces: `storage.Paths{Root, PuzzlesDB, LibraryDB, UserDB, BackupsDir}`.
- Produces: `storage.Open(path string) (*sql.DB, error)` and `storage.Migrate(db, schema string) error`.

- [ ] **Step 1: Add pure-Go SQLite**

Run: `go get modernc.org/sqlite`

Expected: `go.mod` records `modernc.org/sqlite` and its matching transitive dependencies.

- [ ] **Step 2: Write the failing path test**

Create `internal/storage/paths_test.go`:

```go
package storage

import (
	"path/filepath"
	"testing"
)

func TestPathsAt(t *testing.T) {
	p := PathsAt("/tmp/support")
	if p.UserDB != filepath.Join("/tmp/support", "user.sqlite") {
		t.Fatalf("UserDB=%q", p.UserDB)
	}
	if p.BackupsDir != filepath.Join("/tmp/support", "backups") {
		t.Fatalf("BackupsDir=%q", p.BackupsDir)
	}
}
```

Run: `go test ./internal/storage -run TestPathsAt -v`

Expected: FAIL because `PathsAt` is undefined.

- [ ] **Step 3: Implement application-support paths**

Create `internal/storage/paths.go`:

```go
package storage

import (
	"os"
	"path/filepath"
)

type Paths struct {
	Root       string
	PuzzlesDB  string
	LibraryDB  string
	UserDB     string
	BackupsDir string
}

func PathsAt(root string) Paths {
	return Paths{
		Root: root,
		PuzzlesDB: filepath.Join(root, "puzzles.sqlite"),
		LibraryDB: filepath.Join(root, "library.sqlite"),
		UserDB: filepath.Join(root, "user.sqlite"),
		BackupsDir: filepath.Join(root, "backups"),
	}
}

func DefaultPaths() (Paths, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, err
	}
	return PathsAt(filepath.Join(base, "Chess Trainer")), nil
}

func (p Paths) Ensure() error {
	if err := os.MkdirAll(p.Root, 0o700); err != nil {
		return err
	}
	return os.MkdirAll(p.BackupsDir, 0o700)
}
```

- [ ] **Step 4: Write migrations with stable keys and indexes**

Create `internal/storage/migrations/puzzles/001.sql`:

```sql
CREATE TABLE IF NOT EXISTS sources (
  source_id TEXT PRIMARY KEY,
  kind TEXT NOT NULL,
  imported_at INTEGER NOT NULL,
  source_path TEXT NOT NULL,
  checksum TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS puzzles (
  fingerprint TEXT PRIMARY KEY,
  source_fen TEXT,
  prelude_uci TEXT,
  displayed_fen TEXT NOT NULL,
  solver TEXT NOT NULL CHECK (solver IN ('white','black')),
  solution_json TEXT NOT NULL,
  solution_plies INTEGER NOT NULL CHECK (solution_plies > 0)
);
CREATE TABLE IF NOT EXISTS puzzle_sources (
  fingerprint TEXT NOT NULL REFERENCES puzzles(fingerprint) ON DELETE CASCADE,
  source_id TEXT NOT NULL REFERENCES sources(source_id) ON DELETE CASCADE,
  external_id TEXT,
  rating INTEGER,
  popularity INTEGER,
  play_count INTEGER,
  source_url TEXT,
  metadata_json TEXT NOT NULL DEFAULT '{}',
  PRIMARY KEY (fingerprint, source_id)
);
CREATE TABLE IF NOT EXISTS puzzle_themes (
  fingerprint TEXT NOT NULL,
  source_id TEXT NOT NULL,
  theme TEXT NOT NULL,
  PRIMARY KEY (fingerprint, source_id, theme),
  FOREIGN KEY (fingerprint, source_id)
    REFERENCES puzzle_sources(fingerprint, source_id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS import_staging (
  import_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL,
  puzzle_json TEXT NOT NULL,
  PRIMARY KEY (import_id, ordinal)
);
CREATE INDEX IF NOT EXISTS idx_puzzle_sources_rating ON puzzle_sources(source_id, rating);
CREATE INDEX IF NOT EXISTS idx_puzzle_themes_theme ON puzzle_themes(source_id, theme, fingerprint);
```

Create `internal/storage/migrations/user/001.sql`:

```sql
CREATE TABLE IF NOT EXISTS profile (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  learner_rating REAL NOT NULL,
  session_size INTEGER NOT NULL CHECK (session_size IN (5,10,15)),
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE TABLE IF NOT EXISTS sessions (
  session_id TEXT PRIMARY KEY,
  mode TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL,
  current_index INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS session_items (
  session_id TEXT NOT NULL REFERENCES sessions(session_id) ON DELETE CASCADE,
  ordinal INTEGER NOT NULL,
  fingerprint TEXT NOT NULL,
  source_id TEXT NOT NULL,
  state_json TEXT NOT NULL,
  PRIMARY KEY (session_id, ordinal)
);
CREATE TABLE IF NOT EXISTS attempts (
  attempt_id TEXT PRIMARY KEY,
  session_id TEXT,
  fingerprint TEXT NOT NULL,
  source_id TEXT NOT NULL,
  started_at INTEGER NOT NULL,
  completed_at INTEGER,
  incorrect_moves INTEGER NOT NULL DEFAULT 0,
  hints_used INTEGER NOT NULL DEFAULT 0,
  solution_revealed INTEGER NOT NULL DEFAULT 0,
  first_try INTEGER NOT NULL DEFAULT 0,
  duration_ms INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS review_state (
  fingerprint TEXT PRIMARY KEY,
  due_at INTEGER NOT NULL,
  interval_index INTEGER NOT NULL,
  successful_reviews INTEGER NOT NULL,
  last_outcome TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_review_due ON review_state(due_at);
CREATE INDEX IF NOT EXISTS idx_attempts_fingerprint ON attempts(fingerprint, started_at);
```

Create `internal/storage/migrations/library/001.sql`:

```sql
CREATE TABLE IF NOT EXISTS library_metadata (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL
);
```

- [ ] **Step 5: Implement connection policy and migrations**

Create `internal/storage/sqlite.go` with embedded migrations, `sql.Open("sqlite", ...)`, `PRAGMA foreign_keys=ON`, `PRAGMA journal_mode=WAL`, a 5-second busy timeout, a one-connection write pool, and a `schema_migrations(version INTEGER PRIMARY KEY)` table. `Migrate(db, schema)` must apply the sorted `.sql` files for exactly one of `puzzles`, `user`, or `library` in a transaction and reject any other schema name.

Use the exact public functions `Open(path string) (*sql.DB, error)` and `Migrate(db *sql.DB, schema string) error`.

- [ ] **Step 6: Test all three databases**

Create `internal/storage/sqlite_test.go` with a table-driven test that opens three temporary files, calls `Migrate`, queries one required table in each schema, closes, reopens, calls `Migrate` again, and expects no error or duplicate migration row.

Run: `go test ./internal/storage -v`

Expected: PASS with three subtests: `puzzles`, `user`, and `library`.

- [ ] **Step 7: Commit persistence foundations**

```bash
git add go.mod go.sum internal/storage
git commit -m "feat: add local SQLite stores"
```

### Task 4: Implement atomic source staging and catalogue queries

**Files:**
- Create: `internal/puzzles/catalog.go`
- Create: `internal/puzzles/sqlite_catalog.go`
- Test: `internal/puzzles/sqlite_catalog_test.go`

**Interfaces:**
- Consumes: `domain.Puzzle`, `storage.Open`, and the puzzle schema.
- Produces: `Catalog.BeginImport`, `StagedImport.Add`, `Reject`, `Commit`, and `Abort`.
- Produces: `Catalog.Get`, `RatedCandidates`, and `FreePracticeCandidates`.

- [ ] **Step 1: Define the catalogue contracts**

Create `internal/puzzles/catalog.go`:

```go
package puzzles

import (
	"context"
	"time"

	"chess-trainer/internal/domain"
)

type Source struct {
	ID       string
	Kind     string
	Path     string
	Checksum string
	ImportedAt time.Time
}

type Rejection struct {
	Ordinal int64
	Reason  string
}

type ImportReport struct {
	Accepted  int64
	Duplicates int64
	Rejected  int64
	Examples  []Rejection
}

type StagedImport interface {
	Add(context.Context, domain.Puzzle) error
	Reject(Rejection)
	SetChecksum(string)
	Commit(context.Context) (ImportReport, error)
	Abort(context.Context) error
}

type Catalog interface {
	BeginImport(context.Context, Source) (StagedImport, error)
	Get(context.Context, string) (domain.Puzzle, error)
	RatedCandidates(context.Context, int, int, []string, int) ([]domain.Puzzle, error)
	FreePracticeCandidates(context.Context, string, *int, *int, []string, int) ([]domain.Puzzle, error)
}
```

- [ ] **Step 2: Write the atomic replacement test**

Create `internal/puzzles/sqlite_catalog_test.go` with a temporary migrated puzzle DB. Import source `lichess` containing puzzle A, commit, begin a second `lichess` import containing puzzle B, abort, and assert A still exists and B does not. Repeat the second import and commit; assert B exists, A is absent unless another source references it, and the report counts one accepted record.

Run: `go test ./internal/puzzles -run TestSQLiteCatalogAtomicSourceReplacement -v`

Expected: FAIL because `NewSQLiteCatalog` is undefined.

- [ ] **Step 3: Implement staged import semantics**

Create `internal/puzzles/sqlite_catalog.go` implementing the interfaces with these rules:

```text
BeginImport: generate UUID import_id and keep source metadata in memory.
Add: JSON-encode the complete canonical puzzle into import_staging in 1,000-row transactions.
Reject: increment rejected count and retain at most 100 examples.
SetChecksum: retain the lowercase SHA-256 checksum for the source row written at commit.
Commit transaction:
  reject commit if checksum was never set;
  delete the previous source row, cascading only its provenance and themes;
  insert the replacement source row;
  decode each staged puzzle;
  calculate maximum solution-tree depth as solution_plies and INSERT OR IGNORE canonical puzzles;
  count ignored rows as duplicates;
  insert puzzle_sources and source-scoped themes for this source;
  delete canonical puzzles with no remaining provenance;
  delete this import_id's staging rows;
  insert the new source row;
  commit.
Abort: delete only this import_id's staging rows.
```

Return `sql.ErrNoRows` from `Get` when absent. Candidate queries must use indexed rating/theme filters and randomise only after reducing to at most 500 eligible fingerprints.

- [ ] **Step 4: Add candidate-query tests**

Add tests that import rated puzzles at 1200, 1500, and 1800 with themes, then assert:

```go
candidates, err := catalog.RatedCandidates(ctx, 1400, 1600, nil, 10)
if err != nil || len(candidates) != 1 || *candidates[0].Rating != 1500 {
	t.Fatalf("candidates=%v err=%v", candidates, err)
}
```

Also assert exclusions and theme filters remove the matching fingerprints.

Run: `go test ./internal/puzzles -v`

Expected: PASS.

- [ ] **Step 5: Commit the catalogue**

```bash
git add internal/puzzles/catalog.go internal/puzzles/sqlite_catalog.go internal/puzzles/sqlite_catalog_test.go
git commit -m "feat: add atomic puzzle catalogue"
```

### Task 5: Stream and normalize the Lichess puzzle database

**Files:**
- Create: `internal/puzzles/lichess_importer.go`
- Test: `internal/puzzles/lichess_importer_test.go`
- Create: `internal/storage/space_darwin.go`
- Test: `internal/storage/space_darwin_test.go`
- Modify: `go.mod`
- Modify: `go.sum`

**Interfaces:**
- Consumes: `Catalog.BeginImport`, `chessrules.Rules`, and `puzzles.Fingerprint`.
- Produces: `LichessImporter.Import(ctx, sourceID, path, progress) (ImportReport, error)`.

- [ ] **Step 1: Add streaming Zstandard support**

Run: `go get github.com/klauspost/compress/zstd`

- [ ] **Step 2: Write and implement disk-space preflight**

Define `storage.AvailableBytes(path string) (uint64, error)` in `space_darwin.go` using `syscall.Statfs`. Add a pure `RequiredImportBytes(compressedSize int64) uint64` helper returning `compressedSize*10 + 512*1024*1024`. Test the arithmetic directly and integration-test `AvailableBytes(t.TempDir()) > 0`. The importer must fail before `BeginImport` when available bytes are below the requirement.

- [ ] **Step 3: Write a compressed-fixture test covering Lichess setup semantics**

Create a test helper that writes this CSV through `zstd.NewWriter` into a temporary `.csv.zst`:

```csv
PuzzleId,FEN,Moves,Rating,RatingDeviation,Popularity,NbPlays,Themes,GameUrl,OpeningTags
mate1,8/5Q1k/6K1/8/8/8/8/8 b - - 0 1,h7h8 f7f8,1200,60,95,200,mate mateIn1,https://lichess.org/example,
bad,not-a-fen,a1a2,1500,60,10,2,short,,
```

Use an in-memory fake `StagedImport` to capture normalized puzzles. Assert the first captured puzzle has displayed side-to-move White, prelude `h7h8`, solution root `f7f8`, rating 1200, two themes, and a 64-character fingerprint. Assert the report rejects the invalid FEN without losing the valid row.

Run: `go test ./internal/puzzles -run TestLichessImporterNormalizesSetupMove -v`

Expected: FAIL because `LichessImporter` is undefined.

- [ ] **Step 4: Implement the streaming importer**

Create `internal/puzzles/lichess_importer.go` with:

```go
type Progress struct {
	RowsRead int64 `json:"rowsRead"`
	BytesRead int64 `json:"bytesRead"`
}

type ProgressSink func(Progress)

type LichessImporter struct {
	Catalog Catalog
	Rules   chessrules.Rules
}

func (i LichessImporter) Import(
	ctx context.Context,
	sourceID string,
	path string,
	progress ProgressSink,
) (ImportReport, error)
```

Implementation requirements:

- Wrap the file in a counting reader, then `zstd.NewReader`, then `csv.NewReader` with `ReuseRecord = true`.
- Tee the compressed bytes through SHA-256; on EOF call `StagedImport.SetChecksum` with the lowercase hexadecimal digest.
- Validate the exact header names by column lookup rather than fixed ordering.
- Check `ctx.Err()` once per record.
- Parse moves with `strings.Fields`; require at least two plies.
- Apply moves[0] to source FEN; the resulting FEN is displayed.
- Build a single-child move tree from moves[1:].
- Determine solver from the displayed FEN active-color field.
- Parse nullable numeric fields without turning missing data into zero.
- Normalize and sort themes before fingerprinting.
- Emit progress every 10,000 rows and once at completion.
- Send parse or validation failures to `Reject` with 1-based CSV row number.
- Call `Abort` on cancellation or fatal I/O; call `Commit` only on EOF.

- [ ] **Step 5: Add cancellation and malformed-row tests**

Test a reader with 20,000 valid generated rows, cancel after the first progress callback, and assert `context.Canceled`, `Abort` called once, and `Commit` never called. Test missing required header and truncated Zstandard data as fatal errors.

Run: `go test ./internal/puzzles -run 'TestLichessImporter' -v`

Expected: PASS.

- [ ] **Step 6: Run a bounded-memory integration probe**

Add `TestLichessImporterStreamingAllocations` using 100,000 generated rows and `runtime.ReadMemStats`; assert peak allocation growth remains below 128 MiB. Skip only when `testing.Short()` is true.

Run: `go test ./internal/puzzles -run TestLichessImporterStreamingAllocations -count=1 -v`

Expected: PASS without creating a decompressed CSV file.

- [ ] **Step 7: Commit the Lichess importer**

```bash
git add go.mod go.sum internal/puzzles/lichess_importer.go internal/puzzles/lichess_importer_test.go internal/storage/space_darwin.go internal/storage/space_darwin_test.go
git commit -m "feat: stream Lichess puzzle imports"
```

### Task 6: Add cancellable import jobs and Wails bindings

**Files:**
- Create: `internal/importjob/service.go`
- Test: `internal/importjob/service_test.go`
- Create: `internal/app/services.go`
- Modify: `app.go`
- Modify: `main.go`

**Interfaces:**
- Consumes: `LichessImporter.Import`.
- Produces: `App.StartLichessImport(path string) (string, error)`, `App.CancelImport(jobID string) error`, and `App.GetImportResult(jobID string)`.
- Emits Wails event `import:progress` carrying `{jobId, rowsRead, bytesRead}` and `import:finished` carrying the final report or error.

- [ ] **Step 1: Write job lifecycle tests with a blocking fake importer**

Test that `Start` returns immediately with a UUID, progress is forwarded with that UUID, `Cancel` cancels the exact job context, and `Result` changes from running to cancelled. A second concurrent start must have an independent context.

Run: `go test ./internal/importjob -v`

Expected: FAIL because `Service` is undefined.

- [ ] **Step 2: Implement the job service**

Create `internal/importjob/service.go` defining:

```go
type Status string

const (
	Running Status = "running"
	Succeeded Status = "succeeded"
	Failed Status = "failed"
	Cancelled Status = "cancelled"
)

type Result struct {
	JobID  string               `json:"jobId"`
	Status Status               `json:"status"`
	Report puzzles.ImportReport `json:"report"`
	Error  string               `json:"error,omitempty"`
}

type Emitter interface {
	Progress(string, puzzles.Progress)
	Finished(Result)
}
```

Guard the job map with `sync.Mutex`; remove cancel functions only after storing terminal results. Convert `context.Canceled` to `Cancelled`, and all other errors to `Failed`.

Run: `go test ./internal/importjob -v`

Expected: PASS under `go test -race`.

- [ ] **Step 3: Compose databases and services**

Create `internal/app/services.go` with `Open(paths storage.Paths) (*Services, error)` that ensures directories, opens/migrates all three databases, creates `SQLiteCatalog`, `LichessImporter`, and `importjob.Service`, and closes already-opened resources on any failure. `Services.Close()` closes every DB once.

- [ ] **Step 4: Keep Wails code as a thin adapter**

Modify `app.go` so `startup(ctx)` stores the Wails context and the emitter calls `runtime.EventsEmit`. Public bindings delegate to the import job service and contain no CSV, SQLite, or scheduling logic.

Modify `main.go` to construct default paths and services before `wails.Run`, bind `App`, and call `services.Close()` in `OnShutdown`.

- [ ] **Step 5: Verify bindings and build**

Run:

```bash
go test -race ./...
wails generate module
wails build
```

Expected: PASS; generated TypeScript bindings expose the three import methods.

- [ ] **Step 6: Commit import orchestration**

```bash
git add app.go main.go internal/app internal/importjob frontend/wailsjs
git commit -m "feat: expose cancellable puzzle imports"
```

### Task 7: Implement rating, review intervals, and deterministic session selection

**Files:**
- Create: `internal/training/rating.go`
- Test: `internal/training/rating_test.go`
- Create: `internal/training/review.go`
- Test: `internal/training/review_test.go`
- Create: `internal/training/scheduler.go`
- Test: `internal/training/scheduler_test.go`

**Interfaces:**
- Produces: `UpdateRating(current, puzzle float64, score float64, min, max float64) float64`.
- Produces: `NextReview(now time.Time, current ReviewState, outcome Outcome) ReviewState`.
- Produces: `Scheduler.BuildGuided(ctx, profile, now, random) ([]ScheduledPuzzle, error)`.

- [ ] **Step 1: Write exact rating and review tests**

Use these assertions:

```go
if got := UpdateRating(1500, 1500, 1, 400, 3000); got != 1512 {
	t.Fatalf("got %v", got)
}
if got := UpdateRating(1500, 1500, 0, 400, 3000); got != 1488 {
	t.Fatalf("got %v", got)
}
```

For reviews, assert missed/hinted/revealed outcomes are due in one day, then clean successes advance through 3, 7, 21, and 60 days, and a later miss resets to one day.

Run: `go test ./internal/training -run 'TestUpdateRating|TestNextReview' -v`

Expected: FAIL because functions are undefined.

- [ ] **Step 2: Implement rating and review functions**

Implement the design formula with K=24, validate score is exactly 0, 0.5, or 1, round to the nearest whole rating for persistence, and clamp after updating. Define review intervals as `[]time.Duration{24*time.Hour, 72*time.Hour, 7*24*time.Hour, 21*24*time.Hour, 60*24*time.Hour}`.

Run: `go test ./internal/training -run 'TestUpdateRating|TestNextReview' -v`

Expected: PASS.

- [ ] **Step 3: Write scheduler tests with fakes and fixed randomness**

Test a ten-item session with six overdue reviews and rated unseen puzzles. Assert exactly four reviews are chosen first, six unseen puzzles fill the rest, the first unseen query uses learner rating ±100, and widening calls use ±200, ±300, then ±400 only when needed. Assert review and free-practice items never update learner rating.

Run: `go test ./internal/training -run TestScheduler -v`

Expected: FAIL because `Scheduler` is undefined.

- [ ] **Step 4: Implement the scheduler**

Define repository ports in `scheduler.go`:

```go
type CatalogPort interface {
	Get(context.Context, string) (domain.Puzzle, error)
	RatedCandidates(context.Context, int, int, []string, int) ([]domain.Puzzle, error)
}

type UserPort interface {
	DueReviews(context.Context, time.Time, int) ([]ReviewState, error)
	RecentFingerprints(context.Context, int) ([]string, error)
}
```

`BuildGuided` must cap due reviews at four, preserve due-date order, use a supplied `*rand.Rand` to shuffle only eligible new candidates, avoid duplicate fingerprints, and return fewer than the configured session size only when every ±400 query is exhausted.

Run: `go test ./internal/training -v`

Expected: PASS and deterministic results across repeated runs with the same seed.

- [ ] **Step 5: Commit adaptive scheduling**

```bash
git add internal/training/rating.go internal/training/rating_test.go internal/training/review.go internal/training/review_test.go internal/training/scheduler.go internal/training/scheduler_test.go
git commit -m "feat: schedule adaptive puzzle sessions"
```

### Task 8: Persist and resume the complete solving workflow

**Files:**
- Create: `internal/domain/training.go`
- Create: `internal/training/user_store.go`
- Create: `internal/training/service.go`
- Test: `internal/training/service_test.go`
- Modify: `internal/app/services.go`
- Modify: `app.go`

**Interfaces:**
- Produces: `TrainingService.StartGuided`, `Resume`, `PlayMove`, `UseHint`, `Reveal`, `Pause`, and `Summary`.
- Produces Wails bindings with the same names and JSON-safe view models.

- [ ] **Step 1: Define commands and view models**

Create `internal/domain/training.go` defining `SessionView`, `PuzzleView`, `MoveResult`, `HintResult`, and `SessionSummary`. `PuzzleView` exposes displayed FEN, optional prelude, solver, current solution-node path, progress, and hint level; it never exposes future solution moves to the frontend.

- [ ] **Step 2: Write a restart-resume test first**

Using temporary migrated puzzle/user DBs, start a two-puzzle session, play the first correct learner move, close both services, reopen, call `Resume`, and assert the exact FEN, current solution path, puzzle ordinal, incorrect count, and hint level are restored.

Run: `go test ./internal/training -run TestServiceResumesAfterEveryMove -v`

Expected: FAIL because `NewService` is undefined.

- [ ] **Step 3: Implement transactional user persistence**

Create `internal/training/user_store.go` with methods that write session item state and attempt counters in one transaction per learner action. Use JSON only for the branch cursor/presentation state; keep reportable counters in typed columns. Ensure `Pause` changes status but does not complete the attempt.

- [ ] **Step 4: Implement solution-tree traversal**

`PlayMove` must:

- Parse and validate the UCI move against current FEN.
- Accept any child move matching the current solution node.
- For a mate-in-one leaf, accept any legal checkmating move through `Rules.IsCheckmateMove`.
- On wrong moves, increment incorrect count, persist, and return the unchanged FEN.
- On correct moves, persist the learner move, automatically apply the source reply when the chosen branch has exactly one reply, and return the new FEN.
- Complete the attempt only at a solution leaf.
- Update rating only for an unseen rated guided puzzle.
- Create/reset review state for wrong, hinted, or revealed attempts.

`UseHint` returns theme/generic text at level 1, source square at level 2, target square at level 3, then enables `Reveal`. `Reveal` replays the remaining canonical line and records score 0.

- [ ] **Step 5: Add table tests for wrong moves, hints, reveal, mate alternatives, and session completion**

Each test must reopen the service before its final assertion so persistence, not in-memory state, is verified. Add a source-reimport case that removes the currently queued fingerprint: `Resume` must mark that item unavailable, advance without creating a failed attempt, and preserve its historical attempts for reporting.

Run: `go test -race ./internal/training -v`

Expected: PASS.

- [ ] **Step 6: Expose thin Wails training bindings**

Add bindings:

```go
func (a *App) StartGuided() (domain.SessionView, error)
func (a *App) ResumeSession() (*domain.SessionView, error)
func (a *App) PlayMove(sessionID, uci string) (domain.MoveResult, error)
func (a *App) UseHint(sessionID string) (domain.HintResult, error)
func (a *App) RevealSolution(sessionID string) (domain.MoveResult, error)
func (a *App) PauseSession(sessionID string) error
```

Regenerate bindings and run `go test -race ./...`.

- [ ] **Step 7: Commit the persisted trainer core**

```bash
git add app.go internal/app internal/domain/training.go internal/training frontend/wailsjs
git commit -m "feat: persist puzzle solving sessions"
```

### Task 9: Build the frontend shell, home hub, onboarding, and import progress

**Files:**
- Create: `frontend/src/lib/api.ts`
- Create: `frontend/src/lib/navigation.ts`
- Create: `frontend/src/styles/tokens.css`
- Create: `frontend/src/styles/app.css`
- Create: `frontend/src/components/home/HomeHub.svelte`
- Create: `frontend/src/components/import/ImportPanel.svelte`
- Create: `frontend/src/components/parent/InitialSetup.svelte`
- Test: corresponding `*.test.ts` files
- Modify: `frontend/src/App.svelte`
- Modify: `frontend/src/main.ts`

**Interfaces:**
- Consumes generated bindings from Tasks 6 and 8.
- Produces the approved home hub and resumable import/onboarding flows.

- [ ] **Step 1: Introduce an injectable API boundary**

Create `frontend/src/lib/api.ts` exporting an `AppAPI` interface matching Wails bindings and a production implementation that imports generated functions. Export `setAPIForTests(fake)` so component tests never need a Wails runtime.

- [ ] **Step 2: Write failing component tests**

Test that:

- Initial setup asks for a starting rating and session size.
- Home hub renders `Continue today's training`, `Free Practice`, and `Game Library`.
- With no active session, the main label is `Start today's training`.
- Import panel renders rows read, Cancel while running, and accepted/duplicate/rejected counts at completion.

Run: `cd frontend && npm test -- --run`

Expected: FAIL because components do not exist.

- [ ] **Step 3: Implement navigation and visual tokens**

Use a discriminated screen state: `setup | home | import | puzzle | practice | parent | games`. Use the approved warm ivory background, forest green primary action, amber progress, minimum 44px interactive targets, visible keyboard focus, and reduced-motion media query.

- [ ] **Step 4: Implement setup, import, and home components**

Subscribe to Wails `import:progress` and `import:finished` events only while `ImportPanel` is mounted and call the returned unsubscribe functions on destroy. The home card resumes an active session before creating a new one. Keep settings behind the small gear action.

- [ ] **Step 5: Verify accessible behavior**

Run:

```bash
cd frontend
npm test -- --run
npm run build
```

Expected: PASS with no TypeScript errors.

- [ ] **Step 6: Commit child-facing navigation**

```bash
git add frontend/src frontend/package.json frontend/package-lock.json
git commit -m "feat: add home and import experience"
```

### Task 10: Build the custom board-first puzzle screen

**Files:**
- Create: `frontend/src/lib/fen.ts`
- Test: `frontend/src/lib/fen.test.ts`
- Create: `frontend/src/components/chess/ChessBoard.svelte`
- Test: `frontend/src/components/chess/ChessBoard.test.ts`
- Create: `frontend/src/components/puzzle/PuzzleScreen.svelte`
- Test: `frontend/src/components/puzzle/PuzzleScreen.test.ts`

**Interfaces:**
- Consumes: `SessionView`, `PlayMove`, `UseHint`, `RevealSolution`, and `PauseSession`.
- Produces: click-click and drag-drop UCI moves, solver orientation, prelude animation, progressive hints, and session progress.

- [ ] **Step 1: Write FEN presentation tests**

Test standard and sparse FEN piece placement, reject a rank not totaling eight squares, and assert `orientSquares('black')` begins with `h1` and ends with `a8`.

Run: `cd frontend && npm test -- --run src/lib/fen.test.ts`

Expected: FAIL because parser functions are undefined.

- [ ] **Step 2: Implement the pure FEN parser**

Create `fen.ts` with:

```ts
export type Piece = { color: 'white' | 'black'; role: 'pawn'|'knight'|'bishop'|'rook'|'queen'|'king' }
export type BoardMap = Record<string, Piece>
export function parseFEN(fen: string): BoardMap
export function orientSquares(color: 'white'|'black'): string[]
```

Map `PNBRQK`/lowercase to typed pieces, validate eight ranks and eight files, and ignore non-placement FEN fields for rendering.

- [ ] **Step 3: Write board interaction tests**

Render a known FEN, click e2 then e4, and assert `move` emits `{uci:'e2e4'}`. Repeat with drag events. Flip to black and assert square order and coordinate labels. Assert disabled state emits nothing during prelude or automatic reply animation.

- [ ] **Step 4: Implement `ChessBoard.svelte`**

Render 64 semantic buttons with Unicode chess glyphs, accessible piece names, and a bundled system-font fallback stack. Track selected source, accept drop targets, emit UCI with promotion chosen through a small queen/rook/bishop/knight dialog, and expose highlight props for last move, wrong move, hint source, and hint target.

- [ ] **Step 5: Write puzzle-flow tests**

With a fake API, assert prelude animation precedes input, wrong move restores position and shows `Try again`, correct move applies returned FEN, third hint reveals source/target, `Show solution` appears only after level 3, Pause returns home, and the completion summary contains first-try/retry/hint/reveal counts.

- [ ] **Step 6: Implement `PuzzleScreen.svelte`**

Keep the approved board-first layout: board plus a narrow panel containing solver color, `Find the best move`, progress, Hint, and Pause. Do not display rating or future line. Respect `prefers-reduced-motion` by replacing prelude animation with an immediate position update.

- [ ] **Step 7: Verify frontend and packaged build**

Run:

```bash
cd frontend
npm test -- --run
npm run build
cd ..
wails build
```

Expected: PASS and the packaged app opens to the home hub, then renders the board-first solver.

- [ ] **Step 8: Commit the puzzle UI**

```bash
git add frontend/src/lib/fen.ts frontend/src/lib/fen.test.ts frontend/src/components/chess frontend/src/components/puzzle
git commit -m "feat: add board-first puzzle solving"
```

### Task 11: Add Free Practice and parent progress

**Files:**
- Create: `internal/profile/service.go`
- Test: `internal/profile/service_test.go`
- Modify: `internal/training/service.go`
- Modify: `app.go`
- Create: `frontend/src/components/practice/FreePractice.svelte`
- Test: `frontend/src/components/practice/FreePractice.test.ts`
- Create: `frontend/src/components/parent/ParentDashboard.svelte`
- Test: `frontend/src/components/parent/ParentDashboard.test.ts`

**Interfaces:**
- Produces: source/rating/theme/length filtered Free Practice that does not change learner rating.
- Produces: parent settings and summary metrics specified by the design.

- [ ] **Step 1: Write backend summary tests**

Seed attempts across themes and sessions. Assert `ProfileService.Summary` returns learner rating trend, first-attempt accuracy, hint rate, per-theme accuracy, due-review count, and the most recent 20 sessions. Assert `UpdateSettings` accepts only session sizes 5/10/15 and ratings inside the active Lichess catalogue range.

- [ ] **Step 2: Implement profile queries and settings**

Use SQL aggregation rather than loading all attempts. Return percentages as numbers from 0 through 100 rounded to one decimal. Add bindings `GetProfile`, `UpdateProfile`, `GetParentSummary`, `GetPracticeFilters`, and `StartFreePractice`.

- [ ] **Step 3: Test and implement Free Practice**

Component tests select source, optional rating range, themes, and maximum solution length, start a session, and verify the API receives exact filters. Backend tests verify attempts and review creation are saved while learner rating remains unchanged.

- [ ] **Step 4: Test and implement parent dashboard**

Render compact metric cards and accessible tables; do not require chart libraries. Show rating trend as a small in-project SVG polyline with textual min/current/max equivalents. Include session-size and starting/current-rating controls plus an `Import puzzles` action.

- [ ] **Step 5: Run all tests and commit**

Run:

```bash
go test -race ./...
cd frontend
npm test -- --run
npm run build
```

Expected: PASS.

```bash
git add app.go internal/profile internal/training frontend/src/components/practice frontend/src/components/parent frontend/wailsjs
git commit -m "feat: add practice and parent progress"
```

### Task 12: Add backup, recovery, end-to-end tests, and macOS acceptance checks

**Files:**
- Create: `internal/backup/service.go`
- Test: `internal/backup/service_test.go`
- Modify: `app.go`
- Create: `frontend/src/components/parent/BackupPanel.svelte`
- Test: `frontend/src/components/parent/BackupPanel.test.ts`
- Create: `frontend/src/components/parent/RecoveryPanel.svelte`
- Test: `frontend/src/components/parent/RecoveryPanel.test.ts`
- Create: `frontend/playwright.config.ts`
- Create: `frontend/tests/trainer.spec.ts`
- Create: `internal/puzzles/full_import_test.go`
- Create: `docs/operations/local-build.md`

**Interfaces:**
- Produces: validated backup/restore for `user.sqlite`, with optional `library.sqlite`.
- Produces: repeatable macOS build and acceptance commands.

- [ ] **Step 1: Write backup atomicity tests**

Create a profile and attempt, export to a temporary `.zip`, inspect manifest version and SHA-256 hashes, mutate the live DB, restore, and assert original data returns. Corrupt the archived DB and assert restore rejects it without altering live files.

- [ ] **Step 2: Implement backup and restore**

Use SQLite `VACUUM INTO` for consistent snapshots, zip `manifest.json` plus requested DB files, validate every hash and `PRAGMA integrity_check` before restore, close service DBs before atomic rename, and retain timestamped pre-restore copies in `backups/`.

- [ ] **Step 3: Bind file dialogs and add the parent backup panel**

Expose `CreateBackup(includeLibrary bool)` and `RestoreBackup(path string)` through `app.go`. Use Wails save/open dialogs only in the binding adapter; pass selected paths to `BackupService`.

- [ ] **Step 4: Add startup integrity recovery**

Before constructing repositories, run `PRAGMA quick_check` on all existing databases. On failure, return a typed `storage.IntegrityError{Path, Detail}` to `main.go`; launch the Wails shell in recovery mode rather than replacing data. `RecoveryPanel` offers only `Restore backup`, `Open data folder`, and `Quit`. Bind `OpenDataFolder` through `runtime.BrowserOpenURL(url.URL{Scheme: "file", Path: paths.Root}.String())`. Tests must corrupt a copied SQLite file, verify recovery mode, and verify no new profile or catalogue is created over it.

- [ ] **Step 5: Add browser-level flow with a fake API**

Configure Playwright for the Vite dev server. `trainer.spec.ts` must cover setup, completed import report, home hub, wrong then correct puzzle move, pause/resume, third hint and reveal, session summary, Free Practice filters, and parent summary.

Run: `cd frontend && npm run test:e2e`

Expected: all scenarios PASS in WebKit and Chromium.

- [ ] **Step 6: Add a full local database import test**

`internal/puzzles/full_import_test.go` reads `CHESS_TRAINER_LICHESS_PATH`; when unset it skips with an explicit message. When set to the downloaded file it imports into a temporary DB, asserts accepted count is greater than one million, rejected ratio is below 0.1%, RSS growth remains bounded rather than proportional to decompressed size, and a rated candidate query completes within 250 ms.

Run:

```bash
CHESS_TRAINER_LICHESS_PATH="$PWD/Lichess Puzzle Database.csv.zst" go test ./internal/puzzles -run TestFullLichessImport -count=1 -v
```

Expected: PASS and no decompressed `.csv` appears.

- [ ] **Step 7: Run final verification and package smoke test**

Run:

```bash
go test -race ./...
cd frontend
npm test -- --run
npm run test:e2e
npm run build
cd ..
wails build -clean
open "build/bin/Chess Trainer.app"
```

Manually verify double-click launch, import cancellation, guided-session resume after force quit, no listening TCP socket, and home hub readiness within three seconds after import. Record the commands and expected output in `docs/operations/local-build.md`.

- [ ] **Step 8: Commit the complete Lichess trainer**

```bash
git add main.go app.go internal/app internal/backup internal/storage internal/puzzles/full_import_test.go frontend/src/components/parent frontend/playwright.config.ts frontend/tests docs/operations/local-build.md frontend/wailsjs
git commit -m "feat: complete local Lichess trainer"
```

## Plan 1 Completion Gate

Before starting plan 2, confirm:

```bash
git status --short
go test -race ./...
npm --prefix frontend test -- --run
npm --prefix frontend run test:e2e
npm --prefix frontend run build
wails build -clean
```

Expected: clean worktree, all automated checks PASS, `build/bin/Chess Trainer.app` launches by double-click, the full downloaded Lichess database imports without a decompressed intermediate file, and a guided session survives process termination.

## Primary References

- [Wails v2 stable repository and install command](https://github.com/wailsapp/wails)
- [Wails Svelte TypeScript project template](https://wails.io/docs/gettingstarted/firstproject/)
- [Wails production build](https://wails.io/docs/gettingstarted/building/)
- [Lichess puzzle format notes](https://database.lichess.org/)
- [corentings/chess Go API](https://pkg.go.dev/github.com/corentings/chess/v2)
- [modernc SQLite driver](https://pkg.go.dev/modernc.org/sqlite)
- [klauspost Zstandard decoder](https://pkg.go.dev/github.com/klauspost/compress/zstd)
