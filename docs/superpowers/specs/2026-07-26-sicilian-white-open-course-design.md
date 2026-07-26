# Sicilian White Open Course Design

Status: approved in chat; awaiting written-spec review
Date: 2026-07-26

## Goal

Create a new private opening course that teaches White how to meet
`1...c5` with a principled Open Sicilian repertoire.

The learner should become comfortable entering the Open Sicilian structure:

```text
1.e4 c5 2.Nf3
```

then choosing `d4`, understanding `...cxd4`, recapturing with `Nxd4`, and
recognizing Black's major setup families. The course should feel like a
teaching tree of White decisions, not repeated line drilling.

## Course identity

- Course ID: `mco15-sicilian-white`
- Title: `Sicilian Defence for White`
- Content version: `1.0.0`
- Perspective: `white`
- Default depth: `reference`
- Private authoring file:
  `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.authoring.json`
- Private course pack:
  `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse`

This is a new White-perspective course. Do not add White material to
`mco15-sicilian-black`, because one course should keep one learner perspective,
journey state, and orientation model.

## Source strategy

Use the user's local MCO-15 PDF only for private manual authoring and source
coordinate validation.

The source scope is the Sicilian Defence section already mapped during the
Sicilian Black work:

- Printed pages: `244-361`
- PDF pages: `261-378`
- Scan offset: `+17`

Reuse the existing Sicilian Black page mapping as a coordinate reference only.
Do not reuse its Black-perspective course inventory as-is. The new inventory
must be reorganized around White decisions, White plans, and White repertoire
roles.

Public Git may contain this spec, future implementation plans, generic tooling,
and synthetic tests. Public Git must not contain `.ctcourse` files, private
authoring JSON, rendered PDF pages, extracted source tables, MCO-derived prose,
private validation reports, or private checkpoints.

## Product decisions

The user selected:

- Next opening: Sicilian for White.
- Course type: new course, `mco15-sicilian-white`.
- Repertoire style: Open Sicilian mainline, principled.
- Main teaching spine: structure-first Open Sicilian.
- Initial depth: balanced v1.
- Quick style: decision-point skeleton.
- Source scope: full Sicilian chapter range from the MCO PDF.

The course should teach the real Open Sicilian battlefield. It should not be an
anti-Sicilian shortcut course. Non-Open Sicilian material from the source range
may appear at Reference depth as signposts, exclusions, or future-extension
markers, but the required Quick and Standard journey is Open Sicilian.

## Teaching tree

Quick depth is the short Open Sicilian skeleton:

```text
Sicilian Defence for White
├── Enter the Sicilian
│   └── 1.e4 and recognize ...c5
├── Prepare the centre
│   └── choose 2.Nf3
├── Open the centre
│   └── choose d4 and understand ...cxd4
├── Recapture into the Open Sicilian
│   └── choose Nxd4
└── Recognize Black's setup
    ├── ...d6 family
    ├── ...Nc6 family
    └── ...e6 family
```

Standard depth branches from the setup-recognition node:

- `...d6`: Najdorf, Dragon, Scheveningen, and Classical-style structures.
- `...Nc6`: Classical, Sveshnikov, Kalashnikov, and Accelerated Dragon-style
  recognition.
- `...e6`: Kan, Taimanov, Four Knights, and Scheveningen transpositions.
- White plan selection: development square choices, central control, kingside
  attack signals, and short-castle versus long-castle cues.

Reference depth stores deeper lines and branch notes. It should be a lookup
tree, not a required march through every source table.

## Depth model

### Quick

Quick should have about five required lessons:

1. Enter the Sicilian.
2. Prepare the central break with `Nf3`.
3. Open the centre with `d4`.
4. Recapture with `Nxd4`.
5. Recognize Black's first family setup.

Each Quick lesson should have 2-3 required activities. The learner should make
at most one meaningful White decision in each Quick lesson. Do not ask the
learner to replay the same Open Sicilian skeleton repeatedly.

### Standard

Standard should add compact family-recognition and plan lessons for:

- Najdorf;
- Dragon;
- Classical Sicilian;
- Scheveningen;
- Sveshnikov and Kalashnikov;
- Kan and Taimanov.

Each Standard lesson should have 3-4 required activities, with at most one
required White decision from a single position. Standard should teach how White
chooses a plan after recognizing Black's setup, not force every source
subvariation into required play.

### Reference

Reference should capture the full selected Sicilian source scope in a
source-backed tree. Dense table lines, named variations, transpositions,
illustrative game references, tactical side branches, and non-Open Sicilian
signposts should live in optional `reference` activities unless they teach a
critical White decision.

## Lesson experience

The course should reuse the newer decision-point teaching-tree style.

Preferred rhythm:

```text
Concept → Black pressure or family shown → White decision → plan demonstration → recap → optional Reference
```

Rules:

- White repertoire moves are learner decisions.
- Black moves are opponent pressure, branch choices, or demonstrations.
- Required prompts should not repeat the same position and answer inside one
  lesson.
- Avoid raw line-drill repetition of `e4`, `Nf3`, `d4`, and `Nxd4` once the
  skeleton is learned.
- Dense MCO table material belongs in optional `reference` activities.
- Do not include filler/debug text such as `Source note x records...`,
  `Course move found`, `opening activity result`, or placeholder prose about
  concrete analytical subvariations.
- Course validation must have zero warnings, zero missing coverage, and zero
  unexpected coverage.

## Implementation shape

Create a new private authoring inventory and course pack:

- `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.authoring.json`
- `/Users/admin/Documents/Private Chess Courses/mco15-sicilian-white.ctcourse`

Reuse existing public app infrastructure:

- schema-v2 opening course model;
- coursepack validator and importer;
- teaching-tree lesson UI;
- depth filtering and journey persistence;
- app-service smoke harness patterns used by recent opening-course work.

Reuse existing Sicilian Black work only for process patterns and source-page
coordinate mapping. Do not copy Black-side lesson structure, Black repertoire
roles, or Black-oriented explanations into the White course.

## Verification requirements

Implementation must include:

- private source-scope inventory audit;
- private anti-spam audit;
- validation after Quick, Standard, Reference, and final pack states;
- disposable import smoke;
- default catalogue import smoke proving the course coexists with existing
  opening courses;
- app-service journey smoke proving White orientation and representative
  Quick/Standard/Reference lesson starts;
- full repository verification:
  - `go test ./...`;
  - frontend unit tests;
  - `npm --prefix frontend run check`;
  - Chromium opening E2E;
  - WebKit opening E2E.

## Non-goals

- Do not modify `mco15-sicilian-black`.
- Do not build a primarily anti-Sicilian repertoire in this pass.
- Do not add a generic multi-perspective course model.
- Do not special-case Sicilian White in public app code.
- Do not commit private course artifacts or MCO-derived course prose.
