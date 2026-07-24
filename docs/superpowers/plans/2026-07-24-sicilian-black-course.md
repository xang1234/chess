# Sicilian Defence for Black Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build and import a private, full-reference `mco15-sicilian-black` course that teaches Black's Sicilian Defence with a Najdorf-centered learning spine and full Sicilian Reference tree.

**Architecture:** The course is a private schema-v2 `.ctcourse` pack backed by a private source inventory and authoring file. Quick and Standard teach a coherent Najdorf-centered Black repertoire; Reference captures the full selected Sicilian source scope as a browseable tree. The app should need no Sicilian-specific runtime code unless the large private pack exposes a generic opening-course defect.

**Tech Stack:** Go coursepack validator/importer, schema-v2 opening course JSON, Node.js private authoring/audit scripts, `jq`, Poppler `pdftoppm`, private local course files, Playwright opening E2E.

## Global Constraints

- Course title: `Sicilian Defence for Black`
- Course ID: `mco15-sicilian-black`
- Content version: `1.0.0`
- Perspective: `black`
- Default depth: `reference`
- Authoring file: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json`
- Course pack: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse`
- Source PDF: `/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf`
- Source printed pages: `244-361`
- Current PDF-page mapping: printed page plus `17`, so PDF pages `261-378`
- The Sicilian section begins after French Defence and ends before Pirc Defence.
- Main learning spine is Najdorf-centered.
- Quick teaches a playable Black route, not the full Sicilian.
- Standard teaches main practical decisions and common deviations.
- Reference contains the full selected Sicilian source-backed map.
- Board orientation defaults to Black.
- Black repertoire moves are learner decisions.
- White moves appear as opponent pressure, demonstrations, comparisons, or branch choices.
- Quick lessons usually have 2-3 required activities.
- Standard lessons usually have 3-4 required activities.
- Reference activities are optional by default.
- Required decisions do not repeat the same prompt from the same position inside one lesson.
- Do not add generic filler such as `Source note x records...`, raw activity-result JSON, or `Course move found`.
- Coverage validation must have no missing or unexpected records at final validation.
- Course warnings remain at zero unless a deliberately user-facing warning is truly useful.
- No private source prose, rendered pages, authoring inventory, course pack, import outputs, or checkpoints are committed to Git.

---

## File Structure

- `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json`
  - Private source inventory, rendered-page metadata, coverage records, family map, teaching node intent, and stable IDs.
- `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse`
  - Private schema-v2 course pack imported by the app.
- `/private/tmp/mco15-sicilian-pages/`
  - Private rendered Sicilian PDF-page workspace.
- `/private/tmp/mco15-sicilian-source-scope.json`
  - Private page/family boundary record used by inventory and checkpoints.
- `/private/tmp/mco15-sicilian-inventory-check.mjs`
  - Private inventory audit script.
- `/private/tmp/mco15-sicilian-course-audit.mjs`
  - Private course-flow, anti-spam, role, depth, and source-scope audit script.
- `/private/tmp/mco15-sicilian-*.json`
  - Private validation, audit, and import smoke outputs.
- `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-v1-*/`
  - Private task checkpoints with SHA256 manifests.
- `.superpowers/tmp/sicilian-black-import-smoke.go`
  - Ignored Go smoke harness for disposable and default app-catalogue imports.
- `.superpowers/tmp/sicilian-black-orientation-smoke.go`
  - Ignored Go smoke harness for Black perspective and prompt-role checks.
- `docs/superpowers/specs/2026-07-24-sicilian-black-full-reference-course-design.md`
  - Approved public design.
- `docs/superpowers/plans/2026-07-24-sicilian-black-course.md`
  - This public implementation plan.

---

### Task 1: Prepare implementation branch, render source scope, and checkpoint baseline

**Files:**
- Read: `/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf`
- Read: `/Users/admin/Documents/Private Chess Courses/*.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/*.authoring.json`
- Create: `/private/tmp/mco15-sicilian-pages/*.png`
- Create: `/private/tmp/mco15-sicilian-source-scope.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-baseline-before-v1/`

**Interfaces:**
- Consumes: approved spec and existing private course packs.
- Produces: isolated branch, rendered source pages, source-scope JSON, and recoverable baseline checkpoint.

- [ ] **Step 1: Create or enter the implementation branch**

Run from `/Users/admin/Documents/Work/chess`:

```bash
git status --short --branch
git switch -c codex/sicilian-black-full-reference-course
```

Expected:

- The initial status is clean.
- The current branch becomes `codex/sicilian-black-full-reference-course`.

If the branch already exists, run:

```bash
git switch codex/sicilian-black-full-reference-course
git status --short --branch
```

Expected: status is clean on `codex/sicilian-black-full-reference-course`.

- [ ] **Step 2: Verify required private inputs**

Run:

```bash
test -f "/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf"
for course in \
  "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse"
do
  test -f "$course"
  go run ./cmd/coursepack validate "$course" \
    | jq '{courseId,contentVersion,counts:{lessons:.counts.lessons,activities:.counts.activities,prompts:.counts.prompts},missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}'
done
```

Expected:

- The PDF exists.
- All five existing private packs validate.
- Every validation summary has `missing: 0` and `unexpected: 0`.

- [ ] **Step 3: Render Sicilian source pages privately**

Run:

```bash
mkdir -p /private/tmp/mco15-sicilian-pages
/Users/admin/.cache/codex-runtimes/codex-primary-runtime/dependencies/bin/override/pdftoppm \
  -f 261 -l 378 -png -r 180 \
  "/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf" \
  /private/tmp/mco15-sicilian-pages/sicilian
find /private/tmp/mco15-sicilian-pages -maxdepth 1 -type f -name 'sicilian-*.png' | sort | wc -l
```

Expected: `118`.

- [ ] **Step 4: Write source-scope summary**

Run:

```bash
node - <<'JS' > /private/tmp/mco15-sicilian-source-scope.json
const families = [
  { familyId: 'sicilian-foundations', title: 'Sicilian foundations', printedFirst: 244, printedLast: 245, minimumDepth: 'quick' },
  { familyId: 'najdorf', title: 'Najdorf Variation', printedFirst: 246, printedLast: 268, minimumDepth: 'quick' },
  { familyId: 'dragon-accelerated-dragon', title: 'Dragon and Accelerated Dragon', printedFirst: 269, printedLast: 289, minimumDepth: 'reference' },
  { familyId: 'scheveningen-e6-systems', title: 'Scheveningen and ...e6 systems', printedFirst: 290, printedLast: 317, minimumDepth: 'standard' },
  { familyId: 'classical-nc6-e5-systems', title: 'Classical, ...Nc6, and ...e5 systems', printedFirst: 318, printedLast: 345, minimumDepth: 'standard' },
  { familyId: 'anti-sicilians', title: 'Non-open Sicilians and anti-Sicilians', printedFirst: 346, printedLast: 361, minimumDepth: 'quick' }
]
const printedPages = []
for (let page = 244; page <= 361; page += 1) printedPages.push(page)
const pdfPages = printedPages.map((printedPage) => ({
  printedPage,
  pdfPage: printedPage + 17,
  imagePath: `/private/tmp/mco15-sicilian-pages/sicilian-${String(printedPage + 17).padStart(3, '0')}.png`,
  familyId: families.find((family) => printedPage >= family.printedFirst && printedPage <= family.printedLast).familyId
}))
const summary = {
  courseId: 'mco15-sicilian-black',
  title: 'Sicilian Defence for Black',
  perspective: 'black',
  targetContentVersion: '1.0.0',
  sourcePdf: '/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf',
  printedFirst: 244,
  printedLast: 361,
  pdfFirst: 261,
  pdfLast: 378,
  scanOffset: 17,
  pageCount: printedPages.length,
  families,
  pdfPages
}
console.log(JSON.stringify(summary, null, 2))
JS
jq '{courseId,title,perspective,printedFirst,printedLast,pdfFirst,pdfLast,pageCount,families:[.families[] | {familyId,printedFirst,printedLast}]}' \
  /private/tmp/mco15-sicilian-source-scope.json
```

Expected:

- `pageCount` is `118`.
- `printedFirst` is `244`.
- `printedLast` is `361`.
- `pdfFirst` is `261`.
- `pdfLast` is `378`.
- `families` contains exactly six objects with the IDs shown in the command.

- [ ] **Step 5: Checkpoint baseline**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-baseline-before-v1"
mkdir -p "$CHECKPOINT"
cp -p /private/tmp/mco15-sicilian-source-scope.json "$CHECKPOINT/source-scope.json"
for file in \
  "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.authoring.json" \
  "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.authoring.json" \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json" \
  "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.authoring.json" \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.authoring.json"
do
  cp -p "$file" "$CHECKPOINT/"
done
shasum -a 256 "$CHECKPOINT/"*.json "$CHECKPOINT/"*.ctcourse > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
git status --short --untracked-files=all
```

Expected:

- Every checksum prints `OK`.
- `git status` shows no private course file.

---

### Task 2: Build private Sicilian source inventory and authoring shell

**Files:**
- Create: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json`
- Create: `/private/tmp/mco15-sicilian-inventory-check.mjs`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-v1-authoring-inventory/`

**Interfaces:**
- Consumes: `/private/tmp/mco15-sicilian-source-scope.json` and rendered page images.
- Produces: private authoring inventory with stable coverage records and a checked teaching-node skeleton.

- [ ] **Step 1: Create authoring shell**

Run:

```bash
node - <<'JS'
const fs = require('fs')
const scopePath = '/private/tmp/mco15-sicilian-source-scope.json'
const outPath = '/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json'
const scope = JSON.parse(fs.readFileSync(scopePath, 'utf8'))
const inventory = {
  courseId: 'mco15-sicilian-black',
  targetContentVersion: '1.0.0',
  title: 'Sicilian Defence for Black',
  perspective: 'black',
  paths: {
    sourcePdf: scope.sourcePdf,
    renderedPageRoot: '/private/tmp/mco15-sicilian-pages',
    coursePack: '/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse'
  },
  scope,
  inventory: {
    pages: scope.pdfPages,
    coverageRecords: []
  },
  teachingNodes: [],
  teachingEdges: [],
  noteDrafts: {},
  sourceAudit: {
    inventoryMethod: 'manual visual curation from rendered private PDF pages',
    privateUseOnly: true
  }
}
fs.writeFileSync(outPath, `${JSON.stringify(inventory, null, 2)}\n`)
console.log(outPath)
JS
jq '{courseId,targetContentVersion,perspective,pageCount:(.inventory.pages|length),coverageRecords:(.inventory.coverageRecords|length),teachingNodes:(.teachingNodes|length)}' \
  "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json"
```

Expected:

- `pageCount` is `118`.
- `coverageRecords` is `0`.
- `teachingNodes` is `0`.

- [ ] **Step 2: Inventory the rendered source pages**

Open the rendered pages in batches and record the private source inventory in
`/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json`.

Use this exact coverage-record shape:

```json
{
  "coverageId": "p244-overview",
  "printedPage": 244,
  "pdfPage": 261,
  "familyId": "sicilian-foundations",
  "kind": "overview",
  "tableColumn": "",
  "noteLabel": "",
  "variationLabel": "Sicilian Defence overview",
  "depth": "quick",
  "teachingUse": "Explain why Black answers 1.e4 with ...c5."
}
```

Rules:

- Every coverage ID starts with `p<printedPage>-`.
- `printedPage` is always between `244` and `361`.
- `pdfPage` is always `printedPage + 17`.
- `familyId` is one of the six source-scope family IDs.
- `kind` is one of `overview`, `table`, `note`, `illustrative-game`, or `transition`.
- `depth` is one of `quick`, `standard`, or `reference`.
- Dense source note labels stay in private coverage metadata; learner-facing text is original explanation in the later course pack.
- Coverage records for Quick and Standard are a connected subset of the Reference inventory.

- [ ] **Step 3: Add teaching-node skeleton**

Add `teachingNodes` using this exact object shape:

```json
{
  "nodeId": "najdorf-a6-control",
  "chapterId": "najdorf",
  "title": "Why the Najdorf plays ...a6",
  "minimumDepth": "quick",
  "familyId": "najdorf",
  "lessonRole": "main-spine",
  "startAfter": "1.e4 c5 2.Nf3 d6 3.d4 cxd4 4.Nxd4 Nf6 5.Nc3",
  "learnerQuestion": "Why does Black spend a tempo on ...a6 before choosing the centre?",
  "primaryBlackDecision": "a7a6",
  "coverageIds": ["p246-najdorf-entry"]
}
```

Required teaching-node coverage:

- Quick: `sicilian-foundations-c5`, `open-sicilian-map`, `black-d6-nf6`, `najdorf-a6-control`, `najdorf-counterplay-map`, `najdorf-main-response`, `anti-sicilian-quick-map`, `quick-repertoire-checkpoint`.
- Standard: at least twelve additional nodes covering Najdorf attacking tries, anti-Sicilians, Dragon recognition, Scheveningen recognition, Classical recognition, and `...Nc6/...e5` recognition.
- Reference: enough additional nodes to cover every source-scope family and every captured coverage record.

- [ ] **Step 4: Create inventory checker**

Create `/private/tmp/mco15-sicilian-inventory-check.mjs` with this exact content:

```js
import fs from 'node:fs'

