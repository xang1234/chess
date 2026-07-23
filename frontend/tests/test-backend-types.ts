export type PuzzleDefinition = {
  fingerprint: string
  fen: string
  solver: 'white' | 'black'
  legalMoves: string[]
  canReveal?: boolean
  sourceFen?: string
  preludeUci?: string
  currentPath?: number[]
}

type AppliedMoveDefinition = { uci: string; resultingFen: string }

export type ContinueResponse = {
  kind: 'continue'
  uci: string
  appliedMoves: AppliedMoveDefinition[]
  continuation: PuzzleDefinition
}

export type NextResponse = {
  kind: 'next'
  uci: string
  appliedMoves: AppliedMoveDefinition[]
  finalFen: string
  next: PuzzleDefinition
}

export type SummaryResponse = {
  kind: 'summary'
  uci: string
  appliedMoves: AppliedMoveDefinition[]
  finalFen: string
}

export type ResponseDefinition = ContinueResponse | NextResponse | SummaryResponse

export type BoardScenario = {
  kind: 'board'
  first: PuzzleDefinition
  wrongMoves?: string[]
  correct?: ResponseDefinition
  reveal?: ResponseDefinition
}

export type TrainerScenario = { kind: 'trainer' }
export type OpeningScenario = { kind: 'openings'; deepExplorerLength?: number }
export type TestBackendScenario = BoardScenario | TrainerScenario | OpeningScenario

export type TrustedInput = { type: string; trusted: boolean }

export type TestBackendState = {
  moves: string[]
  reveals: number
  trustedInputs: TrustedInput[]
  semanticFrames: string[][]
  openingMoves: string[]
  openingHints: number[]
  openingDepths: string[]
  holdImportOpen(): void
  selectedImportPath(): string
  selectedCoursePath(): string
  reportImportProgress(rowsRead: number, bytesRead: number): void
}

type WireValue<Value> =
  Value extends Promise<infer Resolved> ? Promise<WireValue<Resolved>>
    : Value extends readonly (infer Entry)[] ? WireValue<Entry>[]
      : Value extends object ? {
        [Key in keyof Value as Value[Key] extends (...arguments_: never[]) => unknown
          ? never
          : Key]: WireValue<Value[Key]>
      }
        : Value

type BindingMock<Bindings> = {
  [Key in keyof Bindings]: Bindings[Key] extends (...args: infer Arguments) => infer Result
    ? (...args: Arguments) => WireValue<Result>
    : never
}

export type ModeControllerMock = BindingMock<typeof import('../wailsjs/go/main/ModeController')>
type NormalBindings = typeof import('../wailsjs/go/main/NormalController')
export type NormalControllerMock = Omit<
  BindingMock<NormalBindings>,
  'GetProfile' | 'ResumeSession' | 'ResumeOpeningSession'
> & {
  GetProfile: (...args: Parameters<NormalBindings['GetProfile']>) =>
    Promise<WireValue<Awaited<ReturnType<NormalBindings['GetProfile']>>> | null>
  ResumeSession: (...args: Parameters<NormalBindings['ResumeSession']>) =>
    Promise<WireValue<Awaited<ReturnType<NormalBindings['ResumeSession']>>> | null>
  ResumeOpeningSession: (...args: Parameters<NormalBindings['ResumeOpeningSession']>) =>
    Promise<WireValue<Awaited<ReturnType<NormalBindings['ResumeOpeningSession']>>> | null>
}
export type RecoveryControllerMock = BindingMock<typeof import('../wailsjs/go/main/RecoveryController')>
type RuntimeBindings = typeof import('../wailsjs/runtime/runtime')
export type WireSession = Exclude<Awaited<ReturnType<NormalControllerMock['StartGuided']>>, null>
export type WirePuzzle = NonNullable<WireSession['current']>
export type WireImportResult = Awaited<ReturnType<NormalControllerMock['GetImportResult']>>
export type WireOpeningSession = Awaited<ReturnType<NormalControllerMock['StartOpeningLesson']>>
export type WireOpeningActivity = NonNullable<WireOpeningSession['current']>

export type TestWindow = Window & {
  __testBackend: TestBackendState
  runtime: Pick<RuntimeBindings, 'EventsOnMultiple' | 'EventsOff' | 'EventsOffAll'>
  go: {
    main: {
      ModeController: ModeControllerMock
      NormalController: NormalControllerMock
      RecoveryController: RecoveryControllerMock
    }
  }
}
