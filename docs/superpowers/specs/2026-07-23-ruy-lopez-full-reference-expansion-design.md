# Ruy Lopez Full-Reference Expansion Design

Date: 2026-07-23
Status: Approved written spec

## Summary

Deepen the private `mco15-ruy-lopez-white` course from a seed-level White
opening course into a full-reference Ruy Lopez course based on the user's local
MCO-15 PDF.

This pass updates private course files only:

- `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.authoring.json`
- `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse`

The application runtime should be reused as-is unless a generic import,
journey, lesson, variation, or White-perspective bug is found. No MCO-derived
course content should be committed to Git or bundled into the app.

## Context

The current app has three active private opening courses:

- Italian Game for White, `mco15-italian-white`, version `2.0.0`;
- Ruy Lopez for White, `mco15-ruy-lopez-white`, version `1.0.0`;
- Caro-Kann for Black, `mco15-caro-kann-black`, version `2.0.0`.

The Ruy Lopez course is still intentionally shallow:

- 16 lessons;
- 15 lesson edges;
- 58 activities;
- 11 prompts;
- 38 moves;
- 5 notes;
- printed pages 42-95 declared.

The Caro-Kann course now proves the desired full-reference workflow at a
smaller source scale: private page-by-page authoring, strict source coverage,
Reference depth for dense material, and Quick / Standard paths that stay
usable for learners. Ruy Lopez should now receive the same treatment in one
complete pass.

The MCO-15 PDF is an image-based scan. Text extraction has been unreliable, so
this expansion uses manual visual curation from rendered page images. The
expected Ruy Lopez source scope is printed pages 42-95. The earlier rendering
workflow used the scan offset recorded by the private Ruy v1 checkpoint; this
pass should verify the offset again before authoring.

## Product Decisions

- Deepen Ruy Lopez now, after the Caro-Kann full-reference pass.
- Use option A: one complete full-reference Ruy v2 pass instead of staged
  partial batches.
- Preserve the course ID: `mco15-ruy-lopez-white`.
- Bump `contentVersion` from `1.0.0` to `2.0.0`.
- Preserve existing lesson, prompt, position, move, activity, and edge IDs
  where their meaning remains the same.
- Add new Reference-only lessons where a source family is distinct enough to
  deserve its own node.
- Keep Quick and Standard concise; put dense source detail into Reference.
- Treat White as the learner:
  - White moves are `repertoire` or `alternative`;
  - Black moves are `opponent`;
  - decision prompts occur only where White is to move.
- Keep all MCO-derived prose, notes, source lines, and course data private.

## Goals

- Build a full-reference private Ruy Lopez source inventory for printed pages
  42-95.
- Expand the Ruy Lopez move graph with the major table families, note-letter
  branches, illustrative game references, and alternate move orders that are
  legible in the PDF.
- Preserve the current Quick teaching path as a clear White-first learner
  journey.
- Make Standard depth a practical White repertoire against major Black
  systems.
- Add dense Reference material without making the lesson screen noisy.
- Validate the resulting `.ctcourse` with zero missing and zero unexpected
  source coverage.
- Import the updated pack and confirm it replaces only the Ruy Lopez course
  generation.
- Confirm Italian and Caro-Kann remain active and unaffected.
- Rebuild the app after verification.

## Non-Goals

- Do not add new openings in this pass.
- Do not deepen Italian or Caro-Kann further in this pass.
- Do not create an in-app course editor.
- Do not OCR or import the whole MCO-15 PDF automatically.
- Do not create a second opening-course engine.
- Do not special-case Ruy Lopez in public app code.
- Do not commit private `.ctcourse` files, private authoring inventories, or
  MCO-derived content to Git.

## Course Shape

The course remains:

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

Existing top-level lessons should remain stable where possible:

- `ruy-foundations`
- `ruy-black-third-move`
- `ruy-morphy-a6`
- `ruy-preserve-bishop`
- `ruy-castle-re1`
- `ruy-central-plan`
- `ruy-exchange`
- `ruy-open`
- `ruy-berlin`
- `ruy-steinitz`
- `ruy-closed-main`
- `ruy-anti-marshall`
- `ruy-marshall-warning`
- `ruy-delayed-systems`
- `ruy-closed-systems`
- `ruy-side-systems`