const [inventoryPath, outPath] = process.argv.slice(2)
if (!inventoryPath || !outPath) {
  console.error('usage: node mco15-sicilian-inventory-check.mjs <authoring.json> <out.json>')
  process.exit(2)
}

const inventory = JSON.parse(fs.readFileSync(inventoryPath, 'utf8'))
const coverage = inventory.inventory?.coverageRecords ?? []
const nodes = inventory.teachingNodes ?? []
const families = new Set((inventory.scope?.families ?? []).map((family) => family.familyId))
const coverageIds = new Map()
const problems = []

function addProblem(code, detail) {
  problems.push({ code, detail })
}

if (inventory.courseId !== 'mco15-sicilian-black') addProblem('wrong_course_id', inventory.courseId)
if (inventory.targetContentVersion !== '1.0.0') addProblem('wrong_version', inventory.targetContentVersion)
if (inventory.perspective !== 'black') addProblem('wrong_perspective', inventory.perspective)
if ((inventory.inventory?.pages ?? []).length !== 118) addProblem('wrong_page_count', (inventory.inventory?.pages ?? []).length)

for (const record of coverage) {
  if (coverageIds.has(record.coverageId)) addProblem('duplicate_coverage_id', record.coverageId)
  coverageIds.set(record.coverageId, record)
  if (!/^p\d{3}-[a-z0-9][a-z0-9-]*$/.test(record.coverageId)) addProblem('bad_coverage_id', record.coverageId)
  if (record.printedPage < 244 || record.printedPage > 361) addProblem('coverage_page_out_of_scope', record)
  if (record.pdfPage !== record.printedPage + 17) addProblem('coverage_pdf_mapping_mismatch', record)
  if (!families.has(record.familyId)) addProblem('unknown_family', record)
  if (!['overview', 'table', 'note', 'illustrative-game', 'transition'].includes(record.kind)) addProblem('bad_kind', record)
  if (!['quick', 'standard', 'reference'].includes(record.depth)) addProblem('bad_depth', record)
}

const requiredFamilies = [
  'sicilian-foundations',
  'najdorf',
  'dragon-accelerated-dragon',
  'scheveningen-e6-systems',
  'classical-nc6-e5-systems',
  'anti-sicilians'
]
for (const familyId of requiredFamilies) {
  if (!coverage.some((record) => record.familyId === familyId)) addProblem('missing_family_coverage', familyId)
  if (!nodes.some((node) => node.familyId === familyId)) addProblem('missing_family_node', familyId)
}

for (const node of nodes) {
  if (!/^[a-z0-9][a-z0-9._-]{0,127}$/.test(node.nodeId)) addProblem('bad_node_id', node.nodeId)
  if (!['quick', 'standard', 'reference'].includes(node.minimumDepth)) addProblem('bad_node_depth', node)
  if (!families.has(node.familyId)) addProblem('node_unknown_family', node)
  for (const coverageId of node.coverageIds ?? []) {
    if (!coverageIds.has(coverageId)) addProblem('node_missing_coverage', { nodeId: node.nodeId, coverageId })
  }
}

const summary = {
  courseId: inventory.courseId,
  targetContentVersion: inventory.targetContentVersion,
  perspective: inventory.perspective,
  pages: inventory.inventory?.pages?.length ?? 0,
  coverageRecords: coverage.length,
  teachingNodes: nodes.length,
  coverageByFamily: Object.fromEntries(requiredFamilies.map((familyId) => [
    familyId,
    coverage.filter((record) => record.familyId === familyId).length
  ])),
  quickNodes: nodes.filter((node) => node.minimumDepth === 'quick').length,
  standardNodes: nodes.filter((node) => node.minimumDepth === 'standard').length,
  referenceNodes: nodes.filter((node) => node.minimumDepth === 'reference').length,
  problems
}

fs.writeFileSync(outPath, `${JSON.stringify(summary, null, 2)}\n`)
console.log(JSON.stringify({
  coverageRecords: summary.coverageRecords,
  teachingNodes: summary.teachingNodes,
  problems: summary.problems.length,
  coverageByFamily: summary.coverageByFamily
}, null, 2))
if (problems.length > 0) process.exit(1)
```

- [ ] **Step 5: Run inventory checker and checkpoint**

Run:

```bash
node --check /private/tmp/mco15-sicilian-inventory-check.mjs
node /private/tmp/mco15-sicilian-inventory-check.mjs \
  "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json" \
  /private/tmp/mco15-sicilian-inventory-check.json
jq '{coverageRecords,teachingNodes,quickNodes,standardNodes,referenceNodes,problems:(.problems|length),coverageByFamily}' \
  /private/tmp/mco15-sicilian-inventory-check.json
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-v1-authoring-inventory"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json" "$CHECKPOINT/mco15-sicilian-black.authoring.json"
cp -p /private/tmp/mco15-sicilian-source-scope.json "$CHECKPOINT/source-scope.json"
cp -p /private/tmp/mco15-sicilian-inventory-check.json "$CHECKPOINT/inventory-check.json"
shasum -a 256 "$CHECKPOINT/"*.json > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
```

Expected:

- `problems` is `0`.
- `coverageRecords` is at least `118`.
- `teachingNodes` is at least `60`.
- all six source families have nonzero coverage.
- every checksum prints `OK`.

---

### Task 3: Build Quick Najdorf-centered Black spine

**Files:**
- Create/update: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse`
- Read/update: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json`
- Create: `/private/tmp/mco15-sicilian-quick-validation.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-v1-quick-spine/`

**Interfaces:**
- Consumes: checked authoring inventory.
- Produces: valid schema-v2 Quick-depth Sicilian course spine.

- [ ] **Step 1: Verify SAN helper for the main Najdorf route**

Run:

```bash
go run ./cmd/coursepack sanline \
  --fen "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1" \
  e4 c5 Nf3 d6 d4 cxd4 Nxd4 Nf6 Nc3 a6 \
  | jq '{startFen, finalFen, uci:[.moves[].uci]}'
