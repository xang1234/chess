# Opening Course Library Expansion Design

Date: 2026-07-22
Status: Approved direction, awaiting written-spec review

## Summary

Expand Learn Openings from a single Italian White course into a small private
opening library by adding two new schema-v2 course packs:

- `mco15-ruy-lopez-white` for playing the Ruy Lopez as White.
- `mco15-caro-kann-black` for answering `1.e4` with the Caro-Kann as Black.

This is the smallest useful expansion batch. It proves that the redesigned
teaching-tree engine works for multiple courses, for both learner perspectives,
and for a Black repertoire where the learner often responds to White's opening
choice. It deliberately avoids a full-book import or a large catalogue rollout.

The application runtime should be reused as much as possible. The new work is
primarily private course authoring plus small multi-course polish in the
opening hub if the current UI needs it.

## Context

The current app already supports multiple active opening courses, selected
course depth, course-specific journeys, teaching trees, lesson sessions,
variation exploration, and opening review scheduling. The Italian White v2 pack
validates as schema version 2 with 23 teaching-tree lessons and demonstrates
the desired lesson shape.

The MCO-15 PDF is an image-based scan. Text extraction is empty and the book has
no machine-readable outline, so expansion continues to use manual visual
curation rather than OCR. The table of contents provides clear printed-page
ranges for the pilot:

- Ruy Lopez: printed pages 42-95.
- Caro-Kann Defence: printed pages 171-196.

Private course data remains outside the repository under
`/Users/admin/Documents/Private Chess Courses/`. The repository may contain
specs, tooling, validators, synthetic fixtures, and tests, but not transcribed
MCO-derived course content.

## Approved Product Decisions

- Expand with option C: exactly two new courses first.
- Use Ruy Lopez as the next White course.
- Use Caro-Kann as the first Black course.
- Preserve the Italian v2 teaching-tree model: meaningful teaching nodes,
  flexible activities, Quick / Standard / Reference depth, and optional
  reference material.
- Reuse the existing importer, validator, catalogue, lesson UI, variation
  explorer, journey persistence, and spaced review wherever possible.
- Make only targeted app changes needed for a multi-course library experience.
- Author course prose manually and originally; dense source detail remains
  optional reference material.

## Goals

- Add one practical White repertoire course beyond Italian.
- Add one practical Black repertoire course against `1.e4`.
- Prove White and Black course perspectives behave correctly in the hub,
  lessons, reviews, progress, and variation explorer.
- Keep each Quick course small enough to study as a usable repertoire.
- Keep Standard depth useful without turning the pilot into full reference
  transcription.
- Preserve Reference as the complete private source-backed graph for the
  authored scope.
- Maintain strict source coverage validation for every included page, column,
  note, and move path.
- Keep all MCO-derived material private and out of Git.

## Non-Goals

- Do not import or OCR the full MCO-15 PDF.
- Do not add Sicilian, French, Queen's Gambit, English, King's Indian, or other
  courses in this batch.
- Do not bundle private course packs with the app.
- Do not build an in-app course editor.
- Do not create a second curriculum engine.
- Do not replace tactical puzzles or opening spaced review.
- Do not require engine evaluation, cloud sync, or network access.

## Course Design

### Ruy Lopez for White

The Ruy Lopez course starts from:

`1.e4 e5 2.Nf3 Nc6 3.Bb5`

The teaching tree should favor practical White decisions first, then branch into
Black systems:

- foundations: why `Bb5` pressures Black's centre and shapes development;
- early Black choice: `...a6`, non-`...a6`, and immediate simplifications;
- Quick spine: a coherent White plan against the main `...a6` family;
- Standard branches: Exchange, Open, Berlin, Steinitz-style setups, and one
  main closed structure;
- Reference branches: deeper closed systems, Marshall and Anti-Marshall
  material, and rarer systems from the printed-page scope.

Quick depth should land around 6-8 lessons. Standard should land around 14-18
lessons. Reference may be larger, but required teaching nodes must stay
purposeful and should not mirror every table column as a separate exercise.

### Caro-Kann for Black

The Caro-Kann course starts from:

`1.e4 c6`

The teaching tree should teach Black's choices as the learner's repertoire:

- foundations: the `...c6` and `...d5` structure, and why Black accepts a
  slower first move for solidity;
- Quick spine: one reliable Black setup against White's main central choices;
- Standard branches: Advance, Exchange, Panov-Botvinnik, Classical, and common
  sideline systems;
- practical decisions: when Black frees with `...c5`, when the light-square
  bishop develops, when to accept structural targets, and when to trade into a
  stable endgame;
