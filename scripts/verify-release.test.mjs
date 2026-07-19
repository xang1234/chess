import assert from 'node:assert/strict'
import { mkdtemp, mkdir, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

import {
  PUBLIC_REPOSITORY_URL,
  RELEASE_BUNDLE_IDENTIFIER,
  RELEASE_PLATFORM,
  REQUIRED_TRACKED_FILES,
  assertCleanStatus,
  assertCourseFixtureBoundary,
  assertBundleMetadata,
  assertBundleIdentifier,
  assertDistLegalAssets,
  assertExecutableContents,
  assertExecutableModuleClosure,
  assertExecutableTarget,
  assertFullCommit,
  assertGoToolchain,
  assertNoLocalModuleReplacement,
  assertReleaseEnvironment,
  assertReleaseTag,
  assertRequiredTrackedFiles,
  assertRuntimeArtifacts,
  publicTagQuery,
  resolveTagCommit,
  releaseVersionFromTag,
  verifyCodeSignature,
  verifyPublicTag,
  verifyReleaseInputTree,
} from './verify-release.mjs'
import {
  CHESSGROUND_SOURCE_SHA256,
  GPL3_SHA256,
  GO_LICENSE_SHA256,
  GO_PATENTS_SHA256,
  assertDigest,
  assertTreesEqual,
} from './verify-legal-assets.mjs'

const commit = '0123456789abcdef0123456789abcdef01234567'
const repositoryRoot = fileURLToPath(new URL('..', import.meta.url))

test('rejects a dirty tree and reports the first changed path', () => {
  assert.throws(
    () => assertCleanStatus(' M README.md\n?? required.txt\n'),
    /working tree is not clean: M README\.md/,
  )
})

test('rejects a missing or untracked required release input', () => {
  const tracked = new Set(REQUIRED_TRACKED_FILES)
  tracked.delete('frontend/public/legal/NUNITO_OFL.txt')

  assert.throws(
    () => assertRequiredTrackedFiles(tracked),
    /required release file is not tracked: frontend\/public\/legal\/NUNITO_OFL\.txt/,
  )
})

test('allows only the reviewed synthetic opening course fixture', () => {
  assert.doesNotThrow(() => assertCourseFixtureBoundary(
    ['internal/openings/testdata/mini.ctcourse', 'README.md'],
    { root: repositoryRoot },
  ))
  assert.throws(
    () => assertCourseFixtureBoundary(
      ['internal/openings/testdata/private-source.ctcourse'],
      { root: repositoryRoot },
    ),
    /unreviewed opening course fixture/,
  )
  assert.throws(
    () => assertCourseFixtureBoundary(
      ['internal/openings/testdata/private-source.CTCOURSE'],
      { root: repositoryRoot },
    ),
    /unreviewed opening course fixture/,
  )
  assert.throws(
    () => assertCourseFixtureBoundary(
      ['internal/openings/testdata/..\/testdata\/mini.ctcourse'],
      { root: repositoryRoot },
    ),
    /non-canonical opening course path/,
  )
  assert.throws(
    () => assertCourseFixtureBoundary(
      ['internal/openings/testdata/mini.ctcourse'],
      { root: repositoryRoot, readFixture: () => Buffer.from('changed') },
    ),
    /opening course fixture digest differs/,
  )
})

test('rejects floating Chessground dependency metadata', () => {
  assert.throws(
    () =>
      assertRequiredTrackedFiles(new Set(REQUIRED_TRACKED_FILES), {
        chessgroundVersion: '^10.1.1',
      }),
    /Chessground dependency must be pinned exactly to 10\.1\.1/,
  )
})

test('accepts only a full lowercase commit', () => {
  assert.equal(assertFullCommit(commit), commit)
  assert.throws(() => assertFullCommit('0123456'), /full 40-character lowercase SHA/)
  assert.throws(
    () => assertFullCommit(commit.toUpperCase()),
    /full 40-character lowercase SHA/,
  )
})

test('accepts only plain semantic release tags', () => {
  assert.equal(assertReleaseTag('v1.2.3'), 'v1.2.3')
  for (const invalid of [
    '1.2.3',
    'v1.2',
    'v1.2.3-rc1',
    'release/v1.2.3',
    '../v1.2.3',
    'v1.2.3/../../main',
  ]) {
    assert.throws(() => assertReleaseTag(invalid), /must match v<major>\.<minor>\.<patch>/)
  }
})

test('derives the macOS bundle version exactly from the public tag', () => {
  assert.equal(releaseVersionFromTag('v1.2.3'), '1.2.3')
  assert.throws(
    () => releaseVersionFromTag('release-1.2.3'),
    /release tag must match/,
  )
})

test('accepts a lightweight tag that resolves directly to HEAD', () => {
  const output = `${commit}\trefs/tags/v1.2.3\n`
  assert.equal(resolveTagCommit(output, 'v1.2.3'), commit)
})

test('accepts an annotated tag only through its peeled record', () => {
  const tagObject = 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa'
  const output = [
    `${tagObject}\trefs/tags/v1.2.3`,
    `${commit}\trefs/tags/v1.2.3^{}`,
    '',
  ].join('\n')
  assert.equal(resolveTagCommit(output, 'v1.2.3'), commit)
})

test('rejects a tag resolving to another commit', () => {
  const other = 'ffffffffffffffffffffffffffffffffffffffff'
  assert.throws(
    () => {
      const tagged = resolveTagCommit(`${other}\trefs/tags/v1.2.3\n`, 'v1.2.3')
      assert.equal(tagged, commit, 'tag v1.2.3 does not resolve to HEAD')
    },
    /does not resolve to HEAD/,
  )
})

test('builds a credential-free public tag query against the fixed repository', () => {
  const query = publicTagQuery('v1.2.3', '/tmp/empty-home', '/usr/bin:/bin')

  assert.equal(query.command, 'git')
  assert.deepEqual(query.args, [
    '-c',
    'credential.helper=',
    '-c',
    'http.extraHeader=',
    'ls-remote',
    '--tags',
    PUBLIC_REPOSITORY_URL,
    'refs/tags/v1.2.3',
    'refs/tags/v1.2.3^{}',
  ])
  assert.equal(query.options.cwd, '/tmp/empty-home')
  assert.deepEqual(query.options.env, {
    PATH: '/usr/bin:/bin',
    GIT_CONFIG_NOSYSTEM: '1',
    GIT_CONFIG_GLOBAL: '/dev/null',
    HOME: '/tmp/empty-home',
    GIT_TERMINAL_PROMPT: '0',
    GIT_ASKPASS: '/usr/bin/false',
  })
})

test('rejects when the fixed public GitHub tag is not reachable', async () => {
  const calls = []
  const runner = async (command, args, options) => {
    calls.push({ command, args, options })
    return ''
  }

  await assert.rejects(
    verifyPublicTag({ tag: 'v1.2.3', commit, runner, pathValue: '/bin' }),
    /public tag v1\.2\.3 is not reachable without credentials/,
  )
  assert.equal(calls.length, 1)
  assert.equal(calls[0].args[6], PUBLIC_REPOSITORY_URL)
  assert.equal(calls[0].options.env.GIT_CONFIG_GLOBAL, '/dev/null')
})

test('requires the exact release Go toolchain', () => {
  assert.equal(assertGoToolchain('go1.26.4\n'), 'go1.26.4')
  assert.throws(() => assertGoToolchain('go1.26.3\n'), /must be go1\.26\.4/)
})

test('rejects active and commented local Go module replacements', () => {
  for (const goMod of [
    'replace example.com/a => /Users/me/a\n',
    'replace example.com/a => ../a\n',
    '// replace example.com/a => /tmp/a\n',
    'replace (\n  // example.com/a => ../../a\n)\n',
    'replace example.com/a => "\\x2ftmp\\x2flocal"\n',
    '// replace example.com/a => "\\u002e\\u002e/module"\n',
  ]) {
    assert.throws(
      () => assertNoLocalModuleReplacement(goMod),
      /local or absolute Go module replacement/,
    )
  }
})

test('requires every build cache to live below a fresh release root', () => {
  const root = '/tmp/release-123'
  assert.doesNotThrow(() =>
    assertReleaseEnvironment({
      CHESS_TRAINER_RELEASE_ROOT: root,
      GOMODCACHE: `${root}/go-mod-cache`,
      GOCACHE: `${root}/go-build-cache`,
      npm_config_cache: `${root}/npm-cache`,
      GOWORK: 'off',
      GOTOOLCHAIN: 'local',
      GOENV: 'off',
      GOFLAGS: '',
      NODE_OPTIONS: '',
    }),
  )
  assert.throws(
    () =>
      assertReleaseEnvironment({
        CHESS_TRAINER_RELEASE_ROOT: root,
        GOMODCACHE: '/Users/me/go/pkg/mod',
        GOCACHE: `${root}/go-build-cache`,
        npm_config_cache: `${root}/npm-cache`,
        GOWORK: 'off',
        GOTOOLCHAIN: 'local',
        GOENV: 'off',
        GOFLAGS: '',
        NODE_OPTIONS: '',
      }),
    /GOMODCACHE must resolve beneath CHESS_TRAINER_RELEASE_ROOT/,
  )
  assert.throws(
    () =>
      assertReleaseEnvironment({
        CHESS_TRAINER_RELEASE_ROOT: root,
        GOMODCACHE: `${root}/go-mod-cache`,
        GOCACHE: `${root}/go-build-cache`,
        npm_config_cache: `${root}/npm-cache`,
        GOWORK: 'auto',
        GOTOOLCHAIN: 'local',
        GOENV: 'off',
        GOFLAGS: '',
        NODE_OPTIONS: '',
      }),
    /GOWORK must be off/,
  )
  assert.throws(
    () =>
      assertReleaseEnvironment({
        CHESS_TRAINER_RELEASE_ROOT: root,
        GOMODCACHE: `${root}/go-mod-cache`,
        GOCACHE: `${root}/go-build-cache`,
        npm_config_cache: `${root}/npm-cache`,
        GOWORK: 'off',
        GOTOOLCHAIN: 'local',
        GOENV: '/Users/me/Library/Application Support/go/env',
        GOFLAGS: '-overlay=/tmp/overlay.json',
        NODE_OPTIONS: '',
      }),
    /GOENV must be off/,
  )
  assert.throws(
    () =>
      assertReleaseEnvironment({
        CHESS_TRAINER_RELEASE_ROOT: root,
        GOMODCACHE: `${root}/go-mod-cache`,
        GOCACHE: `${root}/go-build-cache`,
        npm_config_cache: `${root}/npm-cache`,
        GOWORK: 'off',
        GOTOOLCHAIN: 'local',
        GOENV: 'off',
        GOFLAGS: '',
        NODE_OPTIONS: '--require=/tmp/release-hook.cjs',
      }),
    /NODE_OPTIONS must be empty/,
  )

  for (const name of [
    'GOOS',
    'GOARCH',
    'GOAMD64',
    'GOARM64',
    'GOFIPS140',
    'SDKROOT',
    'DEVELOPER_DIR',
    'MACOSX_DEPLOYMENT_TARGET',
  ]) {
    assert.throws(
      () => assertReleaseEnvironment({
        CHESS_TRAINER_RELEASE_ROOT: root,
        GOMODCACHE: `${root}/go-mod-cache`,
        GOCACHE: `${root}/go-build-cache`,
        npm_config_cache: `${root}/npm-cache`,
        GOWORK: 'off',
        GOTOOLCHAIN: 'local',
        GOENV: 'off',
        GOFLAGS: '',
        NODE_OPTIONS: '',
        [name]: 'host-specific-value',
      }),
      new RegExp(`${name} must be unset`),
    )
  }
})

test('rejects ignored or generated build inputs outside explicit output directories', async () => {
  const root = await mkdtemp(path.join(tmpdir(), 'release-input-tree-'))
  const repositoryRoot = path.join(root, 'repository')
  const inputRoot = path.join(root, 'input')
  await mkdir(repositoryRoot)
  await mkdir(inputRoot)
  await writeFile(path.join(repositoryRoot, 'main.go'), 'package main\n')
  await writeFile(path.join(inputRoot, 'main.go'), 'package main\n')
  await writeFile(path.join(inputRoot, 'ignored_overlay.go'), 'package main\n')
  try {
    await assert.rejects(
      verifyReleaseInputTree({
        repositoryRoot,
        inputRoot,
        tracked: new Set(['main.go']),
      }),
      /isolated build tree contains an unexpected input: ignored_overlay\.go/,
    )
  } finally {
    await rm(root, { recursive: true, force: true })
  }
})

test('rejects changed runtime locks or notices', () => {
  assert.throws(
    () =>
      assertRuntimeArtifacts({
        committedLock: Buffer.from('{"formatVersion":1}\n'),
        generatedLock: Buffer.from('{"formatVersion":2}\n'),
        committedNotice: Buffer.from('notice\n'),
        generatedNotice: Buffer.from('notice\n'),
      }),
    /Darwin runtime dependency lock differs/,
  )
})

test('rejects changed source archives, GPL text, and Go legal files', () => {
  assert.throws(
    () => assertDigest('Chessground preferred source', Buffer.from('changed'), CHESSGROUND_SOURCE_SHA256),
    /unexpected SHA-256/,
  )
  assert.throws(
    () => assertDigest('GPL', Buffer.from('truncated GPL'), GPL3_SHA256),
    /unexpected SHA-256/,
  )
  assert.throws(
    () => assertDigest('Go LICENSE', Buffer.from('changed'), GO_LICENSE_SHA256),
    /unexpected SHA-256/,
  )
  assert.throws(
    () => assertDigest('Go PATENTS', Buffer.from('changed'), GO_PATENTS_SHA256),
    /unexpected SHA-256/,
  )
})

test('rejects installed Chessground source that differs from reviewed source', async () => {
  const root = await mkdtemp(path.join(tmpdir(), 'release-source-'))
  const preferred = path.join(root, 'preferred')
  const installed = path.join(root, 'installed')
  await mkdir(preferred)
  await mkdir(installed)
  await writeFile(path.join(preferred, 'board.ts'), 'reviewed\n')
  await writeFile(path.join(installed, 'board.ts'), 'changed\n')
  try {
    await assert.rejects(
      assertTreesEqual(preferred, installed, 'Chessground src'),
      /differs at board\.ts/,
    )
  } finally {
    await rm(root, { recursive: true, force: true })
  }
})

test('rejects dist legal assets that differ from committed public assets', () => {
  assert.throws(
    () =>
      assertDistLegalAssets(
        new Map([['LICENSE.txt', Buffer.from('complete')]]),
        new Map([['LICENSE.txt', Buffer.from('changed')]]),
      ),
    /bundled legal asset differs: LICENSE\.txt/,
  )
})

test('rejects an executable missing the exact commit or legal bytes', () => {
  const documents = new Map([
    ['LICENSE.txt', Buffer.from('complete GPL')],
    ['THIRD_PARTY_NOTICES.md', Buffer.from('all notices')],
  ])
  assert.throws(
    () =>
      assertExecutableContents({
        executable: Buffer.from('complete GPL\nall notices\ndevelopment'),
        commit,
        legalDocuments: documents,
      }),
    /executable does not contain release commit/,
  )
  assert.throws(
    () =>
      assertExecutableContents({
        executable: Buffer.from(`complete GPL\n${commit}`),
        commit,
        legalDocuments: documents,
      }),
    /executable does not contain legal document: THIRD_PARTY_NOTICES\.md/,
  )
})

test('rejects an executable module closure that differs from the runtime lock', () => {
  const runtimeLock = {
    goModules: [
      { path: 'example.com/required', version: 'v1.2.3' },
      { path: 'example.com/second', version: 'v2.0.0' },
    ],
  }
  const missing = [
    '/tmp/app: go1.26.4',
    '\tdep\texample.com/required\tv1.2.3\th1:sum=',
    '',
  ].join('\n')
  assert.throws(
    () => assertExecutableModuleClosure(missing, runtimeLock),
    /executable module closure is missing example\.com\/second@v2\.0\.0/,
  )

  const extra = [
    missing.trimEnd(),
    '\tdep\texample.com/second\tv2.0.0\th1:sum=',
    '\tdep\texample.com/unlocked\tv9.9.9\th1:sum=',
    '',
  ].join('\n')
  assert.throws(
    () => assertExecutableModuleClosure(extra, runtimeLock),
    /executable module closure contains unlocked example\.com\/unlocked@v9\.9\.9/,
  )
})

test('accepts only the declared Darwin arm64 executable target', () => {
  assert.equal(RELEASE_PLATFORM, 'darwin/arm64')
  const buildInfo = [
    '/tmp/Chess Trainer: go1.26.4',
    '\tbuild\tGOARCH=arm64',
    '\tbuild\tGOARM64=v8.0',
    '\tbuild\tGOOS=darwin',
    '',
  ].join('\n')
  assert.doesNotThrow(() => assertExecutableTarget({
    architectures: 'arm64\n',
    buildInfo,
  }))
  assert.throws(
    () => assertExecutableTarget({ architectures: 'x86_64\n', buildInfo }),
    /Mach-O architecture must be arm64/,
  )
  assert.throws(
    () => assertExecutableTarget({
      architectures: 'arm64\n',
      buildInfo: buildInfo.replace('GOARCH=arm64', 'GOARCH=amd64'),
    }),
    /Go build setting GOARCH must be arm64/,
  )
  assert.throws(
    () => assertExecutableTarget({
      architectures: 'arm64\n',
      buildInfo: buildInfo.replace('GOARM64=v8.0', 'GOARM64=v9.0'),
    }),
    /Go build setting GOARM64 must be v8\.0/,
  )
})

test('requires the exact valid macOS bundle identifier', () => {
  assert.equal(RELEASE_BUNDLE_IDENTIFIER, 'com.xang1234.chesstrainer')
  const plist = (identifier) => [
    '<plist><dict>',
    '<key>CFBundleIdentifier</key>',
    `<string>${identifier}</string>`,
    '</dict></plist>',
  ].join('\n')

  assert.equal(
    assertBundleIdentifier(plist(RELEASE_BUNDLE_IDENTIFIER)),
    RELEASE_BUNDLE_IDENTIFIER,
  )
  assert.throws(
    () => assertBundleIdentifier(plist('com.wails.Chess Trainer')),
    /bundle identifier.*letters, numbers, hyphens, and periods/i,
  )
  assert.throws(
    () => assertBundleIdentifier(plist('com.example.chesstrainer')),
    /bundle identifier must be com\.xang1234\.chesstrainer/i,
  )
  assert.throws(
    () => assertBundleIdentifier('<plist><dict></dict></plist>'),
    /CFBundleIdentifier is missing/,
  )
})

test('requires both macOS bundle version fields to match the release tag', () => {
  const plist = (shortVersion, bundleVersion) => [
    '<plist><dict>',
    '<key>CFBundleIdentifier</key>',
    `<string>${RELEASE_BUNDLE_IDENTIFIER}</string>`,
    '<key>CFBundleShortVersionString</key>',
    `<string>${shortVersion}</string>`,
    '<key>CFBundleVersion</key>',
    `<string>${bundleVersion}</string>`,
    '</dict></plist>',
  ].join('\n')

  assert.equal(assertBundleMetadata(plist('1.2.3', '1.2.3'), 'v1.2.3'), '1.2.3')
  assert.throws(
    () => assertBundleMetadata(plist('1.0.0', '1.2.3'), 'v1.2.3'),
    /CFBundleShortVersionString must be 1\.2\.3/,
  )
  assert.throws(
    () => assertBundleMetadata(plist('1.2.3', '1.0.0'), 'v1.2.3'),
    /CFBundleVersion must be 1\.2\.3/,
  )
  assert.throws(
    () => assertBundleMetadata([
      '<plist><dict>',
      '<key>CFBundleIdentifier</key>',
      `<string>${RELEASE_BUNDLE_IDENTIFIER}</string>`,
      '</dict></plist>',
    ].join('\n'), 'v1.2.3'),
    /CFBundleShortVersionString is missing/,
  )
})

test('turns a failed strict code-signature check into one actionable error', async () => {
  const runner = async () => {
    throw new Error('code object is not signed at all')
  }
  await assert.rejects(
    verifyCodeSignature('/tmp/Chess Trainer.app', runner),
    /codesign --verify --deep --strict failed: code object is not signed at all/,
  )
})

test('rejects an ad-hoc signature even when codesign strict verification passes', async () => {
  const runner = async (command, args) => {
    if (command === 'codesign' && args[0] === '--display') {
      return 'Signature=adhoc\nTeamIdentifier=not set\n'
    }
    return ''
  }
  await assert.rejects(
    verifyCodeSignature('/tmp/Chess Trainer.app', runner),
    /Developer ID Application signature/,
  )
})

test('requires Developer ID, hardened runtime, notarization staple, and Gatekeeper acceptance', async () => {
  const calls = []
  const runner = async (command, args) => {
    calls.push([command, ...args])
    if (command === 'codesign' && args[0] === '--display') {
      return [
        'Authority=Developer ID Application: Chess Trainer (TEAM123456)',
        'TeamIdentifier=TEAM123456',
        'Runtime Version=15.0.0',
        'Timestamp=17 Jul 2026 at 12:00:00',
        '',
      ].join('\n')
    }
    return ''
  }

  await verifyCodeSignature('/tmp/Chess Trainer.app', runner)

  assert.ok(calls.some((call) => call[0] === 'xcrun' && call[1] === 'stapler' && call[2] === 'validate'))
  assert.ok(calls.some((call) => call[0] === 'spctl' && call[1] === '--assess'))
})

test('rejects a Developer ID signature without a trusted timestamp', async () => {
  const runner = async (command, args) => {
    if (command === 'codesign' && args[0] === '--display') {
      return [
        'Authority=Developer ID Application: Chess Trainer (TEAM123456)',
        'TeamIdentifier=TEAM123456',
        'Runtime Version=15.0.0',
        '',
      ].join('\n')
    }
    return ''
  }

  await assert.rejects(
    verifyCodeSignature('/tmp/Chess Trainer.app', runner),
    /trusted timestamp/,
  )
})
