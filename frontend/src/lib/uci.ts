export type Square = `${'a' | 'b' | 'c' | 'd' | 'e' | 'f' | 'g' | 'h'}${1 | 2 | 3 | 4 | 5 | 6 | 7 | 8}`
export type Promotion = 'q' | 'r' | 'b' | 'n'
export type ParsedUCI = { from: Square; to: Square; promotion?: Promotion }

const uciPattern = /^([a-h][1-8])([a-h][1-8])([qrbn])?$/

export function parseUCI(value: string): ParsedUCI {
  const matched = uciPattern.exec(value)
  if (!matched) {
    throw new Error(`invalid UCI move ${JSON.stringify(value)}`)
  }
  const parsed: ParsedUCI = {
    from: matched[1] as Square,
    to: matched[2] as Square
  }
  if (matched[3]) parsed.promotion = matched[3] as Promotion
  return parsed
}

export function groupLegalMoves(values: string[]): Map<Square, Square[]> {
  const destinations = new Map<Square, Set<Square>>()
  for (const value of values) {
    const { from, to } = parseUCI(value)
    const grouped = destinations.get(from) ?? new Set<Square>()
    grouped.add(to)
    destinations.set(from, grouped)
  }

  return new Map(
    [...destinations.entries()]
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([from, to]) => [from, [...to].sort()] as const)
  )
}

export function promotionChoices(
  values: string[],
  from: Square,
  to: Square
): Promotion[] {
  const choices = new Set<Promotion>()
  for (const value of values) {
    const parsed = parseUCI(value)
    if (parsed.from === from && parsed.to === to && parsed.promotion) {
      choices.add(parsed.promotion)
    }
  }
  return [...choices].sort()
}

export function moveSquares(value: string): [Square, Square] {
  const parsed = parseUCI(value)
  return [parsed.from, parsed.to]
}
