import { execFile } from 'node:child_process'
import {
  mkdtemp,
  readFile,
  readdir,
  realpath,
  rm,
  stat,
} from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { promisify } from 'node:util'

import {
  CHESSGROUND_SOURCE_SHA256,
  GO_LICENSE_SHA256,
  GO_PATENTS_SHA256,
  SVELTE_SOURCE_SHA256,
  verifyLegalAssets,
} from './verify-legal-assets.mjs'
import { assertNoLocalModuleReplacement } from './go-module-policy.mjs'

export { assertNoLocalModuleReplacement } from './go-module-policy.mjs'

const execFileAsync = promisify(execFile)

export const PUBLIC_REPOSITORY_URL = 'https://github.com/xang1234/chess.git'
export const RELEASE_GO_VERSION = 'go1.26.4'
export const WAILS_MODULE = 'github.com/wailsapp/wails/v2'
export const WAILS_VERSION = 'v2.12.0'

export const LEGAL_ASSETS = Object.freeze([
  'LICENSE.txt',
  'THIRD_PARTY_NOTICES.md',
  'CHESSGROUND_LICENSE.txt',
  'NUNITO_OFL.txt',
])

export const REQUIRED_TRACKED_FILES = Object.freeze([
  'LICENSE',
  'THIRD_PARTY_NOTICES.md',
  'go.mod',
  'go.sum',
  'frontend/package.json',
  'frontend/package-lock.json',
  '.gitignore',
  'README.md',
  'docs/operations/local-build.md',
  'docs/operations/release.md',
  ...LEGAL_ASSETS.map((name) => `frontend/public/legal/${name}`),
  'third_party/runtime-dependencies.lock.json',
  'third_party/legal/go1.26.4/LICENSE',
  'third_party/legal/go1.26.4/PATENTS',
  'third_party/source/chessground-v10.1.1.tar.gz',
  'third_party/source/svelte-v3.59.2.tar.gz',
  'scripts/generate-third-party-notices.mjs',
  'scripts/go-module-policy.mjs',
  'scripts/verify-legal-assets.mjs',
  'scripts/verify-release.mjs',
  'scripts/build-corresponding-source.mjs',
  'scripts/build-release.sh',
])

async function defaultRunner(command, args, options = {}) {
  const { stdout } = await execFileAsync(command, args, {
    cwd: options.cwd,
    env: options.env,
    encoding: options.encoding ?? 'utf8',
    maxBuffer: options.maxBuffer ?? 256 * 1024 * 1024,
  })
  return stdout
}

function defaultPorts() {
  return {
    run: defaultRunner,
    readFile,
    readdir,
    realpath,
    stat,
    mkdtemp,
    rm,
    verifyLegalAssets,
  }
}

function firstStatusLine(status) {
  return status
    .split(/\r?\n/)
    .map((line) => line.trimEnd())
    .find(Boolean)
}

export function assertCleanStatus(status) {
  const first = firstStatusLine(status)
  if (first) {
    throw new Error(`working tree is not clean: ${first.trimStart()}`)
  }
}

export function assertReleaseTag(tag) {
  if (!/^v[0-9]+\.[0-9]+\.[0-9]+$/.test(tag ?? '')) {
    throw new Error('release tag must match v<major>.<minor>.<patch>')
  }
  return tag
}

export function assertFullCommit(commit) {
  const normalized = (commit ?? '').trim()
  if (!/^[0-9a-f]{40}$/.test(normalized)) {
    throw new Error('release commit must be a full 40-character lowercase SHA')
  }
  return normalized
}

function refRecords(output) {
  const records = new Map()
  for (const line of output.split(/\r?\n/)) {
    if (!line.trim()) continue
    const [object, ref, extra] = line.trim().split(/\s+/)
    if (!/^[0-9a-f]{40}$/.test(object) || !ref || extra) {
      throw new Error(`tag query returned malformed record: ${line}`)
    }
    if (records.has(ref) && records.get(ref) !== object) {
      throw new Error(`tag query returned conflicting records for ${ref}`)
    }
    records.set(ref, object)
  }
  return records
}

export function resolveTagCommit(output, tag) {
  assertReleaseTag(tag)
  const records = refRecords(output)
  const direct = records.get(`refs/tags/${tag}`)
  const peeled = records.get(`refs/tags/${tag}^{}`)
  const resolved = peeled ?? direct
  if (!resolved) throw new Error(`tag ${tag} was not found`)
  return resolved
}

