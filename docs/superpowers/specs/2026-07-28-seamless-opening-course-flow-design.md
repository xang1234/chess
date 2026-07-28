# Seamless Opening Course Flow Design

**Date:** 2026-07-28

**Status:** Approved design

## Problem

The opening course is authored as a teaching tree, but the learner should not
experience every teaching node as a separate lesson. Today, finishing the last
activity in a node creates two discontinuities:

1. the lesson screen becomes a roadmap checkpoint and requires another choice;
2. leaving at that boundary discards the transient checkpoint, so later resume
   can reconstruct only a recommendation rather than the continuous journey.

The current sequence is effectively:

`finish activity -> acknowledge result -> checkpoint -> start another lesson`

The full teaching tree compounds the interruption. It replaces the lesson,
shows generic `Study` actions, and does not make the learner's current and next
positions sufficiently obvious. This is especially disruptive for a
ten-year-old who should feel that one opening course is one ongoing lesson.

## Goal

Make movement across teaching-tree nodes indistinguishable from movement
between activities in the same node. Preserve deliberate access to the course
map, clearly label its nodes and branch moves, and make the exact next activity
durably resumable after navigation or application restart.

## Experience Principles

- Teaching nodes remain useful authoring, branching, and progress units, but
  they are not learner-facing screen boundaries.
- The recommended route advances by default. Exploration is available without
  competing with the primary learning action.
- Node completion is persisted silently. There is no node-completion message,
  checkpoint, celebration, or additional Continue action.
- The learner controls the pace after interactive feedback through the existing
  `Next idea` action. That action behaves identically whether the next activity
  belongs to the current node or another node.
- Course progress and navigation state never depend on transient frontend
  presentation state.
- Course-map meaning is communicated with concise text and symbols, never color
  alone.

## Scope

This design changes:

- study-mode transitions between opening teaching nodes;
- durable journey continuation and resume semantics;
- the compact course map shown in the lesson sidebar;
- intentional navigation to another visible course node;
- course completion semantics and related tests.

This design does not change:

- review-session completion summaries;
- opening-course authoring formats or teaching-tree validation;
- depth filtering rules;
- move validation, feedback, hint, reveal, or animation behavior;
- variation exploration;
- puzzle-training flows; or
- the learner's ability to revisit completed opening material.

## Learner Flow

### Continuous study

The learner enters an opening course through `Continue opening lesson`, an
explicit course-map selection, or the initial recommended node. The lesson
screen remains mounted while the learner advances through activities and
nodes.

After an activity is completed:

1. interactive feedback and its existing acknowledgement remain visible when
   appropriate;
2. `Next idea` adopts the next active activity returned by the backend;
3. title, content, board position, move ribbon, progress, path, and sidebar map
   update in place; and
4. the learner continues without seeing a checkpoint, hub, tree replacement,
   node-completion message, or new lesson screen.

Passive zero-frame activities may continue using the existing automatic
advance behavior. Crossing a node boundary must not introduce a new pause.

Only completion of the entire visible route at the selected course depth shows
a course completion summary. A node boundary alone never shows a summary.

### Leaving and returning

Home remains the default application landing screen. When an opening journey
is active, its primary opening action is `Continue opening lesson`. That action
opens the exact durable next activity directly, without first opening the
opening hub or course map.

Leaving immediately after crossing a node boundary and restarting the
application must resume the already-selected next activity. Resume must not
replay the completed node, reopen a checkpoint, or ask the learner to select a
branch that the course already recommended.

### Intentional navigation

The course map is available as a secondary Home action and as a quiet lesson
sidebar control. Merely opening or closing it does not alter the active session
or journey cursor.

The map clearly marks `Your next idea` and provides `Return to your lesson`.
The learner may select any visible completed or available node and choose
`Start here`. That explicit action moves the active journey cursor to the
selected node and replaces the single resumable lesson position. Completed
activity progress remains saved. An unfinished activity at the previous cursor
is not marked complete.

If the learner opens the map and makes no selection, returning restores the
same activity and board state.

## Sidebar Course Map

The lesson sidebar contains a compact local course map. It is a concise
navigation aid, not a miniature rendering of the entire teaching tree.

The local projection includes:

- the learned path leading to the current node;
- the current node;
- the recommended next node;
- immediate visible alternatives at the relevant fork; and
- a `View full map` control.

Each entry exposes three distinct pieces of information:

1. a concise branch-move label such as `3...Nf6` or `4.c3` when the incoming
   edge has one;
2. a plain-language node name such as `Two Knights Defence`; and
3. one explicit progress label.

The supported progress treatments are:

- `✓ Learned`
- `▶ Current`
- `→ Next`
- `○ Available`

The current and next entries receive stronger typographic emphasis. Parent and
child relationships use indentation and simple connector lines. Move notation
appears before the node name and must remain visually distinct from the node
title. State cannot be expressed through color alone.

An illustrative local map is:

```text
COURSE MAP

✓ Italian setup
  3...Bc5
✓ Giuoco Piano
  4.c3
▶ Prepare the centre        Current
  4...Nf6
→ Play d4                   Next

Other replies
  3...Nf6  Two Knights      Available
  3...Be7  Hungarian        Available
```

The highlight updates in place as the course crosses a node boundary. The
sidebar does not scroll the learner away from lesson content or replace the
board.

## Architecture

### Continuous course runner

The backend owns the cross-node transition. The frontend must not simulate
continuity by receiving a completed checkpoint and issuing a second
`StartLesson` request.

When the last required activity in a study node completes, one store
transaction:

1. persists the completed activity and updates node progress;
2. completes the current node-level session;
3. computes the recommended next visible unfinished node from the updated
   teaching-tree projection;
4. creates the next active node-level session at its first incomplete required
   activity;
