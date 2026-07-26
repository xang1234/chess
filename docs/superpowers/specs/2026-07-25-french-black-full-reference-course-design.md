# French Defence for Black full-reference course design

Status: implemented; automated verification passed; live manual Learn Openings acceptance outstanding
Date: 2026-07-25

## Goal

Create a private, full-reference opening course that teaches Black how to meet
`1.e4` with the French Defence.

The course should be practical first and reference-rich second. Quick and
Standard depth should give the learner a solid Classical French repertoire, while
Reference depth exposes the wider French family as a source-backed teaching tree.

The course should appear as a Black-perspective course in the app alongside the
existing Italian White, Ruy Lopez White, Queen's Gambit White, Caro-Kann Black,
Queen's Gambit Defences for Black, and Sicilian Defence for Black courses.

## Course identity

- Course title: `French Defence for Black`
- Course ID: `mco15-french-black`
- Content version: `1.0.0`
- Perspective: `black`
- Default depth: `reference`
- Authoring file: `/Users/admin/Documents/Private Chess Courses/mco15-french-black.authoring.json`
- Course pack: `/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse`

Private source-derived authoring data, rendered pages, course packs, table
lines, note text, and import outputs must remain outside Git.

## Source strategy

Use the user's local MCO-15 PDF only for private manual course authoring and
source-coordinate validation.

The visually verified French Defence source scope is:

- Printed pages: `197-243`
- PDF pages: `214-260`
- Scan offset: `+17`
- Page count: `47`
- The section starts at the French Defence title page and ends immediately before
  the Sicilian Defence title page.

The repository may contain this design, future implementation plans, generic
tooling, validators, synthetic fixtures, and tests. It must not contain
transcribed MCO-derived French course content.

## Product decisions

The user selected:

- Next opening: French Defence for Black.
- Scope: full-reference French course.
- Main Black repertoire spine: Classical French.
- Quick and Standard emphasis: solid Classical setup, with Winawer, MacCutcheon,
  and Burn treated as important branches rather than the first learner path.

This means the course should not be a pure Winawer theory maze. The learner
should first understand why Black plays `...e6` and `...d5`, how the locked
centre shapes both players' plans, why the light-square bishop is a recurring
problem, and how Black frees the position with breaks such as `...c5` and
`...f6`.

## Goals

- Add a serious Black course against `1.e4` with a distinct strategic identity
  from Caro-Kann and Sicilian.
- Teach the French as a defensive decision map, not as repeated opening-stem
  replay.
- Give Quick learners a playable Classical French route.
- Give Standard learners coverage against Advance, Exchange, Tarrasch,
  Classical, and Winawer structures.
- Give Reference learners a browseable source-backed French tree across the full
  selected source scope.
- Keep every required activity meaningful and avoid repeated prompts from the
  same position inside one lesson.
- Preserve strict private source coverage validation.
- Keep all MCO-derived course artifacts private and out of Git.

## Non-goals

- Do not import or OCR the full MCO-15 PDF.
- Do not add White-side anti-French repertoire material as a separate course.
- Do not add Pirc, Alekhine, King's Indian, English, Dutch, or other openings in
  this pass.
- Do not bundle private course packs with the app.
- Do not build an in-app course editor.
- Do not create a second curriculum engine.
- Do not special-case French in app code.
- Do not replace tactical puzzles or opening spaced review.

## Course tree

Organize the course by Black's defensive decisions and the pawn structures White
chooses:

```text
French Defence for Black
├── Foundations
│   ├── Why ...e6 and ...d5
│   ├── The locked centre
│   ├── The light-square bishop problem
│   ├── Black's freeing breaks
│   └── French structure checkpoint
├── Classical main spine
│   ├── Reach the Classical French
│   ├── 3.Nc3 Nf6
│   ├── Solid development with ...Be7
│   ├── Pressure on e5 and d4
│   ├── Timing ...c5
│   ├── Castling and piece coordination
│   └── Quick repertoire checkpoint
├── Major White systems
│   ├── Advance Variation
│   ├── Exchange Variation
│   ├── Tarrasch Variation
│   ├── Classical branches
│   └── Winawer as a serious branch
├── Classical reference branches
│   ├── MacCutcheon choices
│   ├── Burn structures
│   ├── early Bg5 systems
│   ├── Steinitz-style closed structures
│   └── rare Classical move orders
└── Reference map
    ├── Winawer sub-branches
    ├── Advance sidelines
    ├── Tarrasch sidelines
    ├── Exchange structures
    ├── early deviations
    └── source-note and illustrative-game material
```

