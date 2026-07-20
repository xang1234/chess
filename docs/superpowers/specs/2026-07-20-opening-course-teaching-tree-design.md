# Opening Course Teaching Tree Redesign

Date: 2026-07-20
Status: Approved on 2026-07-20

## Summary

Redesign Learn Openings around a persistent teaching tree and flexible,
decision-point lessons. The learner should move continuously from one meaningful
opening idea to the next, see how each lesson connects to previously learned
positions, and resume the journey at the exact activity reached.

The first release upgrades and substantially expands the private Italian Game
course for White. It will teach a coherent main repertoire plus one fully taught
alternative at each major branch, while retaining the complete curated MCO-15
reference material as optional depth. After the Italian release is validated,
the same course engine and interface will support deeper White repertoire work
and additional openings.

This design evolves the existing opening-course engine. It keeps the imported
position and move graph, board interaction, prompts, versioned course
generations, source references, variation explorer, local persistence, and
spaced-review system. It replaces the mandatory five-step staircase, flat
lesson list, single-lesson navigation experience, and completion flow that
returns the learner to the hub.

## Relationship to the Previous Design

This specification supersedes the following parts of
`2026-07-18-italian-opening-course-design.md`:

- the mandatory Explain, Watch, Try, Branch, Recall staircase;
- the flat chapter and lesson navigation model;
- the single-lesson completion and Back Home flow;
- progress presented primarily as completed step counts; and
- the five-lesson Italian teaching layer ending in Mixed Recall.

The earlier design remains authoritative for local-first privacy, external
private course packs, source attribution, versioned imports, move-graph
validation, evaluations, transpositions, the variation explorer, and separation
from tactical puzzles unless this specification explicitly changes a behavior.

## Evidence from the Current Product

The installed Italian course currently contains five lessons. Every lesson has
exactly this enforced sequence:

1. Explain.
2. Watch.
3. Try.
4. Branch.
5. Recall.

For the Giuoco, Evans, Two Knights, and foundation lessons, Try, Branch, and
Recall ask for the same course move from the same position. The course compiler
requires all five kinds in this exact order, so the repetition is a content-model
constraint rather than only a frontend presentation problem.

The current opening session stores one lesson ID and one step index. Durable
progress exists per lesson, but there is no course-level journey that preserves
the learner's current path across lessons. Completing a lesson produces a
summary with Back Home as the primary action. The hub then reconstructs a flat
list of chapters and lessons, leaving the learner to remember where the lesson
fit.

The MCO-15 Italian source has a different and more useful teaching structure:

- Giuoco Piano contrasts active `c3` and `d4` play with quieter `d3` systems,
  then develops tactical and positional branches;
- Evans Gambit branches from the same `...Bc5` position into accepted,
  declined, and major defensive setups; and
- Two Knights' Defence branches at `...Nf6` into `Ng5`, the main `...d5`
  defences, tactical alternatives, and quieter White systems.

The redesigned experience should expose those relationships rather than
flattening each chapter into one repeated move exercise.

## Approved Product Decisions

- Use a brief roadmap checkpoint after each completed teaching node.
- Model the course as a teaching tree of meaningful positions, plans, and
  branches, not a dense move-by-move tree.
- Use flexible decision-point lessons.
- Do not ask for the same correct answer more than once inside a lesson.
- Move purposeful repetition into spaced review.
- Use soft guidance: recommend the next node without hard-locking nearby
  branches.
- Teach one primary White repertoire plus one fully taught alternative at each
  major branch.
- Keep Reference as the canonical in-depth course and derive shorter Quick and
  Standard routes from the same content.
- Deliver the reusable journey architecture and a complete Italian White v2
  course in the first release.
- Evolve the existing opening-course engine rather than layering over the
  rigid five-step model or building a separate curriculum engine.

## Goals

- Make the relationship between learned, current, and upcoming opening ideas
  continuously visible.
- Allow a learner to continue through multiple lessons without returning to the
  opening hub.
- Persist course-level position, path, depth, and activity progress.
- Replace mechanical repetition with distinct instructional decisions.
- Keep source analysis available without flooding the teaching surface.
- Preserve compatible progress and review history during the course upgrade.
- Expand Italian White into a useful repertoire course rather than a five-move
  demonstration.
- Provide a reusable authoring and runtime model for later openings.

## Non-Goals

