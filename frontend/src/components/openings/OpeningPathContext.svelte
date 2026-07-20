<script lang="ts">
  import type { OpeningPathItem } from '../../lib/api'

  export let path: OpeningPathItem[] = []
</script>

{#if path.length > 0}
  <nav class="opening-path" aria-label="Opening course path">
    <ol>
      {#each path as item, index (item.lessonId)}
        <li aria-current={index === path.length - 1 ? 'step' : undefined}>
          {item.title}
        </li>
      {/each}
    </ol>
  </nav>
{/if}
