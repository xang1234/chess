import { execFile } from 'node:child_process'
import {
  cp,
  mkdir,
  mkdtemp,
  readFile,
  readdir,
  realpath,
  rm,
  writeFile,
} from 'node:fs/promises'
import path from 'node:path'
import { promisify } from 'node:util'

import {
  escapeModulePath,
  verifyCorrespondingSourceArchive,
  verifyCorrespondingSourceTree,
} from './corresponding-source-verifier.mjs'
import { createDeterministicTarGzip } from './deterministic-tar.mjs'
import { assertNoLocalModuleReplacement } from './go-module-policy.mjs'
import {
  RELEASE_GO_VERSION,
  RELEASE_TARGET_ENVIRONMENT_VARIABLES,
  WAILS_MODULE,
  WAILS_VERSION,
  assertFullCommit,
  assertGoToolchain,
  assertRealReleaseEnvironment,
  assertReleaseTag,
  isBeneath,
  parseGoDownload,
  releaseVersionFromTag,
} from './release-policy.mjs'
import {
  CHESSGROUND_SOURCE_SHA256,
  GO_LICENSE_SHA256,
  GO_PATENTS_SHA256,
  SVELTE_SOURCE_SHA256,
} from './verify-legal-assets.mjs'
import {
  assertDigest,
  collectTrackedFiles,
  compareText,
  makeTreeWritable,
  renderTrackedFiles,
  requireDirectory,
  requireFile,
  sha256,
  treeDigest,
} from './tree-integrity.mjs'

const execFileAsync = promisify(execFile)
const LEGAL_NAME = /^(?:LICENSE|COPYING|NOTICE)/i

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
    cp,
    createDeterministicTarGzip,
    mkdir,
    mkdtemp,
    readFile,
    readdir,
    realpath,
    rm,
    writeFile,
  }
}

async function assertDownloadedBelowCache(directory, env, ports, label) {
  const [downloaded, cache] = await Promise.all([
    ports.realpath(directory),
    ports.realpath(env.GOMODCACHE),
  ])
  if (!isBeneath(cache, downloaded)) {
    throw new Error(`${label} source did not come from the release GOMODCACHE`)
  }
}

async function copyModuleLegal(module, appRoot, env, ports) {
  const metadata = await parseGoDownload(
    module.path,
    module.version,
    ports.run,
    { cwd: appRoot, env },
  )
  await assertDownloadedBelowCache(
    metadata.Dir,
    env,
    ports,
    `${module.path}@${module.version}`,
  )
  const entries = await ports.readdir(metadata.Dir, { withFileTypes: true })
  const legalEntries = entries
    .filter((entry) => entry.isFile() && LEGAL_NAME.test(entry.name))
    .sort((left, right) => compareText(left.name, right.name))
  if (legalEntries.length === 0) {
    throw new Error(`${module.path}@${module.version} has no module-root legal files`)
  }
  const destination = path.join(
    appRoot,
    'vendor/_licenses',
    `${escapeModulePath(module.path)}@${module.version}`,
  )
  await ports.mkdir(destination, { recursive: true })
  const copied = new Map()
  for (const entry of legalEntries) {
    const content = await ports.readFile(path.join(metadata.Dir, entry.name))
    await ports.writeFile(path.join(destination, entry.name), content)
    copied.set(entry.name, content)
  }
  for (const required of module.legal ?? []) {
    const content = copied.get(required.name)
    if (!content) {
      throw new Error(
        `${module.path}@${module.version} is missing locked legal file ${required.name}`,
      )
    }
    assertDigest(
      `${module.path}@${module.version} ${required.name}`,
      content,
      required.sha256,
    )
  }
  return destination
}

