# Queen's Gambit Defences for Black course design

Status: approved in chat and implemented as private v1.0.0 local course pack
Date: 2026-07-24

## Goal

Create a private, reference-heavy opening course that teaches Black how to meet `1.d4 d5 2.c4` with a serious Queen's Gambit defence repertoire.

The course should balance two full Black families:

- Queen's Gambit Declined, with the Orthodox setup as the main spine.
- Slav/Semi-Slav, with the Slav proper as the main spine and the Semi-Slav as a major branch.

The course should appear as a Black-perspective course in the app, alongside the existing Italian White, Ruy Lopez White, Queen's Gambit White, and Caro-Kann Black courses.

## Course identity

- Course title: `Queen's Gambit Defences for Black`
- Course ID: `mco15-queens-gambit-black`
- Content version: `1.0.0`
- Perspective: `black`
- Authoring file: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.authoring.json`
- Course pack: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-black.ctcourse`

Private source-derived authoring data must remain outside Git.

## Source strategy

Reuse the Queen's Gambit source scope and rendered-page work from the Queen's Gambit White course wherever possible, but create a separate Black authoring inventory.

The reason for a separate inventory is perspective: the White course asks how White builds pressure, while this course asks how Black equalizes, simplifies, counters, or chooses a defensive structure.

The Black inventory should cover:

- Queen's Gambit Declined
- Orthodox QGD
- Exchange QGD
- Lasker
- Tartakower
- Cambridge Springs / Ragozin / Vienna / Semi-Tarrasch recognition where covered by the source scope
- Slav proper
- Semi-Slav
- Meran / Anti-Meran / sharp Semi-Slav reference material
- Tarrasch and Chigorin as side-weapon or recognition systems

Albin and Budapest-style side weapons should only be included if a source-scope check shows they are covered in or cleanly adjacent to the selected Queen's Gambit material. If they require a materially separate source section, leave them for a future side-weapons course.

## Course tree

The course tree should be:

```text
Queen's Gambit Defences for Black
├── Foundations
│   ├── Why Black plays ...d5
│   ├── Accept / decline / support / counter map
│   └── Move-order survival guide
├── QGD Orthodox main spine
│   ├── ...e6 / ...Nf6 / ...Be7 / castle
│   ├── Handling the Bg5 pin
│   ├── White pressure plans
│   ├── Exchange structure
│   ├── Lasker simplification
│   └── Tartakower branch
├── Slav proper main spine
│   ├── ...c6 structure
│   ├── bishop-before-...e6 idea
│   ├── central tension
│   └── queenside pawn structure
├── Semi-Slav branch
│   ├── ...c6 + ...e6 wall
│   ├── Meran
│   ├── Anti-Meran
│   └── sharp tactical branches as Reference
└── Reference side weapons
    ├── Tarrasch / Semi-Tarrasch
    ├── Chigorin
    └── Albin / Budapest only if source scope supports them
```

## Depth model

The user selected a reference-heavy v1.

Use this depth model:

- Quick: core map, basic Black aims, QGD Orthodox entry, Slav proper entry.
- Standard: playable Black repertoire, common White pressure plans, main defensive choices, QGD/Slav/Semi-Slav decision points.
- Reference: dense source table lines, move-order traps, Tartakower details, sharp Semi-Slav branches, Tarrasch/Chigorin side weapons, and any supported Albin/Budapest recognition notes.

Reference depth should be rich, but optional. Dense material should not become a long chain of required learner actions.

## Lesson experience

The course should feel like a guided defence manual, not a puzzle grinder.

Preferred lesson rhythm:

```text
Concept → Black decision → White pressure shown → Black plan recap → optional Reference
```

Black repertoire moves should be the learner's decisions. White moves should be shown as opponent pressure or demonstrations. Avoid requiring the learner to replay the same opening stem in every lesson.

Lesson constraints:

- Board orientation defaults to Black.
- Quick lessons generally have 2-3 required activities.
- Standard lessons generally have 3-4 required activities.
- Reference cards are optional unless they teach a critical defensive decision.
- Required decisions should not repeat the same prompt from the same position inside one lesson.
- Do not add generic spam such as `Source note x records...`, `Course move found`, or raw activity-result text.
- Convert private source notes into useful explanatory prose, or keep them in private reference metadata.

## Move-order teaching

Move-order decision points should be explicit, not hidden in notes.

The course should teach questions such as:

