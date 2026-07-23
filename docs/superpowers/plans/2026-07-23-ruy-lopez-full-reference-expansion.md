# Ruy Lopez Full-Reference Expansion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deepen the private `mco15-ruy-lopez-white` course from a v1 seed pack into a validated v2 full-reference Ruy Lopez pack based on MCO-15 printed pages 42-95.

**Architecture:** Reuse the existing schema-v2 opening course model, `coursepack` validator, SAN-line authoring helper, importer, catalogue replacement flow, teaching-tree UI, variation explorer, and journey rebase behavior. Keep the public repository limited to generic specs, plans, tooling, tests, and app fixes; author all MCO-derived moves, notes, references, and course data privately under `/Users/admin/Documents/Private Chess Courses/`.

**Tech Stack:** Go 1.26.4, `github.com/corentings/chess/v2`, JSON `.ctcourse`, private authoring JSON, Poppler `pdftoppm`, `jq`, Wails 2.12.0, Svelte, Vitest, Playwright, macOS app codesigning.

## Global Constraints

- Course ID remains exactly `mco15-ruy-lopez-white`.
- `contentVersion` must become exactly `2.0.0`.
- Course perspective remains exactly `white`.
- Preserve existing lesson, prompt, position, move, activity, and edge IDs where their meaning remains the same.
- White moves are `repertoire` or `alternative`; Black moves are `opponent`.
- Decision prompts occur only where White is to move.
- Printed-page source scope is Ruy Lopez pages 42-95.
- The expected scan offset for this PDF is printed page 42 at PDF page 59 and printed page 95 at PDF page 112; verify this from rendered images before authoring.
- Keep Quick and Standard concise; put dense source detail into Reference.
- Implement option A as one complete full-reference pass; internal page-band checkpoints are for recovery only and must not be activated as partial courses.
- Do not add new openings in this pass.
- Do not deepen Italian or Caro-Kann in this pass.
- Do not special-case Ruy Lopez in public app code.
- Do not commit private `.ctcourse` files, private authoring inventories, private checkpoint files, or MCO-derived prose to Git.
- Public reports and commits may contain only counts, file paths, validation results, and high-level summaries.

---

## File Map

Repository files:

- Modify: `docs/superpowers/specs/2026-07-23-ruy-lopez-full-reference-expansion-design.md` to align stable Ruy v1 lesson IDs with the real seed pack.
- Create: `docs/superpowers/plans/2026-07-23-ruy-lopez-full-reference-expansion.md`.
- Modify public Go/Svelte files only if a generic defect is found, and only after a failing synthetic test exists.

Private files outside Git:

- Read/update: `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.authoring.json`.
- Read/update: `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse`.
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/ruy-lopez-white-v1-before-full-reference/`.
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/ruy-lopez-white-v2-pages-42-55/`.
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/ruy-lopez-white-v2-pages-56-68/`.
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/ruy-lopez-white-v2-pages-69-82/`.
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/ruy-lopez-white-v2-pages-83-95/`.
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/ruy-lopez-white-v2-full-reference/`.
- Create: `/private/tmp/mco15-ruy-lopez-pages/` rendered PNG source pages.
- Create: `/private/tmp/ruy-lopez-validation.json` validation output.

---

### Task 1: Prepare workspace, render source pages, and checkpoint v1

