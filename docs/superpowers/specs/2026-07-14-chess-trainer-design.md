# Local-First Chess Trainer Design

Date: 2026-07-14
Status: Approved design, awaiting written-spec review

## Summary

Build a child-friendly macOS chess trainer that launches as a normal `Chess Trainer.app`, works without hosting or an internet connection, imports large local puzzle collections, tracks one learner's progress, and provides a basic historical-game library.

The application will use Wails with a Go backend and a Svelte/TypeScript frontend. It will store content and progress in local SQLite databases. Version one will support the downloaded Lichess `.csv.zst` puzzle database, tactical puzzles encoded as PGN, and ordinary PGN game libraries.

An engine is outside version-one scope. The core will define an analysis-provider boundary so later extensions can connect to separately installed UCI executables or remote HTTP services without bundling or implementing an engine.

## Goals

- Launch by double-clicking a normal macOS `.app` with no Terminal use after installation.
- Work fully offline after source files have been obtained.
- Give a ten-year-old developing player an uncluttered guided-puzzle experience.
- Offer both an adaptive daily session and parent-configured free practice.
- Preserve progress across restarts, puzzle catalogue updates, and source reimports.
- Import Lichess puzzles directly from compressed CSV without first expanding the file.
- Import supported tactical PGNs as puzzles.
- Import ordinary PGNs into a searchable, annotated game viewer.
- Keep the domain core independent of Wails so browser access over Tailscale can be added later.
- Keep all engine-specific behavior behind external-provider adapters.

## Version-One Non-Goals

- Bundling, implementing, or downloading a chess engine.
- Showing engine evaluation or engine-generated explanations.
- Hosting the application or exposing a network listener.
- Importing the `rebeccaloran/432k-chess-puzzles` FEN collection.
- Turning arbitrary FEN positions into solved puzzles.
- Editing PGNs or authoring lessons.
- Pause-and-guess historical-game lessons.
- Accounts, cloud sync, telemetry, or multiple learner profiles.
- Online game or puzzle fetching.

## User and Product Model

The app has one learner profile and a visually separate parent/settings area. It does not require authentication or a PIN in version one because it is a single-household local application.

The child-facing home hub has one dominant card for today's guided session and secondary entries for Free Practice and Game Library. The child does not see a persistent numerical rating. The parent area shows progress details and configuration.

## Architecture

### Runtime

- **Desktop shell:** Wails macOS application bundle.
- **Frontend:** Svelte and TypeScript.
- **Backend:** Go.
- **Persistence:** SQLite files beneath `~/Library/Application Support/Chess Trainer/`.
- **Secrets:** macOS Keychain for future remote-engine credentials.
- **Normal networking:** none in version one.

The frontend communicates with a thin Wails binding layer. Bindings delegate to application services and contain no scheduling, parsing, or persistence rules. Core services must not depend on Wails. A later authenticated HTTP adapter can therefore expose the same use cases over a Tailscale network.

Frontend chessboard and parsing dependencies must have licenses compatible with private use and possible later distribution. Do not adopt GPL-only UI code, including Chessground, unless the project explicitly chooses to distribute the combined application under the GPL.

### Services

- `PuzzleCatalog` imports, validates, deduplicates, searches, and retrieves puzzles.
- `TrainingService` creates and resumes sessions, validates solution progress, records attempts, updates review state, and adjusts the learner estimate.
- `GameLibrary` imports, indexes, searches, retrieves, and exports PGN games.
- `ProfileService` manages the single learner's configuration and summary statistics.
- `BackupService` exports and restores user data and optionally the game library.
- `AnalysisProvider` defines the future external-engine contract.

Each service exposes a narrow application-facing interface. File parsing, SQLite access, Wails bindings, and future engine transports are adapters around the service layer.

## Canonical Content Model

### Puzzle

A canonical puzzle contains:

- Stable SHA-256 fingerprint derived from the displayed FEN and normalized solution tree.
- Source identifier and source-local puzzle identifier when available.
- Optional source FEN and presentation prelude.
- Exact FEN to show when the learner is asked to move.
- Solver color.
- A solution tree expressed as normalized UCI moves.
- Optional source rating.
- Normalized themes plus preserved source-specific tags.
- Optional popularity, play count, URL, attribution, and raw metadata.

