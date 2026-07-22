# Caro-Kann Full-Reference Expansion Design

Date: 2026-07-23
Status: Approved design direction, awaiting written-spec review

## Summary

Deepen the private `mco15-caro-kann-black` course from a curated seed pack into
a full-reference Caro-Kann course based on the user's local MCO-15 PDF. This is
the first full-reference pass before doing the larger Ruy Lopez expansion.

The work updates private course files only:

- `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.authoring.json`
- `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse`

The app runtime should be reused as-is unless a generic import, journey,
variation, or Black-perspective bug is found. No MCO-derived course content
should be committed to Git or bundled into the app.

## Context

The current app has three active private opening courses:

- Italian Game for White, `mco15-italian-white`, version `2.0.0`;
- Ruy Lopez for White, `mco15-ruy-lopez-white`, version `1.0.0`;
- Caro-Kann for Black, `mco15-caro-kann-black`, version `1.0.0`.

The current Caro-Kann pack is useful but intentionally shallow:

- 16 lessons;
- 15 lesson edges;
- 63 activities;
- 15 prompts;
- 45 moves;
- 5 notes;
- printed pages 171-196 declared.

By contrast, the Italian course is much richer, with hundreds of moves and
source-backed notes. The Caro-Kann expansion should close that gap by turning
Reference depth into a real private source-backed study tree while keeping
Quick and Standard usable for learners.

The MCO-15 PDF is an image-based scan. Prior attempts at text extraction were
empty, so this expansion uses manual visual curation from rendered pages. The
expected printed-page scope is Caro-Kann Defence pages 171-196, corresponding
to PDF pages 188-213 with the current scan offset.

## Product Decisions

- Deepen Caro-Kann first, before Ruy Lopez, because it has the smaller source
  range and is the best place to prove the full-reference workflow.
- Preserve the course ID: `mco15-caro-kann-black`.
- Bump `contentVersion` from `1.0.0` to `2.0.0`.
- Preserve existing lesson, prompt, position, move, activity, and edge IDs
  where their meaning remains the same.
- Allow new Reference-only lessons and activities when a source branch is
  genuinely distinct.
- Keep Quick and Standard concise; put dense source detail into Reference.
- Treat Black as the learner:
  - White moves are `opponent`;
  - Black course moves are `repertoire` or `alternative`;
  - decision prompts occur only where Black is to move.
- Keep all MCO-derived prose, notes, and course data private.

## Goals

- Build a full-reference private Caro-Kann source inventory for printed pages
  171-196.
- Expand the Caro-Kann move graph with the major table columns, note-letter
  branches, illustrative game references, and alternate move orders that are
  legible in the PDF.
- Preserve the current Quick and Standard teaching path as a clean learner
  journey.
- Add dense Reference material without making the lesson screen noisy.
- Validate the resulting `.ctcourse` with zero missing and zero unexpected
  source coverage.
- Import the updated pack and confirm it replaces only the Caro-Kann course
  generation.
- Confirm Italian and Ruy Lopez remain active and unaffected.
- Rebuild the app after verification.

## Non-Goals

- Do not deepen Ruy Lopez in this pass.
- Do not add new openings.
- Do not create an in-app course editor.
- Do not OCR or import the whole PDF automatically.
- Do not create a second opening-course engine.
- Do not special-case Caro-Kann in public app code.
- Do not commit private `.ctcourse` files, private authoring inventories, or
  MCO-derived prose to Git.

## Course Shape

The course remains:

```json
{
  "schemaVersion": 2,
  "courseId": "mco15-caro-kann-black",
  "contentVersion": "2.0.0",
  "title": "Caro-Kann for Black",
  "perspective": "black",
  "defaultDepth": "reference"
}
```

Existing top-level lessons should remain stable where possible:

- `caro-foundations`
- `caro-d5-structure`
- `caro-mainline-bishop`
- `caro-advance-c5`
- `caro-exchange-structure`
- `caro-panov-targets`
- `caro-sideline-setup`
- `caro-classical-main`
- `caro-advance-space`
- `caro-exchange-minority`
- `caro-panov-iqp`
- `caro-fantasy`
- `caro-two-knights`
- `caro-endgames`
- `caro-sharp-reference`
- `caro-rare-systems`

These IDs are progress anchors. If a lesson's meaning is unchanged, keep its
ID and deepen it with additional demonstrations, comparisons, optional
Reference activities, and richer source-backed graph coverage. If a source
branch is distinct enough to deserve its own node, add a new stable
Reference-only lesson ID rather than overloading an existing lesson.

