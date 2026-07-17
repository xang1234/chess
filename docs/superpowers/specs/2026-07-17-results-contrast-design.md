# Results Contrast Design

## Problem

The completed-quiz results panel appears inside the dark puzzle shell but uses a light ivory background. It currently inherits the shell's near-white foreground color, producing measured contrast ratios of approximately 1.08:1 for the heading and body text and 1.14:1 for the summary cards. The results are therefore difficult to read and fall well below the WCAG AA target of 4.5:1 for normal text.

## Goals

- Keep the dark page and header with the existing light celebratory results panel.
- Make the completion heading, body copy, and summary-card text clearly readable.
- Provide at least a 4.5:1 contrast ratio for normal text in the results panel.
- Preserve all existing quiz-result, session, and navigation behavior.

## Non-goals

- Redesigning the active puzzle panel or chessboard.
- Changing the result values or summary data.
- Introducing a general theme or dark-mode system.

## Design

The contrast correction will remain local to the completion state in `PuzzleScreen.svelte` so it cannot alter other panels or the dark puzzle shell.

- Give `.completion` an explicit dark foreground color using `var(--ink-900)`. The heading and body copy will inherit this readable foreground instead of the shell's near-white color.
- Give the completion eyebrow an explicit `var(--forest-600)` color to retain the celebratory green accent on the ivory panel.
- Give `.summary-grid strong` an explicit `var(--ink-900)` foreground while retaining its current warm tan (`#f4e5bd`) background.
- Leave the existing primary action button, spacing, content, and completion flow unchanged.

## Testing

- Extend the Playwright final-results flow to calculate the effective foreground and background colors for the heading, body copy, and summary cards, then assert that each contrast ratio is at least 4.5:1.
- Keep the existing behavioral assertion that the summary appears only after the child selects **See results**.
- Run the focused Playwright test, the frontend unit tests, Svelte diagnostics, and the production frontend build.
- Perform a final browser check at the desktop viewport to confirm the dark page, ivory panel, and readable results presentation remain visually coherent.

## Acceptance Criteria

- The completion heading, body copy, and summary-card text each have a measured contrast ratio of at least 4.5:1 against their rendered backgrounds.
- The dark page and header, ivory completion panel, green accent, warm tan cards, and existing primary button remain visually intact.
- Result values, the explicit **See results** step, and **Back home** navigation behave exactly as before.