The solution is a tree rather than a flat list. A node contains a legal move and zero or more legal continuation nodes. Linear sources produce a single branch. This permits future sources with multiple accepted moves or variations without changing the trainer. For mate-in-one puzzles, any legal checkmating move is accepted even when the source lists only one.

Source provenance is stored separately from the canonical puzzle so duplicates from multiple sources can share one training identity while retaining attribution.

### Game

A game contains:

- Exact original PGN record.
- Parsed headers.
- Starting position.
- Main line and recursive annotation variations.
- Comments and numeric annotation glyphs.
- A normalized main-line signature for searching and comparison.

Only byte-equivalent or semantically identical records are skipped. Semantic identity is the SHA-256 hash of a canonical PGN serialization containing normalized headers, starting position, full move tree, comments, and annotation glyphs. Differently annotated versions of the same moves therefore remain separate library entries.

### Attempt and Review State

Each attempt records the puzzle fingerprint, session, timestamps, source, completion state, incorrect-move count, hints used, duration, and whether it was solved first try.

Review state records the next due time, current interval, successful review count, and last outcome. It uses the stable puzzle fingerprint rather than a catalogue row ID.

## Import Architecture

All importers implement a shared streaming contract:

1. Probe a selected file and report likely content type.
2. Parse records without loading the entire source into memory.
3. Normalize each record into a puzzle or game.
4. Validate positions, moves, solver color, and required metadata.
5. Write valid records to a staging catalogue in batches.
6. Count duplicates and rejected records with reasons and source locations.
7. Activate the staged content only after the import completes successfully. Activation replaces only the matching source partition and leaves other imported sources intact.

Cancellation or failure discards staging data and preserves the active catalogue. The importer checks free disk space before starting, reports progress, and produces a final accepted/duplicate/rejected summary with representative errors.

### Lichess CSV/Zstandard

`LichessCsvZstImporter` streams Zstandard decompression directly into CSV parsing.

Lichess's FEN represents the position before the opponent's setup move. The importer:

1. Loads the source FEN.
2. Validates and applies the first UCI move.
3. Stores that move as an optional presentation prelude.
4. Stores the resulting position as the displayed FEN.
5. Converts the remaining UCI moves into the solution branch beginning with the learner's move.

It retains Lichess rating, popularity, play count, themes, opening tags, puzzle ID, and source URL.

### Tactical PGN

`TacticsPgnImporter` supports the conventions used by `xinyangz/chess-tactics-pgn`:

- `[SetUp "1"]` and `[FEN "..."]` establish the initial position.
- `[White "solver"]` or `[Black "solver"]` identifies the learner's side.
- If the initial side to move is not the solver, the first main-line move becomes the presentation prelude.
- The remaining main line becomes the solution branch.

Records with illegal positions, illegal moves, missing solver identity, or no learner move are rejected with a reason. Tactical PGNs do not contain reliable ratings or themes, so their puzzles are available in Free Practice but do not enter the unseen-puzzle pool for adaptive guided sessions. Once attempted, missed or hinted tactical-PGN puzzles may enter the review queue.

### Ordinary PGN

`GamePgnImporter` supports one or many games per file. It preserves headers, starting FEN, comments, nested variations, and annotation glyphs, and stores the original record for later export.

PGNs with recognized tactical setup and solver markers are routed to the tactical importer. Clearly ordinary PGNs go to the game library. An ambiguous file prompts the parent to choose between puzzle and game import.

### Canonical Interchange

A documented canonical JSON puzzle format is a post-version-one extension. It will mirror the canonical puzzle fields and solution tree and become the preferred extension point for external converters.

New source formats are added as ordinary Go importer adapters, not dynamically loaded plugins. A generic configurable CSV importer, EPD best-move importer, and standalone-position library are later possibilities.

### Deferred FEN Collection

The `rebeccaloran/432k-chess-puzzles` collection is explicitly deferred. Its records generally contain board placement without reliable side-to-move or solution data, may have multiple solutions, and include fairy positions. It is unsuitable for engine-free guided training. A future position-library feature may accept validated standard-chess FEN records and pass them to an external analysis provider.

The application imports user-supplied source files and does not bundle third-party databases whose redistribution license has not been established.

## Child Experience

### Home Hub

The app opens to the approved simple home hub:

- Large `Continue today's training` card with completed/total progress.
- `Free Practice` secondary action.
- `Game Library` secondary action.
- Small parent/settings entry.

An interrupted session resumes from this card.

