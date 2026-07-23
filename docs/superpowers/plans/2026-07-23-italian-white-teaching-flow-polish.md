# Italian White Teaching-Flow Polish Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Polish the existing private `mco15-italian-white` course so it feels like a continuous guided opening lesson path with fewer forced repeat moves and clearer branch memory.

**Architecture:** Keep the public app generic and reuse the existing schema-v2 course-pack model, validator, catalogue replacement importer, lesson UI, teaching tree, variation explorer, depth projection, and journey rebase behavior. Perform the content audit and edits privately through a generic audit/apply script plus a private edit manifest, then import the updated course as a replacement generation.

**Tech Stack:** Go 1.26.4, JSON `.ctcourse`, private authoring JSON, Node.js scripts for private audit/apply work, `jq`, Wails 2.12.0, Svelte, Vitest, Playwright, macOS app codesigning.

## Global Constraints

- Course ID remains exactly `mco15-italian-white`.
- Current source course version is `2.0.0`; polished target course version is `2.1.0`.
- Course perspective remains exactly `white`.
- Preserve stable lesson IDs, prompt IDs, and activity IDs wherever safe so existing journey and review state can rebase.
- Keep the current source scope: existing MCO-15 Italian material only.
- Do not add a new opening.
- Do not expand Italian beyond the existing source-page scope.
- Do not rewrite the opening-course engine, schema, or app navigation unless a small generic defect blocks this polish and has a failing synthetic test.
- Quick is a coherent beginner White repertoire; Standard is the practical Italian map; Reference keeps the full existing analysis without forcing it into required flow.
- Public commits contain only generic code, synthetic tests, docs, and app fixes.
- Do not commit private `.ctcourse`, private authoring JSON, rendered PDF pages, private checkpoints, private audit reports, private edit manifests, or MCO-derived prose to Git.

---

## File Map

Repository files:

- Already created: `docs/superpowers/specs/2026-07-23-italian-white-teaching-flow-polish-design.md`
- Create: `docs/superpowers/plans/2026-07-23-italian-white-teaching-flow-polish.md`
- Modify public Go/Svelte/tests only when a generic app defect is found and covered by synthetic tests.

Private files outside Git:

- Read/update: `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse`
- Read/update: `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.authoring.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/italian-v2-before-flow-polish/`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/italian-v2.1-flow-polish/`
- Create: `/Users/admin/Documents/Private Chess Courses/italian-white-flow-polish-audit.json`
- Create: `/Users/admin/Documents/Private Chess Courses/italian-white-flow-polish-edit-manifest.json`
- Create: `/private/tmp/italian-flow-audit.mjs`
- Create: `/private/tmp/italian-flow-apply.mjs`
- Create: `/private/tmp/italian-flow-assert.mjs`
- Create: `.superpowers/tmp/italian-import-polish-smoke.go`

---

### Task 1: Baseline checkpoint and teaching-flow audit

**Files:**
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.authoring.json`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/italian-v2-before-flow-polish/`
- Create: `/private/tmp/italian-flow-audit.mjs`
- Create: `/Users/admin/Documents/Private Chess Courses/italian-white-flow-polish-audit.json`

**Interfaces:**
- Consumes: validated Italian v2 private pack.
- Produces: recoverable baseline checkpoint and a private audit report with only IDs, counts, move UCI values, and warning categories.

- [ ] **Step 1: Confirm branch and public working tree state**

Run:

```bash
git status --short --branch
git branch --show-current
git log --oneline -5
```

Expected: branch is `codex/italian-white-teaching-flow-polish`; public working tree is clean except the committed design/plan work for this branch.

- [ ] **Step 2: Validate the current Italian v2 pack**

Run:

```bash
go run ./cmd/coursepack validate "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse" \
  > /private/tmp/italian-v2-before-flow-polish-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/italian-v2-before-flow-polish-validation.json
```

Expected:

```json
{
  "courseId": "mco15-italian-white",
  "contentVersion": "2.0.0",
  "missing": 0,
  "unexpected": 0
}
```

- [ ] **Step 3: Create the private baseline checkpoint**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/italian-v2-before-flow-polish"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse" \
  "$CHECKPOINT/mco15-italian-white.ctcourse"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.authoring.json" \
  "$CHECKPOINT/mco15-italian-white.authoring.json"
cp -p /private/tmp/italian-v2-before-flow-polish-validation.json "$CHECKPOINT/validation.json"
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  "$CHECKPOINT/validation.json" > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/mco15-italian-white.ctcourse" \
  "$CHECKPOINT/mco15-italian-white.authoring.json" \
  "$CHECKPOINT/validation.json" \
  "$CHECKPOINT/summary.json" > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
```

