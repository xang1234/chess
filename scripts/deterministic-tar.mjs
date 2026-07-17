import { lstat, mkdir, readFile, readdir, writeFile } from 'node:fs/promises'
import path from 'node:path'
import { gzipSync } from 'node:zlib'

const BLOCK_SIZE = 512
const TAR_END_SIZE = 2 * BLOCK_SIZE
const MAX_TAR_SIZE = 0o77777777777

function compareText(left, right) {
  if (left < right) return -1
  if (left > right) return 1
  return 0
}

function validateRootName(rootName) {
  if (
    typeof rootName !== 'string' ||
    rootName.length === 0 ||
    rootName.includes('\0') ||
    path.posix.isAbsolute(rootName) ||
    path.posix.normalize(rootName) !== rootName ||
    rootName === '.' ||
    rootName === '..' ||
    rootName.startsWith('../')
  ) {
    throw new Error('tar root name must be a normalized relative path')
  }
  return rootName
}

function portablePath(filename) {
  return filename.split(path.sep).join('/')
}

async function collectEntries(sourceDirectory, rootName) {
  const source = path.resolve(sourceDirectory)
  const sourceDetails = await lstat(source)
  if (!sourceDetails.isDirectory()) {
    throw new Error('tar source must be a directory')
  }

  const entries = [
    {
      archivePath: rootName,
      sourcePath: source,
      type: 'directory',
      executable: true,
    },
  ]

  async function visit(directory, relativeDirectory = '') {
    const children = await readdir(directory, { withFileTypes: true })
    children.sort((left, right) => compareText(left.name, right.name))
    for (const child of children) {
      const relative = path.join(relativeDirectory, child.name)
      const sourcePath = path.join(directory, child.name)
      const archivePath = `${rootName}/${portablePath(relative)}`
      const details = await lstat(sourcePath)
      if (details.isDirectory()) {
        entries.push({
          archivePath,
          sourcePath,
          type: 'directory',
          executable: true,
        })
        await visit(sourcePath, relative)
      } else if (details.isFile()) {
        entries.push({
          archivePath,
          sourcePath,
          type: 'file',
          executable: (details.mode & 0o111) !== 0,
        })
      } else {
        throw new Error(
          `tar source may contain only regular files and directories: ${archivePath}`,
        )
      }
    }
  }

  await visit(source)
  entries.sort((left, right) => compareText(left.archivePath, right.archivePath))
  return entries
}

function splitUstarPath(archivePath) {
  if (Buffer.byteLength(archivePath) <= 100) {
    return { name: archivePath, prefix: '' }
  }

  for (
    let separator = archivePath.lastIndexOf('/');
    separator > 0;
    separator = archivePath.lastIndexOf('/', separator - 1)
  ) {
    const prefix = archivePath.slice(0, separator)
    const name = archivePath.slice(separator + 1)
    if (Buffer.byteLength(prefix) <= 155 && Buffer.byteLength(name) <= 100) {
      return { name, prefix }
    }
  }
  return undefined
}

function writeString(header, offset, length, value) {
  const content = Buffer.from(value, 'utf8')
  if (content.length > length) throw new Error(`tar field is too long: ${value}`)
  content.copy(header, offset)
}

function writeOctal(header, offset, length, value) {
  if (!Number.isSafeInteger(value) || value < 0) {
    throw new Error('tar numeric field must be a non-negative safe integer')
  }
  const digits = value.toString(8)
  if (digits.length > length - 1) throw new Error('tar numeric field is too large')
  writeString(header, offset, length, `${digits.padStart(length - 1, '0')}\0`)
}

