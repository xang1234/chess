# French Defence for Black Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build and import a private, full-reference `mco15-french-black` course that teaches Black's French Defence with a Classical main spine and full French Reference tree.

**Architecture:** The course is a private schema-v2 `.ctcourse` pack backed by a private source inventory and authoring file. Quick and Standard teach a solid Classical French Black repertoire; Reference captures the full selected French source scope as a browseable tree. The app should need no French-specific runtime code unless the private pack exposes a generic opening-course defect.

**Tech Stack:** Go coursepack validator/importer, schema-v2 opening course JSON, Node.js private authoring/audit scripts, `jq`, Poppler `pdftoppm`, private local course files, Playwright opening E2E.

## Global Constraints

- Course title: `French Defence for Black`
- Course ID: `mco15-french-black`
- Content version: `1.0.0`
- Perspective: `black`
- Default depth: `reference`
- Authoring file: `/Users/admin/Documents/Private Chess Courses/mco15-french-black.authoring.json`
- Course pack: `/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse`
- Source PDF: `/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf`
- Source printed pages: `197-243`
- Source PDF pages: `214-260`
- Scan offset: `+17`
- Page count: `47`
- The French section starts at the French Defence title page and ends immediately before the Sicilian Defence title page.
- Main learning spine is Classical French: `1.e4 e6 2.d4 d5 3.Nc3 Nf6`.
- Quick teaches a playable Black Classical route, not the full French.
- Standard teaches main practical decisions and common deviations.
- Reference contains the full selected French source-backed map.
- Board orientation defaults to Black.
- Black repertoire moves are learner decisions.
- White moves appear as opponent pressure, demonstrations, comparisons, or branch choices.
- Quick lessons usually have 2-3 required activities.
- Standard lessons usually have 3-4 required activities.
- Reference activities are optional by default.
- Required decisions do not repeat the same prompt from the same position inside one lesson.
- Do not add generic filler such as `Source note x records...`, raw activity-result JSON, or `Course move found`.
- Coverage validation must have no missing or unexpected records at final validation.
- Course warnings remain at zero unless a deliberately learner-facing warning is useful.
- No private source prose, rendered pages, authoring inventory, course pack, import outputs, or checkpoints are committed to Git.

---

## File Structure

- `/Users/admin/Documents/Private Chess Courses/mco15-french-black.authoring.json`
  - Private source inventory, rendered-page metadata, coverage records, family map, teaching node intent, and stable IDs.
- `/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse`
  - Private schema-v2 course pack imported by the app.
- `/private/tmp/mco15-french-pages/`
  - Private rendered French PDF-page workspace.
- `/private/tmp/mco15-french-source-scope.json`
  - Private page boundary record used by inventory and checkpoints.
- `/private/tmp/mco15-french-inventory-check.mjs`
  - Private inventory audit script.
- `/private/tmp/mco15-french-course-audit.mjs`
  - Private course-flow, anti-spam, role, depth, and source-scope audit script.
- `/private/tmp/mco15-french-*.json`
  - Private validation, audit, and import smoke outputs.
- `/Users/admin/Documents/Private Chess Courses/checkpoints/french-black-v1-*/`
  - Private task checkpoints with SHA256 manifests.
- `.superpowers/tmp/french-black-import-smoke.go`
  - Ignored Go smoke harness for disposable and default app-catalogue imports.
- `.superpowers/tmp/french-black-orientation-smoke.go`
  - Ignored Go smoke harness for Black perspective and prompt-role checks.
- `docs/superpowers/specs/2026-07-25-french-black-full-reference-course-design.md`
  - Approved public design.
- `docs/superpowers/plans/2026-07-25-french-black-course.md`
  - This public implementation plan.

---

### Task 1: Prepare implementation branch, render source scope, and checkpoint baseline

**Files:**
- Read: `/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf`
- Read: `/Users/admin/Documents/Private Chess Courses/*.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/*.authoring.json`
- Create: `/private/tmp/mco15-french-pages/*.png`
- Create: `/private/tmp/mco15-french-source-scope.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/french-black-baseline-before-v1/`

**Interfaces:**
- Consumes: approved French design spec and existing private course packs.
- Produces: isolated branch, rendered source pages, source-scope JSON, and recoverable baseline checkpoint.

- [ ] **Step 1: Create or enter the implementation branch**

Run from `/Users/admin/Documents/Work/chess`:

```bash
git status --short --branch
git switch -c codex/french-black-full-reference-course
```

Expected:

- The initial status is clean.
- The current branch becomes `codex/french-black-full-reference-course`.

If the branch already exists, run:

```bash
git switch codex/french-black-full-reference-course
git status --short --branch
```

Expected: status is clean on `codex/french-black-full-reference-course`.

- [ ] **Step 2: Verify required private inputs**

Run:

```bash
test -f "/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf"
for course in \
  "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse"
do
  test -f "$course"
  go run ./cmd/coursepack validate "$course" \
    | jq '{courseId,contentVersion,counts:{lessons:.counts.lessons,activities:.counts.activities,prompts:.counts.prompts},missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length),warnings:.counts.warnings}'
done
```

Expected:

- The PDF exists.
- All six existing private packs validate.
- Every validation summary has `missing: 0` and `unexpected: 0`.

- [ ] **Step 3: Render French source pages privately**

Run:

```bash
mkdir -p /private/tmp/mco15-french-pages
/Users/admin/.cache/codex-runtimes/codex-primary-runtime/dependencies/bin/override/pdftoppm \
  -f 214 -l 260 -png -r 180 \
  "/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf" \
  /private/tmp/mco15-french-pages/french
find /private/tmp/mco15-french-pages -maxdepth 1 -type f -name 'french-*.png' | sort | wc -l
```

Expected: `47`.

- [ ] **Step 4: Write source-scope summary**

Run:

```bash
node - <<'JS' > /private/tmp/mco15-french-source-scope.json
const printedPages = []
for (let page = 197; page <= 243; page += 1) printedPages.push(page)
const teachingFamilies = [
  { familyId: 'french-foundations', title: 'French foundations', minimumDepth: 'quick' },
  { familyId: 'classical-main-spine', title: 'Classical main spine', minimumDepth: 'quick' },
  { familyId: 'advance-variation', title: 'Advance Variation', minimumDepth: 'quick' },
  { familyId: 'exchange-variation', title: 'Exchange Variation', minimumDepth: 'quick' },
  { familyId: 'tarrasch-variation', title: 'Tarrasch Variation', minimumDepth: 'quick' },
  { familyId: 'winawer-variation', title: 'Winawer Variation', minimumDepth: 'standard' },
  { familyId: 'classical-reference', title: 'Classical reference branches', minimumDepth: 'standard' },
  { familyId: 'french-reference-map', title: 'Full French reference map', minimumDepth: 'reference' }
]
const pdfPages = printedPages.map((printedPage) => ({
  printedPage,
  pdfPage: printedPage + 17,
  imagePath: `/private/tmp/mco15-french-pages/french-${String(printedPage + 17).padStart(3, '0')}.png`
}))
const summary = {
  courseId: 'mco15-french-black',
  title: 'French Defence for Black',
  perspective: 'black',
  targetContentVersion: '1.0.0',
  sourcePdf: '/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf',
  printedFirst: 197,
  printedLast: 243,
  pdfFirst: 214,
  pdfLast: 260,
  scanOffset: 17,
  pageCount: printedPages.length,
  teachingFamilies,
  pdfPages
}
console.log(JSON.stringify(summary, null, 2))
JS
jq '{courseId,title,perspective,printedFirst,printedLast,pdfFirst,pdfLast,pageCount,teachingFamilies:[.teachingFamilies[] | {familyId,minimumDepth}]}' \
  /private/tmp/mco15-french-source-scope.json
```

Expected:

- `pageCount` is `47`.
- `printedFirst` is `197`.
- `printedLast` is `243`.
- `pdfFirst` is `214`.
- `pdfLast` is `260`.
- `teachingFamilies` contains exactly eight objects with the IDs shown in the command.

- [ ] **Step 5: Checkpoint baseline**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/french-black-baseline-before-v1"
mkdir -p "$CHECKPOINT"
cp -p /private/tmp/mco15-french-source-scope.json "$CHECKPOINT/source-scope.json"
for file in \
  "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.authoring.json" \
  "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.authoring.json" \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json" \
  "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.authoring.json" \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.authoring.json" \
  "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json"
do
  cp -p "$file" "$CHECKPOINT/"
