# Chess Trainer PGN Puzzles and Game Library Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the verified Lichess trainer with tactical-PGN puzzle import and a searchable, annotation-preserving, three-pane historical-game library.

**Architecture:** Reuse the Go chess-rules adapter, staged puzzle catalogue, import-job service, and custom board delivered by plan 1. A shared PGN reader produces either tactical puzzle records or full game trees; separate adapters persist those records in `puzzles.sqlite` and `library.sqlite`. This is plan 2 of 3 and starts only after the plan-1 completion gate passes.

**Tech Stack:** Existing Go/Wails/Svelte stack, `github.com/corentings/chess/v2` scanner with variation expansion, SQLite indexed search, Vitest/Testing Library, and Playwright.

## Global Constraints

- Complete `2026-07-14-chess-trainer-foundation-and-puzzles.md` first.
- Preserve every ordinary PGN's headers, starting position, full variation tree, comments, numeric annotation glyphs, and original record.
- Tactical PGNs without reliable ratings/themes are Free Practice only until they enter review through an attempt.
- Route `[SetUp "1"]` + `[FEN]` + exactly one player named `solver` to tactical import; prompt for every ambiguous PGN.
- Reject illegal records individually without losing valid records from the same file.
- Do not import the deferred 432k FEN collection.
- Do not add engine evaluation, PGN editing, autoplay, online fetching, or pause-and-guess behavior.
- Use the same test-first and per-task commit discipline as plan 1.

## Planned File Map

- `internal/domain/game.go` — canonical game headers and recursive move tree.
- `internal/pgn/reader.go` — multi-record streaming PGN reader preserving original text.
- `internal/pgn/detector.go` — tactical/game/ambiguous classification.
- `internal/puzzles/tactics_pgn_importer.go` — solver-tag PGN to canonical puzzle.
- `internal/games/repository.go` — game storage/search port.
- `internal/games/sqlite_repository.go` — `library.sqlite` adapter.
- `internal/games/importer.go` — ordinary PGN batch importer.
- `internal/games/viewer.go` — position/move-tree navigation view models.
- `internal/storage/migrations/library/002.sql` — indexed game schema.
- `internal/importjob/pgn_service.go` — routes confirmed import mode through existing job lifecycle.
- `frontend/src/components/import/PgnImportChoice.svelte` — ambiguity prompt and reports.
- `frontend/src/components/games/GameLibrary.svelte` — selected three-pane workspace.
- `frontend/src/components/games/MoveTree.svelte` — recursive notation/variation UI.

---

### Task 1: Define a lossless canonical game tree and streaming PGN reader

**Files:**
- Create: `internal/domain/game.go`
- Create: `internal/pgn/reader.go`
- Test: `internal/pgn/reader_test.go`

**Interfaces:**
- Consumes: `github.com/corentings/chess/v2` PGN scanner and chess rules.
- Produces: `domain.GameRecord`, `domain.GameNode`, and `pgn.Reader.Next() (domain.GameRecord, error)`.

- [ ] **Step 1: Define the game model**

Create `internal/domain/game.go`:

```go
package domain

type GameNode struct {
	ID         string     `json:"id"`
	ParentID   string     `json:"parentId,omitempty"`
	Ply        int        `json:"ply"`
	UCI        string     `json:"uci"`
	SAN        string     `json:"san"`
	FENAfter   string     `json:"fenAfter"`
	Comments   []string   `json:"comments"`
	NAGs       []int      `json:"nags"`
	Variations []GameLine `json:"variations"`
}

type GameLine struct {
	Nodes []GameNode `json:"nodes"`
}

type GameRecord struct {
	Fingerprint string            `json:"fingerprint"`
	Headers     map[string]string `json:"headers"`
	StartingFEN string            `json:"startingFen"`
	MainLine    GameLine          `json:"mainLine"`
	OriginalPGN string            `json:"originalPgn"`
	Warnings    []string          `json:"warnings"`
}
```

