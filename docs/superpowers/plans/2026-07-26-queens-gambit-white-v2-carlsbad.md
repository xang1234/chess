# Queen's Gambit White v2 Carlsbad Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deepen the private `mco15-queens-gambit-white` course to v2.0.0 with an Exchange QGD / Carlsbad teaching spine and balanced QGD/QGA/Slav-Semi-Slav Standard coverage.

**Architecture:** Reuse the existing schema-v2 `.ctcourse` model, private authoring JSON, coursepack validator, importer, teaching-tree UI, journey rebase behavior, and app-service smoke harnesses. Update only private course files unless the private pack exposes a generic app defect; public Git records the approved design/plan and final implementation status.

**Tech Stack:** Go coursepack validator/importer, schema-v2 opening course JSON, private Node.js audit/generation scripts, `jq`, Poppler-rendered source pages from the private MCO-15 PDF, Playwright opening E2E, Wails/Svelte app runtime.

## Global Constraints

- Existing course ID: `mco15-queens-gambit-white`
- New content version: `2.0.0`
- Title: `Queen's Gambit for White`
- Perspective: `white`
- Default depth: `reference`
- Private authoring file: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json`
- Private course pack: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse`
- Main spine: Exchange QGD / Carlsbad.
- Quick depth: flexible structure lesson; do not force one brittle Exchange move order.
- Standard depth: strategic comparisons plus branch recognition.
- Required Standard coverage: QGD, QGA, Slav, and Semi-Slav.
- Reference depth: full Queen's Gambit map, with Tarrasch and Chigorin optional unless source-scope audit excludes them.
- Use the existing Queen's Gambit White source scope and private MCO-15 PDF workflow from v1.
- White repertoire moves are learner decisions.
- Black moves are opponent pressure, branch choices, or demonstrations.
- Avoid repeated required prompts from the same position inside one lesson.
- Do not add generic filler such as `Source note x records...`, raw activity-result text, or `Course move found`.
- Warnings must be zero.
- Source coverage must have no missing or unexpected records.
- Replace only `mco15-queens-gambit-white`.
- Do not create a new course ID.
- Do not add King's Indian, Nimzo-Indian, Queen's Indian, Grunfeld, Dutch, English, Reti, or Catalan.
- Do not deepen Queen's Gambit Defences for Black in this pass.
- Do not special-case Queen's Gambit in public app code.
- Do not commit private `.ctcourse`, private authoring JSON, rendered PDF pages, private checkpoints, private audit reports, private manifests, or MCO-derived course prose to Git.
- Public reports and final responses may contain only IDs, paths, counts, validation results, and high-level summaries.

---

## File Structure

Repository files:

- Read: `docs/superpowers/specs/2026-07-26-queens-gambit-white-v2-carlsbad-design.md`
- Read: `docs/operations/opening-course-authoring.md`
- Create: `docs/superpowers/plans/2026-07-26-queens-gambit-white-v2-carlsbad.md`
- Modify at the end only: `docs/superpowers/specs/2026-07-26-queens-gambit-white-v2-carlsbad-design.md`
- Modify at the end only: `docs/superpowers/plans/2026-07-26-queens-gambit-white-v2-carlsbad.md`
- Create and keep ignored: `.superpowers/tmp/queens-gambit-white-v2-import-smoke.go`
- Create and keep ignored: `.superpowers/tmp/queens-gambit-white-v2-service-smoke.go`

Private files outside Git:

- Read/update: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json`
- Read/update: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse`
- Create: `/private/tmp/qgw-v2-source-scope.json`
- Create: `/private/tmp/qgw-v2-current-validation.json`
- Create: `/private/tmp/qgw-v2-current-pack-audit.json`
- Create: `/private/tmp/qgw-v2-inventory-check.json`
- Create: `/private/tmp/qgw-v2-quick-validation.json`
- Create: `/private/tmp/qgw-v2-standard-validation.json`
- Create: `/private/tmp/qgw-v2-reference-validation.json`
- Create: `/private/tmp/qgw-v2-final-audit.json`
- Create: `/private/tmp/qgw-v2-final-validation.json`
- Create: `/private/tmp/qgw-v2-disposable-import.json`
- Create: `/private/tmp/qgw-v2-default-import.json`
- Create: `/private/tmp/qgw-v2-service-smoke.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v2-baseline/`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v2-inventory/`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v2-quick/`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v2-standard/`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v2-reference/`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v2-final/`

---

### Task 1: Prepare workspace, validate v1, and checkpoint baseline

**Files:**
- Read: `docs/superpowers/specs/2026-07-26-queens-gambit-white-v2-carlsbad-design.md`
- Read: `docs/operations/opening-course-authoring.md`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json`
- Create: `/private/tmp/qgw-v2-source-scope.json`
- Create: `/private/tmp/qgw-v2-current-validation.json`
- Create: `/private/tmp/qgw-v2-current-pack-audit.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v2-baseline/`

**Interfaces:**
- Consumes: approved v2 design and existing validated v1 Queen's Gambit White pack.
- Produces: private baseline checkpoint and source/current-pack summaries used by every later task.

- [ ] **Step 1: Verify execution branch and public status**

Run:

```bash
git status --short --branch
git branch --show-current
git log --oneline --decorate -8
```

Expected:

- The working branch starts with `codex/`.
- The public worktree has no unrelated modifications.
- If this plan is being run from `main`, create an isolated worktree before private authoring.

- [ ] **Step 2: Validate the existing v1 course**

Run:

```bash
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
  > /private/tmp/qgw-v2-current-validation.json
jq --slurpfile pack "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
  '{courseId,contentVersion,title:$pack[0].title,perspective:$pack[0].perspective,defaultDepth:$pack[0].defaultDepth,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length),warnings:.counts.warnings}' \
  /private/tmp/qgw-v2-current-validation.json
```

Expected:

- `courseId` is `mco15-queens-gambit-white`.
- `contentVersion` is `1.0.0`.
- `perspective` is `white`.
- `defaultDepth` is `reference`.
- `warnings` is `0`.
- `missing` is `0`.
- `unexpected` is `0`.

- [ ] **Step 3: Record the existing source scope**

Run:

```bash
jq '{
  courseId,
  contentVersion,
  title,
  perspective,
  defaultDepth,
  source,
  sourceCoverage,
  counts: {
    positions: (.positions|length),
    moves: (.moves|length),
    notes: (.notes|length),
    chapters: (.chapters|length),
    lessons: (.lessons|length),
    prompts: (.prompts|length),
    lessonEdges: (.lessonEdges|length)
  },
  chapters: [.chapters[] | {chapterId,title,minimumDepth}],
  lessons: [.lessons[] | {lessonId,title,minimumDepth,activityCount:(.activities|length)}]
}' \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
  > /private/tmp/qgw-v2-current-pack-audit.json
jq '{courseId,contentVersion,counts,printedPages:(.sourceCoverage.printedPages|length),expectedReferences:(.sourceCoverage.expectedReferences|length)}' \
  /private/tmp/qgw-v2-current-pack-audit.json
```

Expected:

- The current pack has 38 lessons.
- `printedPages` is `38`.
- `expectedReferences` is `42`.
- The v1 scope includes QGD, QGA, Slav/Semi-Slav, Tarrasch, and Chigorin coverage IDs.

- [ ] **Step 4: Write the v2 source-scope control file**

Run:

