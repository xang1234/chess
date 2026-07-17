import assert from 'node:assert/strict'
import {
  chmod,
  mkdir,
  mkdtemp,
  readFile,
  rm,
  unlink,
  writeFile,
} from 'node:fs/promises'
import { tmpdir } from 'node:os'
import path from 'node:path'
import test from 'node:test'

import {
  buildCorrespondingSource,
  escapeModulePath,
  makeTreeWritable,
  parseGoDownload,
  renderTrackedFiles,
  renderBuilding,
  sha256,
  treeDigest,
  verifyCorrespondingSourceTree,
} from './build-corresponding-source.mjs'

const tag = 'v1.2.3'
const commit = '0123456789abcdef0123456789abcdef01234567'

async function put(filename, content) {
  await mkdir(path.dirname(filename), { recursive: true })
  await writeFile(filename, content)
}

async function completeFixture() {
  const root = await mkdtemp(path.join(tmpdir(), 'corresponding-source-'))
  const app = path.join(root, 'app')
  const modulePath = 'example.com/production'
  const moduleVersion = 'v1.2.3'
  const moduleLicense = Buffer.from('module license\n')
  const moduleNotice = Buffer.from('module notice\n')
  const goLicense = Buffer.from('Go license\n')
  const goPatents = Buffer.from('Go patents\n')
  const chessgroundSource = Buffer.from('complete Chessground source archive')
  const svelteSource = Buffer.from('complete Svelte source archive')
  const goMod = [
    'module example.com/chess-trainer',
    '',
    'go 1.25.0',
    '',
    `require ${modulePath} ${moduleVersion}`,
    '',
  ].join('\n')
  const runtimeLock = {
    formatVersion: 1,
    goToolchain: {
      version: 'go1.26.4',
      source: 'third_party/legal/go1.26.4',
      legal: [
        { name: 'LICENSE', sha256: sha256(goLicense) },
        { name: 'PATENTS', sha256: sha256(goPatents) },
      ],
    },
    goModules: [
      {
        path: modulePath,
        version: moduleVersion,
        legal: [
          { name: 'LICENSE', sha256: sha256(moduleLicense) },
          { name: 'NOTICE', sha256: sha256(moduleNotice) },
        ],
      },
    ],
    frontend: [],
  }

  await put(path.join(app, 'go.mod'), goMod)
  await put(path.join(app, 'main.go'), 'package main\n')
  await put(
    path.join(app, 'third_party/runtime-dependencies.lock.json'),
    `${JSON.stringify(runtimeLock, null, 2)}\n`,
  )
  await put(
    path.join(app, 'third_party/source/chessground-v10.1.1.tar.gz'),
    chessgroundSource,
  )
  await put(
    path.join(app, 'third_party/source/svelte-v3.59.2.tar.gz'),
    svelteSource,
  )

  await put(
    path.join(app, 'vendor/modules.txt'),
    `# ${modulePath} ${moduleVersion}\n## explicit; go 1.25.0\n${modulePath}\n`,
  )
  await put(
    path.join(app, 'vendor', modulePath, 'production.go'),
    'package production\n',
  )
  const legalDirectory = path.join(
    app,
    'vendor/_licenses',
    `${escapeModulePath(modulePath)}@${moduleVersion}`,
  )
  await put(path.join(legalDirectory, 'LICENSE'), moduleLicense)
  await put(path.join(legalDirectory, 'NOTICE'), moduleNotice)

  await put(path.join(root, 'toolchain/go1.26.4/LICENSE'), goLicense)
  await put(path.join(root, 'toolchain/go1.26.4/PATENTS'), goPatents)

  const wailsRoot = path.join(root, 'build-tools/wails-v2.12.0')
  await put(path.join(wailsRoot, 'go.mod'), 'module github.com/wailsapp/wails/v2\n')
  await put(path.join(wailsRoot, 'LICENSE'), 'Wails license\n')
  await put(path.join(wailsRoot, 'cmd/wails/main.go'), 'package main\n')

  const building = [
    '# Building Chess Trainer from corresponding source',
    '',
    'Use the included Go 1.26.4 release toolchain legal files.',
    'Build the Wails CLI from build-tools/wails-v2.12.0/cmd/wails.',
    'Use app/vendor with GOFLAGS=-mod=vendor and GOFLAGS=-mod=mod.',
    'Run npm --prefix frontend ci and npm --prefix frontend run build.',
    'Keep GOWORK=off and GOTOOLCHAIN=local.',
    '',
  ].join('\n')
  await put(path.join(root, 'BUILDING.md'), building)

  const trackedFiles = new Map()
  for (const relative of [
    'go.mod',
    'main.go',
    'third_party/runtime-dependencies.lock.json',
    'third_party/source/chessground-v10.1.1.tar.gz',
    'third_party/source/svelte-v3.59.2.tar.gz',
  ]) {
    trackedFiles.set(relative, sha256(await readFile(path.join(app, relative))))
  }
  const trackedText = renderTrackedFiles(trackedFiles)
  await put(path.join(root, 'TRACKED_FILES.sha256'), trackedText)

  const moduleSourceRoot = path.join(app, 'vendor', modulePath)
  const vendorRoot = path.join(app, 'vendor')
  const runtimeLockContent = await readFile(
    path.join(app, 'third_party/runtime-dependencies.lock.json'),
  )
  const manifest = {
    formatVersion: 1,
    tag,
    commit,
    toolchain: {
      version: 'go1.26.4',
      legal: {
        LICENSE: sha256(goLicense),
        PATENTS: sha256(goPatents),
      },
    },
    trackedFiles: {
      path: 'TRACKED_FILES.sha256',
      count: trackedFiles.size,
      sha256: sha256(trackedText),
    },
    runtimeDependenciesLock: {
      path: 'app/third_party/runtime-dependencies.lock.json',
      sha256: sha256(runtimeLockContent),
    },
    goVendor: {
      path: 'app/vendor',
      treeSha256: await treeDigest(vendorRoot),
      modules: [
        {
          path: modulePath,
          version: moduleVersion,
          sourceTreeSha256: await treeDigest(moduleSourceRoot),
          legalTreeSha256: await treeDigest(legalDirectory),
        },
      ],
    },
    wails: {
      module: 'github.com/wailsapp/wails/v2',
      version: 'v2.12.0',
      path: 'build-tools/wails-v2.12.0',
      treeSha256: await treeDigest(wailsRoot),
    },
    frontendSources: [
      {
        name: '@lichess-org/chessground',
        version: '10.1.1',
        path: 'app/third_party/source/chessground-v10.1.1.tar.gz',
        sha256: sha256(chessgroundSource),
      },
      {
        name: 'svelte',
        version: '3.59.2',
        path: 'app/third_party/source/svelte-v3.59.2.tar.gz',
        sha256: sha256(svelteSource),
      },
    ],
  }
  await put(
    path.join(root, 'SOURCE_MANIFEST.json'),
    `${JSON.stringify(manifest, null, 2)}\n`,
  )

  return {
    root,
    expected: {
      tag,
      commit,
      trackedFiles,
      runtimeLock,
      runtimeLockSha256: sha256(runtimeLockContent),
      goLegal: {
        LICENSE: sha256(goLicense),
        PATENTS: sha256(goPatents),
      },
      wailsTreeSha256: manifest.wails.treeSha256,
      frontendSources: new Map(
        manifest.frontendSources.map((source) => [source.path, source.sha256]),
      ),
    },
  }
}