Node IDs are deterministic path IDs: main-line nodes `m.1`, `m.2`; a first variation branching after `m.3` is `m.3.v1.1`, then `m.3.v1.2`.

- [ ] **Step 2: Write a lossless fixture test**

Use this exact fixture:

```pgn
[Event "Example"]
[Site "Home"]
[Date "2026.07.14"]
[Round "1"]
[White "Alpha"]
[Black "Beta"]
[Result "1-0"]

1. e4 {King pawn} e5 2. Nf3 $1 Nc6 (2... Nf6 {Petroff}) 3. Bb5 a6 1-0
```

Assert headers, original text, six main-line plies, the comment on e4, NAG 1 on Nf3, one variation containing Nf6 and its comment, legal UCI moves, and FEN after every node.

Run: `go test ./internal/pgn -run TestReaderPreservesAnnotatedGame -v`

Expected: FAIL because `NewReader` is undefined.

- [ ] **Step 3: Implement record framing and parsing**

Create `internal/pgn/reader.go` with:

```go
type Reader struct {
	scanner *bufio.Scanner
	pending string
	ordinal int64
}

type RecordError struct {
	Ordinal int64
	Err     error
}

func (e RecordError) Error() string {
	return fmt.Sprintf("PGN record %d: %v", e.Ordinal, e.Err)
}

func (e RecordError) Unwrap() error { return e.Err }

func NewReader(input io.Reader) *Reader {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 64*1024), 16*1024*1024)
	return &Reader{scanner: scanner}
}
```

Implement `Next() (domain.GameRecord, error)`. It must stream one tag-pair/movetext record at a time, retain exact record bytes, call the chess scanner with variation expansion, traverse the resulting move tree, compute UCI/SAN/FEN, copy comments and NAGs, and hash a canonical serialization containing sorted headers plus the complete annotated tree. Accept adjacent tag pairs such as `[Result "*"][SetUp "1"]` by inserting an internal parse-only line break between `][` boundaries outside quoted strings while preserving `OriginalPGN` exactly. Return `io.EOF` only between records; wrap record-local failures in `RecordError` so batch import can continue. If a six-field FEN fails only because its castling field claims unavailable rooks or kings, retry internally with castling `-`, preserve the original PGN, use the sanitized FEN for navigation, and append warning `sanitized invalid castling rights`; do not sanitize any other FEN error.

- [ ] **Step 4: Add multi-game and malformed-record tests**

Concatenate two valid games with one malformed game between them. Assert calls return valid game 1, `RecordError{Ordinal:2}`, valid game 3, then `io.EOF`. Add a game with `[SetUp "1"]` and FEN to verify a nonstandard starting position, and a solver-tagged fixture with `[Result "*"][SetUp "1"]` on one line to verify the referenced collection's adjacent-tag quirk.

Run: `go test ./internal/pgn -v`

Expected: PASS.

- [ ] **Step 5: Commit the shared PGN reader**

```bash
git add internal/domain/game.go internal/pgn/reader.go internal/pgn/reader_test.go
git commit -m "feat: parse annotated PGN trees"
```

### Task 2: Detect tactical, ordinary, and ambiguous PGNs

**Files:**
- Create: `internal/pgn/detector.go`
- Test: `internal/pgn/detector_test.go`

**Interfaces:**
- Consumes: parsed `domain.GameRecord` headers.
- Produces: `pgn.Kind` values `tactical`, `game`, and `ambiguous` plus a human-readable reason.

- [ ] **Step 1: Write classification tests**

Use a table:

