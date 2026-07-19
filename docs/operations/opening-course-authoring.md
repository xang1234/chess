# Private opening-course authoring

Chess Trainer imports opening material from a local `.ctcourse` file. A course
pack is strict UTF-8 JSON: unknown fields, trailing JSON values, illegal chess
moves, and incomplete source coverage are rejected before the active course is
changed. Keep copyrighted or otherwise private packs outside this repository.
Only original synthetic fixtures belong under `internal/openings/testdata/`.

## Schema version 1

The root object requires `schemaVersion: 1`, a stable `courseId`, a non-empty
`contentVersion`, title, description, perspective, default depth, root position
ID and FEN, source metadata, source coverage, and the seven collections shown
below. The importer accepts at most 32 MiB and rejects unknown JSON fields.

```json
{
  "schemaVersion": 1,
  "courseId": "synthetic-open-game",
  "contentVersion": "1.0.0",
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
  "chapters": [],
  "lessons": [],
  "prompts": []
}
```

`courseId`, all record IDs, and lesson step IDs must match
`^[a-z0-9][a-z0-9._-]{0,127}$`. IDs are durable identity, not display text:
keep them stable when publishing a new `contentVersion`. IDs must be unique
within their record collection; step IDs must be unique within their lesson.
Chapter ordinals are positive and unique, and lesson ordinals are positive and
unique within a chapter.

Positions name graph nodes; their FENs are derived from `rootFen` and the move
edges rather than entered independently. A move uses UCI, points from one
position ID to another, and carries an evaluation, minimum depth, training
role, optional notes/variation label, and source reference. Reuse the same
destination position ID when move orders transpose. The compiler rejects
cycles, unreachable nodes, illegal UCI, duplicate outgoing UCI, and move orders
that derive inconsistent versions of one transposition.

## Depth, roles, and lessons

Depth is cumulative in this exact order:

1. `quick` — the smallest usable repertoire.
2. `standard` — Quick plus major branches.
3. `reference` — Standard plus all captured detail.

A visible move cannot depend on a parent hidden at the selected depth. A lesson
cannot be shallower than its chapter, and its positions and moves must be
visible at the lesson's minimum depth. This makes Quick and Standard strict,
connected subsets of Reference.

Set `trainingRole` from the learner's perspective:

- `repertoire` is the one primary course move when the learner is to move;
- `alternative` is another acceptable learner move; and
- `opponent` is required for every edge where the opponent is to move.

A prompt links one position to exactly one primary repertoire move and zero or
more accepted alternative moves from that same position. Each lesson must run
in `explain`, `watch`, `try`, `branch`, `recall` order, include all five kinds,
begin with Explain, and end with Recall. Watch contains a connected move list.
Try, Branch, and Recall reference a prompt and may include at most one automatic
opponent continuation.

Supported evaluation codes are `none`, `equal`, `unclear`, `white_slight`,
`black_slight`, `white_clear`, `black_clear`, `white_winning`, and
`black_winning`. Supported note kinds are `overview`, `history`, `plan`,
`warning`, `explanation`, `evaluation`, `transposition`, and
`illustrative_game`.

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

Success prints the course/content IDs, structural counts, and a coverage report
whose `missing` and `unexpected` arrays are empty. Failure prints sorted JSON
diagnostics with `code`, `path`, and `message`; correct structural or chess
errors before adding more content. The original miniature example is
`internal/openings/testdata/mini.ctcourse`.

In the app, open **Parent settings** → **Import content** → **Choose opening
course**, review the detected course ID/title/source, and choose **Import
course**. Import compiles the whole pack before atomically activating it. The
source file remains external; the validated course is stored in the local
`courses.sqlite` catalogue.

## Updating a course safely

Importing the same `courseId` creates and activates a new generation; do not
change the course ID merely to publish a revision. Stable lesson, step, prompt,
position, and move identities let compatible paused lessons and reviews rebase
to the new generation. Incompatible active work is marked **Restart from
checkpoint**. Removed prompts are archived from the active review queue, while
durable lesson and attempt history remains in `user.sqlite`. If course data is
missing, the app asks for the private pack to be reimported.

`courses.sqlite` is replaceable local content. On startup, an invalid or
incompatible course catalogue and its WAL/SHM files are moved to a timestamped
`.quarantine-<UTC>` path and a clean catalogue is created; `user.sqlite` is not
discarded. Reimport the external packs after a quarantine.

Before any public build, `scripts/verify-release.mjs` rejects every tracked
`.ctcourse` outside `internal/openings/testdata/`. Never copy a private pack
into the repository, build staging tree, or release archive.
