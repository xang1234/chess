# Opening Lesson Board-as-Story Design

**Date:** 2026-07-27
**Status:** Approved direction, awaiting written-spec review

## Purpose

Make opening lessons feel like one continuous chess story instead of a series
of cards interrupted by repeated Continue buttons. The primary learner is a
notation-literate 10-year-old, so the board should carry the action while SAN
provides a persistent map of the line.

This is a generic opening-lesson improvement. It applies to every imported
course and requires no private course re-authoring.

## Evidence

The current interaction has two stages for every required activity:

1. complete the activity by clicking Continue or making a move; and
2. acknowledge an `activity-complete` screen by clicking Continue again.

Across the eight current private course packs, 338 of 514 required activities
are concepts, comparisons, or recaps. Those activity kinds produce no applied
board moves, so their second click acknowledges an unchanged board. Removing
that acknowledgement is expected to eliminate about 37% of current Continue
presses across the library and about 32% in Italian White.

The other continuity problems are visible in the current implementation:

- `acknowledgeActivity` and `adoptSession` always increment
  `boardGeneration`, remounting Chessground even when the FEN is unchanged;
- `movesToHere` contains UCI/FEN animation frames but no SAN suitable for a
  learner-facing move list;
- concepts and recaps often have no variation name because variation
  resolution currently depends on a prompt or activity `moveIds`;
- comparison lines render SAN as inert text while the board remains unused.

## Goals

- Remove acknowledgements that do not reveal or preserve a board change.
- Keep one Chessground instance alive across activities with an identical FEN.
- Show the current opening line in familiar SAN directly beneath the board.
- Keep the current variation name visible across every activity kind.
- Let the learner play comparison lines on the main board.
- Give the remaining primary actions purpose-specific labels.
- Preserve checkpoints, move-result pauses, progress, resume behavior,
  accessibility, reduced motion, and strict response validation.
- Keep the first implementation small enough to ship and evaluate safely.

## Non-goals

- XP, streaks, confidence ratings, rewards, or additional completion screens.
- A full accumulating lesson transcript.
- Clickable arbitrary-ply position browsing from the SAN ribbon.
- Inferring and animating transitions between different activity positions.
- Changing course-pack schema or requiring private pack content edits.
- Changing review scheduling, lesson recommendation, or completion semantics.
- Replacing checkpoints or the final lesson summary.
- Adding global Space or Enter shortcuts. Native focused buttons already
  support keyboard activation.

## Learner Experience

The lesson follows a clear rhythm:

1. **Learn:** read one short concept and choose **Got it**.
2. **Watch:** choose **Play the line** and watch the board move.
3. **Compare:** tap either named plan to play it on the board, then choose
   **Done comparing**.
4. **Try:** make the requested move directly on the board.
5. **Continue:** after a move or demonstration has changed the position,
   choose **Next idea** when finished looking.

Choosing **Got it** or **Done comparing** immediately displays the next idea
when the server reports no applied moves. There is no `Idea complete` flash and
no second Continue button. A checkpoint, lesson completion, demonstration, or
successful learner move still provides an explicit stopping point.

The SAN ribbon remains under the board throughout the lesson. It shows the
variation badge followed by the route to the current position, for example:

> Italian Game · 1.e4 e5 2.Nf3 Nc6 3.Bc4 Bc5 4.c3

The final ply is visually emphasized so the learner can connect the current
position to the notation without searching the line.

## Design

### 1. Skip empty completion screens

The controller fast-forwards a successful activity only when all of these are
true:

- `activityCompleted` is `true`;
- `appliedMoves` is absent or empty;
- `checkpoint` is absent; and
- the returned session is active and contains a next activity.

The rule is based on the result rather than the activity kind. It therefore
remains correct if a future passive activity returns moves or a new passive
kind is added.

The fast path runs after ownership and response validation but before entering
the `animating` or `activity-complete` phase. It will:

1. cancel the completed request token;
2. publish `initialiseOpening(result.session)` directly;
3. clear completion feedback, frames, and last-move highlighting;
4. retain any response or board warning;
5. emit the existing persisted-session event;
6. emit the visible-session change event; and
7. announce `Next idea: <title>.` without rendering a transient message.

The persisted event is emitted before the visible change event. This preserves
the parent shell's saved-session bookkeeping while allowing the visible
session to become authoritative in the same turn.

If the response has frames, a checkpoint, or no active next activity, the
existing animation and `activity-complete` flow remains in force. Malformed
responses remain errors; this design does not relax the decoder.

### 2. Preserve the board when the position is unchanged

Board remount decisions compare the currently displayed FEN with the target
activity FEN:

- identical full FEN: keep the current `boardGeneration`;
- different full FEN: increment `boardGeneration` and use the existing hard
  position replacement;
- explicit board-error recovery: retain the runtime's existing forced-remount
  behavior.

