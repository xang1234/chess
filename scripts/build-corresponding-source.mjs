import assert from 'node:assert/strict'
import { execFile } from 'node:child_process'
import { createHash } from 'node:crypto'
import {
  chmod,
  cp,
  mkdir,
  mkdtemp,
  readFile,
  readdir,
  realpath,
  rm,
  stat,
  writeFile,
} from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { promisify } from 'node:util'

import {
  CHESSGROUND_SOURCE_SHA256,
  GO_LICENSE_SHA256,
  GO_PATENTS_SHA256,
  SVELTE_SOURCE_SHA256,
} from './verify-legal-assets.mjs'
import {
  RELEASE_GO_VERSION,
  WAILS_MODULE,
  WAILS_VERSION,
  assertFullCommit,
  assertGoToolchain,
  assertNoLocalModuleReplacement,
  assertReleaseEnvironment,
  assertReleaseTag,
} from './verify-release.mjs'

const execFileAsync = promisify(execFile)
const LEGAL_NAME = /^(?:LICENSE|COPYING|NOTICE)/i

function compareText(left, right) {
  if (left < right) return -1
  if (left > right) return 1
  return 0
}

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
    chmod,
    cp,
    mkdir,
    mkdtemp,
    readFile,
    readdir,
    realpath,
    rm,
    stat,
    writeFile,
  }
}

export function sha256(content) {
  return createHash('sha256').update(content).digest('hex')
}

async function filesBelow(root, directory = root, ports = defaultPorts()) {
  const files = []
  const entries = await ports.readdir(directory, { withFileTypes: true })
  for (const entry of entries.sort((left, right) =>
    compareText(left.name, right.name),
  )) {
    const filename = path.join(directory, entry.name)
    if (entry.isDirectory()) {
      files.push(...(await filesBelow(root, filename, ports)))
    } else if (entry.isFile()) {
      files.push(path.relative(root, filename))
    } else {
      throw new Error(`source tree contains unsupported entry: ${filename}`)
    }
  }
  return files
}

export async function treeDigest(root, providedPorts) {
  const ports = { ...defaultPorts(), ...providedPorts }
  const hash = createHash('sha256')
  for (const relative of await filesBelow(root, root, ports)) {
    const content = await ports.readFile(path.join(root, relative))
    hash.update(`${sha256(content)}  ${relative.split(path.sep).join('/')}\n`)
  }
  return hash.digest('hex')
}

export async function makeTreeWritable(root, providedPorts) {
  const ports = { ...defaultPorts(), ...providedPorts }

  async function visit(directory) {
    const directoryDetails = await ports.stat(directory)
    await ports.chmod(directory, (directoryDetails.mode & 0o7777) | 0o700)
    const entries = await ports.readdir(directory, { withFileTypes: true })
    for (const entry of entries) {
      const filename = path.join(directory, entry.name)
      if (entry.isDirectory()) {
        await visit(filename)
      } else if (entry.isFile()) {
        const details = await ports.stat(filename)
        await ports.chmod(filename, (details.mode & 0o7777) | 0o600)
      } else {
        throw new Error(`source tree contains unsupported entry: ${filename}`)
      }
    }
  }

  await visit(root)
}

export function escapeModulePath(modulePath) {
  return modulePath.replaceAll('!', '!!').replaceAll('/', '!')
}

export function renderTrackedFiles(trackedFiles) {
  return [...trackedFiles.entries()]
    .sort(([left], [right]) => compareText(left, right))
    .map(([filename, digest]) => `${digest}  ${filename}\n`)
    .join('')
}

function parseTrackedFiles(text) {
  const tracked = new Map()
  for (const line of text.split(/\r?\n/)) {
    if (!line) continue
    const match = /^([0-9a-f]{64})  (.+)$/.exec(line)
    if (!match) throw new Error(`TRACKED_FILES.sha256 has malformed line: ${line}`)
    if (tracked.has(match[2])) {
      throw new Error(`TRACKED_FILES.sha256 repeats path: ${match[2]}`)
    }
    tracked.set(match[2], match[1])
  }
  return tracked
}

function mapsEqual(left, right) {
  if (left.size !== right.size) return false
  for (const [key, value] of left) {
    if (right.get(key) !== value) return false
  }
  return true
}

async function readRequired(filename, label, ports, encoding) {
  try {
    return await ports.readFile(filename, encoding)
  } catch (error) {
    if (error.code === 'ENOENT') throw new Error(`${label} is missing`)
    throw error
  }
}