export function renderBuilding(tag, commit) {
  const releaseVersion = releaseVersionFromTag(tag)
  return `# Building Chess Trainer from corresponding source

This archive corresponds to Chess Trainer ${tag} at commit ${commit}.

Use the exact Go 1.26.4 release toolchain. Its LICENSE and PATENTS files are
included under \`toolchain/go1.26.4/\`. The application module itself retains
the Go 1.25 language level declared in \`app/go.mod\`.

Every command block below starts at the archive root. First fix the Go
environment and prove the exact local toolchain:

\`\`\`bash
export GOWORK=off
export GOTOOLCHAIN=local
export GOENV=off
export GOFLAGS=
unset ${RELEASE_TARGET_ENVIRONMENT_VARIABLES.join(' ')}
test "$(go env GOVERSION)" = go1.26.4
\`\`\`

Build the pinned Wails CLI from the included complete source:

The Wails CLI entry point is \`build-tools/wails-v2.12.0/cmd/wails\`.

\`\`\`bash
(
  cd build-tools/wails-v2.12.0
  go build -mod=mod -o ../../wails ./cmd/wails
)
\`\`\`

Then install the locked frontend inputs and verify their legal/source identity:

\`\`\`bash
(
  cd app
  npm --prefix frontend ci
  npm --prefix frontend run build
  GOFLAGS=-mod=mod npm --prefix frontend run verify:licenses
)
\`\`\`

Finally build with the included \`app/vendor\` tree and inject the matching commit:

\`\`\`bash
(
  cd app
  GOFLAGS=-mod=vendor ../wails build -clean -trimpath \\
    -platform darwin/arm64 -m -nosyncgomod \\
    -ldflags "-X chess-trainer/internal/buildinfo.Commit=${commit}" \\
    -tags ""
  plutil -replace CFBundleShortVersionString -string "${releaseVersion}" "build/bin/Chess Trainer.app/Contents/Info.plist"
  plutil -replace CFBundleVersion -string "${releaseVersion}" "build/bin/Chess Trainer.app/Contents/Info.plist"
)
\`\`\`

Keep \`GOWORK=off\`, \`GOTOOLCHAIN=local\`, and use \`-mod=vendor\` for direct
Go commands against the application source. The frontend install is fixed by
\`frontend/package-lock.json\`; do not replace it with a floating install.
`
}

function parseNulList(output) {
  return output.split('\0').filter(Boolean)
}