Expected: each checksum prints `OK`.

- [ ] **Step 4: Create the private teaching-flow audit script**

Create `/private/tmp/italian-flow-audit.mjs` with this exact content:

```js
import fs from 'node:fs'

const [packPath, outPath] = process.argv.slice(2)
if (!packPath || !outPath) {
  console.error('usage: node italian-flow-audit.mjs <course.ctcourse> <audit.json>')
  process.exit(2)
}

const pack = JSON.parse(fs.readFileSync(packPath, 'utf8'))
const moves = new Map((pack.moves ?? []).map((move) => [move.moveId, move]))
const prompts = new Map((pack.prompts ?? []).map((prompt) => [prompt.promptId, prompt]))
const textMatch = /source note [a-z]+ records|opening activity result|course move found/i

function requiredActivities(lesson) {
  return (lesson.activities ?? []).filter((activity) => activity.required !== false)
}

function promptMove(activity) {
  if (!activity.promptId) return null
  const prompt = prompts.get(activity.promptId)
  if (!prompt) return { activityId: activity.activityId, promptId: activity.promptId, missingPrompt: true }
  const primary = moves.get(prompt.primaryMoveId)
  return {
    activityId: activity.activityId,
    promptId: activity.promptId,
    positionId: prompt.positionId,
    primaryMoveId: prompt.primaryMoveId,
    primaryUci: primary?.uci ?? null,
    acceptedAlternativeUcis: (prompt.acceptedAlternativeMoveIds ?? [])
      .map((moveId) => moves.get(moveId)?.uci)
      .filter(Boolean)
  }
}

function groupedDuplicates(entries, key) {
  const groups = new Map()
  for (const entry of entries) {
    const value = key(entry)
    if (!value) continue
    groups.set(value, [...(groups.get(value) ?? []), entry])
  }
  return [...groups.entries()]
    .filter(([, values]) => values.length > 1)
    .map(([value, values]) => ({ value, activityIds: values.map((entry) => entry.activityId) }))
}

const tacticalMultiDecisionLessons = new Set(['giuoco-moller', 'two-knights-na5'])
const lessons = (pack.lessons ?? []).map((lesson) => {
  const required = requiredActivities(lesson)
  const decisions = required.filter((activity) => activity.kind === 'decision')
  const promptMoves = decisions.map(promptMove).filter(Boolean)
  const requiredText = required
    .flatMap((activity) => [activity.title, activity.instruction, ...(activity.referenceSections ?? []).flatMap((section) => [section.title, section.instruction])])
    .filter((value) => typeof value === 'string')
  const maxRequired = tacticalMultiDecisionLessons.has(lesson.lessonId) ? 5 : 4
  return {
    lessonId: lesson.lessonId,
    chapterId: lesson.chapterId,
    minimumDepth: lesson.minimumDepth,
    activityCount: (lesson.activities ?? []).length,
    requiredCount: required.length,
    decisionCount: decisions.length,
    promptMoves,
    flags: {
      tooManyRequiredActivities: required.length > maxRequired,
      tooManyDecisions: decisions.length > 1 && !tacticalMultiDecisionLessons.has(lesson.lessonId),
      duplicatePromptIds: groupedDuplicates(promptMoves, (entry) => entry.promptId),
      duplicatePrimaryUci: groupedDuplicates(promptMoves, (entry) => `${entry.positionId}:${entry.primaryUci}`),
      genericTextMatches: requiredText.some((value) => textMatch.test(value))
    }
  }
})

const summary = {
  courseId: pack.courseId,
  contentVersion: pack.contentVersion,
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
console.log(JSON.stringify({ courseId: summary.courseId, contentVersion: summary.contentVersion, blockerCount: summary.blockers.length, counts: summary.counts }, null, 2))
```

- [ ] **Step 5: Run the baseline audit**

Run:

```bash
node /private/tmp/italian-flow-audit.mjs \
  "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/italian-white-flow-polish-audit.json"
jq '{courseId,contentVersion,counts,blockers:[.blockers[] | {lessonId,flags}]}' \
  "/Users/admin/Documents/Private Chess Courses/italian-white-flow-polish-audit.json"
```

Expected: output is private and may list lessons to polish. It must not print source prose, rendered page text, or note text.

- [ ] **Step 6: Record the Task 1 private report**

Create `/Users/admin/Documents/Private Chess Courses/checkpoints/italian-v2-before-flow-polish/report.md` with:

```markdown
# Italian v2 teaching-flow baseline

- Baseline course ID: mco15-italian-white
- Baseline content version: 2.0.0
- Baseline validation: zero missing coverage, zero unexpected coverage
- Baseline audit: see /Users/admin/Documents/Private Chess Courses/italian-white-flow-polish-audit.json
- Privacy: no private course files are tracked by Git
```

Run:

```bash
git status --short
git ls-files '*.ctcourse'
```

Expected: private checkpoint and audit files are not tracked. Tracked `.ctcourse` files, if any, are only synthetic fixtures under `internal/openings/testdata/`.

---

### Task 2: Create a private flow-polish edit manifest and generic apply script

**Files:**
- Read: `/Users/admin/Documents/Private Chess Courses/italian-white-flow-polish-audit.json`
- Create: `/Users/admin/Documents/Private Chess Courses/italian-white-flow-polish-edit-manifest.json`
- Create: `/private/tmp/italian-flow-apply.mjs`

**Interfaces:**
- Consumes: audit report from Task 1 and the current private pack.
- Produces: private edit manifest and a generic script that applies manifest edits deterministically.

- [ ] **Step 1: Create the private edit manifest skeleton**

Create `/Users/admin/Documents/Private Chess Courses/italian-white-flow-polish-edit-manifest.json` with this structure, replacing every `lessonEdits`, `promptEdits`, and `edgeEdits` entry with final private edits before running Task 3:

```json
{
  "courseId": "mco15-italian-white",
  "sourceContentVersion": "2.0.0",
  "targetContentVersion": "2.1.0",
  "lessonEdits": [],
  "promptEdits": [],
  "edgeEdits": [],
  "activityRemovalJustifications": [],
  "depthPolicy": {
    "quick": "coherent beginner White repertoire with low repetition",
    "standard": "practical Italian map with major alternatives",
    "reference": "full existing analysis retained outside required flow"
  }
}
```

Manifest rules:

- Every changed learner-facing field must contain final private copy in this private manifest, not in public docs.
- `lessonEdits[].lessonId` must already exist in the pack.
- `lessonEdits[].activityEdits[].activityId` must already exist inside that lesson.
- `lessonEdits[].removeActivityIds` is allowed only when the removed activity duplicates the same instructional job and does not remove the only decision from a lesson.
- `lessonEdits[].activityOrder` must contain exactly the final activity IDs for the lesson after removals.
- `promptEdits` are allowed only when the existing prompt's instructional meaning remains compatible with the original stable prompt ID.
- `activityRemovalJustifications` must list a reason for every removed activity ID.

- [ ] **Step 2: Fill the manifest from the private audit**

Read:

```bash
jq '.blockers[] | {lessonId, flags}' \
  "/Users/admin/Documents/Private Chess Courses/italian-white-flow-polish-audit.json"
```

Edit `/Users/admin/Documents/Private Chess Courses/italian-white-flow-polish-edit-manifest.json` so every lesson flagged by the audit has an explicit `lessonEdits` entry or an explicit reason in `activityRemovalJustifications` stating that no edit is needed because the flagged decisions are distinct.

Use this exact edit shape:

```json
{
  "lessonId": "existing-lesson-id",
  "title": "final private lesson title when changed",
  "objectives": ["final private objective when changed"],
  "minimumDepth": "quick",
  "activityOrder": ["existing-activity-id-1", "existing-activity-id-2"],
  "removeActivityIds": [],
  "activityEdits": [
    {
      "activityId": "existing-activity-id-1",
      "kind": "concept",
      "title": "final private activity title when changed",
      "instruction": "final private activity instruction when changed",
      "required": true,
      "positionId": "existing-position-id",
      "noteIds": [],
      "moveIds": [],
      "promptId": "existing-prompt-id"
    }
  ]
}
```

Fields that do not change may be omitted from an activity edit. Do not create synthetic public examples from MCO text.

