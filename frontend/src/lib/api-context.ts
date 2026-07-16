import { getContext, setContext } from 'svelte'
import type { NormalAPI, RecoveryAPI } from './api'

const normalAPIKey = Symbol('normal-api')
const recoveryAPIKey = Symbol('recovery-api')

export function provideNormalAPI(api: NormalAPI): void {
  setContext(normalAPIKey, api)
}

export function useNormalAPI(): NormalAPI {
  const api = getContext<NormalAPI | undefined>(normalAPIKey)
  if (!api) throw new Error('Normal application API is unavailable')
  return api
}

export function provideRecoveryAPI(api: RecoveryAPI): void {
  setContext(recoveryAPIKey, api)
}

export function useRecoveryAPI(): RecoveryAPI {
  const api = getContext<RecoveryAPI | undefined>(recoveryAPIKey)
  if (!api) throw new Error('Recovery application API is unavailable')
  return api
}
