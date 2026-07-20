<script lang="ts">
  import { createEventDispatcher } from 'svelte'
  import type { OpeningTreeBranch } from './opening-tree'

  export let branch: OpeningTreeBranch

  const dispatch = createEventDispatcher<{ lesson: string }>()

  $: stateLabels = [
    branch.node.progress === 'completed'
      ? 'Complete'
      : branch.node.progress === 'in_progress'
        ? 'In progress'
        : 'Available',
    ...(branch.node.recommended ? ['Recommended'] : []),
    ...(branch.node.reviewDue ? ['Review due'] : []),
    ...(!branch.node.visible ? ['Hidden at this depth'] : [])
  ]
  $: accessibleLabel = [
    branch.incoming?.label,
    branch.node.title,
    ...stateLabels,
    `${branch.node.completedActivities} of ${branch.node.requiredActivities} ideas`
  ].filter(Boolean).join('. ')
</script>

<li
  class:hidden={!branch.node.visible}
  class:recommended={branch.node.recommended}
  class={`opening-tree-node progress-${branch.node.progress}`}
  role="treeitem"
  aria-label={accessibleLabel}
  aria-selected="false"
  aria-expanded={branch.children.length > 0 ? true : undefined}
>
  {#if branch.incoming}
    <span class={`opening-edge-label edge-${branch.incoming.kind}`}>
      {branch.incoming.label || branch.incoming.kind}
    </span>
  {/if}
  <article class="opening-tree-card">
    <div class="opening-tree-copy">
      <strong>{branch.node.title}</strong>
      <p>{branch.node.objective}</p>
      <div class="opening-tree-status" aria-hidden="true">
        {#each stateLabels as label}<span>{label}</span>{/each}
        <span>{branch.node.completedActivities} of {branch.node.requiredActivities} ideas</span>
      </div>
    </div>
    <button
      class="secondary"
      type="button"
      disabled={!branch.node.visible}
      on:click={() => dispatch('lesson', branch.node.lessonId)}
    >Study {branch.node.title}</button>
  </article>

  {#if branch.children.length > 0}
    <ul class="opening-tree-children" role="group">
      {#each branch.children as child (child.node.lessonId)}
        <svelte:self branch={child} on:lesson />
      {/each}
    </ul>
  {/if}
</li>
