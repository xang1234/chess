# Sicilian White Open Course Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Create the private `mco15-sicilian-white` course as a White-perspective, structure-first Open Sicilian repertoire with balanced Standard coverage and Reference-depth source coverage.

**Architecture:** Reuse the existing schema-v2 opening course pack, validator/importer, teaching-tree UI, journey persistence, and app-service smoke patterns. Create a fresh private White course and authoring inventory while using the existing Sicilian Black source page mapping only as a coordinate baseline.

**Tech Stack:** Go coursepack validator/importer, schema-v2 opening course JSON, private Node.js audit/generation scripts, `jq`, local MCO-15 PDF page mapping, Wails/Svelte app runtime, Playwright opening E2E.

## Global Constraints

- Course ID: `mco15-sicilian-white`
- Title: `Sicilian Defence for White`
- Content version: `1.0.0`
- Perspective: `white`
- Default depth: `reference`
- Private authoring file: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.authoring.json`
- Private course pack: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse`
- Source PDF: `/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf`
- Source scope printed pages: `244-361`
- Source scope PDF pages: `261-378`
- Source scan offset: `+17`
- Source coverage target: `118` page-anchor records.
- Repertoire style: Open Sicilian mainline, principled.
- Main teaching spine: structure-first Open Sicilian.
- Quick style: decision-point skeleton.
- Quick lesson count target: `5`.
- Required Standard coverage: Najdorf, Dragon, Classical Sicilian, Scheveningen, Sveshnikov, Kalashnikov, Kan, and Taimanov.
- White repertoire moves are learner decisions.
- Black moves are opponent pressure, branch choices, or demonstrations.
- Do not modify `mco15-sicilian-black`.
- Do not build a primarily anti-Sicilian repertoire in this pass.
- Do not add a generic multi-perspective course model.
- Do not special-case Sicilian White in public app code.
- Do not commit private `.ctcourse`, private authoring JSON, rendered PDF pages, private audit outputs, private checkpoints, or MCO-derived course prose to Git.
- Public reports and final responses may contain only IDs, paths, counts, validation results, and high-level summaries.

---

## File Structure

Repository files:

- Read: `docs/superpowers/specs/2026-07-26-sicilian-white-open-course-design.md`
- Read: `docs/operations/opening-course-authoring.md`
- Read for patterns: `docs/superpowers/plans/2026-07-24-sicilian-black-course.md`
- Read for smoke-harness patterns: `docs/superpowers/plans/2026-07-26-queens-gambit-white-v2-carlsbad.md`
- Create: `docs/superpowers/plans/2026-07-26-sicilian-white-open-course.md`
- Modify at the end only: `docs/superpowers/specs/2026-07-26-sicilian-white-open-course-design.md`
- Modify at the end only: `docs/superpowers/plans/2026-07-26-sicilian-white-open-course.md`
- Create and keep ignored: `.superpowers/tmp/sicilian-white-import-smoke.go`
- Create and keep ignored: `.superpowers/tmp/sicilian-white-service-smoke.go`

Private files outside Git:

- Create/update: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.authoring.json`
- Create/update: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse`
- Read only: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json`
- Read only: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse`
- Read only: `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse`
- Read only: `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse`
- Read only: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse`
- Read only: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse`
- Read only: `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse`
- Read only: `/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse`
- Create: `/private/tmp/sicilian-white-source-scope.json`
- Create: `/private/tmp/sicilian-white-library-baseline.json`
- Create: `/private/tmp/sicilian-white-inventory-check.mjs`
- Create: `/private/tmp/sicilian-white-inventory-check.json`
- Create: `/private/tmp/sicilian-white-quick-validation.json`
- Create: `/private/tmp/sicilian-white-standard-validation.json`
- Create: `/private/tmp/sicilian-white-reference-validation.json`
- Create: `/private/tmp/sicilian-white-course-audit.mjs`
- Create: `/private/tmp/sicilian-white-final-audit.json`
- Create: `/private/tmp/sicilian-white-final-validation.json`
- Create: `/private/tmp/sicilian-white-disposable-import.json`
- Create: `/private/tmp/sicilian-white-default-import.json`
- Create: `/private/tmp/sicilian-white-service-smoke.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-white-v1-baseline/`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-white-v1-inventory/`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-white-v1-quick/`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-white-v1-standard/`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-white-v1-reference/`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-white-v1-final/`

---

### Task 1: Prepare workspace, validate source baseline, and checkpoint library state

**Files:**
- Read: `docs/superpowers/specs/2026-07-26-sicilian-white-open-course-design.md`
- Read: `docs/operations/opening-course-authoring.md`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse`
- Create: `/private/tmp/sicilian-white-source-scope.json`
- Create: `/private/tmp/sicilian-white-library-baseline.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-white-v1-baseline/`

**Interfaces:**
- Consumes: approved Sicilian White design and existing Sicilian Black source-coordinate mapping.
- Produces: private source-scope control file and current-library baseline for all later tasks.

- [ ] **Step 1: Verify execution branch and clean public state**

Run:

```bash
git status --short --branch
git branch --show-current
git log --oneline --decorate -8
```

Expected:

- The execution branch starts with `codex/`.
- If the plan is being run from `main`, create an isolated worktree before private authoring.
- The public worktree has no unrelated modifications.

- [ ] **Step 2: Validate the existing Sicilian Black coordinate baseline**

Run:

```bash
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse" \
  > /private/tmp/sicilian-black-baseline-validation.json
jq --slurpfile pack "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse" \
  '{courseId,contentVersion,title:$pack[0].title,perspective:$pack[0].perspective,defaultDepth:$pack[0].defaultDepth,printedPages:($pack[0].sourceCoverage.printedPages|length),expectedReferences:($pack[0].sourceCoverage.expectedReferences|length),counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length),warnings:.counts.warnings}' \
  /private/tmp/sicilian-black-baseline-validation.json
```

Expected:

- `courseId` is `mco15-sicilian-black`.
- `perspective` is `black`.
- `printedPages` is `118`.
- `expectedReferences` is `118`.
- `warnings` is `0`.
- `missing` is `0`.
- `unexpected` is `0`.

- [ ] **Step 3: Record current opening-library state**

Run:

```bash
node --input-type=module - <<'JS' > /private/tmp/sicilian-white-library-baseline.json
import fs from 'node:fs'

const coursePaths = [
  '/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse',
  '/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse',
  '/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse',
  '/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse',
  '/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse',
  '/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse',
  '/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse'
]
const courses = coursePaths.map((path) => {
  const pack = JSON.parse(fs.readFileSync(path, 'utf8'))
  return {
    path,
    courseId: pack.courseId,
    contentVersion: pack.contentVersion,
    title: pack.title,
    perspective: pack.perspective,
    defaultDepth: pack.defaultDepth,
    lessons: pack.lessons?.length ?? 0,
    moves: pack.moves?.length ?? 0,
    notes: pack.notes?.length ?? 0,
    prompts: pack.prompts?.length ?? 0,
    lessonEdges: pack.lessonEdges?.length ?? 0,
    expectedReferences: pack.sourceCoverage?.expectedReferences?.length ?? 0
  }
})
console.log(JSON.stringify({generatedFor: 'mco15-sicilian-white', courses}, null, 2))
JS
jq '{courseCount:(.courses|length), courses:[.courses[] | {courseId,contentVersion,title,perspective,lessons,expectedReferences}]}' \
  /private/tmp/sicilian-white-library-baseline.json
```

