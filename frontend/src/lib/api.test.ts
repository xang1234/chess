import {
  loadApplicationAPI,
  type NormalAPI,
  type RecoveryAPI
} from './api'

type HasKey<Value, Key extends PropertyKey> = Key extends keyof Value ? true : false
type APIModule = typeof import('./api')

const recoveryHasStartGuided: HasKey<RecoveryAPI, 'startGuided'> = false
const normalHasRecoveryState: HasKey<NormalAPI, 'getRecoveryState'> = false
const moduleHasNormalGetter: HasKey<APIModule, 'getAPI'> = false
const moduleHasRecoveryGetter: HasKey<APIModule, 'getRecoveryAPI'> = false

test('API module exposes only the mode-discriminated bootstrap', async () => {
  expect(recoveryHasStartGuided).toBe(false)
  expect(normalHasRecoveryState).toBe(false)
  expect(moduleHasNormalGetter).toBe(false)
  expect(moduleHasRecoveryGetter).toBe(false)
  expect(loadApplicationAPI).toBeTypeOf('function')
})
