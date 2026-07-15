<script lang="ts">
  import type { RecoveryState } from '../../lib/api'
  import { getAPI } from '../../lib/api'

  export let state: RecoveryState
  let restoring = false
  let error = ''

  async function restore(): Promise<void> {
    restoring = true
    error = ''
    try {
      await getAPI().restoreBackup('')
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause)
    } finally {
      restoring = false
    }
  }
</script>

<section class="recovery-panel panel" aria-labelledby="recovery-title">
  <span class="recovery-mark" aria-hidden="true">♜</span>
  <p class="eyebrow">Data protection</p>
  <h2 id="recovery-title">Your chess data needs recovery</h2>
  <p>The app found a problem and left every file untouched. Restore a trusted backup, or inspect the data folder.</p>
  <details>
    <summary>Technical details</summary>
    <p><strong>File:</strong> {state.path}</p>
    <p>{state.detail}</p>
  </details>
  {#if error}<p class="error" role="alert">{error}</p>{/if}
  <div class="recovery-actions">
    <button class="primary" type="button" disabled={restoring} on:click={restore}>
      {restoring ? 'Restoring…' : 'Restore backup'}
    </button>
    <button class="secondary" type="button" on:click={() => getAPI().openDataFolder()}>Open data folder</button>
    <button class="quiet-action" type="button" on:click={() => getAPI().quit()}>Quit</button>
  </div>
</section>

<style>
  .recovery-panel { text-align: center; }
  .recovery-mark { display: block; color: var(--red-600); font-size: 4rem; }
  details { margin: 22px 0; padding: 14px; border-radius: 12px; text-align: left; background: var(--ivory-100); }
  summary { cursor: pointer; font-weight: 800; }
  details p { overflow-wrap: anywhere; }
  .recovery-actions { display: grid; gap: 10px; }
  .quiet-action { border: 0; color: var(--ink-700); background: transparent; font-weight: 800; }
</style>