```bash
node --input-type=module - <<'JS' > /private/tmp/qgw-v2-source-scope.json
import fs from 'node:fs'

const current = JSON.parse(fs.readFileSync('/private/tmp/qgw-v2-current-pack-audit.json', 'utf8'))
const requiredFamilies = [
  { familyId: 'carlsbad-foundations', title: 'Carlsbad foundations', minimumDepth: 'quick' },
  { familyId: 'carlsbad-strategy', title: 'Carlsbad strategy', minimumDepth: 'quick' },
  { familyId: 'qgd-recognition', title: 'QGD recognition', minimumDepth: 'standard' },
  { familyId: 'qga-recognition', title: 'QGA recognition', minimumDepth: 'standard' },
  { familyId: 'slav-semi-slav-recognition', title: 'Slav and Semi-Slav recognition', minimumDepth: 'standard' },
  { familyId: 'qg-reference-map', title: 'Queen’s Gambit reference map', minimumDepth: 'reference' }
]
const requiredReferenceBranches = [
  'exchange-qgd-carlsbad',
  'minority-attack',
  'central-e4-break',
  'qgd-orthodox',
  'qgd-tartakower',
  'qgd-lasker',
  'qgd-cambridge-springs',
  'qgd-ragozin',
  'qga',
  'slav',
  'semi-slav',
  'tarrasch',
  'chigorin'
]
const summary = {
  courseId: 'mco15-queens-gambit-white',
  fromContentVersion: current.contentVersion,
  targetContentVersion: '2.0.0',
  title: "Queen's Gambit for White",
  perspective: 'white',
  defaultDepth: 'reference',
  privateAuthoringPath: '/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json',
  privatePackPath: '/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse',
  sourceCoverage: current.sourceCoverage,
  requiredFamilies,
  requiredStandardCoverage: ['qgd', 'qga', 'slav', 'semi-slav'],
  requiredReferenceBranches,
  sourceScopePolicy: 'reuse-v1-source-scope',
  privacyPolicy: 'private-course-content-outside-git'
}
console.log(JSON.stringify(summary, null, 2))
JS
jq '{courseId,fromContentVersion,targetContentVersion,perspective,defaultDepth,requiredStandardCoverage,requiredReferenceBranches}' \
  /private/tmp/qgw-v2-source-scope.json
```

Expected:

- `targetContentVersion` is `2.0.0`.
- `requiredStandardCoverage` contains `qgd`, `qga`, `slav`, and `semi-slav`.
- `requiredReferenceBranches` contains `tarrasch` and `chigorin`.

- [ ] **Step 5: Create the v2 baseline checkpoint**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v2-baseline"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" "$CHECKPOINT/mco15-queens-gambit-white-v1.ctcourse"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json" "$CHECKPOINT/mco15-queens-gambit-white-v1.authoring.json"
cp -p /private/tmp/qgw-v2-current-validation.json "$CHECKPOINT/"
cp -p /private/tmp/qgw-v2-current-pack-audit.json "$CHECKPOINT/"
cp -p /private/tmp/qgw-v2-source-scope.json "$CHECKPOINT/"
(
  cd "$CHECKPOINT"
  find . -maxdepth 1 -type f ! -name SHA256SUMS -print0 | sort -z | xargs -0 shasum -a 256 > SHA256SUMS
  shasum -a 256 -c SHA256SUMS
)
```

Expected: every checkpoint file prints `OK`.

---

### Task 2: Build v2 authoring inventory and anti-spam audit

**Files:**
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json`
- Read: `/private/tmp/qgw-v2-source-scope.json`
- Create: `/private/tmp/qgw-v2-inventory-check.mjs`
- Create: `/private/tmp/qgw-v2-inventory-check.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v2-inventory/`

**Interfaces:**
- Consumes: v1 authoring file and v2 source-scope control file from Task 1.
- Produces: private v2 authoring inventory with stable IDs, family tags, teaching nodes, and coverage-audit output.

- [ ] **Step 1: Preserve v1 authoring before editing**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v2-inventory"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json" "$CHECKPOINT/before-inventory.authoring.json"
```

Expected: `before-inventory.authoring.json` exists in the checkpoint.

- [ ] **Step 2: Add the v2 teaching-node manifest to the private authoring file**

Update `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json` so it contains these v2 teaching nodes, with IDs and depth values exactly as shown:

```json
[
  {"nodeId":"carlsbad-recognize-structure","title":"Recognize the Carlsbad structure","chapterId":"carlsbad-foundations","minimumDepth":"quick"},
  {"nodeId":"carlsbad-choose-plan","title":"Choose minority attack or e4","chapterId":"carlsbad-foundations","minimumDepth":"quick"},
  {"nodeId":"carlsbad-develop-for-plan","title":"Develop for the plan","chapterId":"carlsbad-foundations","minimumDepth":"quick"},
  {"nodeId":"carlsbad-meet-counterplay","title":"Meet Black counterplay","chapterId":"carlsbad-strategy","minimumDepth":"quick"},
  {"nodeId":"carlsbad-middlegame-map","title":"Opening into middlegame map","chapterId":"carlsbad-strategy","minimumDepth":"quick"},
  {"nodeId":"carlsbad-minority-attack","title":"Minority attack mechanics","chapterId":"carlsbad-strategy","minimumDepth":"standard"},
  {"nodeId":"carlsbad-central-e4","title":"Central e4 break mechanics","chapterId":"carlsbad-strategy","minimumDepth":"standard"},
  {"nodeId":"qgd-family-recognition","title":"Recognize the QGD family","chapterId":"branch-recognition","minimumDepth":"standard"},
  {"nodeId":"qga-recognition","title":"Recognize QGA and choose White setup","chapterId":"branch-recognition","minimumDepth":"standard"},
  {"nodeId":"slav-semislav-recognition","title":"Recognize Slav and Semi-Slav setups","chapterId":"branch-recognition","minimumDepth":"standard"},
  {"nodeId":"qgd-reference-branches","title":"QGD reference branches","chapterId":"reference-map","minimumDepth":"reference"},
  {"nodeId":"qga-reference-branches","title":"QGA reference branches","chapterId":"reference-map","minimumDepth":"reference"},
  {"nodeId":"slav-semislav-reference-branches","title":"Slav and Semi-Slav reference branches","chapterId":"reference-map","minimumDepth":"reference"},
  {"nodeId":"tarrasch-chigorin-reference","title":"Tarrasch and Chigorin reference branches","chapterId":"reference-map","minimumDepth":"reference"}
]
```

Implementation note: if the authoring file already has a `teachingNodes` array, replace it with this v2 manifest while preserving existing source inventory records that remain valid.

- [ ] **Step 3: Add the v2 chapter map to the private authoring file**

Update the private authoring file so its v2 chapter map contains exactly these chapter IDs:

```json
[
  {"chapterId":"carlsbad-foundations","title":"Carlsbad Foundations","minimumDepth":"quick"},
  {"chapterId":"carlsbad-strategy","title":"Carlsbad Strategy","minimumDepth":"quick"},
  {"chapterId":"branch-recognition","title":"Queen's Gambit Branch Recognition","minimumDepth":"standard"},
  {"chapterId":"reference-map","title":"Queen's Gambit Reference Map","minimumDepth":"reference"}
]
```

Expected: Quick and Standard are centered on Carlsbad and branch recognition; Reference owns the wider Queen's Gambit map.

- [ ] **Step 4: Tag the retained source inventory records**

Update retained or rewritten inventory records in the private authoring file so the inventory can prove Standard and Reference coverage without reading source prose.

Required Standard tags:

```json
["qgd", "qga", "slav", "semi-slav"]
```

Each required Standard tag must appear in at least one retained inventory record under either `standardCoverage` or `tags`.

Required Reference branch tags:

```json
[
  "exchange-qgd-carlsbad",
  "minority-attack",
  "central-e4-break",
  "qgd-orthodox",
  "qgd-tartakower",
  "qgd-lasker",
  "qgd-cambridge-springs",
  "qgd-ragozin",
  "qga",
  "slav",
  "semi-slav",
  "tarrasch",
  "chigorin"
]
```

Each required Reference tag must appear in at least one retained inventory record under either `referenceBranches` or `tags`.

- [ ] **Step 5: Create the inventory checker**

Create `/private/tmp/qgw-v2-inventory-check.mjs` with this exact script:

```javascript
import fs from 'node:fs'

