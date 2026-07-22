# Opening Course Library Expansion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a two-course private opening-library pilot: Ruy Lopez for White and Caro-Kann for Black, while proving the app handles multiple course perspectives correctly.

**Architecture:** Reuse the schema-v2 opening-course engine, importer, teaching-tree UI, lesson screen, variation explorer, journeys, and opening review scheduler. Keep real MCO-derived course data in private `.authoring.json` and `.ctcourse` files outside Git; add only generic repository tests and small multi-course UI polish.

**Tech Stack:** Go 1.26.4, Wails 2.12.0, Svelte 3, TypeScript, Vitest/Testing Library, Playwright, JSON `.ctcourse`, private manual PDF curation from MCO-15.

## Global Constraints

- Private course root: `/Users/admin/Documents/Private Chess Courses/`.
- Source PDF: `/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf`.
- New private course IDs: `mco15-ruy-lopez-white` and `mco15-caro-kann-black`.
- New private authoring files: `mco15-ruy-lopez-white.authoring.json` and `mco15-caro-kann-black.authoring.json`.
- New private packs: `mco15-ruy-lopez-white.ctcourse` and `mco15-caro-kann-black.ctcourse`.
- Ruy Lopez source scope: printed pages 42-95.
- Caro-Kann source scope: printed pages 171-196.
- Use schema version `2`, default depth `reference`, and stable IDs.
- Keep MCO-derived moves, notes, table lines, and prose out of the repository, build outputs, command logs, test fixtures, and final responses.
- Repository fixtures must use original synthetic chess content only.
- Quick depth should be about 6-8 required lessons per new course.
- Standard depth should be about 14-18 required lessons per new course.
- Reference may be larger, but required lessons must remain meaningful teaching nodes rather than one node per source column.
- No app code may special-case Ruy Lopez, Caro-Kann, Italian, or any other course by name.
- No new runtime dependency.

---

### Task 1: Prepare an isolated implementation workspace and private checkpoints

**Files:**
- Read: `docs/superpowers/specs/2026-07-22-opening-course-library-expansion-design.md`
- Read: `docs/operations/opening-course-authoring.md`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/library-expansion-baseline/`

**Interfaces:**
- Consumes: approved design spec and existing private Italian v2 pack.
- Produces: clean implementation branch/worktree, baseline verification, and recoverable private-file checkpoints.

- [ ] **Step 1: Enter an isolated worktree**

From `/Users/admin/Documents/Work/chess`, use `superpowers:using-git-worktrees`. If using the Git fallback, create:

```bash
git worktree add .worktrees/opening-course-library-expansion -b codex/opening-course-library-expansion
```

Expected: the new worktree is on branch `codex/opening-course-library-expansion`.

- [ ] **Step 2: Confirm the repository starts clean**

Run from the implementation worktree:

```bash
git status --short --branch
git log --oneline --decorate -5
```

Expected: no unstaged files. `main` may already be ahead of `origin/main`; do not rewrite or discard those existing local commits.

- [ ] **Step 3: Verify the current Italian private pack**

Run:

```bash
go run ./cmd/coursepack validate "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse" \
  > /private/tmp/italian-v2-validation.json
jq '{courseId,contentVersion,counts:{lessons:.counts.lessons,activities:.counts.activities,prompts:.counts.prompts},missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/italian-v2-validation.json
```

Expected:

```json
{
  "courseId": "mco15-italian-white",
  "contentVersion": "2.0.0",
  "counts": {
    "lessons": 23,
    "activities": 83,
    "prompts": 20
  },
  "missing": 0,
  "unexpected": 0
}
```

- [ ] **Step 4: Checkpoint current private course files without overwriting earlier checkpoints**

Run:

```bash
mkdir -p "/Users/admin/Documents/Private Chess Courses/checkpoints/library-expansion-baseline"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/checkpoints/library-expansion-baseline/mco15-italian-white.ctcourse"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.authoring.json" \
  "/Users/admin/Documents/Private Chess Courses/checkpoints/library-expansion-baseline/mco15-italian-white.authoring.json"
shasum -a 256 "/Users/admin/Documents/Private Chess Courses/checkpoints/library-expansion-baseline/"* \
  > "/Users/admin/Documents/Private Chess Courses/checkpoints/library-expansion-baseline/SHA256SUMS.txt"