The root and Quick path should stay Classical-centered. Other French families
should be visible as branches, not forced into the main Quick route.

## Depth model

Use this depth model:

- Quick: about 7-9 required lessons.
  - Explain `...e6` and `...d5`.
  - Teach the locked-centre structure.
  - Reach `1.e4 e6 2.d4 d5 3.Nc3 Nf6`.
  - Show the solid Classical setup with `...Be7`, castling, and central pressure.
  - Teach one practical answer each to Advance, Exchange, and Tarrasch so the
    learner is not stranded when White avoids `3.Nc3`.

- Standard: about 16-22 required lessons.
  - Add the main Classical branches.
  - Add Winawer recognition and practical branching.
  - Add MacCutcheon and Burn recognition inside the Classical family.
  - Deepen Advance, Exchange, and Tarrasch structure lessons.
  - Teach recurring French decisions: when to break with `...c5`, when `...f6`
    matters, when to solve or accept the light-square bishop problem, and when a
    pawn-chain position calls for patience rather than immediate tactics.

- Reference: the full selected French source scope.
  - Capture source-backed table lines, note-letter material, named variations,
    illustrative references, move-order details, and rare branches.
  - Keep dense material optional unless it directly teaches a critical Black
    decision.
  - Prefer tree-shaped browsing over a long linear reference scroll.

Reference depth can be large, but it should remain navigable. It should not
become screen spam or a sequence of raw source-note placeholders.

## Main Classical design

The main learning spine should teach the Classical French as Black's practical
route:

```text
1.e4 e6
2.d4 d5
3.Nc3 Nf6
```

The Quick and Standard route should explain:

- why Black challenges the centre with a pawn chain instead of immediate open
  play;
- why the e6-d5 structure often gives Black less space but clear counterplay;
- why the light-square bishop is a strategic cost that must be managed;
- why `...Nf6` pressures e4/e5 structures;
- why `...Be7`, castling, and `...c5` form a coherent solid setup;
- how Black decides between immediate central pressure and patient piece
  improvement;
- when Black should treat White's e5 pawn as a target and when to undermine the
  base of the pawn chain.

The course should make the French feel like a set of recurring strategic
questions, not memorization of one brittle line.

## Major family design

### Advance Variation

Teach the locked pawn-chain race. Black should learn where the base of the chain
is, why `...c5` is central to the position, and when queenside pressure matters.
Quick should include a survival map; Standard should add typical piece
placement and break timing; Reference should include source branches and
illustrative games.

### Exchange Variation

Teach symmetrical structures without dismissing them as boring. Black should
learn how to equalize calmly, avoid drifting, and create useful piece play after
the central tension resolves. This branch should be compact in Quick and
Standard, with Reference carrying the rarer move-order details.

### Tarrasch Variation

Teach why `3.Nd2` changes the character of the French. Black should understand
how it avoids some `...Bb4` ideas, what it concedes, and how Black keeps pressure
with central breaks and development. Standard should include practical structure
choices; Reference should include deeper source lines.

### Winawer Variation

Winawer is a major branch, not the first learner spine. It should be treated
seriously because it is central to French theory, but its sharp queen-side and
king-side imbalances should arrive after the learner understands the French
structure. Quick may introduce recognition; Standard should teach the strategic
tradeoff; Reference should carry the dense theory.

### MacCutcheon and Burn

MacCutcheon and Burn belong inside the Classical reference family. Standard
should explain why Black might choose each route. Reference should include the
concrete table lines and note material from the selected source scope.

## Lesson experience

The course should feel like a defensive decision map:

```text
White claims space -> Black identifies structure -> Black chooses a break or
development plan -> the board demonstrates the consequence -> recap -> optional
Reference
```

Lesson constraints:

- Board orientation defaults to Black.
- Black repertoire moves are learner decisions.
- White moves are opponent pressure, demonstrations, comparisons, or branch
  choices.
