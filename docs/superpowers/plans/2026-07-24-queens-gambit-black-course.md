# Queen's Gambit Defences for Black Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build and import a private, reference-heavy `mco15-queens-gambit-black` course that teaches Black's QGD Orthodox and Slav/Semi-Slav defences against the Queen's Gambit.

**Architecture:** The course is a private schema-v2 `.ctcourse` pack backed by a private Black authoring inventory. It reuses the Queen's Gambit White page/render scope but creates Black-perspective notes, prompts, lesson tree edges, source coverage, validation outputs, and checkpoints outside Git.

**Tech Stack:** Go coursepack validator/importer, schema-v2 opening course JSON, Node.js audit scripts, `jq`, private local course files, Playwright opening E2E.

## Global Constraints

- Course title: `Queen's Gambit Defences for Black`
- Course ID: `mco15-queens-gambit-black`
- Content version: `1.0.0`
- Perspective: `black`
- Authoring file: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.authoring.json`
- Course pack: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse`
- Reuse Queen's Gambit White source/page work where possible, but create a separate Black authoring inventory.
- QGD Orthodox is the main QGD spine.
- Tartakower is included as a branch after the Orthodox skeleton is understood.
- Slav proper is the main Slav/Semi-Slav spine.
- Semi-Slav is a major branch.
- Side weapons are Reference-only unless source scope proves they belong cleanly in the selected Queen's Gambit material.
- Board orientation defaults to Black.
- Black repertoire moves are learner decisions.
- White moves appear as opponent pressure, demonstrations, or comparisons.
- Quick lessons generally have 2-3 required activities.
- Standard lessons generally have 3-4 required activities.
- Reference depth is rich but optional.
- Coverage validation must have no missing or unexpected records.
- Course warnings remain at zero unless a deliberately user-facing warning is truly useful.
- Do not add generic spam such as `Source note x records...`, `Course move found`, or raw activity-result text.
- No private source prose, rendered pages, authoring inventory, course pack, import outputs, or checkpoints are committed to Git.

---

## File Structure

- `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.authoring.json`
  - Private Black-perspective source inventory, rendered-page metadata, coverage records, teaching node map, and teaching tree intent.
- `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse`
  - Private schema-v2 course pack imported by the app.
- `/private/tmp/queens-gambit-black-pages/`
  - Optional rendered page workspace if the existing Queen's Gambit White rendered pages are unavailable.
- `/private/tmp/queens-gambit-black-inventory-check.mjs`
  - Private authoring-inventory checker.
- `/private/tmp/queens-gambit-black-course-audit.mjs`
  - Private anti-spam, prompt-duplication, lesson-shape, and Black-perspective audit.
- `/private/tmp/queens-gambit-black-*.json`
  - Private validation, audit, and import smoke outputs.
- `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-black-v1-*/`
  - Private task checkpoints with SHA256 manifests.
- `.superpowers/tmp/queens-gambit-black-import-smoke.go`
  - Ignored Go smoke harness for disposable/default import checks.
- `docs/superpowers/specs/2026-07-24-queens-gambit-black-course-design.md`
  - Approved design.
- `docs/superpowers/plans/2026-07-24-queens-gambit-black-course.md`
  - This implementation plan.

---

### Task 1: Validate source scope and create private baseline

**Files:**
- Read: `/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse`
- Create: `/private/tmp/queens-gambit-black-source-scope.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-black-baseline-before-v1/`

**Interfaces:**
- Consumes: existing Queen's Gambit White rendered/source scope and private course pack.
- Produces: verified source-scope JSON and baseline checkpoint that later tasks rely on.

- [ ] **Step 1: Verify the existing Queen's Gambit White inventory and pack are available**

Run:

```bash
test -f "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json"
test -f "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse"
jq '{courseId,targetContentVersion,coverageRecords:(.inventory.coverageRecords|length),teachingNodes:(.teachingNodes|length),families:(.scope.families|map(.familyId))}' \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json"
go run ./cmd/coursepack validate "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
  | jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}'
```

Expected:

- White authoring inventory exists.
- White course pack exists.
- White authoring `coverageRecords` is `42`.
- White course validation has `missing: 0` and `unexpected: 0`.

- [ ] **Step 2: Verify Queen's Gambit page boundary**

Run:

```bash
node - <<'JS' > /private/tmp/queens-gambit-black-source-scope.json
const fs = require('fs')
const whiteInventoryPath = '/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json'
const white = JSON.parse(fs.readFileSync(whiteInventoryPath, 'utf8'))
const pages = white.inventory.pages ?? []
const coverage = white.inventory.coverageRecords ?? []
const summary = {
  sourcePdf: '/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf',
  reusedFrom: whiteInventoryPath,
  courseId: 'mco15-queens-gambit-black',
  targetContentVersion: '1.0.0',
  pdfFirst: Math.min(...pages.map((page) => page.pdfPage)),
  pdfLastRendered: Math.max(...pages.map((page) => page.pdfPage)),
  printedFirst: Math.min(...pages.map((page) => page.printedPage)),
  printedLast: Math.max(...pages.map((page) => page.printedPage)),
  excludedPdfPages: white.scope.excludedPdfPages ?? [],
  families: white.scope.families,
  coverageRecords: coverage.length,
  coverageByFamily: Object.fromEntries(
    [...new Set(coverage.map((record) => record.familyId))]
      .sort()
      .map((familyId) => [familyId, coverage.filter((record) => record.familyId === familyId).length])
  ),
  sideWeaponScope: {
    tarrasch: coverage.some((record) => record.familyId === 'tarrasch'),
    chigorin: coverage.some((record) => record.familyId === 'chigorin'),
    albin: coverage.some((record) => /albin/i.test(record.coverageId)),
    budapest: coverage.some((record) => /budapest/i.test(record.coverageId))
  }
}
console.log(JSON.stringify(summary, null, 2))
JS
jq . /private/tmp/queens-gambit-black-source-scope.json
```

Expected:

- `pdfFirst` is `406`.
- `pdfLastRendered` is `509`.
- `printedFirst` is `389`.
- `coverageRecords` is `42`.
- `sideWeaponScope.tarrasch` and `sideWeaponScope.chigorin` are `true`.
- `sideWeaponScope.albin` and `sideWeaponScope.budapest` determine whether those systems are included later; if `false`, they remain out of v1.

- [ ] **Step 3: Create baseline checkpoint**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-black-baseline-before-v1"
mkdir -p "$CHECKPOINT"
cp -p /private/tmp/queens-gambit-black-source-scope.json "$CHECKPOINT/source-scope.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json" "$CHECKPOINT/mco15-queens-gambit-white.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" "$CHECKPOINT/mco15-queens-gambit-white.ctcourse"
shasum -a 256 "$CHECKPOINT/"*.json "$CHECKPOINT/"*.ctcourse > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
git status --short
```

Expected:

- Every checksum prints `OK`.
- `git status --short` shows no tracked private file changes.

---

### Task 2: Build Black authoring inventory

**Files:**
- Create: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.authoring.json`
- Create: `/private/tmp/queens-gambit-black-inventory-check.mjs`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-black-v1-authoring-inventory/`

**Interfaces:**
- Consumes: `/private/tmp/queens-gambit-black-source-scope.json` and White authoring inventory.
- Produces: Black authoring inventory with coverage records and teaching node map.

- [ ] **Step 1: Create Black authoring inventory**

Create `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.authoring.json` as a UTF-8 JSON object with this top-level structure:

```json
{
  "courseId": "mco15-queens-gambit-black",
  "targetContentVersion": "1.0.0",
  "paths": {
    "sourcePdf": "/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf",
    "whiteAuthoringInventory": "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json",
    "coursePack": "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse"
  },
  "renderedPdfPages": {},
  "scope": {
    "perspective": "black",
    "reusedQueenGambitWhiteScope": true,
    "families": [],
    "excludedPdfPages": [],
    "sideWeaponPolicy": {
      "tarrasch": "reference",
      "chigorin": "reference",
      "albin": "include-only-if-source-scope-supports",
      "budapest": "include-only-if-source-scope-supports"
    }
  },
  "inventory": {
    "pages": [],
    "coverageRecords": []
  },
  "teachingNodes": [],
  "teachingEdges": [],
  "noteOverrides": {}
}
```

Populate:

- `renderedPdfPages`, `scope.families`, `scope.excludedPdfPages`, `inventory.pages`, and `inventory.coverageRecords` from the White Queen's Gambit inventory.
- Black-specific `teachingNodes` with 42-52 nodes.
- Black-specific `teachingEdges` with exactly one root.
- `noteOverrides` for Black-perspective private summaries when a White summary is misleading from Black's side.

Required `teachingNodes` IDs:

```text
qgb-foundations
qgb-black-family-map
qgb-move-order-survival
qgd-black-orthodox-spine
qgd-black-bg5-response
qgd-black-exchange-structure
qgd-black-lasker-simplification
qgd-black-tartakower-branch
qgd-black-cambridge-ragozin-vienna
qgd-black-semi-tarrasch-recognition
slav-black-main-structure
slav-black-bishop-before-e6
slav-black-central-tension
slav-black-queenside-structure
semi-slav-black-wall
semi-slav-black-meran
semi-slav-black-anti-meran
semi-slav-black-tactical-reference
tarrasch-black-iqp-activity
chigorin-black-piece-pressure
```

Expected:

- `courseId` is `mco15-queens-gambit-black`.
- `targetContentVersion` is `1.0.0`.
- all coverage records from the White inventory are preserved.
- every teaching node has `lessonId`, `familyId`, `minimumDepth`, `coverageIds`, `blackIntent`, and `learnerDecision`.
- no public Git file contains private source prose.

- [ ] **Step 2: Create inventory checker**

Create `/private/tmp/queens-gambit-black-inventory-check.mjs` with this exact content:

```js
import fs from 'node:fs'

const [path] = process.argv.slice(2)
if (!path) {
  console.error('usage: node queens-gambit-black-inventory-check.mjs <authoring.json>')
  process.exit(2)
}

