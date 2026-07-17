import assert from 'node:assert/strict'
import { execFile } from 'node:child_process'
import {
  chmod,
  cp,
  mkdir,
  mkdtemp,
  readFile,
  rm,
  stat,
  symlink,
  writeFile,
} from 'node:fs/promises'
import { tmpdir } from 'node:os'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'
import { promisify } from 'node:util'

const execFileAsync = promisify(execFile)
const repositoryRoot = path.resolve(
  path.dirname(fileURLToPath(import.meta.url)),
  '..',
)
const wrapperSource = path.join(repositoryRoot, 'scripts/build-release.sh')
const commit = '0123456789abcdef0123456789abcdef01234567'

async function exists(filename) {
  try {
    await stat(filename)
    return true
  } catch (error) {
    if (error.code === 'ENOENT') return false
    throw error
  }
}

async function wrapperFixture() {
  const root = await mkdtemp(path.join(tmpdir(), 'release-wrapper-test-'))
  const bin = path.join(root, 'bin')
  const scripts = path.join(root, 'scripts')
  const wrapper = path.join(scripts, 'build-release.sh')
  const log = path.join(root, 'commands.log')
  const sharedCache = path.join(root, 'shared-go-mod-cache')
  const credentialHome = path.join(root, 'credential-home')
  await mkdir(bin)
  await mkdir(scripts)
  await cp(wrapperSource, wrapper)
  await chmod(wrapper, 0o755)
  await mkdir(sharedCache)
  await mkdir(credentialHome)
  await writeFile(path.join(sharedCache, 'poisoned-module'), 'must not be used\n')

  const shim = path.join(bin, 'command-shim')
  await writeFile(
    shim,
    `#!/bin/sh
set -eu
name=$(basename "$0")
if [ -n "\${GOOS-}\${GOARCH-}\${GOAMD64-}\${GOARM64-}\${GOFIPS140-}\${GOEXPERIMENT-}\${SDKROOT-}\${DEVELOPER_DIR-}\${MACOSX_DEPLOYMENT_TARGET-}" ]; then
  printf 'target environment was not sanitized\n' >&2
  exit 43
fi
count_files() {
  find "$1" -mindepth 1 -print 2>/dev/null | wc -l | tr -d ' '
}
printf '%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s\\n' \\
  "$name" \\
  "\${CHESS_TRAINER_RELEASE_ROOT:?}" \\
  "\${HOME:?}" \\
  "\${GOMODCACHE:?}" \\
  "\${GOCACHE:?}" \\
  "\${npm_config_cache:?}" \\
  "\${GOWORK:?}" \\
  "\${GOTOOLCHAIN:?}" \\
  "\${GOENV:?}" \\
  "\${GOFLAGS-}" \\
  "\${NODE_OPTIONS-}" \\
  "$(count_files "$GOMODCACHE")" \\
  "$(count_files "$GOCACHE")" \\
  "$(count_files "$npm_config_cache")" \\
  "$(pwd)" \\
  "$*" >> "\${WRAPPER_LOG:?}"
if [ "\${FAIL_COMMAND:-}" = "$name" ]; then
  exit 42
fi
if [ "$name" = git ] && [ "\${1:-}" = rev-parse ] && [ "\${2:-}" = HEAD ]; then
  printf '%s\\n' "\${FIXTURE_COMMIT:?}"
fi
if [ "$name" = go ] && [ "\${1:-}" = env ] && [ "\${2:-}" = GOVERSION ]; then
  printf 'go1.26.4\\n'
fi
if [ "$name" = go ] && [ "\${1:-}" = mod ] && [ "\${2:-}" = download ]; then
  mkdir -p "$GOMODCACHE/read-only-module/nested"
  printf 'module source\\n' > "$GOMODCACHE/read-only-module/nested/source.go"
  chmod 0444 "$GOMODCACHE/read-only-module/nested/source.go"
  chmod 0555 "$GOMODCACHE/read-only-module/nested" "$GOMODCACHE/read-only-module"
fi
if [ "$name" = ditto ]; then
  if [ "\${1:-}" = -c ]; then
    for destination do :; done
    mkdir -p "$(dirname "$destination")"
    : > "$destination"
  else
    mkdir -p "\${2:?}"
  fi
fi
`,
  )
  await chmod(shim, 0o755)
  for (const name of [
    'git', 'go', 'npm', 'node', 'tar', 'ditto', 'plutil', 'codesign', 'xcrun', 'spctl',
  ]) {
    await symlink('command-shim', path.join(bin, name))
  }

  return { root, bin, credentialHome, log, sharedCache, wrapper }
}

