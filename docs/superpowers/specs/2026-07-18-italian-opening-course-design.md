# Italian Opening Course Design

Date: 2026-07-18
Status: Approved on 2026-07-18

## Summary

Add a private, local-first opening-learning system to Chess Trainer. The pilot
course teaches the Italian Game from White's perspective using manually curated
material from *Modern Chess Openings, 15th Edition* (MCO-15), printed pages
18-41. It covers the Giuoco Piano, Evans Gambit, and Two Knights' Defence.

The learner follows a guided staircase:

1. Explain the idea and variation label.
2. Watch the moves on the existing board.
3. Play the White repertoire moves with guidance.
4. Recognize Black's named branches.
5. Recall the line without guidance.

The course retains reference-level variations, labels, evaluations, notes,
transpositions, and source references. A depth selector derives Quick and
Standard views from the same full graph instead of maintaining abridged copies.

Opening courses have their own domain model, importer, catalogue, service, and
lesson sessions. They reuse the existing chessboard, legal-move rules,
animation, sound, hint concepts, import-job patterns, persistence conventions,
and spaced-review intervals. Tactical puzzle records and semantics remain
unchanged.

## Context and Source Structure

MCO-15 is a reference encyclopedia rather than a lesson sequence. Its reusable
teaching structure is:

- group openings into families and named chapters;
- introduce each chapter with its starting move sequence and a diagram;
- explain the opening's purpose, history, and strategic character;
- divide major responses into parallel labeled table columns;
- attach deeper sidelines, evaluations, transposition references, and
  illustrative-game citations as notes; and
- advise novice learners to choose a small repertoire, learn one response at a
  time, and add deeper variations later.

The application adds the sequencing, prompts, depth tiers, and spaced recall
that the book does not provide.

The PDF is an image-based scan. Generic text extraction returns no useful
content, and its dense parallel tables, chess symbols, footnotes, and
transpositions make automated OCR unsafe for the pilot. The Italian material
will therefore be curated manually and checked visually against the source.

## Approved Product Decisions

- The content is for private use.
- The first course is the Italian Game.
- The first learning track is from White's perspective.
- Store the full reference depth, then filter it into shorter views.
- Curate the pilot manually; PDF and OCR import are outside this feature.
- Keep the copyrighted course data in an external private course pack rather
  than in the application repository or release bundle.
- Use a dedicated opening-course model rather than overloading tactical puzzles
  or ordinary PGN games.
- Use the guided staircase as the primary experience. A variation map and
  book-style notes are secondary reference views.

## Goals

- Import an external Italian course pack through the existing parent import
  area.
- Preserve the complete curated reference graph while offering Quick,
  Standard, and Reference depth.
- Teach one marked White repertoire response at each learner decision point.
- Treat other sound moves as playable alternatives, not chess mistakes.
- Preserve named variations, strategic explanations, structured evaluations,
  transpositions, and source-page references.
- Resume a lesson at the exact step and board position after closing the app.
- Schedule missed or hinted opening positions for later recall.
- Preserve compatible progress when a corrected course pack is reimported.
- Keep the app fully offline and local-first.
- Keep all current puzzle behavior and data compatible.

## Non-Goals

- Importing the full MCO-15 book in this milestone.
- Automatic PDF parsing, OCR, or table reconstruction.
- Bundling or publishing MCO-15 content with the application.
- Teaching the Italian Game from Black's perspective.
- Building an in-app course editor.
- Generating engine evaluations or explanations.
- Classifying every opening by ECO code.
- Replacing the puzzle solution-tree model.
- Treating opening alternatives as tactical failures.
- Cloud sync, accounts, or online course fetching.

## Private Content Boundary

The generic course schema, importer, validator, services, and UI may live in the
application repository. The MCO-derived `.ctcourse` file must remain outside
the repository, source archives, release bundles, and test fixtures. Repository
tests use small synthetic opening courses with original text.

The private pack may retain transcribed source prose and tables for personal
use. Each source-derived item records a printed page and, where applicable, a
table column or note label. If the application is later distributed with an
opening course, that course must use original or appropriately licensed
content.

## Architecture

### Components

- `CoursePackImporter` decodes, validates, stages, and atomically activates an
  external course pack.