These IDs are progress anchors. If a lesson's meaning is unchanged, keep its
ID and deepen it with demonstrations, comparisons, optional Reference
activities, and richer graph coverage. If a source family is too large or too
distinct for an existing node, add a new stable Reference-only lesson rather
than overloading the older seed lesson.

## Depth Strategy

### Quick

Quick remains compact and practical:

- reach the Ruy Lopez as White;
- understand the pressure on Black's centre;
- recognize Black's early choice between the main `...a6` family, Berlin-style
  development, direct simplification, and sharper side systems;
- know one coherent White plan through the main Morphy / Closed structure;
- learn one safe anti-Marshall or practical sidestep concept;
- avoid dense note-letter analysis.

Quick should feel like a first repertoire path, not a table of variations.

### Standard

Standard becomes the working White repertoire:

- Exchange Variation structures;
- Open Ruy Lopez treatment;
- Berlin treatment;
- Steinitz and deferred systems;
- main Morphy / Closed Ruy Lopez plans;
- Marshall and Anti-Marshall choices;
- one practical response path against sharper side systems.

Standard should contain the important decisions a club player must recognize,
but it should not require replaying every historical subvariation as a
puzzle-like activity.

### Reference

Reference becomes the source-backed library:

- late-page Closed Ruy Lopez branches;
- Marshall and Anti-Marshall detail;
- note-letter subvariations;
- illustrative game citations;
- alternate move orders;
- uncommon systems;
- dense source analysis.

Reference material should be available through optional Reference activities
and the variation explorer. Required activities should stay focused on
teaching decisions and ideas.

## Teaching Tree Families

The Ruy v2 teaching tree should be organized by what the learner experiences
as White:

1. Foundations: why the Ruy Lopez is chosen and what White is trying to
   pressure.
2. Black's early choice: the major third-move and fourth-move families.
3. Morphy / main `...a6` family: how White keeps the bishop's pressure while
   preparing central play.
4. Exchange Variation: when simplification changes the pawn structure and what
   White is playing for.
5. Open Ruy Lopez: what changes when Black takes central space and activity
   early.
6. Berlin: how White should understand the structural and endgame signals.
7. Steinitz and deferred systems: when Black defends more solidly and White can
   claim space.
8. Closed Ruy Lopez: the main manoeuvring tree and central break timing.
9. Marshall / Anti-Marshall: the practical crossroads before Black's sharp
   counterplay.
10. Reference systems: rarer or denser source families that are useful to keep
    in the explorer but not in the main learner spine.

## Source Inventory Workflow

The private authoring inventory should be expanded page by page. For each
legible table column, note, or source branch, record enough private metadata to
reconstruct and validate the course pack:

- printed page;
- PDF page image used for visual review;
- table column or note label;
- variation or branch family;
- SAN sequence;
- evaluation where available;
- illustrative game reference where available;
- intended minimum depth: Quick, Standard, or Reference;
- intended activity treatment:
  - decision prompt;
  - demonstration;
  - comparison line;
  - optional Reference section;
  - variation-explorer-only coverage.

The authoring file may contain source-derived chess lines and private notes
because it stays outside the repository. Public reports and commits should
contain only counts, file paths, validation results, and high-level summaries.

If a scan element is illegible, do not fake coverage. Record the issue in the
private checkpoint notes and either omit that element or mark it as a blocked
source item outside the validated `expectedReferences` set.

## Source Coverage Contract

The full-reference pass is accepted only when the generated `.ctcourse`
validates with:

- `courseId = "mco15-ruy-lopez-white"`;
- `contentVersion = "2.0.0"`;
- nonzero lessons, lesson edges, activities, prompts, moves, and notes;
- `coverage.missing` length `0`;
- `coverage.unexpected` length `0`.

Coverage should be strict for everything included in the full-reference
authoring scope. Every declared expected reference must be represented by a
move or note, and every move or note source reference must be declared.

## Lesson Authoring Rules

The lesson shape continues to use schema-v2 activities:

- `concept` for a single idea;
- `demonstration` for a connected move sequence;
- `decision` for one White course move;
- `comparison` for contrasting source branches;
- `recap` for consolidation;
- `reference` for optional dense material.

Required activities should teach meaningful decisions and should not repeat
the same move in several puzzle-like actions. Dense MCO analysis belongs in
optional Reference activities or the variation explorer. This preserves the
lesson redesign goal: courses should feel like lessons, not reused tactics
puzzles.

