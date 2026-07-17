import assert from 'node:assert/strict'
import { execFile } from 'node:child_process'
import { createHash } from 'node:crypto'
import { mkdtemp, readFile, readdir, rm, stat } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import { promisify } from 'node:util'

import { runNoticeGenerator } from './generate-third-party-notices.mjs'

const execFileAsync = promisify(execFile)

export const GPL3_SHA256 =
  '8ceb4b9ee5adedde47b31e975c1d90c73ad27b6b165a1dcd80c7c545eb65b903'
export const CHESSGROUND_VERSION = '10.1.1'
export const CHESSGROUND_INTEGRITY =
  'sha512-IBEs8+J64/zE8QB4NXxsvpjm/tHRjfQAdWwUh4xzqqN+RValgthWHemLnxsmtHFwuxvO4lHd+crp1ecgZxKVoQ=='
export const SVELTE_VERSION = '3.59.2'
export const SVELTE_INTEGRITY =
  'sha512-vzSyuGr3eEoAtT/A6bmajosJZIUWySzY2CzB3w2pgPvnkUjGqlDnsNnA0PMO+mMAhuyMul6C2uuZzY6ELSkzyA=='
export const CHESSGROUND_SOURCE_SHA256 =
  'a926875d49a5a3302bc17051480577ddbc221f879f990cda5c5f6cea38bfecd5'
export const SVELTE_SOURCE_SHA256 =
  '2360bdebd06141a2f0566364c7b42e87140bf2ed9494df29dac0bd43dbf99bad'
export const GO_LICENSE_SHA256 =
  '911f8f5782931320f5b8d1160a76365b83aea6447ee6c04fa6d5591467db9dad'
export const GO_PATENTS_SHA256 =
  '96f408bfae65bf137fc2525d3ecb030271c50c1e90799f87abf8846d8dd505cc'
export const REQUIRED_DARWIN_PRODUCTION_MODULES = Object.freeze(new Map([
  ['github.com/pkg/browser', 'v0.0.0-20240102092130-5ac0b6a4141c'],
  ['github.com/samber/lo', 'v1.49.1'],
  ['github.com/tkrajina/go-reflector', 'v0.5.8'],
  ['github.com/wailsapp/mimetype', 'v1.4.1'],
  ['golang.org/x/net', 'v0.35.0'],
  ['golang.org/x/text', 'v0.22.0'],
]))

export function sha256(content) {
  return createHash('sha256').update(content).digest('hex')
}

export function assertDigest(label, content, expected) {
  assert.equal(
    sha256(content),
    expected,
    `${label} has an unexpected SHA-256`,
  )
}

async function treeFiles(root, directory = root) {
  const files = []
  const entries = await readdir(directory, { withFileTypes: true })
  for (const entry of entries.sort((left, right) =>
    left.name.localeCompare(right.name),
  )) {
    const filename = path.join(directory, entry.name)
    if (entry.isDirectory()) files.push(...(await treeFiles(root, filename)))
    else if (entry.isFile()) files.push(path.relative(root, filename))
    else throw new Error(`unsupported source-tree entry: ${filename}`)
  }
  return files
}

export async function assertTreesEqual(preferred, installed, label) {
  const preferredFiles = await treeFiles(preferred)
  const installedFiles = await treeFiles(installed)
  assert.deepEqual(
    installedFiles,
    preferredFiles,
    `${label} file list differs from preferred source`,
  )

  for (const relative of preferredFiles) {
    const [preferredContent, installedContent] = await Promise.all([
      readFile(path.join(preferred, relative)),
      readFile(path.join(installed, relative)),
    ])
    assert.deepEqual(
      installedContent,
      preferredContent,
      `${label} differs at ${relative}`,
    )
  }
}

export function verifyNoticeContents(notice) {
  assert.match(
    notice,
    /@lichess-org\/chessground 10\.1\.1/,
    'notices must identify Chessground 10.1.1',
  )
  assert.match(notice, /GPL-3\.0-or-later/, 'notices must identify GPL terms')
  assert.match(
    notice,
    /Lichess Team <contact@lichess\.org>/,
    'notices must include the Chessground author metadata',
  )
  assert.match(notice, /svelte 3\.59\.2/, 'notices must identify Svelte 3.59.2')
  assert.match(
    notice,
    /Nunito/,
    'notices must identify the bundled Nunito font',
  )
  assert.match(
    notice,
    /Go runtime and standard library go1\.26\.4/,
    'notices must identify the Go release toolchain',
  )
}