**Files:**
- Read: `/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.authoring.json`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse`
- Create: `/private/tmp/mco15-ruy-lopez-pages/*.png`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/ruy-lopez-white-v1-before-full-reference/`

**Interfaces:**
- Consumes: current validated v1 Ruy Lopez seed pack.
- Produces: recoverable v1 checkpoint and verified rendered source-page images for manual private authoring.

- [ ] **Step 1: Confirm the implementation branch is clean**

Run:

```bash
git status --short --branch
git log --oneline --decorate -5
```

Expected: clean checkout on `codex/ruy-lopez-full-reference-expansion`, rooted after the Caro-Kann full-reference branch commits.

- [ ] **Step 2: Validate the current private v1 pack**

Run:

```bash
go run ./cmd/coursepack validate "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse" \
  > /private/tmp/ruy-lopez-v1-before-full-reference-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/ruy-lopez-v1-before-full-reference-validation.json
```

Expected: `courseId` is `mco15-ruy-lopez-white`, `contentVersion` is `1.0.0`, and both `missing` and `unexpected` are `0`.

- [ ] **Step 3: Create the v1 private checkpoint**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/ruy-lopez-white-v1-before-full-reference"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.authoring.json" \
  "$CHECKPOINT/mco15-ruy-lopez-white.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse" \
  "$CHECKPOINT/mco15-ruy-lopez-white.ctcourse"
cp -p /private/tmp/ruy-lopez-v1-before-full-reference-validation.json \
  "$CHECKPOINT/validation.json"
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/ruy-lopez-v1-before-full-reference-validation.json > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/"* > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
```

Expected: every checksum prints `OK`.

- [ ] **Step 4: Render Ruy Lopez source pages from the local PDF**

Run:

```bash
mkdir -p /private/tmp/mco15-ruy-lopez-pages
pdftoppm -png -r 180 \
  -f 59 -l 112 \
  "/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf" \
  /private/tmp/mco15-ruy-lopez-pages/ruy
ls /private/tmp/mco15-ruy-lopez-pages/ruy-*.png | wc -l
```

Expected: the count is `54`.

- [ ] **Step 5: Verify the rendered page offset visually**

Open the first and last rendered pages:

```bash
open /private/tmp/mco15-ruy-lopez-pages/ruy-059.png
open /private/tmp/mco15-ruy-lopez-pages/ruy-112.png
```

Expected: `ruy-059.png` shows printed page `42`, and `ruy-112.png` shows printed page `95`. If either page does not match, stop authoring and correct the PDF-page range before proceeding.

- [ ] **Step 6: Verify private files remain untracked**

Run:

```bash
git status --short
git ls-files '*.ctcourse'
```

Expected: private files under `/Users/admin/Documents/Private Chess Courses/` are not listed. Tracked `.ctcourse` files are only synthetic fixtures under `internal/openings/testdata/`.

---

### Task 2: Verify reusable authoring helper and expand the private source inventory

**Files:**
- Read: `cmd/coursepack/main.go`
- Read: `cmd/coursepack/main_test.go`
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.authoring.json`
- Read: `/private/tmp/mco15-ruy-lopez-pages/*.png`

**Interfaces:**
- Consumes: rendered source pages and existing `coursepack sanline` helper.
- Produces: private page-by-page source inventory with stable teaching-node and source-coverage intent.

- [ ] **Step 1: Verify `coursepack sanline` converts from initial position**

Run:

```bash
go run ./cmd/coursepack sanline \
  --fen "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1" \
  e4 e5 Nf3 Nc6 Bb5 \
  | jq '{startFen, finalFen, uci:[.moves[].uci]}'
```

Expected:

```json
{
  "uci": ["e2e4", "e7e5", "g1f3", "b8c6", "f1b5"]
}
```

`startFen` and `finalFen` must be non-empty strings.

- [ ] **Step 2: Replace the private authoring page metadata**

In `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.authoring.json`, set `renderedPdfPages` to:

```json
{
  "printedPages": [42,43,44,45,46,47,48,49,50,51,52,53,54,55,56,57,58,59,60,61,62,63,64,65,66,67,68,69,70,71,72,73,74,75,76,77,78,79,80,81,82,83,84,85,86,87,88,89,90,91,92,93,94,95],
  "pdfPages": [59,60,61,62,63,64,65,66,67,68,69,70,71,72,73,74,75,76,77,78,79,80,81,82,83,84,85,86,87,88,89,90,91,92,93,94,95,96,97,98,99,100,101,102,103,104,105,106,107,108,109,110,111,112],
  "offsetVerified": true
}
```

Expected: this records the verified scan offset without storing source prose in the repository.

- [ ] **Step 3: Preserve the stable v1 teaching nodes**

Keep these existing teaching-node IDs in `teachingNodes`:

```text
ruy-foundations
ruy-black-third-move
ruy-morphy-a6
ruy-preserve-bishop
ruy-castle-re1
ruy-central-plan
ruy-exchange
ruy-open
ruy-berlin
ruy-steinitz
ruy-closed-main
ruy-anti-marshall
ruy-marshall-warning
ruy-delayed-systems
ruy-closed-systems
ruy-side-systems
```

Expected: existing learner progress anchors remain available for rebase.

- [ ] **Step 4: Add full-reference teaching nodes using stable family IDs**

Add Reference-depth nodes when the page inventory needs distinct homes for dense material. Use these IDs for the known Ruy v2 families:

```text
ruy-early-systems-reference
ruy-exchange-reference
ruy-open-reference
ruy-berlin-reference
ruy-steinitz-reference
ruy-closed-chigorin-reference
ruy-closed-breyer-reference
ruy-closed-zaitsev-reference
ruy-closed-smyslov-reference
ruy-closed-rare-reference
ruy-marshall-reference
ruy-anti-marshall-reference
ruy-side-systems-reference
```

For each new node, set:

```json
{
  "lessonId": "stable-id-from-list",
  "chapterId": "reference",
  "ordinal": 100,
  "depth": "reference",
  "purpose": "Original one-sentence study purpose for this source family."
}
```

Use unique ordinals in final tree order. The `purpose` text must be original teaching copy, not transcribed source prose.

- [ ] **Step 5: Add page-by-page private source inventory records**

Expand `paths` so every legible table column, note-letter branch, source family, and illustrative game reference chosen for the full-reference scope has one private inventory object with this shape:

```json
{
  "id": "ruy-p42-family-ordinal",
  "chapter": "ruy",
  "column": "private table or note coordinate",
  "printedPage": 42,
  "depth": "reference",
  "variation": "short private variation label",
  "evaluation": "equal",
  "purpose": "Original private authoring purpose."
}
```

Use `depth: "quick"` only for the compact learner spine, `depth: "standard"` for practical repertoire branches, and `depth: "reference"` for dense source coverage. The private inventory may include source move notation and citations in additional private fields, but those fields must remain outside Git.

- [ ] **Step 6: Validate the private authoring inventory shape**

Run:

```bash
jq -e '
  (.renderedPdfPages.offsetVerified == true) and
  (.renderedPdfPages.printedPages | length == 54) and
  (.renderedPdfPages.pdfPages | length == 54) and
  (.teachingNodes | length >= 16) and
  (.teachingEdges | length >= 15) and
  (.paths | length > 6) and
  (([.teachingNodes[].lessonId] | unique | length) == (.teachingNodes | length)) and
  (([.teachingEdges[].edgeId] | unique | length) == (.teachingEdges | length)) and
  (([.paths[].id] | unique | length) == (.paths | length))
' "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.authoring.json"
```

Expected: command exits `0`.

- [ ] **Step 7: Create private page-band inventory checkpoints**

After each page band is inventoried, create a checkpoint:

```bash
for checkpoint in \
  ruy-lopez-white-v2-pages-42-55 \
  ruy-lopez-white-v2-pages-56-68 \
  ruy-lopez-white-v2-pages-69-82 \
  ruy-lopez-white-v2-pages-83-95
do
  CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/$checkpoint"
  mkdir -p "$CHECKPOINT"
  cp -p "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.authoring.json" \
    "$CHECKPOINT/mco15-ruy-lopez-white.authoring.json"
  shasum -a 256 "$CHECKPOINT/"* > "$CHECKPOINT/SHA256SUMS.txt"
done
```

Expected: each checkpoint contains the private authoring file and checksums. These checkpoints are not imported into the app.

---

### Task 3: Build the Ruy v2 private `.ctcourse` root, tree, and learner spine

**Files:**
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.authoring.json`

**Interfaces:**
- Consumes: private inventory and existing v1 course pack.
- Produces: schema-v2 Ruy pack root metadata, stable lessons, teaching tree, and concise Quick / Standard learner spine.

- [ ] **Step 1: Update required root metadata**

In `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse`, set:

```json
{
  "schemaVersion": 2,
  "courseId": "mco15-ruy-lopez-white",
  "contentVersion": "2.0.0",
  "title": "Ruy Lopez for White",
  "perspective": "white",
  "defaultDepth": "reference"
}
```

Expected: the course remains the same import identity and becomes generation v2.

- [ ] **Step 2: Set the printed-page source coverage scope**

Set `sourceCoverage.printedPages` to:

```json
[42,43,44,45,46,47,48,49,50,51,52,53,54,55,56,57,58,59,60,61,62,63,64,65,66,67,68,69,70,71,72,73,74,75,76,77,78,79,80,81,82,83,84,85,86,87,88,89,90,91,92,93,94,95]
```

Populate `sourceCoverage.expectedReferences` with exactly the coverage IDs represented by authored moves and notes. Do not include illegible or intentionally omitted source items in `expectedReferences`.

- [ ] **Step 3: Preserve stable v1 lessons in the `.ctcourse`**

Run:

```bash
jq -e '
  [.lessons[].lessonId] as $ids |
  [
    "ruy-foundations",
    "ruy-black-third-move",
    "ruy-morphy-a6",
    "ruy-preserve-bishop",
    "ruy-castle-re1",
    "ruy-central-plan",
    "ruy-exchange",
    "ruy-open",
    "ruy-berlin",
    "ruy-steinitz",
    "ruy-closed-main",
    "ruy-anti-marshall",
    "ruy-marshall-warning",
    "ruy-delayed-systems",
    "ruy-closed-systems",
    "ruy-side-systems"
  ] | all(. as $id | $ids | index($id))
' "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse"
```

Expected: command exits `0`.

- [ ] **Step 4: Keep Quick required activities concise**

For Quick lessons, keep required activities to a maximum of four per lesson and use this shape:

```json
[
  {
    "kind": "concept",
    "required": true
  },
  {
    "kind": "demonstration",
    "required": true
  },
  {
    "kind": "decision",
    "required": true
  },
  {
    "kind": "recap",
    "required": true
  }
]
```

Omit the demonstration where it repeats the same move already taught by the decision. Required activity instructions must be original teaching copy and must not use generic filler such as source-note warnings.

- [ ] **Step 5: Make Standard practical rather than exhaustive**

For Standard lessons, include required activities only for practical White decisions and major Black-system recognition. Put additional table branches into optional `reference` activities with `required: false`.

Expected Standard coverage: Exchange, Open, Berlin, Steinitz/deferred systems, main Closed structure, Anti-Marshall, and sharper side-system recognition.

- [ ] **Step 6: Validate the learner-spine shape**

Run:

```bash
jq -e '
  ([.lessons[] | select(.minimumDepth == "quick") | .activities | map(select(.required)) | length] | all(. <= 4)) and
  ([.lessons[].activities[] | select(.kind == "reference" and .required == true)] | length == 0)
' "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse"
```

Expected: command exits `0`.

---

### Task 4: Author full Reference move graph, notes, prompts, and activities

**Files:**
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse`
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.authoring.json`
- Read: `/private/tmp/mco15-ruy-lopez-pages/*.png`

**Interfaces:**
- Consumes: private source inventory and Ruy v2 course skeleton.
- Produces: full-reference private move graph, notes, prompts, optional reference activities, and strict source coverage.

- [ ] **Step 1: Convert private SAN lines to UCI and FENs**

First verify the helper shape with a public-safe Ruy foundation smoke test:

```bash
go run ./cmd/coursepack sanline \
  --fen "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1" \
  e4 e5 Nf3 Nc6 Bb5 \
  > /private/tmp/ruy-lopez-sanline.json
jq '{uci:[.moves[].uci], finalFen}' /private/tmp/ruy-lopez-sanline.json
```

Expected: the helper exits `0`, the UCI array is `["e2e4","e7e5","g1f3","b8c6","f1b5"]`, and `finalFen` is non-empty.

For each private source branch, run the same command shape from the branch's current private starting FEN and private SAN sequence recorded in `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.authoring.json`. Those private command arguments may contain source moves in the local terminal, but do not paste those commands into public commits or final reports.

- [ ] **Step 2: Add positions and moves with stable IDs**

For each authored source branch:

1. Reuse an existing `positionId` when the FEN already exists.
2. Add a new `positionId` only when the FEN is new.
3. Reuse an existing `moveId` when the same from-position and UCI edge already exists.
4. Add a new `moveId` when the edge is new.
5. Set `trainingRole` to `repertoire` or `alternative` for White moves, and `opponent` for Black moves.
6. Set `minimumDepth` to the shallowest depth where the move is visible.
7. Attach `sourceRef.coverageId` to every source-backed move.

Expected: the graph remains connected from `rootPositionId`, and transpositions share destination positions instead of duplicating equivalent FEN nodes.

- [ ] **Step 3: Add source-backed notes without lesson spam**

For each source-backed note:

1. Use original teaching prose for learner-facing concept, plan, warning, recap, and explanation notes.
2. Use private source coordinates and citations in `sourceRef`.
3. Add note-letter analysis as `kind: "evaluation"`, `kind: "transposition"`, or `kind: "illustrative_game"` when that is the best reader-facing category.
4. Keep dense notes attached to optional Reference activities or variation explorer moves.

Expected: required activities do not display repeated boilerplate such as generic source-note records.

- [ ] **Step 4: Add prompts only for White-to-move course decisions**

For each prompt:

```json
{
  "promptId": "stable-prompt-id",
  "positionId": "white-to-move-position-id",
  "primaryMoveId": "white-repertoire-or-alternative-move-id",
  "acceptedAlternativeMoveIds": []
}
```

Expected: prompt positions are White to move, and primary moves are White `repertoire` or `alternative` moves from the same position.

- [ ] **Step 5: Add Reference activities as optional dense study**

For every dense source family, add optional Reference activities:

```json
{
  "activityId": "stable-reference-activity-id",
  "kind": "reference",
  "title": "Original short title",
  "instruction": "Original concise guidance for what this branch teaches.",
  "required": false,
  "positionId": "branch-start-position-id",
  "noteIds": ["source-backed-note-id"],
  "moveIds": ["connected-move-id"]
}
```

Expected: the lesson screen keeps a short required path while the variation explorer and optional Reference sections expose deep source coverage.

- [ ] **Step 6: Validate after each internal page band**

After authoring each page band, run:

```bash
go run ./cmd/coursepack validate "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse" \
  > /private/tmp/ruy-lopez-validation.json
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/ruy-lopez-validation.json
```

Expected: validation exits `0` for structurally complete intermediate graph states. `missing` and `unexpected` may be nonzero while the full source inventory is still being reconciled, but they must trend toward `0` and must be `0` before final import.

- [ ] **Step 7: Save recoverable page-band checkpoints**

After each successful internal band validation, run this checkpoint helper with the completed band name:

```bash
save_ruy_band_checkpoint() {
  case "$1" in
    ruy-lopez-white-v2-pages-42-55|ruy-lopez-white-v2-pages-56-68|ruy-lopez-white-v2-pages-69-82|ruy-lopez-white-v2-pages-83-95) ;;
    *) return 64 ;;
  esac
  CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/$1"
  mkdir -p "$CHECKPOINT"
  cp -p "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.authoring.json" \
    "$CHECKPOINT/mco15-ruy-lopez-white.authoring.json"
  cp -p "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse" \
    "$CHECKPOINT/mco15-ruy-lopez-white.ctcourse"
  cp -p /private/tmp/ruy-lopez-validation.json "$CHECKPOINT/validation.json"
  jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
    /private/tmp/ruy-lopez-validation.json > "$CHECKPOINT/summary.json"
  shasum -a 256 "$CHECKPOINT/"* > "$CHECKPOINT/SHA256SUMS.txt"
}
```

Use these exact checkpoint names:

```text
ruy-lopez-white-v2-pages-42-55
ruy-lopez-white-v2-pages-56-68
ruy-lopez-white-v2-pages-69-82
ruy-lopez-white-v2-pages-83-95
```

Expected: every checkpoint is private, recoverable, and excluded from Git.

---

### Task 5: Run Ruy lesson-quality and coverage hardening checks

**Files:**
- Read/update: `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse`
- Read/update: `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.authoring.json`

**Interfaces:**
- Consumes: authored full-reference Ruy v2 pack.
- Produces: non-repetitive lessons, clean source coverage, and private final checkpoint material.

- [ ] **Step 1: Reject generic warning spam in required learner text**

Run:

```bash
if jq -r '
  .lessons[].activities[] | select(.required) |
  (.instruction // empty),
  (.title // empty)
' "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse" |
rg -n 'Source note [a-z] records|records a concrete analytical subvariation|Course move found|opening activity result'; then
  exit 1
fi
```

Expected: command exits `0` with no matches.

- [ ] **Step 2: Reject generic warning spam in required notes**

Run:

```bash
if jq -r '
  ([.lessons[].activities[] | select(.required) | .noteIds[]?] | unique) as $ids |
  .notes[] | select(.noteId as $id | $ids | index($id)) |
  .text
' "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse" |
rg -n 'Source note [a-z] records|records a concrete analytical subvariation|Course move found|opening activity result'; then
  exit 1
fi
```

Expected: command exits `0` with no matches.

- [ ] **Step 3: Enforce short required paths**

Run:

```bash
jq -e '
  [.lessons[] | {
    lessonId,
    required: ([.activities[] | select(.required)] | length)
  } | select(.required > 4)] | length == 0
' "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse"
```

Expected: command exits `0`.

- [ ] **Step 4: Enforce optional Reference activities**

Run:

```bash
jq -e '
  [.lessons[].activities[] | select(.kind == "reference" and .required == true)] | length == 0
' "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse"
```

Expected: command exits `0`.

- [ ] **Step 5: Inspect decision prompt uniqueness**

Run:

```bash
jq -r '
  .lessons[] as $lesson |
  $lesson.activities[] |
  select(.kind == "decision" and .required) |
  [$lesson.lessonId, .positionId, .promptId] | @tsv
' "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse" | sort
```

Expected: output contains no accidental duplicate required decision for the same lesson purpose. The validator must also pass duplicate-decision checks in Step 7.

- [ ] **Step 6: Confirm content has deepened materially from v1**

Run:

```bash
go run ./cmd/coursepack validate "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse" \
  > /private/tmp/ruy-lopez-validation.json
jq -e '
  .courseId == "mco15-ruy-lopez-white" and
  .contentVersion == "2.0.0" and
  (.counts.lessons >= 16) and
  (.counts.lessonEdges >= 15) and
  (.counts.activities > 58) and
  (.counts.prompts >= 11) and
  (.counts.moves > 38) and
  (.counts.notes > 5)
' /private/tmp/ruy-lopez-validation.json
```

Expected: command exits `0`.

- [ ] **Step 7: Require final strict source coverage**

Run:

```bash
jq -e '
  (.coverage.missing | length == 0) and
  (.coverage.unexpected | length == 0)
' /private/tmp/ruy-lopez-validation.json
```

Expected: command exits `0`.

---

### Task 6: Final private checkpoint, import, app verification, and rebuild

**Files:**
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse`
- Read: `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse`
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/ruy-lopez-white-v2-full-reference/`
- Build: `build/bin/Chess Trainer.app`
- Modify repository files only if a generic app defect is found and fixed with a synthetic failing test first.

**Interfaces:**
- Consumes: final validated private Ruy v2 full-reference pack.
- Produces: app catalogue updated to Ruy v2, verified opening-course journey, rebuilt macOS app, clean branch ready for review and push.

- [ ] **Step 1: Create the final v2 private checkpoint**

Run:

```bash
CHECKPOINT="/Users/admin/Documents/Private Chess Courses/checkpoints/ruy-lopez-white-v2-full-reference"
mkdir -p "$CHECKPOINT"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.authoring.json" \
  "$CHECKPOINT/mco15-ruy-lopez-white.authoring.json"
cp -p "/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse" \
  "$CHECKPOINT/mco15-ruy-lopez-white.ctcourse"
cp -p /private/tmp/ruy-lopez-validation.json "$CHECKPOINT/validation.json"
jq '{courseId,contentVersion,counts,missing:(.coverage.missing|length),unexpected:(.coverage.unexpected|length)}' \
  /private/tmp/ruy-lopez-validation.json > "$CHECKPOINT/summary.json"
shasum -a 256 "$CHECKPOINT/"* > "$CHECKPOINT/SHA256SUMS.txt"
(cd "$CHECKPOINT" && shasum -a 256 -c SHA256SUMS.txt)
```

Expected: validation is clean and checksums print `OK`.

- [ ] **Step 2: Validate all private active course packs before import**

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

Expected: every pack validates.

- [ ] **Step 3: Import the final Ruy pack through the app importer**

Use **Parent settings** → **Import content** → **Choose opening course**, select:

```text
/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse
```

Expected import review:

```text
sourceId: mco15-ruy-lopez-white
sourceName: Ruy Lopez for White
replacesExisting: true
```

Choose **Import course**.

- [ ] **Step 4: Verify all three courses remain active**

In the app's Learn Openings hub, confirm:

```text
mco15-italian-white       Italian Game for White     white
mco15-ruy-lopez-white     Ruy Lopez for White        white
mco15-caro-kann-black     Caro-Kann for Black        black
```

Expected: Italian and Caro-Kann remain available; Ruy is active as v2.

- [ ] **Step 5: Verify Ruy learner journey manually**

In the app:

1. Select Ruy Lopez for White.
2. Start one Quick Ruy lesson.
3. Complete one concept or demonstration activity.
4. Complete one White decision activity.
5. Continue into the next lesson without returning to the main menu.
6. Open one Standard Ruy branch.
7. Open one Reference Ruy branch in the variation explorer.
8. Pause and resume the Ruy lesson.

Expected:

```text
board orientation: white
decision legal moves are White-to-move moves where prompted
opening activity result.appliedMoves is always an array
required learner text is not repetitive source-warning boilerplate
Continue Learning resumes Ruy when the Ruy session owns the active session
variation explorer loads deeper branches without pushing the board out of view
```

- [ ] **Step 6: Fix generic app defects only if found**

When a defect is generic, add a failing synthetic test before editing public app code. Use focused commands such as:

```bash
go test ./internal/openings -count=1
npm --prefix frontend test -- --run \
  src/components/openings/OpeningActivityContent.test.ts \
  src/components/openings/OpeningLessonScreen.test.ts \
  src/components/openings/VariationExplorer.test.ts
```

Commit a generic fix with a scoped message:

```bash
git status --short
git add internal/openings frontend/src/components/openings frontend/tests
git commit -m "fix: harden generic opening course behavior"
```

Expected: no private course data is committed.

- [ ] **Step 7: Verify private course files are untracked**

Run:

```bash
git status --short
git ls-files '*.ctcourse'
```

Expected: private files under `/Users/admin/Documents/Private Chess Courses/` are not listed. Tracked `.ctcourse` files are only synthetic fixtures under `internal/openings/testdata/`.

- [ ] **Step 8: Run full verification**

Run:

```bash
npm --prefix frontend run build
go test ./cmd/coursepack -run 'TestRunSANLine|TestRunValidate|TestRunRejectsUnsupportedCommand' -count=1
go test ./... -count=1
go vet ./...
npm --prefix frontend test -- --run --single-thread
npm --prefix frontend run check
npm --prefix frontend run verify:licenses
npm --prefix frontend run test:e2e -- openings.spec.ts --project=chromium
npm --prefix frontend run test:e2e -- openings.spec.ts --project=webkit
```

Expected: every command exits `0`.

- [ ] **Step 9: Rebuild the macOS app**

Run:

```bash
GOWORK=off GOTOOLCHAIN=local go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build -clean -trimpath
codesign --verify --deep --strict --verbose=2 "build/bin/Chess Trainer.app"
file "build/bin/Chess Trainer.app/Contents/MacOS/Chess Trainer"
```

Expected: Wails build exits `0`, codesign verification succeeds, and the app binary is `arm64`.

- [ ] **Step 10: Prepare final branch review**

Run:

```bash
git diff --check
git status --short --branch
git log --oneline --decorate -8
git ls-files '*.ctcourse'
```

Expected: only intended public docs/tooling/app changes are present. Private course data remains outside Git. Request a code-quality review before push if public code changed.

---

## Plan Self-Review

- Spec coverage: Tasks cover private v1 checkpoint, PDF rendering, source inventory, full-reference pack authoring, lesson-quality hardening, strict coverage validation, import, app verification, rebuild, and Git privacy checks.
- Private-content constraint: The plan intentionally omits MCO-derived SAN sequences, table contents, note prose, and source quotations. Those belong only in private files under `/Users/admin/Documents/Private Chess Courses/`.
- Type consistency: Course ID, version, perspective, lesson IDs, checkpoint paths, source page ranges, and validation commands match the approved Ruy design.
- Scope: Public app code changes are gated behind generic reproducible defects and synthetic tests; the expected implementation is private course authoring plus import/rebuild.
