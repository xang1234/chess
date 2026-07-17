import assert from 'node:assert/strict'
import { execFile } from 'node:child_process'
import { createHash } from 'node:crypto'
import {
  mkdir,
  readFile,
  readdir,
  stat,
  writeFile,
} from 'node:fs/promises'
import path from 'node:path'
import { promisify } from 'node:util'
import { fileURLToPath } from 'node:url'

import { assertNoLocalModuleReplacement } from './go-module-policy.mjs'

export { assertNoLocalModuleReplacement } from './go-module-policy.mjs'

const execFileAsync = promisify(execFile)
export const RELEASE_BUILD_TAGS = 'desktop,production'

export const GO_RELEASE_VERSION = 'go1.26.4'
export const GO_LICENSE_SHA256 =
  '911f8f5782931320f5b8d1160a76365b83aea6447ee6c04fa6d5591467db9dad'
export const GO_PATENTS_SHA256 =
  '96f408bfae65bf137fc2525d3ecb030271c50c1e90799f87abf8846d8dd505cc'

const DEFAULT_POLICY = Object.freeze({
  goVersion: GO_RELEASE_VERSION,
  goLicenseSha256: GO_LICENSE_SHA256,
  goPatentsSha256: GO_PATENTS_SHA256,
  goLegalDirectory: `third_party/legal/${GO_RELEASE_VERSION}`,
})

function sha256(content) {
  return createHash('sha256').update(content).digest('hex')
}

export function mergeRuntimeModules(packageSets) {
  const modules = new Map()
  for (const packages of packageSets) {
    for (const pkg of packages) {
      const module = pkg.Module
      if (!module || module.Main) continue
      if (module.Replace && !module.Replace.Version) {
        throw new Error(
          `runtime module ${module.Path} uses a machine-local Go module replacement`,
        )
      }
      if (!module.Path || !module.Version || !module.Dir) {
        throw new Error(`runtime package ${pkg.ImportPath} has incomplete module data`)
      }

      const existing = modules.get(module.Path)
      if (existing && existing.version !== module.Version) {
        throw new Error(
          `runtime module ${module.Path} has conflicting versions `
            + `${existing.version} and ${module.Version}`,
        )
      }
      modules.set(module.Path, {
        path: module.Path,
        version: module.Version,
        directory: module.Replace?.Dir || module.Dir,
      })
    }
  }
  return [...modules.values()].sort((left, right) =>
    left.path.localeCompare(right.path),
  )
}

function renderLegalFiles(legal) {
  return legal
    .map(
      (file) =>
        `### ${file.name}\n\nSHA-256: \`${file.sha256}\`\n\n${file.text.trim()}\n`,
    )
    .join('\n')
}

export function renderThirdPartyNotices(inventory) {
  const sections = [
    '# Third-Party Notices',
    '',
    `Chess Trainer is distributed under ${inventory.applicationLicense}.`,
    '',
    `## Go runtime and standard library ${inventory.goToolchain.version}`,
    '',
    renderLegalFiles(inventory.goToolchain.legal).trimEnd(),
  ]

  for (const module of inventory.goModules) {
    sections.push(
      '',
      `## ${module.path} ${module.version}`,
      '',
      renderLegalFiles(module.legal).trimEnd(),
    )
  }

  for (const dependency of inventory.frontend) {
    sections.push('', `## ${dependency.name} ${dependency.version}`, '')
    for (const metadata of dependency.metadata ?? []) {
      sections.push(metadata, '')
    }
    sections.push(renderLegalFiles(dependency.legal).trimEnd())
  }

  return `${sections.join('\n').trimEnd()}\n`
}

export function validateGoToolchain(
  { version, license, patents },
  policy = DEFAULT_POLICY,
) {
  assert.equal(
    version,
    policy.goVersion,
    `release toolchain must be ${policy.goVersion}`,
  )
  assert.equal(
    sha256(license),
    policy.goLicenseSha256,
    'Go LICENSE must match the reviewed release toolchain',
  )
  assert.equal(
    sha256(patents),
    policy.goPatentsSha256,
    'Go PATENTS must match the reviewed release toolchain',
  )
}

function parseJSONStream(text) {
  const values = []
  let start = -1
  let depth = 0
  let quoted = false
  let escaped = false

  for (let index = 0; index < text.length; index += 1) {
    const character = text[index]
    if (quoted) {
      if (escaped) escaped = false
      else if (character === '\\') escaped = true
      else if (character === '"') quoted = false
      continue
    }
    if (character === '"') {
      quoted = true
      continue
    }
    if (character === '{') {
      if (depth === 0) start = index
      depth += 1
      continue
    }
    if (character === '}') {
      depth -= 1
      if (depth < 0) throw new Error('go list returned malformed JSON')
      if (depth === 0 && start >= 0) {
        values.push(JSON.parse(text.slice(start, index + 1)))
        start = -1
      }
    }
  }

  if (quoted || depth !== 0 || start !== -1) {
    throw new Error('go list returned incomplete JSON')
  }
  return values
}