export function publicTagQuery(tag, emptyHome, pathValue) {
  assertReleaseTag(tag)
  return {
    command: 'git',
    args: [
      '-c',
      'credential.helper=',
      '-c',
      'http.extraHeader=',
      'ls-remote',
      '--tags',
      PUBLIC_REPOSITORY_URL,
      `refs/tags/${tag}`,
      `refs/tags/${tag}^{}`,
    ],
    options: {
      cwd: emptyHome,
      env: {
        PATH: pathValue,
        GIT_CONFIG_NOSYSTEM: '1',
        GIT_CONFIG_GLOBAL: '/dev/null',
        HOME: emptyHome,
        GIT_TERMINAL_PROMPT: '0',
        GIT_ASKPASS: '/usr/bin/false',
      },
    },
  }
}

export async function verifyPublicTag({
  tag,
  commit,
  runner = defaultRunner,
  pathValue = process.env.PATH,
  emptyHome,
  makeTemporaryHome,
  removeTemporaryHome,
}) {
  let ownedHome
  if (!emptyHome) {
    if (!makeTemporaryHome) {
      ownedHome = await mkdtemp(path.join(process.env.TMPDIR ?? '/tmp', 'chess-trainer-public-git-'))
    } else {
      ownedHome = await makeTemporaryHome()
    }
    emptyHome = ownedHome
  }

  try {
    const query = publicTagQuery(tag, emptyHome, pathValue)
    let output
    try {
      output = await runner(query.command, query.args, query.options)
    } catch (error) {
      throw new Error(
        `public tag ${tag} is not reachable without credentials: ${error.message}`,
      )
    }
    if (!output.trim()) {
      throw new Error(`public tag ${tag} is not reachable without credentials`)
    }
    const resolved = resolveTagCommit(output, tag)
    if (resolved !== commit) {
      throw new Error(`public tag ${tag} does not resolve to HEAD`)
    }
  } finally {
    if (ownedHome) {
      if (removeTemporaryHome) await removeTemporaryHome(ownedHome)
      else await rm(ownedHome, { recursive: true, force: true })
    }
  }
}

export function assertRequiredTrackedFiles(
  tracked,
  { chessgroundVersion } = {},
) {
  if (chessgroundVersion !== undefined && chessgroundVersion !== '10.1.1') {
    throw new Error('Chessground dependency must be pinned exactly to 10.1.1')
  }
  for (const required of REQUIRED_TRACKED_FILES) {
    if (!tracked.has(required)) {
      throw new Error(`required release file is not tracked: ${required}`)
    }
  }
}

export function assertGoToolchain(version) {
  const normalized = (version ?? '').trim()
  if (normalized !== RELEASE_GO_VERSION) {
    throw new Error(`release Go toolchain must be ${RELEASE_GO_VERSION}`)
  }
  return normalized
}

function isBeneath(parent, candidate) {
  const relative = path.relative(parent, candidate)
  return relative !== '' && !relative.startsWith(`..${path.sep}`) && relative !== '..' && !path.isAbsolute(relative)
}

export function assertReleaseEnvironment(env) {
  const rootValue = env.CHESS_TRAINER_RELEASE_ROOT
  if (!rootValue) throw new Error('CHESS_TRAINER_RELEASE_ROOT must be set')
  const root = path.resolve(rootValue)
  for (const name of ['GOMODCACHE', 'GOCACHE', 'npm_config_cache']) {
    if (!env[name]) throw new Error(`${name} must be set`)
    if (!isBeneath(root, path.resolve(env[name]))) {
      throw new Error(`${name} must resolve beneath CHESS_TRAINER_RELEASE_ROOT`)
    }
  }
  if (env.GOWORK !== 'off') throw new Error('GOWORK must be off')
  if (env.GOTOOLCHAIN !== 'local') throw new Error('GOTOOLCHAIN must be local')
  if (env.GOENV !== 'off') throw new Error('GOENV must be off')
  if ((env.GOFLAGS ?? '').trim() !== '') {
    throw new Error('GOFLAGS must be empty for release builds')
  }
  if ((env.NODE_OPTIONS ?? '').trim() !== '') {
    throw new Error('NODE_OPTIONS must be empty for release builds')
  }
  return root
}