- Quick lessons usually have 2-3 required activities.
- Standard lessons usually have 3-4 required activities.
- Reference activities are optional by default.
- Each lesson teaches a distinct decision or concept.
- Required decisions should not repeat the same prompt from the same position
  inside one lesson.
- Dense source material should become original explanation, comparison, or
  optional reference cards.
- Do not add generic filler such as `Source note x records...`, raw
  activity-result JSON, or `Course move found`.

## Application changes

First try to import the French course without app runtime changes. The current
opening-course engine already supports multiple courses, Black perspective,
teaching trees, journey persistence, depth selection, variation exploration, and
review scheduling.

Make app changes only if the French pack exposes a generic defect. Any such
change must be:

- generic, not French-specific;
- validated against existing private courses;
- covered by synthetic repository tests where appropriate.

Likely generic checks:

- The opening hub groups French under Black openings.
- Course progress and continuation state stay course-specific.
- Black board orientation and learner move ownership are correct.
- Deep Reference branches remain scrollable and do not push the board out of
  view.

## Data flow

1. A private source-scope inventory records the French printed pages, PDF pages,
   table regions, named variations, note labels, and source coverage IDs.
2. A private authoring JSON file captures the teaching tree, moves, lessons,
   activities, and original instructional prose.
3. Course-generation tooling emits a schema-v2 `.ctcourse` pack.
4. The course-pack validator checks graph legality, legal moves, depth
   visibility, activity shape, teaching-tree connectivity, duplicate decisions,
   warning spam, and source coverage.
5. The app imports the pack into the replaceable course catalogue.
6. `OpeningService.Home` returns the French course alongside existing courses.
7. Lessons, exploration, progress, journey state, and review scheduling continue
   to use existing course ID boundaries.

## Error handling

- Invalid private packs must fail validation before import.
- Missing source coverage must block pack activation.
- Illegal moves, unreachable positions, duplicate outgoing moves, inconsistent
  transpositions, repeated required prompts, and malformed activities must block
  pack activation.
- If the French import fails, existing active courses remain usable.
- If a future French update cannot rebase a resumable journey, the existing
  restart-from-checkpoint behavior applies to the French course only.
- If a source boundary check discovers adjacent French material outside printed
  pages 197-243, update the source-scope record before authoring the pack.

## Test strategy

### Private course-pack validation

- Validate the French pack with `go run ./cmd/coursepack validate`.
- Confirm missing and unexpected coverage counts are zero.
- Confirm warnings are zero unless a warning is deliberately learner-facing.
- Run a private anti-spam audit to catch placeholder note text, raw JSON, and
  repeated generic messages.
- Run a private role audit to ensure Black repertoire decisions are learner
  moves and White continuations are opponent or demonstration moves.

### Repository tests

Add or extend synthetic tests only if current coverage is insufficient. Tests
should prove generic behavior, not French-specific content:

- Black-perspective course orientation works.
- Multiple Black courses coexist without journey or review collisions.
- Deep Reference trees remain usable.
- Activity results require correct shapes such as array-valued applied moves.
- Course picker, Continue Learning, and depth preferences remain course-bound.

### Manual app verification

- Import the French pack together with the existing private course library.
- Confirm `French Defence for Black` appears under Black openings.
- Start Quick, Standard, and Reference French lessons.
- Complete at least one French lesson and confirm journey progress persists.
- Confirm Continue Learning resumes the French course when French owns the
  active session.
- Open the French variation explorer from the root and from a deeper lesson.
- Confirm Reference branch layout remains scrollable without hiding the board.

## Acceptance criteria

- A private schema-v2 course pack exists and validates:
  `/Users/admin/Documents/Private Chess Courses/mco15-french-black.ctcourse`.
- The private course uses ID `mco15-french-black`, version `1.0.0`, and Black
  perspective.
- The source scope covers printed pages `197-243` / PDF pages `214-260`.
- Quick teaches a playable Classical French repertoire.
- Standard covers the main French family decisions: Advance, Exchange, Tarrasch,
  Classical, Winawer, MacCutcheon, and Burn.
- Reference exposes the full selected French source scope as a browseable
  teaching tree.
- Required lesson activities do not repeat the same learner prompt from the same
  position inside one lesson.
- The app imports French without disrupting existing active courses.
- No MCO-derived private course content is added to the repository.
- Required validation, repository tests, and manual app checks pass before the
  implementation is considered complete.