const authoring = JSON.parse(fs.readFileSync(path, 'utf8'))
const required = [
  'qgb-foundations',
  'qgb-black-family-map',
  'qgb-move-order-survival',
  'qgd-black-orthodox-spine',
  'qgd-black-bg5-response',
  'qgd-black-exchange-structure',
  'qgd-black-lasker-simplification',
  'qgd-black-tartakower-branch',
  'qgd-black-cambridge-ragozin-vienna',
  'qgd-black-semi-tarrasch-recognition',
  'slav-black-main-structure',
  'slav-black-bishop-before-e6',
  'slav-black-central-tension',
  'slav-black-queenside-structure',
  'semi-slav-black-wall',
  'semi-slav-black-meran',
  'semi-slav-black-anti-meran',
  'semi-slav-black-tactical-reference',
  'tarrasch-black-iqp-activity',
  'chigorin-black-piece-pressure'
]
const allowedDepths = new Set(['quick', 'standard', 'reference'])
const coverage = new Map((authoring.inventory?.coverageRecords ?? []).map((record) => [record.coverageId, record]))
const nodes = new Map((authoring.teachingNodes ?? []).map((node) => [node.lessonId, node]))
const incoming = new Map()
const outgoing = new Map()
for (const edge of authoring.teachingEdges ?? []) {
  incoming.set(edge.toLessonId, (incoming.get(edge.toLessonId) ?? 0) + 1)
  outgoing.set(edge.fromLessonId, [...(outgoing.get(edge.fromLessonId) ?? []), edge.toLessonId])
}
const roots = [...nodes.keys()].filter((id) => (incoming.get(id) ?? 0) === 0)
const seen = new Set()
const stack = [...roots]
while (stack.length) {
  const id = stack.pop()
  if (seen.has(id)) continue
  seen.add(id)
  for (const child of outgoing.get(id) ?? []) stack.push(child)
}
const badNodes = [...nodes.values()].filter((node) =>
  !allowedDepths.has(node.minimumDepth) ||
  !Array.isArray(node.coverageIds) ||
  node.coverageIds.length === 0 ||
  node.coverageIds.some((id) => !coverage.has(id)) ||
  typeof node.blackIntent !== 'string' ||
  node.blackIntent.trim() === '' ||
  typeof node.learnerDecision !== 'string' ||
  node.learnerDecision.trim() === ''
)
const result = {
  ok: authoring.courseId === 'mco15-queens-gambit-black' &&
    authoring.targetContentVersion === '1.0.0' &&
    coverage.size >= 40 &&
    nodes.size >= 42 &&
    nodes.size <= 52 &&
    roots.length === 1 &&
    roots[0] === 'qgb-foundations' &&
    seen.size === nodes.size &&
    badNodes.length === 0 &&
    required.every((id) => nodes.has(id)),
  courseId: authoring.courseId,
  targetContentVersion: authoring.targetContentVersion,
  coverageRecords: coverage.size,
  teachingNodes: nodes.size,
  teachingEdges: (authoring.teachingEdges ?? []).length,
  roots,
  reachable: seen.size,
  badNodes: badNodes.map((node) => node.lessonId),
  missingRequired: required.filter((id) => !nodes.has(id))
}
console.log(JSON.stringify(result, null, 2))
if (!result.ok) process.exit(1)
```

- [ ] **Step 3: Run checker and checkpoint inventory**

Run:

```bash
node --check /private/tmp/queens-gambit-black-inventory-check.mjs
node /private/tmp/queens-gambit-black-inventory-check.mjs \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.authoring.json"
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-black-v1-authoring-inventory"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.authoring.json" "$CHECKPOINT/mco15-queens-gambit-black.authoring.json"
node /private/tmp/queens-gambit-black-inventory-check.mjs \
  "$CHECKPOINT/mco15-queens-gambit-black.authoring.json" > "$CHECKPOINT/inventory-summary.json"
shasum -a 256 "$CHECKPOINT/"*.json > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
git status --short
```

Expected:

- checker exits `0`.
- `teachingNodes` is between `42` and `52`.
- root is `qgb-foundations`.
- every checksum prints `OK`.
- no private files are tracked by Git.

---

### Task 3: Build Quick Black course spine

**Files:**
- Create/update: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.authoring.json`
- Create: `/private/tmp/queens-gambit-black-quick-validation.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-black-v1-quick-spine/`

**Interfaces:**
- Consumes: valid Black authoring inventory.
- Produces: valid schema-v2 Quick-depth Black repertoire spine.

- [ ] **Step 1: Verify Black-perspective SAN helper from the initial position**

Run:

```bash
go run ./cmd/coursepack sanline \
  --fen "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1" \
  d4 d5 c4 e6 \
  | jq '{startFen, finalFen, uci:[.moves[].uci]}'
go run ./cmd/coursepack sanline \
  --fen "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1" \
  d4 d5 c4 c6 \
  | jq '{startFen, finalFen, uci:[.moves[].uci]}'
```

Expected:

