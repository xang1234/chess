# Sicilian Defence for Black full-reference course design

Status: approved in chat and implemented as private v1.0.0 local course pack
Date: 2026-07-24

## Goal

Create a private, full-reference opening course that teaches Black how to meet
`1.e4` with the Sicilian Defence.

The course should be broad at Reference depth, but practical at Quick and
Standard depth. The learner should get one coherent Black repertoire first,
centered on the Najdorf, while Reference depth exposes the larger Sicilian
family as a source-backed tree.

The course should appear as a Black-perspective course in the app alongside the
existing Italian White, Ruy Lopez White, Queen's Gambit White, Caro-Kann Black,
and Queen's Gambit Defences for Black courses.

## Course identity

- Course title: `Sicilian Defence for Black`
- Course ID: `mco15-sicilian-black`
- Content version: `1.0.0`
- Perspective: `black`
- Default depth: `reference`
- Authoring file: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json`
- Course pack: `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse`

Private source-derived authoring data, rendered pages, course packs, table
lines, and note text must remain outside Git.

## Source strategy

Use the user's local MCO-15 PDF only for private manual course authoring and
source-coordinate validation.

The source scope is the Sicilian Defence section:

- Printed pages: `244-361`
- Current PDF-page mapping: `261-378`, using the same `+17` scan offset already
  verified in previous Caro-Kann and Queen's Gambit course work.
- The section begins after the French Defence and ends before the Pirc Defence.

The rendered table-of-contents check identifies these main families inside the
selected scope:

- Sicilian overview and foundations
- Najdorf Variation
- Dragon and Accelerated Dragon
- Scheveningen Variation
- `...e6` systems, including Taimanov, Paulsen, and Four Knights themes
- Classical Sicilian systems
- `...Nc6` and `...e5` systems, including Sveshnikov, Kalashnikov, and
  Lowenthal themes
- Non-open Sicilians and anti-Sicilians, including Closed Sicilian, Alapin,
  early Bb5 systems, f4 Attack, Smith-Morra Gambit, and other unusual second
  moves

The implementation should inventory all of this source scope privately before
building the pack. The public repository may contain this design, implementation
plans, generic tooling, and synthetic tests, but not transcribed MCO-derived
content.

## Product decision

The user selected:

- Next opening: Sicilian Defence for Black.
- Main spine: Najdorf-centered Black repertoire.
- Scope: option C, full Sicilian Reference coverage.

That means the course is not a tiny Najdorf teaser. It should be a serious
Sicilian course. The design compromise is:

- Quick teaches a playable Black route.
- Standard teaches the main practical decisions and common deviations.
- Reference contains the large source-backed Sicilian map.

Quick and Standard should never feel like a required march through every source
table.

## Course tree

The course tree should be organized by the learner's Black-side decisions:

```text
Sicilian Defence for Black
├── Foundations
│   ├── Why Black plays ...c5
│   ├── Open Sicilian map
│   ├── Anti-Sicilian survival map
│   └── Move-order survival guide
├── Najdorf main spine
│   ├── Reach the Najdorf
│   ├── Why ...a6 matters
│   ├── Development setup and central tension
│   ├── White's main attacking tries
│   ├── Black counterplay themes
│   └── End of Quick repertoire checkpoint
├── Najdorf reference branches
│   ├── Be3 / English Attack structures
│   ├── Bg5 pressure systems
│   ├── Be2 and classical development
│   ├── Bc4 / Sozin-style pressure
│   ├── f4 and kingside expansion
│   └── quieter or rare White systems
├── Dragon and Accelerated Dragon
│   ├── Dragon structure
│   ├── Yugoslav Attack recognition
│   ├── Accelerated Dragon move-order ideas
│   └── Reference-only tactical branches
├── Scheveningen and ...e6 systems
│   ├── Scheveningen structure
│   ├── Taimanov / Paulsen recognition
│   ├── Four Knights themes
│   └── transpositions into Najdorf-style structures
├── Classical and ...Nc6/...e5 systems
│   ├── Classical Sicilian development
│   ├── Richter-Rauzer recognition
│   ├── Sveshnikov / Kalashnikov structures
│   └── Lowenthal-style reference material
└── Anti-Sicilians
    ├── Alapin
    ├── Closed Sicilian
    ├── Bb5 systems
    ├── Grand Prix / f4 Attack
    ├── Smith-Morra Gambit
    └── unusual second moves