## Depth Strategy

### Quick

Quick remains compact and practical:

- answer `1.e4` with `...c6`;
- challenge with `...d5`;
- understand the Classical bishop idea;
- know the Advance `...c5` break;
- recognize Exchange and Panov structures;
- use one compact setup against sidelines.

Quick should avoid dense note-letter analysis. It should contain the shortest
coherent Black repertoire path and a small number of meaningful decisions.

### Standard

Standard becomes the working repertoire:

- Classical main branches;
- Advance-space plans;
- Exchange and minority-play ideas;
- Panov and isolated-queen-pawn plans;
- Fantasy;
- Two Knights;
- common stable endgames.

Standard should add important choices a practical player must recognize, but
not every historical or highly tactical subvariation.

### Reference

Reference becomes the source-backed library:

- sharp sidelines;
- note-letter subvariations;
- illustrative game citations;
- alternate move orders;
- uncommon systems;
- dense source analysis.

Reference material should be available through the lesson's optional
Reference sections and the variation explorer. It should not spam the main
lesson path with repetitive required actions.

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

- `courseId = "mco15-caro-kann-black"`;
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
- `decision` for one Black course move;
- `comparison` for contrasting source branches;
- `recap` for consolidation;
- `reference` for optional dense material.

Required activities should teach meaningful decisions and should not repeat
the same move in several puzzle-like actions. Dense MCO analysis belongs in
optional Reference activities or the variation explorer. This preserves the
lesson redesign goal: courses should feel like lessons, not reused tactics
puzzles.

## Checkpoints

Before modifying private Caro-Kann files, create a recoverable checkpoint of
the current v1 seed pack:

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

- `caro-kann-black-v1-before-full-reference/`
- `caro-kann-black-v2-full-reference/`

## Import and Rebase Verification

After validation, import the new private pack through the existing course
import path. The import should:

- replace the active generation for `mco15-caro-kann-black`;
- not replace Italian or Ruy Lopez;
- preserve stable lesson progress where IDs and activity meanings are
  compatible;
- use the existing restart-from-checkpoint behavior if a session cannot be
  rebased cleanly.

Manual or scripted verification should confirm:

- Learn Openings still lists Italian White, Ruy Lopez White, and Caro-Kann
  Black;
- Caro-Kann board orientation is Black;
- at least one Quick, one Standard, and one Reference Caro-Kann lesson can be
  started;
- one passive activity and one decision activity can be completed;
- the variation explorer loads from the course root and from at least one
  deeper Reference branch;
- Continue Learning remains course-bound.

## App Code Policy

This is primarily private content authoring. Public app code should change
only if the deeper Caro-Kann pack exposes a generic defect. Any such change
must be:

- generic, not Caro-Kann-specific;
- covered by a failing synthetic test first;
- validated against existing Italian, Ruy, and Caro courses;
- free of private course prose or MCO-derived data.

## Testing Strategy

Private course checks:

- validate the current v1 pack before editing;
- validate the v2 pack after each substantial authoring batch;
- record validation counts and coverage status in private checkpoint notes;
- verify checkpoint hashes.

Repository checks if app code changes:

- focused Go tests for compiler/service behavior;
- focused frontend tests for any UI behavior;
- release policy check that no private `.ctcourse` is tracked.

Final verification:

- `go test ./... -count=1`;
- `go vet ./...`;
- `npm --prefix frontend test -- --run --single-thread`;
- `npm --prefix frontend run check`;
- `npm --prefix frontend run build`;
- `npm --prefix frontend run verify:licenses`;
- opening e2e checks where relevant;
- Wails rebuild and codesign verification.

## Risks and Mitigations

- Full visual inventory is time-consuming. Mitigate with page-by-page
  checkpoints and frequent validation.
- The scan may be hard to read. Mitigate by rendering pages at sufficient
  resolution and recording any illegible source items honestly.
- A deeper graph may create illegal move or depth-visibility mistakes. Mitigate
  with small authoring batches and the existing strict validator.
- Preserving IDs can conflict with a cleaner tree. Mitigate by keeping stable
  IDs where meanings match and adding new Reference-only lessons for distinct
  branches.
- Reference material can overwhelm lessons. Mitigate by keeping dense content
  optional and using the variation explorer for source-tree browsing.

## Handoff

After this spec is approved, write an implementation plan for the Caro-Kann
full-reference pass. The plan should prioritize private checkpoints, visual
inventory, incremental pack generation, validation, import/rebase checks, and
final app rebuild.
