import { fireEvent, render, screen } from '@testing-library/svelte'
import { vi } from 'vitest'
import { setAPIForTests } from '../../lib/api'
import { fakeAPI } from '../../test-fakes'
import RecoveryPanel from './RecoveryPanel.svelte'

test('offers only safe recovery actions for a damaged database', async () => {
  const restoreBackup = vi.fn().mockResolvedValue(undefined)
  const openDataFolder = vi.fn().mockResolvedValue(undefined)
  const quit = vi.fn().mockResolvedValue(undefined)
  setAPIForTests(fakeAPI({ restoreBackup, openDataFolder, quit }))
  render(RecoveryPanel, {
    state: { required: true, path: '/data/user.sqlite', detail: 'database disk image is malformed' }
  })

  expect(screen.getByText('Your chess data needs recovery')).toBeInTheDocument()
  expect(screen.getAllByRole('button')).toHaveLength(3)
  await fireEvent.click(screen.getByRole('button', { name: 'Restore backup' }))
  expect(await screen.findByRole('button', { name: 'Restore backup' })).toBeEnabled()
  await fireEvent.click(screen.getByRole('button', { name: 'Open data folder' }))
  await fireEvent.click(screen.getByRole('button', { name: 'Quit' }))
  expect(restoreBackup).toHaveBeenCalledWith('')
  expect(openDataFolder).toHaveBeenCalledOnce()
  expect(quit).toHaveBeenCalledOnce()
})
