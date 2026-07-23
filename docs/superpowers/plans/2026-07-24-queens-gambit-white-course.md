# Queen's Gambit White Course Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a large private schema-v2 `mco15-queens-gambit-white` opening course that teaches the Queen's Gambit family as a White repertoire and imports beside the existing courses.

**Architecture:** Reuse the existing schema-v2 `.ctcourse` model, `coursepack` validator, SAN-line helper, importer, catalogue replacement flow, teaching tree, lesson checkpoints, journey persistence, and variation explorer. Author all Queen's-Gambit source material privately under `/Users/admin/Documents/Private Chess Courses/`; public repository changes are limited to this plan/spec and generic synthetic app fixes if a real generic defect is found.

**Tech Stack:** Go 1.26.4, `github.com/corentings/chess/v2`, JSON `.ctcourse`, private authoring JSON, Poppler `pdftoppm`, Node.js private audit scripts, `jq`, Wails 2.12.0, Svelte, Vitest, Playwright, macOS app codesigning.

## Global Constraints

- New course ID is exactly `mco15-queens-gambit-white`.
- New course title is `Queen's Gambit for White`.
- New course perspective is exactly `white`.
- New course target `contentVersion` is exactly `1.0.0`.
- New course uses `schemaVersion: 2` and `defaultDepth: "reference"`.
- Source scope is the Queen's Gambit family from the double queen-pawn section of the MCO-15 PDF, starting at printed page 389 and ending before the next section.
- Offset verification found the Queen's Gambit section beginning on PDF page 406 and useful Queen's Gambit content ending before the next-section title/blank pages.
- Render PDF pages 406-509 for visual safety, then exclude non-Queen's-Gambit transition/blank pages from source coverage.
- Cover Queen's Gambit Declined, Tarrasch Defence, Queen's Gambit Accepted, Slav/Semi-Slav, and Chigorin Defence.
- Large v1 target is roughly 38-46 total lessons.
- Quick depth target is 7-9 visible lessons.
- Standard depth target is about 22-28 visible lessons.
- White moves are `repertoire` or `alternative`; Black moves are `opponent`.
- Decision prompts occur only where White is to move.
- Required Quick lessons usually have 2-3 required ideas.
- Required Standard lessons usually have 3-4 required ideas.
- Reference lessons may be denser, but must not force repeated decisions from the same position.
- Dense source detail belongs in optional Reference activities and the variation explorer, not in repeated required lesson steps.
- Do not add King's Indian, Nimzo-Indian, Queen's Indian, Grunfeld, Dutch, Budapest, English, Catalan, or other non-Queen's-Gambit courses in this pass.
- Do not OCR or bulk-import the PDF.
- Do not bundle private course packs with the app.
- Do not create an in-app course editor.
- Do not special-case Queen's Gambit in app code.
- Do not commit private `.ctcourse`, private authoring JSON, rendered PDF pages, private checkpoints, private audit reports, private manifests, or MCO-derived course prose to Git.
- Public reports and final responses may contain only IDs, paths, counts, validation results, and high-level summaries.

---

## File Map

Repository files:

- Read: `docs/superpowers/specs/2026-07-24-queens-gambit-white-course-design.md`
- Read: `docs/operations/opening-course-authoring.md`
- Create: `docs/superpowers/plans/2026-07-24-queens-gambit-white-course.md`
- Create if needed and keep ignored: `.superpowers/tmp/queens-gambit-import-smoke.go`
- Modify public Go/Svelte/tests only when a generic app defect is reproduced and covered by synthetic tests.

Private files outside Git:

- Create/update: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json`
- Create/update: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-baseline-before-v1/`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v1-authoring-inventory/`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v1-quick-spine/`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v1-qgd-qga/`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v1-slav-tarrasch-chigorin/`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v1-reference/`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v1-final/`
- Create: `/private/tmp/mco15-queens-gambit-pages/*.png`
- Create: `/private/tmp/queens-gambit-inventory-check.mjs`
- Create: `/private/tmp/queens-gambit-course-audit.mjs`
- Create: `/private/tmp/queens-gambit-*-validation.json`
- Create: `/private/tmp/queens-gambit-import-smoke.json`
- Create: `/private/tmp/queens-gambit-default-import.json`

---

### Task 1: Prepare workspace, validate existing library, render source pages, and checkpoint

**Files:**
- Read: `/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse`
- Create: `/private/tmp/mco15-queens-gambit-pages/*.png`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-baseline-before-v1/`

**Interfaces:**
- Consumes: approved Queen's Gambit design spec and the current validated private course library.
- Produces: a clean implementation workspace, rendered source-page images, and a recoverable private baseline checkpoint.

- [ ] **Step 1: Confirm branch and public working tree state**

Run:

```bash
git status --short --branch
git branch --show-current
git log --oneline --decorate -5
```

Expected: branch starts with `codex/`; public working tree contains only intentional spec/plan changes for this branch.

- [ ] **Step 2: Validate the existing private course library**

Run:

```bash
for pack in \
  "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse"
do
  go run ./cmd/coursepack validate "$pack" \
    | jq '{courseId,contentVersion,counts:{lessons:.counts.lessons,activities:.counts.activities,prompts:.counts.prompts,warnings:.counts.warnings},missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}'
done
```

Expected:

- Italian validates as `mco15-italian-white`, currently `2.1.0`.
- Ruy Lopez validates as `mco15-ruy-lopez-white`, currently `2.0.0`.
- Caro-Kann validates as `mco15-caro-kann-black`, currently `2.0.0`.
- Each validation object has `missing: 0` and `unexpected: 0`.

- [ ] **Step 3: Confirm Queen's Gambit is not already active as a private pack**

Run:

```bash
if [ -e "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" ]; then
  go run ./cmd/coursepack validate "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
    | jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}'
else
  echo "no existing mco15-queens-gambit-white.ctcourse"