const authoringPath = '/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json'
const scopePath = '/private/tmp/qgw-v2-source-scope.json'
const authoring = JSON.parse(fs.readFileSync(authoringPath, 'utf8'))
const scope = JSON.parse(fs.readFileSync(scopePath, 'utf8'))
const failures = []

const teachingNodes = new Set((authoring.teachingNodes || []).map((node) => node.nodeId || node.lessonId))
const requiredNodeIds = [
  'carlsbad-recognize-structure',
  'carlsbad-choose-plan',
  'carlsbad-develop-for-plan',
  'carlsbad-meet-counterplay',
  'carlsbad-middlegame-map',
  'carlsbad-minority-attack',
  'carlsbad-central-e4',
  'qgd-family-recognition',
  'qga-recognition',
  'slav-semislav-recognition',
  'qgd-reference-branches',
  'qga-reference-branches',
  'slav-semislav-reference-branches',
  'tarrasch-chigorin-reference'
]
for (const nodeId of requiredNodeIds) {
  if (!teachingNodes.has(nodeId)) failures.push(`missing teaching node ${nodeId}`)
}

const chapters = new Set((authoring.chapters || authoring.chapterMap || []).map((chapter) => chapter.chapterId))
for (const chapterId of ['carlsbad-foundations', 'carlsbad-strategy', 'branch-recognition', 'reference-map']) {
  if (!chapters.has(chapterId)) failures.push(`missing chapter ${chapterId}`)
}

const expected = new Set(scope.sourceCoverage.expectedReferences || [])
const inventory = Array.isArray(authoring.inventory)
  ? authoring.inventory
  : Array.isArray(authoring.inventory?.coverageRecords)
    ? authoring.inventory.coverageRecords
    : authoring.sourceInventory || []
const captured = new Set()
for (const record of inventory) {
  const coverageId = record.coverageId || record.sourceRef?.coverageId
  if (!coverageId) {
    failures.push(`inventory record missing coverageId: ${JSON.stringify(record).slice(0, 160)}`)
    continue
  }
  captured.add(coverageId)
  if (!expected.has(coverageId)) failures.push(`unexpected inventory coverage ${coverageId}`)
  const text = `${record.summary || ''} ${record.teachingText || ''} ${record.noteText || ''}`
  if (/Source note [a-z] records|Course move found|opening activity result|concrete analytical subvariation/i.test(text)) {
    failures.push(`spam/debug placeholder in inventory ${coverageId}`)
  }
}
for (const coverageId of expected) {
  if (!captured.has(coverageId)) failures.push(`missing inventory coverage ${coverageId}`)
}

const standardTags = new Set(inventory.flatMap((record) => record.standardCoverage || record.tags || []))
for (const tag of ['qgd', 'qga', 'slav', 'semi-slav']) {
  if (!standardTags.has(tag)) failures.push(`missing standard coverage tag ${tag}`)
}
const referenceTags = new Set(inventory.flatMap((record) => record.referenceBranches || record.tags || []))
for (const tag of ['tarrasch', 'chigorin']) {
  if (!referenceTags.has(tag)) failures.push(`missing reference branch tag ${tag}`)
}

const summary = {
  ok: failures.length === 0,
  failures,
  inventoryRecords: inventory.length,
  expectedReferences: expected.size,
  capturedReferences: captured.size,
  teachingNodes: teachingNodes.size,
  chapters: chapters.size
}
console.log(JSON.stringify(summary, null, 2))
if (failures.length) process.exit(1)
```

- [ ] **Step 6: Run the inventory checker**

Run:

```bash
node /private/tmp/qgw-v2-inventory-check.mjs > /private/tmp/qgw-v2-inventory-check.json
jq '{ok,inventoryRecords,expectedReferences,capturedReferences,teachingNodes,chapters,failures}' \
  /private/tmp/qgw-v2-inventory-check.json
```

Expected:

- `ok` is `true`.
- `failures` is `[]`.
- `teachingNodes` is `14`.
- `chapters` is at least `4`.
- `capturedReferences` equals `expectedReferences`.

- [ ] **Step 7: Checkpoint the v2 inventory**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v2-inventory"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json" "$CHECKPOINT/mco15-queens-gambit-white-v2-inventory.authoring.json"
cp -p /private/tmp/qgw-v2-source-scope.json "$CHECKPOINT/"
cp -p /private/tmp/qgw-v2-inventory-check.json "$CHECKPOINT/"
(
  cd "$CHECKPOINT"
  find . -maxdepth 1 -type f ! -name SHA256SUMS -print0 | sort -z | xargs -0 shasum -a 256 > SHA256SUMS
  shasum -a 256 -c SHA256SUMS
)
```

Expected: every checkpoint file prints `OK`.

---

### Task 3: Build and validate the Quick Carlsbad spine

**Files:**
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json`
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse`
- Create: `/private/tmp/qgw-v2-quick-validation.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v2-quick/`

**Interfaces:**
- Consumes: v2 inventory from Task 2.
- Produces: v2 course pack with a coherent Quick Carlsbad path and version `2.0.0`.

- [ ] **Step 1: Replace the Quick lesson layer with the Carlsbad spine**

Update the private authoring file so Quick visible lessons are exactly these lesson IDs, in this order:

```text
carlsbad-recognize-structure
carlsbad-choose-plan
carlsbad-develop-for-plan
carlsbad-meet-counterplay
carlsbad-middlegame-map
```

Each Quick lesson must have 2-3 required activities. The required activities must be selected from `concept`, `comparison`, `decision`, `demonstration`, and `recap`. The learner should make at most one required decision in each Quick lesson.

- [ ] **Step 2: Set the required Quick decision prompts**

Ensure the Quick layer includes these semantic decisions:

```text
carlsbad-recognize-structure: choose or identify the exchange-on-d5 structure.
carlsbad-choose-plan: choose minority attack or central e4 as a plan, not as memorized move order.
carlsbad-develop-for-plan: choose a White development move that supports the selected plan.
carlsbad-meet-counterplay: choose a White response to Black's ...Ne4, ...Bf5, or ...c5 counterplay.
carlsbad-middlegame-map: no required move prompt; use concept/comparison/recap.
```

Expected: no Quick lesson repeats the same prompt position and primary move inside the same lesson.

- [ ] **Step 3: Generate the Quick course pack**

Rewrite `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse` so its top-level fields, Quick lessons, prompts, move records, note records, and lesson edges match the v2 authoring decisions from Steps 1-2. If a private generator script exists beside the authoring file, run that script only after confirming it writes the same `.ctcourse` path; otherwise edit the `.ctcourse` JSON directly and use Step 4 as the acceptance gate.