async function runWrapper(fixture, extraEnv = {}) {
  return execFileAsync('/bin/bash', [fixture.wrapper, 'v1.2.3'], {
    cwd: fixture.root,
    env: {
      ...process.env,
      PATH: `${fixture.bin}:/usr/bin:/bin`,
      HOME: fixture.credentialHome,
      WRAPPER_LOG: fixture.log,
      FIXTURE_COMMIT: commit,
      GOMODCACHE: fixture.sharedCache,
      CHESS_TRAINER_SIGNING_IDENTITY: 'Developer ID Application: Chess Trainer (TEAM123456)',
      CHESS_TRAINER_NOTARY_PROFILE: 'ChessTrainerNotary',
      GOOS: 'linux',
      GOARCH: 'amd64',
      GOAMD64: 'v3',
      GOARM64: 'v9.4',
      GOFIPS140: 'latest',
      SDKROOT: '/poisoned/sdk',
      DEVELOPER_DIR: '/poisoned/xcode',
      MACOSX_DEPLOYMENT_TARGET: '99.0',
      ...extraEnv,
    },
  })
}

function parseLog(text) {
  return text
    .trim()
    .split('\n')
    .filter(Boolean)
    .map((line) => {
      const [
        command,
        releaseRoot,
        home,
        goModCache,
        goCache,
        npmCache,
        goWork,
        goToolchain,
        goEnv,
        goFlags,
        nodeOptions,
        goModFiles,
        goCacheFiles,
        npmCacheFiles,
        cwd,
        args,
      ] = line.split('|')
      return {
        command,
        releaseRoot,
        home,
        goModCache,
        goCache,
        npmCache,
        goWork,
        goToolchain,
        goEnv,
        goFlags,
        nodeOptions,
        goModFiles,
        goCacheFiles,
        npmCacheFiles,
        cwd,
        args,
      }
    })
}