The same helper is used by `acknowledgeActivity`, the new empty-result fast
path, and `adoptSession`. Legal moves, annotations, hints, orientation, and
input state continue updating through the existing reactive ChessBoard
configuration even when the component is preserved.

This change deliberately does not animate different-FEN activity transitions.
The current `movesToHere` route is a deterministic breadth-first route from the
course root, but with transpositions it is not guaranteed to be the route the
learner just played. A later transition-animation design must use explicit
authoritative frames rather than client-side prefix inference.

The duplicate `runtime.cancelRequest()` currently present in
`acknowledgeActivity` is removed while this flow is updated.

### 3. Add a persistent SAN journey ribbon

`OpeningActivityView` gains:

```text
movesToHereSan: string[]
```

The opening service will refactor its root-to-position route calculation into
one internal result containing the chosen move IDs, applied frames, SAN moves,
and nearest non-empty variation name. `movesToHere` and `movesToHereSan` must
come from the same move-ID route and therefore have equal lengths.

Variation-name resolution uses the nearest non-empty name on that route. If
the route has no name, it falls back to the existing prompt-primary or activity
move resolution. This keeps the branch label present for concepts, recaps, and
comparisons without changing private content.

A new `OpeningMoveRibbon.svelte` component appears immediately beneath the
board. It:

- formats SAN into numbered move pairs;
- displays the variation name as a compact badge;
- emphasizes the last ply;
- uses a single horizontally scrollable line instead of wrapping into a tall
  block;
- scrolls the current ply into view when the activity changes;
- exposes the notation as one labelled list to assistive technology; and
- displays `Starting position` when the route is empty.

The ribbon is display-only in this release. Individual plies are not buttons.

On desktop, the board and ribbon together must fit within the existing
board-safe viewport height. The opening board width calculation reserves the
ribbon height and gap rather than allowing the ribbon to push the board below
the viewport. On the existing stacked layout, the ribbon follows the board and
normal document scrolling remains available.

### 4. Make comparison lines playable

`OpeningActivityLine` changes from:

```text
label: string
moves: string[]
```

to:

```text
label: string
moves: string[]
frames: AppliedMove[]
```

The service builds each line's frames by applying its validated move IDs from
the comparison activity position. The SAN and frame arrays must be non-empty
and have equal lengths. The frontend decoder enforces that relationship.

Each comparison article becomes a real button whose accessible name contains
the label and SAN sequence. Selecting a line:

1. cancels any active comparison preview;
2. restores the activity's authoritative `currentFen`;
3. animates that line with the existing move-animation runtime;
4. highlights the selected line while it plays and after it completes; and
5. reveals a **Reset position** action.

Selecting a different line resets the board and immediately plays the new
line. **Reset position** returns to `currentFen`, clears the selected line, and
does not contact the backend.

Comparison playback is preview-only. It does not update the session, progress,
`playedMoveIds`, or review state. Choosing **Done comparing** cancels any
preview, restores `currentFen`, and then performs the existing activity advance
request. The zero-frame fast path displays the next idea after that save.

Reduced-motion mode moves directly to each preview's final FEN while retaining
selection and announcement behavior. Playback cancellation and stale-result
protection use the existing request-owner token pattern.

### 5. Use purposeful action labels

The primary action text is derived from state and activity kind:

| State | Activity | Label |
| --- | --- | --- |
| passive | concept or recap | Got it |
| passive | comparison | Done comparing |
| passive | demonstration | Play the line |
| activity complete | any board-changing result | Next idea |
| ready | decision | no primary completion button; the board is the action |

Hint, Show course move, Retry, Reset position, and Pause lesson retain their
existing meanings. Checkpoint and terminal labels remain unchanged.

## Component and Data Boundaries

- `internal/openings/service_activity.go`
  - produces aligned route frames, SAN, and variation metadata;
  - produces authoritative frames for each comparison line.
- `internal/openings/views.go`
  - adds `movesToHereSan` and comparison `frames` to the public view.
- `frontend/src/lib/contracts/openings.ts`
  - decodes the new arrays strictly and rejects mismatched comparison lines.
- `frontend/src/components/openings/opening-state.ts`
  - retains pure transition validation; a small helper may represent the
    successful empty-result transition so it is unit-testable.
- `frontend/src/components/openings/opening-controller.ts`
  - selects the fast path, controls board remounts, and owns comparison
    preview cancellation and restoration.
- `OpeningMoveRibbon.svelte`
  - owns SAN numbering, current-ply presentation, and ribbon accessibility.
- `OpeningActivityContent.svelte`
  - renders comparison controls and dispatches preview/reset intent.
- `OpeningLessonScreen.svelte`
  - composes the ribbon beneath the board and maps state to primary labels.

No course-pack schema, catalogue storage, or migration changes are required.
All new view fields are derived at read time from already compiled moves.

## Error Handling