```

The root and main continuation route should stay Najdorf-centered. Other
families should be exposed as alternative or reference branches, not forced
into the main Quick path.

## Depth model

Use this depth model:

- Quick: about 7-9 required lessons.
  - Explain `...c5`.
  - Reach the Open Sicilian.
  - Reach the Najdorf.
  - Teach why `...a6` is useful.
  - Give Black one practical plan against the most important White setups.
  - Include one anti-Sicilian survival lesson so the learner is not stranded if
    White avoids `2.Nf3` and `3.d4`.

- Standard: about 18-25 required lessons.
  - Add the main Najdorf branches.
  - Add common anti-Sicilians.
  - Add recognition lessons for Dragon, Scheveningen, Classical, and
    `...Nc6/...e5` systems so the learner understands the Sicilian map.
  - Keep family-recognition lessons compact unless the learner chooses Reference.

- Reference: likely 60+ lessons or nodes.
  - Capture the full selected Sicilian source scope.
  - Include dense table lines, named variations, note-letter material,
    illustrative game references, transpositions, and sharp tactical branches.
  - Keep dense material optional unless it teaches a critical Black decision.

Reference depth can be large, but it should be browseable and tree-shaped. It
should not become a long linear scroll of source notes.

## Lesson experience

The course should feel like a defensive decision map, not a puzzle grinder.

Preferred lesson rhythm:

```text
Concept → White pressure shown → Black decision → counterplay demonstration → recap → optional Reference
```

Black repertoire moves are the learner's decisions. White moves are opponent
pressure, demonstrations, or branch choices. Avoid asking the learner to replay
the same opening stem in every lesson.

Lesson constraints:

- Board orientation defaults to Black.
- Quick lessons usually have 2-3 required activities.
- Standard lessons usually have 3-4 required activities.
- Reference activities are optional by default.
- Each lesson should teach a different decision or concept.
- Required decisions should not repeat the same prompt from the same position
  inside one lesson.
- Dense source material should become useful original explanation, comparison,
  or optional reference cards.
- Do not add generic filler such as `Source note x records...`, raw
  activity-result JSON, or `Course move found`.

## Main Najdorf design

The main learning spine should teach the Najdorf as Black's practical answer to
the Open Sicilian:

```text
1.e4 c5
2.Nf3 d6
3.d4 cxd4
4.Nxd4 Nf6
5.Nc3 a6
```

The spine should explain:

- why Black contests the centre asymmetrically with `...c5`;
- why Black often allows White central space in exchange for counterplay;
- why `...d6` and `...Nf6` fit the Open Sicilian move order;
- why `...a6` controls b5 and prepares queenside expansion;
- when Black plays for `...e5`, `...e6`, `...b5`, piece pressure, or central
  breaks;
- how Black recognizes White's attacking setup before choosing a plan.

The course should not pretend the Najdorf is simple. It should make the
complexity navigable.

## Full Reference family design

Reference depth should cover all major Sicilian families in the selected source
scope, but every family should answer a learner-facing question.

### Dragon and Accelerated Dragon

Teach the fianchetto structure, the race dynamic against kingside attacks, and
the move-order differences that make the Accelerated Dragon distinct. Sharp
tactical lines belong in Reference, not Quick.

### Scheveningen and `...e6` systems

Teach the small-centre structure, flexible development, and transpositional
links with Najdorf-style systems. Taimanov, Paulsen, and Four Knights themes
should be reference branches unless implementation discovers a natural Standard
survival lesson.

### Classical and `...Nc6/...e5` systems

Teach Classical development as a recognizable Sicilian family, then use
Reference depth for Richter-Rauzer, Sveshnikov, Kalashnikov, Lowenthal, and
related sharp systems. These are not the Quick repertoire, but the learner
should understand where they sit in the tree.

### Anti-Sicilians

Anti-Sicilians should be more prominent than pure reference sidelines because
they are common practical obstacles to a Najdorf repertoire. The course should
teach Black how to react when White avoids the Open Sicilian:

- Alapin-style early `c3`;
- Closed Sicilian structures;
- early Bb5 systems;
- f4/Grand Prix-style attacks;
- Smith-Morra Gambit;
- unusual second moves.

Quick should include one anti-Sicilian map. Standard should include several
practical responses. Reference should include the full source-backed branch
coverage.

## Move-order teaching

Move-order decision points should be explicit.

The course should teach questions such as:

- How does Black reach the Najdorf if White cooperates?
- What if White avoids `2.Nf3`?
- What if White plays `2.c3`, `2.Nc3`, `2.f4`, or an early Bb5 system?
- When does Black's `...d6` setup lead to Najdorf, Dragon, Scheveningen, or
  Classical territory?
- When do `...e6`, `...Nc6`, or `...e5` move orders become separate families?
- Which transpositions are safe for a Najdorf player, and which are a deliberate
  repertoire switch?

This is especially important because Black chooses the Sicilian on move one,
but White controls whether the game becomes an Open Sicilian.

## Implementation shape

The implementation should proceed in slices:

1. Build private source inventory.
   - Render the Sicilian source pages privately.
   - Inventory printed pages, table columns, note labels, named variations,
     illustrative references, and candidate teaching nodes.
   - Create `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.authoring.json`.

2. Build Quick spine.
   - Foundations.
   - Open Sicilian entry.
   - Najdorf entry.
   - One practical Black plan.
   - One anti-Sicilian survival map.
   - Black orientation and Black repertoire prompt checks.

3. Build Standard layer.
   - Main Najdorf decision branches.
   - Common anti-Sicilians.
   - Compact recognition lessons for Dragon, Scheveningen, Classical, and
     `...Nc6/...e5` systems.

4. Build Reference layer.
   - Full Najdorf reference.
   - Dragon and Accelerated Dragon.
   - Scheveningen and `...e6` systems.
   - Classical and `...Nc6/...e5` systems.
   - Full anti-Sicilian coverage.
   - Optional reference activities for dense source notes.

5. Generate and validate pack.
   - Produce `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-black.ctcourse`.
   - Validate after each slice with `go run ./cmd/coursepack validate`.
   - Require zero missing and zero unexpected source coverage at final validation.

6. Import and verify.
   - Import into the default local catalogue.
   - Confirm all existing courses remain active.
   - Confirm the new course appears under Black openings.
   - Start Quick, Standard, and Reference Sicilian lessons.
   - Inspect deep Reference branches in the variation explorer.
   - Run repository tests and opening E2E checks.

## App reuse

Do not special-case Sicilian in app code.

The implementation should reuse:

- schema-v2 course packs;
- existing course-pack validator;
- existing importer;
- existing active catalogue replacement behavior;
- existing opening hub grouping by perspective;
- teaching tree UI;
- decision-point lesson UI;
- variation explorer;
- journey persistence;
- opening review scheduling;
- Black board orientation and Black repertoire-role behavior.

If implementation discovers a generic app defect while importing a large
Sicilian pack, fix the generic behavior and cover it with synthetic tests.
Do not add hard-coded Sicilian logic.

## Success criteria

- A private `mco15-sicilian-black` course pack exists and validates.
- The course uses schema version `2`, perspective `black`, and default depth
  `reference`.
- Source coverage for printed pages `244-361` has zero missing and zero
  unexpected records.
- Quick gives a coherent Najdorf-centered Black repertoire path.
- Standard adds main practical branches and common anti-Sicilians.
- Reference exposes the full selected Sicilian source scope as a browseable
  teaching tree.
- Black orientation is correct.
- Black repertoire moves are learner decisions.
- White moves appear as opponent pressure, demonstrations, comparisons, or
  branch choices.
- Existing courses remain active:
  - `mco15-italian-white`
  - `mco15-ruy-lopez-white`
  - `mco15-queens-gambit-white`
  - `mco15-caro-kann-black`
  - `mco15-queens-gambit-black`
- Course warnings remain at zero unless a deliberately user-facing warning is
  truly useful.
- No private source prose, rendered pages, authoring inventory, or course pack
  is committed to Git.

## Out of scope for v1

- Reworking the opening-course UI.
- Rebuilding PDF extraction or OCR.
- Bundling private course packs with the app.
- Creating a public MCO-derived fixture.
- Making the learner complete every Reference branch before progress can move
  forward.
- Adding a White-side anti-Sicilian repertoire course.
- Adding French, Pirc, Alekhine, English, King's Indian, or other new openings
  in this pass.
- Cloud sync of private opening courses.
