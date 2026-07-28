# Opening Position Context and Graph Feedback Design

**Date:** 2026-07-28
**Status:** Approved direction, awaiting written-spec review

## Purpose

Make opening lessons easier to understand and remember by keeping the identity
of the live position visible and by responding intelligently when the learner
plays another authored move.

The primary learner is a notation-literate 10-year-old. The interface should
connect the board, familiar SAN, variation names, and a short plan without
adding another screen or another acknowledgement click.

This is the first release in a larger opening-practice program. It establishes
the context required before deterministic play-in runs, run-aware navigation,
or adaptive recall can be introduced safely.

## Goals

- Answer three questions beside the live board: where am I, how did I arrive,
  and what can happen next?
- Show both the alternatives at the preceding position and the continuations
  from the current position.
- Keep branch information consistent with the route shown in the SAN ribbon,
  including at transpositions.
- Present evaluations from the learner's perspective in chess notation and
  plain language.
- Surface the authored position plan wherever that position appears.
- Recognize legal unexpected moves that already exist in the course graph.
- Distinguish moves available at the selected depth, moves covered later, and
  moves not covered by the course.
- Preserve existing grading, progress, review scheduling, animation, retry,
  accessibility, and resume behavior.
- Require no course-pack re-authoring, database migration, or session-state
  migration.

## Non-goals

- Interactive branch chips or branch animation.
- Automatic refutation playback.
- Deterministic play-in runs or changes to session persistence.
- Randomized or adaptive recall runs.
- Inline recap cards, automatic checkpoint bypass, or other lesson-flow
  changes.
- A clickable or scrubbable SAN ribbon.
- Engine analysis or generated chess explanations.
- Treating a recognized mistake as correct for grading or review.

## Learner Experience

A compact position card appears directly beneath the move ribbon. It describes
the authoritative position used by the current lesson or review activity.

The card contains:

1. the position name;
2. its evaluation symbol and a plain-language interpretation;
3. the first authored `plan` note attached to the position, when one exists;
4. a **You arrived via** row; and
5. a **From here** row.

For example:

> **Giuoco Piano** · = Equal chances
>
> Prepare d4 while keeping the bishop active.
>
> **You arrived via:** Bc5 · Nf6 — **Bc5 Played**
>
> **From here:** c3 Giuoco Piano · b4 Evans Gambit

The branch rows are informational. They never move the board, update the
session, or compete with the lesson's primary action.

When the learner plays a legal unexpected move, the feedback responds at the
level of knowledge present in the course:

> **6.Bb5 — Ruy Lopez**
>
> A sound alternative that keeps pressure on the centre. This lesson is
> practising **6.Bc4**, aiming at f7.

The board remains ready for an immediate retry.

## Design

### 1. Structured position context

`OpeningActivityView` gains one nested position-context value rather than a
collection of unrelated optional fields:

```text
OpeningPositionContextView {
    positionId: string
    label: string
    evaluation: Evaluation
    planText: string?
    arrivedVia: OpeningContextMove[]
    nextMoves: OpeningContextMove[]
}

OpeningContextMove {
    moveId: string
    san: string
    role: TrainingRole
    variationName: string?
    evaluation: Evaluation
    selected: boolean
}
```

`positionContext` is present for every active activity view, including review
activities. Empty optional content is represented by empty strings or arrays
according to the repository's existing JSON conventions; the frontend decoder
still requires the object and its required fields.

The service resolves the live position from session state. This is important
for activities without an explicit `positionId` and avoids duplicating the
frontend's idea of which position is authoritative.

The plan is the first position note whose kind is `plan`, preserving authored
note order. If no such note exists, the plan row is omitted. Other note kinds
continue to appear through the existing teaching and reference-note surfaces.

### 2. Route-consistent arrival choices

The existing activity route value gains its aligned move IDs:

```text
activityRoute {
    MoveIDs: []string
    Frames: []AppliedMove
    SAN: []string
    VariationName: string
}
```

`MoveIDs`, `Frames`, and `SAN` are derived from one route and must have equal
lengths. The same route remains the source of the SAN ribbon.

For a non-root position, the final route move identifies:

- the preceding position;
- the move selected by the route; and
- the visible alternatives leaving that preceding position.

