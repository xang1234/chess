import type { SoundName, SoundService } from '../../lib/sound'
import type { BoardEffects } from './board-effects'
import { InteractiveBoardRuntime } from './interactive-board-runtime'

type FakeSound = SoundService & {
  destroyed: boolean
  played: SoundName[]
}

function fakeSound(muted = false): FakeSound {
  let currentMuted = muted
  return {
    destroyed: false,
    played: [],
    get muted() { return currentMuted },
    unlock: async () => {},
    play(name) { this.played.push(name) },
    setMuted(next) { currentMuted = next },
    toggleMuted() {
      currentMuted = !currentMuted
      return currentMuted
    },
    destroy() { this.destroyed = true }
  }
}

function runtimeFixture(options: { reducedMotion?: boolean; muted?: boolean } = {}) {
  const sound = fakeSound(options.muted)
  const effects: BoardEffects = {
    createSound: () => sound,
    delay: async () => {},
    prefersReducedMotion: () => options.reducedMotion ?? false
  }
  const published: Array<{
    fen: string
    lastMove: [string, string] | undefined
    replaceBoard: boolean
  }> = []
  const runtime = new InteractiveBoardRuntime(effects, {
    publishPosition: (fen, lastMove, replaceBoard) => {
      published.push({ fen, lastMove, replaceBoard })
    }
  })
  return { runtime, sound, published }
}

test('starting a request aborts its predecessor and rejects stale tokens', () => {
  const { runtime } = runtimeFixture()
  runtime.mount()
  const first = runtime.startRequest()
  const second = runtime.startRequest()

  expect(first.controller.signal.aborted).toBe(true)
  expect(runtime.isCurrent(first)).toBe(false)
  expect(runtime.isCurrent(second)).toBe(true)
})

test('destroy aborts requests, destroys sound, and detaches the board', async () => {
  const { runtime, sound } = runtimeFixture()
  const board = { setPosition: vi.fn() }
  runtime.mount()
  runtime.attachBoard(board)
  const token = runtime.startRequest()

  runtime.destroy()

  expect(token.controller.signal.aborted).toBe(true)
  expect(sound.destroyed).toBe(true)
  expect(runtime.isCurrent(token)).toBe(false)
  const warning = await runtime.reconcilePosition('saved-fen', new AbortController().signal, false)
  expect(warning).toContain('Chess board is unavailable')
  expect(board.setPosition).not.toHaveBeenCalled()
})

test('a throwing board republishes the authoritative position with replacement', async () => {
  const { runtime, published } = runtimeFixture()
  runtime.mount()
  runtime.attachBoard({
    setPosition: () => { throw new Error('board exploded') }
  })

  const warning = await runtime.reconcilePosition('saved-fen', new AbortController().signal, false)

  expect(published).toEqual([
    { fen: 'saved-fen', lastMove: undefined, replaceBoard: false },
    { fen: 'saved-fen', lastMove: undefined, replaceBoard: true }
  ])
  expect(warning).toBe(
    'Board reconciliation failed: board exploded. The saved position was restored.'
  )
})

test('warning consumption deduplicates messages and clears pending board warnings', () => {
  const { runtime } = runtimeFixture()
  runtime.noteBoardError('one')

  expect(runtime.consumeWarnings('one', 'one', 'two')).toBe('one two')
  expect(runtime.consumeWarnings()).toBe('')
})

test('mount reports reduced-motion and sound preferences from effects', () => {
  const { runtime } = runtimeFixture({ reducedMotion: true, muted: true })

  expect(runtime.mount()).toEqual({ reducedMotion: true, soundMuted: true })
})