async function assertRealReleaseEnvironment(env, ports) {
  const lexicalRoot = assertReleaseEnvironment(env)
  const root = await ports.realpath(lexicalRoot)
  for (const name of ['GOMODCACHE', 'GOCACHE', 'npm_config_cache']) {
    const cache = await ports.realpath(path.resolve(env[name]))
    if (!isBeneath(root, cache)) {
      throw new Error(`${name} must resolve beneath CHESS_TRAINER_RELEASE_ROOT`)
    }
  }
  return root
}

export function assertRuntimeArtifacts({
  committedLock,
  generatedLock,
  committedNotice,
  generatedNotice,
}) {
  if (!Buffer.from(committedLock).equals(Buffer.from(generatedLock))) {
    throw new Error('Darwin runtime dependency lock differs from generated closure')
  }
  if (!Buffer.from(committedNotice).equals(Buffer.from(generatedNotice))) {
    throw new Error('third-party notices differ from generated runtime notices')
  }
}

export function assertDistLegalAssets(publicDocuments, distDocuments) {
  for (const [name, expected] of publicDocuments) {
    const bundled = distDocuments.get(name)
    if (!bundled || !Buffer.from(bundled).equals(Buffer.from(expected))) {
      throw new Error(`bundled legal asset differs: ${name}`)
    }
  }
}

export function assertExecutableContents({
  executable,
  commit,
  legalDocuments,
}) {
  const binary = Buffer.from(executable)
  const exactCommit = assertFullCommit(commit)
  if (!binary.includes(Buffer.from(exactCommit))) {
    throw new Error(`executable does not contain release commit ${exactCommit}`)
  }
  for (const [name, document] of legalDocuments) {
    if (!binary.includes(Buffer.from(document))) {
      throw new Error(`executable does not contain legal document: ${name}`)
    }
  }
}

export function assertExecutableModuleClosure(output, runtimeLock) {
  const executableModules = new Map()
  for (const line of output.split(/\r?\n/)) {
    if (!line.startsWith('\tdep\t')) continue
    const [, , modulePath, version] = line.split('\t')
    if (!modulePath || !version) {
      throw new Error(`go version -m returned malformed dependency: ${line}`)
    }
    if (executableModules.has(modulePath)) {
      throw new Error(`go version -m repeated dependency ${modulePath}`)
    }
    executableModules.set(modulePath, version)
  }

  const lockedModules = new Map(
    (runtimeLock.goModules ?? []).map((module) => [module.path, module.version]),
  )
  for (const [modulePath, version] of lockedModules) {
    const actual = executableModules.get(modulePath)
    if (actual !== version) {
      const detail = actual ? ` (found ${actual})` : ''
      throw new Error(
        `executable module closure is missing ${modulePath}@${version}${detail}`,
      )
    }
  }
  for (const [modulePath, version] of executableModules) {
    if (!lockedModules.has(modulePath)) {
      throw new Error(
        `executable module closure contains unlocked ${modulePath}@${version}`,
      )
    }
  }
}

export async function verifyCodeSignature(app, runner = defaultRunner) {
  try {
    await runner('codesign', ['--verify', '--deep', '--strict', app], {})
  } catch (error) {
    throw new Error(`codesign --verify --deep --strict failed: ${error.message}`)
  }
}

async function readDocuments(root, directory, names, ports) {
  const documents = new Map()
  for (const name of names) {
    try {
      documents.set(name, await ports.readFile(path.join(root, directory, name)))
    } catch (error) {
      if (error.code === 'ENOENT') {
        throw new Error(`required legal asset is missing: ${path.join(directory, name)}`)
      }
      throw error
    }
  }
  return documents
}

function parseNulList(output) {
  return new Set(output.split('\0').filter(Boolean))
}

const GENERATED_INPUT_DIRECTORIES = new Set([
  'build/bin',
  'build/release',
  'frontend/dist',
  'frontend/node_modules',
])

function portablePath(filename) {
  return filename.split(path.sep).join('/')
}

