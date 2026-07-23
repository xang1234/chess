# Queen's Gambit White Course Design

Date: 2026-07-24
Status: Approved for implementation planning

## Summary

Add a new private schema-v2 opening course:

- Course ID: `mco15-queens-gambit-white`
- Title: `Queen's Gambit for White`
- Perspective: `white`
- Target content version: `1.0.0`
- Source scope: the Queen's Gambit family from the double queen-pawn section of the MCO-15 PDF, starting at printed page 389 and ending before the next section.

This is a large v1 course, not a small teaser. It should cover the Queen's Gambit family only: Queen's Gambit Declined, Tarrasch Defence, Queen's Gambit Accepted, Slav/Semi-Slav, and Chigorin Defence. Offset verification found the section beginning on PDF page 406 and useful Queen's Gambit content ending before the next-section title/blank pages. The course should feel like a White repertoire first and a reference tree second.

The public app should be reused as much as possible. The implementation is primarily private course authoring plus validation/import work. App code changes are allowed only for generic defects discovered while importing or using the new course.

## Product Decisions

- Build option A from the opening choice: Queen's Gambit White next.
- Scope option A: Queen's Gambit family only, not the full `1.d4` universe.
- Course-shape option A: balanced Quick, Standard, and Reference depth across all five selected families.
- Quick-depth option A: White repertoire spine.
- Size option B: large v1, roughly 38-46 total lessons.
- Keep dense table lines, rare variations, and source-specific notes available through optional Reference activities and the variation explorer instead of forcing them into required lesson flow.

## Goals

- Give the learner a serious `1.d4 d5 2.c4` White course alongside Italian White and Ruy Lopez White.
- Teach Queen's Gambit ideas as a playable White repertoire, not as a flat survey of book columns.
- Cover the five Queen's Gambit family responses in the selected PDF scope.
- Preserve the existing teaching-tree journey model: visible branch progress, seamless lesson continuation, depth-specific projection, and course-specific saved progress.
- Retain full selected-source coverage privately while keeping required activities lean.
- Import and run alongside the existing Italian White, Ruy Lopez White, and Caro-Kann Black courses.

## Non-Goals

- Do not add King's Indian, Nimzo-Indian, Queen's Indian, Grunfeld, Dutch, Budapest, English, Catalan, or other non-Queen's-Gambit courses in this spec.
- Do not OCR or bulk-import the PDF.
- Do not bundle private course packs with the app.
- Do not create an in-app course editor.
- Do not special-case Queen's Gambit in app code.
- Do not commit private `.ctcourse`, private authoring JSON, rendered PDF pages, checkpoints, or MCO-derived course prose to Git.

## Course Architecture

The new course is one private schema-v2 pack:

- `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json`
- `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse`

It should follow the established Italian/Ruy/Caro course pattern:

- stable lesson IDs, activity IDs, prompt IDs, move IDs, position IDs, and note IDs;
- `defaultDepth: "reference"`;
- `perspective: "white"`;
- Quick, Standard, and Reference depth projection;
- a teaching tree rooted at the Queen's Gambit starting idea;
- required activities that teach one job at a time;
- optional reference activities and variation explorer branches for dense source coverage.

No public runtime structure should depend on the course ID or opening name. Any public fixes must be generic and covered by synthetic tests.

## Lesson Structure

Target size:

- Quick: 7-9 visible lessons.
- Standard: about 22-28 visible lessons.
- Reference: about 38-46 total lessons.

Chapters:

1. Foundations
2. Queen's Gambit Declined
3. Queen's Gambit Accepted
4. Slav and Semi-Slav
5. Tarrasch and Chigorin

Teaching-tree sketch:

```text
Queen's Gambit Foundations
├─ Queen's Gambit Declined structures
│  ├─ Classical development
│  ├─ Exchange structures
│  ├─ Tactical pressure systems
│  └─ Piece-pressure systems
├─ Queen's Gambit Accepted
│  ├─ Regaining the pawn
│  └─ Central expansion
├─ Slav and Semi-Slav
│  ├─ Slav structure
│  ├─ Semi-Slav tension
│  └─ Meran and anti-Meran style decisions
├─ Tarrasch
│  ├─ Isolated queen's pawn themes
│  └─ White pressure against active pieces
└─ Chigorin
   └─ Early piece pressure and central response
```

Required-flow policy:

- Quick lessons should usually have 2-3 required ideas.
- Standard lessons should usually have 3-4 required ideas.
- Reference lessons may be denser, but must not force repeated decisions from the same position.
- Tactical multi-decision lessons are allowed only when the decisions are genuinely distinct and useful to the learner.
- Recaps, deep table lines, and historical/source detail should be optional when they do not advance the next learner decision.