Required pack fields after generation:

```json
{
  "schemaVersion": 2,
  "courseId": "mco15-queens-gambit-white",
  "contentVersion": "2.0.0",
  "title": "Queen's Gambit for White",
  "perspective": "white",
  "defaultDepth": "reference"
}
```

Expected: the pack contains only source-backed original summaries and does not include copied source prose in public files.

- [ ] **Step 4: Validate the Quick layer**

Run:

```bash
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
  > /private/tmp/qgw-v2-quick-validation.json
jq '{courseId,contentVersion,perspective,defaultDepth,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length),diagnostics}' \
  /private/tmp/qgw-v2-quick-validation.json
```

Expected:

- `courseId` is `mco15-queens-gambit-white`.
- `contentVersion` is `2.0.0`.
- `perspective` is `white`.
- `defaultDepth` is `reference`.
- `counts.warnings` is `0`.
- `counts.lessons` is at least `5`.
- `missing` is `0`.
- `unexpected` is `0`.
- The validator exits `0`; the `diagnostics` projection is `null` or `[]`.

- [ ] **Step 5: Audit Quick lesson shape**

Run:

```bash
jq '{
  quickLessons: [.lessons[] | select(.minimumDepth=="quick") | {lessonId,title,activities:(.activities|length),required:([.activities[] | select(.required==true)]|length),decisionCount:([.activities[] | select(.required==true and .kind=="decision")]|length)}],
  quickLessonCount: ([.lessons[] | select(.minimumDepth=="quick")] | length)
}' "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse"
```

Expected:

- `quickLessonCount` is `5`.
- Every Quick lesson has `required` between `2` and `3`.
- No Quick lesson has `decisionCount` greater than `1`.

- [ ] **Step 6: Checkpoint Quick**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v2-quick"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json" "$CHECKPOINT/"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" "$CHECKPOINT/"
cp -p /private/tmp/qgw-v2-quick-validation.json "$CHECKPOINT/"
(
  cd "$CHECKPOINT"
  find . -maxdepth 1 -type f ! -name SHA256SUMS -print0 | sort -z | xargs -0 shasum -a 256 > SHA256SUMS
  shasum -a 256 -c SHA256SUMS
)
```

Expected: every checkpoint file prints `OK`.

---

### Task 4: Build and validate Standard QGD/QGA/Slav-Semi-Slav coverage

**Files:**
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json`
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse`
- Create: `/private/tmp/qgw-v2-standard-validation.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v2-standard/`

**Interfaces:**
- Consumes: Quick course pack from Task 3.
- Produces: Standard layer with required QGD, QGA, Slav, and Semi-Slav branch recognition.

- [ ] **Step 1: Add Standard Carlsbad strategy lessons**

Add these Standard lesson IDs if they are not already present:

```text
carlsbad-minority-attack
carlsbad-central-e4
```

Required activities:

- `carlsbad-minority-attack` teaches `Rb1`/`b4-b5` style queenside pressure as a plan.
- `carlsbad-central-e4` teaches preparation and timing for the central `e4` break.
- Each has 3-4 required activities.
- Each has at most one required learner decision from a single position.

- [ ] **Step 2: Add Standard branch-recognition lessons**

Add these Standard lesson IDs if they are not already present:

```text
qgd-family-recognition
qga-recognition
slav-semislav-recognition
```

Required activities:

- `qgd-family-recognition` compares Orthodox/Tartakower/Lasker-style QGD setups where source scope supports clean coverage.
- `qga-recognition` teaches White's practical setup after Black accepts the c-pawn.
- `slav-semislav-recognition` teaches the difference between `...c6` Slav support and `...c6` plus `...e6` Semi-Slav tension.
- Each has 3-4 required activities.
- Each has at most one required learner decision from a single position.

- [ ] **Step 3: Wire Standard lessons into the teaching tree**

Update `lessonEdges` so the visible Standard route is:

```text
carlsbad-middlegame-map
├─ carlsbad-minority-attack
├─ carlsbad-central-e4
└─ qgd-family-recognition
   ├─ qga-recognition
   └─ slav-semislav-recognition
```

Use edge kinds:

- `continuation` for the main Carlsbad plan sequence.
- `alternative` with non-empty labels for QGA and Slav/Semi-Slav recognition branches.

Expected: every Standard lesson is reachable from the Quick root at Standard depth.

- [ ] **Step 4: Generate the Standard course pack**

Update `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse` from the private authoring file.

Expected:

- The same course ID and version remain `mco15-queens-gambit-white` / `2.0.0`.
- Quick lessons from Task 3 remain visible and unchanged in meaning.
- Standard lessons include QGD, QGA, Slav, and Semi-Slav recognition.

- [ ] **Step 5: Validate the Standard layer**

Run:

```bash
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
  > /private/tmp/qgw-v2-standard-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length),diagnostics}' \
  /private/tmp/qgw-v2-standard-validation.json
```

Expected:

- `contentVersion` is `2.0.0`.
- `counts.warnings` is `0`.
- `counts.lessons` is at least `10`.
- `counts.prompts` is at least `8`.
- `missing` is `0`.
- `unexpected` is `0`.
- The validator exits `0`; the `diagnostics` projection is `null` or `[]`.

- [ ] **Step 6: Audit Standard coverage**

Run:

```bash
jq '{
  standardLessons: [.lessons[] | select(.minimumDepth=="standard") | {lessonId,title,kinds:[.activities[].kind],required:([.activities[] | select(.required==true)]|length),decisionCount:([.activities[] | select(.required==true and .kind=="decision")]|length)}],
  hasQGD: any(.lessons[]; .lessonId=="qgd-family-recognition"),
  hasQGA: any(.lessons[]; .lessonId=="qga-recognition"),
  hasSlavSemiSlav: any(.lessons[]; .lessonId=="slav-semislav-recognition")
}' "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse"
```

Expected:

- `hasQGD`, `hasQGA`, and `hasSlavSemiSlav` are `true`.
- Every Standard lesson has 3-4 required activities unless the lesson is intentionally a short comparison with 2 required activities.
- No Standard lesson has more than one required decision from the same position.

- [ ] **Step 7: Checkpoint Standard**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v2-standard"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json" "$CHECKPOINT/"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" "$CHECKPOINT/"
cp -p /private/tmp/qgw-v2-standard-validation.json "$CHECKPOINT/"
(
  cd "$CHECKPOINT"
  find . -maxdepth 1 -type f ! -name SHA256SUMS -print0 | sort -z | xargs -0 shasum -a 256 > SHA256SUMS
  shasum -a 256 -c SHA256SUMS
)
```

Expected: every checkpoint file prints `OK`.

---

### Task 5: Build and validate Reference map

**Files:**
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json`
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse`
- Create: `/private/tmp/qgw-v2-reference-validation.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v2-reference/`

**Interfaces:**
- Consumes: Standard course pack from Task 4.
- Produces: full Reference-depth Queen's Gambit map with optional Tarrasch and Chigorin branches.

- [ ] **Step 1: Add Reference branch lessons**

Ensure these Reference lesson IDs are present:

```text
qgd-reference-branches
qga-reference-branches
slav-semislav-reference-branches
tarrasch-chigorin-reference
```

Each Reference lesson must:

- have at least one required `concept` activity so the lesson can open cleanly;
- keep dense branch material in optional `reference` activities;
- avoid required repeated decisions from the same position;
- expose source-backed branches in the variation explorer.

