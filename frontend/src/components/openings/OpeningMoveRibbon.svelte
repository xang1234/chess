<script lang="ts">
  import { afterUpdate } from 'svelte'

  export let variationName: string | undefined = undefined
  export let moves: readonly string[] = []

  let plyElements: HTMLLIElement[] = []

  afterUpdate(() => {
    plyElements.at(-1)?.scrollIntoView?.({ block: 'nearest', inline: 'end' })
  })

  const prefix = (index: number): string =>
    index % 2 === 0 ? `${Math.floor(index / 2) + 1}.` : ''
</script>

<section class="opening-move-ribbon" aria-label="Opening line">
  {#if variationName}
    <span class="variation-badge">{variationName}</span>
  {/if}
  {#if moves.length === 0}
    <span class="starting-position">Starting position</span>
  {:else}
    <ol aria-label="Moves to this position">
      {#each moves as move, index}
        <li
          bind:this={plyElements[index]}
          aria-current={index === moves.length - 1 ? 'step' : undefined}
        >
          {#if prefix(index)}
            <span class="move-number">{prefix(index)}</span>
          {/if}
          <span>{move}</span>
        </li>
      {/each}
    </ol>
  {/if}
</section>

<style>
  .opening-move-ribbon {
    display: flex;
    width: 100%;
    max-height: 52px;
    min-height: 44px;
    padding: 7px 10px;
    overflow-x: auto;
    align-items: center;
    gap: 10px;
    border: 1px solid #3b4843;
    border-radius: 12px;
    color: #f7f4e9;
    background: var(--charcoal-800);
    scrollbar-width: thin;
  }
  .variation-badge {
    position: sticky;
    left: 0;
    z-index: 1;
    flex: 0 0 auto;
    padding: 5px 8px;
    border-radius: 999px;
    color: var(--charcoal-950);
    background: var(--amber-400);
    font-size: 0.78rem;
    font-weight: 900;
  }
  .starting-position {
    color: #e8e4d8;
    font-size: 0.9rem;
    font-weight: 800;
  }
  ol {
    display: flex;
    min-width: max-content;
    margin: 0;
    padding: 0;
    align-items: center;
    gap: 8px;
    list-style: none;
  }
  li {
    display: inline-flex;
    padding: 4px 6px;
    align-items: baseline;
    gap: 4px;
    border-radius: 7px;
    font-weight: 800;
  }
  li[aria-current='step'] {
    color: var(--charcoal-950);
    background: #f4d27b;
  }
  .move-number {
    color: #a9d2b6;
    font-size: 0.75rem;
  }
  li[aria-current='step'] .move-number { color: var(--forest-700); }
</style>