### Board-First Solver

The approved puzzle screen prioritizes the chessboard. The side panel contains only solver color, `Find the best move`, session progress, `Give me a hint`, and `Pause session`.

Puzzle flow:

1. Orient the board toward the solver.
2. Animate the stored presentation prelude when present.
3. Accept drag-and-drop or click-source/click-destination input.
4. On a correct move, highlight it briefly and play the recorded reply.
5. On an incorrect move, restore the piece, show `Try again`, and mark the puzzle for later review without ending the attempt.
6. Continue until the solution branch is complete.
7. Offer a short replay with SAN notation and source themes.

The app saves the active session and puzzle state after every learner move.

Hints are progressive:

1. Reveal a source theme when available; otherwise show a generic checks/captures/threats prompt.
2. Highlight the source square of the expected move.
3. Highlight its destination square.

After the third hint, the hint action becomes `Show solution`. Revealing the line records an unsuccessful attempt, replays the solution, and schedules the puzzle for the next day. This escape remains hidden until the progressive hints have been used, keeping the normal board-first screen uncluttered.

Hints are recorded. The app does not invent explanatory prose without an engine.

The session summary reports first-try solutions, retry solutions, hint use, revealed solutions, and incomplete puzzles using neutral, encouraging language.

## Guided Scheduling and Progress

A guided session defaults to ten puzzles and is resumable. The parent may configure 5, 10, or 15.

- Select up to four currently due review puzzles.
- Fill remaining places with unseen rated Lichess puzzles within 100 rating points of the learner estimate. If necessary, widen the window in 100-point steps to a maximum of 400 points.
- Rank eligible candidates by source quality signals such as popularity and play count.
- Avoid recently seen fingerprints.
- Carry excess overdue reviews into later sessions instead of lengthening today's session.

Review intervals are 1, 3, 7, 21, and 60 days. A missed or hinted puzzle is due the next day. Each clean successful review advances one interval. Any later mistake resets the interval to one day.

The learner estimate starts from a parent-selected value and is clamped to the minimum and maximum rated puzzles in the active Lichess catalogue. It changes only for the first attempt on an unseen rated puzzle in a guided session:

```text
expected = 1 / (1 + 10 ^ ((puzzleRating - learnerRating) / 400))
newRating = learnerRating + 24 * (score - expected)
```

`score` is 1.0 for first-try success without hints, 0.5 for eventual success after a wrong move or hint, and 0.0 when the solution is revealed. Reviews and Free Practice do not alter the estimate.

The child sees daily progress and session summaries. The parent sees rating trend, first-attempt accuracy, hint usage, theme performance, review backlog, and recent sessions.

Free Practice permits source, rating, theme, and puzzle-length filters. Its attempts are saved and can create review items, but they do not affect adaptive difficulty.

## Historical Game Library

The approved all-in-one workspace has three resizable panes:

- Searchable game list.
- Board and move controls.
- Movetext, comments, and variations.

Search and filters cover players, event, site, date/year, result, and opening tags when present. Navigation supports previous/next move, first/final position, left/right arrow keys, clicking notation, choosing variations, and flipping the board.

Version one does not edit annotations, autoplay games, fetch games online, evaluate positions, or create questions from historical games.

## Persistence and Backup

Use three databases:

- `puzzles.sqlite` for the large puzzle catalogue and source/provenance metadata.
- `library.sqlite` for imported games and original PGNs.
- `user.sqlite` for profile settings, attempts, sessions, learner estimate, and review queue.

Migrations are versioned and run before services become available. Before a destructive migration, the app creates a timestamped local backup.

The default backup contains `user.sqlite` plus a manifest. The parent can optionally include `library.sqlite`. The replaceable puzzle catalogue is not included. Restore validates versions and integrity before swapping files.

## External Analysis Boundary

Version one defines but does not implement this interface:

```text
AnalysisProvider
  capabilities() -> supported limits and options
  healthCheck() -> availability
  analyse(position, limits, options) -> streamed analysis updates
  cancel(analysisID)
```

Updates may contain depth, centipawn or mate score, nodes, and one or more principal variations.

Future adapters:

- `UciAnalysisProvider` launches a separately installed local executable, performs the UCI handshake, and uses only capabilities the engine advertises.
- `HttpAnalysisProvider` calls a configured remote service over HTTPS or a private Tailscale address and stores any token in macOS Keychain.

