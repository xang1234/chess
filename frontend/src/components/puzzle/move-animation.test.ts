import type { AppliedMove } from '../../lib/api'
import {
  animateAppliedMoves,
  type AnimationPort,
  type PositionFrame
} from '../chess/move-animation'

const startFen = '4k3/4p3/8/8/8/8/4P3/4K3 w - - 0 1'
const afterE4 = '4k3/4p3/8/8/4P3/8/8/4K3 b - - 0 1'
const afterE5 = '4k3/8/8/4p3/4P3/8/8/4K3 w - - 0 2'

function moves(): AppliedMove[] {
  return [
    { uci: 'e2e4', resultingFen: afterE4 },
    { uci: 'e7e5', resultingFen: afterE5 }
  ]
}

function animationPort(overrides: Partial<AnimationPort> = {}) {
  const frames: PositionFrame[] = []
  const delays: number[] = []
  const recoveries: string[] = []
  const port: AnimationPort = {
    setPosition: (frame) => { frames.push(frame) },
    delay: async (milliseconds) => { delays.push(milliseconds) },
    recover: (fen) => { recoveries.push(fen) },
    ...overrides
  }
  return { port, frames, delays, recoveries }
}

test('reconciles the optimistic move and animates later authoritative frames in order', async () => {
  const { port, frames, delays } = animationPort()
  const steps: string[] = []
  const controller = new AbortController()

  const result = await animateAppliedMoves({
    port,
    startingFen: startFen,
    appliedMoves: moves(),
    optimisticUci: 'e2e4',
    finalFen: afterE5,
    reducedMotion: false,
    signal: controller.signal,
    onStep: (kind, move) => { steps.push(`${kind}:${move.uci}`) }
  })

  expect(result).toEqual({ status: 'completed' })
  expect(frames).toEqual([
    { fen: afterE4, lastMove: ['e2', 'e4'], animate: false },
    { fen: afterE5, lastMove: ['e7', 'e5'], animate: true }
  ])
  expect(delays).toEqual([220, 180])
  expect(steps).toEqual(['move:e2e4', 'move:e7e5'])
})

test('waits for the final visual animation before resolving', async () => {
  let release: (() => void) | undefined
  const { port, frames } = animationPort({
    delay: () => new Promise<void>((resolve) => { release = resolve })
  })
  let settled = false
  const result = animateAppliedMoves({
    port,
    startingFen: startFen,
    appliedMoves: [{ uci: 'e2e4', resultingFen: afterE4 }],
    finalFen: afterE4,
    reducedMotion: false,
    signal: new AbortController().signal
  }).then((value) => {
    settled = true
    return value
  })

  await Promise.resolve()
  expect(frames).toHaveLength(1)
  expect(settled).toBe(false)
  expect(release).toBeTypeOf('function')
  release?.()

  await expect(result).resolves.toEqual({ status: 'completed' })
})

test('reduced motion applies only the final authoritative position', async () => {
  const { port, frames, delays } = animationPort()

  const result = await animateAppliedMoves({
    port,
    startingFen: startFen,
    appliedMoves: moves(),
    optimisticUci: 'e2e4',
    finalFen: afterE5,
    reducedMotion: true,
    signal: new AbortController().signal
  })

  expect(result).toEqual({ status: 'completed' })
  expect(frames).toEqual([
    { fen: afterE5, lastMove: ['e7', 'e5'], animate: false }
  ])
  expect(delays).toEqual([])
})

test('aborts without applying or recovering subsequent frames', async () => {
  const controller = new AbortController()
  controller.abort()
  const { port, frames, recoveries } = animationPort()

  const result = await animateAppliedMoves({
    port,
    startingFen: startFen,
    appliedMoves: moves(),
    finalFen: afterE5,
    reducedMotion: false,
    signal: controller.signal
  })

  expect(result).toEqual({ status: 'aborted' })
  expect(frames).toEqual([])
  expect(recoveries).toEqual([])
})

test('uses the caller-owned recovery boundary after a position sink fails', async () => {
  const setPosition = vi.fn(() => { throw new Error('board detached') })
  const { port, recoveries } = animationPort({ setPosition })

  const result = await animateAppliedMoves({
    port,
    startingFen: startFen,
    appliedMoves: moves(),
    finalFen: afterE5,
    reducedMotion: false,
    signal: new AbortController().signal
  })

  expect(setPosition).toHaveBeenCalledTimes(1)
  expect(recoveries).toEqual([afterE5])
  expect(result).toMatchObject({ status: 'reconciled', warning: expect.stringContaining('board detached') })
})

test('contains a recovery failure and keeps the reconciled outcome usable', async () => {
  const { port } = animationPort({
    setPosition: () => { throw new Error('sink failed') },
    recover: () => { throw new Error('remount failed') }
  })

  const result = await animateAppliedMoves({
    port,
    startingFen: startFen,
    appliedMoves: moves(),
    finalFen: afterE5,
    reducedMotion: false,
    signal: new AbortController().signal
  })

  expect(result).toMatchObject({
    status: 'reconciled',
    warning: expect.stringMatching(/sink failed.*remount failed/)
  })
})
