# Results Contrast Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the completed-quiz heading, body copy, and summary cards readable at WCAG AA contrast while preserving the current light celebratory panel and all quiz behavior.

**Architecture:** Add a Playwright contrast regression to the existing completed-results scenario, using a small test-only color calculation helper. Correct the inherited foreground locally in `PuzzleScreen.svelte` so the dark puzzle shell and unrelated panels remain unchanged.

**Tech Stack:** Svelte 3, TypeScript, Playwright, CSS custom properties

## Global Constraints

- Keep the dark page and header with the existing light celebratory results panel.
- Provide at least a 4.5:1 contrast ratio for normal text in the results panel.
- Preserve all existing quiz-result, session, and navigation behavior.
- Do not redesign the active puzzle panel or chessboard, change summary data, or introduce a general theme system.
- Keep all production styling changes local to the completion state in `PuzzleScreen.svelte`.

---

### Task 1: Enforce Accessible Results Contrast

**Files:**
- Modify: `frontend/tests/board-interactions.spec.ts`
- Modify: `frontend/src/components/puzzle/PuzzleScreen.svelte`

**Interfaces:**
- Consumes: the existing `.completion`, `.eyebrow`, and `.summary-grid strong` elements and the global `--ink-900` and `--forest-600` CSS properties.
- Produces: a test-only `contrastRatio(foreground: Locator, background: Locator): Promise<number>` helper and completion styles that meet a minimum 4.5:1 contrast ratio.

- [ ] **Step 1: Add a contrast calculation helper to the Playwright test file**

Change the Playwright type import and add the following helpers after the FEN constants in `frontend/tests/board-interactions.spec.ts`:

```ts
import { expect, test, type Locator, type Page, type TestInfo } from '@playwright/test'

type RGB = { red: number; green: number; blue: number }

function parseRGB(value: string): RGB {
  const channels = value.match(/[\d.]+/g)?.slice(0, 3).map(Number)
  if (!channels || channels.length !== 3) {
    throw new Error(`Expected an rgb() color, received ${value}`)
  }
  return { red: channels[0], green: channels[1], blue: channels[2] }
}

function relativeLuminance({ red, green, blue }: RGB): number {
  const linear = [red, green, blue].map((channel) => {
    const value = channel / 255
    return value <= 0.04045 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4
  })
  return 0.2126 * linear[0] + 0.7152 * linear[1] + 0.0722 * linear[2]
}

async function contrastRatio(foreground: Locator, background: Locator): Promise<number> {
  const [foregroundColor, backgroundColor] = await Promise.all([
    foreground.evaluate((element) => getComputedStyle(element).color),
    background.evaluate((element) => getComputedStyle(element).backgroundColor)
  ])
  const foregroundLuminance = relativeLuminance(parseRGB(foregroundColor))
  const backgroundLuminance = relativeLuminance(parseRGB(backgroundColor))
  const lighter = Math.max(foregroundLuminance, backgroundLuminance)
  const darker = Math.min(foregroundLuminance, backgroundLuminance)
  return (lighter + 0.05) / (darker + 0.05)
}
```

- [ ] **Step 2: Add failing contrast assertions to the completed-results flow**

Append these assertions immediately after the existing `Training complete!` visibility assertion in the test named `reveals in order and requires See results before the summary`:

```ts
  const completion = page.locator('.completion')
  const heading = page.getByRole('heading', { name: 'Training complete!' })
  const bodyCopy = page.getByText('You finished 1 puzzles.')
  const summaryCards = page.locator('.summary-grid strong')

  expect(await contrastRatio(heading, completion)).toBeGreaterThanOrEqual(4.5)
  expect(await contrastRatio(bodyCopy, completion)).toBeGreaterThanOrEqual(4.5)
  await expect(summaryCards).toHaveCount(4)
  for (const card of await summaryCards.all()) {
    expect(await contrastRatio(card, card)).toBeGreaterThanOrEqual(4.5)
  }
```

- [ ] **Step 3: Run the focused browser test and confirm the regression fails**

Run:

```bash
npm --prefix frontend run test:e2e -- --project=chromium tests/board-interactions.spec.ts --grep "reveals in order and requires See results before the summary"
```

Expected: FAIL on a `toBeGreaterThanOrEqual(4.5)` assertion because the current inherited foreground produces contrast near 1.08:1 on the panel or 1.14:1 on a summary card.

- [ ] **Step 4: Apply the minimal completion-scoped CSS correction**

Replace the completion-related rules in `frontend/src/components/puzzle/PuzzleScreen.svelte` with:

```css
  .completion { color: var(--ink-900); text-align: center; }
  .completion .eyebrow { color: var(--forest-600); }
  .celebration { display: block; color: var(--amber-400); font-size: 4.5rem; line-height: 1; }
  .summary-grid { display: grid; margin: 24px 0; grid-template-columns: 1fr 1fr; gap: 10px; }
  .summary-grid strong {
    padding: 14px 10px;
    border-radius: 12px;
    color: var(--ink-900);
    background: #f4e5bd;
  }
```

- [ ] **Step 5: Re-run the focused browser test and confirm it passes**

Run:

```bash
npm --prefix frontend run test:e2e -- --project=chromium tests/board-interactions.spec.ts --grep "reveals in order and requires See results before the summary"
```

Expected: PASS, including all heading, body, and four summary-card contrast assertions.

- [ ] **Step 6: Run the frontend regression suite and production checks**

Run:

```bash
npm --prefix frontend test -- --run
npm --prefix frontend run check
npm --prefix frontend run build
npm --prefix frontend run test:e2e -- --project=chromium tests/board-interactions.spec.ts
```

Expected: all Vitest tests pass, Svelte and Playwright TypeScript diagnostics report zero errors, the Vite production build succeeds, and the Chromium board-interaction suite passes.

- [ ] **Step 7: Perform a desktop browser visual check**

Open the deterministic test app, complete the one-puzzle reveal flow, select **See results**, and inspect the results state at a desktop viewport.

Expected: the page and header remain dark; the completion panel remains ivory; **Nice work** remains green; the heading, body, and warm tan summary cards use clearly readable dark text; the green **Back home** button and all result values remain unchanged.

- [ ] **Step 8: Commit only the contrast fix and its regression test**

Run:

```bash
git add frontend/tests/board-interactions.spec.ts frontend/src/components/puzzle/PuzzleScreen.svelte
git diff --cached --check
git commit -m "fix: improve quiz results contrast"
```

Expected: one commit containing only the Playwright regression and completion-scoped CSS changes; unrelated working-tree edits remain unstaged.
