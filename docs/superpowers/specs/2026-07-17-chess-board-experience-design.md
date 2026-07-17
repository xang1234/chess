# Chess Board Experience and Puzzle Completion Design

**Date:** 2026-07-17

## Purpose

Replace the current Unicode-piece, HTML-drag chess board with a polished, child-friendly puzzle experience built on Chessground. The new experience must feel familiar to a user of modern online chess sites while remaining fully offline-capable, preserving the Go backend as the rules authority, and complying with Chessground's GPL-3.0-or-later license.

The design addresses four concrete shortcomings in the current trainer:

- Piece appearance varies by platform and the board styling does not feel like a modern chess application.
- Native HTML drag-and-drop does not provide reliable touch behavior, animation, snapback, or clear move affordances.
- The board does not show legal destinations after selection.
- Completing a puzzle immediately replaces the final position with the next puzzle instead of giving the learner a clear finish and an explicit next action.

## Confirmed Product Decisions

- Use `@lichess-org/chessground` as the board interaction and rendering layer.
- License the complete combined application under GPL-3.0-or-later and keep its source repository public.
- Use the approved Classic Green visual direction.
- Support both click-to-move and drag-to-move, including touch input.
- Show legal-move dots on empty destinations and rings around capturable pieces.
- Keep the final solved position visible until the learner presses **Next puzzle**.
- Play restrained move, capture, correct, and incorrect sounds with a persistent mute control.
- Preserve the existing prohibition on implementing a chess engine. Legal move generation remains ordinary rules validation using the application's existing chess library.

## Scope

This change will:

- Replace the internal board implementation with a thin Svelte wrapper around Chessground.
- Add backend-generated legal UCI moves to each active puzzle view.
- Add response data needed to animate accepted moves, automatic replies, and revealed solution lines.
- Add an explicit puzzle interaction state machine.
- Preserve the completed FEN and defer the visible transition to the already-prepared next session.
- Restyle the puzzle screen around a board-dominant Classic Green layout.
- Add local sound feedback, mute persistence, keyboard operation, screen-reader announcements, and reduced-motion handling.
- Add GPL licensing, notices, in-app legal information, source traceability, and release checks.

This change will not:

- Add evaluation, best-move search, hints derived from analysis, or any other engine behavior.
- Copy Chess.com artwork, branding, sounds, source code, or proprietary assets.
- Change puzzle scheduling, rating, review, or session-size behavior.
- Change the puzzle or user database schemas.
- Add a remote hosting requirement or runtime network dependency.

## Approaches Considered

### Chessground

Chessground is selected. It provides mature click, drag, touch, orientation, animation, selection, destination, and snapback behavior while allowing the application to retain its own rules and puzzle logic. Its typed API can be wrapped directly rather than relying on an unmaintained Svelte-specific package.

Chessground is GPL-3.0-or-later. The project will intentionally adopt the same license for the combined distributed application and make matching source available to every application user.

### `cm-chessboard`

`cm-chessboard` is MIT-licensed and supports responsive SVG rendering and move input. It avoids the project-wide GPL obligation but requires more adapter code for the exact interaction and marker behavior. It remains the fallback if licensing policy changes later.

### Extend the custom board

Keeping the current board would avoid a runtime dependency, but reliable pointer capture, touch drag, animation, snapback, promotion, selection, and accessible keyboard behavior would all become application-owned infrastructure. This is higher risk and would duplicate mature board-widget behavior.

## Experience Design

### Visual direction

The board uses Classic Green squares:

- Light square: `#eeeed2`.
- Dark square: `#769656`.
- Selection: a high-contrast yellow treatment that remains distinct on both square colors.
- Last move: a softer yellow treatment than selection.
- Incorrect move: red source and destination treatment plus visible text.
- Hint source and destination: gold markers that do not resemble legal destination dots.

The application background and puzzle side panel use neutral charcoal tones. The board is the dominant element, with minimal decoration and no thick ornamental frame. Coordinates appear only on the outer board edges. Chessground's bundled Cburnett SVG piece presentation is used so pieces are crisp and consistent across macOS and browsers.

The visual language may use familiar online-chess interaction conventions, but it must not copy Chess.com logos, names, proprietary piece artwork, sounds, or other protected brand assets.

### Layout

On a normal desktop window, the screen contains:

- A square board sized from both available width and viewport height.
- A 300-340 pixel side panel containing turn, task, progress, feedback, sound control, hints, reveal, and pause actions.
- A compact success state that replaces ordinary actions after completion.

