# Italian White v2 Course Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. The course is private copyrighted data, so do not dispatch its transcription or source review to subagents.

**Goal:** Upgrade the existing private MCO-15 Italian White pack from five repetitive lessons to a schema-v2 teaching tree of about 23 meaningful nodes while preserving its move graph, source coverage, learner completion, and review history.

**Architecture:** Reuse the external pack's 635 positions, 636 moves, 251 notes, 48 source paths, fingerprints, and printed-page coverage. Replace its teaching layer with stable nodes, explicit edges, flexible activities, and distinct prompts; validate/import it through the teaching-tree engine from the preceding plan.

**Tech Stack:** Private UTF-8 `.ctcourse` JSON, private authoring JSON, Go `cmd/coursepack`, local SQLite, and the Wails macOS app.

## Global Constraints

- Complete `docs/superpowers/plans/2026-07-20-opening-teaching-tree-engine.md` and its code-review gate first.
- Follow `docs/superpowers/specs/2026-07-20-opening-course-teaching-tree-design.md`.
- Source PDF: `/Users/admin/Downloads/kupdf.net_modern-chess-openings-15th-edition.pdf`.
- Private pack: `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse`.
- Private inventory: `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.authoring.json`.
- Keep course ID `mco15-italian-white`; set schema `2`, content version `2.0.0`, default depth `reference`, perspective `white`.
- Preserve full coverage for printed pages 18–41: Giuoco 18–25, Evans 26–29, Two Knights 30–41.
- Review the scan visually; do not trust OCR for moves, symbols, columns, footnotes, or transpositions.
- Never copy private course/PDF content into the repository, build assets, test fixtures, logs, or final response.
- Use original concise teaching prose; dense analysis stays optional.
- Never require the same answer from the same position twice in one lesson.
- Repetition belongs to spaced review; remove the Mixed Recall chapter.
- Preserve lesson IDs `foundations-e4`, `giuoco-c3`, `evans-accepted-c3`, `two-knights-ng5`, and `mixed-exd5`. Rehome `mixed-exd5` into Two Knights.
- Preserve the five existing prompt IDs when their primary moves remain unchanged.
- Use `apply_patch` for private JSON edits. Private checkpoints are copies and SHA-256 digests, not Git commits.

## Baseline

- Schema/content: `1` / `1.0.0`.
- Positions/moves/notes: 635 / 636 / 251.
- Paths: 18 Giuoco, 6 Evans, 24 Two Knights.
- Lessons/prompts: 5 / 5.
- Current lesson pattern repeats one prompt as Try, Branch, and Recall.

---

### Task 1: Protect and Inventory the Baseline

**Files:**
- Read: the two private files above
- Create: `/Users/admin/Documents/Private Chess Courses/checkpoints/italian-v1.0.0/`

**Interfaces:**
- Produces: recoverable baseline, hashes, counts, paths, and stable-ID migration list.

- [ ] **Step 1: Close the app and checkpoint private files**

```bash
mkdir -p '/Users/admin/Documents/Private Chess Courses/checkpoints/italian-v1.0.0'
cp -p '/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse' '/Users/admin/Documents/Private Chess Courses/checkpoints/italian-v1.0.0/mco15-italian-white.ctcourse'
cp -p '/Users/admin/Documents/Private Chess Courses/mco15-italian-white.authoring.json' '/Users/admin/Documents/Private Chess Courses/checkpoints/italian-v1.0.0/mco15-italian-white.authoring.json'
shasum -a 256 '/Users/admin/Documents/Private Chess Courses/checkpoints/italian-v1.0.0/'*
```

Expected: two non-empty digest lines. If files already exist, compare hashes before replacing anything.

- [ ] **Step 2: Validate and inventory read-only**

```bash
go run ./cmd/coursepack validate '/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse'
jq '{schemaVersion,courseId,contentVersion,counts:{positions:(.positions|length),moves:(.moves|length),notes:(.notes|length),lessons:(.lessons|length),prompts:(.prompts|length)},printedPages:.sourceCoverage.printedPages}' '/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse'
jq -r '.paths[] | [.id,.chapter,.column,.printedPage,.depth,.variation,.evaluation] | @tsv' '/Users/admin/Documents/Private Chess Courses/mco15-italian-white.authoring.json'
```

Expected: validator exit 0, baseline counts above, pages 18–41, 48 paths.

- [ ] **Step 3: Confirm private separation**

