export type UnknownRecord = Record<string, unknown>

export function record(value: unknown, path: string): UnknownRecord {
  if (typeof value !== 'object' || value === null || Array.isArray(value)) {
    throw new Error(`${path} must be an object`)
  }
  return value as UnknownRecord
}

export function numberRecord(value: unknown, path: string): Record<string, number> {
  const raw = record(value, path)
  const decoded: Record<string, number> = {}
  for (const [key, entry] of Object.entries(raw)) {
    if (typeof entry !== 'number' || !Number.isFinite(entry) ||
      !Number.isInteger(entry) || entry < 0) {
      throw new Error(`${path}.${key} must be a non-negative integer`)
    }
    decoded[key] = entry
  }
  return decoded
}

export function string(value: unknown, path: string): string {
  if (typeof value !== 'string') throw new Error(`${path} must be a string`)
  return value
}

export function optionalString(value: unknown, path: string): string | undefined {
  return value === undefined ? undefined : string(value, path)
}

export function number(value: unknown, path: string): number {
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    throw new Error(`${path} must be a finite number`)
  }
  return value
}

export function nonNegativeInteger(value: unknown, path: string): number {
  const decoded = number(value, path)
  if (!Number.isInteger(decoded) || decoded < 0) {
    throw new Error(`${path} must be a non-negative integer`)
  }
  return decoded
}

export function positiveInteger(value: unknown, path: string): number {
  const decoded = number(value, path)
  if (!Number.isInteger(decoded) || decoded <= 0) {
    throw new Error(`${path} must be a positive integer`)
  }
  return decoded
}

export function boolean(value: unknown, path: string): boolean {
  if (typeof value !== 'boolean') throw new Error(`${path} must be a boolean`)
  return value
}

export function array<Value>(
  value: unknown,
  path: string,
  decode: (entry: unknown, entryPath: string) => Value
): Value[] {
  if (!Array.isArray(value)) throw new Error(`${path} must be an array`)
  return value.map((entry, index) => decode(entry, `${path}[${index}]`))
}

export function enumeration<Value extends string>(
  value: unknown,
  allowed: readonly Value[],
  path: string
): Value {
  if (typeof value !== 'string' || !allowed.includes(value as Value)) {
    throw new Error(`${path} has unknown value ${JSON.stringify(value)}`)
  }
  return value as Value
}
