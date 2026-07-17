import type { CompletedSessionView } from '../../lib/api'
import type { Square } from '../../lib/uci'
import type { PuzzleState } from './puzzle-state'

export type RenderCommon = {
  announcement: string
  boardGeneration: number
  reducedMotion: boolean
  soundMuted: boolean
}

export type PuzzleRenderState =
  | (RenderCommon & { kind: 'loading' })
  | (RenderCommon & { kind: 'puzzle'; state: PuzzleState })
  | (RenderCommon & {
    kind: 'summary'
    session: CompletedSessionView
    notice: string
  })

export function initialRenderState(): PuzzleRenderState {
  return {
    kind: 'loading',
    announcement: '',
    boardGeneration: 0,
    reducedMotion: false,
    soundMuted: false
  }
}

export function acceptsInput(state: PuzzleState): boolean {
  return state.phase === 'ready' || state.phase === 'incorrect' ||
    (state.phase === 'failed' && state.recoverable)
}

export function feedbackMessage(state: PuzzleState): string {
  switch (state.phase) {
    case 'prelude': return 'Watch the last move…'
    case 'ready': return state.hint?.text ?? ''
    case 'requesting': return requestMessage(state.operation)
    case 'animating': return 'Good move…'
    case 'incorrect': return state.message
    case 'solved': return state.outcome === 'correct' ? 'Correct!' : 'Solution shown'
    case 'failed': return state.message
  }
}

export function requestMessage(operation: 'move' | 'hint' | 'reveal' | 'pause'): string {
  switch (operation) {
    case 'move': return 'Checking that move…'
    case 'hint': return 'Finding a hint…'
    case 'reveal': return 'Preparing the solution…'
    case 'pause': return 'Pausing…'
  }
}

export function optionalSquare(value: string | undefined): Square | undefined {
  return value && /^[a-h][1-8]$/.test(value) ? value as Square : undefined
}