- [ ] **Step 3: Create the generic apply script**

Create `/private/tmp/italian-flow-apply.mjs` with this exact content:

```js
import fs from 'node:fs'

const [packPath, manifestPath, outputPath] = process.argv.slice(2)
if (!packPath || !manifestPath || !outputPath) {
  console.error('usage: node italian-flow-apply.mjs <input.ctcourse> <manifest.json> <output.ctcourse>')
  process.exit(2)
}

const pack = JSON.parse(fs.readFileSync(packPath, 'utf8'))
const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'))

function fail(message) {
  console.error(message)
  process.exit(1)
}

if (manifest.courseId !== pack.courseId) fail(`manifest courseId ${manifest.courseId} does not match ${pack.courseId}`)
if (manifest.sourceContentVersion !== pack.contentVersion) {
  fail(`manifest sourceContentVersion ${manifest.sourceContentVersion} does not match ${pack.contentVersion}`)
}
if (manifest.targetContentVersion !== '2.1.0') fail('targetContentVersion must be 2.1.0')

const lessons = new Map(pack.lessons.map((lesson) => [lesson.lessonId, lesson]))
const prompts = new Map(pack.prompts.map((prompt) => [prompt.promptId, prompt]))
const edges = new Map(pack.lessonEdges.map((edge) => [edge.edgeId, edge]))
const positions = new Set(pack.positions.map((position) => position.positionId))
const moves = new Set(pack.moves.map((move) => move.moveId))
const notes = new Set(pack.notes.map((note) => note.noteId))

const lessonFields = new Set(['title', 'objectives', 'minimumDepth', 'startPositionId'])
const activityFields = new Set(['kind', 'title', 'instruction', 'required', 'positionId', 'noteIds', 'moveIds', 'promptId'])
const promptFields = new Set(['positionId', 'primaryMoveId', 'acceptedAlternativeMoveIds'])
const edgeFields = new Set(['kind', 'label', 'minimumDepth', 'ordinal'])

function applyFields(target, edit, allowed, label) {
  for (const [key, value] of Object.entries(edit)) {
    if (allowed.has(key)) target[key] = value
    else if (!['lessonId', 'activityId', 'edgeId', 'promptId', 'activityEdits', 'activityOrder', 'removeActivityIds'].includes(key)) {
      fail(`${label} contains unsupported field ${key}`)
    }
  }
}

for (const edit of manifest.lessonEdits ?? []) {
  const lesson = lessons.get(edit.lessonId)
  if (!lesson) fail(`unknown lessonId ${edit.lessonId}`)
  applyFields(lesson, edit, lessonFields, `lesson ${edit.lessonId}`)

  const removals = new Set(edit.removeActivityIds ?? [])
  const activities = new Map(lesson.activities.map((activity) => [activity.activityId, activity]))
  for (const removed of removals) {
    if (!activities.has(removed)) fail(`cannot remove unknown activity ${removed} from lesson ${edit.lessonId}`)
  }

  for (const activityEdit of edit.activityEdits ?? []) {
    const activity = activities.get(activityEdit.activityId)
    if (!activity) fail(`unknown activityId ${activityEdit.activityId} in lesson ${edit.lessonId}`)
    applyFields(activity, activityEdit, activityFields, `activity ${activityEdit.activityId}`)
    if (activity.positionId && !positions.has(activity.positionId)) fail(`activity ${activity.activityId} has unknown positionId ${activity.positionId}`)
    if (activity.promptId && !prompts.has(activity.promptId)) fail(`activity ${activity.activityId} has unknown promptId ${activity.promptId}`)
    for (const noteId of activity.noteIds ?? []) if (!notes.has(noteId)) fail(`activity ${activity.activityId} has unknown noteId ${noteId}`)
    for (const moveId of activity.moveIds ?? []) if (!moves.has(moveId)) fail(`activity ${activity.activityId} has unknown moveId ${moveId}`)
  }

  let finalActivities = lesson.activities.filter((activity) => !removals.has(activity.activityId))
  if (edit.activityOrder) {
    const wanted = edit.activityOrder
    const finalIds = new Set(finalActivities.map((activity) => activity.activityId))
    if (wanted.length !== finalIds.size) fail(`activityOrder length mismatch in lesson ${edit.lessonId}`)
    for (const activityId of wanted) if (!finalIds.has(activityId)) fail(`activityOrder includes unknown or removed ${activityId}`)
    const byId = new Map(finalActivities.map((activity) => [activity.activityId, activity]))
    finalActivities = wanted.map((activityId) => byId.get(activityId))
  }
  if (!finalActivities.some((activity) => activity.required !== false)) fail(`lesson ${edit.lessonId} has no required activities`)
  if (!finalActivities.some((activity) => activity.kind === 'decision')) fail(`lesson ${edit.lessonId} has no decision activity`)
  lesson.activities = finalActivities
}

for (const edit of manifest.promptEdits ?? []) {
  const prompt = prompts.get(edit.promptId)
  if (!prompt) fail(`unknown promptId ${edit.promptId}`)
  applyFields(prompt, edit, promptFields, `prompt ${edit.promptId}`)
  if (!positions.has(prompt.positionId)) fail(`prompt ${edit.promptId} has unknown positionId ${prompt.positionId}`)
  if (!moves.has(prompt.primaryMoveId)) fail(`prompt ${edit.promptId} has unknown primaryMoveId ${prompt.primaryMoveId}`)
  for (const moveId of prompt.acceptedAlternativeMoveIds ?? []) {
    if (!moves.has(moveId)) fail(`prompt ${edit.promptId} has unknown acceptedAlternativeMoveId ${moveId}`)
  }
}

for (const edit of manifest.edgeEdits ?? []) {
  const edge = edges.get(edit.edgeId)
  if (!edge) fail(`unknown edgeId ${edit.edgeId}`)
  applyFields(edge, edit, edgeFields, `edge ${edit.edgeId}`)
}

pack.contentVersion = manifest.targetContentVersion
fs.writeFileSync(outputPath, `${JSON.stringify(pack, null, 2)}\n`)
console.log(JSON.stringify({ courseId: pack.courseId, contentVersion: pack.contentVersion, lessons: pack.lessons.length }, null, 2))
```