done
(
  cd "$CHECKPOINT"
  shasum -a 256 * > SHA256SUMS
)
ls -1 "$CHECKPOINT" | sort
```

Expected:

- The checkpoint directory contains the copied existing private packs, copied authoring files, `source-scope.json`, and `SHA256SUMS`.
- `git status --short --ignored` does not show private course files as tracked changes.

---

### Task 2: Build private French source inventory and authoring shell

**Files:**
- Read: `/private/tmp/mco15-french-pages/*.png`
- Read: `/private/tmp/mco15-french-source-scope.json`
- Create: `/Users/admin/Documents/Private Chess Courses/mco15-french-black.authoring.json`
- Create: `/private/tmp/mco15-french-inventory-check.mjs`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/french-black-inventory-v1/`

**Interfaces:**
- Consumes: rendered source pages and source-scope JSON from Task 1.
- Produces: private authoring inventory with stable source coverage IDs that later course-pack moves and notes reference.

- [ ] **Step 1: Initialize the private authoring shell**

Run:

```bash
node - <<'JS'
import fs from 'node:fs'
const scope = JSON.parse(fs.readFileSync('/private/tmp/mco15-french-source-scope.json', 'utf8'))
const teachingNodes = [
  { nodeId: 'foundations-why-e6-d5', title: 'Why ...e6 and ...d5', chapterId: 'foundations', minimumDepth: 'quick' },
  { nodeId: 'foundations-locked-centre', title: 'The locked centre', chapterId: 'foundations', minimumDepth: 'quick' },
  { nodeId: 'foundations-bishop-problem', title: 'The light-square bishop problem', chapterId: 'foundations', minimumDepth: 'quick' },
  { nodeId: 'classical-reach-nf6', title: 'Reach the Classical French', chapterId: 'classical-main-spine', minimumDepth: 'quick' },
  { nodeId: 'classical-solid-be7', title: 'Solid development with ...Be7', chapterId: 'classical-main-spine', minimumDepth: 'quick' },
  { nodeId: 'classical-c5-break', title: 'Timing ...c5', chapterId: 'classical-main-spine', minimumDepth: 'quick' },
  { nodeId: 'advance-survival', title: 'Advance Variation survival map', chapterId: 'major-white-systems', minimumDepth: 'quick' },
  { nodeId: 'exchange-survival', title: 'Exchange Variation survival map', chapterId: 'major-white-systems', minimumDepth: 'quick' },
  { nodeId: 'tarrasch-survival', title: 'Tarrasch Variation survival map', chapterId: 'major-white-systems', minimumDepth: 'quick' },
  { nodeId: 'classical-bg5-main', title: 'Classical pressure after Bg5', chapterId: 'classical-main-spine', minimumDepth: 'standard' },
  { nodeId: 'classical-f6-or-c5', title: 'Choosing ...c5 or ...f6', chapterId: 'classical-main-spine', minimumDepth: 'standard' },
  { nodeId: 'winawer-recognition', title: 'Winawer recognition', chapterId: 'major-white-systems', minimumDepth: 'standard' },
  { nodeId: 'winawer-structure', title: 'Winawer structural tradeoff', chapterId: 'major-white-systems', minimumDepth: 'standard' },
  { nodeId: 'maccutcheon-choice', title: 'MacCutcheon choice', chapterId: 'classical-reference', minimumDepth: 'standard' },
  { nodeId: 'burn-choice', title: 'Burn choice', chapterId: 'classical-reference', minimumDepth: 'standard' },
  { nodeId: 'advance-reference', title: 'Advance reference branches', chapterId: 'french-reference-map', minimumDepth: 'reference' },
  { nodeId: 'exchange-reference', title: 'Exchange reference branches', chapterId: 'french-reference-map', minimumDepth: 'reference' },
  { nodeId: 'tarrasch-reference', title: 'Tarrasch reference branches', chapterId: 'french-reference-map', minimumDepth: 'reference' },
  { nodeId: 'classical-reference', title: 'Classical reference branches', chapterId: 'french-reference-map', minimumDepth: 'reference' },
  { nodeId: 'winawer-reference', title: 'Winawer reference branches', chapterId: 'french-reference-map', minimumDepth: 'reference' },
  { nodeId: 'rare-systems-reference', title: 'Rare French systems', chapterId: 'french-reference-map', minimumDepth: 'reference' }
]
const teachingEdges = [
  ['foundations-why-e6-d5', 'foundations-locked-centre', 'continuation', 'The pawn chain appears', 'quick'],
  ['foundations-locked-centre', 'foundations-bishop-problem', 'continuation', 'Manage the French bishop', 'quick'],
  ['foundations-bishop-problem', 'classical-reach-nf6', 'continuation', 'Enter the Classical', 'quick'],
  ['classical-reach-nf6', 'classical-solid-be7', 'continuation', 'Build the solid setup', 'quick'],
  ['classical-solid-be7', 'classical-c5-break', 'continuation', 'Break the centre', 'quick'],
  ['classical-c5-break', 'advance-survival', 'alternative', 'White advances', 'quick'],
  ['advance-survival', 'exchange-survival', 'alternative', 'White exchanges', 'quick'],
  ['exchange-survival', 'tarrasch-survival', 'alternative', 'White chooses Tarrasch', 'quick'],
  ['classical-solid-be7', 'classical-bg5-main', 'continuation', 'White pins the knight', 'standard'],
  ['classical-bg5-main', 'classical-f6-or-c5', 'continuation', 'Choose the break', 'standard'],
  ['classical-reach-nf6', 'winawer-recognition', 'alternative', 'Black chooses Winawer', 'standard'],
  ['winawer-recognition', 'winawer-structure', 'continuation', 'Understand the imbalance', 'standard'],
  ['classical-bg5-main', 'maccutcheon-choice', 'alternative', 'MacCutcheon branch', 'standard'],
  ['classical-bg5-main', 'burn-choice', 'alternative', 'Burn branch', 'standard'],
  ['advance-survival', 'advance-reference', 'reference', 'Advance reference', 'reference'],
  ['exchange-survival', 'exchange-reference', 'reference', 'Exchange reference', 'reference'],
  ['tarrasch-survival', 'tarrasch-reference', 'reference', 'Tarrasch reference', 'reference'],
  ['classical-f6-or-c5', 'classical-reference', 'reference', 'Classical reference', 'reference'],
  ['winawer-structure', 'winawer-reference', 'reference', 'Winawer reference', 'reference'],
  ['foundations-why-e6-d5', 'rare-systems-reference', 'reference', 'Rare systems', 'reference']
].map(([fromNodeId, toNodeId, kind, label, minimumDepth], index) => ({
  edgeId: `edge-${String(index + 1).padStart(2, '0')}`,
  fromNodeId,
  toNodeId,
  kind,
  label,
  minimumDepth,
  ordinal: index + 1
}))
const authoring = {
  scope,
  renderedPdfPages: scope.pdfPages,
  inventory: [],
  illegibleItems: [],
  paths: {
    sourcePdf: scope.sourcePdf,
    authoring: '/Users/admin/Documents/Private Chess Courses/mco15-french-black.authoring.json',
    coursePack: '/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse'
  },
  teachingNodes,
  teachingEdges,
  noteOverrides: {}
}
fs.writeFileSync(authoring.paths.authoring, JSON.stringify(authoring, null, 2) + '\n')
console.log(authoring.paths.authoring)
JS
jq '{scope:{courseId:.scope.courseId,pageCount:.scope.pageCount},nodes:(.teachingNodes|length),edges:(.teachingEdges|length),inventory:(.inventory|length)}' \
  "/Users/admin/Documents/Private Chess Courses/mco15-french-black.authoring.json"
```

Expected:

- The authoring shell is created.
- `nodes` is `21`.
- `edges` is `20`.
- `inventory` is `0` before visual source inventory.

- [ ] **Step 2: Visually inventory the French source scope**

Open rendered pages in batches:

```bash
open /private/tmp/mco15-french-pages/french-214.png
open /private/tmp/mco15-french-pages/french-220.png
open /private/tmp/mco15-french-pages/french-226.png
open /private/tmp/mco15-french-pages/french-232.png
open /private/tmp/mco15-french-pages/french-238.png
open /private/tmp/mco15-french-pages/french-243.png
```

Edit `/Users/admin/Documents/Private Chess Courses/mco15-french-black.authoring.json` and add one `inventory` object for every source table row, named variation heading, note-letter item, and illustrative-game record in printed pages `197-243`. Use this exact object shape:

```json
{
  "coverageId": "fr-p197-overview-01",
  "printedPage": 197,
  "pdfPage": 214,
  "familyId": "french-foundations",
  "recordType": "overview",
  "tableColumn": "",
  "noteLabel": "",
  "variationLabel": "French Defence overview",
  "teachingNodeIds": ["foundations-why-e6-d5"],
  "minimumDepth": "quick",
  "imagePath": "/private/tmp/mco15-french-pages/french-214.png",
  "privateSummary": "Private source summary written in original words for authoring only."
}
```

Required inventory rules:

- `coverageId` starts with `fr-p` and is globally unique.
- `printedPage` is between `197` and `243`.
- `pdfPage` equals `printedPage + 17`.
- `familyId` is one of the eight IDs in `/private/tmp/mco15-french-source-scope.json`.
- `recordType` is one of `overview`, `table`, `note`, `variation-heading`, `illustrative-game`, or `illegible`.
- `minimumDepth` is one of `quick`, `standard`, or `reference`.
- `teachingNodeIds` contains at least one node ID from `teachingNodes`.
- `privateSummary` uses original wording and stays private.
- Any unreadable source item is recorded in `illegibleItems` with printed page, pdf page, image path, and reason.

Expected:

- The inventory covers every relevant source item in the French section.
- The public repository remains unchanged after this private edit.

- [ ] **Step 3: Create inventory audit script**

Create `/private/tmp/mco15-french-inventory-check.mjs` with this exact content:

```javascript
const fs = require('fs')

const authoringPath = '/Users/admin/Documents/Private Chess Courses/mco15-french-black.authoring.json'
const authoring = JSON.parse(fs.readFileSync(authoringPath, 'utf8'))
const scope = authoring.scope
const inventory = authoring.inventory || []
const families = new Set((scope.teachingFamilies || []).map((family) => family.familyId))
const nodes = new Set((authoring.teachingNodes || []).map((node) => node.nodeId))
const allowedRecordTypes = new Set(['overview', 'table', 'note', 'variation-heading', 'illustrative-game', 'illegible'])
const allowedDepths = new Set(['quick', 'standard', 'reference'])
const failures = []
const seen = new Set()
const pages = new Set()

for (const record of inventory) {
  const path = record.coverageId || '<missing coverageId>'
  if (!record.coverageId || !record.coverageId.startsWith('fr-p')) failures.push(`${path}: coverageId must start with fr-p`)
  if (seen.has(record.coverageId)) failures.push(`${path}: duplicate coverageId`)
  seen.add(record.coverageId)
  if (!Number.isInteger(record.printedPage) || record.printedPage < 197 || record.printedPage > 243) failures.push(`${path}: printedPage outside 197-243`)
  if (record.pdfPage !== record.printedPage + 17) failures.push(`${path}: pdfPage must equal printedPage + 17`)
  if (!families.has(record.familyId)) failures.push(`${path}: unknown familyId ${record.familyId}`)
  if (!allowedRecordTypes.has(record.recordType)) failures.push(`${path}: invalid recordType ${record.recordType}`)
  if (!allowedDepths.has(record.minimumDepth)) failures.push(`${path}: invalid minimumDepth ${record.minimumDepth}`)
  if (!Array.isArray(record.teachingNodeIds) || record.teachingNodeIds.length === 0) failures.push(`${path}: teachingNodeIds must be non-empty`)
  for (const nodeId of record.teachingNodeIds || []) {
    if (!nodes.has(nodeId)) failures.push(`${path}: unknown teaching node ${nodeId}`)
  }
  if (!record.imagePath || !fs.existsSync(record.imagePath)) failures.push(`${path}: imagePath must exist`)
  if (!record.privateSummary || record.privateSummary.trim().length < 20) failures.push(`${path}: privateSummary must be meaningful`)
  pages.add(record.printedPage)
}

const missingPages = []
for (let page = 197; page <= 243; page += 1) {
  if (!pages.has(page)) missingPages.push(page)
}
if (missingPages.length > 0) failures.push(`missing inventory pages: ${missingPages.join(', ')}`)

const summary = {
  authoringPath,
  records: inventory.length,
  uniqueCoverageIds: seen.size,
  pagesCovered: pages.size,
  missingPages,
  illegibleItems: (authoring.illegibleItems || []).length,
  failures
}
console.log(JSON.stringify(summary, null, 2))
if (failures.length > 0) process.exit(1)
```

- [ ] **Step 4: Run inventory audit and checkpoint**

Run:

```bash
node /private/tmp/mco15-french-inventory-check.mjs \
  > /private/tmp/mco15-french-inventory-check.json
jq . /private/tmp/mco15-french-inventory-check.json
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/french-black-inventory-v1"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-french-black.authoring.json" "$CHECKPOINT/"
cp -p /private/tmp/mco15-french-source-scope.json "$CHECKPOINT/"
cp -p /private/tmp/mco15-french-inventory-check.json "$CHECKPOINT/"
(
  cd "$CHECKPOINT"
  shasum -a 256 * > SHA256SUMS
)
```

Expected:

- `failures` is `[]`.
- `pagesCovered` is `47`.
- The checkpoint contains the authoring JSON, source-scope JSON, audit JSON, and `SHA256SUMS`.

---

### Task 3: Build and validate the Quick Classical French layer

**Files:**
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-french-black.authoring.json`
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse`
- Create: `/private/tmp/mco15-french-quick-validation.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/french-black-quick-v1/`

**Interfaces:**
- Consumes: source inventory and teaching nodes from Task 2.
- Produces: a valid schema-v2 French course pack with the complete Quick path.

- [ ] **Step 1: Create the initial schema-v2 course pack**

Create `/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse` as JSON with this required top-level shape:

```json
{
  "schemaVersion": 2,
  "courseId": "mco15-french-black",
  "contentVersion": "1.0.0",
  "title": "French Defence for Black",
  "description": "A Black repertoire and reference course for the French Defence, centered on a solid Classical setup with full-family reference branches.",
  "perspective": "black",
  "defaultDepth": "reference",
  "rootPositionId": "initial",
  "rootFen": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
  "source": {
    "title": "Modern Chess Openings",
    "edition": "15th edition",
    "privateUseNotice": "Private course pack generated from the user's local source review. Do not commit source-derived content."
  },
  "sourceCoverage": {
    "printedPages": [197, 198, 199, 200, 201, 202, 203, 204, 205, 206, 207, 208, 209, 210, 211, 212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225, 226, 227, 228, 229, 230, 231, 232, 233, 234, 235, 236, 237, 238, 239, 240, 241, 242, 243],
    "expectedReferences": []
  },
  "positions": [
    {
      "positionId": "initial",
      "label": "Initial position",
      "evaluation": {"code": "none"},
      "noteIds": []
    }
  ],
  "moves": [],
  "notes": [],
  "lessonEdges": [],
  "chapters": [
    {"chapterId":"foundations","ordinal":1,"title":"French Foundations","overview":"Learn why Black answers 1.e4 with ...e6 and a later ...d5.","minimumDepth":"quick"},
    {"chapterId":"classical-main-spine","ordinal":2,"title":"Classical French Main Spine","overview":"Build a solid Classical French setup around ...Nf6, ...Be7, castling, and central pressure.","minimumDepth":"quick"},
    {"chapterId":"major-white-systems","ordinal":3,"title":"Major White Systems","overview":"Recognize Advance, Exchange, Tarrasch, and Winawer structures.","minimumDepth":"quick"},
    {"chapterId":"classical-reference","ordinal":4,"title":"Classical Reference Branches","overview":"Study MacCutcheon, Burn, and related Classical choices.","minimumDepth":"standard"},
    {"chapterId":"french-reference-map","ordinal":5,"title":"French Reference Map","overview":"Browse the full selected French source scope after the main repertoire is understood.","minimumDepth":"reference"}
  ],
  "lessons": [],
  "prompts": []
}
```

During incremental construction, keep `sourceCoverage.expectedReferences` equal
to the sorted set of coverage IDs currently used by `moves[].sourceRef` and
`notes[].sourceRef`. Task 5 expands it to the full private inventory.

Expected:

- The pack uses schema version `2`.
- It has five chapters.
- It contains only private course data and is not staged in Git.

- [ ] **Step 2: Add Quick opening spine moves and positions**

Use the SAN helper to derive UCI moves and FENs for the core spine:

```bash
go run ./cmd/coursepack sanline \
  --fen "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1" \
  e4 e6 d4 d5 Nc3 Nf6
```

Expected:

- The command returns six moves.
- The Black moves `...e6`, `...d5`, and `...Nf6` become `trainingRole: "repertoire"`.
- White moves become `trainingRole: "opponent"`.

Add positions and moves for:

- `initial`
- `after-e4`
- `after-e6`
- `after-d4`
- `after-d5`
- `after-nc3`
- `after-nf6`

Every move must have:

- `minimumDepth: "quick"`
- a real `sourceRef.coverageId` from the private inventory;
- `evaluation.code` set to one of the allowed values in `internal/openings/types.go`.

After adding Quick moves and notes, set `sourceCoverage.expectedReferences` to
the sorted list of coverage IDs currently used by those Quick moves and notes.

- [ ] **Step 3: Add exactly nine Quick lessons**

Add these Quick lessons to `lessons`:

1. `french-why-e6-d5`
2. `french-locked-centre`
3. `french-bishop-problem`
4. `classical-reach-nf6`
5. `classical-solid-be7`
6. `classical-c5-break`
7. `advance-survival-map`
8. `exchange-survival-map`
9. `tarrasch-survival-map`

Each Quick lesson must have:

- `minimumDepth: "quick"`;
- 2-3 required activities;
- at least one required `decision` activity when it teaches a Black move;
- at most one required decision per prompt position;
- optional `reference` activities only when useful;
- original instructional prose written for the app, not source transcription.

Use this required activity pattern for each strategic lesson:

```json
[
  {
    "activityId": "classical-solid-be7-concept",
    "kind": "concept",
    "title": "Coordinate before breaking",
    "instruction": "Black develops calmly before choosing the central break.",
    "required": true,
    "positionId": "after-nf6",
    "noteIds": ["classical-solid-setup"],
    "moveIds": []
  },
  {
    "activityId": "classical-solid-be7-decision",
    "kind": "decision",
    "title": "Choose Black's solid setup",
    "instruction": "Choose the developing move that keeps the French structure coherent.",
    "required": true,
    "positionId": "after-nf6",
    "noteIds": [],
    "moveIds": [],
    "promptId": "prompt-black-be7"
  },
  {
    "activityId": "classical-solid-be7-recap",
    "kind": "recap",
    "title": "The setup is the point",
    "instruction": "Black is not solving everything at once; the setup prepares pressure and breaks.",
    "required": true,
    "positionId": "after-be7",
    "noteIds": [],
    "moveIds": []
  }
]
```

Change IDs, titles, positions, notes, and prompts for each lesson so no two required decisions repeat the same prompt.

- [ ] **Step 4: Add Quick lesson edges**

Add `lessonEdges` for the Quick route:

```text
french-why-e6-d5 -> french-locked-centre
french-locked-centre -> french-bishop-problem
french-bishop-problem -> classical-reach-nf6
classical-reach-nf6 -> classical-solid-be7
classical-solid-be7 -> classical-c5-break
classical-c5-break -> advance-survival-map
advance-survival-map -> exchange-survival-map
exchange-survival-map -> tarrasch-survival-map
```

Each edge must use:

- `kind: "continuation"` for the main path;
- `minimumDepth: "quick"`;
- increasing `ordinal` values.

- [ ] **Step 5: Validate the Quick pack**

Run:

```bash
go run ./cmd/coursepack validate "/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse" \
  > /private/tmp/mco15-french-quick-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length),diagnostics}' \
  /private/tmp/mco15-french-quick-validation.json
```

Expected:

- `courseId` is `mco15-french-black`.
- `contentVersion` is `1.0.0`.
- `counts.lessons` is `9`.
- `counts.prompts` is at least `4`.
- `diagnostics` is `[]`.
- `missing` is `0`.
- `unexpected` is `0`.

- [ ] **Step 6: Checkpoint Quick layer**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/french-black-quick-v1"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-french-black.authoring.json" "$CHECKPOINT/"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse" "$CHECKPOINT/"
cp -p /private/tmp/mco15-french-quick-validation.json "$CHECKPOINT/"
(
  cd "$CHECKPOINT"
  shasum -a 256 * > SHA256SUMS
)
```

Expected: checkpoint files are present and private.

---

### Task 4: Build Standard layer with major French decisions

**Files:**
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-french-black.authoring.json`
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse`
- Create: `/private/tmp/mco15-french-standard-validation.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/french-black-standard-v1/`

**Interfaces:**
- Consumes: valid Quick pack from Task 3.
- Produces: valid Standard-depth practical French repertoire layer.

- [ ] **Step 1: Add exactly twelve Standard lessons**

Add these Standard lessons:

1. `classical-bg5-pressure`
2. `classical-be7-coordination`
3. `classical-c5-or-f6`
4. `advance-pawn-chain-targets`
5. `advance-piece-placement`
6. `exchange-equalize-with-purpose`
7. `tarrasch-structure-choice`
8. `tarrasch-central-pressure`
9. `winawer-recognition`
10. `winawer-structural-tradeoff`
11. `maccutcheon-choice`
12. `burn-choice`

Each Standard lesson must have:

- `minimumDepth: "standard"`;
- 3-4 required activities;
- at least one comparison or demonstration activity when teaching a structure choice;
- no required `reference` activity;
- Black repertoire prompts only for Black moves.

Expected after this step:

- Quick lessons remain unchanged.
- Total lesson count becomes `21`.
- `sourceCoverage.expectedReferences` is the sorted list of coverage IDs
  currently used by all Quick and Standard moves and notes.

- [ ] **Step 2: Add Standard branch edges**

Add Standard edges from the Quick tree:

- `classical-solid-be7 -> classical-bg5-pressure`
- `classical-bg5-pressure -> classical-be7-coordination`
- `classical-be7-coordination -> classical-c5-or-f6`
- `advance-survival-map -> advance-pawn-chain-targets`
- `advance-pawn-chain-targets -> advance-piece-placement`
- `exchange-survival-map -> exchange-equalize-with-purpose`
- `tarrasch-survival-map -> tarrasch-structure-choice`
- `tarrasch-structure-choice -> tarrasch-central-pressure`
- `classical-reach-nf6 -> winawer-recognition`
- `winawer-recognition -> winawer-structural-tradeoff`
- `classical-bg5-pressure -> maccutcheon-choice`
- `classical-bg5-pressure -> burn-choice`

Use:

- `kind: "continuation"` for same-family expansion;
- `kind: "alternative"` when a branch represents a different Black choice;
- `minimumDepth: "standard"`.

- [ ] **Step 3: Validate Standard layer**

Run:

```bash
go run ./cmd/coursepack validate "/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse" \
  > /private/tmp/mco15-french-standard-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length),diagnostics}' \
  /private/tmp/mco15-french-standard-validation.json
```

Expected:

- `counts.lessons` is `21`.
- `counts.prompts` is at least `12`.
- `counts.warnings` is `0`.
- `diagnostics` is `[]`.
- `missing` is `0`.
- `unexpected` is `0`.

- [ ] **Step 4: Checkpoint Standard layer**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/french-black-standard-v1"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-french-black.authoring.json" "$CHECKPOINT/"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse" "$CHECKPOINT/"
cp -p /private/tmp/mco15-french-standard-validation.json "$CHECKPOINT/"
(
  cd "$CHECKPOINT"
  shasum -a 256 * > SHA256SUMS
)
```

Expected: checkpoint files are present and private.

---

### Task 5: Build full Reference tree and source coverage

**Files:**
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-french-black.authoring.json`
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse`
- Create: `/private/tmp/mco15-french-reference-validation.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/french-black-reference-v1/`

**Interfaces:**
- Consumes: valid Standard pack and audited source inventory.
- Produces: full-reference French course pack with no missing or unexpected source coverage records.

- [ ] **Step 1: Add Reference lessons for every inventory family**

Add Reference lessons and lesson edges so every inventory record belongs to at least one lesson in one of these branches:

- `advance-reference`
- `exchange-reference`
- `tarrasch-reference`
- `classical-reference`
- `winawer-reference`
- `rare-systems-reference`

Reference lesson requirements:

- `minimumDepth: "reference"`;
- at least one required `concept`, `demonstration`, `comparison`, or `recap` activity;
- every `reference` activity has `required: false`;
- dense source records become optional reference cards, original explanations, or comparison activities;
- no raw source-table transcription appears in public files.

Expected:

- Total lesson count is at least `35`.
- Reference branches are connected from Quick or Standard lessons through `kind: "reference"` edges.

- [ ] **Step 2: Complete source coverage**

For every object in `authoring.inventory`, ensure its `coverageId` appears in at least one `sourceRef.coverageId` in:

- `moves[]`
- `notes[]`

Then set:

```json
"sourceCoverage": {
  "printedPages": [197, 198, 199, 200, 201, 202, 203, 204, 205, 206, 207, 208, 209, 210, 211, 212, 213, 214, 215, 216, 217, 218, 219, 220, 221, 222, 223, 224, 225, 226, 227, 228, 229, 230, 231, 232, 233, 234, 235, 236, 237, 238, 239, 240, 241, 242, 243],
  "expectedReferences": []
}
```

Populate `expectedReferences` with every private inventory `coverageId`, sorted lexicographically.

- [ ] **Step 3: Validate full Reference pack**

Run:

```bash
go run ./cmd/coursepack validate "/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse" \
  > /private/tmp/mco15-french-reference-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length),diagnostics}' \
  /private/tmp/mco15-french-reference-validation.json
```

Expected:

- `counts.lessons` is at least `35`.
- `counts.prompts` is at least `15`.
- `counts.warnings` is `0`.
- `missing` is `0`.
- `unexpected` is `0`.
- `diagnostics` is `[]`.

- [ ] **Step 4: Checkpoint Reference layer**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/french-black-reference-v1"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-french-black.authoring.json" "$CHECKPOINT/"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse" "$CHECKPOINT/"
cp -p /private/tmp/mco15-french-reference-validation.json "$CHECKPOINT/"
(
  cd "$CHECKPOINT"
  shasum -a 256 * > SHA256SUMS
)
```

Expected: checkpoint files are present and private.

---

### Task 6: Add final private audits for spam, roles, depth, and course shape

**Files:**
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse`
- Create: `/private/tmp/mco15-french-course-audit.mjs`
- Create: `/private/tmp/mco15-french-final-audit.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/french-black-audited-v1/`

**Interfaces:**
- Consumes: full Reference pack from Task 5.
- Produces: machine-readable private audit proving the pack matches the lesson-design constraints.

- [ ] **Step 1: Create final course audit script**

Create `/private/tmp/mco15-french-course-audit.mjs` with this exact content:

```javascript
const fs = require('fs')

const packPath = process.argv[2] || '/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse'
const pack = JSON.parse(fs.readFileSync(packPath, 'utf8'))
const failures = []
const moves = new Map((pack.moves || []).map((move) => [move.moveId, move]))
const prompts = new Map((pack.prompts || []).map((prompt) => [prompt.promptId, prompt]))
const spamPatterns = [
  /Source note [a-z] records/i,
  /Course move found/i,
  /activity result/i,
  /appliedMoves must be an array/i,
  /private reference only/i
]

function fail(message) {
  failures.push(message)
}

if (pack.schemaVersion !== 2) fail('schemaVersion must be 2')
if (pack.courseId !== 'mco15-french-black') fail('courseId must be mco15-french-black')
if (pack.contentVersion !== '1.0.0') fail('contentVersion must be 1.0.0')
if (pack.perspective !== 'black') fail('perspective must be black')
if (pack.defaultDepth !== 'reference') fail('defaultDepth must be reference')
if (!Array.isArray(pack.sourceCoverage?.printedPages) || pack.sourceCoverage.printedPages.length !== 47) fail('printed page coverage must include 47 pages')

for (const text of JSON.stringify(pack).split('\n')) {
  for (const pattern of spamPatterns) {
    if (pattern.test(text)) fail(`spam text matched ${pattern}: ${text.slice(0, 160)}`)
  }
}

const lessons = pack.lessons || []
if (lessons.length < 35) fail(`lesson count ${lessons.length} is below 35`)
const quickLessons = lessons.filter((lesson) => lesson.minimumDepth === 'quick')
const standardLessons = lessons.filter((lesson) => lesson.minimumDepth === 'standard')
const referenceLessons = lessons.filter((lesson) => lesson.minimumDepth === 'reference')
if (quickLessons.length !== 9) fail(`quick lesson count ${quickLessons.length} must be 9`)
if (standardLessons.length !== 12) fail(`standard lesson count ${standardLessons.length} must be 12`)
if (referenceLessons.length < 14) fail(`reference lesson count ${referenceLessons.length} must be at least 14`)

for (const lesson of lessons) {
  const requiredActivities = (lesson.activities || []).filter((activity) => activity.required)
  if (requiredActivities.length === 0) fail(`${lesson.lessonId}: no required activities`)
  const decisionKeys = new Set()
  for (const activity of lesson.activities || []) {
    if (activity.kind === 'reference' && activity.required) fail(`${lesson.lessonId}/${activity.activityId}: reference activity is required`)
    if (activity.kind !== 'decision') continue
    const prompt = prompts.get(activity.promptId)
    if (!prompt) {
      fail(`${lesson.lessonId}/${activity.activityId}: missing prompt ${activity.promptId}`)
      continue
    }
    const key = `${prompt.positionId}:${prompt.primaryMoveId}`
    if (activity.required && decisionKeys.has(key)) fail(`${lesson.lessonId}: repeated required decision ${key}`)
    if (activity.required) decisionKeys.add(key)
    const move = moves.get(prompt.primaryMoveId)
    if (!move) {
      fail(`${lesson.lessonId}/${activity.activityId}: missing primary move ${prompt.primaryMoveId}`)
      continue
    }
    if (move.trainingRole !== 'repertoire' && move.trainingRole !== 'alternative') {
      fail(`${lesson.lessonId}/${activity.activityId}: primary move ${move.moveId} has role ${move.trainingRole}`)
    }
  }
}

const expected = new Set(pack.sourceCoverage?.expectedReferences || [])
const used = new Set()
for (const move of pack.moves || []) {
  if (move.sourceRef?.coverageId) used.add(move.sourceRef.coverageId)
}
for (const note of pack.notes || []) {
  if (note.sourceRef?.coverageId) used.add(note.sourceRef.coverageId)
}
const missing = [...expected].filter((id) => !used.has(id)).sort()
const unexpected = [...used].filter((id) => !expected.has(id)).sort()
if (missing.length > 0) fail(`missing expected coverage IDs: ${missing.join(', ')}`)
if (unexpected.length > 0) fail(`unexpected coverage IDs: ${unexpected.join(', ')}`)

const summary = {
  packPath,
  courseId: pack.courseId,
  contentVersion: pack.contentVersion,
  perspective: pack.perspective,
  defaultDepth: pack.defaultDepth,
  lessons: lessons.length,
  quickLessons: quickLessons.length,
  standardLessons: standardLessons.length,
  referenceLessons: referenceLessons.length,
  moves: (pack.moves || []).length,
  prompts: (pack.prompts || []).length,
  notes: (pack.notes || []).length,
  expectedCoverage: expected.size,
  usedCoverage: used.size,
  failures
}
console.log(JSON.stringify(summary, null, 2))
if (failures.length > 0) process.exit(1)
```

- [ ] **Step 2: Run final course audit**

Run:

```bash
node /private/tmp/mco15-french-course-audit.mjs \
  "/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse" \
  > /private/tmp/mco15-french-final-audit.json
jq . /private/tmp/mco15-french-final-audit.json
```

Expected:

- `failures` is `[]`.
- `quickLessons` is `9`.
- `standardLessons` is `12`.
- `referenceLessons` is at least `14`.

- [ ] **Step 3: Repeat validator after audit**

Run:

```bash
go run ./cmd/coursepack validate "/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse" \
  > /private/tmp/mco15-french-final-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length),diagnostics}' \
  /private/tmp/mco15-french-final-validation.json
```

Expected:

- `counts.warnings` is `0`.
- `missing` is `0`.
- `unexpected` is `0`.
- `diagnostics` is `[]`.

- [ ] **Step 4: Checkpoint audited final pack**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/french-black-audited-v1"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-french-black.authoring.json" "$CHECKPOINT/"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse" "$CHECKPOINT/"
cp -p /private/tmp/mco15-french-final-audit.json "$CHECKPOINT/"
cp -p /private/tmp/mco15-french-final-validation.json "$CHECKPOINT/"
(
  cd "$CHECKPOINT"
  shasum -a 256 * > SHA256SUMS
)
```

Expected: checkpoint files are present and private.

---

### Task 7: Import French and verify app-level behavior

**Files:**
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse`
- Create: `.superpowers/tmp/french-black-import-smoke.go`
- Create: `.superpowers/tmp/french-black-orientation-smoke.go`
- Create: `/private/tmp/mco15-french-disposable-import.json`
- Create: `/private/tmp/mco15-french-default-import.json`
- Create: `/private/tmp/mco15-french-orientation-smoke.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/french-black-imported-v1/`

**Interfaces:**
- Consumes: audited final private course pack.
- Produces: imported default catalogue, app-level smoke proof, orientation proof, and import checkpoint.

- [ ] **Step 1: Create import smoke harness**

Create `.superpowers/tmp/french-black-import-smoke.go` with this exact content:

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
	if os.Getenv("FRENCH_USE_DEFAULT_DATA") == "1" {
		var err error
		paths, err = storage.DefaultPaths()
		if err != nil {
			panic(err)
		}
	} else {
		dataRoot := os.Getenv("FRENCH_DATA_ROOT")
		if dataRoot == "" {
			fmt.Fprintln(os.Stderr, "FRENCH_DATA_ROOT or FRENCH_USE_DEFAULT_DATA=1 is required")
			os.Exit(2)
		}
		paths = storage.PathsAt(dataRoot)
	}
	coursePaths := strings.Split(os.Getenv("FRENCH_COURSES"), "|")
	if len(coursePaths) == 0 || coursePaths[0] == "" {
		fmt.Fprintln(os.Stderr, "FRENCH_COURSES is required")
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
FRENCH_DATA_ROOT="$(mktemp -d /private/tmp/mco15-french-import.XXXXXX)" \
FRENCH_COURSES="/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse|/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse|/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse|/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse|/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse|/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse|/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse" \
go run .superpowers/tmp/french-black-import-smoke.go \
  > /private/tmp/mco15-french-disposable-import.json
jq '{courses:[.home.courses[] | {courseId,title,perspective}],activeVersions}' \
  /private/tmp/mco15-french-disposable-import.json
```

Expected:

- seven active courses are present.
- `mco15-french-black` is present.
- `activeVersions["mco15-french-black"]` is `1.0.0`.

- [ ] **Step 3: Import French into the default local catalogue**

Run:

```bash
FRENCH_USE_DEFAULT_DATA=1 \
FRENCH_COURSES="/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse" \
go run .superpowers/tmp/french-black-import-smoke.go \
  > /private/tmp/mco15-french-default-import.json
jq '{imports:[.imports[] | {path,report:.report}],courses:[.home.courses[] | {courseId,title,perspective}],activeVersions}' \
  /private/tmp/mco15-french-default-import.json
```

Expected:

- import report for French has `Accepted: 1`.
- `mco15-french-black` is active in the default local catalogue.
- existing courses remain active.

- [ ] **Step 4: Create Black orientation and role smoke harness**

Create `.superpowers/tmp/french-black-orientation-smoke.go` with this exact content:

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

	compiled, err := services.OpeningCatalog.LoadActive(ctx, "mco15-french-black")
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
		"ok":                  compiled.Pack.CourseID == "mco15-french-black" && compiled.Pack.ContentVersion == "1.0.0" && compiled.Pack.Perspective == "black" && compiled.Pack.DefaultDepth == "reference" && len(invalidPrimaryRoles) == 0,
	}
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(result)
	if !result["ok"].(bool) {
		os.Exit(1)
	}
}
```

- [ ] **Step 5: Run orientation smoke and checkpoint import**

Run:

```bash
go run .superpowers/tmp/french-black-orientation-smoke.go \
  > /private/tmp/mco15-french-orientation-smoke.json
jq . /private/tmp/mco15-french-orientation-smoke.json
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/french-black-imported-v1"
mkdir -p "$CHECKPOINT"
cp -p /private/tmp/mco15-french-disposable-import.json "$CHECKPOINT/"
cp -p /private/tmp/mco15-french-default-import.json "$CHECKPOINT/"
cp -p /private/tmp/mco15-french-orientation-smoke.json "$CHECKPOINT/"
(
  cd "$CHECKPOINT"
  shasum -a 256 * > SHA256SUMS
)
```

Expected:

- `ok` is `true`.
- `perspective` is `black`.
- `defaultDepth` is `reference`.
- `invalidPrimaryRoles` is `[]`.

---

### Task 8: Run repository verification, manual app check, and commit public marker

**Files:**
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse`
- Create or modify: `docs/superpowers/plans/2026-07-25-french-black-course.md`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/french-black-v1-final/`

**Interfaces:**
- Consumes: imported French course and audit outputs from Tasks 6-7.
- Produces: final verification record, public implementation marker commit, and clean branch.

- [ ] **Step 1: Run repository tests**

Run:

```bash
go test ./...
npm --prefix frontend run test:e2e -- frontend/tests/openings.spec.ts --project=chromium
npm --prefix frontend run test:e2e -- frontend/tests/openings.spec.ts --project=webkit
```

Expected:

- Go tests pass.
- Opening E2E Chromium passes.
- Opening E2E WebKit passes.

- [ ] **Step 2: Manually verify Learn Openings in the app**

Open the app and verify:

```text
French Defence for Black appears under Black openings.
Quick French starts from the French course, not another course.
The board is oriented for Black.
Quick, Standard, and Reference French lessons can each be started.
At least one Quick French lesson can be completed.
Continue Learning resumes the French course when French owns the active session.
The French variation explorer opens from the course root.
Deep Reference branches scroll while the board remains visible.
No generic source-note spam, raw activity-result text, or Course move found text appears.
```

Expected: every line is true.

- [ ] **Step 3: Create final private checkpoint**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/french-black-v1-final"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-french-black.authoring.json" "$CHECKPOINT/"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse" "$CHECKPOINT/"
cp -p /private/tmp/mco15-french-source-scope.json "$CHECKPOINT/"
cp -p /private/tmp/mco15-french-inventory-check.json "$CHECKPOINT/"
cp -p /private/tmp/mco15-french-final-audit.json "$CHECKPOINT/"
cp -p /private/tmp/mco15-french-final-validation.json "$CHECKPOINT/"
cp -p /private/tmp/mco15-french-disposable-import.json "$CHECKPOINT/"
cp -p /private/tmp/mco15-french-default-import.json "$CHECKPOINT/"
cp -p /private/tmp/mco15-french-orientation-smoke.json "$CHECKPOINT/"
(
  cd "$CHECKPOINT"
  shasum -a 256 * > SHA256SUMS
)
```

Expected: final checkpoint contains every private artifact needed to recover or audit the French v1 work.

- [ ] **Step 4: Confirm no private artifacts are tracked**

Run:

```bash
git status --short --ignored
git ls-files | rg 'mco15-french|Private Chess Courses|mco15-french-pages|french-black.*\\.ctcourse|french-black.*authoring' || true
```

Expected:

- Private files may appear only as ignored or untracked paths.
- `git ls-files` prints nothing for private French course artifacts.

- [ ] **Step 5: Add public implementation marker and commit**

Append this completion note to `docs/superpowers/plans/2026-07-25-french-black-course.md` after all validation passes:

```markdown
## Implementation status

- Private course pack: `/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse`
- Private authoring file: `/Users/admin/Documents/Private Chess Courses/mco15-french-black.authoring.json`
- Content version: `1.0.0`
- Final validation: passed
- Final private audit: passed
- Disposable import smoke: passed
- Default catalogue import: passed
- Manual app verification: passed
```

Then run:

```bash
git add docs/superpowers/plans/2026-07-25-french-black-course.md
git commit -m "docs: mark french black course implemented"
git status --short --branch
```

Expected:

- The commit contains only public docs.
- The worktree is clean except ignored private or temporary files.
