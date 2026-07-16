<script lang="ts">
  import type { RecoveryAPI } from '../lib/api'
  import { provideRecoveryAPI } from '../lib/api-context'

  export let api: RecoveryAPI
  provideRecoveryAPI(api)
</script>

<slot />