- [ ] **Step 4: Validate manifest JSON and script syntax**

Run:

```bash
jq empty "/Users/admin/Documents/Private Chess Courses/italian-white-flow-polish-edit-manifest.json"
node --check /private/tmp/italian-flow-apply.mjs
```

Expected: both commands exit `0`.

- [ ] **Step 5: Record the manifest coverage check**

Run:

```bash
jq '{
  courseId,
  sourceContentVersion,
  targetContentVersion,
  lessonEdits: (.lessonEdits | length),
  promptEdits: (.promptEdits | length),
  edgeEdits: (.edgeEdits | length),
  removalJustifications: (.activityRemovalJustifications | length)
}' "/Users/admin/Documents/Private Chess Courses/italian-white-flow-polish-edit-manifest.json"
```

Expected: `courseId` is `mco15-italian-white`, `sourceContentVersion` is `2.0.0`, `targetContentVersion` is `2.1.0`, and every audit blocker has either a lesson edit or a recorded private justification.

---

### Task 3: Apply private course edits and validate the polished pack

**Files:**
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/italian-white-flow-polish-edit-manifest.json`
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse`
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.authoring.json`
- Create: `/private/tmp/italian-v2.1-flow-polish-validation.json`

**Interfaces:**
- Consumes: manifest and apply script from Task 2.
- Produces: validated private Italian v2.1 course pack.

- [ ] **Step 1: Apply the private manifest into a staged course file**

Run:

```bash
node /private/tmp/italian-flow-apply.mjs \
  "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/italian-white-flow-polish-edit-manifest.json" \
  "/private/tmp/mco15-italian-white.v2.1.ctcourse"
```

Expected: output includes `"courseId": "mco15-italian-white"` and `"contentVersion": "2.1.0"`.

- [ ] **Step 2: Validate the staged polished pack**

Run:

```bash
go run ./cmd/coursepack validate "/private/tmp/mco15-italian-white.v2.1.ctcourse" \
  > /private/tmp/italian-v2.1-flow-polish-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/italian-v2.1-flow-polish-validation.json