```

Expected:

```json
{
  "uci": ["e2e4","c7c5","g1f3","d7d6","d2d4","c5d4","f3d4","g8f6","b1c3","a7a6"]
}
```

- [ ] **Step 2: Create schema-v2 course root**

Create `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse` as a UTF-8 JSON object with:

```json
{
  "schemaVersion": 2,
  "courseId": "mco15-sicilian-black",
  "contentVersion": "1.0.0",
  "title": "Sicilian Defence for Black",
  "description": "A Black repertoire and reference course for the Sicilian Defence, centered on the Najdorf with full-family reference branches.",
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

Set the initial `chapters` array to:

```json
[
  {"chapterId":"foundations","title":"Sicilian Foundations","ordinal":1,"overview":"Learn why Black challenges 1.e4 asymmetrically with ...c5.","minimumDepth":"quick"},
  {"chapterId":"najdorf","title":"Najdorf Main Spine","ordinal":2,"overview":"Reach the Najdorf and understand Black's first practical decisions.","minimumDepth":"quick"},
  {"chapterId":"anti-sicilians","title":"Anti-Sicilian Survival","ordinal":3,"overview":"Recognize when White avoids the Open Sicilian and choose a stable Black response.","minimumDepth":"quick"},
  {"chapterId":"reference-families","title":"Sicilian Reference Families","ordinal":4,"overview":"Browse the wider Sicilian family after the main repertoire is understood.","minimumDepth":"reference"}
]
```

- [ ] **Step 4: Author Quick move graph and prompts**

Create connected positions and moves for:

```text
initial
1.e4
1...c5
2.Nf3
2...d6
3.d4
3...cxd4
4.Nxd4
4...Nf6
5.Nc3
5...a6
```

Add one anti-Sicilian recognition branch from after `1...c5` for each of:

```text
2.c3
2.Nc3
2.f4
```

Rules:

- White moves use `trainingRole: "opponent"`.
- Black's main answers use `trainingRole: "repertoire"`.
- Black acceptable side responses use `trainingRole: "alternative"` only when the prompt is explicitly teaching a branch choice.
- Every decision prompt's primary move points to a Black `repertoire` or `alternative` move.
- Every used coverage ID appears in `sourceCoverage.expectedReferences`.
- During Quick validation, `sourceCoverage.expectedReferences` contains only the coverage IDs used by the Quick pack.

- [ ] **Step 5: Author Quick lessons**

Required Quick lessons:

```text
sicilian-foundations-c5
open-sicilian-map
black-d6-nf6
najdorf-a6-control
najdorf-counterplay-map
najdorf-main-response
anti-sicilian-quick-map
quick-repertoire-checkpoint
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

- `8` Quick lessons.
- every Quick lesson has 2-3 required activities.
- no Quick lesson has more than one required decision.
- no Quick lesson asks the same move from the same position twice.

- [ ] **Step 6: Author Quick teaching tree**

Create `lessonEdges` so the Quick tree is:

```text
sicilian-foundations-c5
└── open-sicilian-map
    ├── black-d6-nf6
    │   └── najdorf-a6-control
    │       └── najdorf-counterplay-map
    │           └── najdorf-main-response
    │               └── quick-repertoire-checkpoint
    └── anti-sicilian-quick-map
```

Requirements:

- `sicilian-foundations-c5` is the single root.
- the anti-Sicilian edge uses `kind: "alternative"` with label `White avoids the Open Sicilian`.
- every Quick lesson is reachable at Quick depth.

- [ ] **Step 7: Validate and checkpoint Quick spine**

Run:

```bash
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse" \
  > /private/tmp/mco15-sicilian-quick-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/mco15-sicilian-quick-validation.json
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-v1-quick-spine"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json" "$CHECKPOINT/mco15-sicilian-black.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse" "$CHECKPOINT/mco15-sicilian-black.ctcourse"
cp -p /private/tmp/mco15-sicilian-quick-validation.json "$CHECKPOINT/validation.json"
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  "$CHECKPOINT/validation.json" > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/"*.json "$CHECKPOINT/"*.ctcourse > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
```

Expected:

- validation exits `0`.
- `courseId` is `mco15-sicilian-black`.
- `contentVersion` is `1.0.0`.
- `counts.lessons` is `8`.
- `counts.warnings` is `0`.
- `missing` is `0`.
- `unexpected` is `0`.
- every checksum prints `OK`.

---

### Task 4: Build Standard layer with main Najdorf branches and practical anti-Sicilians

**Files:**
- Update: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json`
- Update: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse`
- Create: `/private/tmp/mco15-sicilian-standard-validation.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-v1-standard-layer/`

**Interfaces:**
- Consumes: valid Quick spine.
- Produces: valid Standard-depth practical Sicilian repertoire layer.

- [ ] **Step 1: Add Standard Najdorf teaching nodes**

Add Standard lessons for these node IDs:

```text
najdorf-english-attack-map
najdorf-bg5-pressure
najdorf-be2-classical-setup
najdorf-bc4-sozin-pressure
najdorf-f4-kingside-space
najdorf-quiet-systems
```

Each Standard Najdorf lesson must:

- start from a position already connected to the Najdorf spine;
- include one concept or comparison activity before any decision;
- ask at most one required Black decision;
- include optional Reference cards only when they are visible at Reference depth.

- [ ] **Step 2: Add Standard anti-Sicilian lessons**

Add Standard lessons for these node IDs:

```text
alapin-standard-response
closed-sicilian-standard-response
bb5-systems-standard-response
grand-prix-standard-response
smith-morra-standard-response
unusual-second-moves-map
```

Each anti-Sicilian lesson must answer this learner question:

```text
White avoided the Open Sicilian; how does Black keep a playable repertoire?
```

Rules:

- White's deviation moves use `trainingRole: "opponent"`.
- Black's response prompt uses `trainingRole: "repertoire"` or `trainingRole: "alternative"`.
- Do not force the learner through the Najdorf stem before anti-Sicilian lessons.

- [ ] **Step 3: Add Standard family-recognition lessons**

Add Standard recognition lessons for:

```text
dragon-family-recognition
scheveningen-family-recognition
classical-family-recognition
nc6-e5-family-recognition
```

These lessons are not the Quick repertoire. Each should be a compact comparison that explains where the family sits in the Sicilian map.

- [ ] **Step 4: Connect Standard teaching tree**

Add edges so:

```text
najdorf-main-response
├── najdorf-english-attack-map
├── najdorf-bg5-pressure
├── najdorf-be2-classical-setup
├── najdorf-bc4-sozin-pressure
├── najdorf-f4-kingside-space
└── najdorf-quiet-systems

anti-sicilian-quick-map
├── alapin-standard-response
├── closed-sicilian-standard-response
├── bb5-systems-standard-response
├── grand-prix-standard-response
├── smith-morra-standard-response
└── unusual-second-moves-map

open-sicilian-map
├── dragon-family-recognition
├── scheveningen-family-recognition
├── classical-family-recognition
└── nc6-e5-family-recognition
```

Requirements:

- branch edges use `kind: "alternative"` with learner-readable labels.
- family-recognition edges use `minimumDepth: "standard"`.
- every Standard lesson is reachable at Standard depth.

- [ ] **Step 5: Validate and checkpoint Standard layer**

Run:

```bash
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse" \
  > /private/tmp/mco15-sicilian-standard-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/mco15-sicilian-standard-validation.json
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-v1-standard-layer"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json" "$CHECKPOINT/mco15-sicilian-black.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse" "$CHECKPOINT/mco15-sicilian-black.ctcourse"
cp -p /private/tmp/mco15-sicilian-standard-validation.json "$CHECKPOINT/validation.json"
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  "$CHECKPOINT/validation.json" > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/"*.json "$CHECKPOINT/"*.ctcourse > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
```

Expected:

- validation exits `0`.
- `counts.lessons` is between `24` and `28`.
- `counts.warnings` is `0`.
- `missing` is `0`.
- `unexpected` is `0`.
- every checksum prints `OK`.

---

### Task 5: Build full Najdorf Reference branch

**Files:**
- Update: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json`
- Update: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse`
- Create: `/private/tmp/mco15-sicilian-najdorf-reference-validation.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-v1-najdorf-reference/`

**Interfaces:**
- Consumes: valid Standard layer and private source inventory for printed pages `246-268`.
- Produces: full Reference coverage for the Najdorf family.

- [ ] **Step 1: Add Najdorf Reference nodes**

Use rendered pages for printed pages `246-268`, PDF pages `263-285`.

Add Reference nodes under the existing Najdorf Standard branches for:

```text
najdorf-english-attack-reference
najdorf-bg5-reference
najdorf-be2-reference
najdorf-bc4-sozin-reference
najdorf-f4-reference
najdorf-quiet-systems-reference
najdorf-rare-sixth-moves-reference
najdorf-transpositions-reference
```

Requirements:

- every captured Najdorf coverage record appears in `sourceCoverage.expectedReferences`;
- every captured Najdorf coverage record is used by at least one move or note;
- dense table lines are optional Reference activities unless they teach a critical Black decision;
- private illustrative game references stay concise and private.

- [ ] **Step 2: Validate and checkpoint Najdorf Reference**

Run:

```bash
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse" \
  > /private/tmp/mco15-sicilian-najdorf-reference-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/mco15-sicilian-najdorf-reference-validation.json
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-v1-najdorf-reference"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json" "$CHECKPOINT/mco15-sicilian-black.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse" "$CHECKPOINT/mco15-sicilian-black.ctcourse"
cp -p /private/tmp/mco15-sicilian-najdorf-reference-validation.json "$CHECKPOINT/validation.json"
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  "$CHECKPOINT/validation.json" > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/"*.json "$CHECKPOINT/"*.ctcourse > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
```

Expected:

- validation exits `0`.
- `counts.lessons` is between `32` and `42`.
- `counts.warnings` is `0`.
- `missing` is `0`.
- `unexpected` is `0`.
- every checksum prints `OK`.

---

### Task 6: Build Dragon, Accelerated Dragon, Scheveningen, and ...e6 Reference families

**Files:**
- Update: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json`
- Update: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse`
- Create: `/private/tmp/mco15-sicilian-dragon-e6-validation.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-v1-dragon-e6-reference/`

**Interfaces:**
- Consumes: Najdorf Reference pack and private source inventory for printed pages `269-317`.
- Produces: Reference branches for Dragon/Accelerated Dragon, Scheveningen, and `...e6` systems.

- [ ] **Step 1: Add Dragon and Accelerated Dragon Reference nodes**

Use rendered pages for printed pages `269-289`, PDF pages `286-306`.

Add nodes:

```text
dragon-structure-reference
dragon-yugoslav-attack-reference
accelerated-dragon-move-order-reference
dragon-tactical-branches-reference
```

Teaching focus:

- fianchetto structure;
- opposite-wing race recognition;
- move-order differences between Dragon and Accelerated Dragon;
- tactical detail kept optional.

- [ ] **Step 2: Add Scheveningen and ...e6 Reference nodes**

Use rendered pages for printed pages `290-317`, PDF pages `307-334`.

Add nodes:

```text
scheveningen-structure-reference
scheveningen-najdorf-transpositions-reference
taimanov-paulsen-reference
four-knights-e6-reference
e6-systems-rare-lines-reference
```

Teaching focus:

- small-centre structure;
- flexible development;
- where `...e6` move orders transpose into Najdorf-style structures;
- which `...e6` systems are deliberate repertoire switches.

- [ ] **Step 3: Validate and checkpoint Dragon/...e6 layer**

Run:

```bash
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse" \
  > /private/tmp/mco15-sicilian-dragon-e6-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/mco15-sicilian-dragon-e6-validation.json
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-v1-dragon-e6-reference"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json" "$CHECKPOINT/mco15-sicilian-black.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse" "$CHECKPOINT/mco15-sicilian-black.ctcourse"
cp -p /private/tmp/mco15-sicilian-dragon-e6-validation.json "$CHECKPOINT/validation.json"
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  "$CHECKPOINT/validation.json" > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/"*.json "$CHECKPOINT/"*.ctcourse > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
```

Expected:

- validation exits `0`.
- `counts.lessons` is between `41` and `55`.
- `counts.warnings` is `0`.
- `missing` is `0`.
- `unexpected` is `0`.
- every checksum prints `OK`.

---

### Task 7: Build Classical, ...Nc6, and ...e5 Reference families

**Files:**
- Update: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json`
- Update: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse`
- Create: `/private/tmp/mco15-sicilian-classical-nc6-e5-validation.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-v1-classical-nc6-e5-reference/`

**Interfaces:**
- Consumes: Dragon/...e6 Reference pack and private source inventory for printed pages `318-345`.
- Produces: Reference branches for Classical Sicilian, Richter-Rauzer, Sveshnikov, Kalashnikov, Lowenthal, and related `...Nc6/...e5` systems.

- [ ] **Step 1: Add Classical Reference nodes**

Use rendered pages for printed pages `318-335`, PDF pages `335-352`.

Add nodes:

```text
classical-development-reference
richter-rauzer-reference
boleslavsky-sozin-reference
velimirovic-attack-reference
classical-rare-lines-reference
```

Teaching focus:

- Classical development as a recognizable family;
- White attacking patterns against Classical development;
- Black counterplay and central-break timing.

- [ ] **Step 2: Add ...Nc6 and ...e5 Reference nodes**

Use rendered pages for printed pages `336-345`, PDF pages `353-362`.

Add nodes:

```text
sveshnikov-structure-reference
kalashnikov-structure-reference
lowenthal-reference
nc6-e5-transpositions-reference
```

Teaching focus:

- why `...Nc6` and `...e5` become separate Sicilian families;
- which structural concessions Black accepts;
- where these systems diverge from a Najdorf repertoire.

- [ ] **Step 3: Validate and checkpoint Classical/...Nc6/...e5 layer**

Run:

```bash
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse" \
  > /private/tmp/mco15-sicilian-classical-nc6-e5-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/mco15-sicilian-classical-nc6-e5-validation.json
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-v1-classical-nc6-e5-reference"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json" "$CHECKPOINT/mco15-sicilian-black.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse" "$CHECKPOINT/mco15-sicilian-black.ctcourse"
cp -p /private/tmp/mco15-sicilian-classical-nc6-e5-validation.json "$CHECKPOINT/validation.json"
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  "$CHECKPOINT/validation.json" > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/"*.json "$CHECKPOINT/"*.ctcourse > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
```

Expected:

- validation exits `0`.
- `counts.lessons` is between `50` and `66`.
- `counts.warnings` is `0`.
- `missing` is `0`.
- `unexpected` is `0`.
- every checksum prints `OK`.

---

### Task 8: Build full anti-Sicilian Reference branch

**Files:**
- Update: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json`
- Update: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse`
- Create: `/private/tmp/mco15-sicilian-anti-reference-validation.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-v1-anti-sicilian-reference/`

**Interfaces:**
- Consumes: Classical/...Nc6/...e5 Reference pack and private source inventory for printed pages `346-361`.
- Produces: full Reference branch for non-open Sicilians and anti-Sicilians.

- [ ] **Step 1: Add anti-Sicilian Reference nodes**

Use rendered pages for printed pages `346-361`, PDF pages `363-378`.

Add nodes:

```text
alapin-reference
closed-sicilian-reference
bb5-systems-reference
grand-prix-f4-reference
smith-morra-reference
unusual-second-moves-reference
anti-sicilian-transpositions-reference
```

Teaching focus:

- Black should recognize when White avoids the Open Sicilian;
- Black responses should be practical and Black-perspective;
- anti-Sicilian branches should not require replaying the Najdorf stem.

- [ ] **Step 2: Validate and checkpoint anti-Sicilian Reference**

Run:

```bash
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse" \
  > /private/tmp/mco15-sicilian-anti-reference-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/mco15-sicilian-anti-reference-validation.json
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-v1-anti-sicilian-reference"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json" "$CHECKPOINT/mco15-sicilian-black.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse" "$CHECKPOINT/mco15-sicilian-black.ctcourse"
cp -p /private/tmp/mco15-sicilian-anti-reference-validation.json "$CHECKPOINT/validation.json"
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  "$CHECKPOINT/validation.json" > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/"*.json "$CHECKPOINT/"*.ctcourse > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
```

Expected:

- validation exits `0`.
- `counts.lessons` is between `57` and `76`.
- `counts.warnings` is `0`.
- `missing` is `0`.
- `unexpected` is `0`.
- every checksum prints `OK`.

---

### Task 9: Final course audit, validation, and private final checkpoint

**Files:**
- Read/update: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json`
- Read/update: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse`
- Create: `/private/tmp/mco15-sicilian-course-audit.mjs`
- Create: `/private/tmp/mco15-sicilian-final-validation.json`
- Create: `/private/tmp/mco15-sicilian-final-audit.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-v1-final/`

**Interfaces:**
- Consumes: complete private course pack.
- Produces: final validated/audited private pack with recoverable checkpoint.

- [ ] **Step 1: Create final course audit script**

Create `/private/tmp/mco15-sicilian-course-audit.mjs` with this exact content:

```js
import fs from 'node:fs'

const [packPath, outPath] = process.argv.slice(2)
if (!packPath || !outPath) {
  console.error('usage: node mco15-sicilian-course-audit.mjs <course.ctcourse> <audit.json>')
  process.exit(2)
}

const pack = JSON.parse(fs.readFileSync(packPath, 'utf8'))
const prompts = new Map((pack.prompts ?? []).map((prompt) => [prompt.promptId, prompt]))
const moves = new Map((pack.moves ?? []).map((move) => [move.moveId, move]))
const positions = new Map((pack.positions ?? []).map((position) => [position.positionId, position]))
const spamPattern = /source note [a-z]+ records|opening activity result|course move found|appliedmoves/i
const allowedDepths = new Set(['quick', 'standard', 'reference'])
const depthRank = { quick: 1, standard: 2, reference: 3 }

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

function duplicateGroups(values, key) {
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
      duplicatePromptIds: duplicateGroups(promptMoves, (entry) => entry.promptId),
      duplicatePrimaryMoveFromPosition: duplicateGroups(promptMoves, (entry) => `${entry.positionId}:${entry.primaryMoveId}`),
      nonBlackDecisionRole: promptMoves.filter((entry) => entry.primaryRole !== 'repertoire' && entry.primaryRole !== 'alternative'),
      genericSpamText: requiredText.some((value) => spamPattern.test(value))
    }
  }
})