- Do not add another opening beyond Italian White in this release.
- Do not build a general PDF or OCR ingestion pipeline.
- Do not publish or bundle copyrighted MCO-derived course data.
- Do not replace tactical puzzle mechanics or puzzle progress.
- Do not turn every source variation into a required lesson.
- Do not require an engine evaluation service or network connection.
- Do not add a second review scheduler; opening prompts continue to use the
  existing opening-review system.

## Domain Language

### Teaching tree

A rooted instructional tree showing how opening ideas and named branches relate.
It is intentionally coarser than the underlying chess move graph. The move graph
may contain transpositions; the teaching tree gives every lesson one primary
instructional parent so the learner retains a clear path.

### Teaching node

A lesson attached to one meaningful objective, such as preparing `d4` with
`c3`, recognizing the Ulvestad defence, or understanding why the Evans pawn
sacrifice accelerates development. A node is not a single move.

The persisted content unit remains a `Lesson` to retain compatibility with the
existing course engine. The frontend presents each lesson as a teaching node.

### Activity

One instructional element within a lesson. Activities are flexible and do not
follow a universal count or order.

### Journey

Durable course-level state that records where the learner is in the teaching
tree across lesson boundaries. A lesson session is one part of a journey, not
the journey itself.

### Roadmap checkpoint

The brief transition after a node is completed. It shows the path just learned,
reveals connected branches, recommends the next node, and provides a primary
Continue action.

## Learner Journey

### Opening home

The opening home replaces the flat chapter list with the teaching tree. It
shows these states using color plus text or shape so meaning never depends on
color alone:

- completed;
- in progress;
- recommended next;
- review due;
- available; and
- hidden by the selected depth.

The tree starts at shared Italian foundations and visibly branches at Black's
major defensive choices. Learned nodes and their connecting path remain visible
while the learner explores a nearby alternative.

The primary home action is Continue Learning. It resumes the current lesson and
activity when one is in progress; otherwise it opens the currently recommended
node. Review Due remains a separate action because review and new instruction
serve different purposes.

Soft guidance means nearby visible nodes are selectable even if they are not
recommended. The UI may explain a useful predecessor, but it does not hard-lock
the node.

### Depth

Reference is the canonical authored course. Quick and Standard are filtered
routes through that same tree:

- Quick contains the essential coherent White repertoire;
- Standard adds the fully taught alternatives and major Black defences; and
- Reference exposes all teaching nodes and optional source analysis in scope.

Changing depth never deletes or forks progress. A node completed at any depth
remains completed. The compiler must ensure that every depth produces a
connected instructional route and that optional deeper nodes are not required
intermediates for shallower nodes.

### Lesson context

The lesson screen keeps a compact path visible, for example:

`Italian > ...Bc5 > Giuoco Piano > c3 and d4`

Progress uses instructional language such as `2 of 3 ideas learned`, not a
fixed `Step 4 of 5`. The learner can open the relevant local branch of the tree
without abandoning the lesson.

### Completion and continuation

Completing a node opens the roadmap checkpoint instead of a terminal summary.
The checkpoint:

1. marks the node complete;
2. keeps its parent path visible;
3. reveals or emphasizes connected next branches;
4. previews the recommended next lesson; and
5. offers Continue as the primary action.

View Full Tree and Stop for Now are secondary actions. Continue starts the next
node directly within the same course journey. Returning to the application home
is never required between lessons.

## Lesson Experience

### Flexible activity types

Schema v2 supports these instructional activity kinds:

- `concept`: concise explanation with a position and optional annotations;
- `demonstration`: connected move sequence with annotation, pause, and replay;
- `decision`: one meaningful move choice backed by an existing prompt;
- `comparison`: contrast two plans, positions, or move sequences;
- `recap`: concise lesson summary without a required board action; and
- `reference`: optional source analysis, notes, evaluations, and citations.

A lesson may use only the kinds it needs. A conceptual lesson may contain no
move decision. A tactical lesson may contain several decisions, provided they
occur at distinct positions or test materially different ideas.

Activities have stable IDs and may be required or optional. A node becomes
complete when all required activities are completed. Optional Reference
activities do not block completion.

### Example: prepare `d4` in the Giuoco Piano

1. Explain why `c3` supports central expansion.
2. Demonstrate natural development to the `...Bc5` structure.
3. Ask the learner to find `c3` from the critical position.
4. Continue from the resulting line and ask when to play `d4`.
5. Compare the active `c3` and `d4` plan with a quieter `d3` setup.
6. Recap the plan and advance to the roadmap checkpoint.

