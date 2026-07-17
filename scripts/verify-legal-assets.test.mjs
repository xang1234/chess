import assert from 'node:assert/strict'
import { mkdtemp, mkdir, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import path from 'node:path'
import test from 'node:test'

import {
  assertDigest,
  assertTreesEqual,
  verifyLegalAssets,
  verifyNoticeContents,
  verifyPackageMetadata,
  verifyRuntimeLock,
} from './verify-legal-assets.mjs'

async function fixtureRoot() {
  const root = await mkdtemp(path.join(tmpdir(), 'chess-trainer-legal-'))
  await mkdir(path.join(root, 'frontend', 'public', 'legal'), { recursive: true })
  await writeFile(path.join(root, 'LICENSE'), 'truncated GPL\n')
  await writeFile(
    path.join(root, 'frontend', 'public', 'legal', 'LICENSE.txt'),
    'truncated GPL\n',
  )
  return root
}

test('rejects a truncated GPL even when the public copy is identical', async () => {
  const root = await fixtureRoot()

  await assert.rejects(
    verifyLegalAssets({ root, verifyNotices: false }),
    /canonical GPL-3\.0 text/,
  )
})

test('rejects a floating Chessground dependency range', () => {
  const packageJSON = {
    dependencies: { '@lichess-org/chessground': '^10.1.1' },
  }
  const packageLock = {
    packages: {
      'node_modules/@lichess-org/chessground': {
        version: '10.1.1',
        integrity:
          'sha512-IBEs8+J64/zE8QB4NXxsvpjm/tHRjfQAdWwUh4xzqqN+RValgthWHemLnxsmtHFwuxvO4lHd+crp1ecgZxKVoQ==',
      },
      'node_modules/svelte': {
        version: '3.59.2',
        integrity:
          'sha512-vzSyuGr3eEoAtT/A6bmajosJZIUWySzY2CzB3w2pgPvnkUjGqlDnsNnA0PMO+mMAhuyMul6C2uuZzY6ELSkzyA==',
      },
    },
  }

  assert.throws(
    () => verifyPackageMetadata(packageJSON, packageLock),
    /Chessground dependency must be pinned exactly to 10\.1\.1/,
  )
})

test('rejects a changed preferred-source archive', () => {
  assert.throws(
    () =>
      assertDigest(
        'Chessground preferred source',
        Buffer.from('changed archive'),
        'a926875d49a5a3302bc17051480577ddbc221f879f990cda5c5f6cea38bfecd5',
      ),
    /Chessground preferred source has an unexpected SHA-256/,
  )
})

test('rejects installed source that differs from preferred source', async () => {
  const root = await mkdtemp(path.join(tmpdir(), 'chess-trainer-source-'))
  const preferred = path.join(root, 'preferred')
  const installed = path.join(root, 'installed')
  await mkdir(preferred)
  await mkdir(installed)
  await writeFile(path.join(preferred, 'board.ts'), 'reviewed source\n')
  await writeFile(path.join(installed, 'board.ts'), 'changed source\n')

  await assert.rejects(
    assertTreesEqual(preferred, installed, 'Chessground src'),
    /Chessground src differs at board\.ts/,
  )
})

test('rejects notices that omit the bundled Nunito license', () => {
  assert.throws(
    () =>
      verifyNoticeContents(`
        @lichess-org/chessground 10.1.1
        GPL-3.0-or-later
        Lichess Team <contact@lichess.org>
        svelte 3.59.2
        Go runtime and standard library go1.26.4
      `),
    /notices must identify the bundled Nunito font/,
  )
})

test('rejects a runtime lock without the Go patent grant', () => {
  assert.throws(
    () =>
      verifyRuntimeLock({
        formatVersion: 1,
        goToolchain: {
          version: 'go1.26.4',
          legal: [
            {
              name: 'LICENSE',
              sha256:
                '911f8f5782931320f5b8d1160a76365b83aea6447ee6c04fa6d5591467db9dad',
            },
          ],
        },
        goModules: [{ path: 'example.com/module', version: 'v1.0.0' }],
        frontend: [],
      }),
    /runtime lock must include the exact Go PATENTS/,
  )
})

test('rejects a runtime lock missing a Wails production-tag dependency', () => {
  const requiredModules = [
    ['github.com/pkg/browser', 'v0.0.0-20240102092130-5ac0b6a4141c'],
    ['github.com/samber/lo', 'v1.49.1'],
    ['github.com/tkrajina/go-reflector', 'v0.5.8'],
    ['github.com/wailsapp/mimetype', 'v1.4.1'],
    ['golang.org/x/net', 'v0.35.0'],
    ['golang.org/x/text', 'v0.22.0'],
  ]
  const lock = {
    formatVersion: 1,
    goToolchain: {
      version: 'go1.26.4',
      legal: [
        {
          name: 'LICENSE',
          sha256:
            '911f8f5782931320f5b8d1160a76365b83aea6447ee6c04fa6d5591467db9dad',
        },
        {
          name: 'PATENTS',
          sha256:
            '96f408bfae65bf137fc2525d3ecb030271c50c1e90799f87abf8846d8dd505cc',
        },
      ],
    },
    goModules: requiredModules
      .slice(1)
      .map(([modulePath, version]) => ({ path: modulePath, version, legal: [] })),
    frontend: [],
  }

  assert.throws(
    () => verifyRuntimeLock(lock),
    /runtime lock is missing production module github\.com\/pkg\/browser/,
  )
})
