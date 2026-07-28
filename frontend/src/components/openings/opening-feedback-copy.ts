import type {
  OpeningDepth,
  OpeningMoveFeedbackDetail,
  OpeningPerspective
} from '../../lib/api'
import { formatOpeningEvaluation } from './opening-evaluation'

export type OpeningFeedbackCopy = {
  heading: string
  explanation: string
  contrast: string
  announcement: string
}

export function openingFeedbackCopy(
  detail: OpeningMoveFeedbackDetail,
  orientation: OpeningPerspective
): OpeningFeedbackCopy {
  const variationName = detail.variationName?.trim()
  const heading = variationName
    ? `${detail.playedSan} — ${variationName}`
    : detail.playedSan

  const explanationParts = [coverageSentence(detail)]
  if (detail.coverage !== 'unknown') {
    const evaluation = formatOpeningEvaluation(detail.evaluation, orientation)
    if (evaluation) explanationParts.push(asSentence(evaluation.text))
    if (detail.noteText?.trim()) explanationParts.push(asSentence(detail.noteText.trim()))
  }
  const explanation = explanationParts.filter(Boolean).join(' ')

  const plan = detail.planText?.trim()
  const contrast = plan
    ? `This lesson is practising ${detail.expectedSan}: ${asSentence(plan)}`
    : `This lesson is practising ${detail.expectedSan}.`
  const announcement = [asSentence(heading), explanation, contrast].filter(Boolean).join(' ')

  return { heading, explanation, contrast, announcement }
}

function coverageSentence(detail: OpeningMoveFeedbackDetail): string {
  switch (detail.coverage) {
    case 'visible':
      return 'This is a course-known idea.'
    case 'deeper':
      return `Covered later at ${depthLabel(detail.requiredDepth)} depth.`
    case 'unknown':
      return 'This move is not covered from this position.'
    default: {
      const exhaustive: never = detail.coverage
      return exhaustive
    }
  }
}

function depthLabel(depth: OpeningDepth | undefined): string {
  switch (depth) {
    case 'quick': return 'Quick'
    case 'standard': return 'Standard'
    case 'reference': return 'Reference'
    case undefined: return 'a deeper'
    default: {
      const exhaustive: never = depth
      return exhaustive
    }
  }
}

function asSentence(value: string): string {
  const trimmed = value.trim()
  if (!trimmed || /[.!?]$/.test(trimmed)) return trimmed
  return `${trimmed}.`
}
