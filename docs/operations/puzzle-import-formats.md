# Puzzle collection import formats

Chess Trainer imports local puzzle collections from Parent settings under
**Import puzzles**. Choose **Choose puzzle collection**, review the detected
format and source ID, then start the import. The confirmation is authoritative:
the app inspects file content, normalizes the absolute path, and shows the
source ID before it changes the catalogue. Starting the import inspects the file
again so a changed file is not imported under stale inspection details.

The native chooser accepts `.zst`, `.pgn`, `.json`, `.fns`, and `.txt`. An
extension is only a hint for otherwise ambiguous text. Content must match one of
the grammars below; renaming a file does not change its detected format.

## Source IDs and replacement identity

The source ID identifies the active collection. Reimporting that ID replaces
only its active generation; importing a different ID creates or replaces a
different collection.

| Format | Source-ID rule | Confirmation label |
| --- | --- | --- |
| Lichess zstd/CSV | Always `lichess` | Fixed source ID |
| Canonical JSON | Trimmed, non-empty `source.id`; otherwise the normalized absolute path | Embedded source ID or fallback file path |
| Tactical PGN | Trimmed, non-empty `SourceId` from the first game carrying a non-empty `FEN` tag; otherwise the normalized absolute path | Embedded source ID or fallback file path |
| Lucas `.fns` | Normalized absolute path | Fallback file path |
| Linear FEN/UCI | Normalized absolute path | Fallback file path |

Path normalization cleans the path, makes it absolute, and resolves symlinks
when the operating system can resolve them. That exact normalized string is the
source ID for path-based sources. Moving a path-based file therefore creates a
different logical source. Any PGN game may omit `SourceId`, but every explicit
value anywhere in the file must match the identity resolved from the first game
carrying a non-empty `FEN` tag. Reusing an existing source ID with a different
format is rejected because source kind is immutable.

## Lichess zstd/CSV

A Lichess input is a zstd stream whose decompressed first CSV record contains
all of these uniquely named columns (the order may differ and additional
columns are allowed):

```text
PuzzleId,FEN,Moves,Rating,RatingDeviation,Popularity,NbPlays,Themes,GameUrl,OpeningTags
```

`Moves` is whitespace-separated UCI. Its first move is the opponent's setup
move from `FEN`; at least one solution move must remain. The importer validates
the complete line, preserves the source position and setup move for
presentation, and imports the remaining moves as one solution branch. The zstd
magic and required CSV header—not `.zst` alone—identify this format.

## Tactical PGN

Each PGN game is one puzzle:

- `FEN` is required. `SetUp`, when present, must be `1`.
- Exactly one of `White` and `Black` must equal `solver`, ignoring case and
  surrounding whitespace. That player determines the solver color.
- If the FEN already has the solver to move, every mainline move is part of the
  solution. If the opponent is to move, the first mainline move is the prelude;
  the position after it is displayed to the solver.
- At least one solution move must remain after a prelude.
- `PuzzleId` is the external ID. When omitted, the one-based game number is
  used.
- Comments, NAGs, clock annotations, variations, and the result do not add
  accepted solution branches. Only the mainline is imported.

This complete synthetic game begins with Black to move. `1... e4` is the
prelude, so the learner sees the resulting position with White to move and must
play `2. Kf2`:

```pgn
[Event "Opponent prelude example"]
[SourceId "club-tactics"]
[PuzzleId "king-step-1"]
[SetUp "1"]
[FEN "4k3/8/8/4p3/8/8/4P3/4K3 b - - 0 1"]
[White "solver"]
[Black "?"]

1... e4 2. Kf2 *
```

## Canonical JSON

The top level must be one JSON object with `schema` exactly
`chess-trainer-puzzles/v1` and a `puzzles` array. `source` is optional. Unknown
or duplicate structural keys in the top-level, source, puzzle, and move objects,
and a different schema are rejected rather than silently ignored. Keys inside
the arbitrary `metadata` object are not structural schema keys.

This is a complete valid object with source defaults, a branched solution, and
an optional trainer rating:

```json
{
  "schema": "chess-trainer-puzzles/v1",
  "source": {
    "id": "club-tactics",
    "name": "Club tactics",
    "url": "https://example.invalid/club-tactics",
    "attribution": "Example club coach"
  },
  "puzzles": [
    {
      "id": "branch-42",
      "displayedFen": "4k3/8/8/8/8/8/4P3/4K3 w - - 0 1",
      "solver": "white",
      "solution": [
        {
          "uci": "e2e4",
          "children": [
            { "uci": "e8e7", "children": [] },
            { "uci": "e8f7", "children": [] }
          ]
        }
      ],
      "rating": 1450,
      "themes": ["fork"],
      "popularity": 80,
      "playCount": 250,
      "url": "https://example.invalid/club-tactics/branch-42",
      "attribution": "Example analyst",
      "metadata": {
        "chapter": 3,
        "note": "synthetic example"
      }
    }
  ]
}
```

