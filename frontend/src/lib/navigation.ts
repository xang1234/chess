import { writable } from 'svelte/store'

export type Screen = 'setup' | 'home' | 'import' | 'puzzle' | 'practice' | 'parent' | 'games' | 'recovery'

export const screen = writable<Screen>('setup')