export async function verifyReleaseInputTree({
  repositoryRoot,
  inputRoot,
  tracked,
  ports: providedPorts,
}) {
  const ports = { ...defaultPorts(), ...providedPorts }
  const expectedRoot = path.resolve(repositoryRoot)
  const isolatedRoot = path.resolve(inputRoot)

  for (const relative of tracked) {
    let expected
    let actual
    try {
      ;[expected, actual] = await Promise.all([
        ports.readFile(path.join(expectedRoot, relative)),
        ports.readFile(path.join(isolatedRoot, relative)),
      ])
    } catch (error) {
      if (error.code === 'ENOENT') {
        throw new Error(`isolated build tree is missing tracked input: ${relative}`)
      }
      throw error
    }
    if (!Buffer.from(actual).equals(Buffer.from(expected))) {
      throw new Error(`isolated build input differs from tagged source: ${relative}`)
    }
  }

  async function visit(directory, prefix = '') {
    const entries = await ports.readdir(directory, { withFileTypes: true })
    for (const entry of entries) {
      const relative = portablePath(path.join(prefix, entry.name))
      if (entry.isDirectory()) {
        if (!GENERATED_INPUT_DIRECTORIES.has(relative)) {
          await visit(path.join(directory, entry.name), relative)
        }
      } else if (entry.isFile()) {
        if (!tracked.has(relative)) {
          throw new Error(
            `isolated build tree contains an unexpected input: ${relative}`,
          )
        }
      } else {
        throw new Error(`isolated build tree contains unsupported entry: ${relative}`)
      }
    }
  }

  await visit(isolatedRoot)
}

async function readTrackedDigests(root, tracked, ports) {
  const digests = new Map()
  const { sha256 } = await import('./build-corresponding-source.mjs')
  for (const relative of [...tracked].sort()) {
    try {
      digests.set(relative, sha256(await ports.readFile(path.join(root, relative))))
    } catch (error) {
      if (error.code === 'ENOENT') {
        throw new Error(`tracked application file is missing: ${relative}`)
      }
      throw error
    }
  }
  return digests
}

function parseArguments(argv) {
  const values = {}
  for (let index = 0; index < argv.length; index += 2) {
    const key = argv[index]
    const value = argv[index + 1]
    if (!key?.startsWith('--') || value === undefined) {
      throw new Error('usage: verify-release.mjs --phase pre|post --tag <tag> --input-root <path> [--app <path> --source <archive>]')
    }
    values[key.slice(2)] = value
  }
  if (!['pre', 'post'].includes(values.phase)) {
    throw new Error('--phase must be pre or post')
  }
  assertReleaseTag(values.tag)
  if (!values['input-root']) throw new Error('--input-root is required')
  if (values.phase === 'post' && (!values.app || !values.source)) {
    throw new Error('post verification requires --app and --source')
  }
  return values
}

