import assert from 'node:assert/strict'
import { execFile } from 'node:child_process'
import { mkdtemp, readFile, readdir, rm, stat } from 'node:fs/promises'
import path from 'node:path'
import { promisify } from 'node:util'

import { assertNoLocalModuleReplacement } from './go-module-policy.mjs'
import {
  RELEASE_GO_VERSION,
  RELEASE_TARGET_ENVIRONMENT_VARIABLES,
  WAILS_MODULE,
  WAILS_VERSION,
  assertFullCommit,
  assertReleaseTag,
  releaseVersionFromTag,
} from './release-policy.mjs'
import {
  assertDigest,
  makeTreeWritable,
  mapsEqual,
  parseTrackedFiles,
  readRequired,
  requireDirectory,
  requireFile,
  sha256,
  treeDigest,
} from './tree-integrity.mjs'

const execFileAsync = promisify(execFile)

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
  return { run: defaultRunner, mkdtemp, readFile, readdir, rm, stat }
}

export function escapeModulePath(modulePath) {
  return modulePath.replaceAll('!', '!!').replaceAll('/', '!')
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
  const releaseVersion = releaseVersionFromTag(exactTag)
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
    '-platform darwin/arm64',
    '-m',
    '-nosyncgomod',
    '-platform darwin/arm64 -m -nosyncgomod',
  ]) {
    if (!building.includes(required)) {
      throw new Error(`BUILDING.md does not explain required input: ${required}`)
    }
  }
  for (const key of ['CFBundleShortVersionString', 'CFBundleVersion']) {
    const required = `plutil -replace ${key} -string "${releaseVersion}"`
    if (!building.includes(required)) {
      throw new Error(`BUILDING.md does not explain required input: ${key}`)
    }
  }
  const targetEnvironmentUnset = `unset ${RELEASE_TARGET_ENVIRONMENT_VARIABLES.join(' ')}`
  if (!building.split(/\r?\n/).includes(targetEnvironmentUnset)) {
    throw new Error('BUILDING.md does not clear release target environment')
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
      throw new Error(
        `source manifest has the wrong frontend archive digest: ${relative}`,
      )
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