```json
{"uci":["d2d4","d7d5","c2c4","e7e6"]}
```

and:

```json
{"uci":["d2d4","d7d5","c2c4","c7c6"]}
```

- [ ] **Step 2: Create schema-v2 course root**

Create `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse` as a UTF-8 JSON object with:

```json
{
  "schemaVersion": 2,
  "courseId": "mco15-queens-gambit-black",
  "contentVersion": "1.0.0",
  "title": "Queen's Gambit Defences for Black",
  "description": "A Black repertoire course for the Queen's Gambit Declined, Slav, and Semi-Slav defences.",
  "perspective": "black",
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

- [ ] **Step 3: Add Quick chapters**

Set `chapters` to:

```json
[
  {"chapterId":"foundations","title":"Black's Queen's Gambit Foundations","ordinal":1,"overview":"Learn how Black chooses between declining, supporting, and countering White's c-pawn pressure.","minimumDepth":"quick"},
  {"chapterId":"qgd","title":"Queen's Gambit Declined","ordinal":2,"overview":"Use the Orthodox setup as Black's main solid equalizing spine.","minimumDepth":"quick"},
  {"chapterId":"slav","title":"Slav and Semi-Slav","ordinal":3,"overview":"Use ...c6 structures to support d5 and choose when the light bishop leaves the chain.","minimumDepth":"quick"},
  {"chapterId":"side-weapons","title":"Reference Side Weapons","ordinal":4,"overview":"Recognize active side systems without making them the main repertoire.","minimumDepth":"reference"}
]
```

- [ ] **Step 4: Author Quick move graph**

Create connected positions and moves for:

```text
initial
1.d4
1...d5
2.c4
2...e6
2...c6
2...dxc4 as reference recognition
2...Nc6 as reference recognition
```

Rules:

- White moves use `trainingRole: "opponent"`.
- Black course moves use `trainingRole: "repertoire"`.
- Black accepted side alternatives use `trainingRole: "alternative"` only if they are not the primary repertoire at that position.
- Every move has a `sourceRef` coverage ID from the Black authoring inventory.
- Every used coverage ID is in `sourceCoverage.expectedReferences`.

- [ ] **Step 5: Author Quick lessons**

Required Quick lessons:

```text
qgb-foundations
qgb-black-family-map
qgb-move-order-survival
qgd-black-orthodox-spine
slav-black-main-structure
```

Each Quick lesson uses one of these shapes:

```json
[
  {"kind":"concept","required":true},
  {"kind":"decision","required":true},
  {"kind":"recap","required":false}
]
```

or:

```json
[
  {"kind":"concept","required":true},
  {"kind":"demonstration","required":true},
  {"kind":"comparison","required":true},
  {"kind":"recap","required":false}
]
```

Expected:

- 5-7 Quick lessons.
- every Quick lesson has 2-3 required activities.
- at most one required decision per Quick lesson.
- no repeated forced White move from the same position.

- [ ] **Step 6: Author Quick teaching tree**

Create `lessonEdges` so:

```text
qgb-foundations
└── qgb-black-family-map
    ├── qgb-move-order-survival
    ├── qgd-black-orthodox-spine
    └── slav-black-main-structure
```

Requirements:

- `qgb-foundations` is the single root.
- family split edges use `kind: "alternative"` with labels `QGD: ...e6` and `Slav: ...c6`.
- every Quick lesson is reachable at Quick depth.

- [ ] **Step 7: Validate and checkpoint Quick spine**

Run:

```bash
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse" \
  > /private/tmp/queens-gambit-black-quick-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/queens-gambit-black-quick-validation.json
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-black-v1-quick-spine"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.authoring.json" "$CHECKPOINT/mco15-queens-gambit-black.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse" "$CHECKPOINT/mco15-queens-gambit-black.ctcourse"
cp -p /private/tmp/queens-gambit-black-quick-validation.json "$CHECKPOINT/validation.json"
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  "$CHECKPOINT/validation.json" > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/"*.json "$CHECKPOINT/"*.ctcourse > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
```

Expected:

- validation exits `0`.
- course perspective is `black`.
- `counts.lessons` is between `5` and `7`.
- `counts.warnings` is `0`.
- `missing` is `0`.
- `unexpected` is `0`.
- every checksum prints `OK`.

---

### Task 4: Build QGD Orthodox reference-heavy branch

**Files:**
- Update: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse`
- Read/update: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.authoring.json`
- Create: `/private/tmp/queens-gambit-black-qgd-validation.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-black-v1-qgd/`

**Interfaces:**
- Consumes: valid Quick spine.
- Produces: Black QGD Orthodox, Exchange, Lasker, Tartakower, and recognition branches.

- [ ] **Step 1: Add QGD lesson jobs**

Add these lessons:

```text
qgd-black-bg5-response
qgd-black-orthodox-development
qgd-black-castle-safely
qgd-black-exchange-structure
qgd-black-exchange-minority-plan
qgd-black-lasker-simplification
qgd-black-tartakower-branch
qgd-black-cambridge-springs
qgd-black-ragozin-vienna-recognition
qgd-black-semi-tarrasch-recognition
```

Expected:

- QGD chapter has 11-14 lessons including Quick QGD.
- Standard QGD lessons teach Black decisions.
- Dense Cambridge/Ragozin/Vienna/Semi-Tarrasch material can be Reference.
- Tartakower is a branch after Orthodox placement, not the first Quick lesson.

- [ ] **Step 2: Add QGD move graph**

Include connected Black-perspective move paths for:

```text
1.d4 d5 2.c4 e6
3.Nc3 Nf6
4.Bg5 Be7
5.e3 O-O
6.Nf3 h6
...Lasker simplification route
...Tartakower ...b6/...Bb7 route
White cxd5 Exchange structure
Cambridge Springs recognition
Ragozin/Vienna recognition
Semi-Tarrasch recognition
```

Rules:

- Black moves on Black-to-move positions must not use `trainingRole: "opponent"`.
- White moves on White-to-move positions must use `trainingRole: "opponent"`.
- Only one Black repertoire move per Black-to-move position.
- Alternate Black systems from the same position use `trainingRole: "alternative"` only when they are accepted alternatives in prompts.

- [ ] **Step 3: Validate and checkpoint QGD**

Run:

```bash
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse" \
  > /private/tmp/queens-gambit-black-qgd-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/queens-gambit-black-qgd-validation.json
