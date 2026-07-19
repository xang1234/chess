import { loadPreviewApplicationAPI } from './api/preview'
import { loadProductionApplicationAPI } from './api/production'
import type { ApplicationAPI } from './api/types'

export * from './api-contract'
export type {
  ApplicationAPI,
  BackupAPI,
  NormalAPI,
  PracticeRequest,
  RecoveryAPI
} from './api/types'

export function loadApplicationAPI(): Promise<ApplicationAPI> {
  return typeof window !== 'undefined' && window.go
    ? loadProductionApplicationAPI()
    : loadPreviewApplicationAPI()
}