const printedPages = pack.sourceCoverage?.printedPages ?? []
const expectedReferences = pack.sourceCoverage?.expectedReferences ?? []
const quickLessons = (pack.lessons ?? []).filter((lesson) => depthRank[lesson.minimumDepth] <= depthRank.quick)
const standardVisibleLessons = (pack.lessons ?? []).filter((lesson) => depthRank[lesson.minimumDepth] <= depthRank.standard)
const referenceLessons = pack.lessons ?? []
const lessonTitles = referenceLessons.map((lesson) => `${lesson.lessonId} ${lesson.title}`.toLowerCase())
const familyChecks = {
  najdorf: lessonTitles.some((title) => title.includes('najdorf')),
  dragon: lessonTitles.some((title) => title.includes('dragon')),
  scheveningen: lessonTitles.some((title) => title.includes('scheveningen')),
  e6Systems: lessonTitles.some((title) => title.includes('e6') || title.includes('paulsen') || title.includes('taimanov')),
  classical: lessonTitles.some((title) => title.includes('classical') || title.includes('richter')),
  nc6e5: lessonTitles.some((title) => title.includes('sveshnikov') || title.includes('kalashnikov') || title.includes('lowenthal')),
  antiSicilians: lessonTitles.some((title) => title.includes('alapin') || title.includes('closed sicilian') || title.includes('smith-morra'))
}