Expected:

- `courseCount` is `7`.
- No existing course has course ID `mco15-sicilian-white`.
- Existing `mco15-sicilian-black` remains present and unchanged.

- [ ] **Step 4: Create the Sicilian White source-scope control file**

Run:

```bash
node --input-type=module - <<'JS' > /private/tmp/sicilian-white-source-scope.json
import fs from 'node:fs'

const blackAuthoring = JSON.parse(fs.readFileSync('/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json', 'utf8'))
const blackPack = JSON.parse(fs.readFileSync('/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse', 'utf8'))
const scope = blackAuthoring.scope
const expected = blackPack.sourceCoverage.expectedReferences
const requiredFamilies = [
  { familyId: 'open-skeleton', title: 'Open Sicilian skeleton', minimumDepth: 'quick' },
  { familyId: 'd6-family', title: '...d6 systems', minimumDepth: 'standard' },
  { familyId: 'nc6-family', title: '...Nc6 systems', minimumDepth: 'standard' },
  { familyId: 'e6-family', title: '...e6 systems', minimumDepth: 'standard' },
  { familyId: 'reference-map', title: 'Full Sicilian reference map', minimumDepth: 'reference' }
]
const requiredStandardCoverage = [
  'najdorf',
  'dragon',
  'classical',
  'scheveningen',
  'sveshnikov',
  'kalashnikov',
  'kan',
  'taimanov'
]
const requiredReferenceBranches = [
  'sicilian-foundations',
  'najdorf',
  'dragon',
  'accelerated-dragon',
  'scheveningen',
  'classical',
  'sveshnikov',
  'kalashnikov',
  'kan',
  'taimanov',
  'four-knights',
  'anti-sicilian-signposts'
]
const summary = {
  courseId: 'mco15-sicilian-white',
  title: 'Sicilian Defence for White',
  contentVersion: '1.0.0',
  perspective: 'white',
  defaultDepth: 'reference',
  sourcePdf: '/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf',
  printedFirst: 244,
  printedLast: 361,
  pdfFirst: 261,
  pdfLast: 378,
  scanOffset: 17,
  pageCount: 118,
  inheritedCoordinateSource: 'mco15-sicilian-black source scope',
  blackScopeFamilies: scope.families,
  pdfPages: scope.pdfPages,
  expectedReferences: expected,
  requiredFamilies,
  requiredStandardCoverage,
  requiredReferenceBranches,
  sourceScopePolicy: 'reuse-coordinate-mapping-only-create-white-inventory',
  privacyPolicy: 'private-course-content-outside-git'
}
console.log(JSON.stringify(summary, null, 2))
JS
jq '{courseId,title,contentVersion,perspective,printedFirst,printedLast,pdfFirst,pdfLast,pageCount,expectedReferences:(.expectedReferences|length),requiredStandardCoverage,requiredReferenceBranches}' \
  /private/tmp/sicilian-white-source-scope.json
```

Expected:

- `courseId` is `mco15-sicilian-white`.
- `perspective` is `white`.
- `printedFirst` is `244`.
- `printedLast` is `361`.
- `pdfFirst` is `261`.
- `pdfLast` is `378`.
- `pageCount` is `118`.
- `expectedReferences` is `118`.

- [ ] **Step 5: Create the baseline checkpoint**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-white-v1-baseline"
mkdir -p "$CHECKPOINT"
cp -p /private/tmp/sicilian-white-source-scope.json "$CHECKPOINT/"
cp -p /private/tmp/sicilian-white-library-baseline.json "$CHECKPOINT/"
cp -p /private/tmp/sicilian-black-baseline-validation.json "$CHECKPOINT/"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json" "$CHECKPOINT/source-coordinate-baseline-sicilian-black.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse" "$CHECKPOINT/source-coordinate-baseline-sicilian-black.ctcourse"
(
  cd "$CHECKPOINT"
  find . -maxdepth 1 -type f ! -name SHA256SUMS -print0 | sort -z | xargs -0 shasum -a 256 > SHA256SUMS
  shasum -a 256 -c SHA256SUMS
)
```

Expected: every checkpoint payload prints `OK`.

---

### Task 2: Build private Sicilian White source inventory and authoring shell

**Files:**
- Create/update: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.authoring.json`
- Read: `/private/tmp/sicilian-white-source-scope.json`
- Create: `/private/tmp/sicilian-white-inventory-check.mjs`
- Create: `/private/tmp/sicilian-white-inventory-check.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-white-v1-inventory/`

**Interfaces:**
- Consumes: source-scope control file from Task 1.
- Produces: a White-facing authoring shell with 118 source records, stable teaching node IDs, family tags, and audit output.

- [ ] **Step 1: Create the private authoring shell**

Run:

```bash
node --input-type=module - <<'JS'
import fs from 'node:fs'

const scope = JSON.parse(fs.readFileSync('/private/tmp/sicilian-white-source-scope.json', 'utf8'))
const outPath = '/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.authoring.json'
const familyForPage = (printedPage) => {
  if (printedPage <= 245) return 'sicilian-foundations'
  if (printedPage <= 268) return 'najdorf'
  if (printedPage <= 289) return 'dragon-accelerated-dragon'
  if (printedPage <= 317) return 'scheveningen-e6-systems'
  if (printedPage <= 345) return 'classical-nc6-e5-systems'
  return 'anti-sicilians'
}
const minDepthForFamily = (familyId) => {
  if (familyId === 'sicilian-foundations') return 'quick'
  if (familyId === 'anti-sicilians') return 'reference'
  return 'standard'
}
const standardTagsForFamily = (familyId) => {
  if (familyId === 'najdorf') return ['d6', 'najdorf']
  if (familyId === 'dragon-accelerated-dragon') return ['d6', 'dragon', 'accelerated-dragon', 'nc6']
  if (familyId === 'scheveningen-e6-systems') return ['d6', 'e6', 'scheveningen', 'kan', 'taimanov', 'four-knights']
  if (familyId === 'classical-nc6-e5-systems') return ['nc6', 'classical', 'sveshnikov', 'kalashnikov']
  return []
}
const referenceTagsForFamily = (familyId) => {
  if (familyId === 'sicilian-foundations') return ['sicilian-foundations', 'open-sicilian']
  if (familyId === 'najdorf') return ['najdorf', 'd6']
  if (familyId === 'dragon-accelerated-dragon') return ['dragon', 'accelerated-dragon', 'd6', 'nc6']
  if (familyId === 'scheveningen-e6-systems') return ['scheveningen', 'e6', 'kan', 'taimanov', 'four-knights']
  if (familyId === 'classical-nc6-e5-systems') return ['classical', 'nc6', 'sveshnikov', 'kalashnikov']
  return ['anti-sicilian-signposts']
}
const coverageRecords = scope.expectedReferences.map((coverageId) => {
  const match = /^p(\d+)-/.exec(coverageId)
  if (!match) throw new Error(`unexpected coverage id ${coverageId}`)
  const printedPage = Number(match[1])
  const familyId = familyForPage(printedPage)
  return {
    coverageId,
    familyId,
    printedPage,
    pdfPage: printedPage + scope.scanOffset,
    kind: 'page-anchor',
    locator: `Printed p.${printedPage} Sicilian source page anchor`,
    teachingUse: minDepthForFamily(familyId),
    perspectiveUse: 'white',
    standardCoverage: standardTagsForFamily(familyId),
    referenceBranches: referenceTagsForFamily(familyId),
    tags: [...new Set(['open-sicilian', familyId, ...standardTagsForFamily(familyId), ...referenceTagsForFamily(familyId)])].sort(),
    privateSummary: ''
  }
})
const teachingNodes = [
  {lessonId:'open-enter-sicilian',title:'Enter the Sicilian',chapterId:'open-skeleton',minimumDepth:'quick'},
  {lessonId:'open-prepare-nf3',title:'Prepare the centre with Nf3',chapterId:'open-skeleton',minimumDepth:'quick'},
  {lessonId:'open-central-d4',title:'Open the centre with d4',chapterId:'open-skeleton',minimumDepth:'quick'},
  {lessonId:'open-recapture-nxd4',title:'Recapture into the Open Sicilian',chapterId:'open-skeleton',minimumDepth:'quick'},
  {lessonId:'setup-family-recognition',title:\"Recognize Black's setup family\",chapterId:'open-skeleton',minimumDepth:'quick'},
  {lessonId:'d6-family-recognition',title:'Recognize ...d6 systems',chapterId:'d6-systems',minimumDepth:'standard'},
  {lessonId:'najdorf-recognition',title:'Najdorf recognition',chapterId:'d6-systems',minimumDepth:'standard'},
  {lessonId:'dragon-recognition',title:'Dragon recognition',chapterId:'d6-systems',minimumDepth:'standard'},
  {lessonId:'scheveningen-recognition',title:'Scheveningen recognition',chapterId:'d6-systems',minimumDepth:'standard'},
  {lessonId:'nc6-family-recognition',title:'Recognize ...Nc6 systems',chapterId:'nc6-systems',minimumDepth:'standard'},
  {lessonId:'classical-recognition',title:'Classical Sicilian recognition',chapterId:'nc6-systems',minimumDepth:'standard'},
  {lessonId:'sveshnikov-kalashnikov-recognition',title:'Sveshnikov and Kalashnikov recognition',chapterId:'nc6-systems',minimumDepth:'standard'},
  {lessonId:'e6-family-recognition',title:'Recognize ...e6 systems',chapterId:'e6-systems',minimumDepth:'standard'},
  {lessonId:'kan-taimanov-recognition',title:'Kan and Taimanov recognition',chapterId:'e6-systems',minimumDepth:'standard'},
  {lessonId:'white-plan-selection',title:\"Choose White's Open Sicilian plan\",chapterId:'open-plans',minimumDepth:'standard'},
  {lessonId:'d6-reference-branches',title:'...d6 Reference branches',chapterId:'reference-map',minimumDepth:'reference'},
  {lessonId:'nc6-reference-branches',title:'...Nc6 Reference branches',chapterId:'reference-map',minimumDepth:'reference'},
  {lessonId:'e6-reference-branches',title:'...e6 Reference branches',chapterId:'reference-map',minimumDepth:'reference'},
  {lessonId:'anti-sicilian-reference-signposts',title:'Anti-Sicilian reference signposts',chapterId:'reference-map',minimumDepth:'reference'}
].map((node, index) => ({
  nodeId: node.lessonId,
  ...node,
  ordinal: index + 1,
  requiredFlow: node.minimumDepth === 'reference' ? 'concept-reference' : 'concept-decision-recap'
}))
const chapterMap = [
  {chapterId:'open-skeleton',title:'Open Sicilian Skeleton',minimumDepth:'quick'},
  {chapterId:'d6-systems',title:'...d6 Systems',minimumDepth:'standard'},
  {chapterId:'nc6-systems',title:'...Nc6 Systems',minimumDepth:'standard'},
  {chapterId:'e6-systems',title:'...e6 Systems',minimumDepth:'standard'},
  {chapterId:'open-plans',title:'White Plan Selection',minimumDepth:'standard'},
  {chapterId:'reference-map',title:'Sicilian Reference Map',minimumDepth:'reference'}
]
const teachingEdges = [
  ['open-enter-sicilian','open-prepare-nf3','continuation','quick',''],
  ['open-prepare-nf3','open-central-d4','continuation','quick',''],
  ['open-central-d4','open-recapture-nxd4','continuation','quick',''],
  ['open-recapture-nxd4','setup-family-recognition','continuation','quick',''],
  ['setup-family-recognition','d6-family-recognition','alternative','standard','...d6 family'],
  ['setup-family-recognition','nc6-family-recognition','alternative','standard','...Nc6 family'],
  ['setup-family-recognition','e6-family-recognition','alternative','standard','...e6 family'],
  ['d6-family-recognition','najdorf-recognition','alternative','standard','Najdorf'],
  ['d6-family-recognition','dragon-recognition','alternative','standard','Dragon'],
  ['d6-family-recognition','scheveningen-recognition','alternative','standard','Scheveningen'],
  ['nc6-family-recognition','classical-recognition','alternative','standard','Classical'],
  ['nc6-family-recognition','sveshnikov-kalashnikov-recognition','alternative','standard','Sveshnikov / Kalashnikov'],
  ['e6-family-recognition','kan-taimanov-recognition','alternative','standard','Kan / Taimanov'],
  ['setup-family-recognition','white-plan-selection','continuation','standard',''],
  ['d6-family-recognition','d6-reference-branches','reference','reference','...d6 reference'],
  ['nc6-family-recognition','nc6-reference-branches','reference','reference','...Nc6 reference'],
  ['e6-family-recognition','e6-reference-branches','reference','reference','...e6 reference'],
  ['setup-family-recognition','anti-sicilian-reference-signposts','reference','reference','Anti-Sicilian signposts']
].map(([fromLessonId, toLessonId, kind, minimumDepth, label], index) => ({
  edgeId: `edge-${fromLessonId}-to-${toLessonId}`,
  fromLessonId,
  toLessonId,
  ordinal: index + 1,
  kind,
  minimumDepth,
  label
}))
const authoring = {
  courseId: 'mco15-sicilian-white',
  title: 'Sicilian Defence for White',
  perspective: 'white',
  targetContentVersion: '1.0.0',
  paths: {
    authoringPath: outPath,
    coursePath: '/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse'
  },
  scope,
  inventory: {
    pages: scope.pdfPages,
    coverageRecords
  },
  teachingNodes,
  chapterMap,
  teachingEdges,
  noteDrafts: {}
}
fs.writeFileSync(outPath, JSON.stringify(authoring, null, 2) + '\n')
console.log(JSON.stringify({
  outPath,
  teachingNodes: teachingNodes.length,
  teachingEdges: teachingEdges.length,
  coverageRecords: coverageRecords.length,
  chapterMap: chapterMap.length
}, null, 2))
JS
jq '{courseId,title,perspective,targetContentVersion,coverageRecords:(.inventory.coverageRecords|length),teachingNodes:(.teachingNodes|length),teachingEdges:(.teachingEdges|length),chapterMap:(.chapterMap|length)}' \
  "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.authoring.json"
```

Expected:

- `coverageRecords` is `118`.
- `teachingNodes` is `19`.
- `teachingEdges` is `18`.
- `chapterMap` is `6`.
- Every `inventory.coverageRecords[].privateSummary` exists and is empty, ready for White-facing private authoring.

- [ ] **Step 2: Fill White-facing private summaries**

Edit `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.authoring.json` manually from the private PDF/source pages so every `inventory.coverageRecords[].privateSummary` is a short original White-facing summary.

Rules for each summary:

```text
1. Use original wording.
2. Describe what White needs to learn from that page.
3. Keep Black-side lines as opponent pressure or family recognition.
4. Do not copy table prose or note prose.
5. Do not write generic filler such as "Source note x records..." or "concrete analytical subvariation".
6. Use 1-3 concise sentences per page anchor.
```

Expected: every `privateSummary` is non-empty before Step 3.

- [ ] **Step 3: Create the private inventory checker**

Create `/private/tmp/sicilian-white-inventory-check.mjs` with this exact script:

```javascript
import fs from 'node:fs'

const authoringPath = '/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.authoring.json'
const scopePath = '/private/tmp/sicilian-white-source-scope.json'
const authoring = JSON.parse(fs.readFileSync(authoringPath, 'utf8'))
const scope = JSON.parse(fs.readFileSync(scopePath, 'utf8'))
const failures = []

if (authoring.courseId !== 'mco15-sicilian-white') failures.push(`courseId ${authoring.courseId}`)
if (authoring.perspective !== 'white') failures.push(`perspective ${authoring.perspective}`)
if (authoring.targetContentVersion !== '1.0.0') failures.push(`targetContentVersion ${authoring.targetContentVersion}`)

const teachingNodes = new Set((authoring.teachingNodes || []).map((node) => node.nodeId || node.lessonId))
for (const nodeId of [
  'open-enter-sicilian',
  'open-prepare-nf3',
  'open-central-d4',
  'open-recapture-nxd4',
  'setup-family-recognition',
  'd6-family-recognition',
  'najdorf-recognition',
  'dragon-recognition',
  'scheveningen-recognition',
  'nc6-family-recognition',
  'classical-recognition',
  'sveshnikov-kalashnikov-recognition',
  'e6-family-recognition',
  'kan-taimanov-recognition',
  'white-plan-selection',
  'd6-reference-branches',
  'nc6-reference-branches',
  'e6-reference-branches',
  'anti-sicilian-reference-signposts'
]) {
  if (!teachingNodes.has(nodeId)) failures.push(`missing teaching node ${nodeId}`)
}

const chapters = new Set((authoring.chapterMap || authoring.chapters || []).map((chapter) => chapter.chapterId))
for (const chapterId of ['open-skeleton', 'd6-systems', 'nc6-systems', 'e6-systems', 'open-plans', 'reference-map']) {
  if (!chapters.has(chapterId)) failures.push(`missing chapter ${chapterId}`)
}

const inventory = Array.isArray(authoring.inventory)
  ? authoring.inventory
  : Array.isArray(authoring.inventory?.coverageRecords)
    ? authoring.inventory.coverageRecords
    : authoring.sourceInventory || []
const expected = new Set(scope.expectedReferences || [])
const captured = new Set()
for (const record of inventory) {
  const coverageId = record.coverageId || record.sourceRef?.coverageId
  if (!coverageId) {
    failures.push(`inventory record missing coverageId: ${JSON.stringify(record).slice(0, 160)}`)
    continue
  }
  captured.add(coverageId)
  if (!expected.has(coverageId)) failures.push(`unexpected inventory coverage ${coverageId}`)
  if (record.perspectiveUse !== 'white') failures.push(`${coverageId} perspectiveUse ${record.perspectiveUse}`)
  const text = [record.privateSummary || '', record.teachingText || '', record.noteText || ''].join(' ')
  if (text.trim().length < 30) failures.push(`${coverageId} privateSummary too short`)
  if (/Source note [a-z] records|Course move found|opening activity result|concrete analytical subvariation/i.test(text)) {
    failures.push(`spam/debug placeholder in inventory ${coverageId}`)
  }
}
for (const coverageId of expected) {
  if (!captured.has(coverageId)) failures.push(`missing inventory coverage ${coverageId}`)
}

const standardTags = new Set(inventory.flatMap((record) => record.standardCoverage || record.tags || []))
for (const tag of ['najdorf', 'dragon', 'classical', 'scheveningen', 'sveshnikov', 'kalashnikov', 'kan', 'taimanov']) {
  if (!standardTags.has(tag)) failures.push(`missing standard coverage tag ${tag}`)
}
const referenceTags = new Set(inventory.flatMap((record) => record.referenceBranches || record.tags || []))
for (const tag of ['sicilian-foundations', 'najdorf', 'dragon', 'accelerated-dragon', 'scheveningen', 'classical', 'sveshnikov', 'kalashnikov', 'kan', 'taimanov', 'four-knights', 'anti-sicilian-signposts']) {
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

- [ ] **Step 4: Run the inventory checker**

Run:

```bash
node /private/tmp/sicilian-white-inventory-check.mjs > /private/tmp/sicilian-white-inventory-check.json
jq '{ok,inventoryRecords,expectedReferences,capturedReferences,teachingNodes,chapters,failures}' \
  /private/tmp/sicilian-white-inventory-check.json
```

Expected:

- `ok` is `true`.
- `failures` is `[]`.
- `inventoryRecords` is `118`.
- `expectedReferences` is `118`.
- `capturedReferences` is `118`.
- `teachingNodes` is `19`.
- `chapters` is at least `6`.

- [ ] **Step 5: Checkpoint the source inventory**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-white-v1-inventory"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.authoring.json" "$CHECKPOINT/"
cp -p /private/tmp/sicilian-white-source-scope.json "$CHECKPOINT/"
cp -p /private/tmp/sicilian-white-inventory-check.json "$CHECKPOINT/"
(
  cd "$CHECKPOINT"
  find . -maxdepth 1 -type f ! -name SHA256SUMS -print0 | sort -z | xargs -0 shasum -a 256 > SHA256SUMS
  shasum -a 256 -c SHA256SUMS
)
```

Expected: every checkpoint payload prints `OK`.

---

### Task 3: Build and validate the Quick Open Sicilian skeleton

**Files:**
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.authoring.json`
- Create/update: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse`
- Create: `/private/tmp/sicilian-white-quick-validation.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-white-v1-quick/`

**Interfaces:**
- Consumes: White-facing inventory and teaching nodes from Task 2.
- Produces: schema-v2 course pack with validated Quick skeleton and course identity.

- [ ] **Step 1: Author the Quick move graph**

Create the Quick route in the private course pack using these stable position and move IDs:

```text
positions:
initial
after-e4
after-c5
after-nf3
after-d6
after-d4
after-cxd4
after-nxd4
after-nf6
after-nc3
family-split

moves:
m-e4                 initial -> after-e4        e2e4  quick repertoire
m-c5                 after-e4 -> after-c5       c7c5  quick opponent
m-nf3                after-c5 -> after-nf3      g1f3  quick repertoire
m-d6-quick           after-nf3 -> after-d6      d7d6  quick opponent
m-d4                 after-d6 -> after-d4       d2d4  quick repertoire
m-cxd4               after-d4 -> after-cxd4     c5d4  quick opponent
m-nxd4               after-cxd4 -> after-nxd4   f3d4  quick repertoire
m-nf6-quick          after-nxd4 -> after-nf6    g8f6  quick opponent
m-nc3                after-nf6 -> after-nc3     b1c3  quick repertoire
m-a6-family-signal   after-nc3 -> family-split  a7a6  quick opponent
```

