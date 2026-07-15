import { fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { vi } from 'vitest'
import { setAPIForTests } from '../../lib/api'
import { fakeAPI } from '../../test-fakes'
import BackupPanel from './BackupPanel.svelte'

test('creates an optional-library backup and restores through the native picker', async () => {
  const createBackup = vi.fn().mockResolvedValue('/Users/parent/Trainer.zip')
  const restoreBackup = vi.fn().mockResolvedValue(undefined)
  setAPIForTests(fakeAPI({ createBackup, restoreBackup }))
  render(BackupPanel)

  await fireEvent.click(screen.getByLabelText('Include game library'))
  await fireEvent.click(screen.getByRole('button', { name: 'Create backup' }))
  await waitFor(() => expect(createBackup).toHaveBeenCalledWith(true))
  expect(screen.getByText('Backup saved')).toBeInTheDocument()

  await fireEvent.click(screen.getByRole('button', { name: 'Restore backup' }))
  await waitFor(() => expect(restoreBackup).toHaveBeenCalledWith(''))
  await waitFor(() => expect(screen.getByRole('button', { name: 'Restore backup' })).toBeEnabled())
})
