import assert from 'node:assert/strict'
import { execFile } from 'node:child_process'
import { createHash } from 'node:crypto'
import {
  chmod,
  mkdir,
  mkdtemp,
  readFile,
  rm,
  stat,
  symlink,
  utimes,
  writeFile,
} from 'node:fs/promises'
import { tmpdir } from 'node:os'
import path from 'node:path'
import test from 'node:test'
import { promisify } from 'node:util'
import { gunzipSync } from 'node:zlib'

import { createDeterministicTarGzip } from './deterministic-tar.mjs'

const execFileAsync = promisify(execFile)
const BLOCK_SIZE = 512

function digest(content) {
  return createHash('sha256').update(content).digest('hex')
}

function stringField(header, offset, length) {
  const field = header.subarray(offset, offset + length)
  const end = field.indexOf(0)
  return field.subarray(0, end === -1 ? field.length : end).toString('utf8')
}

function octalField(header, offset, length) {
  const value = stringField(header, offset, length).trim()
  return value ? Number.parseInt(value, 8) : 0
}

function paxFields(content) {
  const fields = new Map()
  let offset = 0
  while (offset < content.length) {
    const separator = content.indexOf(0x20, offset)
    assert.notEqual(separator, -1, 'PAX record must include a length separator')
    const length = Number.parseInt(
      content.subarray(offset, separator).toString('ascii'),
      10,
    )
    assert.ok(Number.isSafeInteger(length) && length > 0, 'PAX length must be valid')
    const end = offset + length
    assert.equal(content[end - 1], 0x0a, 'PAX record must end with a newline')
    const record = content.subarray(separator + 1, end - 1).toString('utf8')
    const equals = record.indexOf('=')
    assert.ok(equals > 0, 'PAX record must contain a key and value')
    fields.set(record.slice(0, equals), record.slice(equals + 1))
    offset = end
  }
  assert.equal(offset, content.length, 'PAX records must consume their payload')
  return fields
}

function tarEntries(gzip) {
  const archive = gunzipSync(gzip)
  assert.equal(archive.length % BLOCK_SIZE, 0)
  assert.ok(
    archive.subarray(-2 * BLOCK_SIZE).every((byte) => byte === 0),
    'tar must end with two empty records',
  )

  const entries = []
  let offset = 0
  let pendingPax
  let paxCount = 0
  while (!archive.subarray(offset, offset + BLOCK_SIZE).every((byte) => byte === 0)) {
    const header = archive.subarray(offset, offset + BLOCK_SIZE)
    const expectedChecksum = octalField(header, 148, 8)
    const checksumHeader = Buffer.from(header)
    checksumHeader.fill(0x20, 148, 156)
    const actualChecksum = checksumHeader.reduce((sum, byte) => sum + byte, 0)
    assert.equal(actualChecksum, expectedChecksum, 'tar checksum must be valid')

    const name = stringField(header, 0, 100)
    const prefix = stringField(header, 345, 155)
    const storedPath = prefix ? `${prefix}/${name}` : name
    const size = octalField(header, 124, 12)
    const type = stringField(header, 156, 1) || '0'
    const bodyOffset = offset + BLOCK_SIZE
    const body = archive.subarray(bodyOffset, bodyOffset + size)

    if (type === 'x') {
      pendingPax = paxFields(body)
      paxCount += 1
    } else {
      entries.push({
        path: pendingPax?.get('path') ?? storedPath,
        type,
        mode: octalField(header, 100, 8),
        uid: octalField(header, 108, 8),
        gid: octalField(header, 116, 8),
        mtime: octalField(header, 136, 12),
        user: stringField(header, 265, 32),
        group: stringField(header, 297, 32),
        content: Buffer.from(body),
      })
      pendingPax = undefined
    }
    offset = bodyOffset + Math.ceil(size / BLOCK_SIZE) * BLOCK_SIZE
  }
  assert.equal(pendingPax, undefined, 'PAX header must describe a following entry')
  return { entries, paxCount }
}

async function put(filename, content, mode) {
  await mkdir(path.dirname(filename), { recursive: true })
  await writeFile(filename, content)
  await chmod(filename, mode)
}

async function createEquivalentTree(root, variant) {
  const longDirectory = 'a'.repeat(120)
  const longChild = 'b'.repeat(120)
  await mkdir(path.join(root, 'empty'), { recursive: true, mode: variant ? 0o755 : 0o700 })
  await put(path.join(root, 'docs/readme.txt'), 'same contents\n', variant ? 0o644 : 0o600)
  await put(path.join(root, 'bin/run.sh'), '#!/bin/sh\nexit 0\n', variant ? 0o755 : 0o700)
  await put(
    path.join(root, longDirectory, longChild, 'source.txt'),
    'long path contents\n',
    variant ? 0o644 : 0o600,
  )
  await chmod(path.join(root, 'docs'), variant ? 0o755 : 0o700)
  await chmod(path.join(root, 'bin'), variant ? 0o755 : 0o700)
  await chmod(path.join(root, longDirectory), variant ? 0o755 : 0o700)
  await chmod(path.join(root, longDirectory, longChild), variant ? 0o755 : 0o700)

  const timestamp = variant ? new Date('2035-06-07T08:09:10Z') : new Date('2001-02-03T04:05:06Z')
  for (const relative of [
    '',
    'empty',
    'docs',
    'docs/readme.txt',
    'bin',
    'bin/run.sh',
    longDirectory,
    `${longDirectory}/${longChild}`,
    `${longDirectory}/${longChild}/source.txt`,
  ]) {
    await utimes(path.join(root, relative), timestamp, timestamp)
  }
  return { longDirectory, longChild }
}

