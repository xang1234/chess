import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const bundleIdentifier = 'com.xang1234.chesstrainer'

test('macOS production and development templates use the stable valid bundle identifier', async () => {
  for (const name of ['Info.plist', 'Info.dev.plist']) {
    const template = await readFile(path.join(repositoryRoot, 'build/darwin', name), 'utf8')
    const match = /<key>CFBundleIdentifier<\/key>\s*<string>([^<]+)<\/string>/.exec(template)
    assert.ok(match, `${name} must declare CFBundleIdentifier`)
    assert.equal(match[1], bundleIdentifier)
    assert.match(match[1], /^[A-Za-z0-9.-]+$/)
  }
})
