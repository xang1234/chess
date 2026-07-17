function decodeGoString(token) {
  if (token.startsWith('`') && token.endsWith('`')) {
    return token.slice(1, -1)
  }
  if (!token.startsWith('"') || !token.endsWith('"')) return token

  let decoded = ''
  for (let index = 1; index < token.length - 1; index += 1) {
    const character = token[index]
    if (character !== '\\') {
      decoded += character
      continue
    }

    index += 1
    const escape = token[index]
    const simple = {
      a: '\x07',
      b: '\b',
      f: '\f',
      n: '\n',
      r: '\r',
      t: '\t',
      v: '\v',
      '\\': '\\',
      '"': '"',
    }
    if (Object.hasOwn(simple, escape)) {
      decoded += simple[escape]
      continue
    }

    const widths = { x: 2, u: 4, U: 8 }
    if (Object.hasOwn(widths, escape)) {
      const width = widths[escape]
      const digits = token.slice(index + 1, index + 1 + width)
      if (!new RegExp(`^[0-9a-fA-F]{${width}}$`).test(digits)) return token
      decoded += String.fromCodePoint(Number.parseInt(digits, 16))
      index += width
      continue
    }

    if (/[0-7]/.test(escape)) {
      const digits = token.slice(index, index + 3)
      if (!/^[0-7]{3}$/.test(digits)) return token
      decoded += String.fromCodePoint(Number.parseInt(digits, 8))
      index += 2
      continue
    }
    return token
  }
  return decoded
}

function replacementTarget(line) {
  const arrow = line.indexOf('=>')
  if (arrow < 0) return undefined
  const right = line.slice(arrow + 2).trimStart()
  const match = /^("(?:\\[\s\S]|[^"\\])*"|`[^`]*`|[^\s)]+)/.exec(right)
  return match ? decodeGoString(match[1]) : undefined
}

function localPath(target) {
  return (
    target === '.' ||
    target === '..' ||
    target.startsWith('./') ||
    target.startsWith('../') ||
    target.startsWith('/') ||
    target.startsWith('~/') ||
    target.startsWith('\\\\') ||
    /^[A-Za-z]:[\\/]/.test(target)
  )
}

export function assertNoLocalModuleReplacement(goMod) {
  let replaceBlock = false
  for (const original of goMod.split(/\r?\n/)) {
    let line = original.trim()
    if (line.startsWith('//')) line = line.slice(2).trimStart()
    if (!line) continue

    if (replaceBlock) {
      if (line.startsWith(')')) {
        replaceBlock = false
        continue
      }
      const target = replacementTarget(line)
      if (target && localPath(target)) {
        throw new Error('go.mod contains a local or absolute Go module replacement')
      }
      continue
    }

    if (!/^replace(?:\s|\()/.test(line)) continue
    const rest = line.slice('replace'.length).trimStart()
    if (rest.startsWith('(')) {
      replaceBlock = true
      continue
    }
    const target = replacementTarget(rest)
    if (target && localPath(target)) {
      throw new Error('go.mod contains a local or absolute Go module replacement')
    }
  }
}
