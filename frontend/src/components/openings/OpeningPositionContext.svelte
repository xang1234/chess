<script lang="ts">
  import type { OpeningPerspective, OpeningPositionContext } from '../../lib/api'
  import { formatOpeningEvaluation } from './opening-evaluation'

  export let context: OpeningPositionContext
  export let orientation: OpeningPerspective

  $: positionName = context.label.trim() || 'Current position'
  $: evaluation = formatOpeningEvaluation(context.evaluation, orientation)
</script>

<section class="position-context" aria-label={`Position: ${positionName}`}>
  <header>
    <h3 title={positionName}>{positionName}</h3>
    {#if evaluation}
      <p class="position-evaluation" aria-label={evaluation.label}>{evaluation.text}</p>
    {/if}
  </header>

  {#if context.planText}
    <p class="position-plan">{context.planText}</p>
  {/if}

  <div class="position-branches">
    <section class="branch-group">
      <h4>You arrived via</h4>
      {#if context.arrivedVia.length > 0}
        <ul aria-label="You arrived via">
          {#each context.arrivedVia as move (move.moveId)}
            <li
              class:selected={move.selected}
              class={`role-${move.role}`}
              aria-current={move.selected ? 'true' : undefined}
            >
              <strong>{move.san}</strong>
              {#if move.variationName}<span title={move.variationName}>{move.variationName}</span>{/if}
              {#if move.selected}<em>Played</em>{/if}
            </li>
          {/each}
        </ul>
      {:else}
        <p class="branch-empty">Starting position</p>
      {/if}
    </section>

    <section class="branch-group">
      <h4>From here</h4>
      {#if context.nextMoves.length > 0}
        <ul aria-label="From here">
          {#each context.nextMoves as move (move.moveId)}
            <li class={`role-${move.role}`}>
              <strong>{move.san}</strong>
              {#if move.variationName}<span title={move.variationName}>{move.variationName}</span>{/if}
            </li>
          {/each}
        </ul>
      {:else}
        <p class="branch-empty">End of this course line</p>
      {/if}
    </section>
  </div>
</section>

<style>
  .position-context {
    box-sizing: border-box;
    width: min(100%, 760px);
    padding: 12px 14px;
    border: 1px solid #46544f;
    border-radius: 14px;
    color: #f7f4e9;
    background: #26312d;
    box-shadow: 0 8px 24px rgb(0 0 0 / 18%);
  }
  header { display: flex; min-width: 0; gap: 10px; align-items: baseline; flex-wrap: wrap; }
  h3 {
    max-width: 100%;
    margin: 0;
    overflow: hidden;
    font-size: 1rem;
    line-height: 1.2;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .position-evaluation { margin: 0; color: #f0c14b; font-size: 0.88rem; font-weight: 800; }
  .position-plan {
    display: -webkit-box;
    margin: 7px 0 0;
    overflow: hidden;
    color: #e8e4d8;
    font-size: 0.88rem;
    line-height: 1.35;
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
  }
  .position-branches { display: grid; margin-top: 10px; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
  .branch-group { min-width: 0; }
  h4 { margin: 0 0 5px; color: #b9d5c2; font-size: 0.74rem; letter-spacing: 0.04em; text-transform: uppercase; }
  ul {
    display: flex;
    max-width: 100%;
    margin: 0;
    padding: 0 0 3px;
    overflow-x: auto;
    gap: 6px;
    list-style: none;
    scrollbar-width: thin;
  }
  li {
    display: grid;
    min-width: max-content;
    padding: 5px 8px;
    border: 1px solid #586660;
    border-radius: 8px;
    background: #303c38;
    grid-template-columns: auto auto;
    column-gap: 6px;
  }
  li.selected { border-color: #f0c14b; box-shadow: inset 3px 0 0 #f0c14b; }
  li strong { font-size: 0.85rem; }
  li span {
    max-width: 130px;
    overflow: hidden;
    color: #d8d5ca;
    font-size: 0.76rem;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  li em {
    grid-column: 1 / -1;
    color: #f0c14b;
    font-size: 0.68rem;
    font-style: normal;
    font-weight: 900;
    letter-spacing: 0.05em;
    text-transform: uppercase;
  }
  .branch-empty { margin: 0; color: #d8d5ca; font-size: 0.82rem; }

  @media (max-width: 540px) {
    .position-branches { grid-template-columns: minmax(0, 1fr); gap: 9px; }
  }
</style>