async function withFixture(mutate, expectation) {
  const fixture = await completeFixture()
  try {
    await mutate(fixture)
    await assert.rejects(
      verifyCorrespondingSourceTree({
        sourceRoot: fixture.root,
        ...fixture.expected,
      }),
      expectation,
    )
  } finally {
    await rm(fixture.root, { recursive: true, force: true })
  }
}

test('accepts a complete corresponding-source tree', async () => {
  const fixture = await completeFixture()
  try {
    await verifyCorrespondingSourceTree({
      sourceRoot: fixture.root,
      ...fixture.expected,
    })
  } finally {
    await rm(fixture.root, { recursive: true, force: true })
  }
})

test('rejects a missing tracked application file', async () => {
  await withFixture(
    ({ root }) => unlink(path.join(root, 'app/main.go')),
    /tracked application file is missing: main\.go/,
  )
})

test('rejects a missing production vendor module', async () => {
  await withFixture(
    ({ root }) => rm(path.join(root, 'app/vendor/example.com/production'), { recursive: true }),
    /vendor source is missing for example\.com\/production@v1\.2\.3/,
  )
})

test('rejects a missing locked module license or NOTICE', async () => {
  await withFixture(
    ({ root }) =>
      unlink(
        path.join(
          root,
          'app/vendor/_licenses/example.com!production@v1.2.3/NOTICE',
        ),
      ),
    /locked legal file is missing: example\.com\/production@v1\.2\.3 NOTICE/,
  )
})

test('rejects missing or changed Go legal files', async () => {
  await withFixture(
    ({ root }) => unlink(path.join(root, 'toolchain/go1.26.4/PATENTS')),
    /Go PATENTS is missing/,
  )
  await withFixture(
    ({ root }) => writeFile(path.join(root, 'toolchain/go1.26.4/LICENSE'), 'changed'),
    /Go LICENSE has an unexpected SHA-256/,
  )
})

test('rejects incomplete Wails source without the CLI', async () => {
  await withFixture(
    ({ root }) => unlink(path.join(root, 'build-tools/wails-v2.12.0/cmd/wails/main.go')),
    /Wails source is missing cmd\/wails/,
  )
})

test('rejects a changed frontend preferred-source archive', async () => {
  await withFixture(
    ({ root }) =>
      writeFile(
        path.join(root, 'app/third_party/source/svelte-v3.59.2.tar.gz'),
        'changed',
      ),
    /frontend source archive has an unexpected SHA-256: .*svelte-v3\.59\.2\.tar\.gz/,
  )
})