fi
```

Expected for a first implementation: `no existing mco15-queens-gambit-white.ctcourse`. If a valid file exists because implementation is being resumed, checkpoint it before editing and continue from the closest matching task.

- [ ] **Step 4: Create the private baseline checkpoint**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-baseline-before-v1"
mkdir -p "$CHECKPOINT"
for stem in mco15-italian-white mco15-ruy-lopez-white mco15-caro-kann-black
do
  cp -p "/Users/admin/Documents/Private Chess Courses/${stem}.ctcourse" "$CHECKPOINT/${stem}.ctcourse"
  cp -p "/Users/admin/Documents/Private Chess Courses/${stem}.authoring.json" "$CHECKPOINT/${stem}.authoring.json"
done
for pack in "$CHECKPOINT/"*.ctcourse
do
  go run ./cmd/coursepack validate "$pack" > "$CHECKPOINT/$(basename "$pack" .ctcourse)-validation.json"
done
jq -s '[.[] | {courseId,contentVersion,counts:{lessons:.counts.lessons,activities:.counts.activities,prompts:.counts.prompts},missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}]' \
  "$CHECKPOINT/"*-validation.json > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/"*.ctcourse "$CHECKPOINT/"*.authoring.json "$CHECKPOINT/"*.json > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
```

Expected: every checksum prints `OK`.

- [ ] **Step 5: Render Queen's Gambit source pages from the local PDF**

Run:

```bash
mkdir -p /private/tmp/mco15-queens-gambit-pages
pdftoppm -png -r 180 \
  -f 406 -l 509 \
  "/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf" \
  /private/tmp/mco15-queens-gambit-pages/qg
ls /private/tmp/mco15-queens-gambit-pages/qg-*.png | wc -l
```

Expected: the count is `104`.

- [ ] **Step 6: Verify the rendered page offset visually**

Open these pages:

```bash
open /private/tmp/mco15-queens-gambit-pages/qg-406.png
open /private/tmp/mco15-queens-gambit-pages/qg-507.png
open /private/tmp/mco15-queens-gambit-pages/qg-508.png
open /private/tmp/mco15-queens-gambit-pages/qg-509.png
```

Expected:

- `qg-406.png` starts the Queen's Gambit family at printed page 389.
- `qg-507.png` is still Queen's Gambit family source content.
- `qg-508.png` is a transition/title page for the next section and is not authored into this course.
- `qg-509.png` is blank or non-course material and is not authored into this course.

If these expectations do not match the rendered images, stop and correct the source-page range before authoring.

- [ ] **Step 7: Verify private files remain outside Git**

Run:

```bash
git status --short
git ls-files '*.ctcourse'
```

Expected: private paths under `/Users/admin/Documents/Private Chess Courses/` are absent. Tracked `.ctcourse` files are only synthetic fixtures under `internal/openings/testdata/`.

---

### Task 2: Create the private authoring inventory and inventory checker

