import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'

const allowedOrigin = 'https://github.com'

export type ExternalLinkPorts = {
  runtimeAvailable(): boolean
  runtimeOpen(url: string): void
  browserOpen(url: string, target: string, features: string): unknown
}

function browserPorts(): ExternalLinkPorts {
  return {
    runtimeAvailable: () => typeof window !== 'undefined' &&
      typeof window.runtime?.BrowserOpenURL === 'function',
    runtimeOpen: BrowserOpenURL,
    browserOpen: (url, target, features) => window.open(url, target, features)
  }
}

export function openExternal(url: string, ports: ExternalLinkPorts = browserPorts()): void {
  let parsed: URL
  try {
    parsed = new URL(url)
  } catch {
    throw new Error(`External source links must use ${allowedOrigin}.`)
  }
  if (parsed.origin !== allowedOrigin || parsed.username || parsed.password) {
    throw new Error(`External source links must use ${allowedOrigin}.`)
  }
  if (ports.runtimeAvailable()) {
    ports.runtimeOpen(parsed.href)
    return
  }
  ports.browserOpen(parsed.href, '_blank', 'noopener,noreferrer')
}