Rules:

- The root FEN must be the normal chess initial position.
- `m-e4`, `m-nf3`, `m-d4`, `m-nxd4`, and `m-nc3` are White repertoire moves.
- Black moves are `opponent`.
- The Quick path is legal from `rootFen`.
- Every move has a source reference from the inventory.

- [ ] **Step 2: Author Quick lessons**

The Quick visible lessons must be exactly these lesson IDs in this order:

```text
open-enter-sicilian
open-prepare-nf3
open-central-d4
open-recapture-nxd4
setup-family-recognition
```

Required Quick prompts:

```text
p-open-e4: initial -> m-e4
p-open-nf3: after-c5 -> m-nf3
p-open-d4: after-d6 -> m-d4
p-open-nxd4: after-cxd4 -> m-nxd4
p-open-nc3: after-nf6 -> m-nc3
```

Each Quick lesson must have 2-3 required activities chosen from `concept`, `decision`, `demonstration`, `comparison`, and `recap`. Each Quick lesson has at most one required decision. Do not require the learner to replay earlier Quick decisions inside later Quick lessons.

- [ ] **Step 3: Write the Quick pack**

Write `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse` as schema-v2 JSON with these top-level fields:

```json
{
  "schemaVersion": 2,
  "courseId": "mco15-sicilian-white",
  "contentVersion": "1.0.0",
  "title": "Sicilian Defence for White",
  "perspective": "white",
  "defaultDepth": "reference"
}
```

The pack's Quick lesson tree must connect:

```text
open-enter-sicilian
└─ open-prepare-nf3
   └─ open-central-d4
      └─ open-recapture-nxd4
         └─ setup-family-recognition
```

- [ ] **Step 4: Validate the Quick pack**

Run:

```bash
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse" \
  > /private/tmp/sicilian-white-quick-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length),diagnostics}' \
  /private/tmp/sicilian-white-quick-validation.json
```

Expected:

- `courseId` is `mco15-sicilian-white`.
- `contentVersion` is `1.0.0`.
- `counts.warnings` is `0`.
- `counts.lessons` is at least `5`.
- `missing` is `0`.
- `unexpected` is `0`.
- `diagnostics` is `null` or `[]`.

- [ ] **Step 5: Audit Quick lesson shape**

Run:

```bash
jq '{
  quickLessons: [.lessons[] | select(.minimumDepth=="quick") | {lessonId,title,activities:(.activities|length),required:([.activities[] | select(.required==true)]|length),decisionCount:([.activities[] | select(.required==true and .kind=="decision")]|length)}],
  quickLessonCount: ([.lessons[] | select(.minimumDepth=="quick")] | length)
}' "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse"
```

Expected:

- `quickLessonCount` is `5`.
- Every Quick lesson has `required` between `2` and `3`.
- No Quick lesson has `decisionCount` greater than `1`.

- [ ] **Step 6: Checkpoint Quick**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-white-v1-quick"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.authoring.json" "$CHECKPOINT/"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse" "$CHECKPOINT/"
cp -p /private/tmp/sicilian-white-quick-validation.json "$CHECKPOINT/"
(
  cd "$CHECKPOINT"
  find . -maxdepth 1 -type f ! -name SHA256SUMS -print0 | sort -z | xargs -0 shasum -a 256 > SHA256SUMS
  shasum -a 256 -c SHA256SUMS
)
```

Expected: every checkpoint payload prints `OK`.

---

### Task 4: Build and validate Standard family-recognition coverage

**Files:**
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.authoring.json`
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse`
- Create: `/private/tmp/sicilian-white-standard-validation.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-white-v1-standard/`

**Interfaces:**
- Consumes: Quick pack from Task 3.
- Produces: Standard layer that covers required Open Sicilian families from White's perspective.

- [ ] **Step 1: Add Standard family moves and branch positions**

Add legal branch moves from the Quick family split so Standard lessons can show the required setup families.

Required Standard branch tags and examples:

```text
d6-family: Najdorf, Dragon, Scheveningen, Classical-style structures
nc6-family: Classical, Sveshnikov, Kalashnikov, Accelerated Dragon-style structures
e6-family: Kan, Taimanov, Four Knights, Scheveningen transpositions
```

Required branch-recognition lesson IDs:

```text
d6-family-recognition
najdorf-recognition
dragon-recognition
scheveningen-recognition
nc6-family-recognition
classical-recognition
sveshnikov-kalashnikov-recognition
e6-family-recognition
kan-taimanov-recognition
white-plan-selection
```

- [ ] **Step 2: Author Standard lesson activities**

Each Standard lesson must have 3-4 required activities and at most one required decision.

Required Standard decisions:

```text
d6-family-recognition: identify ...d6 as the flexible Open Sicilian family.
najdorf-recognition: choose the White development/plan signal against ...a6.
dragon-recognition: choose the White setup signal against ...g6.
scheveningen-recognition: choose the White setup signal against ...e6 with ...d6.
nc6-family-recognition: identify ...Nc6 family pressure.
classical-recognition: choose a natural White development setup against Classical ...Nc6/...d6.
sveshnikov-kalashnikov-recognition: identify the ...e5 central clamp.
e6-family-recognition: identify ...e6 as Kan/Taimanov/Scheveningen territory.
kan-taimanov-recognition: choose development that respects ...Qc7/...Nc6 pressure.
white-plan-selection: compare central control, kingside attack, and opposite-side castling signals.
```

Expected: no Standard lesson asks the learner to replay the full `e4 c5 Nf3 d6 d4 cxd4 Nxd4` stem.

- [ ] **Step 3: Wire Standard lessons into the teaching tree**

Update `lessonEdges` so all Standard lessons are reachable from `setup-family-recognition`:

```text
setup-family-recognition
├─ d6-family-recognition
│  ├─ najdorf-recognition
│  ├─ dragon-recognition
│  └─ scheveningen-recognition
├─ nc6-family-recognition
│  ├─ classical-recognition
│  └─ sveshnikov-kalashnikov-recognition
├─ e6-family-recognition
│  └─ kan-taimanov-recognition
└─ white-plan-selection
```

Use `alternative` edge kind with non-empty labels for branch families. Use `continuation` only for the main Quick skeleton and `white-plan-selection` route.

- [ ] **Step 4: Validate the Standard pack**

Run:

```bash
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse" \
  > /private/tmp/sicilian-white-standard-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length),diagnostics}' \
  /private/tmp/sicilian-white-standard-validation.json