On a narrow browser or small window, the board uses the full available width and the panel moves below it. The layout must not depend on a fixed `vh` board size that can clip controls. All non-board controls retain a minimum 44-pixel target.

### Selection and legal destinations

The backend exposes every chess-legal UCI move for the current FEN. It must not expose only solution moves because that would reveal the puzzle answer.

The frontend groups legal UCI moves by source square and passes a `Map<source, destinations>` to Chessground:

- Only pieces with at least one legal move can begin an interaction.
- Selecting a piece applies the strong selected-square treatment.
- Empty destinations show dots.
- Occupied destinations show capture rings.
- Selecting the same square or pressing Escape cancels selection.
- Selecting another movable piece changes selection without submitting a move.

Chessground can otherwise select a piece of the movable color even when that piece has no destination entry. The adapter's selection event guard immediately clears such a selection. Pointer and keyboard tests cover this guard so an immobile piece never receives the selected treatment.

Promotion moves sharing the same source and destination remain distinct in the UCI list. Moving a pawn to a promotion rank opens an accessible queen, rook, bishop, and knight chooser restricted to the legal suffixes supplied by the backend. Cancelling promotion restores the authoritative position.

### Move behavior

The UI is optimistic only for the submitted piece movement. The backend remains authoritative for legality, puzzle correctness, persistence, and the resulting FEN.

For a legal but puzzle-incorrect move:

1. Chessground displays the attempted move.
2. The backend returns `correct: false` while preserving the prior authoritative position.
3. The board animates back to that position.
4. Source and destination receive the red incorrect treatment.
5. The incorrect sound plays and the panel announces **Try again**.
6. Input becomes available again after reconciliation.

For a correct move:

1. The submitted move remains visible.
2. Any automatic reply returned by the backend animates after a short, consistent pause.
3. Move or capture sound is played for each displayed move.
4. The board reconciles to the authoritative returned FEN after the sequence.
5. If the puzzle continues, new legal destinations replace the old ones and input reopens.

The existing puzzle prelude is shown only when `currentPath` is empty and `sourceFen` plus `preludeUci` are present. It starts at `sourceFen`, animates the prelude move, and reconciles to `displayedFen`. A resumed partially solved puzzle skips the prelude and starts directly at `currentFen`. With reduced motion enabled, both fresh and resumed puzzles start directly at their appropriate solver-facing FEN.

Reveal animates the complete remaining solution line rather than jumping to the final position. Input remains locked for the sequence. Reduced motion replaces the sequence with the final authoritative position immediately.

### Completion behavior

Puzzle completion is persisted immediately, as it is today, so a crash or close cannot lose progress. Presentation is deliberately deferred:

1. The backend captures and returns the completed puzzle's final FEN before preparing the next session view.
2. The frontend stores the returned session as `pendingSession` instead of rendering it.
3. The board reconciles to `finalFen`, clears destination markers, and locks input.
4. The side panel changes to **Correct!** or **Solution shown** and plays the appropriate completion feedback.
5. **Next puzzle** applies `pendingSession` locally with no extra backend mutation.
6. On the last item, the button reads **See results** and then displays the existing session summary.

If the application is closed on a non-final success state, reopening resumes from the already-persisted next puzzle. If it is closed on the final success state, reopening lands on the home screen because the session is already complete; its durable attempt and summary statistics remain recorded, but the transient results screen is not reopened. The success acknowledgement is presentation state, not a second durable completion transaction.

## Interaction State Model

`PuzzleScreen` uses a discriminated state model instead of overlapping booleans:

- `prelude`: the prior move is being shown; board input is locked.
- `ready`: the learner may select and move.
- `requesting`: a move, hint, or reveal request is in flight; board input is locked and the operation kind is explicit.
- `animating`: an accepted reply or reveal line is playing; board input is locked.
- `incorrect`: the board has reconciled and retry feedback is visible; input is available.
- `solved`: the final board and success panel are visible; only completion actions are available.
- `failed`: an actionable error is visible after the board has reconciled to authoritative state.

Timers and animation work are owned by the current state and cancelled on session change or component destruction. Stale asynchronous responses are ignored using a request/session identity check.

## Backend Contract

### Legal moves

`chessrules.Rules` gains:

```go
LegalMoves(fen string) ([]string, error)
```

It builds a game at the supplied FEN, enumerates valid moves from the existing chess library, encodes them with UCI notation, and sorts them for deterministic output and tests. Coverage includes ordinary moves, castling, en passant, and every legal promotion suffix.

`domain.PuzzleView` gains:

```go
LegalMoves []string `json:"legalMoves"`
```

The training service computes this field for every available current puzzle view, including after a correct non-final solver turn.