jq '{qgdLessons:([.lessons[] | select(.chapterId=="qgd")] | length), expectedReferences:(.sourceCoverage.expectedReferences|length)}' \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse"
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-black-v1-qgd"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.authoring.json" "$CHECKPOINT/mco15-queens-gambit-black.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse" "$CHECKPOINT/mco15-queens-gambit-black.ctcourse"
cp -p /private/tmp/queens-gambit-black-qgd-validation.json "$CHECKPOINT/validation.json"
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  "$CHECKPOINT/validation.json" > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/"*.json "$CHECKPOINT/"*.ctcourse > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
```

Expected:

- validation exits `0`.
- `qgdLessons` is between `11` and `14`.
- `counts.warnings` is `0`.
- `missing` is `0`.
- `unexpected` is `0`.
- every checksum prints `OK`.

---

### Task 5: Build Slav and Semi-Slav reference-heavy branch

**Files:**
- Update: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse`
- Read/update: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.authoring.json`
- Create: `/private/tmp/queens-gambit-black-slav-validation.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-black-v1-slav-semi-slav/`

**Interfaces:**
- Consumes: valid QGD-expanded pack.
- Produces: Slav proper and Semi-Slav branch coverage.

- [ ] **Step 1: Add Slav/Semi-Slav lesson jobs**

Add these lessons:

```text
slav-black-bishop-before-e6
slav-black-main-development
slav-black-central-tension
slav-black-queenside-structure
slav-black-exchange-recognition
semi-slav-black-wall
semi-slav-black-meran
semi-slav-black-anti-meran
semi-slav-black-tactical-reference
semi-slav-black-moscow-reference
semi-slav-black-geller-reference
```

Expected:

- Slav chapter has 12-16 lessons including Quick Slav.
- Standard lessons teach the playable Slav proper and Semi-Slav wall.
- Meran, Anti-Meran, Moscow, Geller, and tactical branches are Reference where dense.

- [ ] **Step 2: Add Slav/Semi-Slav move graph**

Include connected Black-perspective move paths for:

```text
1.d4 d5 2.c4 c6
3.Nf3 Nf6
4.Nc3 dxc4 / ...Bf5 recognition
...bishop-before-e6 structure
...central exchange structure
...Semi-Slav ...e6 wall
...Meran recognition
...Anti-Meran recognition
...Moscow/Geller/tactical Reference lines where source coverage exists
```

Rules:

- Black's main Slav proper move is the primary repertoire move in Slav lessons.
- Semi-Slav entry is a major branch, not a hidden note.
- Dense Semi-Slav tactical material uses optional Reference activities unless the lesson is explicitly Reference-depth.

- [ ] **Step 3: Validate and checkpoint Slav/Semi-Slav**

Run:

```bash
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse" \
  > /private/tmp/queens-gambit-black-slav-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/queens-gambit-black-slav-validation.json