5. updates the course journey cursor and path to that active session; and
6. commits the old completion, new active session, and journey together.

The public result represents a continuous course journey even if the stored
node-level session ID changes. It returns the next active `OpeningSessionView`
in the same `OpeningActivityResult`. The controller already adopts returned
sessions and can therefore update the existing lesson screen without a route
change or a second network call.

If no visible unfinished node remains, the transaction completes the current
session without creating another active session. The result then contains the
course completion view.

### Recommendation

The next node is computed after applying the completed activity and node
progress. It follows the existing selected-depth visibility and deterministic
teaching-tree recommendation rules.

At a branch, the service automatically selects the recommended child or next
unfinished route. Alternative visible nodes remain available through the
course map. The normal study flow never pauses to request a branch choice.

### Journey cursor

The durable journey continues to identify:

- course and active generation;
- selected depth;
- current lesson and activity;
- current teaching path;
- the active node-level session; and
- the last computed recommendation as a non-authoritative hint.

Resume validates the active session and recomputes recommendations against the
current course generation. The journey cursor, active session, and lesson
progress must never describe different node boundaries after a committed
operation.

### Local map projection

An active study session response includes a small, typed local-map projection
derived from the same teaching-tree and progress snapshot used to select the
current activity. It contains only the entries and relationships required by
the sidebar, including:

- lesson ID and parent lesson ID;
- incoming edge move label;
- concise lesson title;
- progress state;
- current and recommended flags; and
- whether the node is selectable at the current depth.

This projection avoids rendering stale Home-screen data during a long course
run. The full course map remains available through the opening-home course
summary and is refreshed after an intentional navigation change.

## Frontend State and Components

The opening controller removes the study-mode `checkpoint` transition. A
successful activity can resolve to:

- another active activity in the same node;
- an active activity in a different node;
- a final course completion summary; or
- the existing review completion summary.

The controller treats both active results identically. Board reconciliation,
sound, animation, feedback acknowledgement, and `Next idea` remain governed by
the current activity result rather than the node boundary.

`OpeningLessonScreen` keeps its board and panel layout mounted. It adds a
focused sidebar course-map component that consumes the session's local
projection. The existing full `OpeningTeachingTree` remains the intentional
navigation surface, but its actions and labels become contextual:

- active cursor: `Return to your lesson`;
- recommended node: `Start next idea here`;
- other visible node: `Start here`; and
- completed node: `Study again`.

Opening the full map from the lesson preserves the active session in the shell.
Only successful `Start here` adoption replaces it.

## Failure and Recovery

- Cross-node persistence is atomic. If creating the next session or updating
  the journey fails, the activity completion and node transition roll back.
- A recoverable failure leaves the learner on the current board and exposes the
  existing `Retry` action. It must not show a false completed-node state.
- If `Start here` fails, the existing active lesson and journey remain the
  resumable state.
- If a course update removes the hinted next node, the service recomputes a
  valid visible recommendation from retained progress.
- If no valid unfinished node remains, the service returns course completion
  rather than an empty active state.
- Existing restart-required behavior remains available when a changed course
  cannot safely rebase the current activity.

## Accessibility and Child-Facing Copy

- Every map state has text and a symbol; color is supplementary.
- Incoming move notation and lesson titles are announced separately in an
  entry's accessible name.
- `Current`, `Next`, and `Available` use short, consistent language.
- The map preserves keyboard tree navigation and visible focus.
- The primary lesson action remains visually dominant over map navigation.
- No copy exposes implementation terms such as node, cursor, projection, or
  session to the learner.

## Testing

### Backend

- Completing an ordinary activity returns the next activity in the same node.
- Completing a node returns the first incomplete activity in the recommended
  next node as an active session and returns no roadmap checkpoint.
- The old node session, new active session, progress, and journey cursor commit
  atomically.
- Injected failures at each boundary write roll back the entire transition.
- Restart after a committed node boundary resumes the exact next activity and
  board state.
- A real branch follows the deterministic recommendation while keeping visible
  alternatives selectable.
- A course-generation change recomputes an unavailable recommendation safely.
- Completing the last visible unfinished node returns final course completion.
- Review-mode completion behavior remains unchanged.

### Frontend

- Crossing a node boundary never renders the roadmap checkpoint, opening hub,
  or full map.
- The same lesson-screen layout remains mounted while session ID, lesson ID,
  activity, board, path, and local map update.
- Interactive feedback still waits for `Next idea`; that action enters the next
  activity directly.
- Passive zero-frame completion does not add a boundary pause.
- The sidebar displays move labels, concise lesson names, and Learned, Current,
  Next, and Available states without relying on color.
- Opening and closing the map preserves the active session and board.
- `Return to your lesson` restores the active activity.
- `Start here` adopts the selected activity only after the backend succeeds.
- Home continuation opens the exact active opening activity directly.

### End to end

- Complete several activities spanning at least three teaching nodes without
  visiting the hub or full tree.
- Exit immediately after a node boundary, restart the application, and resume
  the exact next activity.
- Open the map, return without selecting, and preserve position.
- Select an alternative available node and continue from it with prior
  completed progress intact.
- Finish the selected-depth route and see one final course summary.

## Acceptance Criteria

- A ten-year-old can advance across a teaching-node boundary without noticing
  a navigation or session boundary.
- No checkpoint, celebration, node-completion copy, full tree, or additional
  Continue action appears at a node boundary.
- The sidebar clearly identifies branch moves, lesson names, learned nodes, the
  current node, the next node, and nearby available alternatives.
- `Continue opening lesson` resumes the exact durable next activity directly.
- Deliberate map navigation is available but secondary and never destroys the
  active position merely by opening the map.
- A crash or application restart cannot strand the journey between completed
  and next nodes.
- The complete automated test suite and signed macOS application build remain
  green.