### Applied moves and final position

`domain.AppliedMove` is introduced as:

```go
type AppliedMove struct {
	UCI          string `json:"uci"`
	ResultingFEN string `json:"resultingFen"`
}
```

`domain.MoveResult` gains:

```go
AppliedMoves []AppliedMove `json:"appliedMoves,omitempty"`
FinalFEN     string        `json:"finalFen,omitempty"`
```

`AppliedMoves` contains the authoritative UCI moves applied by the request in display order, together with the authoritative FEN after each move:

- A correct play request contains the submitted move followed by an automatic reply when present.
- An incorrect play request contains no applied moves because authoritative state did not change.
- A reveal request contains the full remaining revealed line.
- An alternative accepted mate contains the submitted mating move.

The resulting FEN is required for promotion, en passant, and castling animation without duplicating chess move semantics in TypeScript. The move animation coordinator receives the submitted optimistic UCI separately. When it matches the first applied move, the coordinator does not replay the pointer move, but it still applies that step's `resultingFen` to reconcile captures, promotion role, and castling pieces. It then animates each remaining FEN transition and finally reconciles to `session.current.currentFen` or `finalFen`.

`FinalFEN` is present only when `PuzzleCompleted` is true. `completeCurrent` captures it from the completed item state before `CompleteItem` advances the durable session and `prepareAvailable` prepares the next view.

No storage migration is required. The fields are response projections over already-persisted item state.

## Frontend Boundaries

### `ChessBoard.svelte`

This component becomes a thin declarative wrapper. It owns the Chessground instance lifecycle and receives controlled properties for FEN, orientation, legal moves, input enabled state, last move, wrong move, hints, and reduced motion. It emits complete UCI moves only after promotion selection when required.

It must destroy the Chessground instance and listeners on unmount. Runtime assets are local; the board never loads code, CSS, pieces, or data from a CDN.

### Board adapter

A small typed adapter isolates imperative Chessground configuration from Svelte:

- Create, update, and destroy the Chessground API.
- Convert UCI legal moves to destination maps without losing promotion variants.
- Apply FEN and orientation updates.
- Lock and unlock move input.
- Apply last, incorrect, and hint marker modes.
- Animate authoritative applied-move FEN transitions and report completion or cancellation.
- Classify move versus capture sound by comparing authoritative before/after piece sets, so en passant is handled without implementing frontend legality.

No chess legality is implemented in the adapter.

### Puzzle state module

A pure state transition module owns the interaction phases and transition data, including the authoritative starting FEN, submitted UCI, pending response, final FEN, and pending session. `PuzzleScreen.svelte` renders this state and invokes API, board, and sound effects at transition boundaries.

### Sound service

The sound service owns move, capture, correct, and incorrect playback. Sounds are short, local, original project assets whose generator/source is committed and licensed with the application. There is no runtime download and no music.

Sound defaults on at restrained volume. A visible button exposes an accurate muted/unmuted label and `aria-pressed` state. The preference is stored in `localStorage`. Audio context initialization occurs on the first user gesture so later asynchronous feedback works in WebKit.

## Accessibility

Chessground pointer behavior is supplemented by one focusable board control:

- Arrow keys move a keyboard cursor according to board orientation.
- Enter or Space selects a legal source or submits a legal destination.
- Escape cancels selection or promotion.
- A concise instruction is associated with the board.
- A live region announces selected piece/square, incorrect move, completion, reveal, pause, and errors.
- The selected square and keyboard cursor remain visibly distinct.

Feedback never relies on sound or color alone. The side panel always contains text matching incorrect, success, reveal, and error states. `prefers-reduced-motion` disables nonessential movement while preserving final positions and feedback.

## Error and Recovery Behavior

- A backend error after an optimistic move restores the request's authoritative starting FEN, clears stale markers, reports the error, and reopens input when safe.
- A stale or backend-rejected illegal move also reconciles to the request's authoritative starting FEN rather than trusting optimistic frontend state.
- An animation exception or cancellation skips directly to the authoritative final FEN; it must not block progress or require restarting the session.
- Malformed legal-move data locks input and displays an actionable puzzle error instead of allowing arbitrary moves.
- Component destruction cancels prelude, reply, reveal, wrong-feedback, and sound work without updating the next screen.
- The backend continues validating every submitted UCI move even though the board filters destinations.

## GPL Compliance and Public Source

The combined Chess Trainer application is distributed under GPL-3.0-or-later.

Implementation must include:

- A root `LICENSE` containing the GNU GPL version 3 text and a README declaration that the project is available under GPL-3.0-or-later.
- `THIRD_PARTY_NOTICES.md` identifying `@lichess-org/chessground`, its exact version, its copyright notices, and GPL-3.0-or-later terms.
- The Chessground license text in the frontend's built legal assets.
- An in-app **About & Legal** view with project copyright, no-warranty notice, redistribution rights, license text access, Chessground attribution, exact build commit, and a link to `https://github.com/xang1234/chess/tree/<commit>`.
- Public build instructions, dependency lockfiles, source assets, and scripts needed to build and modify the distributed application.
- An exact Chessground dependency pin rather than a floating range.
- Legal files included in the Wails frontend assets so they are present in the packaged macOS application without network access.

Every distributed binary must map to matching public source. A release build records the commit identifier shown by About & Legal. Release verification confirms that the commit is public, the working tree used for the release is clean, dependency locks are committed, and built legal assets are present. Public binary releases use a matching repository tag and provide source access alongside the binary.

Recipients retain all GPL rights, including modification and redistribution. The project must not add an NDA, no-redistribution condition, or other incompatible restriction.

## Testing Strategy

Implementation follows red-green-refactor. The dependency is wrapped so most state behavior can be tested without relying on JSDOM geometry.

### Go tests

- Legal moves are sorted and correct for normal positions.
- Castling, en passant, and all promotion UCI suffixes are present.
- Invalid FEN returns a useful error.
- A correct non-final move refreshes legal moves for the resulting position.
- Correct completion returns applied moves and the completed final FEN while its session points to the next puzzle.
- Automatic replies appear in applied moves after the submitted move, and every step carries its resulting FEN.
- Reveal returns the complete remaining line and final FEN.
- Alternative mate returns the actually played move and final FEN.
- The last puzzle still returns final FEN while the pending session contains the summary.

### Frontend unit and component tests

- UCI grouping retains promotion variants while deduplicating visual destinations.
- Selecting an otherwise movable-color piece with no destination entry is immediately cleared.
- Interaction-state transitions reject stale responses and cancel owned work.
- Chessground is created, updated, disabled, oriented, and destroyed correctly through the adapter boundary.
- A fresh puzzle animates its prelude to `displayedFen`; a resumed non-empty path skips the prelude and starts at `currentFen`.
- Wrong moves snap back and reopen input only after reconciliation.
- Correct replies and reveal lines animate in order and reconcile to the authoritative FEN.
- Completion preserves the final board and does not apply the pending session before Next.
- The last-puzzle button opens results.
- Promotion choice emits the complete UCI suffix.
- Mute state persists and sound calls match move, capture, correct, and incorrect outcomes.
- Reduced motion bypasses animations.

### Browser tests

Playwright covers behavior that requires real pointer and layout support:

- Click-to-move and drag-to-move.
- Touch/pointer cancellation and snapback.
- White and black orientation.
- Selected-square, destination-dot, and capture-ring visibility.
- Promotion chooser.
- Keyboard navigation and live feedback.
- Wrong, correct, reveal, solved, Next, and final results flows.
- Responsive desktop and narrow layouts.
- Persistent mute and reduced motion.

The suite runs in Chromium and WebKit. WebKit is required because the macOS Wails runtime uses a WebKit-based view.

### Verification

Before completion, run:

- `go test ./...`
- `go test -race ./...`
- Frontend unit tests serially where required by existing test isolation.
- `npm run check`
- `npm run build`
- Playwright tests in Chromium and WebKit.
- `wails build`
- GPL release verification against the produced application assets.

## Acceptance Criteria

- Pieces are consistent SVG artwork rather than platform-dependent Unicode glyphs.
- A solver can select or drag a piece with mouse, trackpad, touch, or keyboard.
- Selection, legal empty destinations, and legal captures are unambiguous.
- Legal destinations never disclose which legal move is the puzzle solution.
- A wrong legal move visibly snaps back and leaves the learner on the same puzzle.
- Automatic replies and revealed lines animate in order unless reduced motion is enabled.
- The final solved position remains visible until explicit Next/See results input.
- Closing after non-final completion does not lose progress and resumes at the next persisted puzzle; closing after final completion returns home with the completed statistics retained.
- Sounds are restrained, local, muteable, and do not prevent use when unavailable.
- The puzzle screen works at desktop and narrow browser sizes without clipping.
- No engine implementation or engine analysis is introduced.
- The application and Chessground notices are GPL-compliant, available in-app, and bundled offline.
- Every distributed build identifies a public commit containing its complete corresponding source.
- Go, frontend, browser, production, Wails, and license-verification suites pass.
