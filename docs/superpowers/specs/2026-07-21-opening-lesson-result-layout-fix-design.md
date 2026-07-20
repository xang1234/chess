# Opening Lesson Result and Layout Fix Design

**Status:** Approved direction (option A), awaiting written-spec review
**Date:** 2026-07-21

## Problem

The redesigned opening lesson has two regressions.

First, completing a passive teaching activity can show:

> opening activity result.appliedMoves must be an array

The backend currently marks every completed lesson activity with
`feedback: "expected"`. That feedback is correct for a learner decision, but
not for concept, comparison, recap, reference, or demonstration activities.
For passive activities the backend also returns an empty `AppliedMoves` slice
and empty `FinalFEN`; `omitempty` removes both fields from JSON. The frontend
correctly treats `feedback: "expected"` as a move completion and requires
authoritative move frames, so the two sides contradict each other.

Second, the lesson card has a minimum height but no bounded height. A longer
course path, teaching note, comparison, or reference section therefore grows
the entire two-column grid. The action buttons move below the viewport and the
board is no longer kept in the learner's working view.

## Goals

- Give each activity result one unambiguous semantic shape.
- Preserve strict validation for learner moves and demonstrations.
- Keep the board and the lesson card together in the desktop viewport.
- Keep Continue, Hint, Show course move, and Pause controls visible.
- Allow arbitrarily detailed private course text without growing the card.
- Start each new activity at the top of its lesson information.
- Add regression coverage at the service, contract, component, and browser
  boundaries.

## Non-goals

- Redesigning the variation explorer, teaching tree, or course authoring
  schema.
- Shortening or rewriting private course content.
- Changing opening progress, review scheduling, or lesson recommendation.
- Making the chess board itself scroll internally.
- Guaranteeing simultaneous board-and-card visibility on narrow stacked
  layouts where the physical viewport cannot contain both.

## Chosen Design

### 1. Activity-result contract

Result fields reflect what actually occurred:

| Activity outcome | `feedback` | `appliedMoves` | `finalFen` |
| --- | --- | --- | --- |
| Concept, comparison, recap, or reference completed | omitted | omitted | omitted |
| Demonstration completed | omitted | non-empty authoritative frames | required |
| Expected learner decision completed | `expected` | non-empty authoritative frames | required |
| Playable alternative or off-course move | `alternative` or `off_course` | omitted | omitted |

`completeLessonActivity` will attach `FeedbackExpected` only when it receives a
real learner attempt. Passive activities and demonstrations pass no attempt,
so they will not pretend to be learner move feedback. Demonstrations continue
to return their authoritative animation frames and final position.

The frontend decoder remains strict. It will continue rejecting an expected
decision without frames, a demonstration with frames but no final FEN, and any
alternative/off-course response that mutates lesson progress. It already
accepts a completed passive result with no feedback or move fields, so the fix
belongs at the backend semantic source rather than in a permissive frontend
fallback.

The opening controller continues to animate frames when present. For a passive
completion it uses the request position as the unchanged final position and
shows the existing “Idea complete” transition before the learner explicitly
continues.

### 2. Bounded lesson card with pinned actions

The desktop lesson card becomes a bounded two-zone container:

1. `opening-lesson-scroll` contains the course path, heading, progress,
   activity content, and feedback. It has `min-height: 0`, vertical overflow,
   a stable scrollbar gutter, and an accessible region label.
2. `opening-actions` remains outside that scroll region as a non-shrinking
   footer. Its controls therefore stay visible regardless of lesson depth or
   note length.

The card hides outer overflow. On the desktop two-column layout its height is
`min(760px, calc(100dvh - 120px))`, with the equivalent `vh` declaration first
as a fallback. The existing board sizing remains authoritative; the content
card is prevented from making the grid taller than that board-safe working
area.

When `current.activityId` changes, Svelte recreates the informational scroll
region so its scroll position returns to the top. Opening and closing deeper
analysis within the same activity does not reset the learner's position.

At the existing stacked breakpoint, the board remains above the card and the
document may scroll between them. The card uses
`min(620px, calc(100dvh - 32px))`, again with a `vh` fallback, and retains its
internal information scroll plus pinned action footer. This keeps controls
usable without pretending that a narrow viewport can show a full board and
full card simultaneously.

### 3. Accessibility and interaction

- The scrollable information region is keyboard-focusable and labelled
  “Opening lesson details”.
- The visible focus treatment follows the app's existing focus styles.
- Page scrolling is not trapped; only wheel, trackpad, touch, or keyboard
  input over the information region scrolls its content.
- The action footer keeps normal tab order after the informational content.
- Reduced-motion behavior and board animation behavior are unchanged.

## Error Handling

- Contract violations remain visible as recoverable opening-operation errors;
  no malformed response is silently coerced.
- An expected learner result without authoritative frames remains invalid.
- A passive result carrying only a final FEN remains invalid.
- If the internal region does not overflow, it behaves like ordinary static
  content and displays no unnecessary scrollbar movement.

## Test Strategy

### Backend

- A passive activity completion has empty feedback and no move-frame payload.
- A demonstration has frames and a final FEN but no learner feedback.
- A decision still has expected feedback, frames, and final FEN.
- JSON response tests prove omitted passive fields and retained move fields.

### Frontend contract and controller

- The decoder accepts the exact passive backend response.
- It continues rejecting expected feedback without an `appliedMoves` array.
- The controller completes passive activities without animation frames and
  advances only after explicit Continue.

### Component and browser

- The action footer is structurally outside the labelled scroll region.
- A synthetic activity with long path and note content produces internal
  overflow at a desktop viewport.
- Scrolling changes the lesson region's `scrollTop` while the action footer's
  viewport position remains stable.
- The board remains fully inside the desktop viewport and does not move when
  lesson information is scrolled.
- Advancing to the next activity recreates the region at scroll position zero.

## Acceptance Criteria

- The reported `appliedMoves must be an array` error no longer occurs when a
  learner completes or continues a passive opening activity.
- Expected decisions and demonstrations still animate authoritative moves.
- Long or deep lesson content scrolls inside the card.
- The relevant action buttons remain visible throughout that scrolling.
- At desktop lesson dimensions, the complete board remains visible.
- Existing opening progress and course-tree behavior are unchanged.
- The full Go, frontend, static, opening E2E, production-build, and macOS app
  rebuild gates pass with a clean source tree.
