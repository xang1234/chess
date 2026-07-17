<script lang="ts">
  import { createEventDispatcher, onDestroy, onMount, tick } from 'svelte'
  import type { Promotion } from '../../lib/uci'

  export let choices: Promotion[]
  export let color: 'white' | 'black'
  export let returnFocus: HTMLElement | null = null

  const dispatch = createEventDispatcher<{
    choose: { promotion: Promotion }
    cancel: undefined
  }>()
  const order: Promotion[] = ['q', 'r', 'b', 'n']
  const names: Record<Promotion, string> = {
    q: 'queen',
    r: 'rook',
    b: 'bishop',
    n: 'knight'
  }

  let choiceButtons: HTMLButtonElement[] = []
  let cancelButton: HTMLButtonElement
  let focusRestored = false
  $: available = order.filter((choice) => choices.includes(choice))

  function restoreFocus(): void {
    if (focusRestored) return
    focusRestored = true
    returnFocus?.focus()
  }

  function choose(promotion: Promotion): void {
    if (!available.includes(promotion)) return
    restoreFocus()
    dispatch('choose', { promotion })
  }

  function cancel(): void {
    restoreFocus()
    dispatch('cancel')
  }

  function onKeydown(event: KeyboardEvent): void {
    if (event.key === 'Tab') {
      containFocus(event)
      return
    }
    if (event.key !== 'Escape') return
    event.preventDefault()
    event.stopPropagation()
    cancel()
  }

  function containFocus(event: KeyboardEvent): void {
    const buttons = [...choiceButtons, cancelButton].filter(
      (button): button is HTMLButtonElement => Boolean(button)
    )
    if (buttons.length === 0) return
    const current = buttons.indexOf(document.activeElement as HTMLButtonElement)
    const next = event.shiftKey
      ? (current <= 0 ? buttons.length - 1 : current - 1)
      : (current < 0 || current === buttons.length - 1 ? 0 : current + 1)
    event.preventDefault()
    buttons[next].focus()
  }

  onMount(async () => {
    await tick()
    if (!focusRestored) choiceButtons[0]?.focus()
  })
  onDestroy(restoreFocus)
</script>

<div class="promotion-backdrop" role="presentation">
  <section
    class="promotion-dialog cg-wrap"
    role="dialog"
    aria-modal="true"
    aria-labelledby="promotion-title"
    on:keydown={onKeydown}
  >
    <h3 id="promotion-title">Choose a promotion</h3>
    <div class="promotion-choices">
      {#each available as choice, index}
        <button
          bind:this={choiceButtons[index]}
          type="button"
          aria-label={`Promote to ${names[choice]}`}
          on:click={() => choose(choice)}
        >
          <piece class={`${color} ${names[choice]}`} aria-hidden="true"></piece>
          <span>{names[choice]}</span>
        </button>
      {/each}
    </div>
    <button bind:this={cancelButton} class="cancel" type="button" on:click={cancel}>Cancel</button>
  </section>
</div>

<style>
  .promotion-backdrop {
    position: absolute;
    z-index: 20;
    display: grid;
    background: rgba(23, 30, 27, 0.56);
    inset: 0;
    place-items: center;
  }
  .promotion-dialog {
    width: min(92%, 420px);
    padding: 20px;
    border-radius: 16px;
    background: #fffdf7;
    box-shadow: 0 20px 52px rgba(0, 0, 0, 0.3);
  }
  h3 { margin: 0 0 14px; text-align: center; }
  .promotion-choices { display: grid; grid-template-columns: repeat(auto-fit, minmax(72px, 1fr)); gap: 8px; }
  .promotion-choices button {
    display: grid;
    min-height: 92px;
    padding: 8px;
    border: 2px solid transparent;
    border-radius: 10px;
    background: #f2f0e6;
    color: #26342e;
    font-weight: 800;
    place-items: center;
    text-transform: capitalize;
  }
  .promotion-choices button:hover,
  .promotion-choices button:focus-visible { border-color: #769656; outline: none; }
  piece { display: block; width: 52px; height: 52px; background-position: center; background-repeat: no-repeat; background-size: contain; }
  .cancel { display: block; margin: 14px auto 0; border: 0; background: transparent; color: #42534b; font-weight: 800; }
</style>
