import type { Api } from '@lichess-org/chessground/api'
import type { Config } from '@lichess-org/chessground/config'
import type { Key } from '@lichess-org/chessground/types'
import {
  createChessgroundAdapter,
  type BoardCallbacks,
  type BoardInteraction,
  type GroundFactory
} from './chessground-adapter'

const initialFen = '4k3/8/8/8/8/8/4P3/4K3 w - - 0 1'

function interaction(overrides: Partial<BoardInteraction> = {}): BoardInteraction {
  return {
    orientation: 'white',
    legalMoves: ['e2e3', 'e2e4'],
    inputEnabled: true,
    reducedMotion: false,
    ...overrides
  }
}

function harness(
  suppliedInteraction = interaction(),
  callbacks: BoardCallbacks = { onRoute: vi.fn(), onSelect: vi.fn() }
) {
  const initialConfigs: Config[] = []
  const updates: Config[] = []
  const selectSquare = vi.fn()
  const destroy = vi.fn()
  const api = {
    set: (config: Config) => { updates.push(config) },
    selectSquare,
    destroy
  } as unknown as Api
  const factory: GroundFactory = vi.fn((_element, config) => {
    if (config) initialConfigs.push(config)
    return api
  })
  const element = document.createElement('div')
  const adapter = createChessgroundAdapter(
    element,
    initialFen,
    suppliedInteraction,
    callbacks,
    factory
  )
  return {
    adapter,
    callbacks,
    destroy,
    element,
    factory,
    initial: initialConfigs[0],
    selectSquare,
    updates
  }
}

function destinations(config: Config): Map<Key, Key[]> {
  return config.movable?.dests ?? new Map()
}

test('configures a controlled legal-move board for the solver color', () => {
  const { element, factory, initial } = harness()

  expect(factory).toHaveBeenCalledWith(element, expect.any(Object))
  expect(initial).toMatchObject({
    fen: initialFen,
    orientation: 'white',
    turnColor: 'white',
    coordinates: true,
    coordinatesOnSquares: false,
    autoCastle: true,
    disableContextMenu: true,
    blockTouchScroll: true,
    jsHover: true,
    animation: { enabled: true, duration: 180 },
    movable: {
      free: false,
      color: 'white',
      showDests: true,
      rookCastle: true
    },
    premovable: { enabled: false },
    predroppable: { enabled: false },
    draggable: { enabled: true, showGhost: true, deleteOnDropOff: false },
    selectable: { enabled: true },
    drawable: { enabled: false, visible: true, autoShapes: [] }
  })
  expect(initial).not.toHaveProperty('trustAllEvents', true)
  expect(initial).not.toHaveProperty('viewOnly')
  expect([...destinations(initial).entries()]).toEqual([['e2', ['e3', 'e4']]])
})

test('locks input without view-only mode and never puts FEN in interaction updates', () => {
  const { adapter, updates } = harness()

  adapter.configure(interaction({
    orientation: 'black',
    legalMoves: ['e7e5'],
    inputEnabled: false
  }))

  const update = updates.at(-1)!
  expect(update).not.toHaveProperty('fen')
  expect(update).not.toHaveProperty('viewOnly')
  expect(update).toMatchObject({
    orientation: 'black',
    turnColor: 'black',
    movable: { color: undefined },
    draggable: { enabled: false },
    selectable: { enabled: false }
  })
  expect(destinations(update).size).toBe(0)
})

test('merges wrong, hint, and keyboard markers alongside the native last move', () => {
  const { initial } = harness(interaction({
    lastMove: ['e2', 'e4'],
    wrongMove: ['e2', 'e4'],
    hintSource: 'e2',
    hintTarget: 'e4',
    keyboardCursor: 'e2'
  }))

  expect(initial.lastMove).toEqual(['e2', 'e4'])
  expect(initial.highlight?.custom).toEqual(new Map([
    ['e2', 'wrong-source hint-source keyboard-cursor'],
    ['e4', 'wrong-target hint-target']
  ]))
})

test('composes read-only course annotations and updates or removes them through configure', () => {
  const { adapter, initial, updates } = harness(interaction({
    hintSource: 'e2',
    annotations: [
      { kind: 'square', from: 'd4' },
      { kind: 'arrow', from: 'c2', to: 'c3' }
    ]
  }))

  expect(initial.highlight?.custom).toEqual(new Map([
    ['e2', 'hint-source'],
    ['d4', 'opening-annotation']
  ]))
  expect(initial.drawable).toMatchObject({
    enabled: false,
    visible: true,
    autoShapes: [{ orig: 'c2', dest: 'c3', brush: 'green' }]
  })

  adapter.configure(interaction({ annotations: [] }))
  expect(updates.at(-1)?.highlight?.custom).toEqual(new Map())
  expect(updates.at(-1)?.drawable).toMatchObject({
    enabled: false,
    visible: true,
    autoShapes: []
  })
})

