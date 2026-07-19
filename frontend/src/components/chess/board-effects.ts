import { createSoundService, type SoundService } from '../../lib/sound'

export type BoardEffects = {
  createSound(): SoundService
  delay(milliseconds: number, signal: AbortSignal): Promise<void>
  prefersReducedMotion(): boolean
}

function browserDelay(milliseconds: number, signal: AbortSignal): Promise<void> {
  if (signal.aborted) return Promise.resolve()
  return new Promise((resolve) => {
    const timer = window.setTimeout(finish, milliseconds)
    signal.addEventListener('abort', finish, { once: true })

    function finish(): void {
      window.clearTimeout(timer)
      signal.removeEventListener('abort', finish)
      resolve()
    }
  })
}

export const browserBoardEffects: BoardEffects = {
  createSound: createSoundService,
  delay: browserDelay,
  prefersReducedMotion: () => typeof window !== 'undefined' &&
    typeof window.matchMedia === 'function' &&
    window.matchMedia('(prefers-reduced-motion: reduce)').matches
}
