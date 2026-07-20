import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import path from 'node:path'
import test from 'node:test'
import { fileURLToPath } from 'node:url'

const repositoryRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..')
const backendFile = path.join(repositoryRoot, 'frontend/tests/test-backend.ts')
const backendTypesFile = path.join(repositoryRoot, 'frontend/tests/test-backend-types.ts')

test('browser backend derives bindings and defines one shared controller surface', async () => {
  const [source, typeSource] = await Promise.all([
    readFile(backendFile, 'utf8'),
    readFile(backendTypesFile, 'utf8')
  ])
  const completeSource = `${typeSource}\n${source}`

  assert.doesNotMatch(completeSource, /\b(?:runtime|go):\s*unknown\b/)
  assert.match(typeSource, /typeof import\('\.\.\/wailsjs\/go\/main\/NormalController'\)/)
  assert.equal(
    [...source.matchAll(/\bconst normalController\b/g)].length,
    1,
    'the Wails NormalController mock surface must be declared once'
  )
  assert.match(
    typeSource,
    /type NextResponse\s*=\s*\{[^}]*\bnext:\s*PuzzleDefinition/s,
    'a next response must carry its next puzzle in the discriminated variant'
  )
  assert.doesNotMatch(
    typeSource,
    /type BoardScenario\s*=\s*\{[^}]*\bnext\?:/s,
    'the next puzzle must not be independently optional on a board scenario'
  )
})
