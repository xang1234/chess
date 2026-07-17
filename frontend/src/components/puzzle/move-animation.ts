import type { AppliedMove } from '../../lib/api'
import { classifyMoveFeedback } from '../../lib/move-feedback'
import { moveSquares, type Square } from '../../lib/uci'

const BOARD_ANIMATION_MS = 180
const REPLY_PAUSE_MS = 220

export type PositionFrame = {
  fen: string
  lastMove?: [Square, Square]
  animate: boolean
}

export type AnimationPort = {
  setPosition(frame: PositionFrame): void
  delay(milliseconds: number, signal: AbortSignal): Promise<void>
  recover(finalFen: string): void
}

export type AnimationResult = {
  status: 'completed' | 'aborted' | 'reconciled'
  warning?: string
}

export async function animateAppliedMoves(options: {
  port: AnimationPort
  startingFen: string
  appliedMoves: readonly AppliedMove[]
  optimisticUci?: string
  finalFen: string
  reducedMotion: boolean
  signal: AbortSignal
  onStep?: (kind: 'move' | 'capture', move: AppliedMove) => void
}): Promise<AnimationResult> {
  const {
    port,
    startingFen,
    appliedMoves,
    optimisticUci,
    finalFen,
    reducedMotion,
    signal,
    onStep
  } = options

  if (signal.aborted) return { status: 'aborted' }

  try {
    if (reducedMotion) {
      const finalMove = appliedMoves.at(-1)
      const frame: PositionFrame = {
        fen: finalFen,
        animate: false
      }
      if (finalMove) {
        frame.lastMove = moveSquares(finalMove.uci)
        const previousFen = appliedMoves.at(-2)?.resultingFen ?? startingFen
        onStep?.(
          classifyMoveFeedback(previousFen, finalMove.resultingFen, finalMove.uci),
          finalMove
        )
      }
      port.setPosition(frame)
      return { status: 'completed' }
    }

    let currentFen = startingFen
    for (const [index, move] of appliedMoves.entries()) {
      if (signal.aborted) return { status: 'aborted' }
      if (index > 0) {
        await port.delay(REPLY_PAUSE_MS, signal)
        if (signal.aborted) return { status: 'aborted' }
      }

      const optimistic = index === 0 && optimisticUci === move.uci
      const kind = classifyMoveFeedback(currentFen, move.resultingFen, move.uci)
      port.setPosition({
        fen: move.resultingFen,
        lastMove: moveSquares(move.uci),
        animate: !optimistic
      })
      onStep?.(kind, move)

      if (!optimistic) {
        await port.delay(BOARD_ANIMATION_MS, signal)
        if (signal.aborted) return { status: 'aborted' }
      }
      currentFen = move.resultingFen
    }

    if (currentFen !== finalFen) {
      port.setPosition({ fen: finalFen, animate: false })
    }
    return { status: 'completed' }
  } catch (error) {
    if (signal.aborted) return { status: 'aborted' }

    const animationFailure = errorMessage(error)
    try {
      port.recover(finalFen)
      return {
        status: 'reconciled',
        warning: `Board animation failed: ${animationFailure}. The final position was restored.`
      }
    } catch (recoveryError) {
      return {
        status: 'reconciled',
        warning: `Board animation failed: ${animationFailure}. Recovery failed: ${errorMessage(recoveryError)}.`
      }
    }
  }
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}