```go
tests := []struct {
	name string
	headers map[string]string
	want Kind
}{
	{"white solver", map[string]string{"SetUp":"1", "FEN":"8/8/8/8/8/8/5K2/7k w - - 0 1", "White":"solver", "Black":"?"}, Tactical},
	{"black solver", map[string]string{"SetUp":"1", "FEN":"8/8/8/8/8/8/5K2/7k b - - 0 1", "White":"?", "Black":"solver"}, Tactical},
	{"ordinary", map[string]string{"White":"Capablanca", "Black":"Marshall", "Result":"1-0"}, Game},
	{"setup game without solver", map[string]string{"SetUp":"1", "FEN":"8/8/8/8/8/8/5K2/7k w - - 0 1", "White":"Alpha", "Black":"Beta"}, Game},
	{"solver without setup", map[string]string{"White":"solver", "Black":"Beta"}, Ambiguous},
	{"two solvers", map[string]string{"SetUp":"1", "FEN":"8/8/8/8/8/8/5K2/7k w - - 0 1", "White":"solver", "Black":"solver"}, Ambiguous},
}
```

Run: `go test ./internal/pgn -run TestDetect -v`

Expected: FAIL because `Detect` is undefined.

- [ ] **Step 2: Implement strict detection**

Create `internal/pgn/detector.go` defining:

```go
type Kind string

const (
	Tactical Kind = "tactical"
	Game Kind = "game"
	Ambiguous Kind = "ambiguous"
)

type Detection struct {
	Kind Kind `json:"kind"`
	Reason string `json:"reason"`
}
```

Implement `Detect(headers map[string]string) Detection`. Compare `solver` case-insensitively after trimming. Tactical requires `SetUp=1`, nonempty FEN, and exactly one solver. A record with no solver is Game even when it starts from FEN. A solver without complete setup markers, or two solvers, is Ambiguous.

Run: `go test ./internal/pgn -run TestDetect -v`

Expected: PASS.

- [ ] **Step 3: Commit detection rules**

```bash
git add internal/pgn/detector.go internal/pgn/detector_test.go
git commit -m "feat: classify PGN import modes"
```

### Task 3: Import solver-tagged tactical PGNs as canonical puzzles

**Files:**
- Create: `internal/puzzles/tactics_pgn_importer.go`
- Test: `internal/puzzles/tactics_pgn_importer_test.go`

**Interfaces:**
- Consumes: `pgn.Reader`, `pgn.Detect`, `Catalog.BeginImport`, chess rules, and fingerprinting.
- Produces: `TacticsPGNImporter.Import(ctx, sourceID, path, progress) (ImportReport, error)`.

- [ ] **Step 1: Write tests from the referenced collection's convention**

Use these records:

```pgn
[Event "0708 49919"]
[White "?"]
[Black "solver"]
[Result "*"]
[SetUp "1"]
[FEN "2kr4/1p1r1p2/2p1p2p/8/P1P1Rp2/8/1P3PPP/R5K1 w KQkq - 0 1"]

1. Rxf4 Rd1+ 2. Rxd1 Rxd1# *

[Event "0728 03099"]
[White "solver"]
[Black "?"]
[Result "*"]
[SetUp "1"]
[FEN "3r3k/6pp/3Q4/q7/8/4P2P/6P1/5RK1 b KQkq - 0 1"]

1... Rxd6 2. Rf8# *
```

Assert solver color, first opponent move stored as prelude, displayed FEN after prelude, remaining main line as a single solution branch, nil rating, empty themes, and source metadata containing Event.

Run: `go test ./internal/puzzles -run TestTacticsPGNImporter -v`

Expected: FAIL because `TacticsPGNImporter` is undefined.

- [ ] **Step 2: Implement tactical conversion**

For each record:

- Require `Detect(...).Kind == Tactical`; reject Game or Ambiguous records with reason.
- Derive solver from the player tag.
- If starting FEN active color differs from solver, consume exactly one main-line move as prelude.
- After prelude, require active color equals solver.
- Convert all remaining main-line moves into a linear UCI tree.
- Require at least one solver move and legal full line.
- Keep Event, source file ordinal, original headers, and any castling-rights sanitization warning in source metadata; do not store the large original record in `puzzles.sqlite`.
- Tee source bytes through SHA-256 and call `StagedImport.SetChecksum` before commit.
- Fingerprint and stage the puzzle.
- Continue after `RecordError` and emit progress once per 1,000 records.

