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
import { verifyCorrespondingSourceArchive } from './corresponding-source-verifier.mjs'
import {
  LEGAL_ASSETS,
  PUBLIC_REPOSITORY_URL,
  RELEASE_BUNDLE_IDENTIFIER,
  RELEASE_GO_VERSION,
  RELEASE_PLATFORM,
  REQUIRED_TRACKED_FILES,
  WAILS_MODULE,
  WAILS_VERSION,
  assertCleanStatus,
  assertCourseFixtureBoundary,
  assertFullCommit,
  assertGoToolchain,
  assertRealReleaseEnvironment,
  assertReleaseEnvironment,
  assertReleaseTag,
  assertRequiredTrackedFiles,
  isBeneath,
  parseGoDownload,
  publicTagQuery,
  releaseVersionFromTag,
  resolveTagCommit,
  verifyPublicTag,
} from './release-policy.mjs'
import { sha256, treeDigest } from './tree-integrity.mjs'

export { assertNoLocalModuleReplacement } from './go-module-policy.mjs'
export {
  LEGAL_ASSETS,
  PUBLIC_REPOSITORY_URL,
  RELEASE_BUNDLE_IDENTIFIER,
  RELEASE_GO_VERSION,
  RELEASE_PLATFORM,
  REQUIRED_TRACKED_FILES,
  WAILS_MODULE,
  WAILS_VERSION,
  assertCleanStatus,
  assertCourseFixtureBoundary,
  assertFullCommit,
  assertGoToolchain,
  assertReleaseEnvironment,
  assertReleaseTag,
  assertRequiredTrackedFiles,
  publicTagQuery,
  releaseVersionFromTag,
  resolveTagCommit,
  verifyPublicTag,
}

const execFileAsync = promisify(execFile)

async function defaultRunner(command, args, options = {}) {
  const { stdout, stderr } = await execFileAsync(command, args, {
    cwd: options.cwd,
    env: options.env,
    encoding: options.encoding ?? 'utf8',
    maxBuffer: options.maxBuffer ?? 256 * 1024 * 1024,
  })
  return options.includeStderr ? `${stdout}${stderr}` : stdout
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

export function assertExecutableTarget({ architectures, buildInfo }) {
  const exactArchitectures = (architectures ?? '').trim().split(/\s+/).filter(Boolean)
  if (exactArchitectures.length !== 1 || exactArchitectures[0] !== 'arm64') {
    throw new Error('release Mach-O architecture must be arm64')
  }
  const settings = new Map()
  for (const line of (buildInfo ?? '').split(/\r?\n/)) {
    const match = /^\tbuild\t([^=]+)=(.*)$/.exec(line)
    if (match) settings.set(match[1], match[2])
  }
  for (const [name, expected] of [
    ['GOOS', 'darwin'],
    ['GOARCH', 'arm64'],
    ['GOARM64', 'v8.0'],
  ]) {
    if (settings.get(name) !== expected) {
      throw new Error(`release Go build setting ${name} must be ${expected}`)
    }
  }
}

function plistString(plist, key) {
  const text = Buffer.from(plist).toString('utf8')
  const match = new RegExp(
    `<key>\\s*${key}\\s*<\\/key>\\s*<string>([^<]+)<\\/string>`,
  ).exec(text)
  if (!match) throw new Error(`release app ${key} is missing`)
  return match[1]
}

export function assertBundleIdentifier(plist) {
  const identifier = plistString(plist, 'CFBundleIdentifier')
  if (!/^[A-Za-z0-9.-]+$/.test(identifier)) {
    throw new Error(
      'release app bundle identifier may contain only letters, numbers, hyphens, and periods',
    )
  }
  if (identifier !== RELEASE_BUNDLE_IDENTIFIER) {
    throw new Error(`release app bundle identifier must be ${RELEASE_BUNDLE_IDENTIFIER}`)
  }
  return identifier
}

export function assertBundleMetadata(plist, tag) {
  assertBundleIdentifier(plist)
  const expected = releaseVersionFromTag(tag)
  for (const key of ['CFBundleShortVersionString', 'CFBundleVersion']) {
    const actual = plistString(plist, key)
    if (actual !== expected) {
      throw new Error(`release app ${key} must be ${expected}`)
    }
  }
  return expected
}

export async function verifyCodeSignature(app, runner = defaultRunner) {
  try {
    await runner('codesign', ['--verify', '--deep', '--strict', app], {})
  } catch (error) {
    throw new Error(`codesign --verify --deep --strict failed: ${error.message}`)
  }
  let details
  try {
    details = await runner(
      'codesign',
      ['--display', '--verbose=4', app],
      { includeStderr: true },
    )
  } catch (error) {
    throw new Error(`codesign signature inspection failed: ${error.message}`)
  }
  if (!/^Authority=Developer ID Application:/m.test(details) ||
    !/^TeamIdentifier=(?!not set$)[A-Z0-9]+$/m.test(details)) {
    throw new Error('release app must have a Developer ID Application signature')
  }
  if (!/^Runtime Version=/m.test(details) && !/^flags=.*\bruntime\b/m.test(details)) {
    throw new Error('release app signature must enable the hardened runtime')
  }
  if (!/^Timestamp=(?!none$).+$/m.test(details)) {
    throw new Error('release app signature must include a trusted timestamp')
  }
  try {
    await runner('xcrun', ['stapler', 'validate', app], {})
  } catch (error) {
    throw new Error(`notarization staple validation failed: ${error.message}`)
  }
  try {
    await runner('spctl', ['--assess', '--type', 'execute', '--verbose=4', app], {})
  } catch (error) {
    throw new Error(`Gatekeeper assessment failed: ${error.message}`)
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
  assertBundleMetadata(
    await ports.readFile(path.join(path.resolve(app), 'Contents/Info.plist')),
    tag,
  )
  assertExecutableContents({
    executable: await ports.readFile(executablePath),
    commit,
    legalDocuments,
  })
  await verifyCodeSignature(path.resolve(app), ports.run)

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
  const executableBuildInfo = await ports.run(
    'go',
    ['version', '-m', executablePath],
    { cwd: buildRoot, env },
  )
  assertExecutableModuleClosure(executableBuildInfo, runtimeLock)
  assertExecutableTarget({
    architectures: await ports.run('lipo', ['-archs', executablePath], {
      cwd: buildRoot,
      env,
    }),
    buildInfo: executableBuildInfo,
  })
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
