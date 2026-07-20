# Private opening-course authoring

Chess Trainer imports opening material from a local `.ctcourse` file. A course
pack is strict UTF-8 JSON: unknown fields, trailing JSON values, illegal chess
moves, and incomplete source coverage are rejected before the active course is
changed. Keep copyrighted or otherwise private packs outside this repository.
Only original synthetic fixtures belong under `internal/openings/testdata/`.

## Schema version 2

New courses should use `schemaVersion: 2`. The root requires a stable
`courseId`, a non-empty `contentVersion`, course metadata, source coverage, the
position/move graph, an authored `lessonEdges` teaching tree, chapters,
lessons with `activities`, and prompts.

```json
{
  "schemaVersion": 2,
  "courseId": "synthetic-open-game",
  "contentVersion": "2.0.0",
  "title": "Synthetic Open Game",
  "description": "Original example material.",
  "perspective": "white",
  "defaultDepth": "reference",
  "rootPositionId": "initial",
  "rootFen": "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1",
  "source": {
    "title": "Original synthetic notes",
    "edition": "1",
    "privateUseNotice": "Original example material."
  },
  "sourceCoverage": {
    "printedPages": [1],
    "expectedReferences": ["p1-e4"]
  },
  "positions": [],
  "moves": [],
  "notes": [],
  "lessonEdges": [],
  "chapters": [],
  "lessons": [],
  "prompts": []
}
```

`courseId` and all record IDs must match
`^[a-z0-9][a-z0-9._-]{0,127}$`. IDs are durable identity, not display text:
keep them stable when publishing a new `contentVersion`. IDs are unique within
their collection. Chapter ordinals are positive and unique; lesson ordinals
are positive and unique within a chapter; outgoing edge ordinals are positive
and unique beneath one parent.

Positions name graph nodes. Their FENs are derived from `rootFen` and move
edges rather than entered independently. A move uses UCI, points from one
position ID to another, and carries an evaluation, minimum depth, training
role, optional notes/variation label, and source reference. Reuse the same
destination position ID when move orders transpose. The compiler rejects
cycles, unreachable nodes, illegal UCI, duplicate outgoing UCI, and move orders
that derive inconsistent versions of one transposition.

## Teaching tree

`lessonEdges` makes the course journey explicit. Every lesson except the root
has exactly one parent; the complete course has one root, no cycles, and no
disconnected lessons. Children appear in authored `ordinal` order.

Supported edge kinds are:

- `continuation` — the normal next teaching decision;
- `alternative` — a labelled sibling plan or opponent choice; and
- `reference` — labelled deeper study that is not the main route.

Alternative and reference edges require a non-empty label. A child cannot be
shallower than its parent. An edge's `minimumDepth` cannot be shallower than
either endpoint, and every lesson visible at Quick, Standard, or Reference
must have a connected route at that depth. The single root must be Quick.

```json
{
  "edgeId": "central-plan-to-quiet-plan",
  "fromLessonId": "central-plan",
  "toLessonId": "quiet-plan",
  "ordinal": 1,
  "kind": "continuation",
  "minimumDepth": "standard"
}
```

## Decision-point lessons

A v2 lesson contains an ordered `activities` array. The six supported kinds
are:

- `concept` — explains one idea without requiring a move;
- `demonstration` — replays one connected authored move sequence;
- `decision` — asks for one course move using a prompt;
- `comparison` — contrasts two or more labelled connected move sequences;
- `recap` — consolidates the idea without asking for the same move again; and
- `reference` — optional deep material, kept collapsed in the lesson UI.

Use `required: true` for the shortest coherent learning journey. A lesson must
have at least one required activity. Reference activities must be optional.
Optional activities are still available at the appropriate course depth but
do not block completion.

Concept and recap activities cannot contain a prompt, move sequence, or
comparison. A demonstration requires `positionId` and at least one connected
`moveId`. A comparison requires `positionId` and at least two labelled,
connected lines. A decision requires `positionId` and a `promptId`; it may
include at most one automatic opponent continuation. Annotations may mark one
board `square` or draw an `arrow` between two different valid squares.

Teach each position/move answer once per lesson. The validator rejects two
required decisions that resolve to the same prompt position and primary move.
Prefer Concept → Decision → Recap for a small lesson; add Demonstration or
Comparison only when it teaches new information. Put dense analysis in one or
more optional Reference activities instead of turning each source note into a
learner action.