The learner never has to replay `c3` three times under different labels.

### Board behavior

- A lesson opens at its relevant position.
- Moves to Here is an optional replay of the path into that position.
- Demonstrations reuse the existing legal move, animation, notation, sound, and
  board-orientation behavior.
- Decision activities reuse prompts, accepted alternatives, progressive hints,
  reveal behavior, and move feedback.
- A wrong move receives position-specific explanation and a guided retry.
- A sound alternative may be accepted while explaining why the course
  recommends a different repertoire move.
- Once a required decision is resolved, it is not asked again in that lesson.
- A hinted or revealed decision may still allow the lesson to complete, but it
  should schedule earlier review through the existing outcome model.

### Teaching text and source text

The primary lesson surface contains concise, original teaching text. Dense
analytical subvariations, evaluation symbols, source notes, and illustrative
game citations are collapsed under Deeper Analysis. Source-note placeholders
and repeated private-reference warnings must never appear as a long inline
block.

Most Quick and Standard nodes should take about two to four minutes. Reference
activities may be longer because they are explicitly optional deep dives.

### Repetition

Lessons introduce and connect ideas. Repetition belongs to the opening-review
queue after spacing. The current Mixed Recall lesson is removed; its useful
positions become review prompts with retained semantic fingerprints.

## Course Schema v2

### Reused content

The following existing content remains authoritative:

- course, source, and coverage metadata;
- positions and their evaluations;
- moves, roles, variation names, source references, and transpositions;
- notes;
- prompts and accepted alternatives;
- chapters; and
- lesson IDs, titles, objectives, minimum depth, and start positions.

### New teaching-tree data

The course adds explicit lesson edges. Each edge contains stable source and
target lesson IDs, an ordinal, a label when needed, a minimum depth, and one of
these relationships:

- `continuation`;
- `alternative`; or
- `reference`.

Every non-root lesson has exactly one primary instructional parent. The course
move graph remains responsible for chess transpositions; the teaching tree
does not duplicate or distort positions to represent them.

### Activities

Schema v2 replaces lesson `steps` with `activities`. Every activity contains a
stable ID, kind, title, instruction, required flag, position ID when relevant,
and the activity-specific move, prompt, note, comparison, or annotation data.

Schema v1 remains importable during the transition. The importer normalizes a
v1 lesson into the legacy activity representation so existing fixtures and old
external packs do not fail immediately. Newly authored Italian v2 content uses
schema v2 only.

### Compiler validation

The compiler must validate:

- stable unique lesson, edge, and activity IDs;
- one root and full teaching-tree reachability;
- no teaching-tree cycles;
- one primary incoming edge per non-root lesson;
- deterministic edge ordering;
- valid Quick, Standard, and Reference routes;
- lesson and activity depth consistency;
- referenced positions, moves, prompts, and notes;
- connected demonstration paths;
- prompt and step position consistency for decisions;
- legal automatic opponent continuations;
- activity-specific required and forbidden fields;
- coherent continuation, alternative, and reference labeling; and
- no duplicate required decision that asks for the same answer from the same
  position inside one lesson.

The compiler no longer requires all five legacy phases, a phase order, an
Explain first activity, or a Recall last activity.

## Persistence and Migration

### Course journey state

Add durable journey state keyed by course ID. It stores:

- selected depth;
- current lesson ID;
- current activity ID;
- last visited lesson path;
- last recommendation snapshot;
- active opening-session ID when a board interaction is in progress;
- creation and update timestamps; and
- the course-level state needed to reconnect that session after restart.

The service recomputes the recommended next node from the current tree and
progress whenever the journey is loaded. A stored recommendation is only a
resume hint and cannot strand the learner when course content changes.

### Lesson progress

Continue storing progress by course ID and lesson ID, but treat the stored
stable IDs as activity IDs rather than ordinal step counts. Store node
completion independently from the currently visible number of activities.

Persist progress after every completed activity. An interrupted learner resumes
the exact activity and board state rather than restarting the node.

### Sessions

The existing opening-session machinery remains responsible for an active
lesson or review attempt. It is generalized from step index to activity
position and continues to own the exact board state. The course journey owns
continuity across completed lesson sessions and points to the active session
when one exists. Only one board interaction session needs to be active at a
time, while every course may retain its own journey and progress for future
multi-course support.

### Existing Italian progress

The Italian v2 course retains the existing course ID and stable IDs for concepts
that continue to exist. Migration follows these rules:

- a completed legacy lesson marks its corresponding teaching node complete;
- a completed node remains complete when optional activities are added;
- compatible partial activity credit maps through retained activity IDs or
  explicit aliases;
- existing prompt outcomes and review schedules rebase through semantic
  fingerprints;
- an incompatible active session restarts at the nearest preserved node only
  after showing the learner what was preserved; and
- newly added material is never silently marked complete.

Migration tests must cover the currently possible state: several completed
legacy lessons plus one active lesson session.

## Service and Contract Changes

### Opening home

The opening home response supplies:

- course summary and selected depth;
- teaching-tree nodes and edges;
- each node's progress state;
- the current lesson and activity;
- the current path;
- recommended next lesson;
- due-review count; and
- whether an exact resume target exists.

### Lesson session

The lesson session exposes the activity kind and activity-specific data rather
than assuming every screen is a prompt or legacy passive step. Shared board,
feedback, hint, animation, and source-note fields remain reusable.

### Activity completion

Advancing or resolving an activity persists its stable ID. Completing the last
required activity returns a roadmap-checkpoint payload containing:

- completed lesson ID;
- current path lesson IDs;
- newly emphasized or available child lesson IDs;
- recommended next lesson ID and title;
- updated course progress; and
- due-review change when relevant.

The frontend can therefore render the checkpoint and continue without
round-tripping through the opening home.

## Frontend Design

### Components to reuse

- the existing opening feature entry point and hub shell;
- chessboard rendering and orientation;
- legal-move interaction and animation;
- sound and notation;
- hint and reveal controls;
- feedback presentation;
- source-note and variation-explorer access; and
- existing API and Wails binding patterns.

### Components to add or generalize

- `OpeningTeachingTree` for the full roadmap;
- `OpeningPathContext` for the compact in-lesson path;
- activity renderers for concept, demonstration, decision, comparison, recap,
  and reference;
- `OpeningRoadmapCheckpoint` for seamless lesson transitions; and
- generalized opening controller/state transitions based on activities rather
  than the legacy five kinds.

The current lesson-completion Back Home screen is removed from the primary
journey. The opening hub's ordered lesson rows become the accessible fallback
representation of the same tree rather than the sole navigation model.

### Accessibility and responsive behavior

- Render a semantic nested-list equivalent of the visual tree.
- Keep every node keyboard reachable with a visible focus indicator.
- Pair state colors with text, icons, or shapes.
- Preserve a sensible reading and tab order independently of drawn edges.
- At narrow widths, show the current branch and immediate neighboring branches
  rather than shrinking the entire tree into unreadable nodes.
- Announce activity completion, checkpoint state, and the recommended next node
  through the existing live-region pattern.

## Italian White v2 Scope

The source scope remains MCO-15 printed pages 18-41:

- Giuoco Piano, printed pages 18-25;
- Evans Gambit, printed pages 26-29; and
- Two Knights' Defence, printed pages 30-41.

The target is approximately 22 meaningful teaching nodes. The exact count may
change during manual authoring when adjacent source branches have one shared
instructional purpose or one branch needs more than one distinct decision.

### Shared foundations

- Reach and recognize the Italian position.
- Understand development, castling, pressure on `f7`, central occupation, and
  the importance of `d4`.
- Recognize the major `...Bc5` and `...Nf6` split.

### Giuoco Piano

- Recognize the `...Bc5` structure.
- Understand why `c3` prepares `d4`.
- Choose the correct moment for central occupation.
- Handle resulting central exchanges and development choices.
- Understand the Moller Attack's initiative and sacrifices.
- Learn the quieter `d3` plans and the timing of `...a6` and `...d5`.
- Retain uncommon `c3` and `Nc3` systems as Reference branches.

### Evans Gambit

Evans is the fully taught alternative to the main Giuoco repertoire.

- Understand the purpose of `b4` and the compensation for the pawn.
- Meet the accepted gambit and build the centre with `c3`.
- Coordinate castling, `d4`, and development.
- Recognize major defensive setups, including sharper compromised structures.
- Meet the declined gambit.

### Two Knights' Defence

- Recognize `...Nf6` and understand the purpose of `Ng5`.
- Respond to `...d5` with the critical central decision.
- Learn the principal `...Na5` continuation.
- Recognize the Fritz `...Nd4` defence.
- Recognize the Ulvestad `...b5` defence.
- Understand the Wilkes-Barre `...Bc5` tactical branch.
- Teach quiet `d3` as the full alternative White system.
- Retain the Max Lange, `d5`, and `Nc3` sidelines at Reference depth.