async function defaultCommandRunner(command, args, options = {}) {
  const { stdout } = await execFileAsync(command, args, {
    cwd: options.cwd,
    env: options.env,
    encoding: 'utf8',
    maxBuffer: 128 * 1024 * 1024,
  })
  return stdout
}

async function readJSON(filename) {
  return JSON.parse(await readFile(filename, 'utf8'))
}

async function readLegalFiles(directory) {
  const entries = await readdir(directory, { withFileTypes: true })
  const legalNames = entries
    .filter(
      (entry) =>
        entry.isFile() && /^(?:LICENSE|COPYING|NOTICE)/i.test(entry.name),
    )
    .map((entry) => entry.name)
    .sort()

  if (legalNames.length === 0) {
    throw new Error(`runtime dependency has no license or notice in ${directory}`)
  }

  return Promise.all(
    legalNames.map(async (name) => {
      const content = await readFile(path.join(directory, name))
      return {
        name,
        sha256: sha256(content),
        text: content.toString('utf8'),
      }
    }),
  )
}

async function readFirstExisting(candidates, label) {
  for (const candidate of candidates) {
    try {
      if ((await stat(candidate)).isFile()) return readFile(candidate)
    } catch (error) {
      if (error.code !== 'ENOENT') throw error
    }
  }
  throw new Error(`cannot locate ${label} in the installed Go toolchain`)
}

async function readInstalledGoLegal(goRoot) {
  const parent = path.dirname(goRoot)
  return {
    license: await readFirstExisting(
      [path.join(goRoot, 'LICENSE'), path.join(parent, 'LICENSE')],
      'Go LICENSE',
    ),
    patents: await readFirstExisting(
      [path.join(goRoot, 'PATENTS'), path.join(parent, 'PATENTS')],
      'Go PATENTS',
    ),
  }
}

function stripLegalText(legal) {
  return legal.map(({ name, sha256: digestValue }) => ({
    name,
    sha256: digestValue,
  }))
}

function runtimeLock(inventory, policy) {
  return {
    formatVersion: 1,
    goToolchain: {
      version: inventory.goToolchain.version,
      source: policy.goLegalDirectory,
      legal: stripLegalText(inventory.goToolchain.legal),
    },
    goModules: inventory.goModules.map(({ path: modulePath, version, legal }) => ({
      path: modulePath,
      version,
      legal: stripLegalText(legal),
    })),
    frontend: inventory.frontend.map(({ name, version, metadata, legal }) => ({
      name,
      version,
      metadata,
      legal: stripLegalText(legal),
    })),
  }
}