- [ ] **Step 2: Ensure Reference branches cover the required map**

The Reference layer must include coverage for these branch tags in private inventory or course notes:

```text
exchange-qgd-carlsbad
minority-attack
central-e4-break
qgd-orthodox
qgd-tartakower
qgd-lasker
qgd-cambridge-springs
qgd-ragozin
qga
slav
semi-slav
tarrasch
chigorin
```

Expected: Tarrasch and Chigorin are optional Reference branches, not required Standard lessons.

- [ ] **Step 3: Wire Reference lessons into the teaching tree**

Update `lessonEdges` so Reference lessons are reachable from the relevant Standard parent:

```text
qgd-family-recognition -> qgd-reference-branches
qga-recognition -> qga-reference-branches
slav-semislav-recognition -> slav-semislav-reference-branches
slav-semislav-reference-branches -> tarrasch-chigorin-reference
```

Use `reference` edge kind with non-empty labels.

Expected: every Reference lesson is reachable at Reference depth.

- [ ] **Step 4: Generate the Reference course pack**

Update `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse`.

Expected:

- Quick and Standard lesson semantics remain intact.
- Reference adds optional depth, not required repetition.
- The pack remains schema version 2.

- [ ] **Step 5: Validate the Reference pack**

Run:

```bash
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
  > /private/tmp/qgw-v2-reference-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length),diagnostics}' \
  /private/tmp/qgw-v2-reference-validation.json
```

Expected:

- `contentVersion` is `2.0.0`.
- `counts.warnings` is `0`.
- `counts.lessons` is at least `14`.
- `counts.lessonEdges` is at least `13`.
- `missing` is `0`.
- `unexpected` is `0`.
- The validator exits `0`; the `diagnostics` projection is `null` or `[]`.

- [ ] **Step 6: Audit Reference branch tags**

Run:

```bash
node --input-type=module - <<'JS'
import fs from 'node:fs'
const authoring = JSON.parse(fs.readFileSync('/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json', 'utf8'))
const pack = JSON.parse(fs.readFileSync('/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse', 'utf8'))
const required = ['exchange-qgd-carlsbad','minority-attack','central-e4-break','qgd-orthodox','qgd-tartakower','qgd-lasker','qgd-cambridge-springs','qgd-ragozin','qga','slav','semi-slav','tarrasch','chigorin']
const text = JSON.stringify({authoringInventory: authoring.inventory || authoring.sourceInventory || [], notes: pack.notes, moves: pack.moves, lessons: pack.lessons})
const missing = required.filter((tag) => !text.includes(tag))
console.log(JSON.stringify({ok: missing.length === 0, missing}, null, 2))
if (missing.length) process.exit(1)
JS
```

Expected: `ok` is `true` and `missing` is `[]`.

- [ ] **Step 7: Checkpoint Reference**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v2-reference"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json" "$CHECKPOINT/"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" "$CHECKPOINT/"
cp -p /private/tmp/qgw-v2-reference-validation.json "$CHECKPOINT/"
(
  cd "$CHECKPOINT"
  find . -maxdepth 1 -type f ! -name SHA256SUMS -print0 | sort -z | xargs -0 shasum -a 256 > SHA256SUMS
  shasum -a 256 -c SHA256SUMS
)
```

Expected: every checkpoint file prints `OK`.

---

### Task 6: Run final private audit and import smokes

**Files:**
- Create: `/private/tmp/qgw-v2-course-audit.mjs`
- Create: `/private/tmp/qgw-v2-final-audit.json`
- Create: `/private/tmp/qgw-v2-final-validation.json`
- Create: `.superpowers/tmp/queens-gambit-white-v2-import-smoke.go`
- Create: `.superpowers/tmp/queens-gambit-white-v2-service-smoke.go`
- Create: `/private/tmp/qgw-v2-disposable-import.json`
- Create: `/private/tmp/qgw-v2-default-import.json`
- Create: `/private/tmp/qgw-v2-service-smoke.json`

**Interfaces:**
- Consumes: Reference course pack from Task 5.
- Produces: final private proof that v2 imports safely, preserves other active courses, and exposes correct White orientation.

- [ ] **Step 1: Create final course audit script**

Create `/private/tmp/qgw-v2-course-audit.mjs` with this exact script:

```javascript
import fs from 'node:fs'

const packPath = '/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse'
const pack = JSON.parse(fs.readFileSync(packPath, 'utf8'))
const failures = []

const byId = (items, key) => new Map(items.map((item) => [item[key], item]))
const lessons = byId(pack.lessons || [], 'lessonId')
const prompts = byId(pack.prompts || [], 'promptId')
const moves = byId(pack.moves || [], 'moveId')
const notes = pack.notes || []

if (pack.courseId !== 'mco15-queens-gambit-white') failures.push(`courseId ${pack.courseId}`)
if (pack.contentVersion !== '2.0.0') failures.push(`contentVersion ${pack.contentVersion}`)
if (pack.perspective !== 'white') failures.push(`perspective ${pack.perspective}`)
if (pack.defaultDepth !== 'reference') failures.push(`defaultDepth ${pack.defaultDepth}`)

for (const id of ['carlsbad-recognize-structure','carlsbad-choose-plan','carlsbad-develop-for-plan','carlsbad-meet-counterplay','carlsbad-middlegame-map']) {
  const lesson = lessons.get(id)
  if (!lesson) failures.push(`missing quick lesson ${id}`)
  else if (lesson.minimumDepth !== 'quick') failures.push(`${id} depth ${lesson.minimumDepth}`)
}
for (const id of ['carlsbad-minority-attack','carlsbad-central-e4','qgd-family-recognition','qga-recognition','slav-semislav-recognition']) {
  const lesson = lessons.get(id)
  if (!lesson) failures.push(`missing standard lesson ${id}`)
  else if (lesson.minimumDepth !== 'standard') failures.push(`${id} depth ${lesson.minimumDepth}`)
}
for (const id of ['qgd-reference-branches','qga-reference-branches','slav-semislav-reference-branches','tarrasch-chigorin-reference']) {
  const lesson = lessons.get(id)
  if (!lesson) failures.push(`missing reference lesson ${id}`)
  else if (lesson.minimumDepth !== 'reference') failures.push(`${id} depth ${lesson.minimumDepth}`)
}

for (const lesson of pack.lessons || []) {
  const seenRequiredDecisions = new Set()
  let requiredCount = 0
  let requiredDecisionCount = 0
  for (const activity of lesson.activities || []) {
    if (activity.required) requiredCount += 1
    if (activity.required && activity.kind === 'decision') {
      requiredDecisionCount += 1
      const prompt = prompts.get(activity.promptId)
      if (!prompt) failures.push(`${lesson.lessonId}/${activity.activityId} missing prompt ${activity.promptId}`)
      else {
        const move = moves.get(prompt.primaryMoveId)
        const key = `${prompt.positionId}:${prompt.primaryMoveId}`
        if (seenRequiredDecisions.has(key)) failures.push(`${lesson.lessonId} repeats required decision ${key}`)
        seenRequiredDecisions.add(key)
        if (move?.trainingRole !== 'repertoire' && move?.trainingRole !== 'alternative') {
          failures.push(`${lesson.lessonId}/${activity.activityId} primary move is not White repertoire/alternative`)
        }
      }
    }
    if (activity.kind === 'reference' && activity.required) failures.push(`${lesson.lessonId}/${activity.activityId} required reference activity`)
  }
  if (lesson.minimumDepth === 'quick' && (requiredCount < 2 || requiredCount > 3)) failures.push(`${lesson.lessonId} quick required count ${requiredCount}`)
  if (lesson.minimumDepth === 'quick' && requiredDecisionCount > 1) failures.push(`${lesson.lessonId} quick decision count ${requiredDecisionCount}`)
  if (lesson.minimumDepth === 'standard' && requiredDecisionCount > 1) failures.push(`${lesson.lessonId} standard decision count ${requiredDecisionCount}`)
}