For each puzzle, `displayedFen`, `solver`, and a non-empty `solution` are
required. The solver must match the FEN's active color. Every `uci` move is
normalized and legally replayed from its parent position; sibling moves are
alternatives. `id` defaults to the one-based puzzle number. `rating`, when
present, must be an integer from 100 through 4000. `popularity` and `playCount`
must be non-negative integers. `themes` contains strings, and `metadata` is the
only arbitrary-field extension point.

Optional `sourceFen` and `preludeUci` must appear together. The prelude must be
legal from `sourceFen` and must produce exactly the normalized `displayedFen`.
A puzzle `url` or `attribution` overrides the corresponding source default.

## Lucas `.fns`

Each non-empty, non-comment line has exactly three logical fields. Only the
first two `|` characters delimit fields, so later `|` characters belong to the
movetext:

```text
<six-field FEN>|<description>|<SAN/PGN movetext>
```

The FEN's active color is the solver. Movetext is parsed from that position;
the mainline and legal recursive annotation variations become solution
branches. At least one legal solver move is required. The one-based physical
line number is the external ID. Blank lines and lines whose first non-space
character is `#` are ignored.

This example imports two Black replies as alternatives after `1. e4`:

```text
4k3/8/8/8/8/8/4P3/4K3 w - - 0 1|Difficulty **|1. e4 Kf7 (1... Kd7) 2. Kf2 *
```

The trimmed description is stored as plain-text metadata. A recognizable,
normalized filename stem supplies a theme; generic stems such as `training`,
`tactics`, and `puzzles` do not. Text such as `Difficulty **` or
`Difficulty 1500` is retained as `sourceDifficulty` metadata only.

## Linear FEN/UCI

Each non-empty, non-comment line follows this exact whitespace-separated
grammar:

```text
<six-field FEN> <uci1> [uci2 ...] [difficulty]
```

The first six tokens form the FEN. One or more legal UCI moves form one linear
solution branch. If the final token is an integer, it is removed from the move
line and stored as `sourceDifficulty`; at least one UCI move must still remain.
Promotion letters are normalized to lowercase. Inline comments are not part of
the grammar. Blank lines and lines whose first non-space character is `#` are
ignored. The one-based physical line number is the external ID.

```text
# Synthetic examples
4k3/8/8/8/8/8/4P3/4K3 w - - 0 1 e2e4 e8f7 1375
7k/P7/8/8/8/8/8/K7 w - - 0 1 a7a8Q
```

## Rating and difficulty

Ratings and source difficulty are deliberately different:

| Value | Stored as a trainer rating? | Guided scheduling and learner bounds |
| --- | --- | --- |
| Lichess `Rating` | Yes | Eligible when non-null |
| Canonical JSON `rating` | Yes, after validation in the 100–4000 range | Eligible when non-null |
| Tactical PGN annotations or tags | No | Unrated |
| Lucas description difficulty | No; `sourceDifficulty` metadata only | Unrated |
| Linear final integer | No; `sourceDifficulty` metadata only | Unrated |

Any active occurrence with a validated, non-null trainer rating can participate
in guided sessions and learner-rating bounds. Unrated puzzles remain available
in free practice and through source, theme, and solution-length filters. The
importer never converts a filename, annotation, or difficulty value into a
rating.

## Atomic replacement, cancellation, and reports

An import builds a hidden generation while the previous active generation
remains readable. It calculates a checksum over the exact selected bytes,
validates records, seals and audits the new generation, then changes the source
head in one short activation transaction. Equal normalized cores from different
sources share canonical content while retaining separate occurrences.

- Reimporting the same resolved source ID replaces only that source's head.
- Cancellation abandons the building generation and leaves the previous head
  unchanged.
- A run with zero accepted puzzles is not sealed or activated.
- A fatal error—including conflicting PGN identity, unrecoverable syntax,
  interrupted reads, or catalogue failure—abandons all staged work, even after
  earlier records were accepted.
- Recoverable record-local errors reject that record and continue. The terminal
  report shows accepted, duplicate, and rejected counts plus at most 100
  plain-text examples containing the record ordinal and reason.

Only one puzzle import runs at a time. Progress reports inspection, parsing,
sealing, and activation phases and remains available after navigating away from
the import screen and returning during the same app run.

## Exact resource limits

| Limit | Exact maximum |
| --- | ---: |
| Tactical PGN game | 64 KiB (65,536 bytes) per game |
| Lucas or linear line record | 1 MiB (1,048,576 bytes) per line |
| Canonical JSON puzzle | 2 MiB (2,097,152 bytes) per puzzle object |
| Imported or derived metadata | 64 KiB (65,536 bytes) |
| Tactical PGN tag pairs | 128 per game |
| Solution depth | 256 moves on a root-to-leaf branch |
| Total solution nodes | 512 per puzzle, across all branches |

An over-limit record is rejected when its decoder can safely recover the next
record boundary. If the limit destroys safe framing, the error is fatal and the
entire building generation is abandoned.
