# Queen's Gambit White v2 Carlsbad Deepening Design

Status: implemented; automated verification passed; live manual Learn Openings acceptance outstanding
Date: 2026-07-26

## Goal

Deepen the private `mco15-queens-gambit-white` course into version `2.0.0`.
The redesign should make the course feel like a coherent White repertoire built
around the Exchange Queen's Gambit Declined / Carlsbad structure, while keeping
the broader Queen's Gambit family available as a source-backed reference tree.

The learner should come away with a practical answer to:

- how to reach and recognize the Carlsbad structure;
- when White should choose the queenside minority attack;
- when White should prepare the central `e4` break;
- how to recognize the major Queen's Gambit branches where Carlsbad no longer
  directly applies.

## Course identity

- Existing course ID: `mco15-queens-gambit-white`
- New content version: `2.0.0`
- Title: `Queen's Gambit for White`
- Perspective: `white`
- Default depth: `reference`
- Private authoring file: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.authoring.json`
- Private course pack: `/Users/admin/Documents/Private Chess Courses/mco15-queens-gambit-white.ctcourse`

The course keeps the same ID so app users receive it as an update to the
existing Queen's Gambit White course. Because this is a real redesign, the
version should move to `2.0.0`; existing journey state may be rebased when stable
IDs still match, or marked restart-required by the existing opening-course
journey logic when the lesson structure changed too much.

## Source and privacy constraints

Use the existing Queen's Gambit White source scope and private MCO-15 PDF
workflow already established for v1. Do not commit source-derived course data.

Public Git may contain:

- this design spec;
- a future implementation plan;
- generic validator, importer, or UI fixes if the private course exposes a
  reusable app defect.

Public Git must not contain:

- `.ctcourse` files;
- `.authoring.json` inventories;
- rendered PDF pages;
- source table text or note prose;
- private validation, import, or checkpoint outputs.

## Teaching direction

The main spine is the Exchange QGD / Carlsbad structure. Quick and Standard
should teach the structure and decisions, not force brittle move-order memory.

The course should reduce repetitive move drills. A learner should normally make
one meaningful decision per lesson section, then continue through concept,
comparison, demonstration, or recap activities that explain why the decision
matters.

## Depth model

### Quick

Quick is a flexible structure lesson. It should teach that multiple move orders
can reach the same Carlsbad plan.

Required Quick lessons:

1. Recognize the Carlsbad structure.
   - White has exchanged on d5.
   - Black has recaptured with the e-pawn.
   - The c-file and queenside pawn majority define White's plan.
2. Choose the plan, not just the move.
   - Compare queenside minority attack with central `e4`.
3. Develop for the plan.
   - Typical White coordination: `Bd3`, `Nf3`, `Qc2`, `Nge2` or `Re1`, depending
     on Black's setup.
4. Meet Black's counterplay.
   - Introduce `...Ne4`, `...Bf5`, `...c5`, and kingside activity.
5. Convert opening knowledge into a middlegame map.
   - Show where memorization ends and strategic play begins.

Quick may show both immediate and delayed Exchange move orders, but it should
not require the learner to replay both if the decision is semantically the same.

### Standard

Standard balances strategic comparisons with branch recognition.

Primary teaching mode:

- minority attack vs central `e4` break vs Black counterplay.

Secondary teaching mode:

- identify when Black's setup is QGD, QGA, Slav, or Semi-Slav, and choose the
  right White plan.

Required Standard coverage:

1. QGD / Carlsbad plan selector.
2. Minority attack mechanics.
3. Central `e4` break mechanics.
4. Black's active counterplay in Carlsbad structures.
5. QGD branch recognition, including Orthodox/Tartakower/Lasker where source
   scope supports clean coverage.
6. QGA recognition and White's practical setup.
7. Slav and Semi-Slav recognition and White's practical setup.

Standard should not require Tarrasch or Chigorin as active lessons unless the
implementation finds that v1 already has strong, clean, non-repetitive lesson
material for them. Otherwise they belong in Reference.

### Reference

Reference should expose the full Queen's Gambit map from the selected source
scope as optional browseable material.

Reference branches:

- Exchange QGD / Carlsbad move-order nuances;
- minority attack model structures;
- central `e4` model structures;
- Black counterplay with `...c5`, `...Ne4`, `...Bf5`, and kingside plans;
- Orthodox QGD;
- Tartakower;
- Lasker;
- Cambridge Springs;
- Ragozin if cleanly inside scope;
- Queen's Gambit Accepted;
- Slav;
- Semi-Slav;
- Tarrasch;
- Chigorin.

The Reference layer should be rich but optional. It should not flood the
student's required path with source-note placeholders or encyclopedic note
cards.

## Lesson tree shape

The tree should make the learner's progress visible from one lesson to the
next:

```text
Queen's Gambit White v2
├─ Carlsbad foundations
│  ├─ Recognize the structure
│  ├─ Choose minority attack or e4
│  └─ Develop for the plan
├─ Carlsbad strategy
│  ├─ Minority attack mechanics
│  ├─ Central e4 break
│  └─ Black counterplay
├─ Branch recognition
│  ├─ QGD family
│  ├─ QGA
│  └─ Slav / Semi-Slav
└─ Reference map
   ├─ QGD branches
   ├─ QGA branches
   ├─ Slav / Semi-Slav branches
   └─ Tarrasch / Chigorin optional branches