const courseFlags = {
  wrongCourseId: pack.courseId !== 'mco15-sicilian-black',
  wrongVersion: pack.contentVersion !== '1.0.0',
  wrongPerspective: pack.perspective !== 'black',
  wrongDefaultDepth: pack.defaultDepth !== 'reference',
  wrongRootLessons: roots.length !== 1 || roots[0] !== 'sicilian-foundations-c5',
  unreachableLessons: reachable.size !== lessonIds.size,
  quickLessonCountOutOfRange: quickLessons.length < 7 || quickLessons.length > 9,
  standardLessonCountOutOfRange: standardVisibleLessons.length < 18 || standardVisibleLessons.length > 28,
  referenceLessonCountOutOfRange: referenceLessons.length < 57 || referenceLessons.length > 90,
  printedPageCountWrong: printedPages.length !== 118,
  printedFirstWrong: Math.min(...printedPages) !== 244,
  printedLastWrong: Math.max(...printedPages) !== 361,
  tooFewCoverageReferences: expectedReferences.length < 118,
  missingFamilyCoverage: Object.entries(familyChecks).filter(([, ok]) => !ok).map(([family]) => family)
}

const lessonBlockers = lessons.filter((lesson) => Object.values(lesson.flags).some((value) => Array.isArray(value) ? value.length > 0 : value === true))
const courseBlockers = Object.entries(courseFlags).filter(([, value]) => Array.isArray(value) ? value.length > 0 : value === true)
const summary = {
  courseId: pack.courseId,
  contentVersion: pack.contentVersion,
  perspective: pack.perspective,
  defaultDepth: pack.defaultDepth,
  rootLessons: roots,
  counts: {
    chapters: (pack.chapters ?? []).length,
    lessons: referenceLessons.length,
    quickLessons: quickLessons.length,
    standardVisibleLessons: standardVisibleLessons.length,
    lessonEdges: (pack.lessonEdges ?? []).length,
    prompts: (pack.prompts ?? []).length,
    positions: (pack.positions ?? []).length,
    moves: (pack.moves ?? []).length,
    notes: (pack.notes ?? []).length,
    printedPages: printedPages.length,
    expectedReferences: expectedReferences.length,
    requiredActivities: lessons.reduce((sum, lesson) => sum + lesson.requiredCount, 0),
    requiredDecisions: lessons.reduce((sum, lesson) => sum + lesson.decisionCount, 0)
  },
  familyChecks,
  courseFlags,
  lessons,
  courseBlockers,
  lessonBlockers,
  blockerCount: courseBlockers.length + lessonBlockers.length
}