test('rejects missing exact build instructions', async () => {
  await withFixture(
    ({ root }) => unlink(path.join(root, 'BUILDING.md')),
    /BUILDING\.md is missing/,
  )
})

test('rejects an altered source manifest', async () => {
  await withFixture(async ({ root }) => {
    const filename = path.join(root, 'SOURCE_MANIFEST.json')
    const manifest = JSON.parse(await readFile(filename, 'utf8'))
    manifest.commit = 'ffffffffffffffffffffffffffffffffffffffff'
    await writeFile(filename, `${JSON.stringify(manifest, null, 2)}\n`)
  }, /source manifest commit does not match release commit/)
})

test('rejects an absolute module replacement in extracted source', async () => {
  await withFixture(
    ({ root }) =>
      writeFile(
        path.join(root, 'app/go.mod'),
        'module example.com/chess-trainer\nreplace example.com/a => /tmp/a\n',
      ),
    /local or absolute Go module replacement/,
  )
})

test('parses injected go module download metadata exactly', async () => {
  const calls = []
  const runner = async (command, args, options) => {
    calls.push({ command, args, options })
    return JSON.stringify({
      Path: 'github.com/wailsapp/wails/v2',
      Version: 'v2.12.0',
      Dir: '/release/go-mod-cache/github.com/wailsapp/wails/v2@v2.12.0',
    })
  }
  const result = await parseGoDownload(
    'github.com/wailsapp/wails/v2',
    'v2.12.0',
    runner,
    { cwd: '/source/app', env: { GOWORK: 'off' } },
  )
  assert.equal(result.Dir, '/release/go-mod-cache/github.com/wailsapp/wails/v2@v2.12.0')
  assert.deepEqual(calls[0].args, [
    'mod',
    'download',
    '-json',
    'github.com/wailsapp/wails/v2@v2.12.0',
  ])
})

test('builder rejects a non-release Go toolchain before creating source', async () => {
  const root = await mkdtemp(path.join(tmpdir(), 'source-builder-toolchain-'))
  const repository = path.join(root, 'repository')
  const releaseRoot = path.join(root, 'release')
  const env = {
    CHESS_TRAINER_RELEASE_ROOT: releaseRoot,
    GOMODCACHE: path.join(releaseRoot, 'go-mod-cache'),
    GOCACHE: path.join(releaseRoot, 'go-build-cache'),
    npm_config_cache: path.join(releaseRoot, 'npm-cache'),
    GOWORK: 'off',
    GOTOOLCHAIN: 'local',
    GOENV: 'off',
    GOFLAGS: '',
    NODE_OPTIONS: '',
  }
  try {
    await put(path.join(repository, 'go.mod'), 'module example.com/app\ngo 1.25.0\n')
    for (const directory of [
      releaseRoot,
      env.GOMODCACHE,
      env.GOCACHE,
      env.npm_config_cache,
    ]) {
      await mkdir(directory, { recursive: true })
    }
    await assert.rejects(
      buildCorrespondingSource({
        root: repository,
        tag,
        commit,
        output: path.join(repository, 'source.tar.gz'),
        env,
        ports: {
          run: async (command, args) => {
            assert.equal(command, 'go')
            assert.deepEqual(args, ['env', 'GOVERSION'])
            return 'go1.26.3\n'
          },
        },
      }),
      /release Go toolchain must be go1\.26\.4/,
    )
  } finally {
    await rm(root, { recursive: true, force: true })
  }
})

test('makes a copied read-only module tree removable', async () => {
  const root = await mkdtemp(path.join(tmpdir(), 'readonly-module-copy-'))
  const nested = path.join(root, 'nested')
  const filename = path.join(nested, 'source.go')
  await put(filename, 'package source\n')
  await chmod(filename, 0o444)
  await chmod(nested, 0o555)
  await chmod(root, 0o555)

  await makeTreeWritable(root)
  await rm(root, { recursive: true, force: true })
  await assert.rejects(readFile(filename), /ENOENT/)
})

test('generated build instructions name every verifier-required input', () => {
  const building = renderBuilding(tag, commit)
  for (const required of [
    'Go 1.26.4',
    'build-tools/wails-v2.12.0/cmd/wails',
    'app/vendor',
    'GOFLAGS=-mod=vendor',
    'GOFLAGS=-mod=mod',
    'npm --prefix frontend ci',
    'npm --prefix frontend run build',
    'GOWORK=off',
    'GOTOOLCHAIN=local',
  ]) {
    assert.match(building, new RegExp(required.replaceAll('.', '\\.')))
  }
  assert.match(building, /\(\n  cd build-tools\/wails-v2\.12\.0/)
  assert.match(building, /\(\n  cd app\n  npm --prefix frontend ci/)
  assert.match(building, /\(\n  cd app\n  GOFLAGS=-mod=vendor/)
})