### Depth targets

- Quick: about eight essential nodes forming one coherent White repertoire.
- Standard: about fifteen to seventeen nodes, including the fully taught
  alternatives and major Black defences.
- Reference: the complete teaching tree plus optional source analysis.

Reference depth retains complete curated source coverage, but only material
with a distinct instructional purpose becomes a teaching node. Remaining table
columns, subvariations, evaluations, transpositions, and game citations remain
available through Deeper Analysis and the variation explorer.

## Privacy and Content Separation

The application code, schema migrations, generic fixtures, and tests belong in
the repository. The MCO-derived `.ctcourse` pack remains in the established
external private course directory and must not appear in Git, release assets,
or public test fixtures.

After a successful import, runtime content and learner progress remain in the
existing local SQLite stores. No network service is introduced.

## Error Handling

- Invalid schema, graph, move, prompt, depth, or source coverage data abandons
  the staged generation and leaves the active course unchanged.
- A failed activity-progress save keeps the learner on the current activity and
  presents Retry; the UI must not advance optimistically and lose the answer.
- A missing or stale recommendation is recomputed from the current tree.
- A removed optional node does not damage parent-path progress.
- An incompatible active session enters the explicit preserved-progress restart
  flow rather than silently restarting or completing content.
- If a checkpoint cannot load its next lesson, the completed progress remains
  committed and the full tree provides a safe recovery route.

## Verification Strategy

### Compiler and catalogue

- Accept valid flexible activities and schema-v1 compatibility input.
- Reject malformed activity-specific fields.
- Reject teaching-tree cycles, disconnected nodes, duplicate parents, and
  invalid depth routes.
- Reject duplicate required answers from the same position within one lesson.
- Preserve move legality, source coverage, transposition, evaluation, and
  fingerprint validation.

### Persistence and migration

- Save and reload course journey state.
- Resume the exact lesson, activity, and board position after process restart.
- Migrate completed legacy lessons without revoking completion.
- Map compatible partial progress and safely restart incompatible progress.
- Preserve opening prompt outcomes and due-review scheduling.
- Switch among Quick, Standard, and Reference without deleting progress.

### Service

- Recommend an in-progress node before a new node.
- Derive the next main-path node while leaving alternatives selectable.
- Complete an activity and persist its stable ID exactly once.
- Complete a node, return a valid roadmap checkpoint, and start the recommended
  next node without visiting Home.
- Recover from stale content generations and stale recommendation snapshots.

### Frontend

- Render visual and semantic teaching-tree representations from the same data.
- Show completed, current, recommended, due-review, and available states.
- Render every activity kind without assuming a prompt exists.
- Never request the same solved move repeatedly inside one lesson.
- Keep the compact path visible through the lesson.
- Move from node completion to checkpoint to next lesson seamlessly.
- Support keyboard navigation, live announcements, and narrow layouts.
- Keep dense source analysis collapsed by default.

### Italian course acceptance

- Validate complete curated coverage of printed pages 18-41.
- Confirm the Quick route is coherent and connected.
- Confirm the Standard route includes major defences and taught alternatives.
- Confirm Reference exposes the full teaching and analysis scope.
- Confirm retained legacy lesson and prompt IDs migrate as specified.
- Manually inspect every teaching node for a distinct instructional purpose.
- Complete an end-to-end journey across foundations, a Giuoco branch, a
  roadmap checkpoint, and a Two Knights or Evans branch.
- Confirm no private `.ctcourse` file is tracked by Git.

## Delivery Sequence

1. Add schema-v2 activities and teaching-tree edges with schema-v1 import
   compatibility.
2. Add course-level journey persistence and migrate existing progress.
3. Extend service and Wails contracts for trees, activities, and checkpoints.
4. Replace the flat opening hub with the teaching-tree experience.
5. Generalize the lesson screen and controller for decision-point activities.
6. Author and validate the complete private Italian White v2 course.
7. Import the course, verify the real local migration, run the full test suite,
   and rebuild the application.

Implementation must begin from the repository's then-current state and preserve
the existing uncommitted Italian note-display work or any newer overlapping
changes.

## Future Expansion

After Italian White v2 is validated, further White openings should reuse the
same tree, activity, journey, depth, and review model. Adding a new opening then
becomes primarily course authoring plus source validation rather than another UI
or persistence redesign. Black repertoire tracks remain a later product phase.