export async function buildCorrespondingSource({
  root,
  tag,
  commit,
  output,
  env = process.env,
  ports: providedPorts,
}) {
  const ports = { ...defaultPorts(), ...providedPorts }
  const exactTag = assertReleaseTag(tag)
  const exactCommit = assertFullCommit(commit)
  const realReleaseRoot = await assertRealReleaseEnvironment(env, ports)

  const repositoryRoot = path.resolve(root)
  assertNoLocalModuleReplacement(
    await ports.readFile(path.join(repositoryRoot, 'go.mod'), 'utf8'),
  )
  assertGoToolchain(
    await ports.run('go', ['env', 'GOVERSION'], {
      cwd: repositoryRoot,
      env,
    }),
  )

  const workspace = await ports.mkdtemp(
    path.join(realReleaseRoot, 'corresponding-source-build-'),
  )
  const artifactName = `Chess-Trainer-${exactTag}-corresponding-source`
  const payloadRoot = path.join(workspace, 'payload')
  const sourceRoot = path.join(payloadRoot, artifactName)
  const appRoot = path.join(sourceRoot, 'app')
  try {
    await ports.mkdir(appRoot, { recursive: true })
    const appArchive = path.join(workspace, 'app-source.tar')
    await ports.run(
      'git',
      ['archive', '--format=tar', '--output', appArchive, exactCommit],
      { cwd: repositoryRoot, env },
    )
    await ports.run('tar', ['-xf', appArchive, '-C', appRoot], {
      cwd: repositoryRoot,
      env,
    })

    const trackedPaths = parseNulList(
      await ports.run(
        'git',
        ['ls-tree', '-r', '--name-only', '-z', exactCommit],
        { cwd: repositoryRoot, env },
      ),
    )
    const trackedFiles = await collectTrackedFiles(appRoot, trackedPaths, ports)
    const goMod = await ports.readFile(path.join(appRoot, 'go.mod'), 'utf8')
    assertNoLocalModuleReplacement(goMod)

    await ports.run('go', ['mod', 'vendor', '-o', 'vendor'], {
      cwd: appRoot,
      env,
    })
    const runtimeLockContent = await ports.readFile(
      path.join(appRoot, 'third_party/runtime-dependencies.lock.json'),
    )
    const runtimeLock = JSON.parse(runtimeLockContent.toString('utf8'))
    if (runtimeLock.goToolchain?.version !== RELEASE_GO_VERSION) {
      throw new Error(`runtime lock must require ${RELEASE_GO_VERSION}`)
    }

    const goLegal = {}
    const toolchainRoot = path.join(sourceRoot, 'toolchain', RELEASE_GO_VERSION)
    await ports.mkdir(toolchainRoot, { recursive: true })
    for (const [name, expectedDigest] of [
      ['LICENSE', GO_LICENSE_SHA256],
      ['PATENTS', GO_PATENTS_SHA256],
    ]) {
      const content = await ports.readFile(
        path.join(appRoot, 'third_party/legal', RELEASE_GO_VERSION, name),
      )
      assertDigest(`Go ${name}`, content, expectedDigest)
      const locked = runtimeLock.goToolchain.legal?.find(
        (entry) => entry.name === name,
      )
      if (locked?.sha256 !== expectedDigest) {
        throw new Error(`runtime lock has the wrong Go ${name} digest`)
      }
      await ports.writeFile(path.join(toolchainRoot, name), content)
      goLegal[name] = expectedDigest
    }

    const moduleManifests = []
    for (const module of runtimeLock.goModules ?? []) {
      const legalDirectory = await copyModuleLegal(module, appRoot, env, ports)
      const sourceDirectory = path.join(
        appRoot,
        'vendor',
        ...module.path.split('/'),
      )
      await requireDirectory(
        sourceDirectory,
        `vendor source is missing for ${module.path}@${module.version}`,
        ports,
      )
      moduleManifests.push({
        path: module.path,
        version: module.version,
        sourceTreeSha256: await treeDigest(sourceDirectory, ports),
        legalTreeSha256: await treeDigest(legalDirectory, ports),
      })
    }

    const wailsDownload = await parseGoDownload(
      WAILS_MODULE,
      WAILS_VERSION,
      ports.run,
      { cwd: appRoot, env },
    )
    await assertDownloadedBelowCache(
      wailsDownload.Dir,
      env,
      ports,
      `${WAILS_MODULE}@${WAILS_VERSION}`,
    )
    const wailsRoot = path.join(sourceRoot, 'build-tools/wails-v2.12.0')
    await ports.cp(wailsDownload.Dir, wailsRoot, { recursive: true })
    await makeTreeWritable(wailsRoot, ports)
    await requireFile(
      path.join(wailsRoot, 'cmd/wails/main.go'),
      'downloaded Wails source is missing cmd/wails/main.go',
      ports,
    )

    const frontendSources = []
    for (const source of [
      {
        name: '@lichess-org/chessground',
        version: '10.1.1',
        path: 'app/third_party/source/chessground-v10.1.1.tar.gz',
        sha256: CHESSGROUND_SOURCE_SHA256,
      },
      {
        name: 'svelte',
        version: '3.59.2',
        path: 'app/third_party/source/svelte-v3.59.2.tar.gz',
        sha256: SVELTE_SOURCE_SHA256,
      },
    ]) {
      assertDigest(
        `${source.name} preferred source`,
        await ports.readFile(path.join(sourceRoot, source.path)),
        source.sha256,
      )
      frontendSources.push(source)
    }

    const trackedText = renderTrackedFiles(trackedFiles)
    await ports.writeFile(path.join(sourceRoot, 'TRACKED_FILES.sha256'), trackedText)
    await ports.writeFile(
      path.join(sourceRoot, 'BUILDING.md'),
      renderBuilding(exactTag, exactCommit),
    )

    const vendorRoot = path.join(appRoot, 'vendor')
    const wailsTreeSha256 = await treeDigest(wailsRoot, ports)
    const manifest = {
      formatVersion: 1,
      tag: exactTag,
      commit: exactCommit,
      toolchain: {
        version: RELEASE_GO_VERSION,
        legal: goLegal,
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
        treeSha256: await treeDigest(vendorRoot, ports),
        modules: moduleManifests,
      },
      wails: {
        module: WAILS_MODULE,
        version: WAILS_VERSION,
        path: 'build-tools/wails-v2.12.0',
        treeSha256: wailsTreeSha256,
      },
      frontendSources,
    }
    await ports.writeFile(
      path.join(sourceRoot, 'SOURCE_MANIFEST.json'),
      `${JSON.stringify(manifest, null, 2)}\n`,
    )

    const expectedFrontend = new Map(
      frontendSources.map((source) => [source.path, source.sha256]),
    )
    const expected = {
      tag: exactTag,
      commit: exactCommit,
      trackedFiles,
      runtimeLock,
      runtimeLockSha256: sha256(runtimeLockContent),
      goLegal,
      wailsTreeSha256,
      frontendSources: expectedFrontend,
      ports,
    }
    await verifyCorrespondingSourceTree({ sourceRoot, ...expected })

    const outputPath = path.isAbsolute(output)
      ? output
      : path.resolve(repositoryRoot, output)
    await ports.mkdir(path.dirname(outputPath), { recursive: true })
    await ports.createDeterministicTarGzip({
      sourceDirectory: sourceRoot,
      outputFile: outputPath,
      rootName: artifactName,
    })
    await verifyCorrespondingSourceArchive({
      archive: outputPath,
      temporaryRoot: realReleaseRoot,
      ...expected,
    })
    return { output: outputPath, manifest }
  } finally {
    await makeTreeWritable(workspace, ports)
    await ports.rm(workspace, { recursive: true, force: true })
  }
}