- Reference branches: sharper or less common source lines within printed pages
  171-196.

Quick depth should also land around 6-8 lessons. Standard should land around
14-18 lessons. The Black perspective must be tested carefully because the
learner's repertoire moves are Black moves, while White's moves are opponent
continuations.

## Authoring Workflow

Each course follows the Italian v2 private-authoring pattern:

1. Create a recoverable checkpoint before editing private files.
2. Inventory the source scope visually from the PDF:
   - printed pages;
   - table columns;
   - named variations;
   - note labels;
   - illustrative references;
   - candidate teaching nodes.
3. Create a private authoring JSON file:
   - `mco15-ruy-lopez-white.authoring.json`;
   - `mco15-caro-kann-black.authoring.json`.
4. Author the teaching-tree manifest before writing detailed activities.
5. Build the Quick path first and validate it.
6. Add Standard branches and validate again.
7. Add Reference-only coverage and optional reference activities.
8. Generate the `.ctcourse` pack.
9. Validate with `go run ./cmd/coursepack validate`.
10. Import into the app and verify the learner journey manually.

The resulting private packs are:

- `/Users/admin/Documents/Private Chess Courses/mco15-ruy-lopez-white.ctcourse`
- `/Users/admin/Documents/Private Chess Courses/mco15-caro-kann-black.ctcourse`

## Application Changes

The implementation should first try to import the two new packs without app
runtime changes. If the experience is awkward, make small improvements only:

- group or label courses clearly by perspective in the opening hub;
- ensure course picker copy works when multiple courses exist;
- ensure Continue Learning resumes the selected course rather than another
  course with a resumable session;
- ensure progress, review due state, and depth are visibly course-specific;
- keep the current teaching-tree and lesson screens generic.

No app code should special-case Ruy Lopez or Caro-Kann by name. Repository tests
should use synthetic courses that prove the generic multi-course behavior.

## Data Flow

1. The private authoring file captures source inventory and teaching intent.
2. Course-generation tooling emits a schema-v2 `.ctcourse`.
3. The existing course-pack validator checks graph legality, depth visibility,
   activity shape, teaching-tree connectivity, duplicate decisions, and source
   coverage.
4. The app imports each pack into the replaceable course catalogue.
5. `OpeningService.Home` returns multiple course summaries.
6. `OpeningHub` lets the learner select a course and depth.
7. Lessons, exploration, progress, journey state, and review scheduling continue
   to use the existing course ID boundaries.

## Error Handling

- Invalid private packs must fail validation before import.
- Missing source coverage must block pack activation.
- Illegal moves, unreachable positions, duplicate outgoing moves, inconsistent
  transpositions, and duplicate required decisions must block pack activation.
- If one course pack fails import, existing active courses remain usable.
- If a course update cannot rebase a resumable journey, the existing pause and
  restart-from-checkpoint behavior applies to that course only.

## Test Strategy

### Course-pack validation

- Validate Ruy Lopez and Caro-Kann packs privately before import.
- Record counts, printed-page coverage, source reference coverage, lessons,
  prompts, and activities in private checkpoint notes.
- Keep generated private content out of Git status and Git history.

### Repository tests

- Add or extend synthetic multi-course tests only if current coverage is
  insufficient.
- Prove that two courses with different perspectives can coexist.
- Prove depth preferences, journeys, progress, and reviews remain course-bound.
- Prove the opening hub keeps the selected course stable.
- Prove Black-perspective learner decisions are treated as repertoire moves.

### Manual app verification

- Import Italian White, Ruy Lopez White, and Caro-Kann Black together.
- Confirm all three courses appear and display the right perspective labels.
- Start and complete at least one lesson in each new course.
- Confirm Continue Learning resumes the intended selected course.
- Confirm variation explorer opens from each course root.
- Confirm opening reviews are scheduled separately per course.

## Acceptance Criteria

- Two new private schema-v2 course packs exist and validate:
  `mco15-ruy-lopez-white` and `mco15-caro-kann-black`.
- The app can import and show Italian White, Ruy Lopez White, and Caro-Kann
  Black at the same time.
- White and Black perspectives are visible and behaviorally correct.
- Course progress, depth, journey, reviews, and continuation state stay separate
  by course ID.
- The lesson experience continues to use teaching trees and decision-point
  activities, not the old fixed five-step staircase.
- No MCO-derived course content is added to the repository.
- Required Go, frontend, validation, and manual app checks pass before the
  implementation is considered complete.
