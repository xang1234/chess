import type { SoundName, SoundService } from '../../lib/sound'
import type { Square } from '../../lib/uci'
import type { BoardEffects } from './board-effects'
import type { AnimationPort, PositionFrame } from './move-animation'
import { RequestOwner, type RequestToken } from './request-owner'

export type InteractiveBoardPort = {
  setPosition(
    fen: string,
    lastMove?: [Square, Square],
    animate?: boolean
  ): void
}

export type InteractiveBoardHooks = {
  publishPosition(
    fen: string,
    lastMove: [Square, Square] | undefined,
    replaceBoard: boolean
  ): void
}

export class InteractiveBoardRuntime {
  private readonly requests = new RequestOwner()
  private readonly effects: BoardEffects
  private readonly hooks: InteractiveBoardHooks
  private board: InteractiveBoardPort | undefined
  private sound: SoundService | undefined
  private mounted = false
  private soundUnlockStarted = false
  private recoveringBoard = false
  private pendingBoardWarning = ''

  constructor(effects: BoardEffects, hooks: InteractiveBoardHooks) {
    this.effects = effects
    this.hooks = hooks
  }

  mount(): { reducedMotion: boolean; soundMuted: boolean } {
    if (!this.mounted) {
      this.mounted = true
      this.sound = this.effects.createSound()
    }
    return {
      reducedMotion: this.effects.prefersReducedMotion(),
      soundMuted: this.sound?.muted ?? false
    }
  }

  destroy(): void {
    this.mounted = false
    this.requests.cancel()
    this.sound?.destroy()
    this.sound = undefined
    this.board = undefined
    this.soundUnlockStarted = false
    this.recoveringBoard = false
    this.pendingBoardWarning = ''
  }

  attachBoard(board: InteractiveBoardPort | undefined): void {
    this.board = board
  }

  startRequest(): RequestToken {
    return this.requests.start()
  }

  cancelRequest(): void {
    this.requests.cancel()
  }

  isCurrent(token: RequestToken): boolean {
    return this.mounted && this.requests.isCurrent(token)
  }

  unlockFromPointer(): void {
    this.startSoundUnlock()
  }

  unlockFromKeyboard(key: string): void {
    if (key === 'Enter' || key === ' ') this.startSoundUnlock()
  }

  toggleSound(): boolean {
    return this.sound?.toggleMuted() ?? false
  }

  playSound(kind: SoundName): void {
    this.sound?.play(kind)
  }

  async reconcilePosition(
    fen: string,
    signal: AbortSignal,
    animate: boolean
  ): Promise<string> {
    try {
      this.hooks.publishPosition(fen, undefined, false)
      if (!this.board) throw new Error('Chess board is unavailable')
      this.board.setPosition(fen, undefined, animate)
      if (animate) await this.effects.delay(180, signal)
      return ''
    } catch (error) {
      if (signal.aborted) return ''
      this.recoverBoard(fen)
      return `Board reconciliation failed: ${errorMessage(error)}. The saved position was restored.`
    }
  }

  animationPort(): AnimationPort {
    return {
      setPosition: (frame: PositionFrame) => {
        this.hooks.publishPosition(frame.fen, frame.lastMove, false)
        if (!this.board) throw new Error('Chess board is unavailable')
        this.board.setPosition(frame.fen, frame.lastMove, frame.animate)
      },
      delay: this.effects.delay,
      recover: (finalFen: string) => this.recoverBoard(finalFen)
    }
  }

  noteBoardError(message: string): void {
    this.pendingBoardWarning = message
  }

  isRecovering(): boolean {
    return this.recoveringBoard
  }

  consumeWarnings(...messages: string[]): string {
    const combined = [...messages, this.pendingBoardWarning].filter(Boolean)
    this.pendingBoardWarning = ''
    this.recoveringBoard = false
    return [...new Set(combined)].join(' ')
  }

  private startSoundUnlock(): void {
    if (!this.sound || this.soundUnlockStarted) return
    this.soundUnlockStarted = true
    try {
      void this.sound.unlock().catch(() => { this.soundUnlockStarted = false })
    } catch {
      this.soundUnlockStarted = false
    }
  }

  private recoverBoard(finalFen: string): void {
    this.recoveringBoard = true
    this.hooks.publishPosition(finalFen, undefined, true)
  }
}

function errorMessage(error: unknown): string {
  return error instanceof Error ? error.message : String(error)
}