- `CourseCatalog` stores and retrieves active course content from
  `courses.sqlite`.
- `OpeningService` selects lesson steps, filters graph content by depth,
  validates learner moves, produces authoritative board states, saves lesson
  progress, and schedules reviews.
- `CoursePackValidator` is shared by the application importer and a
  developer-facing validation command used during manual curation.
- `OpeningHub` lists imported courses, due reviews, selected depth, chapters,
  and resume state.
- `OpeningLessonScreen` provides the guided staircase around the existing
  chessboard.
- `VariationExplorer` displays the labeled position graph, book-style notes,
  and alternative branches without changing learner progress.

### Reuse Boundaries

Reuse or extract narrow shared primitives from:

- `frontend/src/components/chess/ChessBoard.svelte` for orientation, legal
  destinations, markers, input, and board updates;
- `frontend/src/components/puzzle/puzzle-effects.ts` for animation, sound, and
  reduced-motion behavior;
- `internal/chessrules` for FEN validation and legal UCI move application;
- `internal/importjob` for asynchronous progress, cancellation, and final
  reports;
- the generation-and-head pattern in the puzzle catalogue for atomic content
  replacement;
- `internal/training/scheduler.go` for the 1, 3, 7, 21, and 60-day interval
  calculation; and
- `NormalShell`, `HomeHub`, and `ImportPanel` for navigation and entry points.

Opening code may depend on shared board, rules, effects, and scheduling
primitives. It must not depend on puzzle records or pretend that an opening move
is a tactical solution. If existing puzzle helpers contain puzzle-specific
language or assumptions, extract only the generic primitive and keep both
domain controllers separate.

### Storage

Use a new replaceable `courses.sqlite` for imported opening content. It follows
the existing content-generation approach:

- a staged generation is tied to a stable course ID;
- validation and writes complete before sealing;
- a course head identifies the active generation; and
- activation replaces only that course's previous generation.

An older sealed generation remains available while a resumable lesson still
references it. After activation, `OpeningService` rebases that session onto the
new generation when its stable lesson, prompt, position, and move references
still exist with compatible fingerprints. If they do not, it pauses the lesson
and offers restart from the nearest retained checkpoint. Generation cleanup
runs only after no resumable session references the old generation.

Opening learner state remains in `user.sqlite`, alongside other personal
progress. It is never deleted when a course generation is replaced.

Conceptual content tables are:

- course generations and course heads;
- courses and chapters;
- positions and move edges;
- notes, evaluations, and source references;
- lessons, steps, prompts, and their graph references; and
- indexes for chapter order, depth, position lookup, and source coverage.

Conceptual user tables are:

- course preferences and selected depth;
- lesson progress and resumable opening sessions;
- prompt outcomes and mastery fingerprints; and
- opening review state.

Database migrations and recovery checks follow the existing storage service
patterns. Course-database corruption is isolated: the application keeps
puzzles and learner history available and asks the parent to reimport the
replaceable course.

## External Course-Pack Format

### File Form

Version one uses a single UTF-8 JSON document with the `.ctcourse` extension.
It does not require a ZIP container or external assets. The root document
contains the manifest and all course content so it can be edited, validated,
versioned privately, and imported as one atomic unit.

Required manifest fields include:

- `schemaVersion`;
- stable `courseId`;
- `contentVersion`;
- title and short description;
- learner perspective (`white` for the pilot);
- source title, edition, and private-use notice;
- default depth;
- start position ID; and
- declared printed-page coverage.

### Position Graph

The course is a directed acyclic graph of positions and legal moves, not a list
of duplicated complete lines.

A position contains:

- a stable `positionId`;
- optional variation and position labels;
- strategic plans, warnings, and evaluation references; and
- optional prompt IDs that can begin at this position.

Only the course root requires a source FEN. Every other position is derived by
applying incoming UCI edges. The importer canonicalizes position identity using
the first four FEN fields: piece placement, side to move, castling rights, and
en-passant target. Halfmove and fullmove counters do not prevent legitimate
opening transpositions from converging.

When multiple move orders target one position ID, all derived canonical
positions must agree. The importer rejects inconsistent transpositions,
unreachable nodes, and cycles.

A move edge contains:

- stable `moveId`, source position ID, destination position ID, and UCI move;
- a minimum depth of `quick`, `standard`, or `reference`;
- a training role of `repertoire`, `opponent`, or `alternative`;
- optional variation name, note references, and structured evaluation;
- source printed page plus optional table-column or note label; and
- optional flags for a branch-recognition or recall checkpoint.

SAN is derived from the legal position and UCI move during import. It is not
trusted as manually entered authority.

Minimum depth is cumulative:

- Quick includes edges marked Quick.
- Standard includes Quick and Standard edges.
- Reference includes every edge.

The validator rejects a shallower edge that depends on a deeper-only parent.
Every learner decision used in recall has exactly one primary repertoire move
for its track. Other legal course edges can be marked as alternatives.

### Notes and Evaluations

Notes are plain text, not executable HTML. Note types include overview,
history, plan, warning, explanation, evaluation, transposition, and
illustrative-game citation.

MCO evaluation symbols are mapped into structured values such as equal,
unclear, White slightly better, Black slightly better, White clearly better,
Black clearly better, White winning, and Black winning. The pack may retain the
source symbol for faithful private display, but application logic uses the
structured value.

### Chapters, Lessons, and Prompts

Chapters provide teaching order over the position graph. A lesson contains:

- stable lesson ID, chapter ID, order, title, and objectives;
- minimum visible depth;
- starting position and referenced path or branch;
- ordered explain, watch, try, branch, and recall steps; and
- prompt IDs with expected repertoire moves and optional accepted alternatives.

Lessons reference shared positions and moves. They do not duplicate chess
lines. A prompt has a semantic fingerprint derived from its stable ID, start
position, expected move set, and relevant continuation. The fingerprint detects
meaningful course updates while the stable ID preserves identity.

## Import and Curation Workflow

### Manual Curation

The private Italian pack is produced outside the repository:

1. Inventory the printed pages, table columns, and labeled notes for MCO-15
   pages 18-41.
2. Enter the chapter overviews, position nodes, legal UCI edges, variation
   labels, notes, evaluations, transpositions, and source references.
3. Mark one default White repertoire move at each recall decision.
4. Assign minimum depth and teaching roles.
5. Define the guided lesson and mixed-recall order.
6. Run the validator and inspect its source-coverage report.
7. Compare the rendered course visually with the PDF before importing it into
   the learner application.

The coverage inventory declares expected source references. The validator
reports missing, duplicate, and unrecognized references and groups captured
items by page and table column. It cannot judge whether manually transcribed
prose is faithful, so that remains a visual review step.

### Application Import

The parent selects a `.ctcourse` file through the existing import area. The
importer:

1. validates file size, UTF-8, JSON shape, and schema version;
2. validates stable IDs, enums, text limits, and source references;
3. walks the graph from the root while applying every move through
   `internal/chessrules`;
4. derives SAN and positions, and validates transpositions and depth
   reachability;
5. validates chapters, lessons, steps, prompts, and repertoire uniqueness;
6. writes a new staged course generation;
7. seals and activates it only after every check succeeds; and
8. reports counts for chapters, positions, moves, variations, notes, lessons,
   and warnings.

Cancellation, parse failure, validation failure, or storage failure abandons
the staged generation and leaves the active course unchanged. Diagnostics name
the course, chapter or variation when known, stable record ID, and exact JSON
path.

Activation also attempts to rebase resumable sessions by stable IDs and
fingerprints. Rebase failure does not roll back valid course content; it pauses
only the incompatible lesson and preserves its history for the learner-facing
restart flow.

After a successful import, the source file is no longer needed at runtime.

## Pilot Course Scope

The White-first Italian course covers the three consecutive MCO-15 chapters
before the Ruy Lopez:

1. Giuoco Piano, printed pages 18-25 (PDF pages 35-42): `1 e4 e5 2 Nf3 Nc6
   3 Bc4 Bc5`, including central `c3` and `d4` play, quieter `d3` systems,
   tactical branches, positional branches, and related notes.
2. Evans Gambit, printed pages 26-29 (PDF pages 43-46): `1 e4 e5 2 Nf3 Nc6
   3 Bc4 Bc5 4 b4`, including accepted, declined, and principal defensive
   setups. It is an optional aggressive White repertoire branch.