```

Expected:

```json
{
  "courseId": "mco15-italian-white",
  "contentVersion": "2.1.0",
  "missing": 0,
  "unexpected": 0
}
```

- [ ] **Step 3: Replace the private active course file**

Run only after validation passes:

```bash
cp -p "/private/tmp/mco15-italian-white.v2.1.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse"
```

Expected: the private course pack is replaced locally; no public Git file changes.

- [ ] **Step 4: Mirror stable polish metadata into the private authoring JSON**

Update `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.authoring.json` so it records:

```json
{
  "flowPolish": {
    "sourceContentVersion": "2.0.0",
    "targetContentVersion": "2.1.0",
    "auditPath": "/Users/admin/Documents/Private Chess Courses/italian-white-flow-polish-audit.json",
    "manifestPath": "/Users/admin/Documents/Private Chess Courses/italian-white-flow-polish-edit-manifest.json",
    "principle": "one lesson, one job; repeated required moves become demonstrations, comparisons, recaps, or optional Reference"
  }
}
```

If the authoring JSON already has a `flowPolish` object, replace it with this object and preserve every unrelated top-level field.

- [ ] **Step 5: Create the polished private checkpoint**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/italian-v2.1-flow-polish"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse" \
  "$CHECKPOINT/mco15-italian-white.ctcourse"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.authoring.json" \
  "$CHECKPOINT/mco15-italian-white.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/italian-white-flow-polish-audit.json" \
  "$CHECKPOINT/baseline-audit.json"
cp -p "/Users/admin/Documents/Private Chess Courses/italian-white-flow-polish-edit-manifest.json" \
  "$CHECKPOINT/edit-manifest.json"
cp -p /private/tmp/italian-v2.1-flow-polish-validation.json "$CHECKPOINT/validation.json"
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  "$CHECKPOINT/validation.json" > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/"*.json "$CHECKPOINT/"*.ctcourse > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
```

Expected: every checksum prints `OK`.

---

### Task 4: Run private lesson-quality hardening checks

**Files:**
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse`
- Read: `/private/tmp/italian-flow-audit.mjs`
- Create: `/private/tmp/italian-flow-assert.mjs`
- Create: `/private/tmp/italian-v2.1-flow-polish-audit.json`

**Interfaces:**
- Consumes: polished private course from Task 3.
- Produces: hard failure for repeated required decisions, warning spam, and oversized required lesson flow.

- [ ] **Step 1: Re-run the audit on the polished pack**

Run:

```bash
node /private/tmp/italian-flow-audit.mjs \
  "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse" \
  /private/tmp/italian-v2.1-flow-polish-audit.json
jq '{courseId,contentVersion,counts,blockerCount:(.blockers|length)}' \
  /private/tmp/italian-v2.1-flow-polish-audit.json
```

Expected: `contentVersion` is `2.1.0`.

- [ ] **Step 2: Create the assertion script**

Create `/private/tmp/italian-flow-assert.mjs` with this exact content:

```js
import fs from 'node:fs'

const [auditPath] = process.argv.slice(2)
if (!auditPath) {
  console.error('usage: node italian-flow-assert.mjs <audit.json>')
  process.exit(2)
}

const audit = JSON.parse(fs.readFileSync(auditPath, 'utf8'))
const allowedMultiDecisionLessons = new Set(['giuoco-moller', 'two-knights-na5'])
const failures = []

if (audit.courseId !== 'mco15-italian-white') failures.push(`courseId ${audit.courseId}`)
if (audit.contentVersion !== '2.1.0') failures.push(`contentVersion ${audit.contentVersion}`)
if (audit.counts.lessons < 20 || audit.counts.lessons > 23) failures.push(`lesson count ${audit.counts.lessons}`)

for (const lesson of audit.lessons) {
  const maxRequired = allowedMultiDecisionLessons.has(lesson.lessonId) ? 5 : 4
  if (lesson.requiredCount > maxRequired) failures.push(`${lesson.lessonId}: requiredCount ${lesson.requiredCount}`)
  if (lesson.decisionCount > 1 && !allowedMultiDecisionLessons.has(lesson.lessonId)) {
    failures.push(`${lesson.lessonId}: decisionCount ${lesson.decisionCount}`)
  }
  if (lesson.flags.duplicatePromptIds.length > 0) failures.push(`${lesson.lessonId}: duplicate prompt IDs`)
  if (lesson.flags.duplicatePrimaryUci.length > 0) failures.push(`${lesson.lessonId}: duplicate primary UCI from same position`)
  if (lesson.flags.genericTextMatches) failures.push(`${lesson.lessonId}: generic warning/spam text`)
}