const serialized = JSON.stringify(pack)
for (const marker of ['Source note ', 'Course move found', 'opening activity result', 'concrete analytical subvariation']) {
  if (serialized.includes(marker)) failures.push(`spam/debug marker ${marker}`)
}
for (const tag of ['qgd','qga','slav','semi-slav','tarrasch','chigorin']) {
  if (!serialized.includes(tag)) failures.push(`missing branch/tag text ${tag}`)
}

const summary = {
  ok: failures.length === 0,
  failures,
  counts: {
    chapters: pack.chapters?.length ?? 0,
    lessons: pack.lessons?.length ?? 0,
    moves: pack.moves?.length ?? 0,
    notes: notes.length,
    prompts: pack.prompts?.length ?? 0,
    lessonEdges: pack.lessonEdges?.length ?? 0
  }
}
console.log(JSON.stringify(summary, null, 2))
if (failures.length) process.exit(1)
```

- [ ] **Step 2: Run final course audit and validator**

Run:

```bash
node /private/tmp/qgw-v2-course-audit.mjs > /private/tmp/qgw-v2-final-audit.json
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
  > /private/tmp/qgw-v2-final-validation.json
jq -s '.[0] as $validation | .[1] as $audit | {auditOk:$audit.ok, auditFailures:$audit.failures, validation:{courseId:$validation.courseId,contentVersion:$validation.contentVersion,counts:$validation.counts,missing:($validation.coverage.missing|length),unexpected:($validation.coverage.unexpected|length),diagnostics:$validation.diagnostics}}' \
  /private/tmp/qgw-v2-final-validation.json /private/tmp/qgw-v2-final-audit.json
```

Expected:

- `auditOk` is `true`.
- `auditFailures` is `[]`.
- validation `courseId` is `mco15-queens-gambit-white`.
- validation `contentVersion` is `2.0.0`.
- `counts.warnings` is `0`.
- `missing` is `0`.
- `unexpected` is `0`.
- The validator exits `0`; the `diagnostics` projection is `null` or `[]`.

- [ ] **Step 3: Create the app-layer import smoke harness**

Create `.superpowers/tmp/queens-gambit-white-v2-import-smoke.go` with this exact content:

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
	paths, err := selectedPaths()
	if err != nil {
		fail(err)
	}
	coursePaths := splitRequired("QGW_COURSES")
	if len(coursePaths) == 0 {
		fail(fmt.Errorf("QGW_COURSES is required"))
	}
	services, err := app.Open(paths)
	if err != nil {
		fail(err)
	}
	defer services.Close()
	if services.CourseImporter == nil || services.Openings == nil || services.OpeningCatalog == nil {
		fail(fmt.Errorf("opening services are unavailable"))
	}

	imports := make([]map[string]any, 0, len(coursePaths))
	for _, coursePath := range coursePaths {
		inspection, err := services.CourseImporter.Inspect(ctx, coursePath)
		if err != nil {
			fail(err)
		}
		report, err := services.CourseImporter.Import(ctx, inspection, nil)
		if err != nil {
			fail(err)
		}
		imports = append(imports, map[string]any{
			"path":       coursePath,
			"sourceId":   inspection.SourceID,
			"sourceName": inspection.SourceName,
			"report":     report,
		})
	}

	home, err := services.Openings.Home(ctx)
	if err != nil {
		fail(err)
	}
	activeCourses := []map[string]string{}
	activeVersions := map[string]string{}
	for _, course := range home.Courses {
		compiled, err := services.OpeningCatalog.LoadActive(ctx, course.CourseID)
		if err != nil {
			fail(err)
		}
		activeVersions[course.CourseID] = compiled.Pack.ContentVersion
		activeCourses = append(activeCourses, map[string]string{
			"courseId":       course.CourseID,
			"title":          course.Title,
			"perspective":    string(course.Perspective),
			"contentVersion": compiled.Pack.ContentVersion,
		})
	}

	missingRequired := []string{}
	for _, courseID := range splitOptional("QGW_REQUIRE_COURSES") {
		if _, ok := activeVersions[courseID]; !ok {
			missingRequired = append(missingRequired, courseID)
		}
	}
	imported, err := services.OpeningCatalog.LoadActive(ctx, "mco15-queens-gambit-white")
	if err != nil {
		fail(err)
	}
	ok := imported.Pack.CourseID == "mco15-queens-gambit-white" &&
		imported.Pack.ContentVersion == "2.0.0" &&
		imported.Pack.Perspective == "white" &&
		len(missingRequired) == 0
	write(map[string]any{
		"ok":                     ok,
		"dataRoot":               paths.Root,
		"imports":                imports,
		"activeCourses":          activeCourses,
		"activeVersions":         activeVersions,
		"missingRequiredCourses": missingRequired,
		"importedCourseId":       imported.Pack.CourseID,
		"importedContentVersion": imported.Pack.ContentVersion,
		"perspective":            imported.Pack.Perspective,
	})
	if !ok {
		os.Exit(1)
	}
}

func selectedPaths() (storage.Paths, error) {
	if os.Getenv("QGW_USE_DEFAULT_DATA") == "1" {
		return storage.DefaultPaths()
	}
	root := strings.TrimSpace(os.Getenv("QGW_DATA_ROOT"))
	if root == "" {
		return storage.Paths{}, fmt.Errorf("QGW_DATA_ROOT or QGW_USE_DEFAULT_DATA=1 is required")
	}
	return storage.PathsAt(root), nil
}

func splitRequired(name string) []string {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return []string{}
	}
	return strings.Split(raw, "|")
}

func splitOptional(name string) []string {
	parts := splitRequired(name)
	out := parts[:0]
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func write(value any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
```

- [ ] **Step 4: Run disposable import smoke**

Run:

```bash
QGW_DATA_ROOT="$(mktemp -d /private/tmp/qgw-v2-import.XXXXXX)" \
QGW_COURSES="/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
go run .superpowers/tmp/queens-gambit-white-v2-import-smoke.go \
  > /private/tmp/qgw-v2-disposable-import.json
jq . /private/tmp/qgw-v2-disposable-import.json
```

Expected:

- `ok` is `true`.
- `importedCourseId` is `mco15-queens-gambit-white`.
- `importedContentVersion` is `2.0.0`.
- `perspective` is `white`.

- [ ] **Step 5: Run default catalogue import smoke**

Run:

```bash
QGW_USE_DEFAULT_DATA=1 \
QGW_COURSES="/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
QGW_REQUIRE_COURSES="mco15-italian-white|mco15-ruy-lopez-white|mco15-queens-gambit-white|mco15-queens-gambit-black|mco15-caro-kann-black|mco15-sicilian-black|mco15-french-black" \
go run .superpowers/tmp/queens-gambit-white-v2-import-smoke.go \
  > /private/tmp/qgw-v2-default-import.json
jq '{ok,activeCourses}' /private/tmp/qgw-v2-default-import.json
```

Expected:

- `ok` is `true`.
- `mco15-queens-gambit-white` has content version `2.0.0`.
- Other active courses remain present.

- [ ] **Step 6: Create the app-service journey smoke harness**

Create `.superpowers/tmp/queens-gambit-white-v2-service-smoke.go` with this exact content:

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"chess-trainer/internal/app"
	"chess-trainer/internal/openings"
	"chess-trainer/internal/storage"
)

const courseID = "mco15-queens-gambit-white"

type lessonTrace struct {
	LessonID     string   `json:"lessonId"`
	Activities   []string `json:"activities"`
	Orientations []string `json:"orientations"`
}

func main() {
	ctx := context.Background()
	dataRoot := strings.TrimSpace(os.Getenv("QGW_SERVICE_DATA_ROOT"))
	if dataRoot == "" {
		fail(fmt.Errorf("QGW_SERVICE_DATA_ROOT is required"))
	}
	coursePath := strings.TrimSpace(os.Getenv("QGW_SERVICE_COURSE"))
	if coursePath == "" {
		fail(fmt.Errorf("QGW_SERVICE_COURSE is required"))
	}
	services, err := app.Open(storage.PathsAt(dataRoot))
	if err != nil {
		fail(err)
	}
	defer services.Close()
	if services.CourseImporter == nil || services.Openings == nil || services.OpeningCatalog == nil {
		fail(fmt.Errorf("opening services are unavailable"))
	}
	inspection, err := services.CourseImporter.Inspect(ctx, coursePath)
	if err != nil {
		fail(err)
	}
	if _, err := services.CourseImporter.Import(ctx, inspection, nil); err != nil {
		fail(err)
	}
	compiled, err := services.OpeningCatalog.LoadActive(ctx, courseID)
	if err != nil {
		fail(err)
	}

	home, err := services.Openings.Home(ctx)
	if err != nil {
		fail(err)
	}
	homeHasWhiteCourse := false
	for _, course := range home.Courses {
		if course.CourseID == courseID && course.Title == "Queen's Gambit for White" && course.Perspective == openings.PerspectiveWhite {
			homeHasWhiteCourse = true
		}
	}

	noteTexts := []string{}
	startedLessons := []string{}
	if err := services.Openings.SetDepth(ctx, courseID, openings.DepthQuick); err != nil {
		fail(err)
	}
	quickTrace, quickNotes, err := completeLesson(ctx, services, compiled, "carlsbad-recognize-structure")
	if err != nil {
		fail(err)
	}
	startedLessons = append(startedLessons, quickTrace.LessonID)
	noteTexts = append(noteTexts, quickNotes...)

	if err := services.Openings.SetDepth(ctx, courseID, openings.DepthStandard); err != nil {
		fail(err)
	}
	standardTraces := []lessonTrace{}
	for _, lessonID := range []string{"qgd-family-recognition", "qga-recognition", "slav-semislav-recognition"} {
		trace, notes, err := startOnly(ctx, services, lessonID)
		if err != nil {
			fail(err)
		}
		startedLessons = append(startedLessons, lessonID)
		standardTraces = append(standardTraces, trace)
		noteTexts = append(noteTexts, notes...)
	}

	if err := services.Openings.SetDepth(ctx, courseID, openings.DepthReference); err != nil {
		fail(err)
	}
	referenceTrace, referenceNotes, err := startOnly(ctx, services, "tarrasch-chigorin-reference")
	if err != nil {
		fail(err)
	}
	startedLessons = append(startedLessons, referenceTrace.LessonID)
	noteTexts = append(noteTexts, referenceNotes...)

	explorer, err := services.Openings.Explore(ctx, courseID, compiled.Pack.RootPositionID, openings.DepthReference)
	if err != nil {
		fail(err)
	}
	for _, note := range explorer.Notes {
		noteTexts = append(noteTexts, note.Text)
	}

	spamMarkers := spamMarkers(noteTexts)
	ok := homeHasWhiteCourse &&
		compiled.Pack.CourseID == courseID &&
		compiled.Pack.ContentVersion == "2.0.0" &&
		compiled.Pack.Perspective == openings.PerspectiveWhite &&
		len(spamMarkers) == 0 &&
		hasAll(startedLessons, []string{"carlsbad-recognize-structure", "qgd-family-recognition", "qga-recognition", "slav-semislav-recognition", "tarrasch-chigorin-reference"})
	write(map[string]any{
		"ok":                  ok,
		"courseId":            compiled.Pack.CourseID,
		"contentVersion":      compiled.Pack.ContentVersion,
		"orientation":         compiled.Pack.Perspective,
		"homeHasWhiteCourse":  homeHasWhiteCourse,
		"rootPositionId":      compiled.Pack.RootPositionID,
		"explorerMoveCount":   len(explorer.Moves),
		"startedLessons":      startedLessons,
		"quickTrace":          quickTrace,
		"standardTraces":      standardTraces,
		"referenceTrace":      referenceTrace,
		"spamMarkers":         spamMarkers,
		"scannedVisibleNotes": len(noteTexts),
	})
	if !ok {
		os.Exit(1)
	}
}

func completeLesson(ctx context.Context, services *app.Services, compiled openings.CompiledCourse, lessonID string) (lessonTrace, []string, error) {
	session, err := services.Openings.StartLesson(ctx, courseID, lessonID)
	if err != nil {
		return lessonTrace{}, nil, err
	}
	trace := lessonTrace{LessonID: lessonID}
	notes := []string{}
	for session.Current != nil {
		current := session.Current
		trace.Activities = append(trace.Activities, current.ActivityID)
		trace.Orientations = append(trace.Orientations, string(current.Orientation))
		if current.Orientation != openings.PerspectiveWhite {
			return trace, notes, fmt.Errorf("%s/%s orientation = %s", lessonID, current.ActivityID, current.Orientation)
		}
		notes = append(notes, activityNotes(current)...)
		var result openings.OpeningActivityResult
		if current.Kind == openings.ActivityDecision {
			uci, err := primaryUCI(compiled, session.LessonID, current.ActivityID)
			if err != nil {
				return trace, notes, err
			}
			result, err = services.Openings.PlayMove(ctx, session.SessionID, uci)
			if err != nil {
				return trace, notes, err
			}
		} else {
			result, err = services.Openings.AdvanceActivity(ctx, session.SessionID)
			if err != nil {
				return trace, notes, err
			}
		}
		session = result.Session
	}
	return trace, notes, nil
}

func startOnly(ctx context.Context, services *app.Services, lessonID string) (lessonTrace, []string, error) {
	session, err := services.Openings.StartLesson(ctx, courseID, lessonID)
	if err != nil {
		return lessonTrace{}, nil, err
	}
	trace := lessonTrace{LessonID: lessonID}
	notes := []string{}
	if session.Current != nil {
		trace.Activities = append(trace.Activities, session.Current.ActivityID)
		trace.Orientations = append(trace.Orientations, string(session.Current.Orientation))
		notes = append(notes, activityNotes(session.Current)...)
		if session.Current.Orientation != openings.PerspectiveWhite {
			return trace, notes, fmt.Errorf("%s orientation = %s", lessonID, session.Current.Orientation)
		}
	}
	return trace, notes, nil
}