```

Expected: `SHA256SUMS.txt` contains two data-file digests plus its own digest only if the shell glob is rerun later. Do not add private checkpoint files to Git.

- [ ] **Step 5: Verify private content is still outside Git**

Run:

```bash
git ls-files '*.ctcourse'
git status --short
```

Expected: tracked `.ctcourse` files are only under `internal/openings/testdata/`; private paths are absent from Git status.

---

### Task 2: Add generic multi-course and Black-perspective safeguards

**Files:**
- Create: `internal/openings/testdata/black_tree.ctcourse`
- Modify: `internal/openings/compiler_test.go`
- Modify: `internal/openings/service_test.go`

**Interfaces:**
- Consumes: `openings.Compile(pack CoursePack, rules RulesPort) (CompiledCourse, error)`.
- Produces: a synthetic Black-perspective schema-v2 fixture and tests that verify Black repertoire moves are legal when Black is to move.

- [ ] **Step 1: Create a synthetic Black repertoire fixture**

Create `internal/openings/testdata/black_tree.ctcourse` with original content only. Use this exact structure and text:

```json
{
  "schemaVersion": 2,
  "courseId": "synthetic-caro-black",
  "contentVersion": "2.0.0",
  "title": "Synthetic Caro-Kann for Black",
  "description": "An original miniature Black repertoire used to verify schema-v2 course behavior.",
  "perspective": "black",
  "defaultDepth": "reference",
  "rootPositionId": "initial",
  "rootFen": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
  "source": {
    "title": "Original synthetic fixture",
    "edition": "Synthetic",
    "privateUseNotice": "Original test material; no book text is included."
  },
  "sourceCoverage": {
    "printedPages": [1],
    "expectedReferences": [
      "p1-overview",
      "p1-e4",
      "p1-c6",
      "p1-d4",
      "p1-d5",
      "p1-e5",
      "p1-c5"
    ]
  },
  "positions": [
    {"positionId": "initial", "label": "Initial position", "evaluation": {"code": "none"}, "noteIds": []},
    {"positionId": "after-e4", "label": "White opens with e4", "evaluation": {"code": "equal"}, "noteIds": ["caro-overview"]},
    {"positionId": "after-c6", "label": "Caro-Kann setup", "evaluation": {"code": "equal"}, "noteIds": ["caro-overview"]},
    {"positionId": "after-d4", "label": "White claims the centre", "evaluation": {"code": "equal"}, "noteIds": []},
    {"positionId": "after-d5", "label": "Black challenges e4", "evaluation": {"code": "equal"}, "noteIds": []},
    {"positionId": "after-e5", "label": "Advance structure", "evaluation": {"code": "equal"}, "noteIds": []},
    {"positionId": "after-c5", "label": "Black hits the chain", "evaluation": {"code": "equal"}, "noteIds": []}
  ],
  "moves": [
    {"moveId": "white-e4", "fromPositionId": "initial", "toPositionId": "after-e4", "uci": "e2e4", "minimumDepth": "quick", "trainingRole": "opponent", "variationName": "King pawn", "evaluation": {"code": "equal"}, "noteIds": [], "sourceRef": {"printedPage": 1, "coverageId": "p1-e4"}},
    {"moveId": "black-c6", "fromPositionId": "after-e4", "toPositionId": "after-c6", "uci": "c7c6", "minimumDepth": "quick", "trainingRole": "repertoire", "variationName": "Caro-Kann", "evaluation": {"code": "equal"}, "noteIds": ["caro-overview"], "sourceRef": {"printedPage": 1, "coverageId": "p1-c6"}},
    {"moveId": "white-d4", "fromPositionId": "after-c6", "toPositionId": "after-d4", "uci": "d2d4", "minimumDepth": "quick", "trainingRole": "opponent", "variationName": "Main centre", "evaluation": {"code": "equal"}, "noteIds": [], "sourceRef": {"printedPage": 1, "coverageId": "p1-d4"}},
    {"moveId": "black-d5", "fromPositionId": "after-d4", "toPositionId": "after-d5", "uci": "d7d5", "minimumDepth": "quick", "trainingRole": "repertoire", "variationName": "Caro-Kann centre", "evaluation": {"code": "equal"}, "noteIds": [], "sourceRef": {"printedPage": 1, "coverageId": "p1-d5"}},
    {"moveId": "white-e5", "fromPositionId": "after-d5", "toPositionId": "after-e5", "uci": "e4e5", "minimumDepth": "standard", "trainingRole": "opponent", "variationName": "Advance", "evaluation": {"code": "equal"}, "noteIds": [], "sourceRef": {"printedPage": 1, "coverageId": "p1-e5"}},
    {"moveId": "black-c5", "fromPositionId": "after-e5", "toPositionId": "after-c5", "uci": "c6c5", "minimumDepth": "standard", "trainingRole": "repertoire", "variationName": "Counter the chain", "evaluation": {"code": "equal"}, "noteIds": [], "sourceRef": {"printedPage": 1, "coverageId": "p1-c5"}}
  ],
  "notes": [
    {"noteId": "caro-overview", "kind": "overview", "text": "Black builds a compact centre and challenges White before committing the pieces.", "sourceRef": {"printedPage": 1, "noteLabel": "overview", "coverageId": "p1-overview"}}
  ],
  "lessonEdges": [
    {"edgeId": "caro-foundation-to-caro-centre", "fromLessonId": "caro-foundation", "toLessonId": "caro-centre", "ordinal": 1, "kind": "continuation", "minimumDepth": "quick"},
    {"edgeId": "caro-centre-to-caro-advance", "fromLessonId": "caro-centre", "toLessonId": "caro-advance", "ordinal": 1, "kind": "alternative", "label": "Advance structure", "minimumDepth": "standard"}
  ],
  "chapters": [
    {"chapterId": "foundations", "ordinal": 1, "title": "Foundations", "overview": "Answer e4 with a compact setup.", "minimumDepth": "quick"},
    {"chapterId": "advance", "ordinal": 2, "title": "Advance", "overview": "Challenge the advanced pawn chain.", "minimumDepth": "standard"}
  ],
  "lessons": [
    {
      "lessonId": "caro-foundation",
      "chapterId": "foundations",
      "ordinal": 1,
      "title": "Answer e4 with c6",
      "objectives": ["Use c6 as Black's compact reply."],
      "minimumDepth": "quick",
      "startPositionId": "after-e4",
      "activities": [
        {"activityId": "caro-c6-concept", "kind": "concept", "title": "Compact first move", "instruction": "Prepare to challenge White's centre with d5.", "required": true, "positionId": "after-e4", "noteIds": ["caro-overview"], "moveIds": []},
        {"activityId": "caro-c6-decision", "kind": "decision", "title": "Choose Black's setup", "instruction": "Play Black's compact reply to e4.", "required": true, "positionId": "after-e4", "noteIds": [], "moveIds": [], "promptId": "prompt-caro-c6"},
        {"activityId": "caro-c6-recap", "kind": "recap", "title": "Keep the structure", "instruction": "Caro-Kann structure begins with c6 and a later d5.", "required": true, "positionId": "after-c6", "noteIds": [], "moveIds": []}
      ]
    },
    {
      "lessonId": "caro-centre",
      "chapterId": "foundations",
      "ordinal": 2,
      "title": "Challenge the centre",
      "objectives": ["Answer d4 with d5."],
      "minimumDepth": "quick",
      "startPositionId": "after-d4",
      "activities": [
        {"activityId": "caro-d5-decision", "kind": "decision", "title": "Strike back", "instruction": "Challenge White's centre immediately.", "required": true, "positionId": "after-d4", "noteIds": [], "moveIds": [], "promptId": "prompt-caro-d5"}
      ]
    },
    {
      "lessonId": "caro-advance",
      "chapterId": "advance",
      "ordinal": 1,
      "title": "Hit the pawn chain",
      "objectives": ["Use c5 against the advanced centre."],
      "minimumDepth": "standard",
      "startPositionId": "after-e5",
      "activities": [
        {"activityId": "caro-c5-decision", "kind": "decision", "title": "Challenge the chain", "instruction": "Use the c-pawn to attack the advanced centre.", "required": true, "positionId": "after-e5", "noteIds": [], "moveIds": [], "promptId": "prompt-caro-c5"}
      ]
    }
  ],
  "prompts": [
    {"promptId": "prompt-caro-c6", "positionId": "after-e4", "primaryMoveId": "black-c6", "acceptedAlternativeMoveIds": []},
    {"promptId": "prompt-caro-d5", "positionId": "after-d4", "primaryMoveId": "black-d5", "acceptedAlternativeMoveIds": []},
    {"promptId": "prompt-caro-c5", "positionId": "after-e5", "primaryMoveId": "black-c5", "acceptedAlternativeMoveIds": []}
  ]
}
```

- [ ] **Step 2: Verify the synthetic Black fixture validates**

Run:

```bash
go run ./cmd/coursepack validate internal/openings/testdata/black_tree.ctcourse
```

Expected: PASS with `courseId` equal to `synthetic-caro-black`, `counts.lessons` equal to `3`, `counts.prompts` equal to `3`, and no missing or unexpected coverage.

- [ ] **Step 3: Add a compiler regression for Black perspective roles**

In `internal/openings/compiler_test.go`, add:

```go
func TestCompileAcceptsBlackPerspectiveRepertoireMoves(t *testing.T) {
	contents, err := os.ReadFile("testdata/black_tree.ctcourse")
	if err != nil {
		t.Fatal(err)
	}
	pack, err := DecodeCoursePack(bytes.NewReader(contents))
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := Compile(pack, chessrules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	if compiled.Pack.Perspective != PerspectiveBlack {
		t.Fatalf("perspective = %q", compiled.Pack.Perspective)
	}
	if got := compiled.Moves["black-c6"].SAN; got != "c6" {
		t.Fatalf("black-c6 SAN = %q, want c6", got)
	}
	if got := compiled.Prompts["prompt-caro-c6"].PrimaryMoveID; got != "black-c6" {
		t.Fatalf("primary move = %q", got)
	}
}
```

If `os` is not already imported in `compiler_test.go`, add it to the import list.

- [ ] **Step 4: Add a service regression for multiple active courses**

In `internal/openings/service_test.go`, add a helper:

```go
func compileBlackTreeCourse(t *testing.T) CompiledCourse {
	t.Helper()
	contents, err := os.ReadFile("testdata/black_tree.ctcourse")
	if err != nil {
		t.Fatal(err)
	}
	pack, err := DecodeCoursePack(bytes.NewReader(contents))
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := Compile(pack, chessrules.Rules{})
	if err != nil {
		t.Fatal(err)
	}
	return compiled
}
```

Then add:

```go
func TestOpeningServiceHomeKeepsMultipleCoursesSeparate(t *testing.T) {
	ctx := context.Background()
	fixture := newOpeningServiceFixture(t)
	black := compileBlackTreeCourse(t)
	if _, err := fixture.catalog.Replace(ctx, black, "/private/synthetic-caro-black.ctcourse", "sha-black"); err != nil {
		t.Fatal(err)
	}
	if err := fixture.service.SetDepth(ctx, black.Pack.CourseID, DepthQuick); err != nil {
		t.Fatal(err)
	}

	home, err := fixture.service.Home(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(home.Courses) != 2 {
		t.Fatalf("courses = %+v", home.Courses)
	}
	byID := map[string]OpeningCourseSummary{}
	for _, course := range home.Courses {
		byID[course.CourseID] = course
	}
	if byID["synthetic-italian"].Perspective != PerspectiveWhite {
		t.Fatalf("white course summary = %+v", byID["synthetic-italian"])
	}
	if byID["synthetic-caro-black"].Perspective != PerspectiveBlack ||
		byID["synthetic-caro-black"].Depth != DepthQuick ||
		byID["synthetic-caro-black"].RecommendedLessonID != "caro-foundation" {
		t.Fatalf("black course summary = %+v", byID["synthetic-caro-black"])
	}
}
```

If `bytes` or `os` are not already imported in `service_test.go`, add them to the import list.

- [ ] **Step 5: Run the focused backend tests**

Run:

```bash
go test ./internal/openings \
  -run 'TestCompileAcceptsBlackPerspectiveRepertoireMoves|TestOpeningServiceHomeKeepsMultipleCoursesSeparate' \
  -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit the backend safeguards**

```bash
git add internal/openings/testdata/black_tree.ctcourse internal/openings/compiler_test.go internal/openings/service_test.go
git commit -m "test: cover black opening course perspective"
```

---

### Task 3: Polish the opening hub for a small course library

**Files:**
- Create: `frontend/src/components/openings/opening-course-groups.ts`
- Create: `frontend/src/components/openings/opening-course-groups.test.ts`
- Modify: `frontend/src/components/openings/OpeningHub.svelte`
- Modify: `frontend/src/components/openings/OpeningHub.test.ts`

**Interfaces:**
- Consumes: `OpeningCourseSummary.perspective`.
- Produces: grouped course-picker options for `White repertoires` and `Black repertoires`.
- Preserves: `select` event detail as the selected course ID.

- [ ] **Step 1: Add a failing grouping-helper test**

Create `frontend/src/components/openings/opening-course-groups.test.ts`:

```ts
import { describe, expect, test } from 'vitest'
import type { OpeningCourseSummary } from '../../lib/api'
import { groupOpeningCourses, perspectiveLabel } from './opening-course-groups'
import { fakeOpeningHome } from '../../test-fakes'

function course(overrides: Partial<OpeningCourseSummary>): OpeningCourseSummary {
  return { ...fakeOpeningHome.courses[0], ...overrides }
}

describe('opening course grouping', () => {
  test('labels course perspectives for reader-facing copy', () => {
    expect(perspectiveLabel('white')).toBe('White')
    expect(perspectiveLabel('black')).toBe('Black')
  })

  test('groups courses by learner perspective without reordering inside a group', () => {
    const groups = groupOpeningCourses([
      course({ courseId: 'italian-white', title: 'Italian Game for White', perspective: 'white' }),
      course({ courseId: 'caro-black', title: 'Caro-Kann for Black', perspective: 'black' }),
      course({ courseId: 'ruy-white', title: 'Ruy Lopez for White', perspective: 'white' })
    ])

    expect(groups).toEqual([
      {
        label: 'White repertoires',
        courses: [
          expect.objectContaining({ courseId: 'italian-white' }),
          expect.objectContaining({ courseId: 'ruy-white' })
        ]
      },
      {
        label: 'Black repertoires',
        courses: [
          expect.objectContaining({ courseId: 'caro-black' })
        ]
      }
    ])
  })
})
```

- [ ] **Step 2: Run the helper test and verify RED**

Run:

```bash
npm --prefix frontend test -- --run src/components/openings/opening-course-groups.test.ts
```

Expected: FAIL because `opening-course-groups.ts` does not exist.

- [ ] **Step 3: Implement the grouping helper**

Create `frontend/src/components/openings/opening-course-groups.ts`:

```ts
import type { OpeningCourseSummary, OpeningPerspective } from '../../lib/api'

export type OpeningCourseGroup = {
  label: string
  courses: OpeningCourseSummary[]
}

export function perspectiveLabel(perspective: OpeningPerspective): string {
  return perspective === 'white' ? 'White' : 'Black'
}

export function groupOpeningCourses(courses: OpeningCourseSummary[]): OpeningCourseGroup[] {
  const white = courses.filter((course) => course.perspective === 'white')
  const black = courses.filter((course) => course.perspective === 'black')
  return [
    { label: 'White repertoires', courses: white },
    { label: 'Black repertoires', courses: black }
  ].filter((group) => group.courses.length > 0)
}
```

- [ ] **Step 4: Use the helper in `OpeningHub.svelte`**

Add the import:

```ts
import { groupOpeningCourses, perspectiveLabel } from './opening-course-groups'
```

Add the reactive grouping:

```ts
$: courseGroups = groupOpeningCourses(home.courses)
```

Replace:

```svelte
<p class="muted">A repertoire for {course.perspective === 'white' ? 'White' : 'Black'}</p>
```

with:

```svelte
<p class="muted">A repertoire for {perspectiveLabel(course.perspective)}</p>
```

Replace the course picker `#each home.courses` block with:

```svelte
{#each courseGroups as group (group.label)}
  <optgroup label={group.label}>
    {#each group.courses as candidate (candidate.courseId)}
      <option value={candidate.courseId}>{candidate.title}</option>
    {/each}
  </optgroup>
{/each}
```

- [ ] **Step 5: Add a component regression for White and Black grouping**

Append to `frontend/src/components/openings/OpeningHub.test.ts`:

```ts
test('groups multiple courses by learner perspective', async () => {
  const black = {
    ...fakeOpeningHome.courses[0],
    courseId: 'caro-kann-black',
    title: 'Caro-Kann for Black',
    perspective: 'black' as const,
    recommendedLessonTitle: 'Answer e4 with c6'
  }
  const { container } = render(OpeningHub, {
    home: { courses: [fakeOpeningHome.courses[0], black] },
    selectedCourseId: black.courseId
  })

  expect(screen.getByRole('heading', { name: 'Caro-Kann for Black' })).toBeInTheDocument()
  expect(screen.getByText('A repertoire for Black')).toBeInTheDocument()
  expect(container.querySelector('optgroup[label="White repertoires"]')).not.toBeNull()
  expect(container.querySelector('optgroup[label="Black repertoires"]')).not.toBeNull()
  expect(screen.getByRole('option', { name: 'Italian Game for White' })).toBeInTheDocument()
  expect(screen.getByRole('option', { name: 'Caro-Kann for Black' })).toBeInTheDocument()
})
```

- [ ] **Step 6: Run frontend focused tests**

Run:

```bash
npm --prefix frontend test -- --run \
  src/components/openings/opening-course-groups.test.ts \
  src/components/openings/OpeningHub.test.ts
```

Expected: PASS.

- [ ] **Step 7: Commit the hub polish**

```bash
git add frontend/src/components/openings/opening-course-groups.ts \
  frontend/src/components/openings/opening-course-groups.test.ts \
  frontend/src/components/openings/OpeningHub.svelte \
  frontend/src/components/openings/OpeningHub.test.ts
git commit -m "feat: group opening courses by perspective"
```

---

### Task 4: Author and validate the private Ruy Lopez White pack

**Files:**
- Create: `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.authoring.json`
- Create: `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse`
- Create checkpoints under: `/Users/admin/Documents/Private Chess Courses/checkpoints/ruy-lopez-white-v1/`

**Interfaces:**
- Consumes: schema-v2 `.ctcourse` contract from `docs/operations/opening-course-authoring.md`.
- Produces: a private validated course with `courseId: "mco15-ruy-lopez-white"` and `perspective: "white"`.

- [ ] **Step 1: Render the Ruy Lopez source pages for visual review**

Render PDF pages corresponding to printed pages 42-95. The Italian mapping showed printed page 18 at PDF page 35, so use PDF pages 59-112 for the first visual pass:

```bash
mkdir -p /private/tmp/mco15-ruy-lopez-pages
/Users/admin/.cache/codex-runtimes/codex-primary-runtime/dependencies/bin/override/pdftoppm \
  -png -f 59 -l 112 -r 160 \
  "/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf" \
  /private/tmp/mco15-ruy-lopez-pages/ruy
```

Expected: PNG files exist under `/private/tmp/mco15-ruy-lopez-pages/`. Visually confirm the first rendered opening page is printed page 42 and the last is printed page 95. If the offset is off, rerender with corrected PDF page numbers and note the corrected range in the private checkpoint README.

- [ ] **Step 2: Create the private authoring inventory**

Create `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.authoring.json` with these top-level keys:

```json
{
  "paths": [],
  "noteOverrides": {},
  "teachingNodes": [],
  "teachingEdges": []
}
```

Populate `paths` by visually reading the source pages. Use this shape for each path:

```json
{
  "id": "ruy-p42-column-1",
  "chapter": "ruy",
  "column": "1",
  "printedPage": 42,
  "depth": "quick",
  "variation": "Ruy Lopez foundation",
  "evaluation": "equal",
  "san": ["e4", "e5", "Nf3", "Nc6", "Bb5"]
}
```

Do not put source prose into this authoring inventory. Use original variation labels and short teaching purposes.

- [ ] **Step 3: Add the exact Ruy teaching-node manifest**

Populate `teachingNodes` with these IDs and purposes:

| Lesson ID | Depth | Purpose |
| --- | --- | --- |
| `ruy-foundations` | quick | Reach the Ruy Lopez and understand the pressure on the centre |
| `ruy-black-third-move` | quick | Compare `...a6`, Berlin, and other third-move systems |
| `ruy-morphy-a6` | quick | Meet `...a6` without losing the bishop's purpose |
| `ruy-preserve-bishop` | quick | Preserve or exchange the bishop intentionally |
| `ruy-castle-re1` | quick | Castle and connect Re1 with central play |
| `ruy-central-plan` | quick | Build the main White centre |
| `ruy-exchange` | standard | Understand the Exchange structure |
| `ruy-open` | standard | Meet the Open Ruy Lopez |
| `ruy-berlin` | standard | Meet the Berlin structure |
| `ruy-steinitz` | standard | Meet Steinitz-style restraint |
| `ruy-closed-main` | standard | Understand a main closed setup |
| `ruy-anti-marshall` | standard | Choose an Anti-Marshall plan |
| `ruy-marshall-warning` | reference | Recognize the Marshall attack as a reference branch |
| `ruy-delayed-systems` | standard | Compare delayed and deferred systems |
| `ruy-closed-systems` | reference | Survey deeper closed-system choices |
| `ruy-side-systems` | reference | Survey rarer source systems |

Use `chapterId` values `foundations`, `morphy`, `systems`, and `reference`. Use ordinals in the listed order within each chapter.

- [ ] **Step 4: Add the Ruy teaching edges**

Populate `teachingEdges` using these relationships:

```text
ruy-foundations -> ruy-black-third-move          continuation
ruy-black-third-move -> ruy-morphy-a6            continuation, ...a6
ruy-morphy-a6 -> ruy-preserve-bishop             continuation
ruy-preserve-bishop -> ruy-castle-re1            continuation
ruy-castle-re1 -> ruy-central-plan               continuation
ruy-morphy-a6 -> ruy-exchange                    alternative, Exchange
ruy-morphy-a6 -> ruy-open                        alternative, Open
ruy-black-third-move -> ruy-berlin               alternative, Berlin
ruy-black-third-move -> ruy-steinitz             alternative, Steinitz
ruy-central-plan -> ruy-closed-main              continuation
ruy-central-plan -> ruy-anti-marshall            alternative, Anti-Marshall
ruy-central-plan -> ruy-marshall-warning         reference, Marshall
ruy-black-third-move -> ruy-delayed-systems      alternative, Deferred systems
ruy-closed-main -> ruy-closed-systems            reference, Closed systems
ruy-black-third-move -> ruy-side-systems         reference, Side systems
```

Use edge IDs `<from>-to-<to>`, positive sibling ordinals, and each child lesson's minimum depth.

- [ ] **Step 5: Author the private `.ctcourse`**

Create `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse` as schema version 2. Required manifest values:

```json
{
  "schemaVersion": 2,
  "courseId": "mco15-ruy-lopez-white",
  "contentVersion": "1.0.0",
  "title": "Ruy Lopez for White",
  "description": "A private Ruy Lopez repertoire for White.",
  "perspective": "white",
  "defaultDepth": "reference",
  "rootPositionId": "initial",
  "rootFen": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
}
```

Add source metadata, source coverage, positions, moves, notes, chapters,
lessons, activities, lesson edges, and prompts according to
`docs/operations/opening-course-authoring.md`. Use original teaching prose.
Do not paste source prose into notes unless it remains private in this file.

- [ ] **Step 6: Validate the Ruy pack until clean**

Run after each authoring batch:

```bash
go run ./cmd/coursepack validate "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse" \
  > /private/tmp/ruy-lopez-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/ruy-lopez-validation.json
```

Expected: `courseId` is `mco15-ruy-lopez-white`, `missing` is `0`, `unexpected` is `0`, and counts include non-zero `lessonEdges`, `activities`, and `prompts`.

- [ ] **Step 7: Checkpoint the validated Ruy private files**

Run:

```bash
mkdir -p "/Users/admin/Documents/Private Chess Courses/checkpoints/ruy-lopez-white-v1"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.authoring.json" \
  "/Users/admin/Documents/Private Chess Courses/checkpoints/ruy-lopez-white-v1/mco15-ruy-lopez-white.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/checkpoints/ruy-lopez-white-v1/mco15-ruy-lopez-white.ctcourse"
cp -p /private/tmp/ruy-lopez-validation.json \
  "/Users/admin/Documents/Private Chess Courses/checkpoints/ruy-lopez-white-v1/validation.json"
shasum -a 256 "/Users/admin/Documents/Private Chess Courses/checkpoints/ruy-lopez-white-v1/"* \
  > "/Users/admin/Documents/Private Chess Courses/checkpoints/ruy-lopez-white-v1/SHA256SUMS.txt"
```

Expected: checkpoint contains the authoring file, pack, validation output, and hashes. Do not commit private files.

---

### Task 5: Author and validate the private Caro-Kann Black pack

**Files:**
- Create: `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.authoring.json`
- Create: `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse`
- Create checkpoints under: `/Users/admin/Documents/Private Chess Courses/checkpoints/caro-kann-black-v1/`

**Interfaces:**
- Consumes: schema-v2 `.ctcourse` contract and Black perspective role validation.
- Produces: a private validated course with `courseId: "mco15-caro-kann-black"` and `perspective: "black"`.

- [ ] **Step 1: Render the Caro-Kann source pages for visual review**

Using the same front-matter offset, printed pages 171-196 correspond to PDF pages 188-213 for the first pass:

```bash
mkdir -p /private/tmp/mco15-caro-kann-pages
/Users/admin/.cache/codex-runtimes/codex-primary-runtime/dependencies/bin/override/pdftoppm \
  -png -f 188 -l 213 -r 160 \
  "/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf" \
  /private/tmp/mco15-caro-kann-pages/caro
```

Expected: PNG files exist under `/private/tmp/mco15-caro-kann-pages/`. Visually confirm the first rendered opening page is printed page 171 and the last is printed page 196. If the offset is off, rerender with corrected PDF page numbers and note the corrected range in the private checkpoint README.

- [ ] **Step 2: Create the private authoring inventory**

Create `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.authoring.json` with:

```json
{
  "paths": [],
  "noteOverrides": {},
  "teachingNodes": [],
  "teachingEdges": []
}
```

Populate `paths` by visual review. Use path IDs like `caro-p171-column-1`,
`caro-p171-note-a`, and `caro-p172-overview`. For Black-course paths, mark
Black repertoire moves with `trainingRole: "repertoire"` in the `.ctcourse`
only when the derived position has Black to move.

- [ ] **Step 3: Add the exact Caro-Kann teaching-node manifest**

Populate `teachingNodes` with these IDs and purposes:

| Lesson ID | Depth | Purpose |
| --- | --- | --- |
| `caro-foundations` | quick | Answer `1.e4` with `...c6` |
| `caro-d5-structure` | quick | Challenge the centre with `...d5` |
| `caro-mainline-bishop` | quick | Develop the light-square bishop before locking it in |
| `caro-advance-c5` | quick | Hit the Advance pawn chain with `...c5` |
| `caro-exchange-structure` | quick | Understand the Exchange structure |
| `caro-panov-targets` | quick | Meet Panov-style central pressure |
| `caro-sideline-setup` | quick | Use a compact setup against sidelines |
| `caro-classical-main` | standard | Teach the main Classical structure |
| `caro-advance-space` | standard | Compare Advance-space plans |
| `caro-exchange-minority` | standard | Handle Exchange minority-play ideas |
| `caro-panov-iqp` | standard | Play against the isolated queen pawn |
| `caro-fantasy` | standard | Meet the Fantasy variation |
| `caro-two-knights` | standard | Meet the Two Knights setup |
| `caro-endgames` | standard | Recognize common stable endgames |
| `caro-sharp-reference` | reference | Keep sharper source lines as reference |
| `caro-rare-systems` | reference | Survey uncommon source systems |

Use `chapterId` values `foundations`, `main-systems`, `sidelines`, and
`reference`. Use ordinals in the listed order within each chapter.

- [ ] **Step 4: Add the Caro-Kann teaching edges**

Populate `teachingEdges` using:

```text
caro-foundations -> caro-d5-structure             continuation
caro-d5-structure -> caro-mainline-bishop         continuation
caro-d5-structure -> caro-advance-c5              alternative, Advance
caro-d5-structure -> caro-exchange-structure      alternative, Exchange
caro-d5-structure -> caro-panov-targets           alternative, Panov
caro-d5-structure -> caro-sideline-setup          alternative, Sidelines
caro-mainline-bishop -> caro-classical-main       continuation
caro-advance-c5 -> caro-advance-space             continuation
caro-exchange-structure -> caro-exchange-minority continuation
caro-panov-targets -> caro-panov-iqp              continuation
caro-sideline-setup -> caro-fantasy               alternative, Fantasy
caro-sideline-setup -> caro-two-knights           alternative, Two Knights
caro-classical-main -> caro-endgames              continuation
caro-mainline-bishop -> caro-sharp-reference      reference, Sharp lines
caro-sideline-setup -> caro-rare-systems          reference, Rare systems
```

Use edge IDs `<from>-to-<to>`, positive sibling ordinals, and each child lesson's minimum depth.

- [ ] **Step 5: Author the private `.ctcourse`**

Create `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse` as schema version 2. Required manifest values:

```json
{
  "schemaVersion": 2,
  "courseId": "mco15-caro-kann-black",
  "contentVersion": "1.0.0",
  "title": "Caro-Kann for Black",
  "description": "A private Caro-Kann repertoire for Black.",
  "perspective": "black",
  "defaultDepth": "reference",
  "rootPositionId": "initial",
  "rootFen": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
}
```

Author the graph so White moves from White-to-move positions are `opponent`,
and Black moves from Black-to-move positions are `repertoire` or `alternative`.
The compiler rejects role/perspective mismatches, so validate frequently.

- [ ] **Step 6: Validate the Caro-Kann pack until clean**

Run:

```bash
go run ./cmd/coursepack validate "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse" \
  > /private/tmp/caro-kann-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/caro-kann-validation.json
```

Expected: `courseId` is `mco15-caro-kann-black`, `missing` is `0`, `unexpected` is `0`, and counts include non-zero `lessonEdges`, `activities`, and `prompts`.

- [ ] **Step 7: Checkpoint the validated Caro-Kann private files**

Run:

```bash
mkdir -p "/Users/admin/Documents/Private Chess Courses/checkpoints/caro-kann-black-v1"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.authoring.json" \
  "/Users/admin/Documents/Private Chess Courses/checkpoints/caro-kann-black-v1/mco15-caro-kann-black.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/checkpoints/caro-kann-black-v1/mco15-caro-kann-black.ctcourse"
cp -p /private/tmp/caro-kann-validation.json \
  "/Users/admin/Documents/Private Chess Courses/checkpoints/caro-kann-black-v1/validation.json"
shasum -a 256 "/Users/admin/Documents/Private Chess Courses/checkpoints/caro-kann-black-v1/"* \
  > "/Users/admin/Documents/Private Chess Courses/checkpoints/caro-kann-black-v1/SHA256SUMS.txt"
```

Expected: checkpoint contains the authoring file, pack, validation output, and hashes. Do not commit private files.

---

### Task 6: Import all three courses and verify the app journey

**Files:**
- Read private packs from `/Users/admin/Documents/Private Chess Courses/`
- Modify only if app defects are found: relevant generic files under `frontend/src/components/openings/`, `frontend/src/components/app/`, or `internal/openings/`

**Interfaces:**
- Consumes: three validated private packs.
- Produces: app catalogue with Italian White, Ruy Lopez White, and Caro-Kann Black active together.

- [ ] **Step 1: Revalidate all private packs before import**

Run:

```bash
for pack in \
  "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse"
do
  go run ./cmd/coursepack validate "$pack" >/dev/null
done
```

Expected: all three commands exit 0.

- [ ] **Step 2: Import through the app UI**

Open the rebuilt or dev app. For each private pack:

1. Open Parent settings.
2. Open Import content.
3. Choose opening course.
4. Select the `.ctcourse` file.
5. Confirm the course ID and title.
6. Import the course.

Expected: each import succeeds without replacing a different course ID.

- [ ] **Step 3: Manually verify the opening hub**

Open Learn Openings and verify:

```text
Italian Game for White        perspective White
Ruy Lopez for White           perspective White
Caro-Kann for Black           perspective Black
```

Expected: course selection remains stable when switching among the three; depth controls are course-specific; each tree root and course title match the selected course.

- [ ] **Step 4: Manually verify lessons and reviews**

For each new course:

1. Start the Quick root lesson.
2. Complete at least one passive activity.
3. Complete at least one decision activity.
4. Pause and resume.
5. Open Explore variations from the course root.

Expected:

```text
Ruy Lopez board orientation: white side
Caro-Kann board orientation: black side
Ruy Lopez decision legal moves are White repertoire moves when White is to move
Caro-Kann decision legal moves are Black repertoire moves when Black is to move
Continue Learning resumes the course owning the active session
Variation Explorer loads from each selected course root
```

- [ ] **Step 5: If a generic app defect appears, fix it with a failing test first**

Use `superpowers:systematic-debugging` for the defect. Add a failing synthetic test, implement only the generic fix, and commit with a message scoped to the defect, for example:

```bash
git add <generic app files>
git commit -m "fix: keep selected opening course stable"
```

Do not add private course content to the commit.

---

### Task 7: Run full verification, rebuild, and prepare finishing options

**Files:**
- Verify: repository source tree
- Verify: private course files remain untracked
- Build: `build/bin/Chess Trainer.app`

**Interfaces:**
- Consumes: repository changes from Tasks 2-3 and private packs from Tasks 4-5.
- Produces: verified local app build and clean branch ready for merge or PR.

- [ ] **Step 1: Verify private course files are untracked**

Run:

```bash
git status --short
git ls-files '*.ctcourse'
```

Expected: private files under `/Users/admin/Documents/Private Chess Courses/` are not listed. Tracked `.ctcourse` files are only synthetic fixtures under `internal/openings/testdata/`.

- [ ] **Step 2: Run backend checks**

Run:

```bash
go test ./... -count=1
go vet ./...
```

Expected: PASS.

- [ ] **Step 3: Run frontend checks**

Run:

```bash
npm --prefix frontend test -- --run --single-thread
npm --prefix frontend run check
npm --prefix frontend run build
npm --prefix frontend run verify:licenses
```

Expected: PASS. If `frontend/wailsjs/go/models.ts` changes only by generated whitespace after Wails build, inspect the diff and restore it before finishing.

- [ ] **Step 4: Run opening browser checks**

Run:

```bash
npm --prefix frontend run test:e2e -- openings.spec.ts --project=chromium
```

Expected: PASS. If WebKit is part of the current local gate, also run:

```bash
npm --prefix frontend run test:e2e -- openings.spec.ts --project=webkit
```

- [ ] **Step 5: Rebuild the macOS app**

Run:

```bash
GOWORK=off GOTOOLCHAIN=local go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build -clean -trimpath
codesign --verify --deep --strict --verbose=2 "build/bin/Chess Trainer.app"
file "build/bin/Chess Trainer.app/Contents/MacOS/Chess Trainer"
```

Expected: Wails build succeeds, codesign verifies, and the binary is `Mach-O 64-bit executable arm64`.

- [ ] **Step 6: Final branch status and finishing workflow**

Run:

```bash
git status --short --branch
git log --oneline --decorate -8
```

Expected: clean branch. Then use `superpowers:finishing-a-development-branch` to offer merge, PR, keep, or discard options. Do not push or merge without the user's selected finishing option.