```

Expected:

- `contentVersion` is `1.0.0`.
- `counts.warnings` is `0`.
- `counts.lessons` is at least `15`.
- `counts.prompts` is at least `10`.
- `missing` is `0`.
- `unexpected` is `0`.
- `diagnostics` is `null` or `[]`.

- [ ] **Step 5: Audit Standard coverage**

Run:

```bash
jq '{
  standardLessons: [.lessons[] | select(.minimumDepth=="standard") | {lessonId,title,kinds:[.activities[].kind],required:([.activities[] | select(.required==true)]|length),decisionCount:([.activities[] | select(.required==true and .kind=="decision")]|length)}],
  requiredIds: {
    d6: any(.lessons[]; .lessonId=="d6-family-recognition"),
    najdorf: any(.lessons[]; .lessonId=="najdorf-recognition"),
    dragon: any(.lessons[]; .lessonId=="dragon-recognition"),
    scheveningen: any(.lessons[]; .lessonId=="scheveningen-recognition"),
    nc6: any(.lessons[]; .lessonId=="nc6-family-recognition"),
    classical: any(.lessons[]; .lessonId=="classical-recognition"),
    sveshnikovKalashnikov: any(.lessons[]; .lessonId=="sveshnikov-kalashnikov-recognition"),
    e6: any(.lessons[]; .lessonId=="e6-family-recognition"),
    kanTaimanov: any(.lessons[]; .lessonId=="kan-taimanov-recognition"),
    planSelection: any(.lessons[]; .lessonId=="white-plan-selection")
  }
}' "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse"
```

Expected:

- Every `requiredIds` value is `true`.
- Every Standard lesson has 3-4 required activities unless a comparison-only bridge intentionally has 2 required activities.
- No Standard lesson has more than one required decision from the same position.

- [ ] **Step 6: Checkpoint Standard**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-white-v1-standard"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.authoring.json" "$CHECKPOINT/"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse" "$CHECKPOINT/"
cp -p /private/tmp/sicilian-white-standard-validation.json "$CHECKPOINT/"
(
  cd "$CHECKPOINT"
  find . -maxdepth 1 -type f ! -name SHA256SUMS -print0 | sort -z | xargs -0 shasum -a 256 > SHA256SUMS
  shasum -a 256 -c SHA256SUMS
)
```

Expected: every checkpoint payload prints `OK`.

---

### Task 5: Build and validate Reference map

**Files:**
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.authoring.json`
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse`
- Create: `/private/tmp/sicilian-white-reference-validation.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-white-v1-reference/`

**Interfaces:**
- Consumes: Standard pack from Task 4.
- Produces: Reference-depth source-backed Sicilian lookup tree with optional dense material.

- [ ] **Step 1: Add Reference lessons**

Ensure these Reference lesson IDs are present:

```text
d6-reference-branches
nc6-reference-branches
e6-reference-branches
anti-sicilian-reference-signposts
```

Each Reference lesson must:

- have at least one required `concept` activity;
- keep dense branch material in optional `reference` activities;
- avoid required repeated decisions;
- expose source-backed branches in the variation explorer.

- [ ] **Step 2: Cover required Reference tags**

The Reference layer must include notes, moves, lesson tags, or private inventory tags for:

```text
sicilian-foundations
najdorf
dragon
accelerated-dragon
scheveningen
classical
sveshnikov
kalashnikov
kan
taimanov
four-knights
anti-sicilian-signposts
```

Expected: Anti-Sicilian content is signposted as optional Reference material, not required Quick/Standard repertoire.

- [ ] **Step 3: Wire Reference lessons into the teaching tree**

Update `lessonEdges` so Reference lessons are reachable:

```text
d6-family-recognition -> d6-reference-branches
nc6-family-recognition -> nc6-reference-branches
e6-family-recognition -> e6-reference-branches
setup-family-recognition -> anti-sicilian-reference-signposts
```

Use `reference` edge kind with non-empty labels.

- [ ] **Step 4: Validate the Reference pack**

Run:

```bash
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse" \
  > /private/tmp/sicilian-white-reference-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length),diagnostics}' \
  /private/tmp/sicilian-white-reference-validation.json
```

Expected:

- `contentVersion` is `1.0.0`.
- `counts.warnings` is `0`.
- `counts.lessons` is at least `19`.
- `counts.lessonEdges` is at least `18`.
- `missing` is `0`.
- `unexpected` is `0`.
- `diagnostics` is `null` or `[]`.

- [ ] **Step 5: Audit Reference source coverage**

Run:

```bash
node --input-type=module - <<'JS'
import fs from 'node:fs'
const authoring = JSON.parse(fs.readFileSync('/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.authoring.json', 'utf8'))
const pack = JSON.parse(fs.readFileSync('/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse', 'utf8'))
const requiredTags = ['sicilian-foundations','najdorf','dragon','accelerated-dragon','scheveningen','classical','sveshnikov','kalashnikov','kan','taimanov','four-knights','anti-sicilian-signposts']
const text = JSON.stringify({authoringInventory: authoring.inventory || authoring.sourceInventory || [], notes: pack.notes, moves: pack.moves, lessons: pack.lessons})
const missingTags = requiredTags.filter((tag) => !text.includes(tag))
const expected = new Set(pack.sourceCoverage.expectedReferences || [])
const seen = new Set()
for (const item of [...(pack.moves || []), ...(pack.notes || [])]) {
  const coverageId = item.sourceRef?.coverageId
  if (coverageId) seen.add(coverageId)
}
const missingCoverage = [...expected].filter((coverageId) => !seen.has(coverageId))
console.log(JSON.stringify({ok: missingTags.length === 0 && missingCoverage.length === 0, missingTags, missingCoverage, expectedReferences: expected.size, capturedReferences: seen.size}, null, 2))
if (missingTags.length || missingCoverage.length) process.exit(1)
JS
```

Expected:

- `ok` is `true`.
- `missingTags` is `[]`.
- `missingCoverage` is `[]`.
- `expectedReferences` is `118`.

- [ ] **Step 6: Checkpoint Reference**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-white-v1-reference"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.authoring.json" "$CHECKPOINT/"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse" "$CHECKPOINT/"
cp -p /private/tmp/sicilian-white-reference-validation.json "$CHECKPOINT/"
(
  cd "$CHECKPOINT"
  find . -maxdepth 1 -type f ! -name SHA256SUMS -print0 | sort -z | xargs -0 shasum -a 256 > SHA256SUMS
  shasum -a 256 -c SHA256SUMS
)
```

Expected: every checkpoint payload prints `OK`.

---

### Task 6: Final private audit and import smokes

**Files:**
- Create: `/private/tmp/sicilian-white-course-audit.mjs`
- Create: `/private/tmp/sicilian-white-final-audit.json`
- Create: `/private/tmp/sicilian-white-final-validation.json`
- Create: `.superpowers/tmp/sicilian-white-import-smoke.go`
- Create: `.superpowers/tmp/sicilian-white-service-smoke.go`
- Create: `/private/tmp/sicilian-white-disposable-import.json`
- Create: `/private/tmp/sicilian-white-default-import.json`
- Create: `/private/tmp/sicilian-white-service-smoke.json`

**Interfaces:**
- Consumes: final Reference course pack from Task 5.
- Produces: proof that the new course imports safely, coexists with existing courses, and exposes White orientation.

- [ ] **Step 1: Create final course audit script**

Create `/private/tmp/sicilian-white-course-audit.mjs` with this exact script:

```javascript
import fs from 'node:fs'

const packPath = '/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse'
const pack = JSON.parse(fs.readFileSync(packPath, 'utf8'))
const failures = []

const byId = (items, key) => new Map(items.map((item) => [item[key], item]))
const lessons = byId(pack.lessons || [], 'lessonId')
const prompts = byId(pack.prompts || [], 'promptId')
const moves = byId(pack.moves || [], 'moveId')
const notes = pack.notes || []

