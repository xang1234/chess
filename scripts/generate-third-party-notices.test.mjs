import assert from 'node:assert/strict'
import { createHash } from 'node:crypto'
import { mkdtemp, mkdir, readFile, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import path from 'node:path'
import test from 'node:test'

import {
  assertNoLocalModuleReplacement,
  mergeRuntimeModules,
  renderThirdPartyNotices,
  runNoticeGenerator,
  validateGoToolchain,
} from './generate-third-party-notices.mjs'

function runtimePackage(path, version, directory = `/modules/${path}@${version}`) {
  return {
    ImportPath: `${path}/runtime-package`,
    Module: { Path: path, Version: version, Dir: directory },
  }
}

test('unions and sorts runtime modules from both Darwin architectures', () => {
  const arm64 = [
    runtimePackage('example.com/shared', 'v1.0.0'),
    runtimePackage('example.com/arm', 'v2.0.0'),
  ]
  const amd64 = [
    runtimePackage('example.com/amd', 'v3.0.0'),
    runtimePackage('example.com/shared', 'v1.0.0'),
  ]

  assert.deepEqual(
    mergeRuntimeModules([arm64, amd64]).map(({ path, version }) => ({
      path,
      version,
    })),
    [
      { path: 'example.com/amd', version: 'v3.0.0' },
      { path: 'example.com/arm', version: 'v2.0.0' },
      { path: 'example.com/shared', version: 'v1.0.0' },
    ],
  )
})

test('rejects active and commented machine-local module replacements', () => {
  assert.throws(
    () =>
      assertNoLocalModuleReplacement(
        'replace example.com/module => /Users/child/module\n',
      ),
    /local or absolute Go module replacement/,
  )
  assert.throws(
    () =>
      assertNoLocalModuleReplacement(
        '// replace example.com/module => ../module\n',
      ),
    /local or absolute Go module replacement/,
  )
  assert.throws(
    () =>
      assertNoLocalModuleReplacement(
        'replace example.com/module => "\\x2ftmp\\x2flocal"\n',
      ),
    /local or absolute Go module replacement/,
  )
  assert.throws(
    () =>
      assertNoLocalModuleReplacement(
        '// replace example.com/module => "\\u002e\\u002e/module"\n',
      ),
    /local or absolute Go module replacement/,
  )
})

test('requires the exact reviewed Go release toolchain and legal bytes', () => {
  assert.throws(
    () =>
      validateGoToolchain({
        version: 'go1.26.5',
        license: Buffer.from('license'),
        patents: Buffer.from('patents'),
      }),
    /release toolchain must be go1\.26\.4/,
  )
})

test('renders every named dependency and complete legal text', () => {
  const notices = renderThirdPartyNotices({
    applicationLicense: 'GPL-3.0-or-later',
    goToolchain: {
      version: 'go1.26.4',
      legal: [
        { name: 'LICENSE', sha256: 'go-license', text: 'GO LICENSE FULL TEXT' },
        { name: 'PATENTS', sha256: 'go-patents', text: 'GO PATENT GRANT' },
      ],
    },
    goModules: [
      {
        path: 'example.com/module',
        version: 'v1.2.3',
        legal: [
          { name: 'LICENSE', sha256: 'module-license', text: 'MODULE LICENSE' },
        ],
      },
    ],
    frontend: [
      {
        name: '@lichess-org/chessground',
        version: '10.1.1',
        metadata: [
          'Lichess Team <contact@lichess.org>',
          'https://github.com/lichess-org/chessground',
          'third_party/source/chessground-v10.1.1.tar.gz',
        ],
        legal: [
          { name: 'LICENSE', sha256: 'chessground-license', text: 'CHESSGROUND GPL' },
        ],
      },
    ],
  })

  assert.match(notices, /Go runtime and standard library go1\.26\.4/)
  assert.match(notices, /GO LICENSE FULL TEXT/)
  assert.match(notices, /GO PATENT GRANT/)
  assert.match(notices, /example\.com\/module v1\.2\.3/)
  assert.match(notices, /MODULE LICENSE/)
  assert.match(notices, /@lichess-org\/chessground 10\.1\.1/)
  assert.match(notices, /Lichess Team <contact@lichess\.org>/)
  assert.match(notices, /CHESSGROUND GPL/)
})

function digest(text) {
  return createHash('sha256').update(text).digest('hex')
}

async function writeFixtureFile(root, relativePath, content) {
  const destination = path.join(root, relativePath)
  await mkdir(path.dirname(destination), { recursive: true })
  await writeFile(destination, content)
}

async function generatorFixture() {
  const root = await mkdtemp(path.join(tmpdir(), 'chess-trainer-notices-'))
  const goLicense = 'TEST GO LICENSE\n'
  const goPatents = 'TEST GO PATENTS\n'

  await writeFixtureFile(root, 'LICENSE', 'TEST APPLICATION GPL\n')
  await writeFixtureFile(root, 'go.mod', 'module fixture\n\ngo 1.25.0\n')
  await writeFixtureFile(
    root,
    'third_party/legal/go-test/LICENSE',
    goLicense,
  )
  await writeFixtureFile(
    root,
    'third_party/legal/go-test/PATENTS',
    goPatents,
  )
  await writeFixtureFile(root, 'toolchain/LICENSE', goLicense)
  await writeFixtureFile(root, 'toolchain/libexec/PATENTS', goPatents)

  await writeFixtureFile(root, 'modules/shared/LICENSE', 'SHARED LICENSE\n')
  await writeFixtureFile(root, 'modules/arm/LICENSE', 'ARM LICENSE\n')
  await writeFixtureFile(root, 'modules/amd/NOTICE', 'AMD NOTICE\n')

  await writeFixtureFile(
    root,
    'frontend/node_modules/@lichess-org/chessground/package.json',
    JSON.stringify({
      name: '@lichess-org/chessground',
      version: '10.1.1',
      author: 'Lichess Team <contact@lichess.org>',
      repository: 'https://github.com/lichess-org/chessground',
    }),
  )
  await writeFixtureFile(
    root,
    'frontend/node_modules/@lichess-org/chessground/LICENSE',
    'CHESSGROUND LICENSE\n',
  )
  await writeFixtureFile(
    root,
    'frontend/node_modules/svelte/package.json',
    JSON.stringify({ name: 'svelte', version: '3.59.2' }),
  )
  await writeFixtureFile(
    root,
    'frontend/node_modules/svelte/LICENSE.md',
    'SVELTE LICENSE\n',
  )
  await writeFixtureFile(
    root,
    'frontend/src/assets/fonts/OFL.txt',
    'NUNITO LICENSE\n',
  )

  const modulePackage = (modulePath, version, directory) => ({
    ImportPath: `${modulePath}/package`,
    Module: { Path: modulePath, Version: version, Dir: directory },
  })
  const packageSets = {
    arm64: [
      modulePackage('example.com/arm', 'v1.0.0', path.join(root, 'modules/arm')),
      modulePackage(
        'example.com/shared',
        'v1.0.0',
        path.join(root, 'modules/shared'),
      ),
    ],
    amd64: [
      modulePackage('example.com/amd', 'v1.0.0', path.join(root, 'modules/amd')),
      modulePackage(
        'example.com/shared',
        'v1.0.0',
        path.join(root, 'modules/shared'),
      ),
    ],
  }
  const calls = []

  const commandRunner = async (_command, args, options = {}) => {
    calls.push({ args, options })
    if (args[0] === 'list') {
      return packageSets[options.env.GOARCH]
        .map((entry) => JSON.stringify(entry))
        .join('\n')
    }
    if (args[0] === 'env') {
      return JSON.stringify({
        GOVERSION: 'go-test',
        GOROOT: path.join(root, 'toolchain/libexec'),
      })
    }
    throw new Error(`unexpected command: ${args.join(' ')}`)
  }

  return {
    root,
    packageSets,
    calls,
    commandRunner,
    policy: {
      goVersion: 'go-test',
      goLicenseSha256: digest(goLicense),
      goPatentsSha256: digest(goPatents),
      goLegalDirectory: 'third_party/legal/go-test',
    },
  }
}

test('writes and checks the unioned runtime dependency inventory', async () => {
  const fixture = await generatorFixture()

  await runNoticeGenerator({ ...fixture, mode: 'write' })

  const listCalls = fixture.calls.filter(({ args }) => args[0] === 'list')
  assert.equal(listCalls.length, 2)
  for (const { args } of listCalls) {
    assert.deepEqual(args, [
      'list',
      '-tags',
      'desktop,production',
      '-deps',
      '-json',
      '.',
    ])
  }

  const lock = JSON.parse(
    await readFile(
      path.join(fixture.root, 'third_party/runtime-dependencies.lock.json'),
      'utf8',
    ),
  )
  assert.deepEqual(
    lock.goModules.map(({ path: modulePath, version }) => ({
      path: modulePath,
      version,
    })),
    [
      { path: 'example.com/amd', version: 'v1.0.0' },
      { path: 'example.com/arm', version: 'v1.0.0' },
      { path: 'example.com/shared', version: 'v1.0.0' },
    ],
  )
  assert.doesNotMatch(JSON.stringify(lock), new RegExp(fixture.root))
  assert.match(
    await readFile(path.join(fixture.root, 'THIRD_PARTY_NOTICES.md'), 'utf8'),
    /AMD NOTICE/,
  )

  await runNoticeGenerator({ ...fixture, mode: 'check' })
})

test('check rejects a version-changed runtime closure', async () => {
  const fixture = await generatorFixture()
  await runNoticeGenerator({ ...fixture, mode: 'write' })
  fixture.packageSets.amd64[0].Module.Version = 'v2.0.0'

  await assert.rejects(
    runNoticeGenerator({ ...fixture, mode: 'check' }),
    /runtime dependency lock is out of date/,
  )
})
