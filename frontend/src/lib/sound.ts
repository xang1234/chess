import moveUrl from '../assets/sounds/move.wav?url'
import captureUrl from '../assets/sounds/capture.wav?url'
import correctUrl from '../assets/sounds/correct.wav?url'
import incorrectUrl from '../assets/sounds/incorrect.wav?url'

export type SoundName = 'move' | 'capture' | 'correct' | 'incorrect'

export interface SoundBackend {
  unlock(): Promise<void>
  play(url: string, volume: number): void
  destroy(): void
}

export interface SoundService {
  readonly muted: boolean
  unlock(): Promise<void>
  play(name: SoundName): void
  setMuted(muted: boolean): void
  toggleMuted(): boolean
  destroy(): void
}

type SoundStorage = Pick<Storage, 'getItem' | 'setItem'>

export type SoundServiceOptions = {
  backend?: SoundBackend
  storage?: SoundStorage | null
  volume?: number
}

const storageKey = 'chess-trainer:sound-muted:v1'
const defaultVolume = 0.22
const soundUrls: Record<SoundName, string> = {
  move: moveUrl,
  capture: captureUrl,
  correct: correctUrl,
  incorrect: incorrectUrl
}

export function createSoundService(options: SoundServiceOptions = {}): SoundService {
  const audio = options.backend ?? createWebAudioBackend(Object.values(soundUrls))
  const storage = options.storage === undefined ? browserStorage() : options.storage ?? undefined
  const volume = normaliseVolume(options.volume)
  let muted = readMuted(storage)
  let destroyed = false

  return {
    get muted() {
      return muted
    },
    unlock: async () => {
      if (destroyed) return
      try {
        await audio.unlock()
      } catch {
        // Unlock may be denied until another user gesture; puzzle state continues.
      }
    },
    play: (name) => {
      if (destroyed || muted) return
      try {
        audio.play(soundUrls[name], volume)
      } catch {
        // Sound feedback is optional and must never interrupt puzzle state.
      }
    },
    setMuted: (nextMuted) => {
      muted = nextMuted
      persistMuted(storage, muted)
    },
    toggleMuted: () => {
      muted = !muted
      persistMuted(storage, muted)
      return muted
    },
    destroy: () => {
      if (destroyed) return
      destroyed = true
      try {
        audio.destroy()
      } catch {
        // The audio backend owns no required application state.
      }
    }
  }
}

function browserStorage(): SoundStorage | undefined {
  try {
    return typeof window === 'undefined' ? undefined : window.localStorage
  } catch {
    return undefined
  }
}

function readMuted(storage: SoundStorage | undefined): boolean {
  try {
    return storage?.getItem(storageKey) === 'true'
  } catch {
    return false
  }
}

function persistMuted(storage: SoundStorage | undefined, muted: boolean): void {
  try {
    storage?.setItem(storageKey, String(muted))
  } catch {
    // Private browsing and managed devices may deny storage access.
  }
}

function normaliseVolume(volume: number | undefined): number {
  if (volume === undefined || !Number.isFinite(volume)) return defaultVolume
  return Math.max(0, Math.min(1, volume))
}

function createWebAudioBackend(urls: readonly string[]): SoundBackend {
  let context: AudioContext | undefined
  let destroyed = false
  const buffers = new Map<string, Promise<AudioBuffer>>()

  function audioContext(): AudioContext {
    if (!context) context = new AudioContext()
    return context
  }

  function load(url: string, activeContext: AudioContext): Promise<AudioBuffer> {
    const existing = buffers.get(url)
    if (existing) return existing
    const loading = fetch(url)
      .then((response) => {
        if (!response.ok) throw new Error(`sound request failed with ${response.status}`)
        return response.arrayBuffer()
      })
      .then((bytes) => activeContext.decodeAudioData(bytes))
    buffers.set(url, loading)
    void loading.catch(() => buffers.delete(url))
    return loading
  }

  async function unlock(): Promise<void> {
    if (destroyed) return
    const activeContext = audioContext()
    const resumed = activeContext.state === 'suspended'
      ? activeContext.resume()
      : Promise.resolve()
    const loading = Promise.all(urls.map((url) => load(url, activeContext)))
    await Promise.all([resumed, loading])
  }

  async function play(url: string, volume: number): Promise<void> {
    if (destroyed) return
    const activeContext = audioContext()
    const resumed = activeContext.state === 'suspended'
      ? activeContext.resume()
      : Promise.resolve()
    const buffer = load(url, activeContext)
    await resumed
    const decoded = await buffer
    if (destroyed || activeContext.state === 'closed') return
    const source = activeContext.createBufferSource()
    const gain = activeContext.createGain()
    source.buffer = decoded
    gain.gain.value = volume
    source.connect(gain)
    gain.connect(activeContext.destination)
    source.start()
  }

  return {
    unlock,
    play: (url, volume) => {
      void play(url, volume).catch(() => undefined)
    },
    destroy: () => {
      if (destroyed) return
      destroyed = true
      buffers.clear()
      const activeContext = context
      context = undefined
      if (!activeContext) return
      try {
        void activeContext.close().catch(() => undefined)
      } catch {
        // Closing audio is best-effort during component teardown.
      }
    }
  }
}
