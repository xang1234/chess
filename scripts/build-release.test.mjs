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
  await mkdir(bin)
  await mkdir(scripts)
  await cp(wrapperSource, wrapper)
  await chmod(wrapper, 0o755)
  await mkdir(sharedCache)
  await writeFile(path.join(sharedCache, 'poisoned-module'), 'must not be used\n')

  const shim = path.join(bin, 'command-shim')
  await writeFile(
    shim,
    `#!/bin/sh
set -eu
name=$(basename "$0")
count_files() {
  find "$1" -mindepth 1 -print 2>/dev/null | wc -l | tr -d ' '
}
printf '%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s|%s\\n' \\
  "$name" \\
  "\${CHESS_TRAINER_RELEASE_ROOT:?}" \\
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
  mkdir -p "\${2:?}"
fi
`,
  )
  await chmod(shim, 0o755)
  for (const name of ['git', 'go', 'npm', 'node', 'tar', 'ditto']) {
    await symlink('command-shim', path.join(bin, name))
  }

  return { root, bin, log, sharedCache, wrapper }
}

async function runWrapper(fixture, extraEnv = {}) {
  return execFileAsync('/bin/bash', [fixture.wrapper, 'v1.2.3'], {
    cwd: fixture.root,
    env: {
      ...process.env,
      PATH: `${fixture.bin}:/usr/bin:/bin`,
      WRAPPER_LOG: fixture.log,
      FIXTURE_COMMIT: commit,
      GOMODCACHE: fixture.sharedCache,
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
          `go run github.com/wailsapp/wails/v2/cmd/wails@v2.12.0 build -clean -trimpath -ldflags -X chess-trainer/internal/buildinfo.Commit=${commit}`,
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
    assert.ok(commands.some((line) => line.startsWith('ditto ')))
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