async function requireDirectory(filename, label, ports) {
  try {
    const details = await ports.stat(filename)
    if (!details.isDirectory()) throw new Error(`${label} is missing`)
  } catch (error) {
    if (error.code === 'ENOENT') throw new Error(`${label} is missing`)
    throw error
  }
}

async function requireFile(filename, label, ports) {
  try {
    const details = await ports.stat(filename)
    if (!details.isFile()) throw new Error(`${label} is missing`)
  } catch (error) {
    if (error.code === 'ENOENT') throw new Error(`${label} is missing`)
    throw error
  }
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
    throw new Error(`go mod download returned incomplete metadata for ${modulePath}@${version}`)
  }
  return metadata
}

function assertDigest(label, content, expected) {
  if (sha256(content) !== expected) {
    throw new Error(`${label} has an unexpected SHA-256`)
  }
}

function manifestModule(manifest, modulePath, version) {
  return manifest.goVendor?.modules?.find(
    (entry) => entry.path === modulePath && entry.version === version,
  )
}

export async function verifyCorrespondingSourceTree({
  sourceRoot,
  tag,
  commit,
  trackedFiles,
  runtimeLock,
  runtimeLockSha256,
  goLegal,
  wailsTreeSha256,
  frontendSources,
  ports: providedPorts,
}) {
  const ports = { ...defaultPorts(), ...providedPorts }
  const exactTag = assertReleaseTag(tag)
  const exactCommit = assertFullCommit(commit)
  const root = path.resolve(sourceRoot)

  const building = await readRequired(
    path.join(root, 'BUILDING.md'),
    'BUILDING.md',
    ports,
    'utf8',
  )
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
    if (!building.includes(required)) {
      throw new Error(`BUILDING.md does not explain required input: ${required}`)
    }
  }

  const manifest = JSON.parse(
    await readRequired(
      path.join(root, 'SOURCE_MANIFEST.json'),
      'SOURCE_MANIFEST.json',
      ports,
      'utf8',
    ),
  )
  if (manifest.formatVersion !== 1) {
    throw new Error('source manifest format is unsupported')
  }
  if (manifest.tag !== exactTag) {
    throw new Error('source manifest tag does not match release tag')
  }
  if (manifest.commit !== exactCommit) {
    throw new Error('source manifest commit does not match release commit')
  }
  if (manifest.toolchain?.version !== RELEASE_GO_VERSION) {
    throw new Error(`source manifest toolchain must be ${RELEASE_GO_VERSION}`)
  }

  const appRoot = path.join(root, 'app')
  const goMod = await readRequired(
    path.join(appRoot, 'go.mod'),
    'tracked application file go.mod',
    ports,
    'utf8',
  )
  assertNoLocalModuleReplacement(goMod)

  for (const [relative, expectedDigest] of frontendSources) {
    const content = await readRequired(
      path.join(root, relative),
      `frontend source archive ${relative}`,
      ports,
    )
    if (sha256(content) !== expectedDigest) {
      throw new Error(
        `frontend source archive has an unexpected SHA-256: ${relative}`,
      )
    }
    const manifestSource = manifest.frontendSources?.find(
      (source) => source.path === relative,
    )
    if (manifestSource?.sha256 !== expectedDigest) {
      throw new Error(`source manifest has the wrong frontend archive digest: ${relative}`)
    }
  }

  for (const name of ['LICENSE', 'PATENTS']) {
    const content = await readRequired(
      path.join(root, 'toolchain', RELEASE_GO_VERSION, name),
      `Go ${name}`,
      ports,
    )
    assertDigest(`Go ${name}`, content, goLegal[name])
    if (manifest.toolchain?.legal?.[name] !== goLegal[name]) {
      throw new Error(`source manifest has the wrong Go ${name} digest`)
    }
  }

  const wailsRoot = path.join(root, 'build-tools/wails-v2.12.0')
  await requireFile(
    path.join(wailsRoot, 'cmd/wails/main.go'),
    'Wails source is missing cmd/wails/main.go',
    ports,
  ).catch((error) => {
    if (/ is missing$/.test(error.message)) {
      throw new Error('Wails source is missing cmd/wails/')
    }
    throw error
  })
  await requireFile(path.join(wailsRoot, 'go.mod'), 'Wails source go.mod', ports)
  await requireFile(path.join(wailsRoot, 'LICENSE'), 'Wails source LICENSE', ports)
  const actualWailsDigest = await treeDigest(wailsRoot, ports)
  if (actualWailsDigest !== wailsTreeSha256) {
    throw new Error('included Wails source differs from the downloaded Wails module')
  }
  if (
    manifest.wails?.module !== WAILS_MODULE ||
    manifest.wails?.version !== WAILS_VERSION ||
    manifest.wails?.treeSha256 !== wailsTreeSha256
  ) {
    throw new Error('source manifest has the wrong Wails source identity')
  }

  const runtimeLockContent = await readRequired(
    path.join(appRoot, 'third_party/runtime-dependencies.lock.json'),
    'runtime dependency lock',
    ports,
  )
  assertDigest('runtime dependency lock', runtimeLockContent, runtimeLockSha256)
  assert.deepEqual(
    JSON.parse(runtimeLockContent.toString('utf8')),
    runtimeLock,
    'corresponding-source runtime dependency lock differs from release input',
  )
  if (
    manifest.runtimeDependenciesLock?.sha256 !== runtimeLockSha256 ||
    manifest.runtimeDependenciesLock?.path !==
      'app/third_party/runtime-dependencies.lock.json'
  ) {
    throw new Error('source manifest has the wrong runtime dependency lock')
  }

  const vendorRoot = path.join(appRoot, 'vendor')
  const modulesText = await readRequired(
    path.join(vendorRoot, 'modules.txt'),
    'production vendor modules.txt',
    ports,
    'utf8',
  )
  for (const module of runtimeLock.goModules ?? []) {
    const identity = `${module.path}@${module.version}`
    const moduleHeader = new RegExp(
      `^# ${module.path.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')} ${module.version.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')}$`,
      'm',
    )
    if (!moduleHeader.test(modulesText)) {
      throw new Error(`production vendor is missing module ${identity}`)
    }
    const sourceDirectory = path.join(vendorRoot, ...module.path.split('/'))
    try {
      await requireDirectory(
        sourceDirectory,
        `vendor source is missing for ${identity}`,
        ports,
      )
    } catch (error) {
      throw new Error(`vendor source is missing for ${identity}`)
    }
    const legalDirectory = path.join(
      vendorRoot,
      '_licenses',
      `${escapeModulePath(module.path)}@${module.version}`,
    )
    for (const legal of module.legal ?? []) {
      const filename = path.join(legalDirectory, legal.name)
      let content
      try {
        content = await ports.readFile(filename)
      } catch (error) {
        if (error.code === 'ENOENT') {
          throw new Error(
            `locked legal file is missing: ${identity} ${legal.name}`,
          )
        }
        throw error
      }
      assertDigest(`${identity} ${legal.name}`, content, legal.sha256)
    }
    const recorded = manifestModule(manifest, module.path, module.version)
    if (!recorded) throw new Error(`source manifest omits vendor module ${identity}`)
    if (recorded.sourceTreeSha256 !== (await treeDigest(sourceDirectory, ports))) {
      throw new Error(`source manifest has the wrong vendor source digest: ${identity}`)
    }
    if (recorded.legalTreeSha256 !== (await treeDigest(legalDirectory, ports))) {
      throw new Error(`source manifest has the wrong vendor legal digest: ${identity}`)
    }
  }
  if (manifest.goVendor?.treeSha256 !== (await treeDigest(vendorRoot, ports))) {
    throw new Error('source manifest has the wrong complete vendor-tree digest')
  }

  const trackedText = await readRequired(
    path.join(root, 'TRACKED_FILES.sha256'),
    'TRACKED_FILES.sha256',
    ports,
    'utf8',
  )
  const recordedTracked = parseTrackedFiles(trackedText)
  if (!mapsEqual(recordedTracked, trackedFiles)) {
    throw new Error('TRACKED_FILES.sha256 does not list every tracked app file exactly')
  }
  for (const [relative, expectedDigest] of trackedFiles) {
    let content
    try {
      content = await ports.readFile(path.join(appRoot, relative))
    } catch (error) {
      if (error.code === 'ENOENT') {
        throw new Error(`tracked application file is missing: ${relative}`)
      }
      throw error
    }
    if (sha256(content) !== expectedDigest) {
      throw new Error(`tracked application file has changed: ${relative}`)
    }
  }
  if (
    manifest.trackedFiles?.path !== 'TRACKED_FILES.sha256' ||
    manifest.trackedFiles?.count !== trackedFiles.size ||
    manifest.trackedFiles?.sha256 !== sha256(trackedText)
  ) {
    throw new Error('source manifest has the wrong tracked-file inventory')
  }
}

