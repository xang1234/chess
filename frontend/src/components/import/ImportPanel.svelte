<script lang="ts">
  import type { ImportSession } from '../../lib/import-session'
  import ImportCard from './ImportCard.svelte'

  export let puzzleSession: ImportSession
  export let courseSession: ImportSession
</script>

<section class="panel import-panel" aria-labelledby="import-title">
  <p class="eyebrow">Local content</p>
  <h2 id="import-title">Import content</h2>
  <p class="muted intro">
    Select a puzzle collection or a private opening course stored on this computer.
  </p>

  <div class="import-grid">
    <ImportCard
      kind="puzzle"
      session={puzzleSession}
      heading="Import puzzle collection"
      description="Lichess, tactical PGN, canonical JSON, Lucas FNS, and linear FEN/UCI collections are streamed directly from this computer. No intermediate copy is created."
      fileLabel="Puzzle collection file"
      chooseLabel="Choose puzzle collection"
      startLabel="Import puzzles"
    />
    <ImportCard
      kind="course"
      session={courseSession}
      heading="Import opening course"
      description="Private .ctcourse files stay on this Mac. The validated course is copied into the local course library."
      fileLabel="Opening course file"
      chooseLabel="Choose opening course"
      startLabel="Import course"
    />
  </div>
</section>

<style>
  .import-panel { max-width: 1180px; }
  .import-panel h2 { margin-bottom: 8px; }
  .intro { margin: 0; }
  .import-grid {
    display: grid;
    margin-top: 22px;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 18px;
  }
  @media (max-width: 850px) {
    .import-grid { grid-template-columns: 1fr; }
  }
</style>
