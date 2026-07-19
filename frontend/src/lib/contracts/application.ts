import {
  array,
  boolean,
  enumeration,
  number,
  optionalString,
  record,
  string
} from './decoder'
import type { SessionMode } from './puzzles'

export type Profile = {
  learnerRating: number
  sessionSize: 5 | 10 | 15
}

export type PracticeSource = {
  id: string
  kind: string
  minimumRating: number
  maximumRating: number
  hasRatingRange: boolean
  maximumPlies: number
}

export type RatingBounds = { minimum: number; maximum: number }

export type PracticeFilters = {
  sources: PracticeSource[]
  themes: string[]
  maximumSolutionPlies: number
  learnerRatingBounds: RatingBounds
}

export type RatingPoint = { rating: number; recordedAt: number }
export type ThemePerformance = { theme: string; attempts: number; accuracy: number }
export type RecentSession = {
  sessionId: string
  mode: SessionMode
  status: 'active' | 'paused' | 'completed'
  updatedAt: number
  total: number
  completed: number
  firstTry: number
  usedHint: number
  revealed: number
}

export type ParentSummary = {
  learnerRating: number
  ratingTrend: RatingPoint[]
  firstAttemptAccuracy: number
  hintRate: number
  themePerformance: ThemePerformance[]
  dueReviews: number
  recentSessions: RecentSession[]
}

export type RecoveryState = { required: boolean; path?: string; detail?: string }
export type ApplicationMode = 'normal' | 'recovery'
export type BuildInfo = { name: string; commit: string; sourceUrl: string }

export function decodeApplicationMode(value: unknown): ApplicationMode {
  return enumeration(value, ['normal', 'recovery'], 'application mode')
}

export function decodeBuildInfo(value: unknown): BuildInfo {
  const raw = record(value, 'build info')
  return {
    name: string(raw.name, 'build info.name'),
    commit: string(raw.commit, 'build info.commit'),
    sourceUrl: string(raw.sourceUrl, 'build info.sourceUrl')
  }
}

export function decodeProfile(value: unknown): Profile | null {
  if (value === null || value === undefined) return null
  const raw = record(value, 'profile')
  const sessionSize = number(raw.sessionSize, 'profile.sessionSize')
  if (sessionSize !== 5 && sessionSize !== 10 && sessionSize !== 15) {
    throw new Error(`profile.sessionSize has unsupported value ${sessionSize}`)
  }
  return { learnerRating: number(raw.learnerRating, 'profile.learnerRating'), sessionSize }
}

export function decodePracticeFilters(value: unknown): PracticeFilters {
  const raw = record(value, 'practice filters')
  return {
    sources: array(raw.sources, 'practice filters.sources', (entry, path) => {
      const source = record(entry, path)
      return {
        id: string(source.id, `${path}.id`),
        kind: string(source.kind, `${path}.kind`),
        minimumRating: number(source.minimumRating, `${path}.minimumRating`),
        maximumRating: number(source.maximumRating, `${path}.maximumRating`),
        hasRatingRange: boolean(source.hasRatingRange, `${path}.hasRatingRange`),
        maximumPlies: number(source.maximumPlies, `${path}.maximumPlies`)
      }
    }),
    themes: array(raw.themes, 'practice filters.themes', string),
    maximumSolutionPlies: number(raw.maximumSolutionPlies, 'practice filters.maximumSolutionPlies'),
    learnerRatingBounds: (() => {
      const bounds = record(raw.learnerRatingBounds, 'practice filters.learnerRatingBounds')
      return {
        minimum: number(bounds.minimum, 'practice filters.learnerRatingBounds.minimum'),
        maximum: number(bounds.maximum, 'practice filters.learnerRatingBounds.maximum')
      }
    })()
  }
}

export function decodeParentSummary(value: unknown): ParentSummary {
  const raw = record(value, 'parent summary')
  return {
    learnerRating: number(raw.learnerRating, 'parent summary.learnerRating'),
    ratingTrend: array(raw.ratingTrend, 'parent summary.ratingTrend', (entry, path) => {
      const point = record(entry, path)
      return {
        rating: number(point.rating, `${path}.rating`),
        recordedAt: number(point.recordedAt, `${path}.recordedAt`)
      }
    }),
    firstAttemptAccuracy: number(raw.firstAttemptAccuracy, 'parent summary.firstAttemptAccuracy'),
    hintRate: number(raw.hintRate, 'parent summary.hintRate'),
    themePerformance: array(raw.themePerformance, 'parent summary.themePerformance', (entry, path) => {
      const performance = record(entry, path)
      return {
        theme: string(performance.theme, `${path}.theme`),
        attempts: number(performance.attempts, `${path}.attempts`),
        accuracy: number(performance.accuracy, `${path}.accuracy`)
      }
    }),
    dueReviews: number(raw.dueReviews, 'parent summary.dueReviews'),
    recentSessions: array(raw.recentSessions, 'parent summary.recentSessions', (entry, path) => {
      const session = record(entry, path)
      return {
        sessionId: string(session.sessionId, `${path}.sessionId`),
        mode: enumeration(session.mode, ['guided', 'practice'], `${path}.mode`),
        status: enumeration(session.status, ['active', 'paused', 'completed'], `${path}.status`),
        updatedAt: number(session.updatedAt, `${path}.updatedAt`),
        total: number(session.total, `${path}.total`),
        completed: number(session.completed, `${path}.completed`),
        firstTry: number(session.firstTry, `${path}.firstTry`),
        usedHint: number(session.usedHint, `${path}.usedHint`),
        revealed: number(session.revealed, `${path}.revealed`)
      }
    })
  }
}

export function decodeRecoveryState(value: unknown): RecoveryState {
  const raw = record(value, 'recovery state')
  return {
    required: boolean(raw.required, 'recovery state.required'),
    path: optionalString(raw.path, 'recovery state.path'),
    detail: optionalString(raw.detail, 'recovery state.detail')
  }
}