function isBeneath(parent, child) {
  const relative = path.relative(parent, child)
  return relative !== '' && relative !== '..' && !relative.startsWith(`..${path.sep}`) && !path.isAbsolute(relative)
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
    -ldflags "-X chess-trainer/internal/buildinfo.Commit=${commit}" \\
    -tags ""
)
\`\`\`

Keep \`GOWORK=off\`, \`GOTOOLCHAIN=local\`, and use \`-mod=vendor\` for direct
Go commands against the application source. The frontend install is fixed by
\`frontend/package-lock.json\`; do not replace it with a floating install.
`
}

async function collectTrackedFiles(appRoot, relativeFiles, ports) {
  const tracked = new Map()
  for (const relative of [...relativeFiles].sort()) {
    tracked.set(relative, sha256(await ports.readFile(path.join(appRoot, relative))))
  }
  return tracked
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
  const releaseRoot = path.resolve(assertReleaseEnvironment(env))
  const realReleaseRoot = await ports.realpath(releaseRoot)
  for (const name of ['GOMODCACHE', 'GOCACHE', 'npm_config_cache']) {
    const realCache = await ports.realpath(env[name])
    if (!isBeneath(realReleaseRoot, realCache)) {
      throw new Error(`${name} must resolve beneath CHESS_TRAINER_RELEASE_ROOT`)
    }
  }

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
    await verifyCorrespondingSourceTree({
      sourceRoot,
      tag: exactTag,
      commit: exactCommit,
      trackedFiles,
      runtimeLock,
      runtimeLockSha256: sha256(runtimeLockContent),
      goLegal,
      wailsTreeSha256,
      frontendSources: expectedFrontend,
      ports,
    })

    const outputPath = path.isAbsolute(output)
      ? output
      : path.resolve(repositoryRoot, output)
    await ports.mkdir(path.dirname(outputPath), { recursive: true })
    await ports.run(
      'tar',
      ['-czf', outputPath, '-C', payloadRoot, artifactName],
      { cwd: repositoryRoot, env },
    )
    await verifyCorrespondingSourceArchive({
      archive: outputPath,
      temporaryRoot: realReleaseRoot,
      tag: exactTag,
      commit: exactCommit,
      trackedFiles,
      runtimeLock,
      runtimeLockSha256: sha256(runtimeLockContent),
      goLegal,
      wailsTreeSha256,
      frontendSources: expectedFrontend,
      ports,
    })
    return { output: outputPath, manifest }
  } finally {
    await makeTreeWritable(workspace, ports)
    await ports.rm(workspace, { recursive: true, force: true })
  }
}