fs.writeFileSync(outPath, `${JSON.stringify(summary, null, 2)}\n`)
console.log(JSON.stringify({
  courseId: summary.courseId,
  contentVersion: summary.contentVersion,
  perspective: summary.perspective,
  blockerCount: summary.blockerCount,
  rootLessons: summary.rootLessons,
  counts: summary.counts,
  familyChecks: summary.familyChecks
}, null, 2))
if (summary.blockerCount > 0) process.exit(1)
```

- [ ] **Step 2: Run final validation and audit**

Run:

```bash
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse" \
  > /private/tmp/mco15-sicilian-final-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/mco15-sicilian-final-validation.json
node --check /private/tmp/mco15-sicilian-course-audit.mjs
node /private/tmp/mco15-sicilian-course-audit.mjs \
  "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse" \
  /private/tmp/mco15-sicilian-final-audit.json
jq '{courseId,contentVersion,perspective,defaultDepth,counts,familyChecks,blockerCount,courseBlockers,lessonBlockers:[.lessonBlockers[] | {lessonId,flags}]}' \
  /private/tmp/mco15-sicilian-final-audit.json
```

Expected:

- validation exits `0`.
- `courseId` is `mco15-sicilian-black`.
- `contentVersion` is `1.0.0`.
- `perspective` is `black`.
- `defaultDepth` is `reference`.
- `counts.warnings` is `0`.
- `missing` is `0`.
- `unexpected` is `0`.
- audit `blockerCount` is `0`.
- `rootLessons` is exactly `["sicilian-foundations-c5"]`.
- all family checks are `true`.

- [ ] **Step 3: Create final private checkpoint**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-v1-final"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json" "$CHECKPOINT/mco15-sicilian-black.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse" "$CHECKPOINT/mco15-sicilian-black.ctcourse"
cp -p /private/tmp/mco15-sicilian-source-scope.json "$CHECKPOINT/source-scope.json"
cp -p /private/tmp/mco15-sicilian-final-validation.json "$CHECKPOINT/validation.json"
cp -p /private/tmp/mco15-sicilian-final-audit.json "$CHECKPOINT/audit.json"
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  "$CHECKPOINT/validation.json" > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/"*.json "$CHECKPOINT/"*.ctcourse > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
git status --short --untracked-files=all
```

Expected:

- every checksum prints `OK`.
- `git status` shows no private source files, rendered pages, authoring files, course packs, import outputs, or checkpoints.

---

### Task 10: Import, app verification, final tests, and handoff

**Files:**
- Create: `.superpowers/tmp/sicilian-black-import-smoke.go`
- Create: `.superpowers/tmp/sicilian-black-orientation-smoke.go`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse`
- Create: `/private/tmp/mco15-sicilian-disposable-import.json`
- Create: `/private/tmp/mco15-sicilian-default-import.json`
- Create: `/private/tmp/mco15-sicilian-orientation-smoke.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-v1-final/import-smoke.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-v1-final/orientation-smoke.json`

**Interfaces:**
- Consumes: final validated private course pack.
- Produces: imported default catalogue, app-level smoke proof, test results, and clean implementation branch.

- [ ] **Step 1: Create import smoke harness**

Create `.superpowers/tmp/sicilian-black-import-smoke.go` with this exact content:

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
	if os.Getenv("SICILIAN_USE_DEFAULT_DATA") == "1" {
		var err error
		paths, err = storage.DefaultPaths()
		if err != nil {
			panic(err)
		}
	} else {
		dataRoot := os.Getenv("SICILIAN_DATA_ROOT")
		if dataRoot == "" {
			fmt.Fprintln(os.Stderr, "SICILIAN_DATA_ROOT or SICILIAN_USE_DEFAULT_DATA=1 is required")
			os.Exit(2)
		}
		paths = storage.PathsAt(dataRoot)
	}
	coursePaths := strings.Split(os.Getenv("SICILIAN_COURSES"), "|")
	if len(coursePaths) == 0 || coursePaths[0] == "" {
		fmt.Fprintln(os.Stderr, "SICILIAN_COURSES is required")
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

- [ ] **Step 2: Run disposable import smoke with all active courses**

Run:

```bash
SICILIAN_DATA_ROOT="$(mktemp -d /private/tmp/mco15-sicilian-import.XXXXXX)" \
SICILIAN_COURSES="/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse|/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse|/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse|/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse|/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse|/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse" \
go run .superpowers/tmp/sicilian-black-import-smoke.go \
  > /private/tmp/mco15-sicilian-disposable-import.json
