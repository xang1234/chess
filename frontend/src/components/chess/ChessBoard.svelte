<script context="module" lang="ts">
  let boardSequence = 0
</script>

<script lang="ts">
  import { createEventDispatcher, onDestroy, onMount } from 'svelte'
  import type { Key } from '@lichess-org/chessground/types'
  import {
    describeSquare,
    orientSquares,
    parseFEN,
    type BoardMap
  } from '../../lib/fen'
  import {
    groupLegalMoves,
    moveKeyboardCursor,
    parseUCI,
    promotionChoices,
    type Promotion,
    type Square
  } from '../../lib/uci'
  import PromotionDialog from './PromotionDialog.svelte'
  import {
    createChessgroundAdapter,
    type BoardAnnotation,
    type BoardInteraction,
    type ChessBoardAdapter,
    type ChessgroundAdapterFactory
  } from './chessground-adapter'

  export let fen: string
  export let orientation: 'white' | 'black' = 'white'
  export let legalMoves: string[] = []
  export let inputEnabled = true
  export let lastMove: [Square, Square] | undefined = undefined
  export let wrongMove: [Square, Square] | undefined = undefined
  export let hintSource: Square | undefined = undefined
  export let hintTarget: Square | undefined = undefined
  export let annotations: readonly BoardAnnotation[] = []
  export let reducedMotion = false
  export let adapterFactory: ChessgroundAdapterFactory = createChessgroundAdapter

  const dispatch = createEventDispatcher<{
    move: { uci: string }
    error: { message: string }
    announce: { message: string }
  }>()
  const boardId = `chess-board-${++boardSequence}`
  const instructionsId = `${boardId}-instructions`
  const arrows = ['ArrowUp', 'ArrowDown', 'ArrowLeft', 'ArrowRight'] as const

  type Arrow = typeof arrows[number]
  type LegalContract = {
    moves: string[]
    destinations: Map<Square, Square[]>
    error: string
  }
  type FenContract = { board: BoardMap; error: string }
  type PendingPromotion = {
    from: Square
    to: Square
    choices: Promotion[]
  }

  let host: HTMLElement
  let boardControl: HTMLElement
  let adapter: ChessBoardAdapter | undefined
  let mounted = false
  let startupError = ''
  let reportedBoardError = ''
  let selected: Square | undefined
  let promotion: PendingPromotion | null = null
  let contract = validateLegalMoves(legalMoves)
  let fenContract = validateFen(fen)
  let keyboardCursor: Square = firstCursor(contract, orientation)

  $: contract = validateLegalMoves(legalMoves)
  $: fenContract = validateFen(fen)
  $: board = fenContract.board
  $: semanticSquares = orientSquares(orientation) as Square[]
  $: semanticRows = Array.from(
    { length: 8 },
    (_, row) => semanticSquares.slice(row * 8, row * 8 + 8)
  )
  $: contractError = contract.error || fenContract.error
  $: boardError = contractError || startupError
  $: effectiveInputEnabled = inputEnabled && !boardError && !promotion
  $: interaction = {
    orientation,
    legalMoves: contract.error ? [] : contract.moves,
    inputEnabled: effectiveInputEnabled,
    lastMove,
    wrongMove,
    hintSource,
    hintTarget,
    annotations,
    keyboardCursor,
    reducedMotion
  } as BoardInteraction
  $: if (adapter) adapter.configure(interaction)
  $: if (mounted) reportBoardError(boardError)

  function validateLegalMoves(values: readonly string[]): LegalContract {
    try {
      const moves = [...values]
      const destinations = groupLegalMoves(moves)
      return { moves, destinations, error: '' }
    } catch (error) {
      return {
        moves: [],
        destinations: new Map(),
        error: `Invalid legal move data: ${errorMessage(error)}. Puzzle input is locked.`
      }
    }
  }

  function validateFen(value: string): FenContract {
    try {
      return { board: parseFEN(value), error: '' }
    } catch (error) {
      return {
        board: {},
        error: `Invalid board position: ${errorMessage(error)}. Puzzle input is locked.`
      }
    }
  }

  function firstCursor(value: LegalContract, color: 'white' | 'black'): Square {
    return value.destinations.keys().next().value ?? (color === 'white' ? 'a1' : 'h8')
  }

  function reportBoardError(message: string): void {
    if (!message) {
      reportedBoardError = ''
      return
    }
    if (message === reportedBoardError) return
    reportedBoardError = message
    selected = undefined
    promotion = null
    adapter?.selectSquare(null)
    dispatch('error', { message })
  }

  function handleSelect(key: Key): void {
    const square = toSquare(key)
    if (!square || !effectiveInputEnabled) {
      selected = undefined
      adapter?.selectSquare(null)
      return
    }
    if (selected === square) {
      clearSelection()
      return
    }
    if (selected && contract.destinations.get(selected)?.includes(square)) return
    if (!contract.destinations.has(square)) {
      selected = undefined
      adapter?.selectSquare(null)
      dispatch('announce', {
        message: `${describeSquare(square, board[square])} has no legal moves.`
      })
      return
    }
    selected = square
    keyboardCursor = square
    dispatch('announce', { message: `Selected ${describeSquare(square, board[square])}.` })
  }

  function handleRoute(fromKey: Key, toKey: Key): void {
    const from = toSquare(fromKey)
    const to = toSquare(toKey)
    if (!from || !to || !effectiveInputEnabled) {
      restoreRejectedRoute(`${fromKey}${toKey}`)
      return
    }
    const matches = contract.moves.filter((value) => {
      const parsed = parseUCI(value)
      return parsed.from === from && parsed.to === to
    })
    if (matches.length === 0) {
      restoreRejectedRoute(`${from}${to}`)
      return
    }

    const choices = promotionChoices(matches, from, to)
    if (choices.length > 0) {
      promotion = { from, to, choices }
      return
    }

    const uci = `${from}${to}`
    if (!matches.includes(uci)) {
      restoreRejectedRoute(uci)
      return
    }
    submitMove(uci, to)
  }

  function restoreRejectedRoute(uci: string): void {
    selected = undefined
    promotion = null
    adapter?.setPosition(fen, lastMove, false)
    adapter?.selectSquare(null)
    dispatch('error', {
      message: `Move ${uci} is not legal in the current position. The board was restored.`
    })
  }

  function submitMove(uci: string, destination: Square): void {
    selected = undefined
    promotion = null
    keyboardCursor = destination
    dispatch('move', { uci })
    dispatch('announce', { message: `Played ${uci}.` })
  }

  function choosePromotion(event: CustomEvent<{ promotion: Promotion }>): void {
    if (!promotion || !promotion.choices.includes(event.detail.promotion)) return
    const { from, to } = promotion
    submitMove(`${from}${to}${event.detail.promotion}`, to)
  }

  function cancelPromotion(): void {
    if (!promotion) return
    promotion = null
    selected = undefined
    adapter?.setPosition(fen, lastMove, false)
    adapter?.selectSquare(null)
    dispatch('announce', { message: 'Promotion cancelled. The board was restored.' })
  }

  function handleKeydown(event: KeyboardEvent): void {
    if (arrows.includes(event.key as Arrow)) {
      event.preventDefault()
      keyboardCursor = moveKeyboardCursor(keyboardCursor, event.key as Arrow, orientation)
      dispatch('announce', { message: describeSquare(keyboardCursor, board[keyboardCursor]) })
      return
    }
    if (event.key === 'Escape') {
      event.preventDefault()
      if (promotion) cancelPromotion()
      else clearSelection()
      return
    }
    if (event.key !== 'Enter' && event.key !== ' ') return
    event.preventDefault()
    activateKeyboardCursor()
  }

  function activateKeyboardCursor(): void {
    if (!effectiveInputEnabled) return
    if (!selected) {
      selectKeyboardSource(keyboardCursor)
      return
    }
    if (selected === keyboardCursor) {
      clearSelection()
      return
    }
    if (contract.destinations.get(selected)?.includes(keyboardCursor)) {
      adapter?.selectSquare(keyboardCursor)
      return
    }
    if (contract.destinations.has(keyboardCursor)) {
      selectKeyboardSource(keyboardCursor)
      return
    }
    clearSelection()
  }

  function selectKeyboardSource(square: Square): void {
    if (!contract.destinations.has(square)) {
      clearSelection()
      dispatch('announce', {
        message: `${describeSquare(square, board[square])} has no legal moves.`
      })
      return
    }
    selected = square
    adapter?.selectSquare(square)
    dispatch('announce', { message: `Selected ${describeSquare(square, board[square])}.` })
  }

  function clearSelection(): void {
    selected = undefined
    adapter?.selectSquare(null)
    dispatch('announce', { message: 'Board selection cleared.' })
  }

  function toSquare(key: Key): Square | undefined {
    return /^[a-h][1-8]$/.test(key) ? key as Square : undefined
  }

  function errorMessage(error: unknown): string {
    return error instanceof Error ? error.message : String(error)
  }

  function squareId(square: Square): string {
    return `${boardId}-square-${square}`
  }

  export function setPosition(
    nextFen: string,
    nextLastMove?: [Square, Square],
    animate = false
  ): void {
    fen = nextFen
    lastMove = nextLastMove
    selected = undefined
    promotion = null
    adapter?.selectSquare(null)
    adapter?.setPosition(nextFen, nextLastMove, animate)
  }

  onMount(() => {
    mounted = true
    try {
      adapter = adapterFactory(host, fen, interaction, {
        onRoute: handleRoute,
        onSelect: handleSelect
      })
    } catch (error) {
      const message = `Chess board could not start: ${errorMessage(error)}. Puzzle input is locked.`
      startupError = message
      reportBoardError(message)
      return
    }
    reportBoardError(boardError)
  })

  onDestroy(() => {
    mounted = false
    adapter?.destroy()
    adapter = undefined
  })