Required lesson text should be original teaching prose. Short source labels,
variation names, move notation, evaluations, and game citations may appear in
private course files where needed for study, but public repository files must
avoid transcribed MCO prose and full table contents.

## Checkpoints

Before modifying private Ruy files, create a recoverable checkpoint of the
current v1 seed pack:

- current authoring JSON;
- current `.ctcourse`;
- current validation output;
- `SHA256SUMS.txt`.

After the v2 full-reference pack validates, create a second checkpoint:

- updated authoring JSON;
- updated `.ctcourse`;
- validation output;
- source-inventory summary;
- `SHA256SUMS.txt`.

The checkpoint directory names should make the versions clear, for example:

- `/Users/admin/Documents/Private Chess Courses/checkpoints/ruy-lopez-white-v1-before-full-reference/`
- `/Users/admin/Documents/Private Chess Courses/checkpoints/ruy-lopez-white-v2-full-reference/`

## Import and App Activation

After private validation, import the updated pack through the existing app
import flow:

- source ID should remain `mco15-ruy-lopez-white`;
- active generation should move from v1 to v2;
- Italian and Caro-Kann active generations should remain unchanged;
- course grouping by perspective should still show White and Black sections
  correctly;
- Continue Learning and journey progress should remain course-specific.

If a user's existing Ruy v1 journey cannot be rebased safely, use the existing
course journey rebase behavior: preserve prior progress where possible and
restart only the affected course journey from a stable checkpoint.

## Application Changes

This is primarily private content authoring. Public app code should change
only if the full Ruy v2 pack exposes a generic defect.

Allowed generic fixes:

- importer or validator bugs found by large White-course graphs;
- journey rebase bugs found when replacing v1 with v2;
- lesson layout bugs caused by deeper Reference activity lists;
- variation explorer bugs caused by larger branching graphs;
- generic copy or grouping issues in the opening hub.

Disallowed changes:

- hard-coded Ruy Lopez logic;
- course-specific UI branches;
- bundled private source content;
- schema changes that are not required by the generic course model.

## Verification

Private content verification:

- validate the v1 checkpoint before editing;
- validate the v2 pack after each substantial authoring batch;
- require zero missing and zero unexpected coverage before import;
- record final counts and hashes in the private checkpoint.

Repository verification:

- `git diff --check`;
- `go test ./cmd/coursepack -run 'TestRunSANLine|TestRunValidate|TestRunRejectsUnsupportedCommand' -count=1`;
- `go test ./... -count=1`;
- `go vet ./...`;
- frontend build, type checks, and unit tests if public app files change;
- targeted opening-course e2e checks if import, journey, or UI behavior
  changes;
- release policy check that no private `.ctcourse` is tracked.

App verification:

- import Italian, Ruy, and Caro packs together;
- confirm Ruy v2 is active and visible under White courses;
- start and complete at least one Ruy lesson;
- inspect a deeper Ruy Reference branch in the variation explorer;
- confirm the app rebuild succeeds after verification.

## Risks and Mitigations

### Risk: Ruy is much larger than Caro-Kann

Mitigation: Keep the one-pass goal, but author internally in page batches with
validation after each batch. Do not activate the course until the full v2 pack
passes strict validation.

### Risk: Required lessons become repetitive

Mitigation: Keep required activities sparse and meaningful. Use optional
Reference activities and the variation explorer for dense source coverage.

### Risk: Manual scan reading introduces transcription errors

Mitigation: Use rendered page images, private checkpoints, SAN validation, and
source coverage validation. Record illegible items instead of inventing
coverage.

### Risk: Existing user progress is invalidated by v2 graph changes

Mitigation: Preserve stable IDs where possible and rely on the existing journey
rebase behavior for affected Ruy sessions only.

### Risk: Private source data leaks into Git

Mitigation: Keep all MCO-derived files under `/Users/admin/Documents/Private Chess Courses/`,
run `git ls-files '*.ctcourse'`, and keep repository commits limited to public
specs, generic code, tests, and documentation.

## Success Criteria

- Private Ruy Lopez pack is updated to `contentVersion = "2.0.0"`.
- Private Ruy source coverage validates with zero missing and zero unexpected
  references.
- Quick and Standard remain learner-friendly and non-repetitive.
- Reference depth contains the dense source-backed Ruy graph for the selected
  printed-page scope.
- The app imports Ruy v2 without disrupting Italian or Caro-Kann.
- The rebuilt app launches with the expanded Ruy lesson available.
- Git history contains no private course content.
