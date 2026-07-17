import {
  createSoundService,
  type SoundBackend,
  type SoundName
} from './sound'

const storageKey = 'chess-trainer:sound-muted:v1'

function backend(overrides: Partial<SoundBackend> = {}): SoundBackend {
  return {
    unlock: vi.fn().mockResolvedValue(undefined),
    play: vi.fn(),
    destroy: vi.fn(),
    ...overrides
  }
}

function storage(initial: Record<string, string> = {}) {
  const values = new Map(Object.entries(initial))
  return {
    getItem: vi.fn((key: string) => values.get(key) ?? null),
    setItem: vi.fn((key: string, value: string) => { values.set(key, value) })
  }
}

test('starts unmuted by default and restores the persisted mute preference', () => {
  const defaultService = createSoundService({ backend: backend(), storage: storage() })
  const mutedService = createSoundService({
    backend: backend(),
    storage: storage({ [storageKey]: 'true' })
  })

  expect(defaultService.muted).toBe(false)
  expect(mutedService.muted).toBe(true)
})

test('persists explicit and toggled mute state', () => {
  const saved = storage()
  const service = createSoundService({ backend: backend(), storage: saved })

  service.setMuted(true)
  expect(service.muted).toBe(true)
  expect(saved.setItem).toHaveBeenLastCalledWith(storageKey, 'true')

  expect(service.toggleMuted()).toBe(false)
  expect(service.muted).toBe(false)
  expect(saved.setItem).toHaveBeenLastCalledWith(storageKey, 'false')
})

test.each<SoundName>(['move', 'capture', 'correct', 'incorrect'])(
  'selects the local %s cue and configured volume',
  (name) => {
    const audio = backend()
    const service = createSoundService({ backend: audio, storage: storage(), volume: 0.31 })

    service.play(name)

    expect(audio.play).toHaveBeenCalledTimes(1)
    expect(audio.play).toHaveBeenCalledWith(
      expect.stringMatching(new RegExp(`${name}\\.wav(?:\\?.*)?$`)),
      0.31
    )
  }
)

test('uses a restrained default gain', () => {
  const audio = backend()
  const service = createSoundService({ backend: audio, storage: storage() })

  service.play('move')

  expect(audio.play).toHaveBeenCalledWith(expect.any(String), 0.22)
})

test('suppresses playback while muted without losing textual service state', () => {
  const audio = backend()
  const service = createSoundService({ backend: audio, storage: storage() })

  service.setMuted(true)
  service.play('correct')

  expect(service.muted).toBe(true)
  expect(audio.play).not.toHaveBeenCalled()
})

test('begins backend unlock synchronously and contains asynchronous failure', async () => {
  const unlock = vi.fn().mockRejectedValue(new Error('audio unavailable'))
  const service = createSoundService({
    backend: backend({ unlock }),
    storage: storage()
  })

  const pending = service.unlock()
  expect(unlock).toHaveBeenCalledTimes(1)
  await expect(pending).resolves.toBeUndefined()
})

test('contains playback and storage failures', () => {
  const audio = backend({ play: vi.fn(() => { throw new Error('speaker failed') }) })
  const brokenStorage = {
    getItem: vi.fn(() => { throw new Error('storage denied') }),
    setItem: vi.fn(() => { throw new Error('storage denied') })
  }

  expect(() => {
    const service = createSoundService({ backend: audio, storage: brokenStorage })
    service.play('incorrect')
    service.setMuted(true)
  }).not.toThrow()
})

test('destroys its backend exactly once and ignores later work', async () => {
  const audio = backend()
  const service = createSoundService({ backend: audio, storage: storage() })

  service.destroy()
  service.destroy()
  service.play('move')
  await service.unlock()

  expect(audio.destroy).toHaveBeenCalledTimes(1)
  expect(audio.play).not.toHaveBeenCalled()
  expect(audio.unlock).not.toHaveBeenCalled()
})