export async function verifyCorrespondingSourceArchive({
  archive,
  temporaryRoot,
  ports: providedPorts,
  ...expected
}) {
  const ports = { ...defaultPorts(), ...providedPorts }
  const workspace = await ports.mkdtemp(
    path.join(path.resolve(temporaryRoot), 'corresponding-source-verify-'),
  )
  try {
    await ports.run('tar', ['-xzf', archive, '-C', workspace], {})
    const roots = (await ports.readdir(workspace, { withFileTypes: true })).filter(
      (entry) => entry.isDirectory(),
    )
    if (roots.length !== 1) {
      throw new Error('corresponding-source archive must contain one root directory')
    }
    await verifyCorrespondingSourceTree({
      sourceRoot: path.join(workspace, roots[0].name),
      ports,
      ...expected,
    })
  } finally {
    await makeTreeWritable(workspace, ports)
    await ports.rm(workspace, { recursive: true, force: true })
  }
}

function parseArguments(argv) {
  const values = {}
  for (let index = 0; index < argv.length; index += 2) {
    const key = argv[index]
    const value = argv[index + 1]
    if (!key?.startsWith('--') || value === undefined) {
      throw new Error('usage: build-corresponding-source.mjs --tag <tag> --commit <sha> --output <archive>')
    }
    values[key.slice(2)] = value
  }
  assertReleaseTag(values.tag)
  assertFullCommit(values.commit)
  if (!values.output) throw new Error('--output is required')
  return values
}

const isCLI =
  process.argv[1] &&
  path.resolve(process.argv[1]) === fileURLToPath(import.meta.url)

if (isCLI) {
  try {
    const values = parseArguments(process.argv.slice(2))
    await buildCorrespondingSource({
      root: path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..'),
      tag: values.tag,
      commit: values.commit,
      output: values.output,
    })
  } catch (error) {
    console.error(error.message)
    process.exitCode = 1
  }
}