Those alternatives populate **You arrived via**, in compiled author order.
The route move has `selected: true`; all others are false. Only moves visible
at the session's selected depth are included.

At a transposition, this row intentionally describes the same canonical route
chosen for the ribbon rather than claiming to reconstruct historical learner
actions. This keeps the two displays internally consistent. At the course
root, `arrivedVia` is empty and the component renders **Starting position**.

### 3. Current-position continuations

`nextMoves` contains all depth-visible authored outgoing moves from the live
position, again in compiled author order. No move is marked selected in this
row.

Every chip displays SAN. A non-empty variation name appears as secondary text.
Training role and evaluation are carried in the contract for semantic styling,
accessible descriptions, and reuse by later branch previews, but the card does
not ask a 10-year-old to interpret internal role names.

When no authored continuation is visible at the selected depth, the row says
**End of this course line**.

### 4. Learner-relative evaluation

The backend continues to return the source `Evaluation` value. A single pure
frontend formatter maps it to a chess symbol and accessible phrase using the
course orientation:

| Source evaluation | White-course wording | Black-course wording |
| --- | --- | --- |
| equal | `= Equal chances` | `= Equal chances` |
| unclear | `∞ Unclear position` | `∞ Unclear position` |
| white slight | `⩲ Slightly better for you` | `⩲ Slightly better for White` |
| black slight | `⩱ Slightly better for Black` | `⩱ Slightly better for you` |
| white clear | `± Clearly better for you` | `± Clearly better for White` |
| black clear | `∓ Clearly better for Black` | `∓ Clearly better for you` |
| white winning | `+- Winning for you` | `+- Winning for White` |
| black winning | `-+ Winning for Black` | `-+ Winning for you` |
| none | omitted | omitted |

If the source includes a non-empty `sourceSymbol`, the visible chess symbol
uses it while the accessible phrase continues to come from the normalized
evaluation code. The same formatter is used in the position card and mistake
feedback.

### 5. Structured graph-aware feedback

`OpeningActivityResult` gains an optional structured detail:

```text
OpeningMoveFeedbackDetail {
    coverage: visible | deeper | unknown
    playedSan: string
    variationName: string?
    evaluation: Evaluation
    noteText: string?
    expectedSan: string
    planText: string?
    requiredDepth: Depth?
}
```

The existing top-level `feedback` classification remains authoritative for
controller behavior and grading. The structured value controls presentation;
the server's `message` remains a concise fallback for compatibility and error
surfaces.

After legality checking and the existing expected/accepted-alternative checks,
`PlayMove` searches all authored outgoing moves from the prompt position by
UCI:

1. A match visible at the session depth produces `coverage: visible`.
2. A match hidden only by minimum depth produces `coverage: deeper` and its
   required depth.
3. No match produces `coverage: unknown`.

For a matched move, the detail includes its SAN, variation, evaluation, and
the first useful authored note in this priority order: `explanation`, `plan`,
`warning`, then `evaluation`. If the move has no suitable note, `noteText` is
empty. The current position's plan is supplied independently as `planText`.

An accepted alternative retains `feedback: alternative` and today's grading
semantics. It uses the same detail structure with `coverage: visible`, so all
recognized moves receive a consistent explanation.

All other unexpected legal moves retain `feedback: off_course`, increment the
existing incorrect-move count exactly once, and leave the attempt active.
Recognizing a move never grants completion credit.

### 6. Feedback presentation

The frontend renders structured feedback as a short block with:

- a headline containing the played SAN and variation name when known;
- one explanation sentence derived from coverage, evaluation, and authored
  note data;
- one contrast sentence naming the expected move and current plan; and
- the existing retry affordance.

Visible authored moves are described as course-known ideas. Deeper moves are
labelled **Covered later** and name the required depth. Unknown moves are
described honestly as not covered from this position; the UI does not call
them bad or attempt to invent a refutation.

The feedback block uses the existing announced status region. Its heading and
body are text, not controls. A miss never leaves a preview or altered board
position behind.

### 7. Layout and accessibility

`OpeningPositionContext.svelte` is composed immediately below
`OpeningMoveRibbon.svelte` in the board column.

- The header keeps position name and evaluation together.
- The plan is clamped visually to a compact number of lines without truncating
  its accessible text.