jq '{lessons:(.lessons|length), slavLessons:([.lessons[] | select(.chapterId=="slav")] | length), referenceLessons:([.lessons[] | select(.minimumDepth=="reference")] | length), expectedReferences:(.sourceCoverage.expectedReferences|length)}' \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse"
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-black-v1-slav-semi-slav"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.authoring.json" "$CHECKPOINT/mco15-queens-gambit-black.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse" "$CHECKPOINT/mco15-queens-gambit-black.ctcourse"
cp -p /private/tmp/queens-gambit-black-slav-validation.json "$CHECKPOINT/validation.json"
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  "$CHECKPOINT/validation.json" > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/"*.json "$CHECKPOINT/"*.ctcourse > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
```

Expected:

- validation exits `0`.
- `slavLessons` is between `12` and `16`.
- total lessons is at least `28`.
- `counts.warnings` is `0`.
- `missing` is `0`.
- `unexpected` is `0`.
- every checksum prints `OK`.

---

### Task 6: Add side weapons, Reference coverage, and course-flow audit

**Files:**
- Update: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse`
- Read/update: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.authoring.json`
- Create: `/private/tmp/queens-gambit-black-course-audit.mjs`
- Create: `/private/tmp/queens-gambit-black-reference-validation.json`
- Create: `/private/tmp/queens-gambit-black-reference-audit.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-black-v1-reference/`

**Interfaces:**
- Consumes: valid QGD + Slav/Semi-Slav pack and complete authoring inventory.
- Produces: Reference-heavy v1 pack with anti-spam and Black-flow audit.

- [ ] **Step 1: Add side-weapon and full Reference coverage**

Add lessons/optional Reference cards for:

```text
tarrasch-black-iqp-activity
tarrasch-black-pressure-plan
tarrasch-black-endgame-risk
chigorin-black-piece-pressure
chigorin-black-center-pressure
chigorin-black-development-tradeoff
albin-black-recognition only if source-scope supports it
budapest-black-recognition only if source-scope supports it
```

Expected:

- Tarrasch and Chigorin are present as Reference side systems.
- Albin/Budapest are absent unless `/private/tmp/queens-gambit-black-source-scope.json` shows source support.
- all selected inventory coverage records are captured.
- total lessons is between `38` and `52`.
- optional Reference activities exist for dense table areas.

- [ ] **Step 2: Create course-flow audit script**

Create `/private/tmp/queens-gambit-black-course-audit.mjs` with this exact content:

```js
import fs from 'node:fs'

const [packPath, outPath] = process.argv.slice(2)
if (!packPath || !outPath) {
  console.error('usage: node queens-gambit-black-course-audit.mjs <course.ctcourse> <audit.json>')
  process.exit(2)
}

const pack = JSON.parse(fs.readFileSync(packPath, 'utf8'))
const prompts = new Map((pack.prompts ?? []).map((prompt) => [prompt.promptId, prompt]))
const moves = new Map((pack.moves ?? []).map((move) => [move.moveId, move]))
const positions = new Map((pack.positions ?? []).map((position) => [position.positionId, position]))
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
    primaryUci: primary?.uci ?? null,
    primaryRole: primary?.trainingRole ?? null
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

const incoming = new Map()
const children = new Map()
for (const edge of pack.lessonEdges ?? []) {
  incoming.set(edge.toLessonId, (incoming.get(edge.toLessonId) ?? 0) + 1)
  children.set(edge.fromLessonId, [...(children.get(edge.fromLessonId) ?? []), edge.toLessonId])
}
const lessonIds = new Set((pack.lessons ?? []).map((lesson) => lesson.lessonId))
const roots = [...lessonIds].filter((lessonId) => (incoming.get(lessonId) ?? 0) === 0)
const reachable = new Set()
const stack = [...roots]
while (stack.length) {
  const lessonId = stack.pop()
  if (reachable.has(lessonId)) continue
  reachable.add(lessonId)
  for (const child of children.get(lessonId) ?? []) stack.push(child)
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
    startPositionExists: positions.has(lesson.startPositionId),
    activityCount: (lesson.activities ?? []).length,
    requiredCount: required.length,
    decisionCount: decisions.length,
    promptMoves,
    flags: {
      invalidDepth: !allowedDepths.has(lesson.minimumDepth),
      hiddenFromTree: !reachable.has(lesson.lessonId),
      missingStartPosition: !positions.has(lesson.startPositionId),
      tooManyRequiredActivities: required.length > maxRequired,
      tooManyDecisions: decisions.length > 2,
      duplicatePromptIds: duplicates(promptMoves, (entry) => entry.promptId),
      duplicatePrimaryMoveFromPosition: duplicates(promptMoves, (entry) => `${entry.positionId}:${entry.primaryMoveId}`),
      nonBlackDecisionRole: promptMoves.filter((entry) => entry.primaryRole !== 'repertoire' && entry.primaryRole !== 'alternative'),
      genericSpamText: requiredText.some((value) => spamPattern.test(value))
    }
  }
})

const summary = {
  courseId: pack.courseId,
  contentVersion: pack.contentVersion,
  perspective: pack.perspective,
  rootLessons: roots,
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
  rootLessons: summary.rootLessons,
  counts: summary.counts
}, null, 2))
```

- [ ] **Step 3: Validate and audit Reference pack**

Run:

```bash
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse" \
  > /private/tmp/queens-gambit-black-reference-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/queens-gambit-black-reference-validation.json
node --check /private/tmp/queens-gambit-black-course-audit.mjs
node /private/tmp/queens-gambit-black-course-audit.mjs \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse" \
  /private/tmp/queens-gambit-black-reference-audit.json
jq '{courseId,contentVersion,perspective,counts,rootLessons,blockerCount:(.blockers|length),blockers:[.blockers[] | {lessonId,flags}]}' \
  /private/tmp/queens-gambit-black-reference-audit.json
