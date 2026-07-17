import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import PromotionDialog from './PromotionDialog.svelte'

function focusTarget(): HTMLButtonElement {
  const target = document.createElement('button')
  target.textContent = 'Board focus target'
  target.dataset.promotionFocusTarget = 'true'
  document.body.appendChild(target)
  target.focus()
  return target
}

afterEach(() => {
  document.querySelectorAll('[data-promotion-focus-target]').forEach((target) => target.remove())
})

test('offers only legal suffixes, focuses the first choice, and restores focus after choosing', async () => {
  const returnFocus = focusTarget()
  const choices: string[] = []
  const { component } = render(PromotionDialog, {
    choices: ['q', 'n'],
    color: 'white',
    returnFocus
  })
  component.$on('choose', (event) => choices.push(event.detail.promotion))

  const queen = screen.getByRole('button', { name: 'Promote to queen' })
  const knight = screen.getByRole('button', { name: 'Promote to knight' })
  expect(screen.queryByRole('button', { name: 'Promote to rook' })).not.toBeInTheDocument()
  expect(screen.queryByRole('button', { name: 'Promote to bishop' })).not.toBeInTheDocument()
  await waitFor(() => expect(queen).toHaveFocus())

  await fireEvent.click(knight)
  expect(choices).toEqual(['n'])
  expect(returnFocus).toHaveFocus()
})

test('Escape cancels promotion and restores board focus', async () => {
  const returnFocus = focusTarget()
  const cancellations: number[] = []
  const { component } = render(PromotionDialog, {
    choices: ['r', 'b'],
    color: 'black',
    returnFocus
  })
  component.$on('cancel', () => cancellations.push(1))

  await fireEvent.keyDown(screen.getByRole('dialog'), { key: 'Escape' })

  expect(cancellations).toEqual([1])
  expect(returnFocus).toHaveFocus()
})

test('contains Tab focus within the modal choices and Cancel button', async () => {
  render(PromotionDialog, {
    choices: ['q', 'n'],
    color: 'white',
    returnFocus: focusTarget()
  })
  const queen = screen.getByRole('button', { name: 'Promote to queen' })
  const cancel = screen.getByRole('button', { name: 'Cancel' })
  await waitFor(() => expect(queen).toHaveFocus())

  cancel.focus()
  await fireEvent.keyDown(cancel, { key: 'Tab' })
  expect(queen).toHaveFocus()

  queen.focus()
  await fireEvent.keyDown(queen, { key: 'Tab', shiftKey: true })
  expect(cancel).toHaveFocus()
})