export function verifyRuntimeLock(lock) {
  assert.equal(lock.formatVersion, 1, 'unsupported runtime lock format')
  assert.equal(
    lock.goToolchain?.version,
    'go1.26.4',
    'runtime lock must identify go1.26.4',
  )
  const goLicense = lock.goToolchain?.legal?.find(
    (entry) => entry.name === 'LICENSE',
  )
  const goPatents = lock.goToolchain?.legal?.find(
    (entry) => entry.name === 'PATENTS',
  )
  assert.equal(
    goLicense?.sha256,
    '911f8f5782931320f5b8d1160a76365b83aea6447ee6c04fa6d5591467db9dad',
    'runtime lock must include the exact Go LICENSE',
  )
  assert.equal(
    goPatents?.sha256,
    '96f408bfae65bf137fc2525d3ecb030271c50c1e90799f87abf8846d8dd505cc',
    'runtime lock must include the exact Go PATENTS',
  )
  assert.ok(lock.goModules?.length > 0, 'runtime lock must include Go modules')
  const lockedModules = new Map(
    lock.goModules.map((module) => [module.path, module.version]),
  )
  for (const [modulePath, version] of REQUIRED_DARWIN_PRODUCTION_MODULES) {
    assert.equal(
      lockedModules.get(modulePath),
      version,
      `runtime lock is missing production module ${modulePath}@${version}`,
    )
  }
  assert.doesNotMatch(
    JSON.stringify(lock),
    /\/Users\/|(?:^|["\s])\.\.?(?:\/|["\s])/,
    'runtime lock must not contain a machine-local path',
  )
}

export function verifyPackageMetadata(packageJSON, packageLock) {
  assert.equal(
    packageJSON.dependencies?.['@lichess-org/chessground'],
    CHESSGROUND_VERSION,
    `Chessground dependency must be pinned exactly to ${CHESSGROUND_VERSION}`,
  )

  const chessground =
    packageLock.packages?.['node_modules/@lichess-org/chessground']
  assert.equal(
    chessground?.version,
    CHESSGROUND_VERSION,
    `package-lock Chessground version must be ${CHESSGROUND_VERSION}`,
  )
  assert.equal(
    chessground?.integrity,
    CHESSGROUND_INTEGRITY,
    'package-lock Chessground integrity must match the reviewed package',
  )

  const svelte = packageLock.packages?.['node_modules/svelte']
  assert.equal(
    svelte?.version,
    SVELTE_VERSION,
    `package-lock Svelte version must be ${SVELTE_VERSION}`,
  )
  assert.equal(
    svelte?.integrity,
    SVELTE_INTEGRITY,
    'package-lock Svelte integrity must match the reviewed package',
  )
}

async function assertPathExists(filename, label) {
  try {
    await stat(filename)
  } catch (error) {
    if (error.code === 'ENOENT') throw new Error(`${label} is missing`)
    throw error
  }
}

async function extractArchive(archive) {
  const directory = await mkdtemp(path.join(tmpdir(), 'chess-trainer-source-'))
  await execFileAsync('tar', ['-xzf', archive, '-C', directory])
  const roots = (await readdir(directory, { withFileTypes: true })).filter(
    (entry) => entry.isDirectory(),
  )
  if (roots.length !== 1) {
    await rm(directory, { recursive: true, force: true })
    throw new Error('preferred-source archive must contain one root directory')
  }
  return {
    directory,
    sourceRoot: path.join(directory, roots[0].name),
  }
}

async function verifyChessgroundSource(root) {
  const archive = path.join(
    root,
    'third_party/source/chessground-v10.1.1.tar.gz',
  )
  const extracted = await extractArchive(archive)
  try {
    for (const required of [
      'tsconfig.json',
      'pnpm-lock.yaml',
      'package.json',
      'tests/fen.test.ts',
      'src/chessground.ts',
      'assets/chessground.cburnett.css',
      'LICENSE',
    ]) {
      await assertPathExists(
        path.join(extracted.sourceRoot, required),
        `Chessground preferred-source ${required}`,
      )
    }

    const installed = path.join(
      root,
      'frontend/node_modules/@lichess-org/chessground',
    )
    await assertTreesEqual(
      path.join(extracted.sourceRoot, 'src'),
      path.join(installed, 'src'),
      'Chessground src',
    )
    await assertTreesEqual(
      path.join(extracted.sourceRoot, 'assets'),
      path.join(installed, 'assets'),
      'Chessground assets',
    )
    assert.deepEqual(
      await readFile(path.join(extracted.sourceRoot, 'LICENSE')),
      await readFile(path.join(installed, 'LICENSE')),
      'Chessground installed license differs from preferred source',
    )
  } finally {
    await rm(extracted.directory, { recursive: true, force: true })
  }
}

