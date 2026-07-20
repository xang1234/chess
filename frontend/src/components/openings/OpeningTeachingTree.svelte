<script lang="ts">
  import type { OpeningTeachingTree } from '../../lib/api'
  import OpeningTreeBranch from './OpeningTreeBranch.svelte'
  import { projectOpeningTree } from './opening-tree'

  export let tree: OpeningTeachingTree
  export let courseTitle: string

  $: projection = projectOpeningTree(tree)
</script>

<div class="opening-tree-scroll">
  <ul class="opening-teaching-tree" role="tree" aria-label={`${courseTitle} course roadmap`}>
    <OpeningTreeBranch branch={projection} on:lesson />
  </ul>
</div>