- [ ] **Step 3: Test invalid and unrated behavior**

Test illegal FEN, missing learner move, two solvers, and an ordinary historical game. Assert each becomes a rejection and does not abort valid records. Persist imported puzzles and assert `RatedCandidates` excludes them while `FreePracticeCandidates(sourceID, nil, nil, nil, limit)` includes them.

Run: `go test ./internal/puzzles -run 'TestTacticsPGNImporter' -v`

Expected: PASS.

- [ ] **Step 4: Commit tactical PGN support**

```bash
git add internal/puzzles/tactics_pgn_importer.go internal/puzzles/tactics_pgn_importer_test.go
git commit -m "feat: import tactical PGN puzzles"
```

### Task 4: Add indexed game storage and ordinary PGN batch import

**Files:**
- Create: `internal/storage/migrations/library/002.sql`
- Create: `internal/games/repository.go`
- Create: `internal/games/sqlite_repository.go`
- Test: `internal/games/sqlite_repository_test.go`
- Create: `internal/games/importer.go`
- Test: `internal/games/importer_test.go`

**Interfaces:**
- Produces: `games.Repository.Upsert`, `Get`, `Search`, and `Delete`.
- Produces: `games.Importer.Import(ctx, path, progress) (GameImportReport, error)`.

Define `GameImportReport` with integer `Accepted`, `Duplicates`, and `Rejected` fields plus at most 100 `pgn.RecordError` examples, matching the puzzle-import report behavior.

- [ ] **Step 1: Create the library schema**

Create `internal/storage/migrations/library/002.sql`:

```sql
CREATE TABLE games (
  fingerprint TEXT PRIMARY KEY,
  original_pgn TEXT NOT NULL,
  starting_fen TEXT NOT NULL,
  tree_json TEXT NOT NULL,
  event TEXT NOT NULL DEFAULT '',
  site TEXT NOT NULL DEFAULT '',
  played_date TEXT NOT NULL DEFAULT '',
  played_year INTEGER,
  round TEXT NOT NULL DEFAULT '',
  white TEXT NOT NULL DEFAULT '',
  black TEXT NOT NULL DEFAULT '',
  result TEXT NOT NULL DEFAULT '',
  opening TEXT NOT NULL DEFAULT '',
  imported_at INTEGER NOT NULL
);
CREATE INDEX idx_games_white ON games(white COLLATE NOCASE);
CREATE INDEX idx_games_black ON games(black COLLATE NOCASE);
CREATE INDEX idx_games_event ON games(event COLLATE NOCASE);
CREATE INDEX idx_games_year ON games(played_year);
```

Add a migration test asserting an existing `library.sqlite` at version 1 upgrades without losing `library_metadata`.

- [ ] **Step 2: Define repository search types and tests**

Create `internal/games/repository.go` with:

```go
type SearchFilter struct {
	Text string `json:"text"`
	Player string `json:"player"`
	Event string `json:"event"`
	YearFrom *int `json:"yearFrom,omitempty"`
	YearTo *int `json:"yearTo,omitempty"`
	Result string `json:"result"`
	Limit int `json:"limit"`
	Offset int `json:"offset"`
}

type Summary struct {
	Fingerprint string `json:"fingerprint"`
	White string `json:"white"`
	Black string `json:"black"`
	Event string `json:"event"`
	Year *int `json:"year,omitempty"`
	Result string `json:"result"`
}
```

Seed four famous games and assert case-insensitive player, event, year range, result, pagination, and exact fingerprint lookup.

- [ ] **Step 3: Implement the SQLite repository**

Use prepared statements and explicit allowlisted filter clauses; never concatenate user values into SQL. Encode the move tree as JSON and retain original PGN. `Upsert` uses `INSERT OR IGNORE` and returns `inserted bool` so exact duplicates can be counted.