## Quick Depth

Quick depth should be a playable White repertoire spine:

1. Establish `1.d4 d5 2.c4` as the course root.
2. Teach why White offers pressure on the centre rather than immediately resolving it.
3. Show how Black's major family choice changes White's next plan.
4. Give the learner one coherent practical White response against each major family.
5. Keep the learner moving through the tree without repeatedly returning to the course menu.

Quick depth is not a full survey. It should answer: "What do I play as White, and why does it fit the Queen's Gambit idea?"

## Standard Depth

Standard depth should be the practical map. It should add the most useful branches inside each family:

- QGD: classical development, exchange structures, and selected tactical/piece-pressure systems.
- QGA: pawn recovery, central expansion, and development tempo.
- Slav/Semi-Slav: structure, central tension, and practical decision points.
- Tarrasch: isolated queen's pawn structures and White's pressure plan.
- Chigorin: early piece pressure and White's central response.

Standard lessons should emphasize decision points and structure recognition rather than exhaustive memorization.

## Reference Depth

Reference depth should cover the selected printed-page scope more completely while preserving the lesson-first experience:

- include source-backed move graph branches for the selected pages;
- use optional reference activities for dense side notes;
- keep rare or very deep continuations available in the variation explorer;
- avoid generic warning text such as source-note spam;
- maintain strict source coverage for every included page, note, variation, and move path.

Reference can be substantial, but it should still be navigable as a tree.

## Authoring Workflow

Use the established private manual-curation workflow:

1. Create a private baseline checkpoint before adding the new course files.
2. Visually inventory the selected PDF pages: printed pages, family sections, table columns, note labels, named variations, and candidate teaching nodes.
3. Create the private authoring JSON.
4. Draft the teaching tree before writing detailed activities.
5. Build Quick depth first and validate.
6. Add Standard branches and validate.
7. Add Reference branches and optional material, then validate again.
8. Generate the private `.ctcourse` pack.
9. Import into a disposable app data root with the existing courses.
10. Import into the default app catalogue after validation passes.

Private checkpoints should be created before and after generation under:

`/Users/admin/Documents/Private Chess Courses/checkpoints/`

## Data Flow

1. Private visual source inventory informs the private authoring JSON.
2. Course-generation scripts or manual authoring produce a schema-v2 `.ctcourse`.
3. `go run ./cmd/coursepack validate` checks graph legality, activity shape, depth visibility, duplicate decisions, and source coverage.
4. The existing importer inspects and imports the pack into the replaceable course catalogue.
5. `OpeningService.Home` returns Queen's Gambit alongside the existing courses.
6. The current opening hub, lesson screen, teaching tree, journey persistence, checkpoints, reviews, and variation explorer operate by course ID.

## Error Handling

- Invalid course packs must fail validation before import.
- Missing or unexpected source coverage must block activation.
- Illegal moves, unreachable positions, duplicate outgoing moves, inconsistent transpositions, or duplicate required decisions must block activation.
- If Queen's Gambit import fails, existing active courses must remain usable.
- If a future Queen's Gambit update cannot rebase a resumable journey, the existing restart-required course behavior applies only to that course.

## Testing and Acceptance

Private validation must confirm:

- course ID is `mco15-queens-gambit-white`;
- perspective is `white`;
- content version is `1.0.0`;
- zero missing coverage;
- zero unexpected coverage;
- legal move graph;
- no repeated forced decisions from the same position;
- no warning-spam generic text.

App import acceptance:

- Italian White, Ruy Lopez White, Queen's Gambit White, and Caro-Kann Black can be active together.
- Course picker shows Queen's Gambit correctly.
- Depth selection works for Queen's Gambit.
- Continue Learning remains course-specific.
- The teaching tree displays branch progress.
- Lesson completion advances seamlessly.
- Variation explorer remains scrollable and keeps the board visible on deep branches.

Repository verification:

- `go test ./...`
- `go vet ./...`
- `npm --prefix frontend run check`
- `npm --prefix frontend test -- --run --single-thread`
- opening E2E tests for Chromium and WebKit
- frontend build
- Wails rebuild when implementation is complete or explicitly requested

## Privacy Boundaries

The public repository may contain:

- this design spec;
- a future implementation plan;
- generic scripts/tests/fixes that do not contain private source material;
- synthetic `.ctcourse` fixtures only.

The public repository must not contain:

- `mco15-queens-gambit-white.ctcourse`;
- `mco15-queens-gambit-white.authoring.json`;
- rendered PDF pages;
- private checkpoints;
- private edit manifests;
- MCO-derived course prose or dense table transcription.