func primaryUCI(compiled openings.CompiledCourse, lessonID string, activityID string) (string, error) {
	lesson, ok := compiled.Lessons[lessonID]
	if !ok {
		return "", fmt.Errorf("missing lesson %s", lessonID)
	}
	for _, activity := range lesson.Activities {
		if activity.ActivityID != activityID {
			continue
		}
		prompt, ok := compiled.Prompts[activity.PromptID]
		if !ok {
			return "", fmt.Errorf("%s/%s missing prompt %s", lessonID, activityID, activity.PromptID)
		}
		move, ok := compiled.Moves[prompt.PrimaryMoveID]
		if !ok {
			return "", fmt.Errorf("%s/%s missing primary move %s", lessonID, activityID, prompt.PrimaryMoveID)
		}
		if move.TrainingRole != openings.RoleRepertoire && move.TrainingRole != openings.RoleAlternative {
			return "", fmt.Errorf("%s/%s primary move role = %s", lessonID, activityID, move.TrainingRole)
		}
		return move.UCI, nil
	}
	return "", fmt.Errorf("missing activity %s/%s", lessonID, activityID)
}

func activityNotes(activity *openings.OpeningActivityView) []string {
	if activity == nil {
		return []string{}
	}
	notes := append([]string{}, activity.TeachingNoteTexts...)
	notes = append(notes, activity.ReferenceNoteTexts...)
	for _, section := range activity.ReferenceSections {
		notes = append(notes, section.NoteTexts...)
	}
	return notes
}

func spamMarkers(notes []string) []string {
	text := strings.Join(notes, "\n")
	markers := []string{}
	for _, marker := range []string{"Source note ", "Course move found", "opening activity result", "concrete analytical subvariation"} {
		if strings.Contains(text, marker) {
			markers = append(markers, marker)
		}
	}
	return markers
}

func hasAll(got []string, want []string) bool {
	seen := map[string]bool{}
	for _, id := range got {
		seen[id] = true
	}
	for _, id := range want {
		if !seen[id] {
			return false
		}
	}
	return true
}

func write(value any) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fail(err)
	}
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
```

Run:

```bash
QGW_SERVICE_DATA_ROOT="$(mktemp -d /private/tmp/qgw-v2-service.XXXXXX)" \
QGW_SERVICE_COURSE="/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" \
go run .superpowers/tmp/queens-gambit-white-v2-service-smoke.go \
  > /private/tmp/qgw-v2-service-smoke.json
jq . /private/tmp/qgw-v2-service-smoke.json
```

Expected:

- `ok` is `true`.
- `courseId` is `mco15-queens-gambit-white`.
- `contentVersion` is `2.0.0`.
- `orientation` is `white`.
- `startedLessons` includes the Quick, QGD, QGA, Slav/Semi-Slav, and Reference lesson IDs listed above.
- `spamMarkers` is `[]`.

---

### Task 7: Final verification, checkpoint, and public implementation status

**Files:**
- Read: `/private/tmp/qgw-v2-final-audit.json`
- Read: `/private/tmp/qgw-v2-final-validation.json`
- Read: `/private/tmp/qgw-v2-disposable-import.json`
- Read: `/private/tmp/qgw-v2-default-import.json`
- Read: `/private/tmp/qgw-v2-service-smoke.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v2-final/`
- Modify: `docs/superpowers/specs/2026-07-26-queens-gambit-white-v2-carlsbad-design.md`
- Modify: `docs/superpowers/plans/2026-07-26-queens-gambit-white-v2-carlsbad.md`

**Interfaces:**
- Consumes: final private pack and validation/import artifacts from Task 6.
- Produces: final private checkpoint and public docs-only implementation status.

- [ ] **Step 1: Run repository verification**

Run:

```bash
go test ./...
npm --prefix frontend test -- --run --single-thread
npm --prefix frontend run check
npm --prefix frontend run test:e2e -- frontend/tests/openings.spec.ts --project=chromium
npm --prefix frontend run test:e2e -- frontend/tests/openings.spec.ts --project=webkit
```

Expected:

- Go tests pass.
- Frontend unit tests pass.
- Svelte/type checks pass.
- Opening E2E passes in Chromium and WebKit.

- [ ] **Step 2: Create final private checkpoint**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/queens-gambit-white-v2-final"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json" "$CHECKPOINT/"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse" "$CHECKPOINT/"
cp -p /private/tmp/qgw-v2-source-scope.json "$CHECKPOINT/"
cp -p /private/tmp/qgw-v2-current-pack-audit.json "$CHECKPOINT/"
cp -p /private/tmp/qgw-v2-inventory-check.json "$CHECKPOINT/"
cp -p /private/tmp/qgw-v2-final-audit.json "$CHECKPOINT/"
cp -p /private/tmp/qgw-v2-final-validation.json "$CHECKPOINT/"
cp -p /private/tmp/qgw-v2-disposable-import.json "$CHECKPOINT/"
cp -p /private/tmp/qgw-v2-default-import.json "$CHECKPOINT/"
cp -p /private/tmp/qgw-v2-service-smoke.json "$CHECKPOINT/"
(
  cd "$CHECKPOINT"
  find . -maxdepth 1 -type f ! -name SHA256SUMS -print0 | sort -z | xargs -0 shasum -a 256 > SHA256SUMS
  shasum -a 256 -c SHA256SUMS
)
```

Expected: every checkpoint payload prints `OK`.

- [ ] **Step 3: Confirm no private artifacts are tracked**

Run:

```bash
git status --short --ignored
git ls-files | rg 'Private Chess Courses|qgw-v2|mco15-queens-gambit-white\\.(ctcourse|authoring\\.json)$|/mco15-queens-gambit-white\\.(ctcourse|authoring\\.json)$' || true
```

Expected:

- Private files may appear only as ignored/untracked paths outside Git.
- `git ls-files` prints no private Queen's Gambit course artifacts.

- [ ] **Step 4: Update public implementation status**

Append this status block to `docs/superpowers/plans/2026-07-26-queens-gambit-white-v2-carlsbad.md`:

```markdown
## Implementation status

- Private course pack: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse`
- Private authoring file: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json`
- Content version: `2.0.0`
- Final validation: passed
- Final private audit: passed
- Disposable import smoke: passed
- Default catalogue import: passed
- App-service journey smoke: passed
- Automated app verification: passed via real Queen's Gambit app-service smoke plus Chromium/WebKit Playwright opening E2E
- Manual live app verification: not performed unless a claimable in-app browser tab or controllable Wails desktop session was available during implementation
```

Update `docs/superpowers/specs/2026-07-26-queens-gambit-white-v2-carlsbad-design.md` status to:

```markdown
Status: implemented; automated verification passed; live manual Learn Openings acceptance outstanding
```

If live manual app acceptance was genuinely completed, replace the final two status lines with a precise pass statement and include the exact app/session used.

- [ ] **Step 5: Commit public docs**

Run:

```bash
git add \
  docs/superpowers/specs/2026-07-26-queens-gambit-white-v2-carlsbad-design.md \
  docs/superpowers/plans/2026-07-26-queens-gambit-white-v2-carlsbad.md
git commit -m "docs: mark queens gambit white v2 implemented"
git status --short --branch
```

Expected:

- The commit contains only public docs.
- No private course artifacts are staged or committed.

## Implementation status

- Private course pack: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse`
- Private authoring file: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json`
- Content version: `2.0.0`
- Final validation: passed
- Final private audit: passed
- Disposable import smoke: passed
- Default catalogue import: passed
- App-service journey smoke: passed
- Automated app verification: passed via real Queen's Gambit app-service smoke plus Chromium/WebKit Playwright opening E2E
- Manual live app verification: not performed; app-service journey smoke and Chromium/WebKit opening E2E passed