```

Edges should connect lessons by decision logic, not by source-page order. The
source order remains available through source references and the variation
explorer.

## Activity design rules

- Required activities should be mostly `concept`, `comparison`, `decision`,
  `demonstration`, and `recap`.
- Avoid repeated required prompts from the same position inside one lesson.
- Prefer one meaningful learner move, followed by explanation and comparison.
- Reference-only activities should be optional unless they teach a practical
  Standard decision.
- White repertoire moves are learner decisions.
- Black moves should be presented as opponent pressure, branch choices, or
  demonstrations.
- Do not add generic filler such as `Source note x records...`, raw
  activity-result text, or `Course move found`.

## Import and compatibility

The implementation should replace only `mco15-queens-gambit-white`.

After import:

- Italian White remains active.
- Ruy Lopez White remains active.
- Caro-Kann Black remains active.
- Queen's Gambit Defences for Black remains active.
- Sicilian Black remains active.
- French Black remains active if imported locally.

The app should not special-case Queen's Gambit by course ID or title. Any public
code changes must be generic.

## Validation and acceptance

Private validation must prove:

- course ID is `mco15-queens-gambit-white`;
- content version is `2.0.0`;
- perspective is `white`;
- default depth is `reference`;
- warnings are zero;
- source coverage has no missing or unexpected records;
- required Standard lessons cover QGD, QGA, Slav, and Semi-Slav;
- Tarrasch and Chigorin are present in Reference unless source-scope audit
  shows they are outside the selected Queen's Gambit scope;
- no private source artifacts are tracked by Git.

Automated app verification should include:

- coursepack validation;
- private inventory/final audits;
- disposable import smoke;
- default catalogue import smoke;
- orientation smoke confirming White perspective;
- opening E2E tests for teaching-tree journey and deep variation scrolling.

Live manual app acceptance, when a controllable app session is available:

- Queen's Gambit for White appears under White openings.
- Quick starts the v2 course and teaches the Carlsbad structure.
- Standard includes QGD, QGA, Slav, and Semi-Slav recognition.
- Reference exposes Tarrasch and Chigorin as optional branches.
- Continue Learning resumes the Queen's Gambit journey.
- Variation explorer opens from the course root and remains usable on deep
  branches.
- No generic note spam or raw debug text appears.

## Out of scope

- Creating a new course ID.
- Building a full `1.d4` universe course.
- Adding King's Indian, Nimzo-Indian, Queen's Indian, Grunfeld, Dutch, English,
  Reti, or Catalan.
- Deepening Queen's Gambit Defences for Black in the same pass.
- Public app behavior that only applies to Queen's Gambit by name.