jq '{courses:[.home.courses[] | {courseId,title,perspective}],activeVersions}' \
  /private/tmp/mco15-sicilian-disposable-import.json
```

Expected:

- six active courses are present.
- `mco15-sicilian-black` is present.
- `activeVersions["mco15-sicilian-black"]` is `1.0.0`.

- [ ] **Step 3: Import Sicilian into the default local catalogue**

Run:

```bash
SICILIAN_USE_DEFAULT_DATA=1 \
SICILIAN_COURSES="/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse" \
go run .superpowers/tmp/sicilian-black-import-smoke.go \
  > /private/tmp/mco15-sicilian-default-import.json
jq '{imports:[.imports[] | {path,report:.report}],courses:[.home.courses[] | {courseId,title,perspective}],activeVersions}' \
  /private/tmp/mco15-sicilian-default-import.json
```

Expected:

- import report for Sicilian has `Accepted: 1`.
- `mco15-sicilian-black` is active in the default local catalogue.
- existing courses remain active.

- [ ] **Step 4: Create Black orientation and role smoke harness**

Create `.superpowers/tmp/sicilian-black-orientation-smoke.go` with this exact content:

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

	compiled, err := services.OpeningCatalog.LoadActive(ctx, "mco15-sicilian-black")
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
		"defaultDepth":        compiled.Pack.DefaultDepth,
		"lessons":             len(compiled.Lessons),
		"prompts":             len(compiled.Prompts),
		"invalidPrimaryRoles": invalidPrimaryRoles,
		"ok":                  compiled.Pack.CourseID == "mco15-sicilian-black" && compiled.Pack.ContentVersion == "1.0.0" && compiled.Pack.Perspective == "black" && compiled.Pack.DefaultDepth == "reference" && len(invalidPrimaryRoles) == 0,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(result)
	if !result["ok"].(bool) {
		os.Exit(1)
	}
}
```

- [ ] **Step 5: Run orientation smoke**

Run:

```bash
go run .superpowers/tmp/sicilian-black-orientation-smoke.go \
  > /private/tmp/mco15-sicilian-orientation-smoke.json
jq . /private/tmp/mco15-sicilian-orientation-smoke.json
cp -p /private/tmp/mco15-sicilian-disposable-import.json \
  "/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-v1-final/disposable-import.json"
cp -p /private/tmp/mco15-sicilian-default-import.json \
  "/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-v1-final/default-import.json"
cp -p /private/tmp/mco15-sicilian-orientation-smoke.json \
  "/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-black-v1-final/orientation-smoke.json"
```

Expected:

- `ok` is `true`.
- `perspective` is `black`.
- `defaultDepth` is `reference`.
- `invalidPrimaryRoles` is an empty array.

- [ ] **Step 6: Run repository and opening E2E verification**

Run:

```bash
npm --prefix frontend run test:e2e -- frontend/tests/openings.spec.ts --project=chromium
npm --prefix frontend run test:e2e -- frontend/tests/openings.spec.ts --project=webkit
go test ./...
git diff --check
git status --short --untracked-files=all
```

Expected:

- Chromium opening E2E passes.
- WebKit opening E2E passes.
- `go test ./...` passes.
- `git diff --check` exits `0`.
- `git status` shows only intended public docs/code changes plus ignored `.superpowers/tmp` files if they are visible with a broader untracked setting; no private course files appear.

- [ ] **Step 7: Rebuild and relaunch app**

Run the pinned local Wails build:

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
file "build/bin/Chess Trainer.app/Contents/MacOS/Chess Trainer"
open "build/bin/Chess Trainer.app"
pgrep -fl "Chess Trainer.app/Contents/MacOS/Chess Trainer"
```

Expected:

- Wails build succeeds.
- generated binding files are whitespace-normalized.
- codesign verifies.
- app process is running from `/Users/admin/Documents/Work/chess/build/bin/Chess Trainer.app/Contents/MacOS/Chess Trainer`.
- the local app can open Learn Openings and show `Sicilian Defence for Black` under Black openings.

- [ ] **Step 8: Commit public implementation artifacts**

Commit only public repository changes. Do not stage private course files or `/private/tmp` artifacts.

Run:

```bash
git status --short --untracked-files=all
git diff --stat
```

If implementation changed only private course data and verification artifacts,
update the spec status line to record completion:

```bash
perl -0pi -e 's/Status: approved in chat, implementation plan written/Status: approved in chat and implemented as private v1.0.0 local course pack/' \
  docs/superpowers/specs/2026-07-24-sicilian-black-full-reference-course-design.md
git diff -- docs/superpowers/specs/2026-07-24-sicilian-black-full-reference-course-design.md
git add docs/superpowers/specs/2026-07-24-sicilian-black-full-reference-course-design.md
git commit -m "docs: mark sicilian black course implemented"
```

If implementation required generic app fixes, stage only the reviewed public code/test/docs files and commit with the message:

```bash
git commit -m "feat: support large sicilian opening course"
```

Expected:

- private course files are not staged.
- final implementation commit contains only public docs, generic code, generic tests, or synthetic fixtures.
- `git status --short --untracked-files=all` is clean except ignored private workflow files.

---

## Final Review Gate

Before merge or push, run:

```bash
go test ./...
npm --prefix frontend run test:e2e -- frontend/tests/openings.spec.ts --project=chromium
npm --prefix frontend run test:e2e -- frontend/tests/openings.spec.ts --project=webkit
go run ./cmd/coursepack validate "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse" \
  > /private/tmp/mco15-sicilian-final-validation-repeat.json
node /private/tmp/mco15-sicilian-course-audit.mjs \
  "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse" \
  /private/tmp/mco15-sicilian-final-audit-repeat.json
git diff --check
git status --short --untracked-files=all
```

Expected:

- all commands exit `0`.
- repeated validation has zero missing and zero unexpected source coverage.
- repeated audit has `blockerCount: 0`.
- working tree contains no private source-derived files.

Then request strict code review with `$thermo-nuclear-code-quality-review` if public app code changed, or request course-plan/workflow review if the final diff is docs-only plus private course output.
