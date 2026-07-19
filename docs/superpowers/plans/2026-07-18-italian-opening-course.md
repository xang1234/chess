# Italian Opening Course Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a private, external-course-pack workflow that teaches a full-reference White-side Italian Game course through guided lessons, depth filters, spaced review, and a variation explorer without changing tactical puzzle semantics.

**Architecture:** A strict `.ctcourse` decoder compiles a position graph through the existing Go chess-rules adapter, then atomically activates it in a replaceable `courses.sqlite` catalogue. A separate opening service persists lessons and reviews in `user.sqlite`; Wails exposes opening-specific views to Svelte screens that reuse Chessground, authoritative move frames, animation, hints, sounds, and the app shell.

**Tech Stack:** Go module language 1.25 with release toolchain Go 1.26.4, Wails v2.12.0, `github.com/corentings/chess/v2` v2.5.1, `modernc.org/sqlite` v1.53.0, Svelte 3, TypeScript 4.6, Vite 3, Chessground 10.1.1, Vitest, Testing Library, and Playwright.

**Design source:** `docs/superpowers/specs/2026-07-18-italian-opening-course-design.md`

## Repository Reconciliation (2026-07-19)

The implementation starts from commit `26dc685`, which added the multi-format
puzzle inspection pipeline after this plan was approved. Preserve that newer
behavior while executing the intent of Tasks 1, 6, and 7:

- Task 1 moves the current shared `Format`, `Inspection`, `SourceIDOrigin`,
  phased `Progress`, `Rejection`, and `Report` values into `internal/importing`;
  puzzle names remain aliases, and `Report` gains the course `Counts` map.
- Task 6 composes one format router behind the existing single import-job writer
  gate. The puzzle collection importer keeps all five adapters, while the course
  importer adds `coursepack` inspection and import support.
- Task 7 parameterizes the current inspect-before-start import session for
  puzzle and course configurations. It must retain format labels, replacement
  warnings, phased progress, terminal precedence, cancellation, and strict
  decoding introduced by the newer puzzle-import work.

These are compatibility updates to implementation mechanics; the approved
course-pack, learning, privacy, and UX requirements are unchanged.

## Global Constraints

- At execution time, use `superpowers:using-git-worktrees` and create an isolated `codex/italian-opening-course` branch; do not implement directly on `main`.
- Use test-first RED, GREEN, refactor cycles. Run the focused failing test before production edits and commit after every task.
- Keep normal runtime fully offline. Add no HTTP listener, telemetry, online course fetch, OCR, or chess engine.
- Keep puzzle `SessionMode`, solution trees, rating, attempts, and review behavior unchanged.
- Treat every FEN, SAN string, legal-move list, and applied-move frame returned by Go as authoritative; TypeScript must not recreate chess rules.
- Store replaceable opening content in `courses.sqlite`; store preferences, sessions, mastery, attempts, and reviews in `user.sqlite`.
- Version-one `.ctcourse` is one UTF-8 JSON file with `schemaVersion: 1`; it is not ZIP and has no external assets.
- Treat all course titles, instructions, and notes as plain text. Render them with normal Svelte text interpolation; never use `{@html}` for imported content.
- Quick is a subset of Standard, and Standard is a subset of Reference. One graph supplies all three views.
- The pilot perspective is White. Every recall prompt has one primary repertoire move; alternatives are neutral, playable choices and are not tactical errors.
- The MCO-derived course pack stays outside the repository, corresponding-source archive, and release bundle. Repository fixtures contain only synthetic, original text.
- Preserve old course generations while an active or paused opening session references them. Rebase by stable IDs and semantic fingerprints before cleanup.
- Generate Wails bindings with `/Users/admin/go/bin/wails generate module -tags bindings`; do not hand-maintain generated bindings.
- Preserve the existing GPL-3.0-or-later release, legal-asset, source-archive, and Go 1.26.4 verification rules.
- At every backend checkpoint run the focused package and `go test ./... -count=1`. At frontend checkpoints run focused Vitest, the complete single-thread suite, and `npm --prefix frontend run check`.

## File Structure

### Import and storage foundation

- Create `internal/importing/types.go` for domain-neutral import progress, reports, counts, and rejection examples.
- Modify `internal/puzzles/{catalog,lichess_importer}.go` and `internal/importjob/service.go` to consume aliases from `internal/importing` without changing their JSON contracts.
- Modify `internal/storage/paths.go` and `internal/storage/sqlite.go`; create `internal/storage/migrations/courses/001.sql` for the replaceable course catalogue.
- Create `internal/storage/replaceable.go` for course-database quarantine and recreation.

### Course domain and persistence

- Create `internal/openings/types.go` for depth, perspective, role, evaluation, source reference, graph, lesson, prompt, and compiled-course types.
- Create `internal/openings/coursepack.go` for strict, size-bounded JSON decoding.
- Create `internal/openings/compiler.go`, `fingerprint.go`, and `coverage.go` for graph validation, FEN/SAN derivation, transpositions, tiers, prompts, and coverage diagnostics.
- Create `internal/openings/catalog.go` for atomic generation replacement, active reads, protected cleanup, and explorer queries.
- Create `internal/openings/importer.go` and `cmd/coursepack/main.go` for app import and developer validation.
- Create `internal/openings/testdata/mini.ctcourse` as an original synthetic fixture.

### Learner state and service

- Create `internal/storage/migrations/user/004.sql` for opening preferences, sessions, lesson progress, attempts, and reviews.
- Create `internal/spacedreview/schedule.go`; modify `internal/training/review.go` to delegate interval arithmetic without changing puzzle behavior.
- Create `internal/openings/user_store.go`, `views.go`, `service.go`, `service_steps.go`, and `service_rebase.go`.
- Move the generic `domain.AppliedMove` type from `internal/domain/training.go` to `internal/domain/chess.go` without changing its JSON shape.

### Application and frontend

- Modify `internal/app/services.go`, `normal_controller.go`, `controller_actions.go`, and generated Wails bindings for course imports and opening lesson APIs.
- Generalize `frontend/src/lib/import-session.ts`; extend `ImportPanel.svelte` and `ParentDashboard.svelte` with a separate course-pack import surface.
- Extend `frontend/src/lib/{api-contract,api,navigation}.ts` and `frontend/src/test-fakes.ts` with strict opening views and decoders.
- Create `frontend/src/components/openings/OpeningHub.svelte`, `OpeningLessonScreen.svelte`, `opening-controller.ts`, `opening-state.ts`, and `VariationExplorer.svelte` with focused tests.
- Move generic animation/effect helpers from `frontend/src/components/puzzle/` to `frontend/src/components/chess/` and keep puzzle imports working.
- Modify `HomeHub.svelte`, `NormalShell.svelte`, and `App.test.ts` for the new navigation and resumable opening state.
- Create `frontend/tests/openings.spec.ts`; update `frontend/tests/test-backend.ts` and operational documentation.

### Private content outside the repository

- Create `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse` only after the validator and learner are complete.
- Never stage or commit that file. Validate it directly from its external path.

---

### Task 1: Extract Domain-Neutral Import Contracts

**Files:**

- Create: `internal/importing/types.go`
- Test: `internal/importing/types_test.go`
- Modify: `internal/puzzles/catalog.go`
- Modify: `internal/puzzles/lichess_importer.go`
- Modify: `internal/importjob/service.go`
- Modify: `internal/importjob/service_test.go`

**Interfaces:**

- Produces: `importing.Progress`, `ProgressSink`, `Rejection`, and `Report`.
- Preserves: aliases `puzzles.Progress`, `puzzles.ProgressSink`, `puzzles.Rejection`, and `puzzles.ImportReport` so existing puzzle callers compile unchanged.
- Adds: optional `Report.Counts map[string]int64` for course-specific counts while preserving `accepted`, `duplicates`, `rejected`, and `examples` JSON fields.

- [ ] **Step 1: Write the failing shared-contract JSON test**

Create `internal/importing/types_test.go`:

```go
package importing

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestReportKeepsLegacyFieldsAndOptionalCounts(t *testing.T) {
	encoded, err := json.Marshal(Report{
		Accepted: 2,
		Rejected: 1,
		Counts: map[string]int64{"chapters": 3, "moves": 40},
	})
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, fragment := range []string{
		`"accepted":2`, `"duplicates":0`, `"rejected":1`,
		`"counts":{"chapters":3,"moves":40}`,
	} {
		if !strings.Contains(text, fragment) {
			t.Fatalf("JSON %s is missing %s", text, fragment)
		}
	}
}
```

- [ ] **Step 2: Run the test and verify RED**

Run: `go test ./internal/importing -run TestReportKeepsLegacyFieldsAndOptionalCounts -v`

Expected: FAIL because package `internal/importing` and `Report` do not exist.

- [ ] **Step 3: Implement the shared values and puzzle aliases**

Create `internal/importing/types.go`:

```go
package importing

type Progress struct {
	RowsRead  int64 `json:"rowsRead"`
	BytesRead int64 `json:"bytesRead"`
}

type ProgressSink func(Progress)

type Rejection struct {
	Ordinal int64  `json:"ordinal"`
	Reason  string `json:"reason"`
}

type Report struct {
	Accepted   int64            `json:"accepted"`
	Duplicates int64            `json:"duplicates"`
	Rejected   int64            `json:"rejected"`
	Examples   []Rejection      `json:"examples"`
	Counts     map[string]int64 `json:"counts,omitempty"`
}
```

Replace the four existing puzzle type declarations with aliases:

```go
type Progress = importing.Progress
type ProgressSink = importing.ProgressSink
type Rejection = importing.Rejection
type ImportReport = importing.Report
```

Change `internal/importjob` to import `internal/importing` directly in its interfaces, state, and emitter. Keep the puzzle aliases so every current puzzle test and public JSON shape remains valid.

Update result clone tests so mutating a returned `Counts` map cannot mutate the stored job result or a later event; clone `Counts` just as defensively as `Examples`.

- [ ] **Step 4: Run focused and complete backend tests**

Run:

```bash
go test ./internal/importing ./internal/importjob ./internal/puzzles -count=1
go test ./... -count=1
```

Expected: PASS; existing puzzle import tests need no behavior changes.

- [ ] **Step 5: Commit the import boundary**

```bash
git add internal/importing internal/importjob internal/puzzles/catalog.go internal/puzzles/lichess_importer.go
git diff --cached --check
git commit -m "refactor: share import job contracts"
```

### Task 2: Add the Replaceable Course Store

**Files:**

- Modify: `internal/storage/paths.go`
- Modify: `internal/storage/paths_test.go`
- Modify: `internal/storage/sqlite.go`
- Modify: `internal/storage/sqlite_test.go`
- Create: `internal/storage/migrations/courses/001.sql`
- Create: `internal/storage/replaceable.go`
- Test: `internal/storage/replaceable_test.go`
- Modify: `internal/backup/service.go`
- Modify: `internal/backup/service_test.go`

**Interfaces:**

- Produces: `storage.Paths.CoursesDB == <root>/courses.sqlite`.
- Produces: `storage.OpenReplaceable(path, "courses", now) (*sql.DB, QuarantineNotice, error)`.
- Preserves: backups contain `user.sqlite` and optional `library.sqlite`, never `courses.sqlite`; backup destinations may not overwrite `courses.sqlite`.

- [ ] **Step 1: Write failing path, migration, and quarantine tests**

Add these assertions to the existing storage tests:

```go
if p.CoursesDB != filepath.Join("/tmp/support", "courses.sqlite") {
	t.Fatalf("CoursesDB=%q", p.CoursesDB)
}
```

Add `{schema: "courses", table: "course_generations", migrations: 1}` to `TestMigrateCreatesEachSchemaAndIsIdempotent`.

Create `internal/storage/replaceable_test.go` with a test that writes non-SQLite bytes to `courses.sqlite`, calls `OpenReplaceable` with a fixed UTC time, and asserts:

```go
if notice.QuarantinedPath != path+".quarantine-20260718T050000Z" {
	t.Fatalf("quarantine path = %q", notice.QuarantinedPath)
}
if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&migrations); err != nil {
	t.Fatal(err)
}
if migrations != 1 {
	t.Fatalf("migrations = %d, want 1", migrations)
}
```

- [ ] **Step 2: Run the focused tests and verify RED**

Run: `go test ./internal/storage -run 'TestPathsAt|TestMigrateCreatesEachSchema|TestOpenReplaceable' -v`

Expected: FAIL because `CoursesDB`, the `courses` schema, and `OpenReplaceable` do not exist.

- [ ] **Step 3: Add the course schema**

Add `CoursesDB` to `storage.Paths`, initialize it in `PathsAt`, and admit `courses` in `migrationNames`.

Create `internal/storage/migrations/courses/001.sql` with these exact tables and constraints:

```sql
CREATE TABLE course_generations (
  generation_id TEXT PRIMARY KEY,
  course_id TEXT NOT NULL,
  status TEXT NOT NULL CHECK(status IN ('building','sealed','abandoned')),
  source_path TEXT NOT NULL,
  checksum TEXT,
  schema_version INTEGER NOT NULL,
  content_version TEXT NOT NULL,
  started_at INTEGER NOT NULL CHECK(started_at > 0),
  sealed_at INTEGER,
  UNIQUE(course_id, generation_id)
);
CREATE TABLE course_heads (
  course_id TEXT PRIMARY KEY,
  generation_id TEXT NOT NULL,
  FOREIGN KEY(course_id, generation_id)
    REFERENCES course_generations(course_id, generation_id)
);
CREATE TABLE courses (
  generation_id TEXT PRIMARY KEY REFERENCES course_generations(generation_id) ON DELETE CASCADE,
  course_id TEXT NOT NULL,
  title TEXT NOT NULL,
  description TEXT NOT NULL,
  perspective TEXT NOT NULL CHECK(perspective IN ('white','black')),
  default_depth TEXT NOT NULL CHECK(default_depth IN ('quick','standard','reference')),
  root_position_id TEXT NOT NULL,
  source_json TEXT NOT NULL,
  coverage_json TEXT NOT NULL
);
CREATE TABLE course_chapters (
  generation_id TEXT NOT NULL REFERENCES course_generations(generation_id) ON DELETE CASCADE,
  chapter_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL CHECK(ordinal > 0),
  title TEXT NOT NULL,
  overview TEXT NOT NULL,
  minimum_depth TEXT NOT NULL CHECK(minimum_depth IN ('quick','standard','reference')),
  PRIMARY KEY(generation_id, chapter_id),
  UNIQUE(generation_id, ordinal)
);
CREATE TABLE course_positions (
  generation_id TEXT NOT NULL REFERENCES course_generations(generation_id) ON DELETE CASCADE,
  position_id TEXT NOT NULL,
  fen TEXT NOT NULL,
  label TEXT NOT NULL,
  evaluation_json TEXT NOT NULL,
  note_ids_json TEXT NOT NULL,
  PRIMARY KEY(generation_id, position_id)
);
CREATE TABLE course_moves (
  generation_id TEXT NOT NULL REFERENCES course_generations(generation_id) ON DELETE CASCADE,
  move_id TEXT NOT NULL,
  from_position_id TEXT NOT NULL,
  to_position_id TEXT NOT NULL,
  uci TEXT NOT NULL,
  san TEXT NOT NULL,
  minimum_depth TEXT NOT NULL CHECK(minimum_depth IN ('quick','standard','reference')),
  training_role TEXT NOT NULL CHECK(training_role IN ('repertoire','opponent','alternative')),
  variation_name TEXT NOT NULL,
  evaluation_json TEXT NOT NULL,
  note_ids_json TEXT NOT NULL,
  source_ref_json TEXT NOT NULL,
  PRIMARY KEY(generation_id, move_id),
  FOREIGN KEY(generation_id, from_position_id)
    REFERENCES course_positions(generation_id, position_id),
  FOREIGN KEY(generation_id, to_position_id)
    REFERENCES course_positions(generation_id, position_id)
);
CREATE TABLE course_notes (
  generation_id TEXT NOT NULL REFERENCES course_generations(generation_id) ON DELETE CASCADE,
  note_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  text TEXT NOT NULL,
  source_ref_json TEXT NOT NULL,
  PRIMARY KEY(generation_id, note_id)
);
CREATE TABLE course_lessons (
  generation_id TEXT NOT NULL REFERENCES course_generations(generation_id) ON DELETE CASCADE,
  lesson_id TEXT NOT NULL,
  chapter_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL CHECK(ordinal > 0),
  title TEXT NOT NULL,
  objectives_json TEXT NOT NULL,
  minimum_depth TEXT NOT NULL CHECK(minimum_depth IN ('quick','standard','reference')),
  start_position_id TEXT NOT NULL,
  PRIMARY KEY(generation_id, lesson_id),
  FOREIGN KEY(generation_id, chapter_id)
    REFERENCES course_chapters(generation_id, chapter_id),
  FOREIGN KEY(generation_id, start_position_id)
    REFERENCES course_positions(generation_id, position_id)
);
CREATE TABLE course_prompts (
  generation_id TEXT NOT NULL REFERENCES course_generations(generation_id) ON DELETE CASCADE,
  prompt_id TEXT NOT NULL,
  position_id TEXT NOT NULL,
  primary_move_id TEXT NOT NULL,
  accepted_move_ids_json TEXT NOT NULL,
  semantic_fingerprint TEXT NOT NULL,
  PRIMARY KEY(generation_id, prompt_id),
  FOREIGN KEY(generation_id, position_id)
    REFERENCES course_positions(generation_id, position_id),
  FOREIGN KEY(generation_id, primary_move_id)
    REFERENCES course_moves(generation_id, move_id)
);
CREATE TABLE course_lesson_steps (
  generation_id TEXT NOT NULL REFERENCES course_generations(generation_id) ON DELETE CASCADE,
  lesson_id TEXT NOT NULL,
  ordinal INTEGER NOT NULL CHECK(ordinal > 0),
  step_id TEXT NOT NULL,
  kind TEXT NOT NULL CHECK(kind IN ('explain','watch','try','branch','recall')),
  position_id TEXT NOT NULL,
  data_json TEXT NOT NULL,
  PRIMARY KEY(generation_id, lesson_id, step_id),
  UNIQUE(generation_id, lesson_id, ordinal),
  FOREIGN KEY(generation_id, lesson_id)
    REFERENCES course_lessons(generation_id, lesson_id),
  FOREIGN KEY(generation_id, position_id)
    REFERENCES course_positions(generation_id, position_id)
);
CREATE INDEX idx_course_generations_cleanup ON course_generations(status, generation_id);
CREATE INDEX idx_course_moves_from ON course_moves(generation_id, from_position_id, minimum_depth);
CREATE INDEX idx_course_lessons_chapter ON course_lessons(generation_id, chapter_id, ordinal);
```

- [ ] **Step 4: Implement replaceable-store quarantine**

If the database exists, `OpenReplaceable` first opens it read-only and requires `PRAGMA quick_check` to return exactly `ok`, then calls `PreflightMigrations`. On any integrity, preflight, open, or migration error, close handles, rename the database plus existing `-wal` and `-shm` siblings to the same timestamped quarantine prefix, and create a fresh migrated database. Return a notice containing the original error and quarantine path. If quarantine or recreation fails, return the error so application composition can disable openings without entering global recovery.

Use this public shape:

```go
type QuarantineNotice struct {
	QuarantinedPath string
	Detail          string
}

func OpenReplaceable(
	path string,
	schema string,
	now func() time.Time,
) (*sql.DB, QuarantineNotice, error)
```

- [ ] **Step 5: Keep course content out of backups**

Add `paths.CoursesDB` to the managed-destination rejection list in `backup.Service.validateDestination`. Do not add it to `databaseNames`, backup manifests, or restore replacement. Add a test proving a backup destination equal to `courses.sqlite` fails while a normal user backup still contains no course database.

- [ ] **Step 6: Run tests and commit**

```bash
go test ./internal/storage ./internal/backup -count=1
go test ./... -count=1
git add internal/storage internal/backup
git diff --cached --check
git commit -m "feat: add replaceable course store"
```

### Task 3: Define and Strictly Decode Course Packs

**Files:**

- Create: `internal/openings/types.go`
- Create: `internal/openings/coursepack.go`
- Test: `internal/openings/coursepack_test.go`
- Create: `internal/openings/testdata/mini.ctcourse`

**Interfaces:**

- Produces: `openings.Depth`, `Perspective`, `TrainingRole`, `StepKind`, `Evaluation`, `SourceRef`, `CoursePack`, and all nested source types.
- Produces: `DecodeCoursePack(io.Reader) (CoursePack, error)` with unknown-field and trailing-document rejection.
- Enforces: schema version 1, maximum file size 32 MiB at importer/CLI call sites, not inside the pure decoder.

- [ ] **Step 1: Write failing strict-decoder tests**

Create table-driven tests for a valid synthetic pack, an unknown field, a second JSON document, invalid UTF-8, an unsupported schema version, and an empty course ID. The key assertions are:

```go
pack, err := DecodeCoursePack(bytes.NewReader(valid))
if err != nil {
	t.Fatal(err)
}
if pack.CourseID != "synthetic-italian" || pack.DefaultDepth != DepthReference {
	t.Fatalf("decoded pack = %+v", pack)
}

_, err = DecodeCoursePack(strings.NewReader(validWithUnknownField))
if err == nil || !strings.Contains(err.Error(), "unknown field") {
	t.Fatalf("unknown field error = %v", err)
}
```

- [ ] **Step 2: Run the decoder tests and verify RED**

Run: `go test ./internal/openings -run TestDecodeCoursePack -v`

Expected: FAIL because the package and decoder do not exist.

- [ ] **Step 3: Add exact source types**

Define enum values exactly as the schema and SQL checks use:

```go
type Depth string
const (
	DepthQuick Depth = "quick"
	DepthStandard Depth = "standard"
	DepthReference Depth = "reference"
)

type Perspective string
const (
	PerspectiveWhite Perspective = "white"
	PerspectiveBlack Perspective = "black"
)

type TrainingRole string
const (
	RoleRepertoire TrainingRole = "repertoire"
	RoleOpponent TrainingRole = "opponent"
	RoleAlternative TrainingRole = "alternative"
)

type StepKind string
const (
	StepExplain StepKind = "explain"
	StepWatch StepKind = "watch"
	StepTry StepKind = "try"
	StepBranch StepKind = "branch"
	StepRecall StepKind = "recall"
)
```

Define evaluation codes exactly as `none`, `equal`, `unclear`, `white_slight`, `black_slight`, `white_clear`, `black_clear`, `white_winning`, and `black_winning`:

```go
type EvaluationCode string
const (
	EvaluationNone         EvaluationCode = "none"
	EvaluationEqual        EvaluationCode = "equal"
	EvaluationUnclear      EvaluationCode = "unclear"
	EvaluationWhiteSlight  EvaluationCode = "white_slight"
	EvaluationBlackSlight  EvaluationCode = "black_slight"
	EvaluationWhiteClear   EvaluationCode = "white_clear"
	EvaluationBlackClear   EvaluationCode = "black_clear"
	EvaluationWhiteWinning EvaluationCode = "white_winning"
	EvaluationBlackWinning EvaluationCode = "black_winning"
)

type Evaluation struct {
	Code         EvaluationCode `json:"code"`
	SourceSymbol string         `json:"sourceSymbol,omitempty"`
}

type SourceRef struct {
	PrintedPage int    `json:"printedPage"`
	TableColumn string `json:"tableColumn,omitempty"`
	NoteLabel   string `json:"noteLabel,omitempty"`
	CoverageID  string `json:"coverageId"`
}

type SourceCoverage struct {
	PrintedPages       []int    `json:"printedPages"`
	ExpectedReferences []string `json:"expectedReferences"`
}

type CourseSource struct {
	Title            string `json:"title"`
	Edition          string `json:"edition"`
	PrivateUseNotice string `json:"privateUseNotice"`
}
```

Use these source collections and JSON fields:

```go
type Position struct {
	PositionID string     `json:"positionId"`
	Label      string     `json:"label,omitempty"`
	Evaluation Evaluation `json:"evaluation"`
	NoteIDs    []string   `json:"noteIds"`
}

type Move struct {
	MoveID         string       `json:"moveId"`
	FromPositionID string       `json:"fromPositionId"`
	ToPositionID   string       `json:"toPositionId"`
	UCI            string       `json:"uci"`
	MinimumDepth   Depth        `json:"minimumDepth"`
	TrainingRole   TrainingRole `json:"trainingRole"`
	VariationName  string       `json:"variationName,omitempty"`
	Evaluation     Evaluation   `json:"evaluation"`
	NoteIDs        []string     `json:"noteIds"`
	SourceRef      SourceRef    `json:"sourceRef"`
}

type Note struct {
	NoteID    string    `json:"noteId"`
	Kind      string    `json:"kind"`
	Text      string    `json:"text"`
	SourceRef SourceRef `json:"sourceRef"`
}

type Chapter struct {
	ChapterID    string `json:"chapterId"`
	Ordinal      int    `json:"ordinal"`
	Title        string `json:"title"`
	Overview     string `json:"overview"`
	MinimumDepth Depth  `json:"minimumDepth"`
}

type Lesson struct {
	LessonID        string       `json:"lessonId"`
	ChapterID       string       `json:"chapterId"`
	Ordinal         int          `json:"ordinal"`
	Title           string       `json:"title"`
	Objectives      []string     `json:"objectives"`
	MinimumDepth    Depth        `json:"minimumDepth"`
	StartPositionID string      `json:"startPositionId"`
	Steps           []LessonStep `json:"steps"`
}

type LessonStep struct {
	StepID      string   `json:"stepId"`
	Kind        StepKind `json:"kind"`
	PositionID  string   `json:"positionId"`
	Title       string   `json:"title"`
	Instruction string   `json:"instruction"`
	NoteIDs     []string `json:"noteIds"`
	MoveIDs     []string `json:"moveIds"`
	PromptID    string   `json:"promptId,omitempty"`
}

type Prompt struct {
	PromptID                   string   `json:"promptId"`
	PositionID                 string   `json:"positionId"`
	PrimaryMoveID              string   `json:"primaryMoveId"`
	AcceptedAlternativeMoveIDs []string `json:"acceptedAlternativeMoveIds"`
}

type CoursePack struct {
	SchemaVersion  int            `json:"schemaVersion"`
	CourseID       string         `json:"courseId"`
	ContentVersion string         `json:"contentVersion"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	Perspective    Perspective    `json:"perspective"`
	DefaultDepth   Depth          `json:"defaultDepth"`
	RootPositionID string         `json:"rootPositionId"`
	RootFEN        string         `json:"rootFen"`
	Source         CourseSource   `json:"source"`
	SourceCoverage SourceCoverage `json:"sourceCoverage"`
	Positions      []Position     `json:"positions"`
	Moves          []Move         `json:"moves"`
	Notes          []Note         `json:"notes"`
	Chapters       []Chapter      `json:"chapters"`
	Lessons        []Lesson       `json:"lessons"`
	Prompts        []Prompt       `json:"prompts"`
}
```

`CoursePack.Source` contains `title`, `edition`, and `privateUseNotice`. Note kinds are restricted during compile to `overview`, `history`, `plan`, `warning`, `explanation`, `evaluation`, `transposition`, and `illustrative_game`.

Use slices, not maps, in the source format so duplicate IDs can be diagnosed before indexing.

`SourceRef` contains positive `printedPage`, optional `tableColumn`, optional `noteLabel`, and required `coverageId`. `SourceCoverage` contains ordered `printedPages` and `expectedReferences`; expected references are stable coverage IDs such as `giuoco-p18-overview` and `giuoco-p19-column-a`, not automatically derived page strings. Several moves from one table column may intentionally share one coverage ID.

- [ ] **Step 4: Implement strict JSON decoding**

Use `json.Decoder.DisallowUnknownFields()`, decode exactly one root value, then require the next decode to return `io.EOF`. Tee all bytes read into a buffer and reject `!utf8.Valid(raw.Bytes())`; Go's JSON decoder alone replaces malformed UTF-8 and is therefore insufficient. Validate only root-level schema and required text in this function; graph validation belongs to Task 4.

```go
func DecodeCoursePack(reader io.Reader) (CoursePack, error) {
	var raw bytes.Buffer
	decoder := json.NewDecoder(io.TeeReader(reader, &raw))
	decoder.DisallowUnknownFields()
	var pack CoursePack
	if err := decoder.Decode(&pack); err != nil {
		return CoursePack{}, fmt.Errorf("decode course pack: %w", err)
	}
	if err := requireJSONEOF(decoder); err != nil {
		return CoursePack{}, err
	}
	if !utf8.Valid(raw.Bytes()) {
		return CoursePack{}, errors.New("course pack is not valid UTF-8")
	}
	if pack.SchemaVersion != 1 {
		return CoursePack{}, fmt.Errorf("unsupported course schema version %d", pack.SchemaVersion)
	}
	if strings.TrimSpace(pack.CourseID) == "" {
		return CoursePack{}, errors.New("courseId is required")
	}
	return pack, nil
}
```

- [ ] **Step 5: Add the original synthetic fixture**

Create `internal/openings/testdata/mini.ctcourse` with initial-position root FEN; a Quick path `1.e4 e5 2.Nf3 Nc6 3.Bc4 Bc5 4.c3`; a Standard opponent branch `3...Nf6 4.d3`; and a Reference-only alternative `4.b4`. Include one original overview note, three chapters, one lesson with all five step kinds, one recall prompt whose primary move is `c2c3`, source references on printed page 1, and an expected coverage list matching every source reference. Do not copy book prose or labels beyond public chess opening names.

- [ ] **Step 6: Run tests and commit**

```bash
go test ./internal/openings -run TestDecodeCoursePack -count=1
go test ./... -count=1
git add internal/openings/types.go internal/openings/coursepack.go internal/openings/coursepack_test.go internal/openings/testdata/mini.ctcourse
git diff --cached --check
git commit -m "feat: define opening course packs"
```

### Task 4: Compile and Validate the Position Graph

**Files:**

- Create: `internal/openings/compiler.go`
- Create: `internal/openings/fingerprint.go`
- Create: `internal/openings/coverage.go`
- Test: `internal/openings/compiler_test.go`
- Test: `internal/openings/coverage_test.go`

**Interfaces:**

- Consumes: `CoursePack` from Task 3 and the existing chess-rules methods.
- Produces: `Compile(CoursePack, RulesPort) (CompiledCourse, error)`.
- Produces: `ValidationError.Diagnostics []Diagnostic`, each with a stable code, JSON path, and message.
- Produces: derived FEN and SAN, cumulative depth indexes, semantic prompt fingerprints, and `CoverageReport`.

- [ ] **Step 1: Write failing compiler tests**

Define the rules port in the test against the real adapter:

```go
type RulesPort interface {
	ApplyUCI(string, string) (string, error)
	SAN(string, string) (string, error)
	LegalMoves(string) ([]string, error)
}
```

Add tests that compile `testdata/mini.ctcourse` and assert:

```go
compiled, err := Compile(pack, chessrules.Rules{})
if err != nil {
	t.Fatal(err)
}
if got := compiled.Moves["white-c3"].SAN; got != "c3" {
	t.Fatalf("SAN = %q, want c3", got)
}
if compiled.Prompts["recall-c3"].SemanticFingerprint == "" {
	t.Fatal("prompt fingerprint is empty")
}
if len(compiled.VisibleMoves(compiled.Positions["after-bc5"].ID, DepthQuick)) != 1 {
	t.Fatal("Quick depth did not filter reference alternatives")
}
```

Add mutation tests for duplicate IDs, illegal UCI, unreachable node, a cycle, a Standard parent followed by a Quick child, a prompt whose primary edge does not leave its position, two incoming paths deriving different canonical FENs, missing coverage, and unexpected coverage.

- [ ] **Step 2: Run the compiler tests and verify RED**

Run: `go test ./internal/openings -run 'TestCompile|TestCoverage' -v`

Expected: FAIL because `Compile`, `CompiledCourse`, and coverage validation do not exist.

- [ ] **Step 3: Define compiled values and validation diagnostics**

Add these stable public shapes:

```go
type Diagnostic struct {
	Code    string `json:"code"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

type ValidationError struct {
	Diagnostics []Diagnostic
}

type CompiledPosition struct {
	ID         string
	FEN        string
	Label      string
	Evaluation Evaluation
	NoteIDs    []string
}

type CompiledMove struct {
	Move
	SAN string
}

type CompiledPrompt struct {
	Prompt
	SemanticFingerprint string
}

type CoverageItem struct {
	CoverageID  string   `json:"coverageId"`
	PrintedPage int      `json:"printedPage"`
	TableColumn string   `json:"tableColumn,omitempty"`
	NoteLabel   string   `json:"noteLabel,omitempty"`
	RecordIDs   []string `json:"recordIds"`
}

type CoverageReport struct {
	Expected   []string       `json:"expected"`
	Captured   []CoverageItem `json:"captured"`
	Missing    []string       `json:"missing"`
	Unexpected []string       `json:"unexpected"`
}

type CompiledCourse struct {
	Pack       CoursePack
	Positions  map[string]CompiledPosition
	Moves      map[string]CompiledMove
	Notes      map[string]Note
	Chapters   map[string]Chapter
	Lessons    map[string]Lesson
	Prompts    map[string]CompiledPrompt
	Outgoing   map[string][]string
	Incoming   map[string][]string
	Coverage   CoverageReport
}

func (c CompiledCourse) VisibleMoves(positionID string, depth Depth) []CompiledMove
```

`ValidationError.Error()` sorts diagnostics by path, code, then message and joins them with newlines. This makes CLI and app failures deterministic.

- [ ] **Step 4: Implement identity, enum, and reference validation**

Accept IDs matching `^[a-z0-9][a-z0-9._-]{0,127}$`. Reject duplicate IDs within each collection, invalid enums, empty titles/instructions where required, text over 20,000 UTF-8 bytes, missing notes, chapters, positions, moves, lessons, prompts, and a prompt primary move whose `fromPositionId` differs from the prompt position.

At a position where the course perspective is to move, allow at most one outgoing `repertoire` edge across the full Reference graph. Require every prompt primary edge to be that repertoire edge. Accepted alternatives must be unique, leave the same position, differ from the primary, and have role `alternative`. Reject repertoire or alternative roles when the opponent is to move, and reject `opponent` when the learner is to move.

Require each lesson to contain all five teaching phases in nondecreasing explain, watch, try, branch, recall order. Repeated phases are allowed, but every lesson must begin with explain and end with recall. Require unique chapter ordinals, unique lesson ordinals within a chapter, and unique step IDs within a lesson.

For every recall, try, or branch step require a prompt and require the step position to equal the prompt position. For explain and watch steps reject a prompt. Explain steps have no moves. Watch steps require at least one move ID, start at the step position, and form one continuous edge path. A prompt step may name at most one automatic continuation edge; it must have role `opponent` and leave the primary move's destination. This gives the service an authoritative Black reply after a successful learner move without auto-playing a second White choice.

Require each lesson's minimum-depth rank to be at least its chapter's minimum-depth rank, and every lesson start/step position to be reachable at the lesson minimum. Every watch or prompt-primary move referenced by a lesson must also be visible at that minimum; accepted alternatives are filtered individually by their own edge depth.

- [ ] **Step 5: Derive the legal graph and transpositions**

Walk from `rootPositionId` using a color-marked DFS for cycle detection and a queue for FEN derivation. The root position gets `rootFen`. For every outgoing edge:

```go
san, err := rules.SAN(from.FEN, edge.UCI)
nextFEN, err := rules.ApplyUCI(from.FEN, edge.UCI)
canonical := CanonicalPosition(nextFEN)
```

`CanonicalPosition` splits FEN on ASCII spaces and joins exactly the first four fields. Reject a FEN with fewer than four fields. When a destination already has a derived FEN, compare canonical values and emit `inconsistent_transposition` with both incoming move IDs on mismatch.

Compute a depth rank Quick=0, Standard=1, Reference=2. For each of the three selected depths, walk only edges visible at that depth and reject any visible edge whose source is unreachable in that same walk. This accepts a position reached through both Quick and Reference move orders while still preventing a Quick child from depending only on a Standard parent. Sort every outgoing and incoming move list by source slice order for deterministic lessons and explorer display.

- [ ] **Step 6: Compute semantic fingerprints and coverage**

Fingerprint each prompt as lowercase SHA-256 hex of canonical JSON containing `promptId`, the canonical source position, primary move ID/UCI/destination, and sorted accepted alternative move IDs/UCI/destinations. Do not include prose, source page, or variation labels, so editorial corrections preserve mastery.

Validate that declared printed pages are positive and unique and that every source reference's printed page is declared and its coverage ID is non-empty. Compare the set of captured move/note coverage IDs with `sourceCoverage.expectedReferences`; report sorted missing and unexpected IDs. Group captured records into `CoverageItem` by coverage ID, require every record in a group to use the same page/column/note coordinates, sort its record IDs, and order items by printed page then table column, note label, and coverage ID. Repeated captured IDs are allowed because all moves in one table column share its coverage unit. Duplicate IDs inside the expected-reference list are an error. Any missing or unexpected reference is a compile error for a pack that declares expected references.

- [ ] **Step 7: Run tests and commit**

```bash
go test ./internal/openings -run 'TestCompile|TestCoverage' -count=1
go test ./... -count=1
git add internal/openings/compiler.go internal/openings/compiler_test.go internal/openings/fingerprint.go internal/openings/coverage.go internal/openings/coverage_test.go
git diff --cached --check
git commit -m "feat: validate opening course graphs"
```

### Task 5: Persist, Import, and Validate Course Packs

**Files:**

- Create: `internal/openings/catalog.go`
- Test: `internal/openings/catalog_test.go`
- Create: `internal/openings/importer.go`
- Test: `internal/openings/importer_test.go`
- Create: `cmd/coursepack/main.go`
- Test: `cmd/coursepack/main_test.go`

**Interfaces:**

- Consumes: compiled courses and the `courses.sqlite` schema.
- Produces: `NewSQLiteCatalog(*sql.DB) *SQLiteCatalog`.
- Produces: atomic `Replace`, active summary/load, generation load, explorer-position load, and protected cleanup methods.
- Produces: `Importer.Import(context.Context, string, string, importing.ProgressSink) (importing.Report, error)`.
- Produces: `go run ./cmd/coursepack validate <path>` with deterministic JSON output.

- [ ] **Step 1: Write the failing atomic replacement test**

Create a migrated temporary course database, compile the fixture, and assert this sequence:

```go
first, err := catalog.Replace(ctx, compiledV1, "/private/v1.ctcourse", "sha-v1")
if err != nil {
	t.Fatal(err)
}
second, err := catalog.Replace(ctx, compiledV2, "/private/v2.ctcourse", "sha-v2")
if err != nil {
	t.Fatal(err)
}
active, err := catalog.ActiveGenerationID(ctx, compiledV1.Pack.CourseID)
if err != nil || active != second.GenerationID {
	t.Fatalf("active = %q err=%v", active, err)
}
if _, err := catalog.LoadGeneration(ctx, first.GenerationID); err != nil {
	t.Fatalf("old generation disappeared before cleanup: %v", err)
}
```

Also test transaction rollback by injecting a context cancellation during row insertion and proving the previous head remains active.

- [ ] **Step 2: Run the catalogue test and verify RED**

Run: `go test ./internal/openings -run TestSQLiteCatalog -v`

Expected: FAIL because `SQLiteCatalog` and `Replace` do not exist.

- [ ] **Step 3: Implement atomic catalogue replacement and reads**

Use one SQLite transaction per `Replace`:

1. insert a UUID generation in `building` state;
2. insert course, chapter, position, move, note, lesson, prompt, and step rows in deterministic source order;
3. update the generation to `sealed` with checksum and timestamp;
4. upsert `course_heads`; and
5. commit.

Return:

```go
type ReplaceResult struct {
	CourseID     string
	GenerationID string
	PreviousHead string
}

type CourseSummary struct {
	CourseID       string
	GenerationID   string
	ContentVersion string
	Title          string
	RootPositionID string
	Perspective    Perspective
	DefaultDepth   Depth
}
```

Implement these exact reads:

```go
func (c *SQLiteCatalog) ListActive(context.Context) ([]CourseSummary, error)
func (c *SQLiteCatalog) ActiveGenerationID(context.Context, string) (string, error)
func (c *SQLiteCatalog) LoadActive(context.Context, string) (CompiledCourse, error)
func (c *SQLiteCatalog) LoadGeneration(context.Context, string) (CompiledCourse, error)
func (c *SQLiteCatalog) CleanupBatch(context.Context, map[string]struct{}, int) (bool, error)
```

`CleanupBatch` deletes only sealed generations that are neither a current head nor present in the protected-generation set. It processes at most the supplied positive limit.

- [ ] **Step 4: Write failing importer and size-boundary tests**

Test a successful fixture import, a 32 MiB plus one byte file, an invalid pack that leaves the old head unchanged, cancellation, and a report containing exact counts:

```go
want := map[string]int64{
	"chapters": 3,
	"positions": int64(len(compiled.Positions)),
	"moves": int64(len(compiled.Moves)),
	"variations": countDistinctVariationNames(compiled),
	"notes": int64(len(compiled.Notes)),
	"lessons": int64(len(compiled.Lessons)),
	"prompts": int64(len(compiled.Prompts)),
	"warnings": countNotesOfKind(compiled, "warning"),
}
if !reflect.DeepEqual(report.Counts, want) {
	t.Fatalf("counts = %#v, want %#v", report.Counts, want)
}
```

- [ ] **Step 5: Implement the importer**

Use `const MaxCoursePackBytes int64 = 32 << 20`. Open and stat the selected path and reject a known larger size before allocation. Also wrap the stream with a Max+1 counting limit so non-regular files and files that grow after `Stat` are rejected whenever the observed byte count exceeds the limit. Compute SHA-256 while decoding, compile, emit monotonic progress after decode and after each validated collection, then call `catalog.Replace`. Set `Accepted` to one activated course and put structural totals in `Counts`: chapters, positions, moves, distinct non-empty variation names, notes, lessons, prompts, and notes whose kind is `warning`. The request `sourceID` is only job metadata; the pack's validated `courseId` controls replacement.

- [ ] **Step 6: Add the validator CLI**

Support only this exact command form:

```text
coursepack validate <path>
```

On success write indented JSON with `courseId`, `contentVersion`, `counts`, and `coverage`. On validation failure write deterministic diagnostics to stderr and exit 1. On usage error exit 2. The CLI uses the same 32 MiB bound, decoder, compiler, and real `chessrules.Rules` as the app. Its counts use the same helper as the importer, including distinct variations and warning notes.

Run:

```bash
go run ./cmd/coursepack validate internal/openings/testdata/mini.ctcourse
```

Expected: exit 0; JSON contains `"courseId": "synthetic-italian"`, three chapters, zero missing coverage, and zero unexpected coverage.

- [ ] **Step 7: Run tests and commit**

```bash
go test ./internal/openings ./cmd/coursepack -count=1
go test ./... -count=1
git add internal/openings/catalog.go internal/openings/catalog_test.go internal/openings/importer.go internal/openings/importer_test.go cmd/coursepack
git diff --cached --check
git commit -m "feat: import opening course packs"
```

### Task 6: Compose Course Import in the Go Application

**Files:**

- Modify: `internal/importjob/service.go`
- Modify: `internal/importjob/service_test.go`
- Modify: `internal/app/services.go`
- Modify: `internal/app/services_test.go`
- Modify: `application_runtime_test.go`
- Modify: `normal_controller.go`
- Modify: `controller_actions.go`
- Modify: `controllers_test.go`
- Regenerate: `frontend/wailsjs/go/models.ts`
- Regenerate: `frontend/wailsjs/go/main/NormalController.js`
- Regenerate: `frontend/wailsjs/go/main/NormalController.d.ts`

**Interfaces:**

- Adds: `importjob.KindCourse == "course"`.
- Adds: `NormalController.ChooseOpeningCourseFile()` and `StartOpeningCourseImport(path)`.
- Adds to `app.Services`: `CoursesDB`, `OpeningCatalog`, `CourseImporter`, and `CourseNotice`.
- Keeps: one global import-job writer gate, so puzzle and course imports cannot mutate stores concurrently.

- [ ] **Step 1: Write failing service-composition tests**

Extend `TestOpenCreatesAndClosesAllStores` to require `paths.CoursesDB`, non-nil `CoursesDB`, `OpeningCatalog`, and `CourseImporter`. Close the services and assert `CoursesDB.Ping()` fails.

Add a corrupt-course test that writes invalid bytes to `courses.sqlite`, opens the application, and asserts normal mode still starts, `CourseNotice` names the quarantine path, and puzzles remain available.

Add a controller-dialog test that asserts the title is `Choose an opening course` and the only filter is `Opening course (*.ctcourse)`.

- [ ] **Step 2: Run focused tests and verify RED**

Run:

```bash
go test ./internal/app -run 'TestOpenCreates|TestCorruptCourse' -v
go test . -run 'Test.*OpeningCourse' -v
```

Expected: FAIL because course services and bindings are absent.

- [ ] **Step 3: Generalize import jobs for the course kind**

Add `KindCourse`. Change `importjob.Importer`, `Emitter`, result fields, and clone helpers to use `internal/importing` directly. Existing puzzle aliases from Task 1 keep puzzle implementations unchanged. Preserve the existing single active job, event ordering, cancel, close, and maintenance tests.

Add a small maintenance group in `internal/app/services.go`:

```go
type importMaintenance []importjob.Maintenance

func (group importMaintenance) CleanupBatch(ctx context.Context, limit int) (bool, error) {
	for _, maintenance := range group {
		more, err := maintenance.CleanupBatch(ctx, limit)
		if err != nil || more {
			return more, err
		}
	}
	return false, nil
}
```

The opening catalogue adapter initially protects no old generations; Task 10 replaces it with session-aware protection before multiple generations can be cleaned.

Use this explicit temporary adapter in Task 6:

```go
type unprotectedCourseMaintenance struct {
	catalog *openings.SQLiteCatalog
}

func (m unprotectedCourseMaintenance) CleanupBatch(
	ctx context.Context,
	limit int,
) (bool, error) {
	return m.catalog.CleanupBatch(ctx, map[string]struct{}{}, limit)
}
```

- [ ] **Step 4: Compose replaceable course storage and importer**

After durable user/puzzle/library stores open, call `storage.OpenReplaceable(paths.CoursesDB, "courses", time.Now)`. A quarantine notice populates `Services.CourseNotice`; an unrecoverable replaceable-store error leaves course fields nil and also populates the notice without returning global recovery.

When the course DB is available, create `openings.NewSQLiteCatalog`, an `openings.Importer` using `chessrules.Rules{}`, and register both `KindLichess` and `KindCourse` in the same `importjob.Service`. Close `CoursesDB` during quiesce after import jobs stop. Do not add it to durable integrity checks or backup contents.

- [ ] **Step 5: Add thin controller bindings**

Implement:

```go
func (c *NormalController) ChooseOpeningCourseFile() (string, error)

func (c *NormalController) StartOpeningCourseImport(path string) (string, error) {
	return c.StartImport(importjob.ImportRequest{
		Kind: importjob.KindCourse, SourceID: "coursepack", Path: path,
	})
}
```

Rename the private shared operation from `StartPuzzleImport` to `StartImport`; keep the existing public `StartPuzzleImport` wrapper if generated or backend tests still call it. Course and puzzle imports share `CancelImport`, `GetImportResult`, and Wails events.

- [ ] **Step 6: Regenerate bindings and verify**

```bash
/Users/admin/go/bin/wails generate module -tags bindings
perl -pi -e 's/[ \t]+$//' frontend/wailsjs/go/models.ts
perl -0pi -e 's/\s+\z/\n/' frontend/wailsjs/go/models.ts
chmod 0644 frontend/wailsjs/runtime/package.json frontend/wailsjs/runtime/runtime.d.ts frontend/wailsjs/runtime/runtime.js
go test ./... -count=1
git diff --check
```

Expected: PASS; generated `NormalController` bindings contain both course-import methods and existing puzzle method signatures remain present.

- [ ] **Step 7: Commit backend import integration**

```bash
git add internal/importjob internal/app application_runtime_test.go normal_controller.go controller_actions.go controllers_test.go frontend/wailsjs
git diff --cached --check
git commit -m "feat: expose private course imports"
```

### Task 7: Add the Course Import Surface to Svelte

**Files:**

- Modify: `frontend/src/lib/api-contract.ts`
- Modify: `frontend/src/lib/api.ts`
- Test: `frontend/src/lib/api.test.ts`
- Modify: `frontend/src/lib/import-session.ts`
- Test: `frontend/src/lib/import-session.test.ts`
- Create: `frontend/src/components/import/ImportCard.svelte`
- Test: `frontend/src/components/import/ImportCard.test.ts`
- Modify: `frontend/src/components/import/ImportPanel.svelte`
- Modify: `frontend/src/components/import/ImportPanel.test.ts`
- Modify: `frontend/src/components/parent/ParentDashboard.svelte`
- Modify: `frontend/src/components/parent/ParentDashboard.test.ts`
- Modify: `frontend/src/components/app/NormalShell.svelte`
- Modify: `frontend/src/test-fakes.ts`

**Interfaces:**

- Adds to `NormalAPI`: `chooseOpeningCourseFile()` and `startOpeningCourseImport(path)`.
- Adds to `ImportReport`: `counts: Record<string, number>` defaulting to an empty object when omitted.
- Produces: `createImportSession(api, kind)` where `kind` is `puzzle` or `course`.
- Produces: one import screen with independent puzzle and opening-course cards sharing backend busy/cancel events.

- [ ] **Step 1: Write failing API and session tests**

Add a decoder test proving this payload is accepted:

```ts
const result = decodeImportResult({
  jobId: 'course-job',
  status: 'succeeded',
  report: {
    accepted: 1, duplicates: 0, rejected: 0,
    counts: { chapters: 3, moves: 40, lessons: 8 }
  }
})
expect(result.report.counts).toEqual({ chapters: 3, moves: 40, lessons: 8 })
```

Add import-session tests asserting the `course` configuration calls `chooseOpeningCourseFile` and `startOpeningCourseImport`, while `puzzle` keeps the current methods.

- [ ] **Step 2: Run focused tests and verify RED**

Run:

```bash
npm --prefix frontend test -- --run frontend/src/lib/api.test.ts frontend/src/lib/import-session.test.ts
```

Expected: FAIL because course APIs, report counts, and configured sessions do not exist.

- [ ] **Step 3: Extend strict API decoding and production adapters**

Add:

```ts
export type ImportKind = 'puzzle' | 'course'
export type ImportReport = {
  accepted: number
  duplicates: number
  rejected: number
  counts: Record<string, number>
}
```

Create a `numberRecord` decoder that rejects arrays, non-finite values, negative counts, and non-integer counts. Decode absent `counts` as `{}` for legacy puzzle events. Wire production and preview API methods to generated Wails functions and update `fakeAPI` defaults.

- [ ] **Step 4: Generalize the import session**

Change the constructor to:

```ts
export function createImportSession(
  api: () => NormalAPI,
  kind: ImportKind
): ImportSession
```

Inside `selectFile` and `start`, select methods explicitly:

```ts
const choose = kind === 'course'
  ? api().chooseOpeningCourseFile
  : api().choosePuzzleImportFile
const start = kind === 'course'
  ? api().startOpeningCourseImport
  : api().startLichessImport
```

Retain job-ID filtering, monotonic progress, terminal-result precedence, cancellation, and navigation persistence.

- [ ] **Step 5: Build reusable import cards**

`ImportCard.svelte` receives `kind`, `session`, heading, description, file label, choose label, and start label. For successful course imports show `chapters`, `moves`, and `lessons` from `report.counts`; for puzzle imports retain accepted/duplicate/rejected copy. Use `rows read` for puzzles and `course records checked` for courses.

`ImportPanel.svelte` renders:

- `Import Lichess puzzles` with the existing compressed-file copy; and
- `Import opening course` with `Private .ctcourse files stay on this Mac.`

`ParentDashboard` changes its one action to `Import content`; `NormalShell` owns and connects both long-lived import sessions and passes them to `ImportPanel`.

- [ ] **Step 6: Run frontend tests and checks**

```bash
npm --prefix frontend test -- --run frontend/src/lib/api.test.ts frontend/src/lib/import-session.test.ts frontend/src/components/import/ImportCard.test.ts frontend/src/components/import/ImportPanel.test.ts frontend/src/components/parent/ParentDashboard.test.ts
npm --prefix frontend test -- --run --single-thread
npm --prefix frontend run check
```

Expected: all PASS with zero Svelte warnings.

- [ ] **Step 7: Commit course import UI**

```bash
git add frontend/src/lib frontend/src/components/import frontend/src/components/parent/ParentDashboard.svelte frontend/src/components/parent/ParentDashboard.test.ts frontend/src/components/app/NormalShell.svelte frontend/src/test-fakes.ts
git diff --cached --check
git commit -m "feat: add private course import UI"
```

### Task 8: Persist Opening Progress and Share Review Arithmetic

**Files:**

- Create: `internal/storage/migrations/user/004.sql`
- Modify: `internal/storage/sqlite_test.go`
- Create: `internal/spacedreview/schedule.go`
- Test: `internal/spacedreview/schedule_test.go`
- Modify: `internal/training/review.go`
- Modify: `internal/training/review_test.go`
- Create: `internal/openings/user_store.go`
- Test: `internal/openings/user_store_test.go`

**Interfaces:**

- Produces: opening preferences, resumable sessions, attempts, lesson progress, prompt mastery, and opening reviews in `user.sqlite`.
- Produces: `spacedreview.Next(now, state, outcome)` using 1, 3, 7, 21, and 60-day intervals.
- Preserves: every existing `training.NextReview` result byte-for-byte at the Go value level.
- Produces: `openings.UserStore` methods consumed by the opening service in Task 9.

- [ ] **Step 1: Write the failing migration test**

Update the expected user migration count from 3 to 4. Add a schema-signature test requiring these tables and indexes.

Create `internal/storage/migrations/user/004.sql` only after observing RED; the target schema is:

```sql
CREATE TABLE opening_preferences (
  course_id TEXT PRIMARY KEY,
  depth TEXT NOT NULL CHECK(depth IN ('quick','standard','reference')),
  updated_at INTEGER NOT NULL
);
CREATE TABLE opening_sessions (
  session_id TEXT PRIMARY KEY,
  course_id TEXT NOT NULL,
  generation_id TEXT NOT NULL,
  lesson_id TEXT NOT NULL,
  mode TEXT NOT NULL CHECK(mode IN ('lesson','review')),
  status TEXT NOT NULL CHECK(status IN ('active','paused','completed','restart_required')),
  depth TEXT NOT NULL CHECK(depth IN ('quick','standard','reference')),
  step_index INTEGER NOT NULL CHECK(step_index >= 0),
  state_json TEXT NOT NULL,
  created_at INTEGER NOT NULL,
  updated_at INTEGER NOT NULL
);
CREATE TABLE opening_lesson_progress (
  course_id TEXT NOT NULL,
  lesson_id TEXT NOT NULL,
  completed_step_ids_json TEXT NOT NULL,
  completed_steps INTEGER NOT NULL CHECK(completed_steps >= 0),
  total_steps INTEGER NOT NULL CHECK(total_steps > 0),
  completed_at INTEGER,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY(course_id, lesson_id)
);
CREATE TABLE opening_attempts (
  attempt_id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  course_id TEXT NOT NULL,
  prompt_id TEXT NOT NULL,
  semantic_fingerprint TEXT NOT NULL,
  started_at INTEGER NOT NULL,
  completed_at INTEGER,
  outcome TEXT,
  incorrect_moves INTEGER NOT NULL DEFAULT 0,
  alternatives_tried INTEGER NOT NULL DEFAULT 0,
  hints_used INTEGER NOT NULL DEFAULT 0,
  revealed INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE opening_prompt_progress (
  course_id TEXT NOT NULL,
  prompt_id TEXT NOT NULL,
  semantic_fingerprint TEXT NOT NULL,
  last_outcome TEXT NOT NULL,
  updated_at INTEGER NOT NULL,
  PRIMARY KEY(course_id, prompt_id)
);
CREATE TABLE opening_review_state (
  course_id TEXT NOT NULL,
  prompt_id TEXT NOT NULL,
  semantic_fingerprint TEXT NOT NULL,
  due_at INTEGER NOT NULL,
  interval_index INTEGER NOT NULL CHECK(interval_index BETWEEN 0 AND 4),
  successful_reviews INTEGER NOT NULL CHECK(successful_reviews >= 0),
  last_outcome TEXT NOT NULL,
  status TEXT NOT NULL CHECK(status IN ('active','archived')),
  PRIMARY KEY(course_id, prompt_id)
);
CREATE INDEX idx_opening_sessions_resume ON opening_sessions(status, updated_at);
CREATE UNIQUE INDEX idx_opening_sessions_single_resumable
  ON opening_sessions((1))
  WHERE status IN ('active','paused','restart_required');
CREATE INDEX idx_opening_reviews_due ON opening_review_state(status, due_at, course_id, prompt_id);
CREATE INDEX idx_opening_attempts_prompt ON opening_attempts(course_id, prompt_id, started_at);
```

- [ ] **Step 2: Run migration tests and verify RED, then add migration**

Run: `go test ./internal/storage -run 'TestMigrateCreatesEachSchema|TestUserMigration004' -v`

Expected before the SQL file: FAIL with user migration count 3 or missing tables. Add the migration and rerun; expect PASS.

- [ ] **Step 3: Extract shared spaced-review arithmetic test-first**

Create `internal/spacedreview/schedule_test.go` covering clean advancement, maximum clamp, and missed/hinted/revealed reset. Implement:

```go
type Outcome string
const (
	Clean Outcome = "clean"
	Missed Outcome = "missed"
	Hinted Outcome = "hinted"
	Revealed Outcome = "revealed"
)

type State struct {
	IntervalIndex     int
	SuccessfulReviews int
}

type Scheduled struct {
	State State
	DueAt time.Time
}

func Next(now time.Time, current State, outcome Outcome) Scheduled
```

The interval list is exactly 24h, 72h, 7d, 21d, and 60d. Permit `IntervalIndex == -1` only as an in-memory new-item input; a clean result advances it to 0 and schedules one day. Missed, hinted, and revealed reset to index 0 and zero successes. Unknown outcomes panic, matching existing training behavior.

Modify `training.NextReview` to translate its existing state through `spacedreview.Next`. Run `go test ./internal/training -run TestNextReview -count=1` before and after; expected outputs must not change.

- [ ] **Step 4: Write failing user-store transaction tests**

Test:

- default depth falls back to a course's supplied default and `SetDepth` persists;
- session creation and `SaveSession` resume the exact step/FEN/path/hint state;
- creation rejects a second active, paused, or restart-required opening session, while a completed session does not block the next lesson or review;
- lesson completion stores stable step IDs and projects progress by intersecting them with the active lesson after an update;
- an alternative increments `alternatives_tried` but not `incorrect_moves`;
- clean first learning creates a one-day review using index 0;
- hinted/missed/revealed reset review to one day;
- clean due review advances through 3, 7, 21, and 60 days;
- completion updates attempt, prompt progress, review, lesson progress, and session in one transaction; and
- `ProtectedGenerationIDs` returns generations referenced by active, paused, or restart-required sessions.

- [ ] **Step 5: Implement the opening user store**

Use these persisted values:

```go
type OpeningSessionMode string
const (
	OpeningModeLesson OpeningSessionMode = "lesson"
	OpeningModeReview OpeningSessionMode = "review"
)

type OpeningSessionStatus string
const (
	OpeningStatusActive          OpeningSessionStatus = "active"
	OpeningStatusPaused          OpeningSessionStatus = "paused"
	OpeningStatusCompleted       OpeningSessionStatus = "completed"
	OpeningStatusRestartRequired OpeningSessionStatus = "restart_required"
)

type SessionState struct {
	PositionID       string   `json:"positionId"`
	CurrentFEN       string   `json:"currentFen"`
	PlayedMoveIDs    []string `json:"playedMoveIds"`
	ReviewPromptIDs  []string `json:"reviewPromptIds,omitempty"`
	ReviewIndex      int      `json:"reviewIndex,omitempty"`
	HintLevel        int      `json:"hintLevel"`
	IncorrectMoves   int      `json:"incorrectMoves"`
	AlternativesTried int     `json:"alternativesTried"`
	HintsUsed        int      `json:"hintsUsed"`
	Revealed         bool     `json:"revealed"`
	AttemptID        string   `json:"attemptId"`
	PromptID         string   `json:"promptId,omitempty"`
	StartedAt        time.Time `json:"startedAt"`
}

type StoredSession struct {
	ID, CourseID, GenerationID, LessonID string
	Mode OpeningSessionMode
	Status OpeningSessionStatus
	Depth Depth
	StepIndex int
	State SessionState
}
```

Expose `Depth`, `SetDepth`, `CreateSession`, `LoadSession`, `ResumableSession`, `SaveSession`, `CompletePrompt`, `CompleteLesson`, `LessonProgress`, `DueReviews`, `Review`, `ProtectedGenerationIDs`, and `SetSessionStatus`. `CompleteLesson` persists the ordered stable step IDs as JSON plus cached counts; `LessonProgress` intersects those IDs with the active lesson and uses the active lesson's step count, so inserted or removed steps cannot silently masquerade as completed. Every multi-table completion uses one SQL transaction.

For review mode, persist the ordered due prompt IDs and current review index in `SessionState`; use the sentinel lesson ID `review` because review prompts may originate in different lessons. `StartReview` and resume must never depend on that sentinel being a real course lesson.

- [ ] **Step 6: Run tests and commit**

```bash
go test ./internal/storage ./internal/spacedreview ./internal/training ./internal/openings -count=1
go test ./... -count=1
git add internal/storage/migrations/user/004.sql internal/storage/sqlite_test.go internal/spacedreview internal/training/review.go internal/training/review_test.go internal/openings/user_store.go internal/openings/user_store_test.go
git diff --cached --check
git commit -m "feat: persist opening learning progress"
```

### Task 9: Implement the Guided Opening Service

**Files:**

- Modify: `internal/domain/chess.go`
- Modify: `internal/domain/training.go`
- Modify: `internal/domain/training_test.go`
- Create: `internal/openings/views.go`
- Create: `internal/openings/service.go`
- Create: `internal/openings/service_steps.go`
- Test: `internal/openings/service_test.go`
- Test: `internal/openings/service_response_test.go`

**Interfaces:**

- Consumes: `SQLiteCatalog`, `UserStore`, `chessrules.Rules`, and compiled lesson steps.
- Produces: opening home, lesson session, move, hint, summary, and review view types without adding opening variants to puzzle unions.
- Produces: `Home`, `SetDepth`, `StartLesson`, `Resume`, `Advance`, `PlayMove`, `UseHint`, `Reveal`, `Pause`, and `StartReview`.

The service exposes these exact method signatures; Task 10 adds `Restart` beside them:

```go
func (s *Service) Home(context.Context) (OpeningHomeView, error)
func (s *Service) SetDepth(context.Context, string, Depth) error
func (s *Service) StartLesson(context.Context, string, string) (OpeningSessionView, error)
func (s *Service) Resume(context.Context) (*OpeningSessionView, error)
func (s *Service) Advance(context.Context, string) (OpeningStepResult, error)
func (s *Service) PlayMove(context.Context, string, string) (OpeningStepResult, error)
func (s *Service) UseHint(context.Context, string) (OpeningHintResult, error)
func (s *Service) Reveal(context.Context, string) (OpeningStepResult, error)
func (s *Service) Pause(context.Context, string) error
func (s *Service) StartReview(context.Context, string) (OpeningSessionView, error)
```

- [ ] **Step 1: Move the generic applied-move frame without changing JSON**

Move this exact declaration from `internal/domain/training.go` to `internal/domain/chess.go`:

```go
type AppliedMove struct {
	UCI          string `json:"uci"`
	ResultingFEN string `json:"resultingFen"`
}
```

Run: `go test ./internal/domain ./internal/training -count=1`

Expected: PASS and all existing generated JSON tests remain unchanged.

- [ ] **Step 2: Write failing view-contract tests**

Define and JSON-test these exact view discriminators:

```go
type OpeningSessionView struct {
	SessionID string `json:"sessionId"`
	Mode OpeningSessionMode `json:"mode"`
	Status OpeningSessionStatus `json:"status"`
	CourseID string `json:"courseId"`
	GenerationID string `json:"generationId"`
	LessonID string `json:"lessonId"`
	Depth Depth `json:"depth"`
	Current *OpeningStepView `json:"current,omitempty"`
	Summary *OpeningSummary `json:"summary,omitempty"`
	Notice string `json:"notice,omitempty"`
}

type OpeningStepView struct {
	StepID string `json:"stepId"`
	Kind StepKind `json:"kind"`
	Title string `json:"title"`
	Instruction string `json:"instruction"`
	VariationName string `json:"variationName,omitempty"`
	PositionID string `json:"positionId"`
	CurrentFEN string `json:"currentFen"`
	Orientation Perspective `json:"orientation"`
	LegalMoves []string `json:"legalMoves"`
	NoteTexts []string `json:"noteTexts"`
	StepNumber int `json:"stepNumber"`
	StepTotal int `json:"stepTotal"`
	HintLevel int `json:"hintLevel"`
	CanReveal bool `json:"canReveal"`
}

type MoveFeedback string
const (
	FeedbackExpected MoveFeedback = "expected"
	FeedbackAlternative MoveFeedback = "alternative"
	FeedbackOffCourse MoveFeedback = "off_course"
)

type OpeningHomeView struct {
	Notice  string                 `json:"notice,omitempty"`
	Courses []OpeningCourseSummary `json:"courses"`
}

type OpeningCourseSummary struct {
	CourseID         string                  `json:"courseId"`
	Title            string                  `json:"title"`
	Perspective      Perspective             `json:"perspective"`
	Depth            Depth                   `json:"depth"`
	RootPositionID   string                  `json:"rootPositionId"`
	CompletedLessons int                     `json:"completedLessons"`
	TotalLessons     int                     `json:"totalLessons"`
	DueReviews       int                     `json:"dueReviews"`
	NextLessonID     string                  `json:"nextLessonId,omitempty"`
	NextLessonTitle  string                  `json:"nextLessonTitle,omitempty"`
	HasResumable     bool                    `json:"hasResumable"`
	Chapters         []OpeningChapterSummary `json:"chapters"`
}

type OpeningChapterSummary struct {
	ChapterID string                 `json:"chapterId"`
	Title     string                 `json:"title"`
	Lessons   []OpeningLessonSummary `json:"lessons"`
}

type OpeningLessonSummary struct {
	LessonID      string `json:"lessonId"`
	Title         string `json:"title"`
	CompletedSteps int   `json:"completedSteps"`
	TotalSteps     int    `json:"totalSteps"`
	Completed      bool   `json:"completed"`
}

type OpeningSummary struct {
	TotalPrompts       int `json:"totalPrompts"`
	PositionsRecalled  int `json:"positionsRecalled"`
	BranchesRecognized int `json:"branchesRecognized"`
	Retried            int `json:"retried"`
	UsedHint           int `json:"usedHint"`
	Revealed           int `json:"revealed"`
}

type OpeningStepResult struct {
	Session       OpeningSessionView   `json:"session"`
	StepCompleted bool                 `json:"stepCompleted"`
	Feedback      MoveFeedback         `json:"feedback,omitempty"`
	Message       string               `json:"message,omitempty"`
	AppliedMoves  []domain.AppliedMove `json:"appliedMoves,omitempty"`
	FinalFEN      string               `json:"finalFen,omitempty"`
}

type OpeningHintResult struct {
	Session      OpeningSessionView `json:"session"`
	Level        int                `json:"level"`
	Text         string             `json:"text"`
	SourceSquare string             `json:"sourceSquare,omitempty"`
	TargetSquare string             `json:"targetSquare,omitempty"`
	CanReveal    bool               `json:"canReveal"`
}
```

- [ ] **Step 3: Write failing service sequencing tests**

Using the synthetic course, prove:

1. `Home` lists chapters and filters lessons by selected depth.
2. `StartLesson` begins on explain with White orientation and input disabled.
3. `Advance` moves explain to watch, then returns authoritative animation frames for watch.
4. try/branch/recall views expose every legal move but never expose primary move IDs.
5. primary repertoire play returns `expected`, persists the next step, and supplies final FEN.
6. a course alternative returns `alternative`, restores the prompt position, increments only alternatives tried, and uses neutral copy.
7. another legal move returns `off_course`, restores the prompt position, increments incorrect moves, and derives `That move is playable, but this lesson is practicing <primary SAN>.` from the compiled primary edge; the synthetic fixture therefore ends with `c3.`
8. an illegal UCI returns an error without persistence.
9. hints reveal plan text, source square, destination square, then enable reveal.
10. reveal animates the primary move, records revealed, and advances.
11. pause and resume preserve exact state.
12. completion returns positions-recalled, branches-recognized, retries, hints, and reveals without changing puzzle rating.
13. `StartReview` persists an ordered due-prompt queue, advances its review index after each answer, and completes when the queue is exhausted.

- [ ] **Step 4: Implement the service state machine**

Use one service:

```go
type Service struct {
	catalog *SQLiteCatalog
	store *UserStore
	rules RulesPort
	now func() time.Time
	notice string
}

func NewService(catalog *SQLiteCatalog, store *UserStore, rules RulesPort, notice string) *Service
```

Return the injected replaceable-store notice from `Home` without treating it as a fatal learner error. Filter lessons and edges by depth rank. Build views from persisted generation IDs, never silently switch to the active head mid-session. For explain, disable input with an empty legal-move slice. For watch, derive and return frames for every configured move. For prompt steps, return all legal moves from `rules.LegalMoves`, compare submitted UCI to compiled outgoing edges, and keep expected move IDs server-side.

Opponent edges configured in the current step are automatic frames. A primary learner move completes the prompt. Alternatives are acknowledged then the authoritative prompt FEN is returned unchanged. Legal off-course moves are validated by `ApplyUCI` but not retained on the course path.

- [ ] **Step 5: Implement hints, outcomes, and reviews**

Hint level 1 selects the first plan/explanation note referenced by the prompt's position or primary edge, falling back to `Remember the plan for this position.` Levels 2 and 3 reveal UCI source and target squares. Level 4 enables reveal and returns `Show the course move.`

On prompt completion map state to spaced-review outcomes: revealed, hinted, missed, then clean. A newly learned clean prompt starts at interval index -1 so it becomes due in one day. Complete all attempt, prompt, review, lesson-progress, and session changes through `UserStore` transactions.

- [ ] **Step 6: Run service tests and commit**

```bash
go test ./internal/domain ./internal/openings ./internal/training -count=1
go test ./... -count=1
git add internal/domain internal/openings/views.go internal/openings/service.go internal/openings/service_steps.go internal/openings/service_test.go internal/openings/service_response_test.go
git diff --cached --check
git commit -m "feat: add guided opening lessons"
```

### Task 10: Rebase Sessions and Protect Old Generations

**Files:**

- Create: `internal/openings/service_rebase.go`
- Test: `internal/openings/service_rebase_test.go`
- Modify: `internal/openings/user_store.go`
- Modify: `internal/openings/user_store_test.go`
- Modify: `internal/openings/catalog.go`
- Modify: `internal/openings/catalog_test.go`
- Modify: `internal/app/services.go`
- Modify: `internal/app/services_test.go`

**Interfaces:**

- Adds: `Service.Restart(sessionID)` for an incompatible paused lesson.
- Adds: resume-time generation rebase by stable lesson, step, position, move, and semantic prompt fingerprints.
- Adds: archival of removed or changed prompt reviews.
- Adds: `SessionAwareMaintenance.CleanupBatch` that protects every referenced generation.
- Composes: `Services.OpeningStore` and `Services.Openings` when the course store is available.

```go
func (s *Service) Restart(context.Context, string) (OpeningSessionView, error)
```

- [ ] **Step 1: Write failing compatible and incompatible rebase tests**

Create one session on generation V1, then activate V2 in three variants:

```go
t.Run("unchanged prompt rebases in place", func(t *testing.T) {
	resumed, err := service.Resume(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.GenerationID != v2.GenerationID || resumed.Current.StepID != "recall-c3-step" {
		t.Fatalf("resumed = %+v", resumed)
	}
})

t.Run("changed prompt requires checkpoint restart", func(t *testing.T) {
	resumed, err := service.Resume(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Status != "restart_required" {
		t.Fatalf("status = %q", resumed.Status)
	}
	if !strings.Contains(resumed.Notice, "course was updated") {
		t.Fatalf("notice = %q", resumed.Notice)
	}
})
```

The third variant removes the lesson and must offer restart at the first visible lesson, not attach the old state to another lesson ID.

- [ ] **Step 2: Run rebase tests and verify RED**

Run: `go test ./internal/openings -run 'Test.*Rebase|Test.*Restart' -v`

Expected: FAIL because resume does not inspect active course generations.

- [ ] **Step 3: Implement deterministic rebase and restart**

When a resumable session generation differs from the active head:

1. load both generations;
2. find the same lesson ID and current stable step ID;
3. require the same canonical position and every already played stable move ID;
4. for a prompt step, require the same semantic fingerprint;
5. on success update only generation ID and normalized state references; and
6. on failure set `restart_required` and store the greatest earlier explain or watch step whose stable step ID and canonical position still match.

Extend `SessionState` with optional `RestartStepIndex`. `Restart` accepts only `restart_required`, clears prompt attempt state, switches to the active generation, sets the retained checkpoint or step zero, and returns an active view. Preserve completed attempt history.

For review mode, do not look for the sentinel lesson ID. Rebase each queued prompt only when the prompt ID and semantic fingerprint still match. Drop and archive changed or removed prompts, preserve queue order for matches, and complete the review session with a notice if no queued prompt survives.

Add a missing-generation test that simulates course-store quarantine after an app restart. `Resume` must return a restart-required notice instructing the parent to reimport the private course rather than returning a SQL error or affecting puzzle sessions.

- [ ] **Step 4: Reconcile prompt progress and reviews**

On `Home` and after successful rebase, compare stored prompt ID/fingerprint pairs with the active compiled course. Keep exact matches. For a changed fingerprint, archive the old review and retain attempt history; the next completion creates a fresh active review. For a removed prompt, mark its review `archived`. Never move mastery between different prompt IDs.

- [ ] **Step 5: Protect old generations during cleanup**

Implement:

```go
type SessionAwareMaintenance struct {
	Catalog *SQLiteCatalog
	Store   *UserStore
}

func (m SessionAwareMaintenance) CleanupBatch(
	ctx context.Context,
	limit int,
) (bool, error) {
	protected, err := m.Store.ProtectedGenerationIDs(ctx)
	if err != nil {
		return false, err
	}
	return m.Catalog.CleanupBatch(ctx, protected, limit)
}
```

Test that a paused V1 session protects V1 after V2 activation, then restart/complete the session and prove the next cleanup deletes V1 but never the V2 head.

- [ ] **Step 6: Compose learner services**

When `CoursesDB` is available, compose `openings.NewUserStore(UserDB)` and `openings.NewService(OpeningCatalog, OpeningStore, chessrules.Rules{}, CourseNotice)`. Replace the temporary course maintenance adapter from Task 6 with `SessionAwareMaintenance`. If course storage is unavailable, leave these fields nil and preserve `CourseNotice`; normal puzzle services stay operational.

- [ ] **Step 7: Run tests and commit**

```bash
go test ./internal/openings ./internal/app -count=1
go test ./... -count=1
git add internal/openings internal/app/services.go internal/app/services_test.go
git diff --cached --check
git commit -m "feat: preserve opening sessions across updates"
```

### Task 11: Expose Strict Opening APIs Through Wails and TypeScript

**Files:**

- Modify: `normal_controller.go`
- Modify: `controllers_test.go`
- Regenerate: `frontend/wailsjs/go/models.ts`
- Regenerate: `frontend/wailsjs/go/main/NormalController.js`
- Regenerate: `frontend/wailsjs/go/main/NormalController.d.ts`
- Modify: `frontend/src/lib/api-contract.ts`
- Modify: `frontend/src/lib/api.ts`
- Test: `frontend/src/lib/api.test.ts`
- Modify: `frontend/src/test-fakes.ts`

**Interfaces:**

- Produces thin controller methods for opening home, depth, lessons, reviews, moves, hints, reveal, pause, resume, restart, and step advance.
- Produces strict TypeScript unions for active, completed, and restart-required opening sessions.
- Preserves all current puzzle API names and decoders.

- [ ] **Step 1: Write failing controller delegation tests**

Add a fake opening service port or compose a synthetic service, then verify each binding delegates under `AcquireOperation`:

```go
GetOpeningHome()
SetOpeningDepth(courseID string, depth openings.Depth)
StartOpeningLesson(courseID, lessonID string)
ResumeOpeningSession()
RestartOpeningSession(sessionID string)
AdvanceOpeningStep(sessionID string)
PlayOpeningMove(sessionID, uci string)
UseOpeningHint(sessionID string)
RevealOpeningMove(sessionID string)
PauseOpeningSession(sessionID string)
StartOpeningReview(courseID string)
```

When opening services are nil, return `Opening courses are unavailable. Reimport the private course pack.` rather than panic.

- [ ] **Step 2: Run controller tests and verify RED**

Run: `go test . -run TestNormalControllerOpening -v`

Expected: FAIL because the methods are absent.

- [ ] **Step 3: Implement thin bindings and regenerate Wails output**

Controller methods call only `c.services.Openings`; they contain no graph, SQL, review, or move rules. Regenerate:

```bash
/Users/admin/go/bin/wails generate module -tags bindings
perl -pi -e 's/[ \t]+$//' frontend/wailsjs/go/models.ts
perl -0pi -e 's/\s+\z/\n/' frontend/wailsjs/go/models.ts
chmod 0644 frontend/wailsjs/runtime/package.json frontend/wailsjs/runtime/runtime.d.ts frontend/wailsjs/runtime/runtime.js
```

Expected: generated methods match all signatures above and `models.ts` contains opening view models plus the unchanged `AppliedMove` JSON fields.

- [ ] **Step 4: Write failing TypeScript decoder tests**

Cover:

- active session requires `current` and forbids `summary`;
- completed session requires `summary` and forbids `current`;
- restart-required session requires `notice` and no enabled board input;
- depth, step kind, feedback, mode, and status reject unknown strings;
- step numbers are positive integers and `legalMoves` are strings;
- an expected completed result requires authoritative move frames and final FEN;
- alternative/off-course results forbid applied frames and final FEN; and
- absent optional arrays decode as empty only where Go intentionally uses `omitempty`.

- [ ] **Step 5: Add exact frontend contracts and adapters**

Define:

```ts
export type OpeningDepth = 'quick' | 'standard' | 'reference'
export type OpeningStepKind = 'explain' | 'watch' | 'try' | 'branch' | 'recall'
export type OpeningMoveFeedback = 'expected' | 'alternative' | 'off_course'
export type OpeningSessionMode = 'lesson' | 'review'
export type OpeningSessionStatus = 'active' | 'completed' | 'restart_required'
```

Add `OpeningHomeView`, `OpeningCourseSummary`, `OpeningChapterSummary`, `OpeningLessonSummary`, `OpeningStepView`, discriminated `OpeningSessionView`, `OpeningStepResult`, `OpeningHintResult`, and `OpeningSummary`. Production methods decode every Wails payload before returning it. Preview and `fakeAPI` return a small original synthetic course and opening lesson.

- [ ] **Step 6: Run backend/frontend contracts and commit**

```bash
go test ./... -count=1
npm --prefix frontend test -- --run frontend/src/lib/api.test.ts
npm --prefix frontend test -- --run --single-thread
npm --prefix frontend run check
git add normal_controller.go controllers_test.go frontend/wailsjs frontend/src/lib/api-contract.ts frontend/src/lib/api.ts frontend/src/lib/api.test.ts frontend/src/test-fakes.ts
git diff --cached --check
git commit -m "feat: bind opening lesson APIs"
```

### Task 12: Add Learn Openings Navigation and the Course Hub

**Files:**

- Modify: `frontend/src/lib/navigation.ts`
- Modify: `frontend/src/components/home/HomeHub.svelte`
- Modify: `frontend/src/components/home/HomeHub.test.ts`
- Create: `frontend/src/components/openings/OpeningHub.svelte`
- Test: `frontend/src/components/openings/OpeningHub.test.ts`
- Modify: `frontend/src/components/app/NormalShell.svelte`
- Modify: `frontend/src/App.test.ts`
- Modify: `frontend/src/styles/app.css`

**Interfaces:**

- Adds screens: `openings`, `opening-lesson`, and `opening-explorer`.
- Adds a Home card labeled `Learn Openings` with next lesson and due-review count.
- Produces an opening hub with course depth, progress, chapters, lessons, continue, review, and explorer actions.

- [ ] **Step 1: Write failing Home and hub tests**

Extend `HomeHub.test.ts` to expect `Learn Openings` and copy for imported and empty states. Create hub tests that assert:

```ts
expect(screen.getByRole('heading', { name: 'Italian Game for White' })).toBeInTheDocument()
expect(screen.getByLabelText('Course depth')).toHaveValue('reference')
expect(screen.getByRole('button', { name: 'Continue Giuoco Piano' })).toBeInTheDocument()
expect(screen.getByRole('button', { name: 'Review 3 due positions' })).toBeInTheDocument()
expect(screen.getByRole('button', { name: 'Explore variations' })).toBeInTheDocument()
```

An empty course list must show `Import a private .ctcourse file from Parent settings.` and no start button.

- [ ] **Step 2: Run focused tests and verify RED**

Run:

```bash
npm --prefix frontend test -- --run frontend/src/components/home/HomeHub.test.ts frontend/src/components/openings/OpeningHub.test.ts
```

Expected: FAIL because the opening navigation and hub are absent.

- [ ] **Step 3: Implement navigation and Home summary**

Add the three screen values. After profile existence is known, `NormalShell.initialise` starts profile, puzzle-resume, opening-resume, and opening-home reads together. Wrap only the two opening reads so an unavailable course service becomes an empty opening state with the controller's reimport notice; then await the resulting safe promises with `Promise.all`. Preserve existing fatal handling for profile and puzzle failures. Keep `activeSession` and `activeOpeningSession` separate; neither replaces the other. A course-storage notice appears on the opening card/hub but does not become the shell-wide fatal error.

`HomeHub` dispatches `openings` and displays:

- no imported course: `Import a private course`;
- resumable opening session: `Continue your Italian lesson`; or
- due reviews: `<n> opening reviews due`.

- [ ] **Step 4: Implement the course hub**

Render course title/perspective, cumulative progress, due reviews, depth selector, ordered chapters, and ordered visible lessons. Changing depth calls `setOpeningDepth`, then refreshes `getOpeningHome`; it never clears progress. Dispatch typed `lesson`, `review`, and `explore` events upward.

Reference-only lessons remain absent at Quick/Standard rather than rendered as disabled. Completed lessons stay visible with neutral completion copy. Use buttons and native select controls with accessible names from the tests.

- [ ] **Step 5: Integrate hub actions into NormalShell**

Starting or continuing a lesson sets `activeOpeningSession` and routes to `opening-lesson`. Starting review calls `startOpeningReview`. Explorer routing stores course ID and root position ID. Home navigation preserves a returned persisted opening session just as puzzle solved-state navigation preserves its pending session.

- [ ] **Step 6: Run tests and commit**

```bash
npm --prefix frontend test -- --run frontend/src/components/home/HomeHub.test.ts frontend/src/components/openings/OpeningHub.test.ts frontend/src/App.test.ts
npm --prefix frontend test -- --run --single-thread
npm --prefix frontend run check
git add frontend/src/lib/navigation.ts frontend/src/components/home frontend/src/components/openings/OpeningHub.svelte frontend/src/components/openings/OpeningHub.test.ts frontend/src/components/app/NormalShell.svelte frontend/src/App.test.ts frontend/src/styles/app.css
git diff --cached --check
git commit -m "feat: add opening course hub"
```

### Task 13: Build the Guided Opening Lesson Screen

**Files:**

- Create: `frontend/src/components/chess/board-effects.ts`
- Create: `frontend/src/components/chess/move-animation.ts`
- Create: `frontend/src/components/chess/request-owner.ts`
- Modify: `frontend/src/components/puzzle/puzzle-effects.ts`
- Modify: `frontend/src/components/puzzle/move-animation.ts`
- Modify: `frontend/src/components/puzzle/request-owner.ts`
- Modify: `frontend/src/components/puzzle/puzzle-controller.ts`
- Modify: `frontend/src/components/chess/ChessBoard.svelte`
- Modify: `frontend/src/components/chess/ChessBoard.test.ts`
- Create: `frontend/src/components/openings/opening-state.ts`
- Test: `frontend/src/components/openings/opening-state.test.ts`
- Create: `frontend/src/components/openings/opening-controller.ts`
- Test: `frontend/src/components/openings/opening-controller.test.ts`
- Create: `frontend/src/components/openings/OpeningLessonScreen.svelte`
- Test: `frontend/src/components/openings/OpeningLessonScreen.test.ts`
- Modify: `frontend/src/components/app/NormalShell.svelte`
- Modify: `frontend/src/App.test.ts`
- Modify: `frontend/src/styles/app.css`

**Interfaces:**

- Extracts generic board effects, animation, and request ownership while preserving puzzle exports and behavior.
- Produces an explicit opening interaction state for explain, ready, requesting, animating, step-complete, summary, restart-required, and failed phases.
- Produces a controller that defers the already-persisted next step until the learner presses Continue.
- Produces the approved board-first explain/watch/try/branch/recall screen.

- [ ] **Step 1: Move generic helpers with characterization tests**

Copy the existing effect, move-animation, and request-owner implementations into `components/chess` under names `BoardEffects`, `browserBoardEffects`, `animateAppliedMoves`, and `RequestOwner`. Replace the old puzzle files with re-exports:

```ts
export {
  browserBoardEffects as browserPuzzleEffects,
  type BoardEffects as PuzzleEffects
} from '../chess/board-effects'
```

Use equivalent re-exports for animation and request ownership. Update puzzle-controller imports only after the current focused puzzle tests pass unchanged.

Run:

```bash
npm --prefix frontend test -- --run frontend/src/components/puzzle/move-animation.test.ts frontend/src/components/puzzle/puzzle-controller.test.ts
```

Expected: PASS before and after extraction.

- [ ] **Step 2: Generalize board error copy**

Replace `current puzzle` in `ChessBoard` invalid-data messages with `current position`. Update exact test expectations and run `ChessBoard.test.ts`. Do not change legal destinations, promotion, keyboard, markers, or Chessground configuration.

- [ ] **Step 3: Write failing opening-state tests**

Define:

```ts
export type OpeningState =
  | { phase: 'passive'; session: ActiveOpeningSessionView; fen: string }
  | { phase: 'ready'; session: ActiveOpeningSessionView; fen: string; hint: OpeningHintResult | null }
  | { phase: 'requesting'; session: ActiveOpeningSessionView; fen: string; requestId: number; operation: OpeningOperation }
  | { phase: 'animating'; session: ActiveOpeningSessionView; fen: string; requestId: number }
  | { phase: 'step-complete'; session: ActiveOpeningSessionView; fen: string; pending: OpeningSessionView; message: string }
  | { phase: 'summary'; session: CompletedOpeningSessionView }
  | { phase: 'restart-required'; session: RestartRequiredOpeningSessionView }
  | { phase: 'failed'; session: ActiveOpeningSessionView; fen: string; message: string; recoverable: boolean }

export type OpeningOperation = 'advance' | 'move' | 'hint' | 'reveal' | 'pause' | 'restart'
```

Tests must prove stale responses are ignored, off-course and alternative results restore authoritative FEN, successful frames enter step-complete with pending persisted session, Continue adopts pending state, and completed pending state becomes summary only after Continue.

- [ ] **Step 4: Run state tests and verify RED, then implement pure transitions**

Run: `npm --prefix frontend test -- --run frontend/src/components/openings/opening-state.test.ts`

Expected: FAIL because the state module is absent. Implement pure functions `initialiseOpening`, `beginOpeningRequest`, `markOpeningFeedback`, `beginOpeningAnimation`, `completeOpeningStep`, `finishOpeningHint`, `acknowledgeOpeningStep`, and `failOpeningRequest`; rerun and expect PASS.

- [ ] **Step 5: Write failing controller tests**

With fake API, board, effects, and events, cover:

- explain Continue adopts the returned watch step;
- watch animation uses every authoritative frame;
- expected drag optimistically avoids reanimating the learner's first move;
- alternative and off-course responses restore the board and show distinct neutral copy;
- hints show note, source, target, then reveal;
- reveal animates the primary move and records pending persisted state;
- stale move/hint responses after navigation cannot alter the view;
- animation failure restores final FEN and announces a warning;
- pause returns home without discarding persisted state;
- restart-required calls restart and adopts the returned checkpoint; and
- reduced motion jumps directly to authoritative final FEN.

- [ ] **Step 6: Implement the opening controller**

Follow the puzzle controller's ports and subscription pattern, but call opening-specific API methods. Reuse `animateAppliedMoves`, `RequestOwner`, `SoundService`, and `BoardEffects`. Keep expected move IDs out of frontend state. Emit:

```ts
export type OpeningControllerEvents = {
  home(completed: boolean): void
  change(session: OpeningSessionView): void
  persisted(session: OpeningSessionView): void
}
```

For expected results, store the returned already-persisted next session as `pending`; the current solved position remains visible until Continue. For alternative/off-course results, reconcile the board to the current authoritative FEN before unlocking input.

- [ ] **Step 7: Build the lesson component test-first**

The component renders:

- board oriented to White for the pilot;
- eyebrow with chapter/variation and `Step n of m`;
- step title and instruction;
- explain/watch notes with a `Continue` action;
- try/branch/recall board input and progressive Hint;
- `Reference notes` disclosure;
- neutral feedback for alternatives and off-course moves;
- `Show course move` only when allowed;
- `Pause lesson`;
- step-complete Continue;
- restart-required explanation and `Restart from checkpoint`; and
- completion summary with positions recalled, branches recognized, retries, hints, and reveals.

Use the current dark side panel and board sizing; add opening-specific classes without duplicating Chessground CSS.

- [ ] **Step 8: Integrate deferred opening state in NormalShell**

Mirror the puzzle `deferredSession` behavior with a separate `deferredOpeningSession`. Home from step-complete adopts the persisted next/completed session. Explicit Continue changes only visible presentation state. Completing an opening lesson clears only opening state, never an active puzzle session.

- [ ] **Step 9: Run tests and commit**

```bash
npm --prefix frontend test -- --run frontend/src/components/chess/ChessBoard.test.ts frontend/src/components/puzzle/puzzle-controller.test.ts frontend/src/components/openings/opening-state.test.ts frontend/src/components/openings/opening-controller.test.ts frontend/src/components/openings/OpeningLessonScreen.test.ts frontend/src/App.test.ts
npm --prefix frontend test -- --run --single-thread
npm --prefix frontend run check
git add frontend/src/components/chess frontend/src/components/puzzle frontend/src/components/openings frontend/src/components/app/NormalShell.svelte frontend/src/App.test.ts frontend/src/styles/app.css
git diff --cached --check
git commit -m "feat: add guided opening lesson screen"
```

### Task 14: Add the Read-Only Variation Explorer

**Files:**

- Modify: `internal/openings/views.go`
- Modify: `internal/openings/service.go`
- Test: `internal/openings/explorer_test.go`
- Modify: `normal_controller.go`
- Modify: `controllers_test.go`
- Regenerate: `frontend/wailsjs/go/models.ts`
- Regenerate: `frontend/wailsjs/go/main/NormalController.js`
- Regenerate: `frontend/wailsjs/go/main/NormalController.d.ts`
- Modify: `frontend/src/lib/api-contract.ts`
- Modify: `frontend/src/lib/api.ts`
- Test: `frontend/src/lib/api.test.ts`
- Create: `frontend/src/components/openings/VariationExplorer.svelte`
- Test: `frontend/src/components/openings/VariationExplorer.test.ts`
- Modify: `frontend/src/components/app/NormalShell.svelte`

**Interfaces:**

- Produces: `Service.Explore(courseID, positionID, depth)` with no learner-state writes.
- Adds: `NormalController.GetOpeningPosition(courseID, positionID, depth)`.
- Produces: board, outgoing branch, notes, evaluation, source references, and transposition indicators.

- [ ] **Step 1: Write failing backend explorer tests**

Assert that Reference returns Quick, Standard, and Reference edges; Quick excludes deeper edges; outgoing moves preserve source order; incoming move count identifies a transposition; notes/evaluations/source references are present; and repeated calls leave `opening_attempts`, `opening_prompt_progress`, and `opening_review_state` row counts unchanged.

Use these view fields:

```go
type ExplorerMove struct {
	MoveID string `json:"moveId"`
	UCI string `json:"uci"`
	SAN string `json:"san"`
	ToPositionID string `json:"toPositionId"`
	Role TrainingRole `json:"role"`
	VariationName string `json:"variationName,omitempty"`
	Evaluation Evaluation `json:"evaluation"`
	SourceRef SourceRef `json:"sourceRef"`
}

type NoteView struct {
	Kind string `json:"kind"`
	Text string `json:"text"`
	SourceRef SourceRef `json:"sourceRef"`
}

type ExplorerPositionView struct {
	CourseID string `json:"courseId"`
	PositionID string `json:"positionId"`
	FEN string `json:"fen"`
	Label string `json:"label"`
	Evaluation Evaluation `json:"evaluation"`
	Notes []NoteView `json:"notes"`
	Moves []ExplorerMove `json:"moves"`
	IncomingPaths int `json:"incomingPaths"`
}
```

- [ ] **Step 2: Run backend tests and verify RED, then implement**

Run: `go test ./internal/openings -run TestExplore -v`

Expected: FAIL because the view and service method are absent. Implement as read-only catalogue projection and rerun; expect PASS.

- [ ] **Step 3: Bind and strictly decode explorer responses**

Add the controller method, regenerate Wails bindings with the normalized command from Task 11, add `getOpeningPosition` to `NormalAPI`, and reject unknown role/evaluation values, malformed source references, notes, or moves in the TypeScript decoder.

- [ ] **Step 4: Write failing component tests**

Render the root and assert board FEN, position label, evaluation label, notes, source page, and branch buttons by SAN/variation. Click `...Bc5`, assert the API receives its destination position, then press Back and assert the prior position is restored from local history. Verify no training API method is called.

- [ ] **Step 5: Implement the explorer workspace**

Use a three-part responsive layout: branch/history list, read-only `ChessBoard`, and notes/movetext panel. Clicking an edge fetches the destination view and appends local history. Back pops history; Reset returns to the course root. Show `This position is reached by <n> move orders` when `incomingPaths > 1`. Provide `Practice this branch` only when a visible lesson ID is explicitly returned later; version one omits the button rather than inventing a lesson.

- [ ] **Step 6: Run tests and commit**

```bash
go test ./internal/openings . -count=1
npm --prefix frontend test -- --run frontend/src/lib/api.test.ts frontend/src/components/openings/VariationExplorer.test.ts
npm --prefix frontend test -- --run --single-thread
npm --prefix frontend run check
git add internal/openings normal_controller.go controllers_test.go frontend/wailsjs frontend/src/lib frontend/src/components/openings/VariationExplorer.svelte frontend/src/components/openings/VariationExplorer.test.ts frontend/src/components/app/NormalShell.svelte
git diff --cached --check
git commit -m "feat: add opening variation explorer"
```

### Task 15: Verify the Complete Generic Course Platform

**Files:**

- Create: `frontend/tests/openings.spec.ts`
- Modify: `frontend/tests/test-backend.ts`
- Modify: `frontend/tests/trainer.spec.ts`
- Modify: `scripts/release-policy.mjs`
- Modify: `scripts/verify-release.test.mjs`
- Create: `docs/operations/opening-course-authoring.md`
- Modify: `docs/operations/local-build.md`
- Modify: `README.md`

**Interfaces:**

- Produces: browser coverage for import, depth, lesson, feedback, hint, resume, review, and explorer.
- Produces: a release-policy guard permitting synthetic `.ctcourse` fixtures only below `internal/openings/testdata/`.
- Produces: generic authoring and private acceptance instructions without MCO-derived content.

- [ ] **Step 1: Write the failing end-to-end opening test**

`frontend/tests/openings.spec.ts` performs this exact flow:

1. finish initial learner setup;
2. open Parent settings and Import content;
3. choose `/Users/family/Documents/synthetic-italian.ctcourse`;
4. import and see 3 chapters, move count, and lesson count;
5. return home and open Learn Openings;
6. select Reference depth;
7. start the Giuoco Piano lesson;
8. advance Explain and Watch;
9. play a configured legal alternative and see `Playable alternative`;
10. play a configured off-course move and see neutral course copy;
11. request plan, source-square, destination-square, and reveal hints;
12. reveal, pause, return home, and resume the exact next step;
13. complete the recall and see the opening summary;
14. start the due review and complete it cleanly;
15. change to Quick and confirm progress remains;
16. open the explorer, navigate a branch, and return; and
17. confirm the ordinary tactical trainer still starts and accepts its configured move.

Run: `npm --prefix frontend run test:e2e -- openings.spec.ts`

Expected: FAIL because the browser test backend has no opening-course scenario or state yet.

- [ ] **Step 2: Extend the browser test backend and make the E2E pass**

Add an original synthetic course, course import result counts, opening home, active lesson steps, applied move frames, hint levels, persisted resume state, one pre-existing due review, depth changes, and explorer positions. Record backend state for selected course path, opening moves, opening hints, and depth.

Keep existing trainer scenarios unchanged so puzzle E2E remains a regression suite. Rerun `npm --prefix frontend run test:e2e -- openings.spec.ts`; expect PASS.

- [ ] **Step 3: Add the private-course release boundary test-first**

Add and export:

```js
export function assertCourseFixtureBoundary(paths) {
  for (const filename of paths) {
    if (!filename.endsWith('.ctcourse')) continue
    if (!filename.startsWith('internal/openings/testdata/')) {
      throw new Error(`private opening course must not be tracked: ${filename}`)
    }
  }
}
```

Call it with tracked paths during release verification. Test that the synthetic fixture path passes and `courses/mco15-italian-white.ctcourse` fails. Do not inspect or copy external private files into release staging.

- [ ] **Step 4: Document the generic authoring workflow**

`docs/operations/opening-course-authoring.md` documents schema version 1, ID rules, depth ordering, roles, source coverage, compile diagnostics, validator command, import, update/rebase behavior, and the rule that copyrighted private packs stay outside the repository. Use only synthetic JSON excerpts.

Update local build acceptance with course DB quarantine, synthetic import, full lesson, exact resume, review, depth, and explorer checks. Update README with `Learn Openings` and external private-pack support; do not imply MCO is bundled.

- [ ] **Step 5: Run the complete verification matrix**

```bash
go test ./... -count=1
go test -race ./... -count=1
npm --prefix frontend run build
npm --prefix frontend run verify:licenses
npm --prefix frontend test -- --run --single-thread
npm --prefix frontend run check
npm --prefix frontend run test:e2e
node --test scripts/verify-release.test.mjs
git diff --check
```

Expected: all commands PASS; Svelte check reports zero errors and zero warnings; Playwright passes existing trainer/board suites and the new opening suite.

- [ ] **Step 6: Commit generic platform acceptance**

```bash
git add frontend/tests scripts/release-policy.mjs scripts/verify-release.test.mjs docs/operations README.md
git diff --cached --check
git commit -m "test: verify private opening courses"
```

### Task 16: Curate and Accept the Private Italian Course

**Files:**

- Create outside repository: `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse`
- Read source: `/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf`
- Do not modify or stage repository files during transcription.

**Interfaces:**

- Consumes: the validated schema and CLI from Tasks 3-5.
- Produces externally: course ID `mco15-italian-white`, content version `1.0.0`, White perspective, Reference default depth.
- Covers: printed pages 18-41, corresponding to PDF pages 35-58.
- Preserves: MCO-derived data outside Git and release artifacts.

- [ ] **Step 1: Create the external private directory and course root**

Run:

```bash
mkdir -p "/Users/admin/Documents/Private Chess Courses"
```

Create the `.ctcourse` root with `schemaVersion: 1`, the exact course/content IDs above, title `Italian Game for White`, initial chess FEN, source title/edition/private-use notice, and Reference default. Use `apply_patch` against the external file; do not use shell redirection.

- [ ] **Step 2: Inventory source coverage before transcribing moves**

Visually inspect PDF pages 35-58 and enter `sourceCoverage.expectedReferences` for every chapter overview, table column, labeled note, and continuation page. Group the inventory exactly as:

- Giuoco Piano: printed pages 18-25, PDF pages 35-42;
- Evans Gambit: printed pages 26-29, PDF pages 43-46; and
- Two Knights' Defence: printed pages 30-41, PDF pages 47-58.

The expected list is the independent completeness checklist; do not generate it from already-entered moves.

- [ ] **Step 3: Transcribe and validate one chapter at a time**

For each chapter, manually enter overview/history/plan notes, position nodes, table-column moves, evaluations, transpositions, illustrative-game citations, and source references. After each printed page, run:

```bash
go run ./cmd/coursepack validate "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse"
```

Expected during partial work: only declared coverage references not yet captured may fail. Illegal UCI, inconsistent transposition, duplicate ID, dangling reference, cycle, tier inversion, and unexpected source-reference counts must remain zero before proceeding to the next page.

- [ ] **Step 4: Add the teaching layer and depth policy**

Create exactly five teaching chapters: Italian Foundations, Giuoco Piano, Evans Gambit, Two Knights' Defence, and Mixed Recall. The three middle chapters map to the three MCO source chapters. Every lesson follows explain, watch, try, branch, recall.

Assign depth deterministically:

- Quick: shared foundations plus the leftmost principal table line at each major Black response, excluding the optional Evans chapter;
- Standard: Quick plus every named major response and the Evans accepted/declined main lines; and
- Reference: every captured column, subvariation, note, and alternative on printed pages 18-41.

At each White decision, mark the White move used by the leftmost principal table line as `repertoire`. Mark Black branches `opponent`; mark other White choices `alternative`. If two columns transpose, target the same position ID rather than duplicating the position.

- [ ] **Step 5: Run final automated private-pack acceptance**

Run:

```bash
go run ./cmd/coursepack validate "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse"
git status --short
git ls-files '*.ctcourse'
```

Expected:

- validator exit 0;
- course ID and content version match the approved values;
- counts report exactly 5 teaching chapters, including the 3 source chapters plus Foundations and Mixed Recall;
- coverage has zero missing or unexpected references and no duplicate IDs in the expected-reference inventory;
- every move is legal, every graph node is reachable, every transposition is consistent, and every recall prompt has exactly one primary move;
- `git status --short` has no private course entry; and
- `git ls-files '*.ctcourse'` lists only synthetic fixtures below `internal/openings/testdata/`.

- [ ] **Step 6: Import and visually compare every source section**

Import the external pack through Parent settings. In Reference depth, compare each chapter's move columns, variation labels, evaluations, notes, transpositions, and printed-page links against PDF pages 35-58. Confirm Quick and Standard are strict subsets and never show a line whose parent is hidden.

Complete one lesson with a clean move, an alternative, an off-course move, hints, reveal, pause/resume, and review. Confirm no opening action changes puzzle rating or tactical history.

- [ ] **Step 7: Record completion without committing the course**

Do not run `git add` on the external directory. Add the final acceptance date and validator summary to the Codex task handoff or a private note outside the repository. The repository remains clean after this content task.

---

## Final Verification

After Task 16, run once more from the isolated worktree:

```bash
go test ./... -count=1
go test -race ./... -count=1
npm --prefix frontend run build
npm --prefix frontend run verify:licenses
npm --prefix frontend test -- --run --single-thread
npm --prefix frontend run check
npm --prefix frontend run test:e2e
node --test scripts/verify-release.test.mjs
git status --short
```

Expected: every command passes; the worktree contains only intentional committed generic platform changes; the private MCO course exists only at `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse`.
