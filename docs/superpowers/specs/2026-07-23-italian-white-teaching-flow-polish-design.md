# Italian White Teaching-Flow Polish Design

**Date:** 2026-07-23  
**Branch:** `codex/italian-white-teaching-flow-polish`  
**Status:** Approved design, pending implementation plan

## Purpose

Improve the existing private Italian Game for White course so it feels like a
guided opening lesson path instead of a puzzle-like sequence of repeated moves.

The current private pack, `mco15-italian-white`, is already broad enough for
this pass: version `2.0.0` validates with 23 lessons, 83 activities, 636 moves,
251 notes, 20 prompts, 48 variations, and no validation warnings. The problem to
solve is pacing, continuity, and lesson shape, not raw source coverage.

## User Experience Goal

The learner should experience Italian White as a continuous teaching tree:

- foundations explain what the Italian is and why White develops this way;
- Giuoco Piano teaches the main calm centre-building choices;
- Evans Gambit is a clearly labelled aggressive alternative;
- Two Knights' Defence is a clearly labelled tactical branch;
- completing one lesson naturally leads to the next recommended lesson;
- the course tree remains the learner's map, not a menu they repeatedly forget.

The app should ask the learner to make a move only when the move is genuinely
the point of the lesson. Extra repetitions should become demonstrations,
comparisons, recaps, or Reference-only analysis.

## Scope

### In scope

- Polish the existing Italian White private course pack and authoring inventory.
- Keep the existing course ID: `mco15-italian-white`.
- Preserve stable lesson IDs, prompt IDs, and activity IDs wherever safe so
  existing journey and review state can rebase.
- Reduce duplicate required decision activities that ask for the same obvious
  move within a lesson.
- Rebalance activities so each lesson normally has one primary student decision
  plus supporting teaching material.
- Improve lesson titles, objectives, instructions, summaries, and reference
  sections where they affect learner clarity.
- Tune Quick, Standard, and Reference depth so each depth has a distinct role.
- Validate and re-import the private course locally.
- Verify Italian still coexists with Ruy Lopez White and Caro-Kann Black.
- Rebuild the macOS app after import.

### Out of scope

- Adding a new opening.
- Expanding Italian beyond the existing MCO-15 Italian source-page scope.
- Rewriting the opening-course engine, schema, or app navigation unless a small
  generic defect blocks the teaching-flow polish.
- Committing private `.ctcourse`, private authoring JSON, rendered PDF pages, or
  MCO-derived prose into Git.
- Publishing source-derived lesson content in public tests or docs.

## Course Design Principles

### One lesson, one job

Each teaching node should have one clear instructional purpose. If a lesson
currently asks the learner to make the same move repeatedly, keep the best
decision activity and convert the remaining repetitions into:

- a passive demonstration;
- a comparison card;
- a recap;
- a collapsed Reference section; or
- an optional Reference activity.

### Required activities stay lean

Required activities should be the learner's path through the course. Reference
material should enrich that path without bloating it.

Target shape:

- Quick lessons: 2-3 required ideas each.
- Standard lessons: 3-4 required ideas each.
- Tactical branches may use 4-5 required ideas only when each asks a distinct
  decision or recognition question.
- Reference activities are optional and should not block course completion.

### Depths have distinct jobs

- **Quick:** a coherent beginner White repertoire through the Italian, with
  minimal branching and low repetition.
- **Standard:** the practical Italian map, including Evans and major Two
  Knights/Giuoco choices.
- **Reference:** complete existing source coverage through deeper analysis and
  the variation explorer, without forcing every source line into the learner's
  required path.

### The tree should teach memory

The tree is not only navigation. It should help the learner remember how choices
branch:

- shared Italian foundations at the root;
- the `...Bc5` / `...Nf6` family split remains obvious;
- Giuoco, Evans, and Two Knights nodes should visually and textually read as
  related branches, not unrelated mini-puzzles;
- checkpoints should show the path just learned and provide a direct next-step
  button.

## Proposed Course Changes

This pass starts from the current Italian v2 course and edits the private pack.
The exact activity edits will be decided by audit, but the intended shape is:

1. Keep the 23-lesson teaching tree unless the audit finds adjacent nodes that
   clearly duplicate one instructional job.