async function verifySvelteSource(root) {
  const archive = path.join(root, 'third_party/source/svelte-v3.59.2.tar.gz')
  const extracted = await extractArchive(archive)
  try {
    for (const required of [
      'package.json',
      'package-lock.json',
      'rollup.config.js',
      'src/runtime/internal/index.ts',
      'test/css/index.ts',
      'LICENSE.md',
    ]) {
      await assertPathExists(
        path.join(extracted.sourceRoot, required),
        `Svelte preferred-source ${required}`,
      )
    }
    const sourcePackage = JSON.parse(
      await readFile(path.join(extracted.sourceRoot, 'package.json'), 'utf8'),
    )
    assert.equal(sourcePackage.version, SVELTE_VERSION)
    assert.deepEqual(
      await readFile(path.join(extracted.sourceRoot, 'LICENSE.md')),
      await readFile(path.join(root, 'frontend/node_modules/svelte/LICENSE.md')),
      'Svelte installed license differs from preferred source',
    )
  } finally {
    await rm(extracted.directory, { recursive: true, force: true })
  }
}

export async function verifyLegalAssets({ root, verifyNotices = true }) {
  const applicationLicense = await readFile(path.join(root, 'LICENSE'))
  const publicLicense = await readFile(
    path.join(root, 'frontend', 'public', 'legal', 'LICENSE.txt'),
  )

  assert.deepEqual(
    publicLicense,
    applicationLicense,
    'public application license must match the root license byte-for-byte',
  )
  assert.equal(
    sha256(applicationLicense),
    GPL3_SHA256,
    'root LICENSE must contain the canonical GPL-3.0 text',
  )

  const packageJSON = JSON.parse(
    await readFile(path.join(root, 'frontend', 'package.json'), 'utf8'),
  )
  const packageLock = JSON.parse(
    await readFile(path.join(root, 'frontend', 'package-lock.json'), 'utf8'),
  )
  verifyPackageMetadata(packageJSON, packageLock)

  const [
    rootNotice,
    publicNotice,
    publicChessgroundLicense,
    installedChessgroundLicense,
    publicNunitoLicense,
    repositoryNunitoLicense,
    chessgroundSourceArchive,
    svelteSourceArchive,
    goLicense,
    goPatents,
    runtimeLockText,
  ] = await Promise.all([
    readFile(path.join(root, 'THIRD_PARTY_NOTICES.md')),
    readFile(
      path.join(root, 'frontend/public/legal/THIRD_PARTY_NOTICES.md'),
    ),
    readFile(path.join(root, 'frontend/public/legal/CHESSGROUND_LICENSE.txt')),
    readFile(
      path.join(root, 'frontend/node_modules/@lichess-org/chessground/LICENSE'),
    ),
    readFile(path.join(root, 'frontend/public/legal/NUNITO_OFL.txt')),
    readFile(path.join(root, 'frontend/src/assets/fonts/OFL.txt')),
    readFile(
      path.join(root, 'third_party/source/chessground-v10.1.1.tar.gz'),
    ),
    readFile(path.join(root, 'third_party/source/svelte-v3.59.2.tar.gz')),
    readFile(path.join(root, 'third_party/legal/go1.26.4/LICENSE')),
    readFile(path.join(root, 'third_party/legal/go1.26.4/PATENTS')),
    readFile(path.join(root, 'third_party/runtime-dependencies.lock.json'), 'utf8'),
  ])

  assert.deepEqual(
    publicNotice,
    rootNotice,
    'public third-party notices must match the root copy byte-for-byte',
  )
  assert.deepEqual(
    publicChessgroundLicense,
    installedChessgroundLicense,
    'public Chessground license must match the installed package',
  )
  assert.deepEqual(
    publicNunitoLicense,
    repositoryNunitoLicense,
    'public Nunito license must match the bundled font license',
  )
  verifyNoticeContents(rootNotice.toString('utf8'))
  assertDigest(
    'Chessground preferred source',
    chessgroundSourceArchive,
    CHESSGROUND_SOURCE_SHA256,
  )
  assertDigest(
    'Svelte preferred source',
    svelteSourceArchive,
    SVELTE_SOURCE_SHA256,
  )
  assertDigest('Go LICENSE', goLicense, GO_LICENSE_SHA256)
  assertDigest('Go PATENTS', goPatents, GO_PATENTS_SHA256)
  verifyRuntimeLock(JSON.parse(runtimeLockText))

  await verifyChessgroundSource(root)
  await verifySvelteSource(root)
  if (verifyNotices) {
    await runNoticeGenerator({ root, mode: 'check' })
  }
}

const isCLI = process.argv[1]
  && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)

if (isCLI) {
  await verifyLegalAssets({
    root: path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..'),
  })
}