if (pack.courseId !== 'mco15-sicilian-white') failures.push(`courseId ${pack.courseId}`)
if (pack.contentVersion !== '1.0.0') failures.push(`contentVersion ${pack.contentVersion}`)
if (pack.title !== 'Sicilian Defence for White') failures.push(`title ${pack.title}`)
if (pack.perspective !== 'white') failures.push(`perspective ${pack.perspective}`)
if (pack.defaultDepth !== 'reference') failures.push(`defaultDepth ${pack.defaultDepth}`)

for (const id of ['open-enter-sicilian','open-prepare-nf3','open-central-d4','open-recapture-nxd4','setup-family-recognition']) {
  const lesson = lessons.get(id)
  if (!lesson) failures.push(`missing quick lesson ${id}`)
  else if (lesson.minimumDepth !== 'quick') failures.push(`${id} depth ${lesson.minimumDepth}`)
}
for (const id of ['d6-family-recognition','najdorf-recognition','dragon-recognition','scheveningen-recognition','nc6-family-recognition','classical-recognition','sveshnikov-kalashnikov-recognition','e6-family-recognition','kan-taimanov-recognition','white-plan-selection']) {
  const lesson = lessons.get(id)
  if (!lesson) failures.push(`missing standard lesson ${id}`)
  else if (lesson.minimumDepth !== 'standard') failures.push(`${id} depth ${lesson.minimumDepth}`)
}
for (const id of ['d6-reference-branches','nc6-reference-branches','e6-reference-branches','anti-sicilian-reference-signposts']) {
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
for (const tag of ['najdorf','dragon','classical','scheveningen','sveshnikov','kalashnikov','kan','taimanov']) {
  if (!serialized.includes(tag)) failures.push(`missing standard branch/tag text ${tag}`)
}
const expectedReferences = pack.sourceCoverage?.expectedReferences ?? []
if (expectedReferences.length !== 118) failures.push(`expectedReferences ${expectedReferences.length}`)

const summary = {
  ok: failures.length === 0,
  failures,
  counts: {
    chapters: pack.chapters?.length ?? 0,
    lessons: pack.lessons?.length ?? 0,
    moves: pack.moves?.length ?? 0,
    notes: notes.length,
    prompts: pack.prompts?.length ?? 0,
    lessonEdges: pack.lessonEdges?.length ?? 0,
    expectedReferences: expectedReferences.length
  }
}
console.log(JSON.stringify(summary, null, 2))
if (failures.length) process.exit(1)
```

- [ ] **Step 2: Run final audit and validator**

Run:

```bash
node /private/tmp/sicilian-white-course-audit.mjs > /private/tmp/sicilian-white-final-audit.json
go run ./cmd/coursepack validate \
  "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse" \
  > /private/tmp/sicilian-white-final-validation.json
jq -s '.[0] as $validation | .[1] as $audit | {auditOk:$audit.ok, auditFailures:$audit.failures, validation:{courseId:$validation.courseId,contentVersion:$validation.contentVersion,counts:$validation.counts,missing:($validation.coverage.missing|length),unexpected:($validation.coverage.unexpected|length),diagnostics:$validation.diagnostics}}' \
  /private/tmp/sicilian-white-final-validation.json /private/tmp/sicilian-white-final-audit.json
```

Expected:

- `auditOk` is `true`.
- `auditFailures` is `[]`.
- validation `courseId` is `mco15-sicilian-white`.
- validation `contentVersion` is `1.0.0`.
- `counts.warnings` is `0`.
- `missing` is `0`.
- `unexpected` is `0`.
- `diagnostics` is `null` or `[]`.

- [ ] **Step 3: Create app-layer import smoke harness**

Create `.superpowers/tmp/sicilian-white-import-smoke.go` with this exact content:

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
	coursePaths := splitRequired("SICILIAN_WHITE_COURSES")
	if len(coursePaths) == 0 {
		fail(fmt.Errorf("SICILIAN_WHITE_COURSES is required"))
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
			"path": coursePath,
			"sourceId": inspection.SourceID,
			"sourceName": inspection.SourceName,
			"report": report,
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
			"courseId": course.CourseID,
			"title": course.Title,
			"perspective": string(course.Perspective),
			"contentVersion": compiled.Pack.ContentVersion,
		})
	}
	missingRequired := []string{}
	for _, courseID := range splitOptional("SICILIAN_WHITE_REQUIRE_COURSES") {
		if _, ok := activeVersions[courseID]; !ok {
			missingRequired = append(missingRequired, courseID)
		}
	}
	imported, err := services.OpeningCatalog.LoadActive(ctx, "mco15-sicilian-white")
	if err != nil {
		fail(err)
	}
	ok := imported.Pack.CourseID == "mco15-sicilian-white" &&
		imported.Pack.ContentVersion == "1.0.0" &&
		imported.Pack.Perspective == "white" &&
		len(missingRequired) == 0
	write(map[string]any{
		"ok": ok,
		"dataRoot": paths.Root,
		"imports": imports,
		"activeCourses": activeCourses,
		"activeVersions": activeVersions,
		"missingRequiredCourses": missingRequired,
		"importedCourseId": imported.Pack.CourseID,
		"importedContentVersion": imported.Pack.ContentVersion,
		"perspective": imported.Pack.Perspective,
	})
	if !ok {
		os.Exit(1)
	}
}

func selectedPaths() (storage.Paths, error) {
	if os.Getenv("SICILIAN_WHITE_USE_DEFAULT_DATA") == "1" {
		return storage.DefaultPaths()
	}
	root := strings.TrimSpace(os.Getenv("SICILIAN_WHITE_DATA_ROOT"))
	if root == "" {
		return storage.Paths{}, fmt.Errorf("SICILIAN_WHITE_DATA_ROOT or SICILIAN_WHITE_USE_DEFAULT_DATA=1 is required")
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
SICILIAN_WHITE_DATA_ROOT="$(mktemp -d /private/tmp/sicilian-white-import.XXXXXX)" \
SICILIAN_WHITE_COURSES="/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse" \
go run .superpowers/tmp/sicilian-white-import-smoke.go \
  > /private/tmp/sicilian-white-disposable-import.json
jq . /private/tmp/sicilian-white-disposable-import.json
```

Expected:

- `ok` is `true`.
- `importedCourseId` is `mco15-sicilian-white`.
- `importedContentVersion` is `1.0.0`.
- `perspective` is `white`.

- [ ] **Step 5: Run default catalogue import smoke**

Run:

```bash
SICILIAN_WHITE_USE_DEFAULT_DATA=1 \
SICILIAN_WHITE_COURSES="/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse" \
SICILIAN_WHITE_REQUIRE_COURSES="mco15-italian-white|mco15-ruy-lopez-white|mco15-queens-gambit-white|mco15-queens-gambit-black|mco15-caro-kann-black|mco15-sicilian-black|mco15-french-black|mco15-sicilian-white" \
go run .superpowers/tmp/sicilian-white-import-smoke.go \
  > /private/tmp/sicilian-white-default-import.json
jq '{ok,activeCourses,missingRequiredCourses}' /private/tmp/sicilian-white-default-import.json
```

Expected:

- `ok` is `true`.
- `mco15-sicilian-white` is present with content version `1.0.0`.
- `mco15-sicilian-black` remains present.
- Existing White and Black courses remain present.

- [ ] **Step 6: Create app-service journey smoke harness**

Create `.superpowers/tmp/sicilian-white-service-smoke.go` with this exact content:

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

const courseID = "mco15-sicilian-white"

type lessonTrace struct {
	LessonID     string   `json:"lessonId"`
	Activities   []string `json:"activities"`
	Orientations []string `json:"orientations"`
}

func main() {
	ctx := context.Background()
	dataRoot := strings.TrimSpace(os.Getenv("SICILIAN_WHITE_SERVICE_DATA_ROOT"))
	if dataRoot == "" {
		fail(fmt.Errorf("SICILIAN_WHITE_SERVICE_DATA_ROOT is required"))
	}
	coursePath := strings.TrimSpace(os.Getenv("SICILIAN_WHITE_SERVICE_COURSE"))
	if coursePath == "" {
		fail(fmt.Errorf("SICILIAN_WHITE_SERVICE_COURSE is required"))
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
		if course.CourseID == courseID && course.Title == "Sicilian Defence for White" && course.Perspective == openings.PerspectiveWhite {
			homeHasWhiteCourse = true
		}
	}
	noteTexts := []string{}
	startedLessons := []string{}
	if err := services.Openings.SetDepth(ctx, courseID, openings.DepthQuick); err != nil {
		fail(err)
	}
	quickTrace, quickNotes, err := completeLesson(ctx, services, compiled, "open-enter-sicilian")
	if err != nil {
		fail(err)
	}
	startedLessons = append(startedLessons, quickTrace.LessonID)
	noteTexts = append(noteTexts, quickNotes...)
	if err := services.Openings.SetDepth(ctx, courseID, openings.DepthStandard); err != nil {
		fail(err)
	}
	standardTraces := []lessonTrace{}
	for _, lessonID := range []string{"najdorf-recognition", "dragon-recognition", "sveshnikov-kalashnikov-recognition", "kan-taimanov-recognition"} {
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
	referenceTrace, referenceNotes, err := startOnly(ctx, services, "anti-sicilian-reference-signposts")
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
		compiled.Pack.ContentVersion == "1.0.0" &&
		compiled.Pack.Perspective == openings.PerspectiveWhite &&
		len(spamMarkers) == 0 &&
		hasAll(startedLessons, []string{"open-enter-sicilian", "najdorf-recognition", "dragon-recognition", "sveshnikov-kalashnikov-recognition", "kan-taimanov-recognition", "anti-sicilian-reference-signposts"})
	write(map[string]any{
		"ok": ok,
		"courseId": compiled.Pack.CourseID,
		"contentVersion": compiled.Pack.ContentVersion,
		"orientation": compiled.Pack.Perspective,
		"homeHasWhiteCourse": homeHasWhiteCourse,
		"rootPositionId": compiled.Pack.RootPositionID,
		"explorerMoveCount": len(explorer.Moves),
		"startedLessons": startedLessons,
		"quickTrace": quickTrace,
		"standardTraces": standardTraces,
		"referenceTrace": referenceTrace,
		"spamMarkers": spamMarkers,
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

- [ ] **Step 7: Run app-service journey smoke**

Run:

```bash
SICILIAN_WHITE_SERVICE_DATA_ROOT="$(mktemp -d /private/tmp/sicilian-white-service.XXXXXX)" \
SICILIAN_WHITE_SERVICE_COURSE="/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse" \
go run .superpowers/tmp/sicilian-white-service-smoke.go \
  > /private/tmp/sicilian-white-service-smoke.json
jq . /private/tmp/sicilian-white-service-smoke.json
```

Expected:

- `ok` is `true`.
- `courseId` is `mco15-sicilian-white`.
- `contentVersion` is `1.0.0`.
- `orientation` is `white`.
- `startedLessons` includes the Quick, Najdorf, Dragon, Sveshnikov/Kalashnikov, Kan/Taimanov, and Reference lesson IDs listed above.
- `spamMarkers` is `[]`.

---

### Task 7: Final verification, checkpoint, and public implementation status

**Files:**
- Read: `/private/tmp/sicilian-white-final-audit.json`
- Read: `/private/tmp/sicilian-white-final-validation.json`
- Read: `/private/tmp/sicilian-white-disposable-import.json`
- Read: `/private/tmp/sicilian-white-default-import.json`
- Read: `/private/tmp/sicilian-white-service-smoke.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-white-v1-final/`
- Modify: `docs/superpowers/specs/2026-07-26-sicilian-white-open-course-design.md`
- Modify: `docs/superpowers/plans/2026-07-26-sicilian-white-open-course.md`

**Interfaces:**
- Consumes: final private course pack and verification outputs from Task 6.
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
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/sicilian-white-v1-final"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.authoring.json" "$CHECKPOINT/"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse" "$CHECKPOINT/"
cp -p /private/tmp/sicilian-white-source-scope.json "$CHECKPOINT/"
cp -p /private/tmp/sicilian-white-library-baseline.json "$CHECKPOINT/"
cp -p /private/tmp/sicilian-white-inventory-check.json "$CHECKPOINT/"
cp -p /private/tmp/sicilian-white-final-audit.json "$CHECKPOINT/"
cp -p /private/tmp/sicilian-white-final-validation.json "$CHECKPOINT/"
cp -p /private/tmp/sicilian-white-disposable-import.json "$CHECKPOINT/"
cp -p /private/tmp/sicilian-white-default-import.json "$CHECKPOINT/"
cp -p /private/tmp/sicilian-white-service-smoke.json "$CHECKPOINT/"
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
git ls-files | rg 'Private Chess Courses|mco15-sicilian-white\.(ctcourse|authoring\.json)$|/mco15-sicilian-white\.(ctcourse|authoring\.json)$' || true
```

Expected:

- Private files may appear only as ignored/untracked paths outside Git.
- `git ls-files` prints no private Sicilian White course artifacts.

- [ ] **Step 4: Update public implementation status**

Append this status block to `docs/superpowers/plans/2026-07-26-sicilian-white-open-course.md`:

```markdown
## Implementation status

- Private course pack: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse`
- Private authoring file: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.authoring.json`
- Content version: `1.0.0`
- Final validation: passed
- Final private audit: passed
- Disposable import smoke: passed
- Default catalogue import: passed
- App-service journey smoke: passed
- Automated app verification: passed via real Sicilian White app-service smoke plus Chromium/WebKit Playwright opening E2E
- Manual live app verification: not performed unless a claimable in-app browser tab or controllable Wails desktop session was available during implementation
```

Update `docs/superpowers/specs/2026-07-26-sicilian-white-open-course-design.md` status to:

```markdown
Status: implemented; automated verification passed; live manual Learn Openings acceptance outstanding
```

If live manual app acceptance was genuinely completed, replace the final two status lines with a precise pass statement and include the exact app/session used.

- [ ] **Step 5: Commit public docs**

Run:

```bash
git add \
  docs/superpowers/specs/2026-07-26-sicilian-white-open-course-design.md \
  docs/superpowers/plans/2026-07-26-sicilian-white-open-course.md
git commit -m "docs: mark sicilian white course implemented"
git status --short --branch
```

Expected:

- The commit contains only public docs.
- No private course artifacts are staged or committed.