Analysis is optional and separate from puzzle correctness. Initial engine-enabled surfaces are a game-viewer analysis panel, post-solution puzzle analysis, and future standalone-position analysis. Provider crashes, malformed output, cancellation, and timeouts produce `Engine unavailable` without affecting training or stored content.

## Failure Handling

- Invalid individual import records are quarantined in the report while valid records continue.
- A failed, cancelled, or interrupted import never replaces active content.
- Database operations that update attempts, review state, and session progress are transactional.
- If a source puzzle disappears after reimport, its historical attempts remain visible; it is omitted from future scheduling until the fingerprint returns.
- If the active puzzle is unavailable after a catalogue change, the session marks it unavailable and advances without treating it as a learner failure.
- App startup detects database integrity or migration failures and offers restore-from-backup or open-data-folder actions rather than silently resetting progress.
- Future engine failures are isolated from the UI and databases by the provider adapter.

## Privacy and Network Posture

- No accounts, telemetry, analytics, or cloud storage.
- No HTTP listener in version one.
- Imported content and progress remain under the user's application-support directory.
- Future Tailscale browser access is opt-in, authenticated, and implemented as a new adapter over existing application services.
- Future remote-engine credentials are never stored in the ordinary SQLite settings tables.

## Testing Strategy

### Unit and Fixture Tests

- Lichess setup semantics for both solver colors.
- Castling, promotion, en passant, checkmate, and mate-in-one alternative acceptance.
- Tactical PGN solver tags, optional prelude, comments, nested variations, and invalid records.
- Ordinary PGN preservation and exact-duplicate handling.
- Puzzle normalization and stable fingerprinting.
- Scheduler selection with fixed clocks and random seeds.
- Learner-rating calculation and clamping.
- Review interval advancement and reset.
- Analysis-provider contract with a fake provider only.

### Integration Tests

- Stream a compressed Lichess fixture into a staged catalogue.
- Cancel and interrupt imports without replacing active content.
- Import mixed-validity PGNs and verify the report.
- Create, pause, restart, and resume a guided session.
- Reimport a source and retain fingerprint-linked progress.
- Import, search, and navigate annotated historical games.
- Backup and restore user data.

### End-to-End and Packaging Tests

- First launch and parent-selected initial rating.
- File-picker import through the Wails UI.
- Home hub to guided session to review queue.
- Free-practice tactical PGN flow.
- App termination during a puzzle followed by exact resume.
- macOS `.app` build and double-click launch smoke test.

### Performance Expectations

- Import memory use is bounded and does not grow with the decompressed source size.
- The UI remains responsive and cancellable during full Lichess import.
- After import, app startup reaches the home hub within three seconds on the target Mac.
- Selecting the next puzzle or executing a normal indexed library search completes within 250 ms under typical local load.

## Delivery Sequence

1. Wails shell, service boundaries, databases, migrations, and packaging smoke test.
2. Canonical puzzle model and Lichess streaming importer.
3. Board-first solver, attempt recording, and session resume.
4. Guided scheduler, review queue, parent progress, and Free Practice.
5. Shared PGN parser, tactical PGN importer, and import reports.
6. Historical game library and all-in-one viewer.
7. Backup/restore and full-database performance hardening.
8. Analysis-provider interface and fake-provider contract tests, without real adapters.

## Acceptance Criteria

- A user can double-click `Chess Trainer.app`, import the supplied Lichess `.csv.zst`, and train without a Terminal, host, account, or internet connection.
- The child can complete, pause, and resume guided sessions with correct setup-move and solution behavior.
- The daily session combines due reviews and suitable unseen rated puzzles according to this specification.
- Progress survives app restarts and source reimports.
- The referenced tactical PGN convention imports into Free Practice and review scheduling.
- Ordinary annotated PGNs import into the searchable three-pane game viewer without losing comments or variations.
- A malformed record or interrupted import does not corrupt active content or learner progress.
- Version one contains no engine binary, engine implementation, network listener, or FEN-derived unsolved puzzle mode.

## References

- [Lichess open puzzle database and format notes](https://database.lichess.org/)
- [Wails documentation](https://wails.io/docs/introduction/)
- [xinyangz/chess-tactics-pgn](https://github.com/xinyangz/chess-tactics-pgn)
- [rebeccaloran/432k-chess-puzzles](https://github.com/rebeccaloran/432k-chess-puzzles)
