import { mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { fileURLToPath } from 'node:url'

const sampleRate = 44_100
const outputDirectory = fileURLToPath(new URL('../src/assets/sounds/', import.meta.url))
const checking = process.argv.includes('--check')

const cues = {
  move: [tap(330, 0.055, 0.58)],
  capture: [tap(310, 0.042, 0.62), silence(0.008), tap(220, 0.042, 0.66)],
  correct: [
    note(523.25, 0.055, 0.38),
    silence(0.012),
    note(659.25, 0.055, 0.38),
    silence(0.012),
    note(783.99, 0.061, 0.42)
  ],
  incorrect: [note(246.94, 0.075, 0.42), silence(0.012), note(174.61, 0.075, 0.46)]
}

const generated = Object.fromEntries(
  Object.entries(cues).map(([name, segments]) => [`${name}.wav`, renderWav(segments)])
)

if (checking) {
  const mismatches = []
  for (const [filename, expected] of Object.entries(generated)) {
    try {
      if (!readFileSync(`${outputDirectory}${filename}`).equals(expected)) mismatches.push(filename)
    } catch {
      mismatches.push(filename)
    }
  }
  if (mismatches.length > 0) {
    console.error(`Generated sound assets differ: ${mismatches.join(', ')}`)
    process.exitCode = 1
  }
} else {
  mkdirSync(outputDirectory, { recursive: true })
  for (const [filename, bytes] of Object.entries(generated)) {
    writeFileSync(`${outputDirectory}${filename}`, bytes)
  }
}

function tap(frequency, duration, amplitude) {
  return { frequency, duration, amplitude, decay: 5.2, harmonic: 0.38 }
}

function note(frequency, duration, amplitude) {
  return { frequency, duration, amplitude, decay: 0.65, harmonic: 0.12 }
}

function silence(duration) {
  return { duration, amplitude: 0 }
}

function renderWav(segments) {
  const sampleCount = segments.reduce(
    (total, segment) => total + Math.round(segment.duration * sampleRate),
    0
  )
  const bytes = Buffer.alloc(44 + sampleCount * 2)
  writeHeader(bytes, sampleCount)
  let offset = 44
  for (const segment of segments) {
    const segmentSamples = Math.round(segment.duration * sampleRate)
    for (let index = 0; index < segmentSamples; index++) {
      const sample = segment.amplitude === 0 ? 0 : renderSample(segment, index, segmentSamples)
      bytes.writeInt16LE(Math.round(clamp(sample, -1, 1) * 32_767), offset)
      offset += 2
    }
  }
  return bytes
}

function renderSample(segment, index, sampleCount) {
  const time = index / sampleRate
  const progress = index / Math.max(1, sampleCount - 1)
  const attack = Math.min(1, time / 0.004)
  const release = Math.min(1, (1 - progress) / 0.16)
  const decay = Math.exp(-segment.decay * progress)
  const fundamental = Math.sin(2 * Math.PI * segment.frequency * time)
  const harmonic = Math.sin(4 * Math.PI * segment.frequency * time + Math.PI / 7)
  return segment.amplitude * attack * release * decay
    * (fundamental + segment.harmonic * harmonic)
}

function writeHeader(bytes, sampleCount) {
  const dataSize = sampleCount * 2
  bytes.write('RIFF', 0, 'ascii')
  bytes.writeUInt32LE(36 + dataSize, 4)
  bytes.write('WAVE', 8, 'ascii')
  bytes.write('fmt ', 12, 'ascii')
  bytes.writeUInt32LE(16, 16)
  bytes.writeUInt16LE(1, 20)
  bytes.writeUInt16LE(1, 22)
  bytes.writeUInt32LE(sampleRate, 24)
  bytes.writeUInt32LE(sampleRate * 2, 28)
  bytes.writeUInt16LE(2, 32)
  bytes.writeUInt16LE(16, 34)
  bytes.write('data', 36, 'ascii')
  bytes.writeUInt32LE(dataSize, 40)
}

function clamp(value, minimum, maximum) {
  return Math.max(minimum, Math.min(maximum, value))
}