- Failure to derive a route or apply a comparison line fails activity-view
  construction with context identifying the activity or comparison label.
- Contract mismatches such as unequal SAN/frame lengths are rejected rather
  than hidden.
- A comparison animation error restores the authoritative activity FEN, keeps
  the activity incomplete, and presents the warning through the existing
  recoverable notice area.
- Starting a backend action, pausing, adopting an external session, or leaving
  the screen cancels any comparison preview first.
- A failed passive save remains on the same activity and exposes Retry. Retrying
  a successful zero-frame save uses the same fast path.
- Checkpoints and completed sessions can never enter the empty-result fast path.

## Accessibility and Child-Focused Interaction

- All comparison lines and actions remain native buttons with visible focus.
- No global keyboard handler is added.
- The ribbon has a concise accessible label and does not announce its entire
  contents on every animation frame.
- Comparison playback announces start, completion, cancellation, and reset.
- The next activity heading receives normal visual prominence; the live region
  announces only the new idea, not an intermediate completion.
- Touch targets retain the existing minimum 46px height.
- Motion follows the saved reduced-motion preference.
- The design adds no required activity or click. Comparison playback is
  optional and completing a comparison never requires playing both lines.

## Testing Strategy

### Go service and response tests

- Concepts, comparisons, and recaps still complete without applied moves.
- Demonstrations and decisions still return authoritative non-empty frames.
- Every activity view returns SAN aligned with `movesToHere`.
- Root-position activities return empty route arrays and a safe variation
  fallback.
- Concepts and recaps inherit the nearest route variation name.
- Comparison lines return correct SAN and frames from the activity FEN.
- Invalid comparison application reports a contextual error.
- JSON response tests include the new fields without changing course-pack
  persistence.

### Frontend contract tests

- Valid route SAN and comparison frames decode.
- A comparison with empty frames or differing SAN/frame lengths is rejected.
- Existing strict activity-result validation remains unchanged.

### State and controller tests

- An active zero-frame result skips `animating` and `activity-complete`.
- The fast path emits persisted and visible-session events exactly once and in
  that order.
- A zero-frame checkpoint or terminal result still stops explicitly.
- Move and demonstration results still animate and wait for **Next idea**.
- An unchanged FEN preserves `boardGeneration`; a changed FEN increments it.
- External same-FEN session adoption updates activity state without remounting.
- Comparison previews reset before playback, cancel cleanly, and never change
  persisted progress.
- Completing, pausing, or leaving during a preview restores authoritative
  state and ignores stale animation callbacks.

### Component tests

- The SAN ribbon renders numbered SAN, variation badge, and last-ply emphasis.
- The ribbon handles starting positions and horizontal overflow.
- Concept, recap, comparison, demonstration, and completion states use the
  approved action labels.
- Comparison lines are keyboard-operable buttons and Reset position appears
  only after a preview starts.
- The active feedback live region does not re-announce the full move ribbon.

### Browser acceptance

- Complete a three-idea concept/comparison/decision lesson with no unchanged
  `Idea complete` screen.
- Confirm the board does not flash between same-position ideas.
- Play both sides of a comparison, reset, and complete the activity.
- Confirm the SAN ribbon matches the board position throughout the lesson.
- Complete a demonstration and a decision and verify both retain the explicit
  board-inspection pause.
- Reach a lesson checkpoint and verify it is never skipped.
- Verify desktop board plus ribbon remain in the viewport and the stacked
  layout remains usable at narrow widths.
- Repeat the flow with keyboard navigation and reduced motion.

## Delivery Sequence

The implementation should land in three reviewable slices:

1. Empty-result fast path, purposeful labels, conditional board remount, and
   removal of the duplicate cancellation call.
2. Route SAN/variation contract and persistent move ribbon.
3. Authoritative comparison frames and playable comparison controls.

Each slice must pass its focused Go/frontend tests before the next slice and
the full repository gates before completion.

## Success Criteria

- No zero-move, non-terminal activity renders an intermediate completion
  screen.
- Existing course progress and resume behavior remain correct.
- The Italian course requires about one-third fewer Continue presses without
  deleting teaching material.
- Same-FEN activity changes do not recreate Chessground.
- Every active lesson position has a readable SAN route or an explicit
  `Starting position` label.
- Every authored comparison line can be watched on the main board.
- Checkpoints, demonstrations, and successful decisions retain meaningful
  pauses.
- The complete flow is usable by pointer, touch, keyboard, screen reader, and
  with reduced motion.

## Deferred Follow-up

After observing the streamlined flow, two ideas may be reconsidered:

- **Cross-activity animation:** add an explicit server-provided transition
  frame contract. Do not infer transitions from BFS route prefixes.
- **Compact lesson trail:** expose a collapsed `Learned so far` summary or the
  immediately previous idea if learners demonstrably need to look back. Do not
  accumulate full historical cards in the narrow lesson panel by default.