2. For each lesson, classify activities as:
   - primary decision;
   - explanation or demonstration;
   - comparison;
   - recap;
   - optional Reference.
3. Keep only distinct required learner decisions.
4. Move repeated move-entry drills out of required flow.
5. Strengthen the transition copy between foundations, Giuoco, Evans, and Two
   Knights so the learner sees why the next branch matters.
6. Keep existing Reference coverage intact unless a validation problem or
   duplicated spam text is found.

## Data and Architecture

No new public schema is expected.

Private course artifacts remain under:

- `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.ctcourse`
- `/Users/admin/Documents/Private Chess Courses/mco15-italian-white.authoring.json`
- `/Users/admin/Documents/Private Chess Courses/checkpoints/...`

The app continues using the existing opening-course architecture:

- strict course-pack validation;
- course catalogue import and replacement;
- persisted journey and lesson progress;
- depth-specific tree projection;
- opening session rebase across course generations;
- generic Svelte course hub, lesson screen, and variation explorer.

If implementation discovers a generic app defect while polishing Italian, the
fix may be made only if it is small, course-agnostic, and tested with synthetic
fixtures. Course-specific content remains private.

## Progress and Rebase Requirements

The update should be safe for existing Italian learners:

- preserve the course ID;
- preserve compatible lesson IDs and required activity IDs where possible;
- import as a replacement generation;
- verify active sessions either rebase cleanly or enter the existing explicit
  restart-required flow;
- verify completed lesson progress is not revoked by optional Reference edits.

If an activity ID must change because its meaning changes materially, document
the reason in the private task report and ensure the restart/rebase behavior is
intentional.

## Error Handling

- A failed private validation must stop the import and leave the active Italian
  course unchanged.
- A failed replacement import must not affect Ruy Lopez or Caro-Kann.
- Any incompatible active Italian session should use the existing restart flow,
  not silently skip or complete content.
- If Reference material is retained but hidden at lower depth, Quick and
  Standard progress should remain stable.

## Testing and Acceptance

### Private course acceptance

- Validate `mco15-italian-white.ctcourse` with zero missing or unexpected
  coverage references.
- Confirm counts remain in the expected range for a polish pass:
  - keep 23 lessons by default; allow 20-22 only when the private audit shows
    adjacent lessons duplicate one instructional job and the merge preserves
    stable progress safely;
  - moves and notes remain broadly comparable to current v2;
  - required activities decrease or become less repetitive;
  - optional Reference material remains available.
- Run a lesson-quality script that flags:
  - repeated required moves within one lesson;
  - repeated generic warning text;
  - lessons with too many required prompts;
  - passive activities missing `appliedMoves` arrays where demonstrations use
    replayed moves.

### App acceptance

- Import the polished Italian course into the local catalogue.
- Confirm Italian, Ruy Lopez, and Caro-Kann are all still active.
- Confirm Italian Quick, Standard, and Reference depth render correctly.
- Complete a foundation-to-Giuoco path without returning to the home menu.
- Complete or pause/resume one Evans or Two Knights branch.
- Verify Continue Learning returns to the current Italian lesson.
- Verify the variation explorer still exposes the Italian reference tree and
  keeps the board visible on deep branches.
- Restart the rebuilt app and confirm the updated Italian course is present.

### Public repository acceptance

- Public commits contain only generic code/tests/docs.
- No private `.ctcourse`, authoring JSON, rendered PDF pages, or source-derived
  prose are tracked by Git.
- Synthetic fixtures remain original and non-source-derived.
- Run the relevant Go, frontend, e2e, validation, and rebuild checks before
  merge.

## Success Criteria

Italian White feels like a lesson path:

- fewer forced repeat moves;
- clearer branch memory;
- smoother continuation from lesson to lesson;
- Quick is beginner-friendly;
- Standard is practical;
- Reference is rich but not noisy;
- current user journey survives the replacement generation where compatible;
- the rebuilt app opens the polished Italian course automatically from the local
  catalogue.

## Non-Goals and Future Work

This pass does not add Sicilian, French, Queen's Gambit, or Black-side Italian
courses. After Italian White is polished, the next content choice should be a
new opening only if the existing three-course library feels balanced enough.

Future work may add a more formal authoring linter for “one lesson, one job,”
but this pass can begin with private audit scripts and targeted course edits.