if (failures.length > 0) {
  console.error(JSON.stringify({ failures }, null, 2))
  process.exit(1)
}

console.log(JSON.stringify({
  ok: true,
  courseId: audit.courseId,
  contentVersion: audit.contentVersion,
  counts: audit.counts
}, null, 2))
```

- [ ] **Step 3: Run hardening assertions**

Run:

```bash
node --check /private/tmp/italian-flow-assert.mjs
node /private/tmp/italian-flow-assert.mjs /private/tmp/italian-v2.1-flow-polish-audit.json
```

Expected: output contains `"ok": true`.

- [ ] **Step 4: Manually inspect learner flow without quoting source text**

Run:

```bash
jq '.lessons[] | {
  lessonId,
  chapterId,
  minimumDepth,
  required: ([.activities[] | select(.required != false)] | length),
  decisions: ([.activities[] | select(.required != false and .kind == "decision")] | length),
  activityIds: [.activities[].activityId]
}' "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse"
```

Expected:

- Quick lessons have 2-3 required ideas unless one distinct recap/demonstration makes 4 useful.
- Standard lessons have 3-4 required ideas unless the lesson is a tactical allowed multi-decision branch.
- Dense source details remain available through notes, optional activities, or the variation explorer.
- The output contains IDs and counts only; do not paste private source prose into public reports.

- [ ] **Step 5: Record the private hardening report**

Create `/Users/admin/Documents/Private Chess Courses/checkpoints/italian-v2.1-flow-polish/hardening-report.md` with:

```markdown
# Italian v2.1 flow-polish hardening

- Course ID: mco15-italian-white
- Content version: 2.1.0
- Audit path: /private/tmp/italian-v2.1-flow-polish-audit.json
- Assertions: passed
- Manual learner-flow inspection: completed without public source prose
- Privacy: private course files remain outside Git
```

Run:

```bash
git status --short
```

Expected: only public plan/spec files are tracked or modified; private checkpoint/report files do not appear.

---

### Task 5: Import polished Italian and verify app-level course behavior

**Files:**
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse`
- Create: `.superpowers/tmp/italian-import-polish-smoke.go`
- Create: `/private/tmp/italian-import-polish-smoke.json`

**Interfaces:**
- Consumes: validated Italian v2.1 pack.
- Produces: active local catalogue with Italian v2.1 while Ruy and Caro remain active.

- [ ] **Step 1: Validate all three private course packs**

Run:

```bash
for pack in \
  "/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse" \
  "/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse"
do
  go run ./cmd/coursepack validate "$pack" \
    | jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}'
done
```

Expected: Italian is `2.1.0`; Ruy and Caro remain valid; every printed validation object has `missing: 0` and `unexpected: 0`.

- [ ] **Step 2: Create a production-path import smoke harness**

Create `.superpowers/tmp/italian-import-polish-smoke.go` with a small Go program that imports course packs through `app.Open` and `services.CourseImporter.Inspect/Import`, then emits JSON with active course summaries.