Run: `go test ./internal/games -run TestSQLiteRepository -v`

Expected: PASS.

- [ ] **Step 4: Write and implement ordinary batch import**

Test one duplicate, two valid ordinary games, one malformed record, and one tactical record. Ordinary importer must insert the two games, count the duplicate, reject malformed and tactical records with reasons, and keep going. Use one transaction per 500 games and roll back the current batch on fatal DB failure.

Run: `go test ./internal/games -run TestImporter -v`

Expected: PASS after creating `internal/games/importer.go`.

- [ ] **Step 5: Commit game persistence**

```bash
git add internal/storage/migrations/library/002.sql internal/storage/sqlite_test.go internal/games
git commit -m "feat: store and search PGN games"
```

### Task 5: Route PGN files through import jobs and ambiguity confirmation

**Files:**
- Create: `internal/importjob/pgn_service.go`
- Test: `internal/importjob/pgn_service_test.go`
- Modify: `internal/app/services.go`
- Modify: `app.go`
- Create: `frontend/src/components/import/PgnImportChoice.svelte`
- Test: `frontend/src/components/import/PgnImportChoice.test.ts`

**Interfaces:**
- Produces: `InspectPGN(path) (PGNInspection, error)` and `StartPGNImport(path, confirmedKind) (jobID, error)`.
- Reuses: `import:progress` and `import:finished` events with content kind included.

- [ ] **Step 1: Write inspection tests**

Inspect at most the first 20 parseable records. Return unanimous Tactical or Game only when every inspected record agrees; otherwise return Ambiguous with counts and reasons. Empty or fully malformed files return an error.

- [ ] **Step 2: Implement inspection and confirmed routing**

Define:

```go
type PGNInspection struct {
	Suggested pgn.Kind `json:"suggested"`
	Tactical int `json:"tactical"`
	Games int `json:"games"`
	Ambiguous int `json:"ambiguous"`
	Malformed int `json:"malformed"`
}
```

When suggested kind is Ambiguous, reject an empty confirmation. Route Tactical to `TacticsPGNImporter` and Game to `games.Importer`. A user confirmation applies to ambiguous records only; clearly opposite records are rejected rather than silently coerced.

- [ ] **Step 3: Add Wails bindings and generated types**

Expose `InspectPGN`, `StartPGNImport`, and the existing cancel/result methods. Regenerate bindings and verify backend tests.

- [ ] **Step 4: Test and implement the ambiguity prompt**

The component shows inspected counts, recommends the unanimous mode, and for Ambiguous requires an explicit `Import as puzzles` or `Import as games` click. It displays the same accepted/duplicate/rejected report as Lichess imports.

Run: `cd frontend && npm test -- --run src/components/import/PgnImportChoice.test.ts`

Expected: PASS.

- [ ] **Step 5: Commit PGN job routing**

```bash
git add app.go internal/app internal/importjob frontend/src/components/import frontend/wailsjs
git commit -m "feat: route PGN imports safely"
```

### Task 6: Build game-viewer navigation services

**Files:**
- Create: `internal/games/viewer.go`
- Test: `internal/games/viewer_test.go`
- Modify: `app.go`

**Interfaces:**
- Produces: `SearchGames`, `OpenGame`, `SelectGameNode`, and `ExportOriginalPGN` bindings.
- Produces: `GameView{Summary, FEN, CurrentNodeID, MainLine, Variations, Comments, CanPrevious, CanNext}`.

- [ ] **Step 1: Write navigation tests**

Open the annotated fixture from Task 1. Assert start position, next/previous, first/final, selecting variation node `m.3.v1.1`, correct FEN/comment/NAG at that node, board flip as presentation-only state, and exact original PGN export.

Run: `go test ./internal/games -run TestViewer -v`

Expected: FAIL because `Viewer` is undefined.

- [ ] **Step 2: Implement stateless navigation**

