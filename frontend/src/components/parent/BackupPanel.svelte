<script lang="ts">
  import { getAPI } from '../../lib/api'

  let includeLibrary = false
  let busy = false
  let message = ''
  let error = ''

  async function create(): Promise<void> {
    busy = true
    message = ''
    error = ''
    try {
      const path = await getAPI().createBackup(includeLibrary)
      if (path) message = 'Backup saved'
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause)
    } finally {
      busy = false
    }
  }

  async function restore(): Promise<void> {
    busy = true
    message = ''
    error = ''
    try {
      await getAPI().restoreBackup('')
    } catch (cause) {
      error = cause instanceof Error ? cause.message : String(cause)
    } finally {
      busy = false
    }
  }
</script>

<article class="dashboard-card backup-panel">
  <h3>Backup & restore</h3>
  <p class="muted">Keep a portable copy of progress and settings. Restoring closes and restarts the app.</p>
  <label class="include-library">
    <input type="checkbox" bind:checked={includeLibrary} />
    Include game library
  </label>
  <div class="button-row">
    <button class="primary" type="button" disabled={busy} on:click={create}>Create backup</button>
    <button class="secondary" type="button" disabled={busy} on:click={restore}>Restore backup</button>
  </div>
  {#if message}<p class="saved" role="status">{message}</p>{/if}
  {#if error}<p class="error" role="alert">{error}</p>{/if}
</article>

<style>
  .backup-panel { margin-top: 16px; }
  .backup-panel h3 { margin: 0 0 8px; }
  .include-library { display: flex; margin: 18px 0; align-items: center; }
  .include-library input { width: 22px; min-height: 22px; margin: 0; }
  .saved { color: var(--forest-700); font-weight: 800; }
</style>
