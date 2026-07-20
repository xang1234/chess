import type { Page } from '@playwright/test'
import type { TestBackendState, TestWindow } from './test-backend-types'

export async function backendState<Value>(page: Page, key: keyof TestBackendState): Promise<Value> {
  return page.evaluate((name) => {
    const state = (window as unknown as TestWindow).__testBackend
    return state[name] as Value
  }, key)
}

export async function holdImportOpen(page: Page): Promise<void> {
  await page.evaluate(() => (window as unknown as TestWindow).__testBackend.holdImportOpen())
}

export async function selectedImportPath(page: Page): Promise<string> {
  return page.evaluate(() => (window as unknown as TestWindow).__testBackend.selectedImportPath())
}

export async function selectedCoursePath(page: Page): Promise<string> {
  return page.evaluate(() => (window as unknown as TestWindow).__testBackend.selectedCoursePath())
}

export async function reportImportProgress(
  page: Page,
  rowsRead: number,
  bytesRead: number
): Promise<void> {
  await page.evaluate(([rows, bytes]) => {
    (window as unknown as TestWindow).__testBackend.reportImportProgress(rows, bytes)
  }, [rowsRead, bytesRead])
}

export async function observeSemanticFrames(page: Page): Promise<void> {
  await page.evaluate(() => {
    const root = document.querySelector('[role="grid"]')
    if (!root) throw new Error('semantic chess grid is missing')
    const state = (window as unknown as TestWindow).__testBackend
    const snapshot = () => {
      const labels = [...root.querySelectorAll('[role="gridcell"]')]
        .map((cell) => cell.getAttribute('aria-label') ?? '')
        .filter((label) => !label.startsWith('Empty '))
        .sort()
      const previous = state.semanticFrames.at(-1)
      if (JSON.stringify(previous) !== JSON.stringify(labels)) {
        state.semanticFrames.push(labels)
      }
    }
    let queued = false
    new MutationObserver(() => {
      if (queued) return
      queued = true
      queueMicrotask(() => {
        queued = false
        snapshot()
      })
    }).observe(root, { attributes: true, subtree: true, attributeFilter: ['aria-label'] })
    snapshot()
  })
}
