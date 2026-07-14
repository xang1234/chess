<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import { orientSquares, parseFEN, type Piece } from '../../lib/fen'

  export let fen: string
  export let orientation: 'white' | 'black' = 'white'
  export let disabled = false
  export let lastMove: string[] = []
  export let wrongMove: string[] = []
  export let hintSource = ''
  export let hintTarget = ''

  const dispatch = createEventDispatcher<{ move: { uci: string } }>()
  const glyphs: Record<Piece['color'], Record<Piece['role'], string>> = {
    white: { pawn: '♙', knight: '♘', bishop: '♗', rook: '♖', queen: '♕', king: '♔' },
    black: { pawn: '♟', knight: '♞', bishop: '♝', rook: '♜', queen: '♛', king: '♚' }
  }

  let selected = ''
  let promotion: { from: string; to: string } | null = null
  $: board = parseFEN(fen)
  $: squares = orientSquares(orientation)

  function accessibleName(square: string, piece: Piece | undefined): string {
    if (!piece) return `Empty ${square}`
    return `${piece.color === 'white' ? 'White' : 'Black'} ${piece.role} on ${square}`
  }

  function choose(square: string): void {
    if (disabled) return
    if (!selected) {
      if (board[square]) selected = square
      return
    }
    if (selected === square) {
      selected = ''
      return
    }
    requestMove(selected, square)
  }

  function requestMove(from: string, to: string): void {
    const piece = board[from]
    if (!piece) return
    const promotionRank = piece.color === 'white' ? '8' : '1'
    if (piece.role === 'pawn' && to[1] === promotionRank) {
      promotion = { from, to }
      return
    }
    dispatch('move', { uci: from + to })
    selected = ''
  }

  function promote(piece: 'q' | 'r' | 'b' | 'n'): void {
    if (!promotion) return
    dispatch('move', { uci: promotion.from + promotion.to + piece })
    selected = ''
    promotion = null
  }

  function startDrag(square: string): void {
    if (!disabled && board[square]) selected = square
  }

  function drop(square: string): void {
    if (!disabled && selected) requestMove(selected, square)
  }

  function isLight(square: string): boolean {
    return (square.charCodeAt(0) - 97 + Number(square[1])) % 2 === 1
  }
</script>

<div class="board-wrap">
  <div class="chess-board" role="grid" aria-label={`Chess board, ${orientation} side`}>
    {#each squares as square}
      <button
        type="button"
        role="gridcell"
        data-square={square}
        aria-label={accessibleName(square, board[square])}
        aria-selected={selected === square}
        disabled={disabled}
        draggable={!disabled && Boolean(board[square])}
        class:light={isLight(square)}
        class:dark={!isLight(square)}
        class:selected={selected === square}
        class:last={lastMove.includes(square)}
        class:wrong={wrongMove.includes(square)}
        class:hint-source={hintSource === square}
        class:hint-target={hintTarget === square}
        on:click={() => choose(square)}
        on:dragstart={() => startDrag(square)}
        on:dragover|preventDefault
        on:drop={() => drop(square)}
      >
        {#if board[square]}
          <span class="piece" aria-hidden="true">{glyphs[board[square].color][board[square].role]}</span>
        {/if}
        <span class="coordinate" aria-hidden="true">{square}</span>
      </button>
    {/each}
  </div>

  {#if promotion}
    <div class="promotion-backdrop" role="presentation">
      <section class="promotion-dialog" role="dialog" aria-modal="true" aria-labelledby="promotion-title">
        <h3 id="promotion-title">Choose a promotion</h3>
        <div class="promotion-pieces">
          <button type="button" aria-label="Promote to queen" on:click={() => promote('q')}>♕</button>
          <button type="button" aria-label="Promote to rook" on:click={() => promote('r')}>♖</button>
          <button type="button" aria-label="Promote to bishop" on:click={() => promote('b')}>♗</button>
          <button type="button" aria-label="Promote to knight" on:click={() => promote('n')}>♘</button>
        </div>
      </section>
    </div>
  {/if}
</div>

<style>
  .board-wrap { position: relative; width: min(72vh, 680px, 100%); aspect-ratio: 1; }
  .chess-board {
    display: grid;
    width: 100%;
    height: 100%;
    grid-template-columns: repeat(8, 1fr);
    overflow: hidden;
    border: 8px solid #5a4937;
    border-radius: 14px;
    box-shadow: 0 20px 48px rgba(48, 37, 26, 0.2);
  }
  [role='gridcell'] {
    position: relative;
    display: grid;
    min-width: 0;
    min-height: 0;
    padding: 0;
    place-items: center;
    border: 0;
    border-radius: 0;
    color: #17231e;
  }
  [role='gridcell']:disabled { opacity: 1; cursor: default; }
  .light { background: #e9d9b5; }
  .dark { background: #72906f; }
  .piece {
    position: relative;
    z-index: 2;
    font-family: 'Arial Unicode MS', 'Noto Sans Symbols 2', 'Apple Symbols', sans-serif;
    font-size: clamp(2rem, 7.2vmin, 4.6rem);
    line-height: 1;
    filter: drop-shadow(0 2px 1px rgba(255, 255, 255, 0.25));
  }
  .coordinate { position: absolute; right: 3px; bottom: 1px; z-index: 3; font-size: clamp(0.45rem, 1.2vmin, 0.7rem); font-weight: 800; opacity: 0.7; }
  .selected::after, .hint-source::after, .hint-target::after, .wrong::after, .last::after {
    position: absolute;
    z-index: 1;
    width: 64%;
    height: 64%;
    border-radius: 50%;
    content: '';
  }
  .selected::after { border: 4px solid #efb444; }
  .last::after { background: rgba(239, 180, 68, 0.42); }
  .wrong::after { border: 4px solid #b23f34; background: rgba(178, 63, 52, 0.22); }
  .hint-source::after { border: 4px solid #f4d276; }
  .hint-target::after { background: rgba(244, 210, 118, 0.6); }
  .promotion-backdrop { position: absolute; inset: 0; z-index: 10; display: grid; place-items: center; background: rgba(28, 45, 38, 0.45); }
  .promotion-dialog { padding: 18px; border-radius: 16px; background: #fffdf7; box-shadow: 0 18px 44px rgba(0, 0, 0, 0.25); }
  .promotion-dialog h3 { margin: 0 0 12px; }
  .promotion-pieces { display: flex; gap: 8px; }
  .promotion-pieces button { width: 56px; border: 1px solid #eadfc7; border-radius: 10px; background: white; font-size: 2rem; }
</style>
