import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const scriptsRoot = path.dirname(fileURLToPath(import.meta.url))

async function source(name) {
  return readFile(path.join(scriptsRoot, name), 'utf8')
}

test('release CLIs use acyclic policy, integrity, and source-verifier modules', async () => {
  const [releaseVerifier, sourceBuilder] = await Promise.all([
    source('verify-release.mjs'),
    source('build-corresponding-source.mjs'),
  ])

  assert.doesNotMatch(
    sourceBuilder,
    /from\s+['"]\.\/verify-release\.mjs['"]/,
    'the source builder must not import the release verifier CLI',
  )
  assert.doesNotMatch(
    releaseVerifier,
    /import\(\s*['"]\.\/build-corresponding-source\.mjs['"]\s*\)/,
    'the release verifier must not dynamically import the source builder CLI',
  )

  for (const moduleName of [
    'release-policy.mjs',
    'tree-integrity.mjs',
    'corresponding-source-verifier.mjs',
  ]) {
    const moduleSource = await source(moduleName)
    assert.ok(moduleSource.trim(), `${moduleName} must not be empty`)
  }
})

test('local build instructions run the complete release script test suite', async () => {
  const localBuild = await readFile(
    path.resolve(scriptsRoot, '../docs/operations/local-build.md'),
    'utf8',
  )
  assert.match(localBuild, /node --test scripts\/\*\.test\.mjs/)
})