test('forwards only currently legal routes and clears immobile selections', () => {
  const callbacks: BoardCallbacks = { onRoute: vi.fn(), onSelect: vi.fn() }
  const { adapter, initial, selectSquare, updates } = harness(interaction(), callbacks)

  initial.events?.select?.('a1')
  expect(callbacks.onSelect).toHaveBeenCalledWith('a1')
  expect(selectSquare).toHaveBeenCalledWith(null)

  initial.events?.select?.('e2')
  expect(callbacks.onSelect).toHaveBeenCalledWith('e2')
  expect(selectSquare).toHaveBeenCalledTimes(1)

  initial.movable?.events?.after?.('e2', 'e4', { premove: false })
  initial.movable?.events?.after?.('e2', 'e5', { premove: false })
  expect(callbacks.onRoute).toHaveBeenCalledTimes(1)
  expect(callbacks.onRoute).toHaveBeenCalledWith('e2', 'e4')

  adapter.configure(interaction({ legalMoves: ['a2a3'] }))
  updates.at(-1)?.movable?.events?.after?.('e2', 'e4', { premove: false })
  expect(callbacks.onRoute).toHaveBeenCalledTimes(1)
})

test('does not echo programmatic keyboard selection as a second pointer selection', () => {
  const callbacks: BoardCallbacks = { onRoute: vi.fn(), onSelect: vi.fn() }
  let selectionEvent: ((key: Key) => void) | undefined
  const api = {
    set: vi.fn(),
    selectSquare: vi.fn((key: Key | null) => {
      if (key) selectionEvent?.(key)
    }),
    destroy: vi.fn()
  } as unknown as Api
  const factory: GroundFactory = vi.fn((_element, config) => {
    selectionEvent = config?.events?.select
    return api
  })
  const adapter = createChessgroundAdapter(
    document.createElement('div'),
    initialFen,
    interaction(),
    callbacks,
    factory
  )

  adapter.selectSquare('e2')

  expect(callbacks.onSelect).not.toHaveBeenCalled()
  selectionEvent?.('e2')
  expect(callbacks.onSelect).toHaveBeenCalledOnce()
})

test('uses setPosition as the sole authoritative FEN boundary', () => {
  const { adapter, updates } = harness()

  adapter.configure(interaction({ hintSource: 'e2' }))
  adapter.setPosition('4k3/8/8/8/4P3/8/8/4K3 b - - 0 1', ['e2', 'e4'], true)
  adapter.configure(interaction({ reducedMotion: true }))
  adapter.setPosition(initialFen, undefined, true)

  expect(updates.filter((config) => Object.hasOwn(config, 'fen'))).toHaveLength(2)
  expect(updates[1]).toMatchObject({
    fen: '4k3/8/8/8/4P3/8/8/4K3 b - - 0 1',
    lastMove: ['e2', 'e4'],
    animation: { enabled: true, duration: 180 }
  })
  expect(updates[3]).toMatchObject({
    fen: initialFen,
    lastMove: undefined,
    animation: { enabled: false, duration: 180 }
  })
})

test('rejects malformed legal UCI before constructing or updating the board', () => {
  const factory: GroundFactory = vi.fn()
  expect(() => createChessgroundAdapter(
    document.createElement('div'),
    initialFen,
    interaction({ legalMoves: ['e2-e4'] }),
    { onRoute: vi.fn(), onSelect: vi.fn() },
    factory
  )).toThrow(/e2-e4/)
  expect(factory).not.toHaveBeenCalled()

  const { adapter, updates } = harness()
  expect(() => adapter.configure(interaction({ legalMoves: ['bad'] }))).toThrow(/bad/)
  expect(updates).toEqual([])
})

test('contains deferred callbacks after idempotent destruction', () => {
  const callbacks: BoardCallbacks = { onRoute: vi.fn(), onSelect: vi.fn() }
  const { adapter, destroy, initial } = harness(interaction(), callbacks)
  const select = initial.events?.select
  const after = initial.movable?.events?.after

  adapter.destroy()
  adapter.destroy()
  select?.('e2')
  after?.('e2', 'e4', { premove: false })

  expect(destroy).toHaveBeenCalledTimes(1)
  expect(callbacks.onSelect).not.toHaveBeenCalled()
  expect(callbacks.onRoute).not.toHaveBeenCalled()
})
