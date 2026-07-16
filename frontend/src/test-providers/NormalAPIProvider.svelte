<script lang="ts">
  import type { NormalAPI } from '../lib/api'
  import { provideNormalAPI } from '../lib/api-context'

  export let api: NormalAPI
  provideNormalAPI(api)
</script>

<slot />
