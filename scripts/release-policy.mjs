import { execFile } from 'node:child_process'
import { mkdtemp, realpath, rm } from 'node:fs/promises'
import path from 'node:path'
import { promisify } from 'node:util'

const execFileAsync = promisify(execFile)

export const PUBLIC_REPOSITORY_URL = 'https://github.com/xang1234/chess.git'
export const RELEASE_GO_VERSION = 'go1.26.4'
export const RELEASE_PLATFORM = 'darwin/arm64'
export const RELEASE_BUNDLE_IDENTIFIER = 'com.xang1234.chesstrainer'
export const WAILS_MODULE = 'github.com/wailsapp/wails/v2'
export const WAILS_VERSION = 'v2.12.0'

export const RELEASE_TARGET_ENVIRONMENT_VARIABLES = Object.freeze([
  'GOOS',
  'GOARCH',
  'GOAMD64',
  'GOARM64',
  'GOFIPS140',
  'GOEXPERIMENT',
  'SDKROOT',
  'DEVELOPER_DIR',
  'MACOSX_DEPLOYMENT_TARGET',
])

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
  'wails.json',
  'build/darwin/Info.plist',
  'build/darwin/Info.dev.plist',
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
  'scripts/release-policy.mjs',
  'scripts/tree-integrity.mjs',
  'scripts/deterministic-tar.mjs',
  'scripts/corresponding-source-verifier.mjs',
  'scripts/corresponding-source-builder.mjs',
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

export function releaseVersionFromTag(tag) {
  return assertReleaseTag(tag).slice(1)
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
      ownedHome = await mkdtemp(
        path.join(process.env.TMPDIR ?? '/tmp', 'chess-trainer-public-git-'),
      )
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

export function assertCourseFixtureBoundary(paths) {
  for (const filename of paths) {
    if (!filename.endsWith('.ctcourse')) continue
    if (!filename.startsWith('internal/openings/testdata/')) {
      throw new Error(`private opening course must not be tracked: ${filename}`)
    }
  }
}

export function assertRequiredTrackedFiles(
  tracked,
  { chessgroundVersion } = {},
) {
  assertCourseFixtureBoundary(tracked)
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

export function isBeneath(parent, candidate) {
  const relative = path.relative(parent, candidate)
  return relative !== '' &&
    !relative.startsWith(`..${path.sep}`) &&
    relative !== '..' &&
    !path.isAbsolute(relative)
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
  for (const name of RELEASE_TARGET_ENVIRONMENT_VARIABLES) {
    if ((env[name] ?? '').trim() !== '') throw new Error(`${name} must be unset`)
  }
  return root
}

export async function assertRealReleaseEnvironment(
  env,
  ports = { realpath },
) {
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

export async function parseGoDownload(
  modulePath,
  version,
  runner = defaultRunner,
  options = {},
) {
  const output = await runner(
    'go',
    ['mod', 'download', '-json', `${modulePath}@${version}`],
    options,
  )
  let metadata
  try {
    metadata = JSON.parse(output)
  } catch (error) {
    throw new Error(`go mod download returned invalid JSON for ${modulePath}@${version}`)
  }
  if (metadata.Error) {
    throw new Error(`cannot download ${modulePath}@${version}: ${metadata.Error}`)
  }
  if (
    metadata.Path !== modulePath ||
    metadata.Version !== version ||
    !metadata.Dir
  ) {
    throw new Error(
      `go mod download returned incomplete metadata for ${modulePath}@${version}`,
    )
  }
  return metadata
}