```

Expected:

- validation exits `0`.
- `courseId` is `mco15-queens-gambit-black`.
- `contentVersion` is `1.0.0`.
- `perspective` is `black`.
- `counts.lessons` is between `38` and `52`.
- `counts.warnings` is `0`.
- `missing` is `0`.
- `unexpected` is `0`.
- audit `blockerCount` is `0`.
- `rootLessons` is exactly `["qgb-foundations"]`.

- [ ] **Step 4: Checkpoint Reference complete**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-black-v1-reference"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.authoring.json" "$CHECKPOINT/mco15-queens-gambit-black.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse" "$CHECKPOINT/mco15-queens-gambit-black.ctcourse"
cp -p /private/tmp/queens-gambit-black-reference-validation.json "$CHECKPOINT/validation.json"
cp -p /private/tmp/queens-gambit-black-reference-audit.json "$CHECKPOINT/audit.json"
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  "$CHECKPOINT/validation.json" > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/"*.json "$CHECKPOINT/"*.ctcourse > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
git status --short
```

Expected:

- every checksum prints `OK`.
- no private files are tracked by Git.

---

### Task 7: Import, app verification, and final checkpoint

**Files:**
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse`
- Create: `.superpowers/tmp/queens-gambit-black-import-smoke.go`
- Create: `.superpowers/tmp/queens-gambit-black-orientation-smoke.go`
- Create: `/private/tmp/queens-gambit-black-import-smoke.json`
- Create: `/private/tmp/queens-gambit-black-default-import.json`
- Create: `/private/tmp/queens-gambit-black-orientation-smoke.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-black-v1-final/`

**Interfaces:**
- Consumes: full validated Black course pack.
- Produces: active local app catalogue with five courses.

- [ ] **Step 1: Validate all five private course packs**

Run:

```bash
for pack in \
  "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse"
do
  go run ./cmd/coursepack validate "$pack" \
    | jq '{courseId,contentVersion,counts:{lessons:.counts.lessons,activities:.counts.activities,prompts:.counts.prompts,warnings:.counts.warnings},missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}'
done
```

Expected:

- every object has `missing: 0` and `unexpected: 0`.
- Queen's Gambit Black has `contentVersion: "1.0.0"` and `warnings: 0`.

- [ ] **Step 2: Create ignored import smoke harness**

Run:

```bash
mkdir -p .superpowers/tmp
git check-ignore .superpowers .superpowers/tmp
```

Expected:

- both paths are ignored.

Create `.superpowers/tmp/queens-gambit-black-import-smoke.go` with this exact content:

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
	if os.Getenv("QGB_USE_DEFAULT_DATA") == "1" {
		var err error
		paths, err = storage.DefaultPaths()
		if err != nil {
			panic(err)
		}
	} else {
		dataRoot := os.Getenv("QGB_DATA_ROOT")
		if dataRoot == "" {
			fmt.Fprintln(os.Stderr, "QGB_DATA_ROOT or QGB_USE_DEFAULT_DATA=1 is required")
			os.Exit(2)
		}
		paths = storage.PathsAt(dataRoot)
	}
	coursePaths := strings.Split(os.Getenv("QGB_COURSES"), "|")
	if len(coursePaths) == 0 || coursePaths[0] == "" {
		fmt.Fprintln(os.Stderr, "QGB_COURSES is required")
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
QGB_ROOT="$(mktemp -d)"
QGB_DATA_ROOT="$QGB_ROOT" \
QGB_COURSES="/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse|/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse|/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse|/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse|/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse" \
go run .superpowers/tmp/queens-gambit-black-import-smoke.go \
  > /private/tmp/queens-gambit-black-import-smoke.json
jq '{
  imports: [.imports[] | {sourceId:.inspection.sourceId, accepted:.report.accepted}],
  activeVersions,
  courses: [.home.courses[] | {courseId,title,perspective,totalLessons,depth}]
}' /private/tmp/queens-gambit-black-import-smoke.json
```

Expected:

- imports include all five course IDs.
- every `accepted` value is `1`.
- `activeVersions["mco15-queens-gambit-black"]` is `"1.0.0"`.
- disposable home includes five active courses.
- course list includes both White and Black Queen's Gambit courses with different perspectives.

- [ ] **Step 4: Import Queen's Gambit Black into default local catalogue**

Before importing into default data, close any running Chess Trainer app instance.

Run:

```bash
QGB_USE_DEFAULT_DATA=1 \
QGB_COURSES="/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse" \
go run .superpowers/tmp/queens-gambit-black-import-smoke.go \
  > /private/tmp/queens-gambit-black-default-import.json
jq '{
  imports: [.imports[] | {sourceId:.inspection.sourceId, sourceName:.inspection.sourceName, accepted:.report.accepted}],
  activeVersions,
  courses: [.home.courses[] | {courseId,title,perspective,totalLessons,depth}]
}' /private/tmp/queens-gambit-black-default-import.json
```

Expected:

- Queen's Gambit Black import has `accepted: 1`.
- `activeVersions["mco15-queens-gambit-black"]` is `"1.0.0"`.
- Italian, Ruy Lopez, Queen's Gambit White, and Caro-Kann remain active.

- [ ] **Step 5: Create and run Black orientation smoke**

Create `.superpowers/tmp/queens-gambit-black-orientation-smoke.go` with this exact content:

```go
package main

import (
	"context"
	"encoding/json"
	"os"

	"chess-trainer/internal/app"
	"chess-trainer/internal/storage"
)

func main() {
	ctx := context.Background()
	paths, err := storage.DefaultPaths()
	if err != nil {
		panic(err)
	}
	services, err := app.Open(paths)
	if err != nil {
		panic(err)
	}
	defer services.Close()

	compiled, err := services.OpeningCatalog.LoadActive(ctx, "mco15-queens-gambit-black")
	if err != nil {
		panic(err)
	}
	invalidPrimaryRoles := []map[string]string{}
	for promptID, prompt := range compiled.Prompts {
		move, ok := compiled.Moves[prompt.PrimaryMoveID]
		if !ok {
			invalidPrimaryRoles = append(invalidPrimaryRoles, map[string]string{
				"promptId": promptID,
				"moveId":   prompt.PrimaryMoveID,
				"role":     "missing",
			})
			continue
		}
		if move.TrainingRole != "repertoire" && move.TrainingRole != "alternative" {
			invalidPrimaryRoles = append(invalidPrimaryRoles, map[string]string{
				"promptId": promptID,
				"moveId":   prompt.PrimaryMoveID,
				"role":     string(move.TrainingRole),
			})
		}
	}
	result := map[string]any{
		"courseId":            compiled.Pack.CourseID,
		"contentVersion":      compiled.Pack.ContentVersion,
		"perspective":         compiled.Pack.Perspective,
		"lessons":             len(compiled.Lessons),
		"prompts":             len(compiled.Prompts),
		"invalidPrimaryRoles": invalidPrimaryRoles,
		"ok":                  compiled.Pack.CourseID == "mco15-queens-gambit-black" && compiled.Pack.ContentVersion == "1.0.0" && compiled.Pack.Perspective == "black" && len(invalidPrimaryRoles) == 0,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(result)
	if !result["ok"].(bool) {
		os.Exit(1)
	}
}
```

Run:

```bash
go run .superpowers/tmp/queens-gambit-black-orientation-smoke.go \
  > /private/tmp/queens-gambit-black-orientation-smoke.json
jq '{courseId,contentVersion,perspective,lessons,prompts,invalidPrimaryRoles,ok}' \
  /private/tmp/queens-gambit-black-orientation-smoke.json
```

Expected:

- `courseId` is `mco15-queens-gambit-black`.
- `contentVersion` is `1.0.0`.
- `perspective` is `black`.
- `invalidPrimaryRoles` is `[]`.
- `ok` is `true`.

- [ ] **Step 6: Run app and UI verification**

Run:

```bash
npm --prefix frontend run test:e2e -- frontend/tests/openings.spec.ts --project=chromium
npm --prefix frontend run test:e2e -- frontend/tests/openings.spec.ts --project=webkit
go test ./...
```

Expected:

- Chromium opening E2E passes.
- WebKit opening E2E passes.
- `go test ./...` passes.

- [ ] **Step 7: Record final checkpoint**

Create `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-black-v1-final/ui-acceptance.md` with:

```markdown
# Queen's Gambit Black v1 UI acceptance

- Course ID: mco15-queens-gambit-black
- Content version: 1.0.0
- Disposable import smoke: passed
- Default catalogue import: accepted 1 replacement for mco15-queens-gambit-black
- Existing active courses preserved: mco15-italian-white, mco15-ruy-lopez-white, mco15-queens-gambit-white, mco15-caro-kann-black
- Opening E2E Chromium: passed
- Opening E2E WebKit: passed
- Black orientation smoke: passed
- Go test suite: passed
- Privacy: recorded only IDs, versions, counts, and UI-state facts
```

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-black-v1-final"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.authoring.json" "$CHECKPOINT/mco15-queens-gambit-black.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse" "$CHECKPOINT/mco15-queens-gambit-black.ctcourse"
cp -p /private/tmp/queens-gambit-black-reference-validation.json "$CHECKPOINT/validation.json"
cp -p /private/tmp/queens-gambit-black-reference-audit.json "$CHECKPOINT/audit.json"
cp -p /private/tmp/queens-gambit-black-import-smoke.json "$CHECKPOINT/import-smoke.json"
cp -p /private/tmp/queens-gambit-black-default-import.json "$CHECKPOINT/default-import.json"
cp -p /private/tmp/queens-gambit-black-orientation-smoke.json "$CHECKPOINT/orientation-smoke.json"
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  "$CHECKPOINT/validation.json" > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/"*.json "$CHECKPOINT/"*.ctcourse "$CHECKPOINT/"*.md > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
git status --short --branch --untracked-files=all
```

Expected:

- every checksum prints `OK`.
- Git status shows no private course artifacts.
- final report can state the default local catalogue has five active courses.