```json
{
  "lessonId": "central-plan",
  "chapterId": "plans",
  "ordinal": 1,
  "title": "Prepare the centre",
  "objectives": ["Connect c3 with a later d4"],
  "minimumDepth": "quick",
  "startPositionId": "after-bc5",
  "activities": [
    {
      "activityId": "central-concept",
      "kind": "concept",
      "title": "Build the centre",
      "instruction": "Prepare d4 without blocking the active bishop.",
      "required": true,
      "positionId": "after-bc5",
      "noteIds": ["central-plan-note"],
      "moveIds": [],
      "annotations": [{"kind": "arrow", "from": "c2", "to": "c3"}]
    },
    {
      "activityId": "central-decision",
      "kind": "decision",
      "title": "Choose the preparation",
      "instruction": "Play the move that supports d4.",
      "required": true,
      "positionId": "after-bc5",
      "noteIds": [],
      "moveIds": [],
      "promptId": "choose-c3"
    },
    {
      "activityId": "central-recap",
      "kind": "recap",
      "title": "Keep the plan",
      "instruction": "Develop, prepare d4, then choose the right moment to occupy it.",
      "required": true,
      "positionId": "after-c3",
      "noteIds": [],
      "moveIds": []
    }
  ]
}
```

## Depth, roles, and prompts

Depth is cumulative in this exact order:

1. `quick` — the smallest usable repertoire.
2. `standard` — Quick plus major branches.
3. `reference` — Standard plus all captured detail.

A visible move cannot depend on a parent hidden at the selected depth. A lesson
cannot be shallower than its chapter, and its positions and moves must be
visible at the lesson's minimum depth. This makes Quick and Standard strict,
connected subsets of Reference.

Set `trainingRole` from the learner's perspective: `repertoire` is the primary
course move when the learner is to move, `alternative` is another acceptable
learner move, and `opponent` is required where the opponent is to move. A
prompt links one position to one primary answer and zero or more accepted
alternatives from that same position. The primary answer normally has the
`repertoire` role. A lesson on an explicit alternative branch may instead use
that branch's `alternative` move as its primary answer without changing the
graph's single main-repertoire move.

Supported evaluation codes are `none`, `equal`, `unclear`, `white_slight`,
`black_slight`, `white_clear`, `black_clear`, `white_winning`, and
`black_winning`. Supported note kinds are `overview`, `history`, `plan`,
`warning`, `explanation`, `evaluation`, `transposition`, and
`illustrative_game`.

## Schema version 1 compatibility

Existing schema v1 packs remain importable. The importer normalizes their five
ordered `steps` (`explain`, `watch`, `try`, `branch`, `recall`) into stable v2
activity IDs and synthesizes a continuation tree from authored chapter/lesson
order. Do not add `activities` or `lessonEdges` to a v1 file. New authoring
should use v2 so the teaching tree and decision-point journey are intentional.

Stable lesson and activity IDs preserve completion across compatible course
updates. Reordering activities is safe when IDs and their meaning stay the
same. Removing or changing the meaning of an ID may require the learner to
restart from the nearest valid checkpoint.

## Source coverage

Inventory the source before transcribing it. Put every expected table column,
note, overview, and continuation in `sourceCoverage.expectedReferences`, and
declare every positive printed page in `printedPages`. Each move and note then
uses a source coordinate such as:

```json
{
  "printedPage": 1,
  "tableColumn": "A",
  "noteLabel": "overview",
  "coverageId": "p1-column-a"
}
```

One coverage ID may support multiple records only when all of its coordinates
match. Validation reports independently declared but uncaptured IDs as
`missing_coverage`, captured but undeclared IDs as `unexpected_coverage`, and
conflicting coordinates as `coverage_coordinate_conflict`.

## Validate and import

Validate after each small authoring batch from the repository root:

```bash
go run ./cmd/coursepack validate "/absolute/path/to/course.ctcourse"
```

Success prints course/content IDs, structural counts including `activities`
and `lessonEdges`, and a coverage report whose `missing` and `unexpected`
arrays are empty. Failure prints sorted JSON diagnostics with `code`, `path`,
and `message`. Synthetic examples are
`internal/openings/testdata/mini.ctcourse` (v1) and
`internal/openings/testdata/tree.ctcourse` (v2).

In the app, open **Parent settings** → **Import content** → **Choose opening
course**, review the detected course ID/title/source, and choose **Import
course**. Import compiles the whole pack before atomically activating it. The
source file remains external; the validated course is stored in the local
`courses.sqlite` catalogue.

## Updating a course safely

Importing the same `courseId` creates and activates a new generation; do not
change the course ID merely to publish a revision. Stable lesson, activity,
prompt, position, and move identities let compatible paused lessons and
reviews rebase to the new generation. Incompatible active work is marked
**Restart from checkpoint**. Removed prompts are archived from the active
review queue, while durable lesson and attempt history remains in
`user.sqlite`. If course data is missing, the app asks for the private pack to
be reimported.

`courses.sqlite` is replaceable local content. On startup, an invalid or
incompatible course catalogue and its WAL/SHM files are moved to a timestamped
`.quarantine-<UTC>` path and a clean catalogue is created; `user.sqlite` is not
discarded. Reimport the external packs after a quarantine.

Before any public build, `scripts/verify-release.mjs` rejects every tracked
`.ctcourse` outside `internal/openings/testdata/`. Never copy a private pack
into the repository, build staging tree, or release archive.
