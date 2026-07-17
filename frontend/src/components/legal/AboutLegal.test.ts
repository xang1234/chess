import { fireEvent, render, screen } from '@testing-library/svelte'
import type { BuildInfo } from '../../lib/api'
import type { LegalDocuments } from '../../lib/legal-assets'
import AboutLegal from './AboutLegal.svelte'

const commit = '0123456789abcdef0123456789abcdef01234567'
const buildInfo: BuildInfo = {
  name: 'Chess Trainer',
  commit,
  sourceUrl: `https://github.com/xang1234/chess/tree/${commit}`
}
const documents: LegalDocuments = {
  application: 'complete application GPL text',
  notices: 'complete third-party notices',
  chessground: 'complete Chessground GPL text',
  nunito: 'complete Nunito OFL text'
}

test('shows rights, attribution, and exact matching source before documents load', () => {
  const loadDocuments = vi.fn(() => new Promise<LegalDocuments>(() => {}))
  render(AboutLegal, { buildInfo, loadDocuments, openExternal: vi.fn() })

  expect(screen.getByText('Copyright © 2026 David Ten and Chess Trainer contributors')).toBeInTheDocument()
  expect(screen.getByText(/modify and redistribute.*GPL-3\.0-or-later/i)).toBeInTheDocument()
  expect(screen.getByText(/WITHOUT ANY WARRANTY/)).toBeInTheDocument()
  expect(screen.getByText('@lichess-org/chessground 10.1.1')).toBeInTheDocument()
  expect(screen.getByText('Lichess Team <contact@lichess.org>')).toBeInTheDocument()
  expect(screen.getByText('https://github.com/lichess-org/chessground')).toBeInTheDocument()
  expect(screen.getByText('Copyright 2016 The Nunito Project Authors (contact@sansoxygen.com)')).toBeInTheDocument()
  expect(screen.getByText(/SIL Open Font License 1\.1/)).toBeInTheDocument()
  expect(screen.getByText('third_party/source/chessground-v10.1.1.tar.gz')).toBeInTheDocument()
  expect(screen.getByText(commit)).toBeInTheDocument()
  expect(screen.getByText(buildInfo.sourceUrl)).toBeInTheDocument()
  expect(loadDocuments).toHaveBeenCalledOnce()
})

test('renders all bundled legal documents in labelled disclosure regions', async () => {
  render(AboutLegal, {
    buildInfo,
    loadDocuments: async () => documents,
    openExternal: vi.fn()
  })

  expect(await screen.findByLabelText('Application license text')).toHaveTextContent(documents.application)
  expect(screen.getByLabelText('Third-party notices text')).toHaveTextContent(documents.notices)
  expect(screen.getByLabelText('Chessground license text')).toHaveTextContent(documents.chessground)
  expect(screen.getByLabelText('Nunito license text')).toHaveTextContent(documents.nunito)
  expect(screen.getByText('Application license (GPL-3.0-or-later)')).toBeInTheDocument()
  expect(screen.getByText('Third-party notices')).toBeInTheDocument()
  expect(screen.getByText('Chessground license')).toBeInTheDocument()
  expect(screen.getByText('Nunito SIL Open Font License 1.1')).toBeInTheDocument()
})

test('keeps rights and source visible when a bundled legal asset cannot load', async () => {
  render(AboutLegal, {
    buildInfo,
    loadDocuments: async () => { throw new Error('missing /legal/LICENSE.txt') },
    openExternal: vi.fn()
  })

  expect(await screen.findByRole('alert')).toHaveTextContent('missing /legal/LICENSE.txt')
  expect(screen.getByText(/modify and redistribute.*GPL-3\.0-or-later/i)).toBeInTheDocument()
  expect(screen.getByText(buildInfo.sourceUrl)).toBeInTheDocument()
})

test('opens only the supplied exact source URL and dispatches Back', async () => {
  const openExternal = vi.fn()
  const { component } = render(AboutLegal, {
    buildInfo,
    loadDocuments: async () => documents,
    openExternal
  })
  let wentBack = false
  component.$on('back', () => { wentBack = true })

  await fireEvent.click(screen.getByRole('button', { name: 'Open matching source' }))
  await fireEvent.click(screen.getByRole('button', { name: 'Back' }))

  expect(openExternal).toHaveBeenCalledOnce()
  expect(openExternal).toHaveBeenCalledWith(buildInfo.sourceUrl)
  expect(wentBack).toBe(true)
})