</script>

<div class="board-wrap">
  <p id={instructionsId} class="visually-hidden">
    Use the arrow keys to move around the board. Press Enter or Space to select a piece and its destination. Press Escape to cancel selection.
  </p>
  <div
    bind:this={boardControl}
    class="board-control"
    role="grid"
    tabindex="0"
    aria-label={`Chess board, ${orientation} side`}
    aria-describedby={instructionsId}
    aria-activedescendant={squareId(keyboardCursor)}
    aria-disabled={!effectiveInputEnabled}
    on:keydown={handleKeydown}
  >
    <div bind:this={host} class="chessground-host" aria-hidden="true"></div>
    <div class="semantic-grid">
      {#each semanticRows as row}
        <span role="row">
          {#each row as square}
            <span
              id={squareId(square)}
              role="gridcell"
              aria-label={describeSquare(square, board[square])}
              aria-selected={selected === square}
            ></span>
          {/each}
        </span>
      {/each}
    </div>
  </div>

  {#if boardError}
    <p class="visually-hidden" role="alert">{boardError}</p>
  {/if}

  {#if promotion}
    <PromotionDialog
      choices={promotion.choices}
      color={orientation}
      returnFocus={boardControl}
      on:choose={choosePromotion}
      on:cancel={cancelPromotion}
    />
  {/if}
</div>

<style>
  .board-wrap {
    position: relative;
    width: 100%;
    max-width: 100%;
    aspect-ratio: 1;
  }
  .board-control,
  .chessground-host {
    width: 100%;
    height: 100%;
  }
  .board-control {
    position: relative;
    border-radius: 6px;
    outline: none;
  }
  .board-control:focus-visible {
    outline: 4px solid #f0c14b;
    outline-offset: 4px;
  }
  .semantic-grid,
  .visually-hidden {
    position: absolute !important;
    width: 1px !important;
    height: 1px !important;
    padding: 0 !important;
    overflow: hidden !important;
    border: 0 !important;
    clip: rect(0 0 0 0) !important;
    clip-path: inset(50%) !important;
    white-space: nowrap !important;
  }
</style>
