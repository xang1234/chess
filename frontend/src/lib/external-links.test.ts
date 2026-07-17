import { openExternal, type ExternalLinkPorts } from './external-links'

const sourceURL = 'https://github.com/xang1234/chess/tree/0123456789abcdef0123456789abcdef01234567'

function ports(runtimeAvailable: boolean) {
  const runtimeOpen = vi.fn()
  const browserOpen = vi.fn()
  const value: ExternalLinkPorts = {
    runtimeAvailable: () => runtimeAvailable,
    runtimeOpen,
    browserOpen
  }
  return { browserOpen, runtimeOpen, value }
}

test('uses Wails BrowserOpenURL when the runtime is available', () => {
  const target = ports(true)

  openExternal(sourceURL, target.value)

  expect(target.runtimeOpen).toHaveBeenCalledWith(sourceURL)
  expect(target.browserOpen).not.toHaveBeenCalled()
})

test('uses a noopener browser tab in preview mode', () => {
  const target = ports(false)

  openExternal(sourceURL, target.value)

  expect(target.runtimeOpen).not.toHaveBeenCalled()
  expect(target.browserOpen).toHaveBeenCalledWith(
    sourceURL,
    '_blank',
    'noopener,noreferrer'
  )
})

test.each([
  'http://github.com/xang1234/chess',
  'https://github.com.evil.example/xang1234/chess',
  'https://example.com/xang1234/chess',
  'javascript:alert(1)'
])('rejects non-GitHub external URL %s', (url) => {
  const target = ports(true)

  expect(() => openExternal(url, target.value)).toThrow(/https:\/\/github\.com/i)
  expect(target.runtimeOpen).not.toHaveBeenCalled()
  expect(target.browserOpen).not.toHaveBeenCalled()
})