**Files:**
- Create/update: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json`
- Read: `/private/tmp/mco15-queens-gambit-pages/*.png`
- Create: `/private/tmp/queens-gambit-inventory-check.mjs`

**Interfaces:**
- Consumes: rendered source pages and the source scope from Task 1.
- Produces: private authoring inventory with stable family IDs, teaching-node intent, and source-coverage records.

- [ ] **Step 1: Create the private authoring JSON skeleton**

Create `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json` with this structure:

```json
{
  "courseId": "mco15-queens-gambit-white",
  "targetContentVersion": "1.0.0",
  "scope": {
    "title": "Queen's Gambit for White",
    "perspective": "white",
    "source": "MCO-15 private visual curation",
    "sourceScope": "Queen's Gambit family only",
    "families": [
      {"familyId": "qgd", "name": "Queen's Gambit Declined", "printedStart": 389, "pdfStart": 406},
      {"familyId": "tarrasch", "name": "Tarrasch Defence", "printedStart": 440, "pdfStart": 457},
      {"familyId": "qga", "name": "Queen's Gambit Accepted", "printedStart": 449, "pdfStart": 466},
      {"familyId": "slav-semi-slav", "name": "Slav and Semi-Slav Defence", "printedStart": 460, "pdfStart": 477},
      {"familyId": "chigorin", "name": "Chigorin Defence", "printedStart": 488, "pdfStart": 505}
    ],
    "excludedPdfPages": [508, 509]
  },
  "renderedPdfPages": {
    "pdfFirst": 406,
    "pdfLastRendered": 509,
    "offsetVerified": true,
    "renderDirectory": "/private/tmp/mco15-queens-gambit-pages"
  },
  "inventory": {
    "pages": [],
    "coverageRecords": [],
    "illegibleItems": []
  },
  "teachingNodes": [],
  "teachingEdges": [],
  "paths": {
    "authoringPath": "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json",
    "coursePath": "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse"
  },
  "noteOverrides": {}
}
```

Expected: the file is private and not visible in `git status`.

- [ ] **Step 2: Populate the private source inventory**

Using the rendered images, add page records to `inventory.pages` and source-reference records to `inventory.coverageRecords`. Each record must use this shape:

```json
{
  "coverageId": "qg-p389-overview",
  "familyId": "qgd",
  "printedPage": 389,
  "pdfPage": 406,
  "kind": "overview",
  "locator": "private visual locator",
  "teachingUse": "quick|standard|reference",
  "privateSummary": "private authoring summary"
}
```

Allowed `kind` values:

```json
["overview", "table-column", "note", "variation-heading", "illustrative-game", "transposition"]
```

Privacy rule: `locator` and `privateSummary` may contain private source-derived authoring notes because this file is outside Git. Do not paste those fields into public commits, test fixtures, or final reports.

- [ ] **Step 3: Add initial teaching-node intent privately**

Populate `teachingNodes` with 38-46 private node records using this shape:

```json
{
  "lessonId": "qg-foundations",
  "chapterId": "foundations",
  "minimumDepth": "quick",
  "familyId": "foundations",
  "role": "root",
  "requiredFlow": "concept-decision-recap",
  "coverageIds": ["qg-p389-overview"]
}
```

Required starting IDs:

```text
qg-foundations
qg-accept-gambit-idea
qg-black-family-split
qgd-e6-structure
qga-dxc4-recover
slav-c6-structure
tarrasch-c5-iqp
chigorin-nc6-pressure
```

Expected: these IDs become durable lesson IDs unless their teaching meaning changes before first import.

- [ ] **Step 4: Add initial teaching-edge intent privately**

Populate `teachingEdges` with a single-root tree. The first edges must use this shape:

```json
[
  {
    "edgeId": "qg-foundations-to-accept-gambit-idea",
    "fromLessonId": "qg-foundations",
    "toLessonId": "qg-accept-gambit-idea",
    "kind": "continuation",
    "minimumDepth": "quick",
    "ordinal": 1
  },
  {
    "edgeId": "qg-accept-gambit-idea-to-black-family-split",
    "fromLessonId": "qg-accept-gambit-idea",
    "toLessonId": "qg-black-family-split",
    "kind": "continuation",
    "minimumDepth": "quick",
    "ordinal": 1
  }
]
```

Expected: every non-root `teachingNodes[].lessonId` has exactly one incoming edge by the time the private `.ctcourse` is generated.

- [ ] **Step 5: Create the private inventory checker**

Create `/private/tmp/queens-gambit-inventory-check.mjs` with this exact content:

```js
import fs from 'node:fs'

const [authoringPath] = process.argv.slice(2)
if (!authoringPath) {
  console.error('usage: node queens-gambit-inventory-check.mjs <authoring.json>')
  process.exit(2)
}

const authoring = JSON.parse(fs.readFileSync(authoringPath, 'utf8'))
const failures = []
const familyIds = new Set((authoring.scope?.families ?? []).map((family) => family.familyId))
const nodeIds = new Set()
const coverageIds = new Set()
const allowedKinds = new Set(['overview', 'table-column', 'note', 'variation-heading', 'illustrative-game', 'transposition'])
const spamPattern = /source note [a-z]+ records|opening activity result|course move found/i

function requireString(value, label) {
  if (typeof value !== 'string' || value.trim() === '') failures.push(`${label} must be a non-empty string`)
}

if (authoring.courseId !== 'mco15-queens-gambit-white') failures.push(`courseId ${authoring.courseId}`)
if (authoring.targetContentVersion !== '1.0.0') failures.push(`targetContentVersion ${authoring.targetContentVersion}`)
if (authoring.scope?.perspective !== 'white') failures.push(`perspective ${authoring.scope?.perspective}`)
if (!authoring.renderedPdfPages?.offsetVerified) failures.push('renderedPdfPages.offsetVerified must be true')
if (authoring.renderedPdfPages?.pdfFirst !== 406) failures.push(`pdfFirst ${authoring.renderedPdfPages?.pdfFirst}`)

for (const record of authoring.inventory?.coverageRecords ?? []) {
  requireString(record.coverageId, 'coverageId')
  if (coverageIds.has(record.coverageId)) failures.push(`duplicate coverageId ${record.coverageId}`)
  coverageIds.add(record.coverageId)
  if (!familyIds.has(record.familyId) && record.familyId !== 'foundations') failures.push(`unknown familyId ${record.familyId}`)
  if (!Number.isInteger(record.printedPage) || record.printedPage < 389) failures.push(`invalid printedPage for ${record.coverageId}`)
  if (!Number.isInteger(record.pdfPage) || record.pdfPage < 406) failures.push(`invalid pdfPage for ${record.coverageId}`)
  if (!allowedKinds.has(record.kind)) failures.push(`invalid kind ${record.kind} for ${record.coverageId}`)
  if (typeof record.privateSummary === 'string' && spamPattern.test(record.privateSummary)) {
    failures.push(`generic spam text in ${record.coverageId}`)
  }
}

for (const node of authoring.teachingNodes ?? []) {
  requireString(node.lessonId, 'lessonId')
  if (nodeIds.has(node.lessonId)) failures.push(`duplicate lessonId ${node.lessonId}`)
  nodeIds.add(node.lessonId)
  if (!['quick', 'standard', 'reference'].includes(node.minimumDepth)) failures.push(`invalid depth for ${node.lessonId}`)
  for (const coverageId of node.coverageIds ?? []) {
    if (!coverageIds.has(coverageId)) failures.push(`${node.lessonId} references unknown coverageId ${coverageId}`)
  }
}

const incoming = new Map()
for (const edge of authoring.teachingEdges ?? []) {
  requireString(edge.edgeId, 'edgeId')
  if (!nodeIds.has(edge.fromLessonId)) failures.push(`${edge.edgeId} unknown fromLessonId ${edge.fromLessonId}`)
  if (!nodeIds.has(edge.toLessonId)) failures.push(`${edge.edgeId} unknown toLessonId ${edge.toLessonId}`)
  incoming.set(edge.toLessonId, (incoming.get(edge.toLessonId) ?? 0) + 1)
  if (!['continuation', 'alternative', 'reference'].includes(edge.kind)) failures.push(`${edge.edgeId} invalid kind ${edge.kind}`)
  if (!['quick', 'standard', 'reference'].includes(edge.minimumDepth)) failures.push(`${edge.edgeId} invalid depth ${edge.minimumDepth}`)
}

if (!nodeIds.has('qg-foundations')) failures.push('missing qg-foundations')
for (const lessonId of nodeIds) {
  if (lessonId === 'qg-foundations') continue
  if ((incoming.get(lessonId) ?? 0) !== 1) failures.push(`${lessonId} must have exactly one incoming edge`)
}

if (failures.length > 0) {
  console.error(JSON.stringify({ failures }, null, 2))
  process.exit(1)
}

console.log(JSON.stringify({
  ok: true,
  courseId: authoring.courseId,
  coverageRecords: coverageIds.size,
  teachingNodes: nodeIds.size,
  teachingEdges: (authoring.teachingEdges ?? []).length
}, null, 2))
```

- [ ] **Step 6: Run the private inventory checker**

Run:

```bash
node --check /private/tmp/queens-gambit-inventory-check.mjs
node /private/tmp/queens-gambit-inventory-check.mjs \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json"
```

Expected: output contains `"ok": true`, `teachingNodes` is between `38` and `46`, and no private prose is printed.

- [ ] **Step 7: Checkpoint the authoring inventory**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v1-authoring-inventory"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json" \
  "$CHECKPOINT/mco15-queens-gambit-white.authoring.json"
node /private/tmp/queens-gambit-inventory-check.mjs \
  "$CHECKPOINT/mco15-queens-gambit-white.authoring.json" > "$CHECKPOINT/inventory-summary.json"
shasum -a 256 "$CHECKPOINT/"*.json > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
git status --short
```

Expected: checksums print `OK`; private checkpoint files do not appear in Git.

---

### Task 3: Build the private `.ctcourse` root and Quick repertoire spine

**Files:**
- Create/update: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse`
- Read/update: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json`
- Create: `/private/tmp/queens-gambit-quick-validation.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v1-quick-spine/`

**Interfaces:**
- Consumes: private authoring inventory and existing schema-v2 authoring rules.
- Produces: a valid private Quick-depth Queen's Gambit course spine.

- [ ] **Step 1: Verify the SAN helper from the initial position**

Run:

```bash
go run ./cmd/coursepack sanline \
  --fen "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1" \
  d4 d5 c4 \
  | jq '{startFen, finalFen, uci:[.moves[].uci]}'
```

Expected:

```json
{
  "uci": ["d2d4", "d7d5", "c2c4"]
}
```

`startFen` and `finalFen` must be non-empty strings.

- [ ] **Step 2: Create the schema-v2 course root**

In `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse`, create a UTF-8 JSON object with these root fields:

```json
{
  "schemaVersion": 2,
  "courseId": "mco15-queens-gambit-white",
  "contentVersion": "1.0.0",
  "title": "Queen's Gambit for White",
  "description": "A White repertoire course for the Queen's Gambit family.",
  "perspective": "white",
  "defaultDepth": "reference",
  "rootPositionId": "initial",
  "rootFen": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
  "source": {
    "title": "Modern Chess Openings",
    "edition": "15th",
    "privateUseNotice": "Private local course pack; source-derived course data is not committed to Git."
  },
  "sourceCoverage": {
    "printedPages": [],
    "expectedReferences": []
  },
  "positions": [],
  "moves": [],
  "notes": [],
  "lessonEdges": [],
  "chapters": [],
  "lessons": [],
  "prompts": []
}
```

Expected: the file has exactly one JSON object and `schemaVersion` is `2`.

- [ ] **Step 3: Add the five chapters**

Set `chapters` to these records:

```json
[
  {"chapterId": "foundations", "title": "Queen's Gambit Foundations", "ordinal": 1, "minimumDepth": "quick"},
  {"chapterId": "qgd", "title": "Queen's Gambit Declined", "ordinal": 2, "minimumDepth": "quick"},
  {"chapterId": "qga", "title": "Queen's Gambit Accepted", "ordinal": 3, "minimumDepth": "quick"},
  {"chapterId": "slav", "title": "Slav and Semi-Slav", "ordinal": 4, "minimumDepth": "quick"},
  {"chapterId": "tarrasch-chigorin", "title": "Tarrasch and Chigorin", "ordinal": 5, "minimumDepth": "quick"}
]
```

Expected: chapter IDs match the private authoring inventory and every chapter is visible at Quick.

- [ ] **Step 4: Author the Quick move graph privately**

Using `coursepack sanline`, create connected positions and moves for the Quick spine:

```text
initial
1.d4
1...d5
2.c4
2...e6
2...dxc4
2...c6
2...Nc6
2...e6 3.Nc3 c5
```

For each move record:

- use UCI from `coursepack sanline`;
- set `minimumDepth` to `quick`;
- set White learner moves to `trainingRole: "repertoire"` unless the branch is an accepted alternative;
- set Black replies to `trainingRole: "opponent"`;
- attach source references from the private authoring inventory;
- add each used coverage ID to `sourceCoverage.expectedReferences`.

Expected: every move is legal from its `fromPositionId`, every `toPositionId` exists, and no duplicate outgoing UCI exists from one position.

- [ ] **Step 5: Author the Quick teaching lessons privately**

Create 7-9 Quick lessons. Required initial lesson IDs:

```text
qg-foundations
qg-accept-gambit-idea
qg-black-family-split
qgd-e6-structure
qga-dxc4-recover
slav-c6-structure
tarrasch-c5-iqp
chigorin-nc6-pressure
```

Each lesson must use one of these activity shapes:

```json
[
  {"kind": "concept", "required": true},
  {"kind": "decision", "required": true},
  {"kind": "recap", "required": false}
]
```

or:

```json
[
  {"kind": "concept", "required": true},
  {"kind": "demonstration", "required": true},
  {"kind": "comparison", "required": true},
  {"kind": "recap", "required": false}
]
```

Expected: each Quick lesson has 2-3 required activities, at most one required decision, and no forced repeated move from the same position.

- [ ] **Step 6: Author the Quick teaching tree**

Create `lessonEdges` so Quick has one connected route from `qg-foundations` through the main White spine, with labelled alternatives for the five Black family replies. Edge requirements:

- `qg-foundations` is the single root.
- Every Quick lesson has a connected Quick-depth path from `qg-foundations`.
- Family-split child edges use `kind: "alternative"` and labels based on Black's family move or family name.
- No edge points to a lesson shallower than its parent.

Expected: `go run ./cmd/coursepack validate` reports no teaching-tree diagnostics.

- [ ] **Step 7: Validate the Quick spine**

Run:

```bash
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
  > /private/tmp/queens-gambit-quick-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/queens-gambit-quick-validation.json
```

Expected:

- `courseId` is `mco15-queens-gambit-white`.
- `contentVersion` is `1.0.0`.
- `counts.lessons` is between `7` and `9`.
- `missing` is `0`.
- `unexpected` is `0`.

- [ ] **Step 8: Checkpoint the Quick spine**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v1-quick-spine"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json" "$CHECKPOINT/mco15-queens-gambit-white.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" "$CHECKPOINT/mco15-queens-gambit-white.ctcourse"
cp -p /private/tmp/queens-gambit-quick-validation.json "$CHECKPOINT/validation.json"
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  "$CHECKPOINT/validation.json" > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/"*.json "$CHECKPOINT/"*.ctcourse > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
```

Expected: every checksum prints `OK`.

---

### Task 4: Add Standard QGD and QGA branches

**Files:**
- Update: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse`
- Update: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json`
- Read: `/private/tmp/mco15-queens-gambit-pages/qg-406.png` through `/private/tmp/mco15-queens-gambit-pages/qg-456.png`
- Read: `/private/tmp/mco15-queens-gambit-pages/qg-466.png` through `/private/tmp/mco15-queens-gambit-pages/qg-476.png`
- Create: `/private/tmp/queens-gambit-qgd-qga-validation.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v1-qgd-qga/`

**Interfaces:**
- Consumes: valid Quick spine from Task 3.
- Produces: Standard-depth Queen's Gambit Declined and Queen's Gambit Accepted branch coverage.

- [ ] **Step 1: Author QGD Standard teaching nodes**

Add QGD lessons for these teaching jobs:

```text
qgd-classical-development
qgd-exchange-structure
qgd-orthodox-piece-placement
qgd-cambridge-springs-pressure
qgd-lasker-simplification
qgd-tartakower-setup
qgd-ragozin-pressure
qgd-vienna-pressure
qgd-semi-tarrasch-structure
```

Expected:

- QGD Standard branch has about 9-13 total QGD lessons including Quick QGD lessons.
- QGD lessons use decision activities where White chooses structure, development, central tension, or recapture plan.
- Dense sublines remain Reference-only.

- [ ] **Step 2: Author QGA Standard teaching nodes**

Add QGA lessons for these teaching jobs:

```text
qga-regain-pawn
qga-central-expansion
qga-e3-development
qga-e4-centre
qga-black-holds-pawn-warning
qga-endgame-structure
```

Expected:

- QGA branch has about 6-8 total QGA lessons including Quick QGA lessons.
- Required activities teach pawn recovery, central occupation, and development tempo.
- Rare QGA source lines remain Reference-only.

- [ ] **Step 3: Add QGD and QGA source coverage**

For each authored QGD/QGA move and note:

- attach a `sourceRef` from `inventory.coverageRecords`;
- add the same ID to `.sourceCoverage.expectedReferences`;
- keep coordinates consistent for each reused `coverageId`;
- add private note text only to private authoring/course files.

Run:

```bash
jq '{
  printedPages: (.sourceCoverage.printedPages | length),
  expectedReferences: (.sourceCoverage.expectedReferences | length),
  qgdLessons: ([.lessons[] | select(.chapterId=="qgd")] | length),
  qgaLessons: ([.lessons[] | select(.chapterId=="qga")] | length)
}' "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse"
```

Expected: counts increase from the Quick checkpoint and no source prose is printed.

- [ ] **Step 4: Validate after QGD/QGA**

Run:

```bash
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
  > /private/tmp/queens-gambit-qgd-qga-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/queens-gambit-qgd-qga-validation.json
```

Expected:

- `counts.lessons` is at least `18`.
- `missing` is `0`.
- `unexpected` is `0`.

- [ ] **Step 5: Checkpoint QGD/QGA**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v1-qgd-qga"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json" "$CHECKPOINT/mco15-queens-gambit-white.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" "$CHECKPOINT/mco15-queens-gambit-white.ctcourse"
cp -p /private/tmp/queens-gambit-qgd-qga-validation.json "$CHECKPOINT/validation.json"
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  "$CHECKPOINT/validation.json" > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/"*.json "$CHECKPOINT/"*.ctcourse > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
```

Expected: every checksum prints `OK`.

---

### Task 5: Add Standard Slav/Semi-Slav, Tarrasch, and Chigorin branches

**Files:**
- Update: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse`
- Update: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json`
- Read: `/private/tmp/mco15-queens-gambit-pages/qg-457.png` through `/private/tmp/mco15-queens-gambit-pages/qg-465.png`
- Read: `/private/tmp/mco15-queens-gambit-pages/qg-477.png` through `/private/tmp/mco15-queens-gambit-pages/qg-507.png`
- Create: `/private/tmp/queens-gambit-slav-tarrasch-chigorin-validation.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v1-slav-tarrasch-chigorin/`

**Interfaces:**
- Consumes: QGD/QGA Standard pack from Task 4.
- Produces: Standard-depth coverage for all five selected Queen's Gambit families.

- [ ] **Step 1: Author Slav and Semi-Slav Standard teaching nodes**

Add lessons for these teaching jobs:

```text
slav-main-structure
slav-bishop-before-e6
slav-central-tension
slav-quiet-development
semi-slav-e6-c6-wall
semi-slav-meran-decision
semi-slav-anti-meran-plan
semi-slav-tactical-warning
```

Expected:

- Slav/Semi-Slav branch has about 9-12 total lessons including Quick Slav lessons.
- Required lessons teach structure and White decision points rather than every table column.

- [ ] **Step 2: Author Tarrasch Standard teaching nodes**

Add lessons for these teaching jobs:

```text
tarrasch-iqp-foundation
tarrasch-pressure-d5
tarrasch-piece-activity
tarrasch-endgame-pressure
```

Expected:

- Tarrasch branch has about 4-6 total lessons including Quick Tarrasch lessons.
- Required lessons connect the isolated queen's pawn to White's pressure plan.

- [ ] **Step 3: Author Chigorin Standard teaching nodes**

Add lessons for these teaching jobs:

```text
chigorin-piece-pressure
chigorin-central-response
chigorin-development-choice
```

Expected:

- Chigorin branch has about 3-5 total lessons including Quick Chigorin lessons.
- Required lessons teach White's central response to early Black piece pressure.

- [ ] **Step 4: Add all Standard source coverage**

For each authored Slav/Semi-Slav, Tarrasch, and Chigorin move and note:

- attach a `sourceRef` from the private inventory;
- add the same ID to `.sourceCoverage.expectedReferences`;
- mark rare or very deep items as `minimumDepth: "reference"`;
- keep required Standard lesson flow at 3-4 required activities.

Run:

```bash
jq '{
  lessons: (.lessons | length),
  standardLessons: ([.lessons[] | select(.minimumDepth=="quick" or .minimumDepth=="standard")] | length),
  slavLessons: ([.lessons[] | select(.chapterId=="slav")] | length),
  tarraschChigorinLessons: ([.lessons[] | select(.chapterId=="tarrasch-chigorin")] | length),
  expectedReferences: (.sourceCoverage.expectedReferences | length)
}' "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse"
```

Expected: `standardLessons` is between `22` and `28`.

- [ ] **Step 5: Validate after all Standard branches**

Run:

```bash
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
  > /private/tmp/queens-gambit-slav-tarrasch-chigorin-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/queens-gambit-slav-tarrasch-chigorin-validation.json
```

Expected:

- `counts.lessons` is at least `30`.
- `missing` is `0`.
- `unexpected` is `0`.

- [ ] **Step 6: Checkpoint Standard complete**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v1-slav-tarrasch-chigorin"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json" "$CHECKPOINT/mco15-queens-gambit-white.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" "$CHECKPOINT/mco15-queens-gambit-white.ctcourse"
cp -p /private/tmp/queens-gambit-slav-tarrasch-chigorin-validation.json "$CHECKPOINT/validation.json"
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  "$CHECKPOINT/validation.json" > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/"*.json "$CHECKPOINT/"*.ctcourse > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
```

Expected: every checksum prints `OK`.

---

### Task 6: Add Reference coverage and run private lesson-quality hardening

**Files:**
- Update: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse`
- Update: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json`
- Create: `/private/tmp/queens-gambit-course-audit.mjs`
- Create: `/private/tmp/queens-gambit-reference-validation.json`
- Create: `/private/tmp/queens-gambit-reference-audit.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v1-reference/`

**Interfaces:**
- Consumes: Standard-complete pack from Task 5 and full private source inventory from Task 2.
- Produces: full large v1 Reference course with hardening checks for lesson flow and warning spam.

- [ ] **Step 1: Add Reference-only move graph and optional activities**

For remaining selected Queen's Gambit inventory records:

- add source-backed moves and notes to the private `.ctcourse`;
- add `sourceCoverage.expectedReferences` for every included record;
- set dense branch moves to `minimumDepth: "reference"`;
- add optional `reference` activities where a lesson needs deep context;
- keep rare/deep continuations accessible through the variation explorer.

Run:

```bash
jq '{
  lessons: (.lessons | length),
  moves: (.moves | length),
  notes: (.notes | length),
  prompts: (.prompts | length),
  referenceLessons: ([.lessons[] | select(.minimumDepth=="reference")] | length),
  optionalReferenceActivities: ([.lessons[].activities[] | select(.kind=="reference" and .required==false)] | length)
}' "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse"
```

Expected: `lessons` is between `38` and `46`; optional reference activities are present for dense source areas.

- [ ] **Step 2: Create the private course audit script**

Create `/private/tmp/queens-gambit-course-audit.mjs` with this exact content:

```js
import fs from 'node:fs'

const [packPath, outPath] = process.argv.slice(2)
if (!packPath || !outPath) {
  console.error('usage: node queens-gambit-course-audit.mjs <course.ctcourse> <audit.json>')
  process.exit(2)
}

const pack = JSON.parse(fs.readFileSync(packPath, 'utf8'))
const prompts = new Map((pack.prompts ?? []).map((prompt) => [prompt.promptId, prompt]))
const moves = new Map((pack.moves ?? []).map((move) => [move.moveId, move]))
const spamPattern = /source note [a-z]+ records|opening activity result|course move found/i
const allowedDepths = new Set(['quick', 'standard', 'reference'])

function requiredActivities(lesson) {
  return (lesson.activities ?? []).filter((activity) => activity.required !== false)
}

function promptMove(activity) {
  if (!activity.promptId) return null
  const prompt = prompts.get(activity.promptId)
  const primary = prompt ? moves.get(prompt.primaryMoveId) : null
  return {
    activityId: activity.activityId,
    promptId: activity.promptId,
    positionId: prompt?.positionId ?? null,
    primaryMoveId: prompt?.primaryMoveId ?? null,
    primaryUci: primary?.uci ?? null
  }
}

function duplicates(values, key) {
  const groups = new Map()
  for (const value of values) {
    const groupKey = key(value)
    if (!groupKey) continue
    groups.set(groupKey, [...(groups.get(groupKey) ?? []), value])
  }
  return [...groups.entries()]
    .filter(([, group]) => group.length > 1)
    .map(([value, group]) => ({ value, activityIds: group.map((entry) => entry.activityId) }))
}

const lessons = (pack.lessons ?? []).map((lesson) => {
  const required = requiredActivities(lesson)
  const decisions = required.filter((activity) => activity.kind === 'decision')
  const promptMoves = decisions.map(promptMove).filter(Boolean)
  const requiredText = required
    .flatMap((activity) => [
      activity.title,
      activity.instruction,
      ...(activity.referenceSections ?? []).flatMap((section) => [section.title, section.instruction])
    ])
    .filter((value) => typeof value === 'string')
  const maxRequired = lesson.minimumDepth === 'quick' ? 4 : 5
  return {
    lessonId: lesson.lessonId,
    chapterId: lesson.chapterId,
    minimumDepth: lesson.minimumDepth,
    activityCount: (lesson.activities ?? []).length,
    requiredCount: required.length,
    decisionCount: decisions.length,
    promptMoves,
    flags: {
      invalidDepth: !allowedDepths.has(lesson.minimumDepth),
      tooManyRequiredActivities: required.length > maxRequired,
      tooManyDecisions: decisions.length > 2,
      duplicatePromptIds: duplicates(promptMoves, (entry) => entry.promptId),
      duplicatePrimaryMoveFromPosition: duplicates(promptMoves, (entry) => `${entry.positionId}:${entry.primaryMoveId}`),
      genericSpamText: requiredText.some((value) => spamPattern.test(value))
    }
  }
})

const summary = {
  courseId: pack.courseId,
  contentVersion: pack.contentVersion,
  perspective: pack.perspective,
  counts: {
    chapters: (pack.chapters ?? []).length,
    lessons: (pack.lessons ?? []).length,
    lessonEdges: (pack.lessonEdges ?? []).length,
    prompts: (pack.prompts ?? []).length,
    positions: (pack.positions ?? []).length,
    moves: (pack.moves ?? []).length,
    notes: (pack.notes ?? []).length,
    requiredActivities: lessons.reduce((sum, lesson) => sum + lesson.requiredCount, 0),
    requiredDecisions: lessons.reduce((sum, lesson) => sum + lesson.decisionCount, 0)
  },
  lessons,
  blockers: lessons.filter((lesson) => Object.values(lesson.flags).some((value) => Array.isArray(value) ? value.length > 0 : value === true))
}

fs.writeFileSync(outPath, `${JSON.stringify(summary, null, 2)}\n`)
console.log(JSON.stringify({
  courseId: summary.courseId,
  contentVersion: summary.contentVersion,
  perspective: summary.perspective,
  blockerCount: summary.blockers.length,
  counts: summary.counts
}, null, 2))
```

- [ ] **Step 3: Validate the full Reference pack**

Run:

```bash
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
  > /private/tmp/queens-gambit-reference-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/queens-gambit-reference-validation.json
```

Expected:

- `courseId` is `mco15-queens-gambit-white`.
- `contentVersion` is `1.0.0`.
- `counts.lessons` is between `38` and `46`.
- `counts.warnings` is `0`.
- `missing` is `0`.
- `unexpected` is `0`.

- [ ] **Step 4: Run the private course-flow audit**

Run:

```bash
node --check /private/tmp/queens-gambit-course-audit.mjs
node /private/tmp/queens-gambit-course-audit.mjs \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
  /private/tmp/queens-gambit-reference-audit.json
jq '{courseId,contentVersion,perspective,counts,blockerCount:(.blockers|length),blockers:[.blockers[] | {lessonId,flags}]}' \
  /private/tmp/queens-gambit-reference-audit.json
```

Expected: `blockerCount` is `0`. If blockers appear, fix the private course by reducing required recaps/duplicate decisions or moving dense material to optional Reference, then rerun validation and audit.

- [ ] **Step 5: Inspect lesson counts without printing private prose**

Run:

```bash
jq '.lessons[] | {
  lessonId,
  chapterId,
  minimumDepth,
  required: ([.activities[] | select(.required != false)] | length),
  decisions: ([.activities[] | select(.required != false and .kind == "decision")] | length),
  activities: [.activities[] | {activityId,kind,required}]
}' "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse"
```

Expected:

- Quick lessons mostly show `required` between `2` and `3`.
- Standard lessons mostly show `required` between `3` and `4`.
- No lesson prints private source prose.

- [ ] **Step 6: Checkpoint Reference complete**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v1-reference"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json" "$CHECKPOINT/mco15-queens-gambit-white.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" "$CHECKPOINT/mco15-queens-gambit-white.ctcourse"
cp -p /private/tmp/queens-gambit-reference-validation.json "$CHECKPOINT/validation.json"
cp -p /private/tmp/queens-gambit-reference-audit.json "$CHECKPOINT/audit.json"
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  "$CHECKPOINT/validation.json" > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/"*.json "$CHECKPOINT/"*.ctcourse > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
```

Expected: every checksum prints `OK`.

---

### Task 7: Import Queen's Gambit and verify app-level behavior

**Files:**
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse`
- Create: `.superpowers/tmp/queens-gambit-import-smoke.go`
- Create: `/private/tmp/queens-gambit-import-smoke.json`
- Create: `/private/tmp/queens-gambit-default-import.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v1-final/`

**Interfaces:**
- Consumes: fully validated Queen's Gambit private course pack.
- Produces: active local app catalogue containing Italian, Ruy Lopez, Queen's Gambit, and Caro-Kann.

- [ ] **Step 1: Validate all four private course packs**

Run:

```bash
for pack in \
  "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse"
do
  go run ./cmd/coursepack validate "$pack" \
    | jq '{courseId,contentVersion,counts:{lessons:.counts.lessons,activities:.counts.activities,prompts:.counts.prompts,warnings:.counts.warnings},missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}'
done
```

Expected: every object has `missing: 0`, `unexpected: 0`; Queen's Gambit has `contentVersion: "1.0.0"` and `warnings: 0`.

- [ ] **Step 2: Create the ignored import smoke harness**

Run:

```bash
mkdir -p .superpowers/tmp
git check-ignore .superpowers .superpowers/tmp
```

Expected: both paths are ignored.

Create `.superpowers/tmp/queens-gambit-import-smoke.go` with this exact content:

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"chess-trainer/internal/app"
	"chess-trainer/internal/storage"
)

func main() {
	ctx := context.Background()
	var paths storage.Paths
	if os.Getenv("QG_USE_DEFAULT_DATA") == "1" {
		var err error
		paths, err = storage.DefaultPaths()
		if err != nil {
			panic(err)
		}
	} else {
		dataRoot := os.Getenv("QG_DATA_ROOT")
		if dataRoot == "" {
			fmt.Fprintln(os.Stderr, "QG_DATA_ROOT or QG_USE_DEFAULT_DATA=1 is required")
			os.Exit(2)
		}
		paths = storage.PathsAt(dataRoot)
	}
	coursePaths := strings.Split(os.Getenv("QG_COURSES"), "|")
	if len(coursePaths) == 0 || coursePaths[0] == "" {
		fmt.Fprintln(os.Stderr, "QG_COURSES is required")
		os.Exit(2)
	}

	services, err := app.Open(paths)
	if err != nil {
		panic(err)
	}
	defer services.Close()

	results := make([]map[string]any, 0, len(coursePaths))
	for _, coursePath := range coursePaths {
		inspection, err := services.CourseImporter.Inspect(ctx, coursePath)
		if err != nil {
			panic(err)
		}
		report, err := services.CourseImporter.Import(ctx, inspection, nil)
		if err != nil {
			panic(err)
		}
		results = append(results, map[string]any{
			"path":       coursePath,
			"inspection": inspection,
			"report":     report,
		})
	}
	home, err := services.Openings.Home(ctx)
	if err != nil {
		panic(err)
	}
	activeVersions := map[string]string{}
	for _, course := range home.Courses {
		compiled, err := services.OpeningCatalog.LoadActive(ctx, course.CourseID)
		if err != nil {
			panic(err)
		}
		activeVersions[course.CourseID] = compiled.Pack.ContentVersion
	}

	encoded := json.NewEncoder(os.Stdout)
	encoded.SetIndent("", "  ")
	_ = encoded.Encode(map[string]any{
		"dataRoot":       paths.Root,
		"imports":        results,
		"home":           home,
		"activeVersions": activeVersions,
	})
}
```

- [ ] **Step 3: Run disposable import smoke**

Run:

```bash
QG_ROOT="$(mktemp -d)"
QG_DATA_ROOT="$QG_ROOT" \
QG_COURSES="/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse|/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse|/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse|/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse" \
go run .superpowers/tmp/queens-gambit-import-smoke.go \
  > /private/tmp/queens-gambit-import-smoke.json
jq '{
  imports: [.imports[] | {sourceId:.inspection.sourceId, accepted:.report.accepted}],
  activeVersions,
  courses: [.home.courses[] | {courseId,title,perspective,totalLessons,depth}]
}' /private/tmp/queens-gambit-import-smoke.json
```

Expected:

- imports include all four course IDs;
- every `accepted` value is `1`;
- `activeVersions["mco15-queens-gambit-white"]` is `"1.0.0"`;
- disposable home includes four active courses.

- [ ] **Step 4: Import Queen's Gambit into the default local catalogue**

Before importing into default data, close any running Chess Trainer app instance. Then run:

```bash
QG_USE_DEFAULT_DATA=1 \
QG_COURSES="/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
go run .superpowers/tmp/queens-gambit-import-smoke.go \
  > /private/tmp/queens-gambit-default-import.json
jq '{
  imports: [.imports[] | {sourceId:.inspection.sourceId, sourceName:.inspection.sourceName, accepted:.report.accepted}],
  activeVersions,
  courses: [.home.courses[] | {courseId,title,perspective,totalLessons,depth}]
}' /private/tmp/queens-gambit-default-import.json
```

Expected:

- Queen's Gambit import has `accepted: 1`.
- `activeVersions["mco15-queens-gambit-white"]` is `"1.0.0"`.
- Italian, Ruy Lopez, and Caro-Kann remain active.

- [ ] **Step 5: Run UI acceptance for the opening flow**

Run:

```bash
npm --prefix frontend run test:e2e -- frontend/tests/openings.spec.ts --project=chromium
npm --prefix frontend run test:e2e -- frontend/tests/openings.spec.ts --project=webkit
```

Expected: both projects pass. These synthetic tests cover teaching-tree selection, course-specific journey flow, scrollable lesson details, and deep variation explorer board visibility.

- [ ] **Step 6: Record private final checkpoint and UI acceptance**

Create `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v1-final/ui-acceptance.md` with:

```markdown
# Queen's Gambit v1 UI acceptance

- Course ID: mco15-queens-gambit-white
- Content version: 1.0.0
- Disposable import smoke: passed
- Default catalogue import: accepted 1 replacement for mco15-queens-gambit-white
- Existing active courses preserved: mco15-italian-white, mco15-ruy-lopez-white, mco15-caro-kann-black
- Opening E2E Chromium: passed
- Opening E2E WebKit: passed
- Privacy: recorded only IDs, versions, counts, and UI-state facts
```

Then run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v1-final"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json" "$CHECKPOINT/mco15-queens-gambit-white.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" "$CHECKPOINT/mco15-queens-gambit-white.ctcourse"
cp -p /private/tmp/queens-gambit-reference-validation.json "$CHECKPOINT/validation.json"
cp -p /private/tmp/queens-gambit-reference-audit.json "$CHECKPOINT/audit.json"
cp -p /private/tmp/queens-gambit-import-smoke.json "$CHECKPOINT/import-smoke.json"
cp -p /private/tmp/queens-gambit-default-import.json "$CHECKPOINT/default-import.json"
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  "$CHECKPOINT/validation.json" > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/"*.json "$CHECKPOINT/"*.ctcourse "$CHECKPOINT/"*.md > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
```

Expected: every checksum prints `OK`.

---

### Task 8: Rebuild, verify, privacy-check, and prepare handoff

**Files:**
- Verify: repository public diff
- Verify: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v1-final/`
- Build: `build/bin/Chess Trainer.app`

**Interfaces:**
- Consumes: imported Queen's Gambit v1 private course and any public generic fixes.
- Produces: rebuilt app, clean privacy boundary, and merge-ready branch.

- [ ] **Step 1: Run full public repository verification**

Run:

```bash
go test ./...
go vet ./...
npm --prefix frontend run check
npm --prefix frontend test -- --run --single-thread
npm --prefix frontend run test:e2e -- frontend/tests/openings.spec.ts --project=chromium
npm --prefix frontend run test:e2e -- frontend/tests/openings.spec.ts --project=webkit
npm --prefix frontend run build
git diff --check
```

Expected: every command exits `0`.

- [ ] **Step 2: Rebuild the macOS app**

Run:

```bash
GOWORK=off GOTOOLCHAIN=local go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build -clean -trimpath
codesign --verify --deep --strict --verbose=2 "build/bin/Chess Trainer.app"
file "build/bin/Chess Trainer.app/Contents/MacOS/Chess Trainer"
```

Expected: Wails build exits `0`; codesign verification passes; `file` reports Mach-O 64-bit executable arm64.

- [ ] **Step 3: Restore generated whitespace-only binding churn if Wails introduces it**

Run:

```bash
git diff -- frontend/wailsjs/go/models.ts | sed -n '1,80p'
```

If the diff is whitespace-only generated churn, run:

```bash
git restore --source=HEAD -- frontend/wailsjs/go/models.ts
```

Expected: no unrelated generated binding diff remains.

- [ ] **Step 4: Verify privacy boundaries**

Run:

```bash
git status --short
git diff --name-only
git ls-files '*.ctcourse'
find "/Users/admin/Documents/Private Chess Courses" -maxdepth 2 -type f \
  \( -name '*queens-gambit*' -o -name '*mco15-queens*' \) | sort
```

Expected:

- public Git diff contains only the design/plan and any generic app/test fixes;
- no private Queen's Gambit course file appears in `git status`;
- tracked `.ctcourse` files are only synthetic fixtures under `internal/openings/testdata/`;
- private course, authoring, checkpoint, validation, and audit files remain under `/Users/admin/Documents/Private Chess Courses/` or `/private/tmp/`.

- [ ] **Step 5: Run strict review only if public code changed**

If public app/test code changed, run the thermo-nuclear review workflow against the public diff. The review must find:

- no course-specific special cases;
- no private source prose;
- no large-file or spaghetti regression;
- generic synthetic tests for any app behavior change.

If only docs changed publicly and all course edits are private, record that no public code review was needed beyond verification and privacy checks.

- [ ] **Step 6: Commit new public fixes only**

Run:

```bash
git status --short
```

If generic public fixes were required, commit those in separate focused commits with messages that describe the generic behavior. The design and implementation plan are expected to already be committed before execution starts, so do not create an empty docs commit. Do not stage private course files, rendered PDF pages, private scripts outside the repository, or ignored `.superpowers/tmp` files.

Expected: either there are no public changes to commit, or only reviewed generic public fixes are committed. Private files are not staged.

- [ ] **Step 7: Final handoff**

Report:

- final private Queen's Gambit course version and validation counts;
- checkpoint path;
- import status;
- app rebuild status;
- verification commands that passed;
- public commit list;
- confirmation that private course files were not committed.

Then offer the standard finishing options for the branch.

---

## Plan Self-Review Notes

- Spec coverage: Tasks cover private page rendering, offset verification, authoring inventory, Quick/Standard/Reference lesson construction, source coverage validation, lesson-flow hardening, four-course import, UI acceptance, app rebuild, and Git privacy boundaries.
- Privacy: The plan includes only high-level opening family names, public file paths, IDs, commands, and validation shapes. Private source prose and dense source lines remain in private files.
- Scope: Public code changes are gated behind generic defects and synthetic tests; the expected implementation is private course authoring plus import/rebuild.
