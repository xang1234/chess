export type LegalDocuments = {
  application: string
  notices: string
  chessground: string
  nunito: string
}

const legalPaths = {
  application: '/legal/LICENSE.txt',
  notices: '/legal/THIRD_PARTY_NOTICES.md',
  chessground: '/legal/CHESSGROUND_LICENSE.txt',
  nunito: '/legal/NUNITO_OFL.txt'
} as const

export async function loadLegalDocuments(
  fetcher: typeof fetch = fetch
): Promise<LegalDocuments> {
  const documents = {} as LegalDocuments
  const keys = Object.keys(legalPaths) as Array<keyof LegalDocuments>
  for (const key of keys) {
    const path = legalPaths[key]
    const response = await fetcher(path)
    if (!response.ok) {
      throw new Error(
        `Unable to load bundled legal asset ${path}: ${response.status} ${response.statusText}`.trim()
      )
    }
    documents[key] = await response.text()
  }
  return documents
}