function createHeader({ archivePath, mode, size, type }) {
  const pathFields = splitUstarPath(archivePath)
  if (!pathFields) throw new Error(`tar path requires an extended header: ${archivePath}`)
  if (size > MAX_TAR_SIZE) throw new Error(`tar entry is too large: ${archivePath}`)

  const header = Buffer.alloc(BLOCK_SIZE)
  writeString(header, 0, 100, pathFields.name)
  writeOctal(header, 100, 8, mode)
  writeOctal(header, 108, 8, 0)
  writeOctal(header, 116, 8, 0)
  writeOctal(header, 124, 12, size)
  writeOctal(header, 136, 12, 0)
  header.fill(0x20, 148, 156)
  writeString(header, 156, 1, type)
  writeString(header, 257, 6, 'ustar\0')
  writeString(header, 263, 2, '00')
  writeString(header, 265, 32, 'root')
  writeString(header, 297, 32, 'root')
  writeOctal(header, 329, 8, 0)
  writeOctal(header, 337, 8, 0)
  writeString(header, 345, 155, pathFields.prefix)

  const checksum = header.reduce((sum, byte) => sum + byte, 0)
  const checksumDigits = checksum.toString(8)
  if (checksumDigits.length > 6) throw new Error('tar checksum is too large')
  writeString(header, 148, 6, checksumDigits.padStart(6, '0'))
  header[154] = 0
  header[155] = 0x20
  return header
}

function paddedContent(content) {
  const padding = (BLOCK_SIZE - (content.length % BLOCK_SIZE)) % BLOCK_SIZE
  return padding === 0 ? [content] : [content, Buffer.alloc(padding)]
}

function paxRecord(key, value) {
  const body = `${key}=${value}\n`
  let length = Buffer.byteLength(body) + 2
  while (true) {
    const exactLength = Buffer.byteLength(body) + String(length).length + 1
    if (exactLength === length) return Buffer.from(`${length} ${body}`, 'utf8')
    length = exactLength
  }
}

function appendTarEntry(chunks, header, content) {
  chunks.push(header, ...paddedContent(content))
}

function canonicalGzip(tar) {
  const gzip = Buffer.from(gzipSync(tar, { level: 9, mtime: 0 }))
  if (
    gzip[0] !== 0x1f ||
    gzip[1] !== 0x8b ||
    gzip[2] !== 0x08 ||
    gzip[3] !== 0x00
  ) {
    throw new Error('gzip encoder returned an unsupported header')
  }
  gzip.writeUInt32LE(0, 4)
  gzip[8] = 0x02
  gzip[9] = 0xff
  return gzip
}

export async function createDeterministicTarGzip({
  sourceDirectory,
  outputFile,
  rootName,
}) {
  const canonicalRootName = validateRootName(rootName)
  const entries = await collectEntries(sourceDirectory, canonicalRootName)
  const chunks = []
  let extendedHeaderIndex = 0

  for (const entry of entries) {
    const content = entry.type === 'file'
      ? await readFile(entry.sourcePath)
      : Buffer.alloc(0)
    let headerPath = entry.archivePath
    if (!splitUstarPath(headerPath)) {
      extendedHeaderIndex += 1
      const identity = String(extendedHeaderIndex).padStart(8, '0')
      const extended = paxRecord('path', headerPath)
      appendTarEntry(
        chunks,
        createHeader({
          archivePath: `PaxHeaders/${identity}`,
          mode: 0o644,
          size: extended.length,
          type: 'x',
        }),
        extended,
      )
      headerPath = `PaxEntries/${identity}`
    }

    appendTarEntry(
      chunks,
      createHeader({
        archivePath: headerPath,
        mode: entry.type === 'directory' || entry.executable ? 0o755 : 0o644,
        size: content.length,
        type: entry.type === 'directory' ? '5' : '0',
      }),
      content,
    )
  }

  chunks.push(Buffer.alloc(TAR_END_SIZE))
  const compressed = canonicalGzip(Buffer.concat(chunks))
  const absoluteOutput = path.resolve(outputFile)
  await mkdir(path.dirname(absoluteOutput), { recursive: true })
  await writeFile(absoluteOutput, compressed)
  return absoluteOutput
}