`Viewer.Open(fingerprint, nodeID)` loads one immutable game and resolves the requested node path. Do not keep per-window cursor state in Go; the frontend owns the current node ID and each selection returns a complete view. Define deterministic helpers for main-line previous/next and variation entry/return.

- [ ] **Step 3: Add bindings and search latency test**

Expose typed methods through `app.go`. Seed 100,000 compact game rows in a performance test and assert an indexed player/year search with limit 50 completes in under 250 ms on the target Mac.

Run: `go test ./internal/games -v`

Expected: PASS.

- [ ] **Step 4: Commit viewer services**

```bash
git add app.go internal/games/viewer.go internal/games/viewer_test.go frontend/wailsjs
git commit -m "feat: navigate historical games"
```

### Task 7: Implement the approved three-pane game library

**Files:**
- Create: `frontend/src/components/games/GameLibrary.svelte`
- Test: `frontend/src/components/games/GameLibrary.test.ts`
- Create: `frontend/src/components/games/MoveTree.svelte`
- Test: `frontend/src/components/games/MoveTree.test.ts`
- Modify: `frontend/src/App.svelte`
- Modify: `frontend/src/lib/api.ts`
- Modify: `frontend/tests/trainer.spec.ts`

**Interfaces:**
- Consumes: game search/view/export bindings and the existing `ChessBoard`.
- Produces: the approved all-in-one list/board/movetext workspace.

- [ ] **Step 1: Write interaction tests**

Test search debounce, player/event/year/result filters, row selection, previous/next/first/final controls, left/right keys, notation click, variation expansion/selection, comments/NAG display, board flip, PGN export, empty state, and malformed-import report link.

Run: `cd frontend && npm test -- --run src/components/games`

Expected: FAIL because components do not exist.

- [ ] **Step 2: Implement recursive notation**

`MoveTree.svelte` receives immutable nodes and `currentNodeID`, emits `select`, renders SAN in semantic buttons, shows NAG symbols with accessible labels, and nests variations in parentheses. It must not parse chess rules or mutate the tree.

- [ ] **Step 3: Implement the resizable three-pane workspace**

Use CSS grid with two keyboard-accessible separators. Persist pane percentages in `localStorage` under `game-library-pane-sizes`; clamp each pane to at least 18%. Collapse the game list into a drawer below 900px. Reuse the custom board in read-only mode.

- [ ] **Step 4: Extend Playwright coverage**

Add ordinary PGN import, search, game selection, arrow-key navigation, variation selection, board flip, and exact PGN export download. Assert there is no evaluation bar, autoplay control, or lesson-question action.

- [ ] **Step 5: Run full verification**

Run:

```bash
go test -race ./...
cd frontend
npm test -- --run
npm run test:e2e
npm run build
cd ..
wails build -clean
```

Expected: PASS.

- [ ] **Step 6: Commit PGN and game-library completion**

```bash
git add frontend/src/components/games frontend/src/App.svelte frontend/src/lib/api.ts frontend/tests/trainer.spec.ts
git commit -m "feat: add historical game library"
```

## Plan 2 Completion Gate

Verify the referenced 100k tactical PGN file with an opt-in integration test:

```bash
CHESS_TRAINER_TACTICS_PGN_PATH=/path/to/tactics.pgn go test ./internal/puzzles -run TestFullTacticsPGNImport -count=1 -v
go test -race ./...
npm --prefix frontend test -- --run
npm --prefix frontend run test:e2e
npm --prefix frontend run build
wails build -clean
```

Expected: valid tactical records import into Free Practice, ordinary PGNs preserve annotations in the three-pane viewer, malformed records are reported without aborting batches, all checks PASS, and no engine or FEN collection has been added.

## Primary References

- [Referenced tactical PGN collection](https://github.com/xinyangz/chess-tactics-pgn)
- [corentings/chess PGN scanning and variation APIs](https://pkg.go.dev/github.com/corentings/chess/v2)
- [Wails generated TypeScript bindings](https://wails.io/docs/howdoesitwork/)
