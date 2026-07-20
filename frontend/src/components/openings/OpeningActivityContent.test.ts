import { fireEvent, render, screen } from '@testing-library/svelte'
import { fakeOpeningSession } from '../../test-fakes'
import type { AppliedMoveFrames } from '../../lib/api'
import OpeningActivityContent from './OpeningActivityContent.svelte'

test('renders teaching, comparisons, replay controls, and collapsed deeper analysis', async () => {
  const activity = {
    ...fakeOpeningSession.current,
    kind: 'comparison' as const,
    teachingNoteTexts: ['Both plans develop before the central break.'],
    referenceNoteTexts: ['Dense source evaluation stays optional.'],
    comparison: [
      { label: 'Active centre', moves: ['c3', 'd4'] },
      { label: 'Quiet setup', moves: ['d3', 'Re1'] }
    ],
    movesToHere: [{
      uci: 'e2e4',
      resultingFen: fakeOpeningSession.current.currentFen
    }] as AppliedMoveFrames,
    referenceSections: [{
      activityId: 'deep-line',
      title: 'A concrete sideline',
      instruction: 'Compare the move order.',
      noteTexts: ['A historical game citation.'],
      annotations: []
    }]
  }
  const { component } = render(OpeningActivityContent, { activity, canReplayDemonstration: false })
  const replay: string[] = []
  component.$on('replayMoves', () => replay.push('moves'))

  expect(screen.getByText('Both plans develop before the central break.')).toBeVisible()
  expect(screen.getByText('Active centre')).toBeVisible()
  expect(screen.getByText('Quiet setup')).toBeVisible()
  expect(screen.getByText('Dense source evaluation stays optional.')).not.toBeVisible()
  expect(screen.getByText('A historical game citation.')).not.toBeVisible()

  await fireEvent.click(screen.getByRole('button', { name: 'Replay moves to here' }))
  expect(replay).toEqual(['moves'])
  await fireEvent.click(screen.getByText('Deeper analysis'))
  expect(screen.getByText('Dense source evaluation stays optional.')).toBeVisible()
  expect(screen.getByText('A historical game citation.')).toBeVisible()
})

test('offers a separate demonstration replay when authoritative frames are ready', async () => {
  const { component } = render(OpeningActivityContent, {
    activity: { ...fakeOpeningSession.current, kind: 'demonstration' },
    canReplayDemonstration: true
  })
  const replay = vi.fn()
  component.$on('replayDemonstration', replay)

  await fireEvent.click(screen.getByRole('button', { name: 'Replay demonstration' }))
  expect(replay).toHaveBeenCalledOnce()
})