- If White plays `Nf3` before `Nc3`, what changes?
- When does Black stay in QGD territory?
- When does `...c6` become Slav proper versus Semi-Slav?
- When is `...e6` a solid QGD choice versus a Semi-Slav wall?
- Which White move orders make Tartakower, Ragozin, Cambridge Springs, or Semi-Tarrasch ideas relevant?

This is especially important because the course is Black-perspective: the learner must know what White's move order permits before choosing a defensive system.

## Main QGD design

The QGD half should use the Orthodox setup as the main spine:

```text
1.d4 d5 2.c4 e6
3.Nc3 Nf6
4.Bg5 Be7
5.e3 O-O
```

The Orthodox spine should teach:

- why Black supports d5 with `...e6`
- why `...Nf6` and `...Be7` are calm equalizing moves
- when Black castles
- how Black handles the Bg5 pin
- how Black reacts to White's Exchange structure
- when Black simplifies
- when Black solves the light-squared bishop with Tartakower ideas

Tartakower should be included as a branch after the Orthodox skeleton is understood, not as the first Quick path.

## Slav/Semi-Slav design

The Slav/Semi-Slav half should be nearly equal in importance to QGD.

Use Slav proper as the main spine:

```text
1.d4 d5 2.c4 c6
```

Teach:

- why `...c6` supports d5
- why Black often wants the light bishop active before `...e6`
- how White's central tension changes the structure
- when queenside pawn structure matters

Then branch into Semi-Slav:

```text
...c6 + ...e6
```

Teach:

- the wall structure
- Meran recognition
- Anti-Meran plans
- sharp tactical branches as Reference rather than first-pass requirements

## Side weapons and recognition

Reference-only side weapons should be included where source scope supports them.

The intent is recognition and preparedness, not making these the main repertoire. The course identity should remain solid QGD/Slav, not a collection of surprise gambits.

Tarrasch, Semi-Tarrasch, and Chigorin should be treated as side systems from the selected Queen's Gambit scope. Albin/Budapest should be included only after source-scope verification.

## Implementation shape

The implementation should proceed in slices:

1. Build Black authoring inventory.
   - Reuse the Queen's Gambit White source/page work.
   - Create Black-specific lesson-node intent.
   - Map reusable coverage records to Black teaching purposes.

2. Build Quick spine.
   - Foundations.
   - QGD Orthodox map.
   - Slav proper map.
   - Black perspective and orientation checks.

3. Build QGD reference-heavy branch.
   - Orthodox main line.
   - Exchange.
   - Lasker.
   - Tartakower.
   - Cambridge/Ragozin/Vienna/Semi-Tarrasch as reference or side branches.

4. Build Slav/Semi-Slav reference-heavy branch.
   - Slav proper.
   - bishop-before-`...e6`.
   - Semi-Slav wall.
   - Meran / Anti-Meran / sharp tactical branches as Reference.

5. Add side weapons / recognition.
   - Tarrasch and Chigorin from the current Queen's Gambit scope.
   - Albin/Budapest only if source-scope check supports them.

6. Import and verify.
   - Validate all private packs.
   - Import into the default local catalogue.
   - Confirm all existing courses remain active.
   - Run opening E2E tests.
   - Add Black-orientation and Black decision-flow smoke checks if existing tests do not already cover them.

## Success criteria

- A private `mco15-queens-gambit-black` course pack exists and validates.
- The app catalogue shows Queen's Gambit Defences under Black openings.
- Existing courses remain active:
  - `mco15-italian-white`
  - `mco15-ruy-lopez-white`
  - `mco15-queens-gambit-white`
  - `mco15-caro-kann-black`
- Black orientation is correct.
- Black repertoire moves are learner decisions.
- White moves appear as opponent pressure, demonstrations, or comparisons.
- Quick/Standard paths avoid repetitive puzzle-like loops.
- Reference depth is rich but optional.
- Coverage validation has no missing or unexpected records.
- Course warnings remain at zero unless a deliberately user-facing warning is truly useful.
- No private source prose, rendered pages, authoring inventory, or course pack is committed to Git.

## Out of scope for v1

- Reworking the opening-course UI.
- Rebuilding the PDF extraction pipeline.
- Merging White and Black Queen's Gambit courses into one bidirectional course.
- Full standalone Albin/Budapest repertoires if their source sections are outside the selected scope.
- Online/cloud sync of private course packs.