- Branch rows use horizontally scrollable labelled lists on narrow screens.
- Chips are list items, not buttons.
- The selected arrival chip includes a visible **Played** marker and an
  accessible selected description; color is not the only indicator.
- Long position and variation names may truncate visually but remain available
  to assistive technology.
- The board-safe viewport calculation reserves the card's bounded desktop
  height. On the existing stacked layout, the card follows normal document
  flow.
- Reduced-motion behavior is unchanged because the card introduces no motion.

During comparison preview playback, the card continues to describe the
authoritative activity position. During successful activity animation, the
old context remains until the controller adopts the next activity after the
animation settles. Failed moves keep the current context unchanged.

## Component and Data Boundaries

- `internal/openings/service_activity.go`
  - adds move IDs to the aligned activity route;
  - derives position context, arrival alternatives, and next moves.
- `internal/openings/service_decision.go`
  - recognizes authored outgoing moves after expected-move checks;
  - constructs structured mistake detail without changing completion rules.
- `internal/openings/views.go`
  - defines the position-context, context-move, and feedback-detail contracts.
- `frontend/src/lib/contracts/openings.ts`
  - strictly decodes all new structured values and enums.
- `frontend/src/components/openings/opening-evaluation.ts`
  - owns the pure learner-relative evaluation formatter.
- `OpeningPositionContext.svelte`
  - renders the board-adjacent identity and branch card.
- `OpeningLessonScreen.svelte`
  - composes the card and routes structured feedback into the existing status
    surface.

No catalogue table, course-pack schema, or persisted `SessionState` field
changes in this release.

## Error Handling

- A missing live position or a route whose IDs, SAN, and frames are misaligned
  fails activity-view construction with position and activity context.
- Missing note references remain compiler or catalogue integrity failures; the
  view does not silently invent note text.
- Optional labels, evaluations, variation names, and suitable plan notes may be
  absent and degrade by omission.
- An authored graph match with malformed metadata falls back to the existing
  concise `message` only if the full activity result still passes contract
  validation; otherwise the response is rejected through the current error
  path.
- Existing request ownership, stale-response protection, retry, and board
  recovery behavior remain unchanged.

## Verification

Backend tests cover:

- root and non-root position context;
- route alignment and selected arrival moves;
- transposition route consistency;
- selected-depth filtering for both branch rows;
- position-plan selection and missing optional content;
- visible, deeper, unknown, expected, and accepted-alternative moves;
- unchanged attempt counters and completion semantics; and
- both white- and black-perspective course data.

Frontend tests cover:

- strict contract decoding and malformed response rejection;
- every evaluation code from both orientations;
- selected markers that do not rely on color;
- root, terminal, long-name, long-plan, and multi-branch rendering;
- structured feedback for visible, deeper, and unknown moves;
- immediate board retry after a miss;
- comparison previews leaving context unchanged; and
- desktop and narrow-layout behavior.

Release verification requires:

- all Go tests;
- Go race tests for the affected packages;
- frontend unit and component tests;
- frontend type checking;
- a production frontend build; and
- rebuilding the desktop application bundle.

## Follow-on Release Order

After this context and feedback release is implemented and verified:

1. specify bounded deterministic play-in runs with explicit canonical route
   identity, a distinct persisted run phase, a skip action, and first-visit
   demonstration behavior;
2. implement deterministic runs and run-aware progress;
3. add historical ribbon preview with an explicit **Return to live** state;
4. evaluate adaptive recall runs using persisted encounter and outcome data;
5. consider inline recaps as a separate flow experiment; and
6. keep automatic checkpoint bypass deferred unless usage evidence shows that
   checkpoints are the remaining material interruption.

## Acceptance Criteria

- Every active lesson and review activity exposes a position-context object.
- The card accurately shows the current position label, learner-relative
  evaluation, authored plan, route-consistent arrival choices, and visible
  outgoing choices.
- Root, terminal, transposed, and partially authored positions render without
  misleading placeholders.
- Legal unexpected moves receive visible, deeper, or unknown feedback based on
  the authored graph.
- Grading, review scheduling, activity progress, and retry behavior are
  unchanged.
- The feature adds no completion click and no interactive control unrelated to
  the current lesson action.
- Accessibility checks, automated tests, production build, race tests, and the
  desktop application rebuild pass.