```bash
git status --short
git ls-files '*.ctcourse'
```

Expected: only synthetic fixtures are tracked; neither private path appears.

---

### Task 2: Author the Teaching-Tree Manifest

**Files:**
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.authoring.json`

**Interfaces:**
- Produces: 23 `teachingNodes` and 22 `teachingEdges`.

- [ ] **Step 1: Add the exact node manifest**

| Lesson ID | Chapter | Depth | Purpose |
|---|---|---:|---|
| `foundations-e4` | foundations | quick | Reach the Italian and learn shared goals |
| `foundations-black-split` | foundations | quick | Compare `...Bc5` and `...Nf6` |
| `giuoco-c3` | giuoco | quick | Prepare `d4` with `c3` |
| `giuoco-d4` | giuoco | quick | Time central occupation with `d4` |
| `giuoco-bb4-check` | giuoco | quick | Meet `...Bb4+` while preserving the plan |
| `giuoco-moller` | giuoco | standard | Understand Møller initiative and sacrifices |
| `giuoco-quiet-d3` | giuoco | standard | Understand the quiet `d3` structure |
| `giuoco-queenside-plans` | giuoco | standard | Compare Re1, Nbd2, and b4 plans |
| `giuoco-side-systems` | giuoco | reference | Compare less common c3, Nc3, and castling systems |
| `evans-b4` | evans | standard | Understand compensation behind `b4` |
| `evans-accepted-c3` | evans | standard | Build the accepted-gambit centre |
| `evans-castle-or-d4` | evans | standard | Coordinate castling and `d4` |
| `evans-defensive-setups` | evans | standard | Recognize Lasker, compromised, `...d6`, and `...Be7` |
| `evans-declined` | evans | standard | Meet the declined gambit |
| `two-knights-ng5` | two-knights | quick | Understand `Ng5` |
| `mixed-exd5` | two-knights | quick | Answer `...d5` with the critical capture |
| `two-knights-na5` | two-knights | quick | Learn the principal `...Na5` continuation |
| `two-knights-fritz` | two-knights | standard | Recognize Fritz `...Nd4` |
| `two-knights-ulvestad` | two-knights | standard | Recognize Ulvestad `...b5` |
| `two-knights-wilkes-barre` | two-knights | standard | Compare Wilkes-Barre choices |
| `two-knights-quiet-d3` | two-knights | standard | Teach quiet `d3` as the full alternative |
| `two-knights-max-lange` | two-knights | reference | Understand the Max Lange attack |
| `two-knights-side-systems` | two-knights | reference | Compare `5.e5`, `Nc3`, and sidelines |

- [ ] **Step 2: Add exact parent edges**

```text
foundations-e4 -> foundations-black-split                         continuation
foundations-black-split -> giuoco-c3                              continuation, ...Bc5
foundations-black-split -> two-knights-ng5                        continuation, ...Nf6
foundations-black-split -> evans-b4                               alternative, Evans Gambit
giuoco-c3 -> giuoco-d4                                            continuation
giuoco-d4 -> giuoco-bb4-check                                     continuation
giuoco-bb4-check -> giuoco-moller                                 alternative
giuoco-c3 -> giuoco-quiet-d3                                      alternative
giuoco-quiet-d3 -> giuoco-queenside-plans                         continuation
giuoco-d4 -> giuoco-side-systems                                  reference
evans-b4 -> evans-accepted-c3                                     continuation
evans-accepted-c3 -> evans-castle-or-d4                           continuation
evans-castle-or-d4 -> evans-defensive-setups                      continuation
evans-b4 -> evans-declined                                        alternative
two-knights-ng5 -> mixed-exd5                                     continuation
mixed-exd5 -> two-knights-na5                                     continuation
mixed-exd5 -> two-knights-fritz                                   alternative
mixed-exd5 -> two-knights-ulvestad                                alternative
two-knights-ng5 -> two-knights-wilkes-barre                       alternative
two-knights-ng5 -> two-knights-quiet-d3                            alternative
two-knights-ng5 -> two-knights-max-lange                          reference
two-knights-max-lange -> two-knights-side-systems                  reference
```

Use edge IDs `<from>-to-<to>`, authored sibling ordinals in listed order, and child's minimum depth.

- [ ] **Step 3: Validate the private manifest**

Use `jq` to assert 23 unique node IDs, 22 unique edge IDs, one root, no blank parent/child, and this retained list exactly once:

```json
["foundations-e4","giuoco-c3","evans-accepted-c3","two-knights-ng5","mixed-exd5"]
```

Expected: `mixed-recall` is absent. Save a digest checkpoint under `checkpoints/italian-v2-manifest/` and confirm Git status is unchanged.

- [ ] **Step 4: Reserve exact prompt identities**

Reuse `prompt-foundations-e4`, `prompt-giuoco-c3`, `prompt-evans-c3`, `prompt-two-knights-ng5`, and `prompt-mixed-exd5`. Create these stable new IDs for distinct decisions:

```text
prompt-giuoco-d4
prompt-giuoco-bd2
prompt-moller-castle
prompt-moller-d5
prompt-giuoco-d3
prompt-evans-b4
prompt-evans-castle
prompt-evans-declined-a4
prompt-two-knights-bb5-check
prompt-two-knights-dxc6
prompt-fritz-c3
prompt-ulvestad-bf1
prompt-two-knights-d3
prompt-max-lange-d4
prompt-max-lange-castle
```

`prompt-evans-castle` teaches `O-O` from the Lasker/main move order; immediate `d4` remains the comparison. Every new prompt primary move must already exist in the graph, and accepted alternatives must leave the same position.

---

### Task 3: Curate Foundations and Giuoco

**Files:**
- Modify: both private JSON files
- Read visually: printed pages 18–25 / PDF pages 35–42

**Interfaces:**
- Produces: nine nodes with distinct decisions and original teaching prose.

- [ ] **Step 1: Map existing paths**

```text
main plan: giuoco-1
...Bb4+ and Møller: giuoco-2 through giuoco-6
quiet d3 and queenside plans: giuoco-7 through giuoco-12
side systems: giuoco-13 through giuoco-18
```

- [ ] **Step 2: Author exact required activity sequences**

```text
foundations-e4: Concept -> Demonstration -> Decision e4 -> Recap
foundations-black-split: Concept -> Comparison (...Bc5 / ...Nf6) -> Recap
giuoco-c3: Concept -> Demonstration -> Decision c3 -> Recap
giuoco-d4: Concept -> Decision d4 -> Comparison (active d4 / quiet d3) -> Recap
giuoco-bb4-check: Demonstration -> Decision Bd2 -> Comparison (Bd2 / Kf1 / Nc3) -> Recap
giuoco-moller: Concept -> Demonstration -> Decision O-O -> later Decision d5 -> Recap
giuoco-quiet-d3: Concept -> Decision d3 -> Comparison (Re1 / Nbd2 / b4) -> Recap
giuoco-queenside-plans: Concept -> Comparison (three plans) -> Recap
giuoco-side-systems: Concept -> Comparison (4...Bb6 / 4...d6 / 4.Nc3 / 4.O-O) -> optional Reference
```

Every Decision points to one existing graph move and unique position. Promote a move and its parents to Standard only when a Standard lesson requires it; never duplicate a graph node.

- [ ] **Step 3: Apply teaching-text rules**

Use one objective sentence and at most three short teaching paragraphs per required activity. Put columns, evaluations, and game citations in optional Reference. Primary teaching text must not match `Source note [a-z] records`, `records a concrete analytical subvariation`, `private reference`, or `Course move found`.

- [ ] **Step 4: Validate and checkpoint**

For this intermediate checkpoint, build a temporary private pack containing only the authored Foundations/Giuoco lessons and only edges whose two endpoints are present. Retain the full shared positions, moves, notes, prompts, and source-coverage records so no source material is deleted while later chapters are unfinished. Run the validator and require zero activity, duplicate-decision, move-depth, tree, graph, and coverage diagnostics. Save the authoring source and generated pack under `checkpoints/italian-v2-giuoco/` with hashes; do not import this partial pack into the app.

---

### Task 4: Curate Evans

**Files:**
- Modify: both private JSON files
- Read visually: printed pages 26–29 / PDF pages 43–46

**Interfaces:**
- Produces: five-node fully taught alternative to Giuoco.

- [ ] **Step 1: Map paths and author activities**

```text
evans-b4: shared root before evans-1..6; Concept -> Decision b4 -> Recap
evans-accepted-c3: evans-1,2; Demonstration -> Decision c3 -> Recap
evans-castle-or-d4: evans-1,2; Comparison -> one Decision on selected main move order -> Recap
evans-defensive-setups: evans-1,3,4,5; Concept -> Comparison -> optional Reference
evans-declined: evans-6; Demonstration -> Decision a4 after ...Bb6 -> Recap
```

Keep `evans-accepted-c3` and its existing prompt identity. `evans-b4` introduces the branch and never repeats c3.

- [ ] **Step 2: Validate depth and checkpoint**

Run validator. Require all Evans nodes hidden at Quick and visible at Standard/Reference, zero missing/unexpected coverage, no duplicate decisions, hidden moves, or tree errors. Save `checkpoints/italian-v2-evans/` with hashes.

---

### Task 5: Curate Two Knights and Rehome Mixed Recall

**Files:**
- Modify: both private JSON files
- Read visually: printed pages 30–41 / PDF pages 47–58

**Interfaces:**
- Produces: nine-node main/alternative route and stable `mixed-exd5` migration.

- [ ] **Step 1: Map all source paths**

```text
two-knights-ng5 / mixed-exd5 / two-knights-na5: two-knights-1..8
two-knights-fritz: two-knights-9
two-knights-ulvestad: two-knights-10
two-knights-wilkes-barre: two-knights-11,12
two-knights-max-lange: two-knights-13..18
two-knights-side-systems: two-knights-19,20,24
two-knights-quiet-d3: two-knights-21..23
```

- [ ] **Step 2: Author exact required activities**

```text
two-knights-ng5: Concept -> Decision Ng5 -> Recap
mixed-exd5: Demonstration -> Decision exd5 -> Comparison of Black replies -> Recap
two-knights-na5: Demonstration -> Decision Bb5+ -> later Decision dxc6 -> Recap
two-knights-fritz: Concept -> Demonstration -> Decision c3 -> Recap
two-knights-ulvestad: Concept -> Demonstration -> Decision Bf1 -> Recap
two-knights-wilkes-barre: Concept -> Comparison (Bxf7+ / Nxf7) -> optional Reference
two-knights-quiet-d3: Concept -> Decision d3 -> Demonstration of castling/development -> Recap
two-knights-max-lange: Concept -> Decision d4 -> later Decision O-O -> Comparison -> Recap
two-knights-side-systems: Concept -> Comparison (5.e5 / 4.Nc3) -> optional Reference
```

Move `mixed-exd5` from the removed Mixed Recall chapter into Two Knights. Keep its lesson and prompt IDs. Remove duplicate Try/Branch/Recall activities; future repetition comes from review only.

- [ ] **Step 3: Validate route policy**

Quick exposes `two-knights-ng5`, `mixed-exd5`, `two-knights-na5`. Standard adds Fritz, Ulvestad, Wilkes-Barre, and the fully taught quiet `d3` alternative. Max Lange and side systems remain Reference. Promote every required move's parent path to the same depth when needed.

- [ ] **Step 4: Validate and checkpoint**

Run validator and require zero diagnostics. Save both files under `checkpoints/italian-v2-two-knights/` with hashes.

---

### Task 6: Assemble and Validate the Complete v2 Pack

**Files:**
- Modify: `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse`

**Interfaces:**
- Produces: importable schema-v2 private pack.

- [ ] **Step 1: Set exact metadata and teaching collections**

```json
{
  "schemaVersion": 2,
  "courseId": "mco15-italian-white",
  "contentVersion": "2.0.0",
  "perspective": "white",
  "defaultDepth": "reference"
}
```

Replace every legacy `steps` collection with authored `activities`; add 22 edges; remove chapter `mixed-recall`; retain foundations, Giuoco, Evans, and Two Knights.

- [ ] **Step 2: Run deterministic validation twice**

```bash
go run ./cmd/coursepack validate '/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse'
go run ./cmd/coursepack validate '/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse'
```

Expected both times: course `mco15-italian-white`, version `2.0.0`, 4 chapters, 23 lessons, 22 edges, authored activity count, empty missing/unexpected coverage, identical output.

- [ ] **Step 3: Run private quality scans**

```bash
jq -r '.lessons[].activities[] | select(.required) | [.activityId,.kind,.positionId,.promptId] | @tsv' '/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse'
jq -r '.lessons[] | .lessonId as $lesson | .activities[] | select(.kind=="decision") | [$lesson,.positionId,.promptId] | @tsv' '/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse' | sort
jq -r '.lessons[].activities[] | select(.required) | .instruction' '/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse' | rg -n 'Source note [a-z] records|records a concrete analytical subvariation|Course move found'
jq -r '([.lessons[].activities[] | select(.required) | .noteIds[]] | unique) as $ids | .notes[] | select(.noteId as $id | $ids | index($id)) | .text' '/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse' | rg -n 'Source note [a-z] records|records a concrete analytical subvariation|Course move found'
```

Expected: all required activities have stable IDs, no lesson repeats a decision key, and both placeholder scans return no matches. Bibliographic detail appears only in optional Reference/explorer records.

- [ ] **Step 4: Verify private separation and final checkpoint**

```bash
git status --short
git ls-files '*.ctcourse'
shasum -a 256 '/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse' '/Users/admin/Documents/Private Chess Courses/mco15-italian-white.authoring.json'
```

Expected: no private file is tracked. Save exact copies and digests under `checkpoints/italian-v2-final/`.

---

### Task 7: Test Migration Against Copied Real Data

**Files:**
- Read/copy: `/Users/admin/Library/Application Support/Chess Trainer/user.sqlite`
- Read/copy: `/Users/admin/Library/Application Support/Chess Trainer/courses.sqlite`
- Create: one `mktemp -d` test root outside the repository

**Interfaces:**
- Produces: evidence that completions, active session, fingerprints, and reviews survive import.

- [ ] **Step 1: Capture read-only pre-import state**

With the app closed:

```bash
sqlite3 -readonly '/Users/admin/Library/Application Support/Chess Trainer/user.sqlite' "SELECT lesson_id, completed_at IS NOT NULL FROM opening_lesson_progress WHERE course_id='mco15-italian-white' ORDER BY lesson_id;"
sqlite3 -readonly '/Users/admin/Library/Application Support/Chess Trainer/user.sqlite' "SELECT lesson_id,status,activity_index FROM opening_sessions WHERE course_id='mco15-italian-white' AND status IN ('active','paused','restart_required');"
```

Expected: retained legacy IDs and at most one resumable session.

- [ ] **Step 2: Import only into an isolated copy**

Create a temp root with `mktemp -d`; copy both databases plus WAL/SHM files when present. Start the service/app against that explicit root and import v2. Do not mutate the real databases.

- [ ] **Step 3: Assert migration outcomes**

Verify retained completed nodes stay complete; `mixed-exd5` appears under Two Knights with retained completion; compatible active work resumes at matching activity; incompatible repeated legacy steps enter explicit nearest-activity restart; unchanged fingerprints retain reviews; removed/changed fingerprints archive only themselves; new lessons remain incomplete.

- [ ] **Step 4: Remove only the temp root**

Print and inspect the exact temp path; confirm it is neither the repository nor Application Support; delete only that directory.

---

### Task 8: Import, Accept, and Rebuild

**Files:**
- Import: private v2 pack through the app
- Update: local SQLite only through normal migrations/import

**Interfaces:**
- Produces: validated real learner journey and rebuilt macOS app.

- [ ] **Step 1: Back up real app data**

With the app closed, copy `/Users/admin/Library/Application Support/Chess Trainer/` to a timestamped sibling backup. Open both copied SQLite databases read-only before continuing.

- [ ] **Step 2: Import through Parent settings**

Choose the external v2 pack, confirm course ID/title/version/counts, and import. Do not replace SQLite files manually.

- [ ] **Step 3: Walk the acceptance journey**

Verify:

1. tree shows prior completion, one recommendation, and visible branches;
2. Quick is a coherent essential route;
3. Standard adds Evans and major Two Knights defences;
4. Reference exposes every node and optional analysis;
5. `giuoco-c3` asks for c3 once, then a different decision;
6. node completion opens the checkpoint;
7. Continue starts the next node without hub navigation;
8. pause/restart resumes exact activity;
9. depth changes preserve progress;
10. spaced review replaces Mixed Recall;
11. source analysis stays collapsed; and
12. Variation Explorer retains full graph/citations.

- [ ] **Step 4: Verify and build**

```bash
go test ./... -count=1
npm --prefix frontend test -- --run --single-thread
npm --prefix frontend run check
npm --prefix frontend run build
npm --prefix frontend run verify:licenses
go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build
git diff --check
git status --short
git ls-files '*.ctcourse'
```

Expected: all exit 0; rebuilt app is under `build/bin/`; no private pack is tracked.

- [ ] **Step 5: Retain recovery checkpoints**

Record v2 private hashes and real-data backup path in the private checkpoint directory. Keep the v1 checkpoint until the journey and several review cycles are confirmed; do not delete it in this plan.
