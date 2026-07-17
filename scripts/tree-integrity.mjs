import { createHash } from 'node:crypto'
import { chmod, readFile, readdir, stat } from 'node:fs/promises'
import path from 'node:path'

function defaultPorts() {
  return { chmod, readFile, readdir, stat }
}

export function compareText(left, right) {
  if (left < right) return -1
  if (left > right) return 1
  return 0
}

export function sha256(content) {
  return createHash('sha256').update(content).digest('hex')
}

export function assertDigest(label, content, expected) {
  if (sha256(content) !== expected) {
    throw new Error(`${label} has an unexpected SHA-256`)
  }
}

async function filesBelow(root, directory, ports) {
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

export function renderTrackedFiles(trackedFiles) {
  return [...trackedFiles.entries()]
    .sort(([left], [right]) => compareText(left, right))
    .map(([filename, digest]) => `${digest}  ${filename}\n`)
    .join('')
}

export function parseTrackedFiles(text) {
  const tracked = new Map()
  for (const line of text.split(/\r?\n/)) {
    if (!line) continue
    const match = /^([0-9a-f]{64})  (.+)$/.exec(line)
    if (!match) {
      throw new Error(`TRACKED_FILES.sha256 has malformed line: ${line}`)
    }
    if (tracked.has(match[2])) {
      throw new Error(`TRACKED_FILES.sha256 repeats path: ${match[2]}`)
    }
    tracked.set(match[2], match[1])
  }
  return tracked
}

export function mapsEqual(left, right) {
  if (left.size !== right.size) return false
  for (const [key, value] of left) {
    if (right.get(key) !== value) return false
  }
  return true
}

export async function readRequired(filename, label, ports, encoding) {
  try {
    return await ports.readFile(filename, encoding)
  } catch (error) {
    if (error.code === 'ENOENT') throw new Error(`${label} is missing`)
    throw error
  }
}

export async function requireDirectory(filename, label, ports) {
  try {
    const details = await ports.stat(filename)
    if (!details.isDirectory()) throw new Error(`${label} is missing`)
  } catch (error) {
    if (error.code === 'ENOENT') throw new Error(`${label} is missing`)
    throw error
  }
}

export async function requireFile(filename, label, ports) {
  try {
    const details = await ports.stat(filename)
    if (!details.isFile()) throw new Error(`${label} is missing`)
  } catch (error) {
    if (error.code === 'ENOENT') throw new Error(`${label} is missing`)
    throw error
  }
}

export async function collectTrackedFiles(appRoot, relativeFiles, providedPorts) {
  const ports = { ...defaultPorts(), ...providedPorts }
  const tracked = new Map()
  for (const relative of [...relativeFiles].sort(compareText)) {
    tracked.set(relative, sha256(await ports.readFile(path.join(appRoot, relative))))
  }
  return tracked
}