Use this exact code; do not add the file to Git:

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
	if os.Getenv("ITALIAN_FLOW_USE_DEFAULT_DATA") == "1" {
		var err error
		paths, err = storage.DefaultPaths()
		if err != nil {
			panic(err)
		}
	} else {
		dataRoot := os.Getenv("ITALIAN_FLOW_DATA_ROOT")
		if dataRoot == "" {
			fmt.Fprintln(os.Stderr, "ITALIAN_FLOW_DATA_ROOT or ITALIAN_FLOW_USE_DEFAULT_DATA=1 is required")
			os.Exit(2)
		}
		paths = storage.PathsAt(dataRoot)
	}
	coursePaths := strings.Split(os.Getenv("ITALIAN_FLOW_COURSES"), "|")
	if len(coursePaths) == 0 || coursePaths[0] == "" {
		fmt.Fprintln(os.Stderr, "ITALIAN_FLOW_COURSES is required")
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

- [ ] **Step 3: Run the import smoke against a disposable data root**

Run:

```bash
mkdir -p .superpowers/tmp
ITALIAN_FLOW_ROOT="$(mktemp -d)"
ITALIAN_FLOW_DATA_ROOT="$ITALIAN_FLOW_ROOT" \
ITALIAN_FLOW_COURSES="/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse|/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse|/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse" \
go run .superpowers/tmp/italian-import-polish-smoke.go \
  > /private/tmp/italian-import-polish-smoke.json
jq '{
  imports: [.imports[] | {sourceId:.inspection.sourceId, accepted:.report.accepted}],
  activeVersions,
  courses: [.home.courses[] | {courseId,title,perspective,completedLessons,totalLessons,depth}]
}' /private/tmp/italian-import-polish-smoke.json
```

Expected: imports include `mco15-italian-white`, `mco15-ruy-lopez-white`, and `mco15-caro-kann-black`; every `accepted` value is `1`; disposable home includes all three courses.

- [ ] **Step 4: Import the polished pack into the default local catalogue**

Before importing into default app data, close any running Chess Trainer app instance.

Run:

```bash
ITALIAN_FLOW_USE_DEFAULT_DATA=1 \
ITALIAN_FLOW_COURSES="/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse" \
go run .superpowers/tmp/italian-import-polish-smoke.go \
  > /private/tmp/italian-default-import-polish.json
jq '{
  imports: [.imports[] | {sourceId:.inspection.sourceId, sourceName:.inspection.sourceName, accepted:.report.accepted}],
  activeVersions,
  courses: [.home.courses[] | {courseId,title,perspective,totalLessons,depth}]
}' /private/tmp/italian-default-import-polish.json
```

Expected after import: `.activeVersions["mco15-italian-white"]` is `2.1.0`; Ruy Lopez and Caro-Kann remain active and unchanged.

- [ ] **Step 5: Run UI acceptance for Italian flow**

Use the app or Wails dev browser to verify:

- Learn Openings lists Italian, Ruy Lopez, and Caro-Kann.
- Select Italian.
- Quick depth renders a coherent beginner tree.
- Standard depth renders Giuoco, Evans, and Two Knights practical branches.
- Reference depth exposes the rich Italian tree and variation explorer.
- Continue Learning opens the current Italian lesson.
- Complete or pause/resume one foundations-to-Giuoco path without returning to Home.
- Open the variation explorer and navigate at least ten Italian branch clicks; the board remains fully inside the viewport.

Record only IDs, counts, and UI-state facts in `/Users/admin/Documents/Private Chess Courses/checkpoints/italian-v2.1-flow-polish/ui-acceptance.md`.

---

### Task 6: Rebuild, verify, review, and prepare merge

**Files:**
- Read/verify: repository public diff
- Read/verify: `/Users/admin/Documents/Private Chess Courses/checkpoints/italian-v2.1-flow-polish/`
- Build: `build/bin/Chess Trainer.app`

**Interfaces:**
- Consumes: imported polished Italian course and any public generic fixes.
- Produces: rebuilt app and merge-ready branch with private files excluded.

- [ ] **Step 1: Run public repository verification**

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
  \( -name '*italian*' -o -name '*flow-polish*' \) | sort
```

Expected:

- public Git diff contains only the plan/spec and any generic app/test fixes;
- no private course file appears in `git status`;
- tracked `.ctcourse` files are only synthetic fixtures under `internal/openings/testdata/`;
- private checkpoint and course files remain under `/Users/admin/Documents/Private Chess Courses/`.

- [ ] **Step 5: Request strict review if public code changed**

If public app/test code changed, run the thermo-nuclear review workflow against the public diff. The review must find:

- no course-specific special cases;
- no private source prose;
- no large-file or spaghetti regression;
- generic synthetic tests for any app behavior change.

If only docs changed publicly and all course edits are private, record that no public code review was needed beyond privacy and verification checks.

- [ ] **Step 6: Commit public changes**

Commit only public files:

```bash
git add docs/superpowers/specs/2026-07-23-italian-white-teaching-flow-polish-design.md \
  docs/superpowers/plans/2026-07-23-italian-white-teaching-flow-polish.md
git status --short
git commit -m "docs: plan italian white teaching flow polish"
```

If generic public fixes were required, commit those in separate focused commits before the plan/docs commit, with messages that describe the generic behavior rather than Italian-specific content.

Expected: private files are not staged.

- [ ] **Step 7: Prepare final handoff**

Report:

- final Italian private course version and validation counts;
- checkpoint path;
- import status;
- app rebuild status;
- exact verification commands that passed;
- public commit list;
- confirmation that private course files were not committed.

Then offer the standard finishing options for the branch.
