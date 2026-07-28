const deepInitialFen = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1'

export function buildDeepExplorerPositions(length: number) {
  const buildPosition = (positionId: string, depth: number, sideBranch: boolean) => {
    const terminal = sideBranch || depth >= length
    const branchCount = depth >= 5 ? 7 : 1
    return {
      courseId: 'synthetic-italian', positionId,
      fen: depth % 2 === 0
        ? deepInitialFen
        : 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1',
      label: depth === 0 ? 'Initial position' : `Deep explorer position ${depth}`,
      evaluation: { code: 'equal' },
      notes: Array.from({ length: depth >= 6 ? 10 : 1 }, (_value, index) => ({
        kind: 'overview',
        text: `Scrollable reference note ${index + 1} for explorer depth ${depth}.`,
        sourceRef: {
          printedPage: 1, noteLabel: `deep-${depth}-${index + 1}`,
          coverageId: `synthetic-deep-note-${depth}-${index + 1}`
        }
      })),
      moves: terminal ? [] : Array.from({ length: branchCount }, (_value, index) => ({
        moveId: `deep-${depth}-${index}`, uci: depth % 2 === 0 ? 'e2e4' : 'e7e5',
        san: index === 0 ? `M${depth + 1}` : `S${depth + 1}.${index}`,
        toPositionId: index === 0
          ? `deep-position-${depth + 1}`
          : `deep-side-${depth + 1}-${index}`,
        role: 'repertoire', variationName: index === 0 ? 'Main continuation' : 'Side branch',
        evaluation: { code: 'equal' },
        sourceRef: { printedPage: 1, coverageId: `synthetic-deep-move-${depth}-${index}` }
      })),
      incomingPaths: depth === 0 ? 0 : 1
    }
  }
  const positions: Record<string, ReturnType<typeof buildPosition>> = {}
  for (let depth = 0; depth <= length; depth++) {
    const positionId = depth === 0 ? 'initial' : `deep-position-${depth}`
    positions[positionId] = buildPosition(positionId, depth, false)
    if (depth < length && depth >= 5) {
      for (let index = 1; index < 7; index++) {
        const sideId = `deep-side-${depth + 1}-${index}`
        positions[sideId] = buildPosition(sideId, depth + 1, true)
      }
    }
  }
  return positions
}

export const openingBackendFixtures = {
  initialFen: 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1',
  promptFen: 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4',
  afterC3Fen: 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/2P2N2/PP1P1PPP/RNBQK2R b KQkq - 0 4',
  watchFrames: [
    {
      uci: 'e2e4',
      resultingFen: 'rnbqkbnr/pppppppp/8/8/4P3/8/PPPP1PPP/RNBQKBNR b KQkq e3 0 1'
    },
    {
      uci: 'e7e5',
      resultingFen: 'rnbqkbnr/pppp1ppp/8/4p3/4P3/8/PPPP1PPP/RNBQKBNR w KQkq e6 0 2'
    },
    {
      uci: 'g1f3',
      resultingFen: 'rnbqkbnr/pppp1ppp/8/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R b KQkq - 1 2'
    },
    {
      uci: 'b8c6',
      resultingFen: 'r1bqkbnr/pppp1ppp/2n5/4p3/4P3/5N2/PPPP1PPP/RNBQKB1R w KQkq - 2 3'
    },
    {
      uci: 'f1c4',
      resultingFen: 'r1bqkbnr/pppp1ppp/2n5/4p3/2B1P3/5N2/PPPP1PPP/RNBQK2R b KQkq - 3 3'
    },
    {
      uci: 'f8c5',
      resultingFen: 'r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4'
    }
  ],
  comparisonLines: [
    {
      label: 'Solid setup',
      moves: ['d6'],
      frames: [{
        uci: 'd7d6',
        resultingFen: 'r1bqk1nr/ppp2ppp/2np4/2b1p3/2B1P3/2P2N2/PP1P1PPP/RNBQK2R w KQkq - 0 5'
      }]
    },
    {
      label: 'Quick development',
      moves: ['Nf6'],
      frames: [{
        uci: 'g8f6',
        resultingFen: 'r1bqk2r/pppp1ppp/2n2n2/2b1p3/2B1P3/2P2N2/PP1P1PPP/RNBQK2R w KQkq - 1 5'
      }]
    }
  ],
  referenceSection: {
    activityId: 'giuoco-reference',
    title: 'Why c3 is flexible',
    instruction: 'Compare the immediate centre occupation with the quieter setup.',
    positionId: 'after-bc5',
    noteTexts: ['Reference analysis: c3 supports d4 while preserving the active bishop.'],
    annotations: [{ kind: 'arrow', from: 'c2', to: 'c3' }]
  },
  promptPositionContext: {
    positionId: 'after-bc5', label: 'Giuoco Piano',
    evaluation: { code: 'equal', sourceSymbol: '=' },
    planText: 'Prepare d4 while keeping the bishop active.',
    arrivedVia: [{
      moveId: 'black-bc5', san: 'Bc5', role: 'opponent',
      variationName: 'Giuoco Piano', evaluation: { code: 'equal' }, selected: true
    }, {
      moveId: 'black-nf6', san: 'Nf6', role: 'opponent',
      variationName: 'Two Knights', evaluation: { code: 'unclear' }, selected: false
    }],
    nextMoves: [{
      moveId: 'white-c3', san: 'c3', role: 'repertoire',
      variationName: 'Giuoco Piano', evaluation: { code: 'equal' }, selected: false
    }, {
      moveId: 'white-b4', san: 'b4', role: 'alternative',
      variationName: 'Evans idea', evaluation: { code: 'unclear' }, selected: false
    }]
  },
  afterC3PositionContext: {
    positionId: 'after-c3', label: 'Prepare d4', evaluation: { code: 'equal' },
    arrivedVia: [{
      moveId: 'white-c3', san: 'c3', role: 'repertoire',
      variationName: 'Giuoco Piano', evaluation: { code: 'equal' }, selected: true
    }],
    nextMoves: []
  },
  b4FeedbackDetail: {
    coverage: 'visible', playedSan: 'b4', variationName: 'Evans idea',
    evaluation: { code: 'unclear' },
    noteText: 'A sound alternative that keeps pressure on the centre.',
    expectedSan: 'c3',
    planText: 'Prepare d4 while keeping the bishop active.'
  },
  d3FeedbackDetail: {
    coverage: 'visible', playedSan: 'd3', variationName: 'Quiet Italian',
    evaluation: { code: 'equal' },
    noteText: 'Develop first and keep the centre compact.', expectedSan: 'c3',
    planText: 'Prepare d4 while keeping the bishop active.'
  },
  courseChooserPath: '/Users/family/Documents/synthetic-italian.ctcourse',
  courseInspection: {
    path: '/Users/family/Documents/synthetic-italian.ctcourse',
    filename: 'synthetic-italian.ctcourse',
    format: 'coursepack',
    formatLabel: 'Opening course',
    sourceId: 'synthetic-italian',
    sourceIdOrigin: 'embedded',
    sourceName: 'Synthetic Italian for White',
    replacesExisting: false
  } as const
}