3. Two Knights' Defence, printed pages 30-41 (PDF pages 47-58): `1 e4 e5
   2 Nf3 Nc6 3 Bc4 Nf6`, including `4 Ng5`, quieter White choices, and Black's
   major defensive branches.

The course begins with a shared foundations lesson for `1 e4 e5 2 Nf3 Nc6
3 Bc4` and ends with mixed branch recognition and recall. Reference depth
retains all curated table columns and notes in scope. Quick and Standard select
subsets by minimum depth; they are not separate transcriptions.

## Child Experience

### Navigation

Add a `Learn Openings` card to `HomeHub`. It shows the next Italian lesson and
the number of opening reviews currently due. It is separate from tactical
training so a child always knows which kind of practice is starting.

`OpeningHub` shows:

- imported course title and White perspective;
- Quick, Standard, or Reference depth selector;
- overall progress and due review count;
- ordered chapter and lesson cards;
- continue and start-review actions; and
- a secondary variation-explorer action.

Changing depth only changes visible lessons and branches. It never deletes or
forks progress.

### Guided Staircase

The primary lesson screen reuses the current board-first layout. Its side panel
shows chapter, variation label, current step, objective, depth, feedback,
reference-note access, hints, and pause or continue actions.

The five step types behave as follows:

1. Explain shows the starting position, concise idea, plan, and variation
   label.
2. Watch animates the authoritative course moves with SAN notation and optional
   notes at marked positions.
3. Try automates Black's course response and asks the learner to play White's
   repertoire move with progressive guidance.
4. Branch presents a meaningful Black response and asks the learner to
   recognize or answer the named branch.
5. Recall removes arrows and move prompts and asks the learner to reproduce a
   short path from a meaningful checkpoint.

The backend owns the authoritative FEN and applies every learner and automated
move. The frontend submits only UCI input and renders returned move frames, as
the puzzle controller does today.

### Feedback and Hints

The opening service distinguishes:

- the expected primary repertoire move;
- a sound alternative present in the course graph;
- another legal but off-course move; and
- an illegal move, which the board should normally prevent.

A legal off-course move is restored with neutral feedback such as: `That move
is playable, but this lesson is practicing 4.c3.` It is recorded as a retry for
this course prompt, not described as a chess blunder. A course alternative may
be acknowledged by name before returning to the marked repertoire line.

Hints are progressive:

1. reveal the relevant plan or course note;
2. highlight the source square;
3. highlight the destination square; and
4. show and animate the course move.

Hints and reveals affect opening mastery and review scheduling but never the
tactical learner rating.

### Variation Explorer

The explorer is a secondary reference workspace with the board, labeled branch
tree, movetext, notes, evaluations, and source references. Selecting a branch
updates the board. Transpositions show that multiple paths reach the same
position. Exploring does not create attempts or alter mastery unless the
learner explicitly starts an optional practice lesson from that branch.

## Opening Sessions, Progress, and Reviews

Opening sessions are separate from puzzle `SessionMode` and puzzle summaries.
An opening session stores course generation, course and lesson IDs, selected
depth, step index, current graph position, played path, hint state, retry state,
and timestamps. Save after every learner move and step transition.

Progress is keyed by stable course, lesson, and prompt IDs. Each prompt outcome
records first try, retried, hinted, revealed, or incomplete. Child-facing
summaries use positions recalled, branches recognized, and reviews due rather
than ratings.

Opening review intervals reuse 1, 3, 7, 21, and 60 days:

- a clean recall advances one interval;
- a wrong course move, hint, or reveal makes the prompt due the next day; and
- incomplete prompts retain their previous mastery and remain available to
  resume.

Reviews start from meaningful decision positions and request the repertoire
move or a short continuation. They do not always replay from move one. A mixed
review may first show Black's branch and then request White's response.

When a course update retains a prompt ID and semantic fingerprint, mastery and
review state remain. If the fingerprint changes, retain the history but reset
that prompt for relearning. Removed prompts are archived from active review;
they are not silently reassigned to another position.

## Application API Shape

Expose opening-specific application use cases rather than adding opening modes
to puzzle union types. The exact transport structs are finalized in the
implementation plan, but the service boundary includes:

- list courses and opening-home summaries;
- get or change course depth;
- list chapters and lessons;
- start or resume a lesson;
- apply an opening move;
- request a hint or reveal;
- advance or pause a lesson;
- start and answer due reviews;
- retrieve a position and branches for the explorer; and
- import, cancel, and report a course-pack job.

Every active lesson view includes an authoritative current FEN, orientation,
legal moves, display step, course labels, progress, and allowed actions. It does
not send the complete hidden expected path when doing so would reveal a recall
answer.

## Error Handling and Recovery

- Invalid packs never partially replace an active course.
- Import reports identify exact JSON paths and stable record IDs.
- Unknown schema versions fail with an actionable compatibility message.
- Illegal moves report the source position, UCI value, and source reference.
- Inconsistent transpositions report both incoming paths.
- Missing or invalid lesson references identify the lesson and step.
- A disappeared original pack does not affect already imported content.
- A missing course generation during resume returns the learner to the course
  hub and explains that the course must be reimported.
- A course update that invalidates the active lesson safely pauses that lesson
  and offers restart from its nearest retained checkpoint.
- Replaceable `courses.sqlite` corruption prompts quarantine and reimport while
  preserving `user.sqlite` and normal puzzle use.
- Board or animation failure uses the existing accessible error and reduced-
  motion fallbacks.

## Testing and Verification

### Backend

- Course-pack decoding and strict schema validation.
- Stable-ID, enum, text, and source-reference validation.
- Legal move application, derived SAN, FEN canonicalization, and root
  reachability.
- Valid transpositions, inconsistent transpositions, cycles, and dangling
  graph references.
- Quick, Standard, and Reference cumulative filtering.
- Repertoire uniqueness and alternative-move behavior.
- Staging, cancellation, activation, rollback, and generation cleanup.
- Content-update preservation and prompt-fingerprint invalidation.
- Compatible session rebase, incompatible-session pause, and old-generation
  cleanup protection.
- Explain, watch, try, branch, and recall sequencing.
- Authoritative move frames, neutral off-course feedback, hints, reveal, pause,
  and exact resume.
- Review outcomes and 1, 3, 7, 21, and 60-day interval transitions.
- Course corruption isolation from puzzles and user history.

### Frontend

- Runtime decoding for opening API responses.
- Home card, opening hub, depth selector, chapter progress, and due reviews.
- Opening controller state transitions and stale-request ownership.
- Board orientation, automated Black replies, animation ordering, and reduced
  motion.
- Expected, alternative, and off-course feedback.
- Hint markers, note display, pause, resume, and completion summaries.
- Variation explorer navigation and transposition indicators.

### End-to-End and Content Acceptance

Playwright covers:

1. import a synthetic course;
2. select Reference depth;
3. start an Italian-style lesson;
4. watch a line and play an expected move;
5. play a legal off-course move and retry;
6. use a hint;
7. close and resume at the exact position;
8. finish the lesson and create a review;
9. complete the review;
10. change to Quick depth without losing progress; and
11. reject a bad update while keeping the active course.

The private Italian acceptance pass additionally verifies every captured move
as legal and compares chapter names, table columns, notes, evaluations,
transpositions, and printed-page references against PDF pages 35-58. Existing
Go, frontend, Playwright, build, and license checks must remain green.

## Success Criteria

- A private `.ctcourse` file imports without requiring network access or OCR.
- The learner can complete and resume the full guided staircase from White's
  perspective.
- Quick, Standard, and Reference are cumulative filtered views of one course
  graph.
- Giuoco Piano, Evans Gambit, and Two Knights' Defence content in scope is
  traceable to printed pages 18-41.
- Named variations, notes, evaluations, and transpositions are available in the
  explorer.
- Every imported move is legal and every recall prompt has one marked White
  repertoire move.
- Legal alternatives receive neutral course feedback.
- Missed and hinted positions enter spaced review without changing puzzle
  rating.
- Course updates preserve unchanged progress and never partially activate.
- Puzzle training continues to behave exactly as before.
- No MCO-derived course content enters the repository or release artifacts.

## Deferred Extensions

- A Black-side Italian track.
- Additional opening-family packs and an all-book index.
- An in-app repertoire editor.
- PGN import into a draft course graph.
- Assisted OCR with human verification.
- ECO classification and cross-course transposition search.
- Optional engine-backed position checks behind the existing future analysis
  boundary.