test('equivalent trees produce byte-identical canonical tar.gz archives', async () => {
  const temporary = await mkdtemp(path.join(tmpdir(), 'deterministic-tar-'))
  const left = path.join(temporary, 'left')
  const right = path.join(temporary, 'right')
  const leftArchive = path.join(temporary, 'left.tar.gz')
  const rightArchive = path.join(temporary, 'right.tar.gz')
  await mkdir(left)
  await mkdir(right)
  await createEquivalentTree(left, false)
  await createEquivalentTree(right, true)

  try {
    await createDeterministicTarGzip({
      sourceDirectory: left,
      outputFile: leftArchive,
      rootName: 'release',
    })
    await createDeterministicTarGzip({
      sourceDirectory: right,
      outputFile: rightArchive,
      rootName: 'release',
    })

    const leftBytes = await readFile(leftArchive)
    const rightBytes = await readFile(rightArchive)
    assert.equal(digest(leftBytes), digest(rightBytes))
    assert.deepEqual(leftBytes, rightBytes)
    assert.deepEqual([...leftBytes.subarray(0, 10)], [
      0x1f,
      0x8b,
      0x08,
      0x00,
      0x00,
      0x00,
      0x00,
      0x00,
      0x02,
      0xff,
    ])
  } finally {
    await rm(temporary, { recursive: true, force: true })
  }
})

test('archive metadata and ordering are canonical and long paths use PAX safely', async () => {
  const temporary = await mkdtemp(path.join(tmpdir(), 'deterministic-tar-metadata-'))
  const source = path.join(temporary, 'source')
  const output = path.join(temporary, 'source.tar.gz')
  await mkdir(source)
  const { longDirectory, longChild } = await createEquivalentTree(source, false)

  try {
    await createDeterministicTarGzip({
      sourceDirectory: source,
      outputFile: output,
      rootName: 'release',
    })
    const { entries, paxCount } = tarEntries(await readFile(output))
    const paths = entries.map((entry) => entry.path)
    assert.deepEqual(paths, [...paths].sort())
    assert.ok(paxCount > 0, 'a path exceeding ustar limits must use PAX')

    const byPath = new Map(entries.map((entry) => [entry.path, entry]))
    const longPath = `release/${longDirectory}/${longChild}/source.txt`
    assert.equal(byPath.get(longPath)?.content.toString('utf8'), 'long path contents\n')
    for (const entry of entries) {
      assert.equal(entry.uid, 0)
      assert.equal(entry.gid, 0)
      assert.equal(entry.user, 'root')
      assert.equal(entry.group, 'root')
      assert.equal(entry.mtime, 0)
      assert.equal(entry.mode, entry.type === '5' || entry.path === 'release/bin/run.sh' ? 0o755 : 0o644)
    }
  } finally {
    await rm(temporary, { recursive: true, force: true })
  }
})

test('standard tar extraction preserves contents, empty directories, and executable semantics', async () => {
  const temporary = await mkdtemp(path.join(tmpdir(), 'deterministic-tar-extract-'))
  const source = path.join(temporary, 'source')
  const output = path.join(temporary, 'source.tar.gz')
  const extracted = path.join(temporary, 'extracted')
  await mkdir(source)
  await mkdir(extracted)
  const { longDirectory, longChild } = await createEquivalentTree(source, false)

  try {
    await createDeterministicTarGzip({
      sourceDirectory: source,
      outputFile: output,
      rootName: 'release',
    })
    await execFileAsync('tar', ['-xzf', output, '-C', extracted])

    const release = path.join(extracted, 'release')
    assert.equal(await readFile(path.join(release, 'docs/readme.txt'), 'utf8'), 'same contents\n')
    assert.equal(
      await readFile(path.join(release, longDirectory, longChild, 'source.txt'), 'utf8'),
      'long path contents\n',
    )
    assert.ok((await stat(path.join(release, 'empty'))).isDirectory())
    assert.notEqual((await stat(path.join(release, 'bin/run.sh'))).mode & 0o111, 0)
    assert.equal((await stat(path.join(release, 'docs/readme.txt'))).mode & 0o111, 0)
  } finally {
    await rm(temporary, { recursive: true, force: true })
  }
})

test('rejects source trees containing symbolic links', async () => {
  const temporary = await mkdtemp(path.join(tmpdir(), 'deterministic-tar-link-'))
  const source = path.join(temporary, 'source')
  await mkdir(source)
  await writeFile(path.join(source, 'target.txt'), 'target\n')
  await symlink('target.txt', path.join(source, 'link.txt'))

  try {
    await assert.rejects(
      createDeterministicTarGzip({
        sourceDirectory: source,
        outputFile: path.join(temporary, 'source.tar.gz'),
        rootName: 'release',
      }),
      /only regular files and directories.*link\.txt/i,
    )
  } finally {
    await rm(temporary, { recursive: true, force: true })
  }
})

test('rejects an archive root that could escape the extraction directory', async () => {
  const temporary = await mkdtemp(path.join(tmpdir(), 'deterministic-tar-root-'))
  const source = path.join(temporary, 'source')
  await mkdir(source)

  try {
    await assert.rejects(
      createDeterministicTarGzip({
        sourceDirectory: source,
        outputFile: path.join(temporary, 'source.tar.gz'),
        rootName: '..',
      }),
      /root name must be a normalized relative path/i,
    )
  } finally {
    await rm(temporary, { recursive: true, force: true })
  }
})
