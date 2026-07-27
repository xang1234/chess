import { fireEvent, render, screen } from '@testing-library/svelte'
import { fakeOpeningSession } from '../../test-fakes'
import type { AppliedMoveFrames, OpeningActivityView } from '../../lib/api'
import OpeningActivityContent from './OpeningActivityContent.svelte'

function comparisonActivity(): OpeningActivityView {
  return {
    ...fakeOpeningSession.current,
    kind: 'comparison' as const,
    teachingNoteTexts: ['Both plans develop before the central break.'],
    referenceNoteTexts: ['Dense source evaluation stays optional.'],
    comparison: [
      {
        label: 'Active centre',
        moves: ['c3', 'd4'],
        frames: [
          { uci: 'c2c3', resultingFen: 'after-c3' },
          { uci: 'd2d4', resultingFen: 'after-d4' }
        ] as AppliedMoveFrames
      },
      {
        label: 'Quiet setup',
        moves: ['d3', 'Re1'],
        frames: [
          { uci: 'd2d3', resultingFen: 'after-d3' },
          { uci: 'f1e1', resultingFen: 'after-re1' }
        ] as AppliedMoveFrames
      }
    ],
    movesToHere: [{
      uci: 'e2e4',
      resultingFen: fakeOpeningSession.current.currentFen
    }] as AppliedMoveFrames,
    movesToHereSan: ['e4'],
    referenceSections: [{
      activityId: 'deep-line',
      title: 'A concrete sideline',
      instruction: 'Compare the move order.',
      noteTexts: ['A historical game citation.'],
      annotations: []
    }]
  }
}

test('renders teaching, comparisons, replay controls, and collapsed deeper analysis', async () => {
  const activity = comparisonActivity()
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

test('dispatches playable comparisons and exposes reset', async () => {
  const { component } = render(OpeningActivityContent, {
    activity: comparisonActivity(),
    comparisonPreviewIndex: 0,
    comparisonPreviewPlaying: false
  })
  const previews: number[] = []
  const resets = vi.fn()
  component.$on('previewComparison', (event) => previews.push(event.detail.index))
  component.$on('resetComparison', resets)

  expect(screen.getByRole('button', {
    name: 'Play Active centre: c3, d4'
  })).toHaveAttribute('aria-pressed', 'true')
  expect(screen.getByText('c3 → d4')).toBeVisible()
  await fireEvent.click(screen.getByRole('button', {
    name: 'Play Quiet setup: d3, Re1'
  }))
  expect(previews).toEqual([1])
  await fireEvent.click(screen.getByRole('button', { name: 'Reset position' }))
  expect(resets).toHaveBeenCalledOnce()
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