export async function verifyRelease({
  root,
  phase,
  tag,
  app,
  source,
  inputRoot,
  env = process.env,
  ports: providedPorts,
}) {
  const ports = { ...defaultPorts(), ...providedPorts }
  const absoluteRoot = path.resolve(root)
  const releaseRoot = await assertRealReleaseEnvironment(env, ports)
  if (!inputRoot) throw new Error('release verification requires an isolated --input-root')
  const buildRoot = await ports.realpath(path.resolve(inputRoot))
  if (!isBeneath(releaseRoot, buildRoot)) {
    throw new Error('isolated release input must resolve beneath CHESS_TRAINER_RELEASE_ROOT')
  }
  assertReleaseTag(tag)

  const status = await ports.run(
    'git',
    ['status', '--porcelain=v1', '--untracked-files=all'],
    { cwd: absoluteRoot, env },
  )
  assertCleanStatus(status)

  const commit = assertFullCommit(
    await ports.run('git', ['rev-parse', 'HEAD'], { cwd: absoluteRoot, env }),
  )
  const localRefs = await ports.run(
    'git',
    ['show-ref', '--tags', '--dereference'],
    { cwd: absoluteRoot, env },
  )
  if (resolveTagCommit(localRefs, tag) !== commit) {
    throw new Error(`local tag ${tag} does not resolve to HEAD`)
  }

  await verifyPublicTag({
    tag,
    commit,
    runner: ports.run,
    pathValue: env.PATH,
    makeTemporaryHome: () =>
      ports.mkdtemp(path.join(releaseRoot, 'credential-free-git-home-')),
    removeTemporaryHome: (directory) =>
      ports.rm(directory, { recursive: true, force: true }),
  })

  const tracked = parseNulList(
    await ports.run('git', ['ls-files', '-z'], {
      cwd: absoluteRoot,
      env,
      encoding: 'utf8',
    }),
  )
  await verifyReleaseInputTree({
    repositoryRoot: absoluteRoot,
    inputRoot: buildRoot,
    tracked,
    ports,
  })
  const packageJSON = JSON.parse(
    await ports.readFile(path.join(buildRoot, 'frontend/package.json'), 'utf8'),
  )
  assertRequiredTrackedFiles(tracked, {
    chessgroundVersion:
      packageJSON.dependencies?.['@lichess-org/chessground'],
  })

  const goMod = await ports.readFile(path.join(buildRoot, 'go.mod'), 'utf8')
  assertNoLocalModuleReplacement(goMod)
  assertGoToolchain(
    await ports.run('go', ['env', 'GOVERSION'], { cwd: buildRoot, env }),
  )
  await ports.run('go', ['mod', 'verify'], { cwd: buildRoot, env })

  try {
    await ports.verifyLegalAssets({ root: buildRoot })
  } catch (error) {
    throw new Error(`legal input verification failed: ${error.message}`)
  }

  if (phase === 'pre') return { commit, tracked }
  if (phase !== 'post') throw new Error('release phase must be pre or post')
  if (!app || !source) throw new Error('post verification requires app and source')

  const publicDocuments = await readDocuments(
    buildRoot,
    'frontend/public/legal',
    LEGAL_ASSETS,
    ports,
  )
  const distDocuments = await readDocuments(
    buildRoot,
    'frontend/dist/legal',
    LEGAL_ASSETS,
    ports,
  )
  assertDistLegalAssets(publicDocuments, distDocuments)

  const legalDocuments = new Map(publicDocuments)
  legalDocuments.set(
    'Go LICENSE',
    await ports.readFile(path.join(buildRoot, 'third_party/legal/go1.26.4/LICENSE')),
  )
  legalDocuments.set(
    'Go PATENTS',
    await ports.readFile(path.join(buildRoot, 'third_party/legal/go1.26.4/PATENTS')),
  )
  const executableName = JSON.parse(
    await ports.readFile(path.join(buildRoot, 'wails.json'), 'utf8'),
  ).outputfilename
  const executablePath = path.join(path.resolve(app), 'Contents/MacOS', executableName)
  assertExecutableContents({
    executable: await ports.readFile(executablePath),
    commit,
    legalDocuments,
  })
  await verifyCodeSignature(path.resolve(app), ports.run)

  const {
    parseGoDownload,
    sha256,
    treeDigest,
    verifyCorrespondingSourceArchive,
  } = await import('./build-corresponding-source.mjs')
  const wailsDownload = await parseGoDownload(
    WAILS_MODULE,
    WAILS_VERSION,
    ports.run,
    { cwd: buildRoot, env },
  )
  const trackedFiles = await readTrackedDigests(buildRoot, tracked, ports)
  const runtimeLockContent = await ports.readFile(
    path.join(buildRoot, 'third_party/runtime-dependencies.lock.json'),
  )
  const runtimeLock = JSON.parse(runtimeLockContent.toString('utf8'))
  assertExecutableModuleClosure(
    await ports.run('go', ['version', '-m', executablePath], {
      cwd: buildRoot,
      env,
    }),
    runtimeLock,
  )
  await verifyCorrespondingSourceArchive({
    archive: path.resolve(source),
    temporaryRoot: releaseRoot,
    tag,
    commit,
    trackedFiles,
    runtimeLock,
    runtimeLockSha256: sha256(runtimeLockContent),
    goLegal: {
      LICENSE: GO_LICENSE_SHA256,
      PATENTS: GO_PATENTS_SHA256,
    },
    wailsTreeSha256: await treeDigest(wailsDownload.Dir),
    frontendSources: new Map([
      [
        'app/third_party/source/chessground-v10.1.1.tar.gz',
        CHESSGROUND_SOURCE_SHA256,
      ],
      [
        'app/third_party/source/svelte-v3.59.2.tar.gz',
        SVELTE_SOURCE_SHA256,
      ],
    ]),
    ports,
  })
  return { commit, tracked }
}

const isCLI =
  process.argv[1] &&
  path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)

if (isCLI) {
  try {
    const values = parseArguments(process.argv.slice(2))
    await verifyRelease({
      root: path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..'),
      phase: values.phase,
      tag: values.tag,
      app: values.app,
      source: values.source,
      inputRoot: values['input-root'],
    })
  } catch (error) {
    console.error(error.message)
    process.exitCode = 1
  }
}
