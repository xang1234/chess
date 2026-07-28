import { render, screen, within } from '@testing-library/svelte'
import type { OpeningMoveFeedbackDetail } from '../../lib/api'
import OpeningMoveFeedback from './OpeningMoveFeedback.svelte'

const detail: OpeningMoveFeedbackDetail = {
  coverage: 'visible',
  playedSan: 'b4',
  variationName: 'Evans idea',
  evaluation: { code: 'unclear' },
  noteText: 'A sound alternative that keeps pressure on the centre.',
  expectedSan: 'c3',
  planText: 'Prepare d4 while keeping the bishop active.'
}

test('renders structured move feedback in one polite atomic status region', () => {
  render(OpeningMoveFeedback, {
    props: {
      feedback: 'alternative',
      detail,
      orientation: 'white',
      fallbackMessage: 'Playable alternative.'
    }
  })

  const status = screen.getByRole('status')
  expect(status).toHaveAttribute('aria-live', 'polite')
  expect(status).toHaveAttribute('aria-atomic', 'true')
  expect(within(status).getByRole('heading', { name: 'b4 — Evans idea' })).toBeVisible()
  expect(status).toHaveTextContent('∞ Unclear position.')
  expect(status).toHaveTextContent('A sound alternative that keeps pressure on the centre.')
  expect(status).toHaveTextContent(
    'This lesson is practising c3: Prepare d4 while keeping the bishop active.'
  )
  expect(within(status).queryAllByRole('button')).toHaveLength(0)
})

test('keeps the server message as a compatibility fallback', () => {
  render(OpeningMoveFeedback, {
    props: {
      feedback: 'off_course',
      detail: null,
      orientation: 'black',
      fallbackMessage: 'This lesson is practising c3.'
    }
  })

  const status = screen.getByRole('status')
  expect(status).toHaveTextContent('This lesson is practising c3.')
  expect(within(status).queryByRole('heading')).not.toBeInTheDocument()
  expect(within(status).queryAllByRole('button')).toHaveLength(0)
})