test('runs the supported release pipeline in isolated empty caches and cleans up', async () => {
  const fixture = await wrapperFixture()
  try {
    await runWrapper(fixture)
    const calls = parseLog(await readFile(fixture.log, 'utf8'))
    assert.ok(calls.length >= 10)

    const roots = new Set(calls.map((call) => call.releaseRoot))
    assert.equal(roots.size, 1)
    const [releaseRoot] = roots
    assert.notEqual(releaseRoot, fixture.sharedCache)
    assert.equal(await exists(releaseRoot), false, 'release root must be removed')

    for (const call of calls) {
      assert.ok(call.goModCache.startsWith(`${releaseRoot}/`))
      assert.ok(call.goCache.startsWith(`${releaseRoot}/`))
      assert.ok(call.npmCache.startsWith(`${releaseRoot}/`))
      assert.notEqual(call.goModCache, fixture.sharedCache)
      assert.equal(call.goWork, 'off')
      assert.equal(call.goToolchain, 'local')
      assert.equal(call.goEnv, 'off')
      assert.equal(call.goFlags, '')
      assert.equal(call.nodeOptions, '')
    }
    assert.equal(calls[0].goModFiles, '0')
    assert.equal(calls[0].goCacheFiles, '0')
    assert.equal(calls[0].npmCacheFiles, '0')
    assert.ok(calls.some((call) => Number(call.goModFiles) > 0))

    const isolatedHome = path.join(releaseRoot, 'home')
    for (const call of calls) {
      const usesAppleCredentials =
        call.command === 'codesign' ||
        (call.command === 'xcrun' && call.args.startsWith('notarytool submit '))
      assert.equal(
        path.resolve(call.home),
        path.resolve(
          usesAppleCredentials ? fixture.credentialHome : isolatedHome,
        ),
        `${call.command} ${call.args} received the wrong HOME`,
      )
    }

    const commands = calls.map((call) => `${call.command} ${call.args}`)
    assert.ok(
      commands.some((line) =>
        /git archive --format=tar --output .* [0-9a-f]{40}$/.test(line),
      ),
    )
    assert.ok(commands.some((line) => /tar -xf .* -C .*/.test(line)))
    assert.ok(commands.some((line) => /npm --prefix frontend ci$/.test(line)))
    assert.ok(commands.some((line) => /npm --prefix frontend run build$/.test(line)))
    assert.ok(commands.some((line) => /go mod download all$/.test(line)))
    assert.ok(commands.some((line) => /go mod verify$/.test(line)))
    assert.ok(
      commands.some((line) =>
        /node .*scripts\/verify-release\.mjs --phase pre --tag v1\.2\.3 --input-root .*/.test(line),
      ),
    )
    assert.ok(
      commands.some((line) =>
        line.includes(
          `go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build -clean -trimpath -platform darwin/arm64 -m -nosyncgomod -ldflags -X chess-trainer/internal/buildinfo.Commit=${commit}`,
        ),
      ),
    )
    assert.ok(
      commands.some((line) =>
        /node .*scripts\/build-corresponding-source\.mjs --tag v1\.2\.3 --commit [0-9a-f]{40} --output .*build\/release\/Chess-Trainer-v1\.2\.3-corresponding-source\.tar\.gz$/.test(line),
      ),
    )
    assert.ok(
      commands.some((line) =>
        /node .*scripts\/verify-release\.mjs --phase post .* --input-root .* --source .*build\/release\/Chess-Trainer-v1\.2\.3-corresponding-source\.tar\.gz$/.test(line),
      ),
    )
    const wailsBuild = calls.find(
      (call) =>
        call.command === 'go' &&
        call.args.startsWith('run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build'),
    )
    assert.ok(wailsBuild)
    assert.equal(path.basename(wailsBuild.cwd), 'app-source')
    assert.notEqual(wailsBuild.cwd, fixture.root)
    const shortVersion = commands.findIndex((line) =>
      line.includes(
        'plutil -replace CFBundleShortVersionString -string 1.2.3 ',
      ),
    )
    const bundleVersion = commands.findIndex((line) =>
      line.includes('plutil -replace CFBundleVersion -string 1.2.3 '),
    )
    const codeSign = commands.findIndex((line) => line.startsWith('codesign '))
    assert.ok(shortVersion >= 0)
    assert.ok(bundleVersion >= 0)
    assert.ok(codeSign > shortVersion)
    assert.ok(codeSign > bundleVersion)
    assert.ok(commands.some((line) => line.startsWith('ditto ')))
    assert.ok(commands.some((line) =>
      line.includes('codesign --force --options runtime --timestamp --sign Developer ID Application: Chess Trainer (TEAM123456)'),
    ))
    assert.ok(commands.some((line) =>
      /xcrun notarytool submit .*macOS-arm64-notary\.zip --keychain-profile ChessTrainerNotary --wait$/.test(line),
    ))
    assert.ok(commands.some((line) => line.startsWith('xcrun stapler staple ')))
    assert.ok(commands.some((line) => line.startsWith('xcrun stapler validate ')))
    assert.ok(commands.some((line) => line.startsWith('spctl --assess --type execute --verbose=4 ')))
    assert.ok(commands.some((line) =>
      /ditto -c -k --keepParent .*Chess Trainer\.app .*Chess-Trainer-v1\.2\.3-macOS-arm64\.zip$/.test(line),
    ))
  } finally {
    await rm(fixture.root, { recursive: true, force: true })
  }
})

test('refuses a public release without signing and notarization credentials', async () => {
  const fixture = await wrapperFixture()
  try {
    await assert.rejects(
      runWrapper(fixture, {
        CHESS_TRAINER_SIGNING_IDENTITY: '',
        CHESS_TRAINER_NOTARY_PROFILE: '',
      }),
      /CHESS_TRAINER_SIGNING_IDENTITY/,
    )
  } finally {
    await rm(fixture.root, { recursive: true, force: true })
  }
})

test('uses a unique release root for each invocation', async () => {
  const first = await wrapperFixture()
  const second = await wrapperFixture()
  try {
    await runWrapper(first)
    await runWrapper(second)
    const firstRoot = parseLog(await readFile(first.log, 'utf8'))[0].releaseRoot
    const secondRoot = parseLog(await readFile(second.log, 'utf8'))[0].releaseRoot
    assert.notEqual(firstRoot, secondRoot)
  } finally {
    await rm(first.root, { recursive: true, force: true })
    await rm(second.root, { recursive: true, force: true })
  }
})

test('removes the release root after an injected subprocess failure', async () => {
  const fixture = await wrapperFixture()
  try {
    await assert.rejects(
      runWrapper(fixture, { FAIL_COMMAND: 'node' }),
      (error) => error.code === 42,
    )
    const calls = parseLog(await readFile(fixture.log, 'utf8'))
    assert.ok(calls.length > 0)
    assert.equal(await exists(calls[0].releaseRoot), false)
  } finally {
    await rm(fixture.root, { recursive: true, force: true })
  }
})
