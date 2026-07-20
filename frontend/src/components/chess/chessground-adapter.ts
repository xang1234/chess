import { Chessground } from '@lichess-org/chessground'
import type { Api } from '@lichess-org/chessground/api'
import type { Config } from '@lichess-org/chessground/config'
import type { Dests, Key, KeyPair, SquareClasses } from '@lichess-org/chessground/types'
import { groupLegalMoves } from '../../lib/uci'

const ANIMATION_DURATION_MS = 180

export type GroundFactory = (element: HTMLElement, config?: Config) => Api

export type BoardCallbacks = {
  onRoute(from: Key, to: Key): void
  onSelect(key: Key): void
}

export type BoardAnnotation =
  | { kind: 'square'; from: Key }
  | { kind: 'arrow'; from: Key; to: Key }

export type BoardInteraction = {
  orientation: 'white' | 'black'
  legalMoves: readonly string[]
  inputEnabled: boolean
  lastMove?: KeyPair
  wrongMove?: KeyPair
  hintSource?: Key
  hintTarget?: Key
  keyboardCursor?: Key
  annotations?: readonly BoardAnnotation[]
  reducedMotion: boolean
}

export interface ChessBoardAdapter {
  configure(interaction: BoardInteraction): void
  setPosition(fen: string, lastMove?: KeyPair, animate?: boolean): void
  selectSquare(key: Key | null): void
  destroy(): void
}

export type ChessgroundAdapterFactory = (
  element: HTMLElement,
  initialFen: string,
  interaction: BoardInteraction,
  callbacks: BoardCallbacks
) => ChessBoardAdapter

export function createChessgroundAdapter(
  element: HTMLElement,
  initialFen: string,
  initialInteraction: BoardInteraction,
  callbacks: BoardCallbacks,
  factory: GroundFactory = Chessground
): ChessBoardAdapter {
  let interaction = initialInteraction
  let destinations = validatedDestinations(interaction.legalMoves)
  let destroyed = false
  let api: Api | undefined
  const programmaticSelections: Key[] = []

  const onSelect = (key: Key): void => {
    if (destroyed) return
    const programmaticIndex = programmaticSelections.indexOf(key)
    if (programmaticIndex >= 0) {
      programmaticSelections.splice(programmaticIndex, 1)
      return
    }
    if (!interaction.inputEnabled) {
      api?.selectSquare(null)
      return
    }
    callbacks.onSelect(key)
    if (!destinations.has(key)) api?.selectSquare(null)
  }

  const onRoute = (from: Key, to: Key): void => {
    if (destroyed || !interaction.inputEnabled) return
    if (destinations.get(from)?.includes(to)) {
      callbacks.onRoute(from, to)
    } else {
      api?.selectSquare(null)
    }
  }

  api = factory(element, {
    fen: initialFen,
    ...interactionConfig(interaction, destinations, onSelect, onRoute)
  })

  return {
    configure: (nextInteraction) => {
      if (destroyed) return
      const nextDestinations = validatedDestinations(nextInteraction.legalMoves)
      interaction = nextInteraction
      destinations = nextDestinations
      api?.set(interactionConfig(interaction, destinations, onSelect, onRoute))
    },
    setPosition: (fen, lastMove, animate = false) => {
      if (destroyed) return
      api?.set({
        fen,
        lastMove,
        animation: {
          enabled: animate && !interaction.reducedMotion,
          duration: ANIMATION_DURATION_MS
        }
      })
    },
    selectSquare: (key) => {
      if (destroyed) return
      if (key) programmaticSelections.push(key)
      try {
        api?.selectSquare(key)
      } catch (error) {
        const queuedIndex = key ? programmaticSelections.lastIndexOf(key) : -1
        if (queuedIndex >= 0) programmaticSelections.splice(queuedIndex, 1)
        throw error
      }
    },
    destroy: () => {
      if (destroyed) return
      destroyed = true
      programmaticSelections.length = 0
      api?.destroy()
    }
  }
}

function validatedDestinations(legalMoves: readonly string[]): Dests {
  const grouped = groupLegalMoves([...legalMoves])
  return new Map(
    [...grouped].map(([from, to]) => [from as Key, [...to] as Key[]])
  )
}

function interactionConfig(
  interaction: BoardInteraction,
  destinations: Dests,
  onSelect: (key: Key) => void,
  onRoute: (from: Key, to: Key) => void
): Config {
  const enabled = interaction.inputEnabled
  return {
    orientation: interaction.orientation,
    turnColor: interaction.orientation,
    lastMove: interaction.lastMove,
    coordinates: true,
    coordinatesOnSquares: false,
    autoCastle: true,
    disableContextMenu: true,
    blockTouchScroll: true,
    jsHover: true,
    highlight: {
      lastMove: true,
      custom: markerClasses(interaction)
    },
    animation: {
      enabled: !interaction.reducedMotion,
      duration: ANIMATION_DURATION_MS
    },
    movable: {
      free: false,
      color: enabled ? interaction.orientation : undefined,
      dests: enabled ? destinations : new Map(),
      showDests: true,
      events: { after: onRoute },
      rookCastle: true
    },
    premovable: { enabled: false },
    predroppable: { enabled: false },
    draggable: {
      enabled,
      showGhost: true,
      deleteOnDropOff: false
    },
    selectable: { enabled },
    events: { select: onSelect },
    drawable: {
      enabled: false,
      visible: true,
      autoShapes: (interaction.annotations ?? [])
        .filter((annotation): annotation is Extract<BoardAnnotation, { kind: 'arrow' }> => (
          annotation.kind === 'arrow'
        ))
        .map((annotation) => ({
          orig: annotation.from,
          dest: annotation.to,
          brush: 'green'
        }))
    }
  }
}

function markerClasses(interaction: BoardInteraction): SquareClasses {
  const classes: SquareClasses = new Map()
  if (interaction.wrongMove) {
    addMarker(classes, interaction.wrongMove[0], 'wrong-source')
    addMarker(classes, interaction.wrongMove[1], 'wrong-target')
  }
  if (interaction.hintSource) addMarker(classes, interaction.hintSource, 'hint-source')
  if (interaction.hintTarget) addMarker(classes, interaction.hintTarget, 'hint-target')
  if (interaction.keyboardCursor) {
    addMarker(classes, interaction.keyboardCursor, 'keyboard-cursor')
  }
  for (const annotation of interaction.annotations ?? []) {
    if (annotation.kind === 'square') addMarker(classes, annotation.from, 'opening-annotation')
  }
  return classes
}

function addMarker(classes: SquareClasses, key: Key, marker: string): void {
  const existing = classes.get(key)
  classes.set(key, existing ? `${existing} ${marker}` : marker)
}
