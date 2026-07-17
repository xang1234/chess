import { loadLegalDocuments } from './legal-assets'

const paths = [
  '/legal/LICENSE.txt',
  '/legal/THIRD_PARTY_NOTICES.md',
  '/legal/CHESSGROUND_LICENSE.txt',
  '/legal/NUNITO_OFL.txt'
]

function response(body: string, ok = true): Response {
  return {
    ok,
    status: ok ? 200 : 404,
    statusText: ok ? 'OK' : 'Not Found',
    text: async () => body
  } as Response
}

test('loads all four bundled legal documents from fixed local paths', async () => {
  const fetcher = vi.fn(async (input: RequestInfo | URL) => response(`contents:${String(input)}`))

  const documents = await loadLegalDocuments(fetcher as typeof fetch)

  expect(fetcher.mock.calls.map(([input]) => String(input))).toEqual(paths)
  expect(fetcher.mock.calls.every(([input]) => String(input).startsWith('/legal/'))).toBe(true)
  expect(documents).toEqual({
    application: 'contents:/legal/LICENSE.txt',
    notices: 'contents:/legal/THIRD_PARTY_NOTICES.md',
    chessground: 'contents:/legal/CHESSGROUND_LICENSE.txt',
    nunito: 'contents:/legal/NUNITO_OFL.txt'
  })
})

test('identifies a missing bundled legal asset', async () => {
  const fetcher = vi.fn(async (input: RequestInfo | URL) => {
    const path = String(input)
    return path === '/legal/CHESSGROUND_LICENSE.txt'
      ? response('', false)
      : response(`contents:${path}`)
  })

  await expect(loadLegalDocuments(fetcher as typeof fetch))
    .rejects.toThrow(/CHESSGROUND_LICENSE\.txt.*404.*Not Found/i)
})