async function collectInventory({ root, commandRunner, policy }) {
  const goMod = await readFile(path.join(root, 'go.mod'), 'utf8')
  assertNoLocalModuleReplacement(goMod)

  const packageSets = []
  for (const architecture of ['arm64', 'amd64']) {
    const output = await commandRunner(
      'go',
      ['list', '-tags', RELEASE_BUILD_TAGS, '-deps', '-json', '.'],
      {
        cwd: root,
        env: {
          ...process.env,
          CGO_ENABLED: '1',
          GOOS: 'darwin',
          GOARCH: architecture,
        },
      },
    )
    packageSets.push(parseJSONStream(output))
  }

  const goEnvironment = JSON.parse(
    await commandRunner('go', ['env', '-json', 'GOVERSION', 'GOROOT'], {
      cwd: root,
      env: process.env,
    }),
  )
  const committedGoDirectory = path.join(root, policy.goLegalDirectory)
  const committedGo = {
    version: goEnvironment.GOVERSION,
    license: await readFile(path.join(committedGoDirectory, 'LICENSE')),
    patents: await readFile(path.join(committedGoDirectory, 'PATENTS')),
  }
  validateGoToolchain(committedGo, policy)

  const installedGo = await readInstalledGoLegal(goEnvironment.GOROOT)
  assert.deepEqual(
    installedGo.license,
    committedGo.license,
    'installed Go LICENSE differs from the reviewed committed copy',
  )
  assert.deepEqual(
    installedGo.patents,
    committedGo.patents,
    'installed Go PATENTS differs from the reviewed committed copy',
  )

  const goModules = []
  for (const module of mergeRuntimeModules(packageSets)) {
    goModules.push({
      path: module.path,
      version: module.version,
      legal: await readLegalFiles(module.directory),
    })
  }

  const chessgroundDirectory = path.join(
    root,
    'frontend/node_modules/@lichess-org/chessground',
  )
  const chessgroundPackage = await readJSON(
    path.join(chessgroundDirectory, 'package.json'),
  )
  const svelteDirectory = path.join(root, 'frontend/node_modules/svelte')
  const sveltePackage = await readJSON(path.join(svelteDirectory, 'package.json'))
  const nunitoLicense = await readFile(
    path.join(root, 'frontend/src/assets/fonts/OFL.txt'),
  )

  const frontend = [
    {
      name: '@lichess-org/chessground',
      version: chessgroundPackage.version,
      metadata: [
        chessgroundPackage.author,
        typeof chessgroundPackage.repository === 'string'
          ? chessgroundPackage.repository
          : chessgroundPackage.repository?.url,
        'GPL-3.0-or-later',
        'Preferred source: third_party/source/chessground-v10.1.1.tar.gz',
      ].filter(Boolean),
      legal: await readLegalFiles(chessgroundDirectory),
    },
    {
      name: 'svelte',
      version: sveltePackage.version,
      metadata: [
        'MIT',
        'Preferred source: third_party/source/svelte-v3.59.2.tar.gz',
      ],
      legal: await readLegalFiles(svelteDirectory),
    },
    {
      name: 'Nunito',
      version: 'v16',
      metadata: [
        'Copyright 2016 The Nunito Project Authors (contact@sansoxygen.com)',
        'SIL Open Font License 1.1',
      ],
      legal: [
        {
          name: 'OFL.txt',
          sha256: sha256(nunitoLicense),
          text: nunitoLicense.toString('utf8'),
        },
      ],
    },
  ]

  return {
    applicationLicense: 'GPL-3.0-or-later',
    goToolchain: {
      version: committedGo.version,
      legal: [
        {
          name: 'LICENSE',
          sha256: sha256(committedGo.license),
          text: committedGo.license.toString('utf8'),
        },
        {
          name: 'PATENTS',
          sha256: sha256(committedGo.patents),
          text: committedGo.patents.toString('utf8'),
        },
      ],
    },
    goModules,
    frontend,
  }
}

function generatedFiles(root, inventory, policy, applicationLicense) {
  const notices = renderThirdPartyNotices(inventory)
  const chessground = inventory.frontend.find(
    (dependency) => dependency.name === '@lichess-org/chessground',
  )
  const nunito = inventory.frontend.find(
    (dependency) => dependency.name === 'Nunito',
  )
  return new Map([
    [
      path.join(root, 'third_party/runtime-dependencies.lock.json'),
      `${JSON.stringify(runtimeLock(inventory, policy), null, 2)}\n`,
    ],
    [path.join(root, 'THIRD_PARTY_NOTICES.md'), notices],
    [
      path.join(root, 'frontend/public/legal/THIRD_PARTY_NOTICES.md'),
      notices,
    ],
    [
      path.join(root, 'frontend/public/legal/LICENSE.txt'),
      applicationLicense.toString('utf8'),
    ],
    [
      path.join(root, 'frontend/public/legal/CHESSGROUND_LICENSE.txt'),
      chessground.legal.find((file) => file.name === 'LICENSE').text,
    ],
    [
      path.join(root, 'frontend/public/legal/NUNITO_OFL.txt'),
      nunito.legal[0].text,
    ],
  ])
}

export async function runNoticeGenerator({
  root,
  mode,
  commandRunner = defaultCommandRunner,
  policy = DEFAULT_POLICY,
}) {
  if (mode !== 'write' && mode !== 'check') {
    throw new Error('notice generator mode must be write or check')
  }

  const inventory = await collectInventory({ root, commandRunner, policy })
  const applicationLicense = await readFile(path.join(root, 'LICENSE'))
  const files = generatedFiles(root, inventory, policy, applicationLicense)

  for (const [filename, expected] of files) {
    if (mode === 'write') {
      await mkdir(path.dirname(filename), { recursive: true })
      await writeFile(filename, expected)
      continue
    }

    let actual
    try {
      actual = await readFile(filename, 'utf8')
    } catch (error) {
      if (error.code === 'ENOENT') {
        throw new Error(`generated legal file is missing: ${path.relative(root, filename)}`)
      }
      throw error
    }
    if (actual !== expected) {
      const relative = path.relative(root, filename)
      if (relative === 'third_party/runtime-dependencies.lock.json') {
        throw new Error('runtime dependency lock is out of date')
      }
      throw new Error(`generated legal file is out of date: ${relative}`)
    }
  }
}

const isCLI = process.argv[1]
  && path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)

if (isCLI) {
  const requested = process.argv.slice(2)
  if (requested.length !== 1 || !['--write', '--check'].includes(requested[0])) {
    throw new Error('usage: generate-third-party-notices.mjs --write|--check')
  }
  await runNoticeGenerator({
    root: path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..'),
    mode: requested[0].slice(2),
  })
}
