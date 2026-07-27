import { fakeOpeningSession } from '../../test-fakes'
import { decodeOpeningSession } from './openings'

test('opening activity decoder keeps aligned route SAN', () => {
  const current = {
    ...fakeOpeningSession.current,
    movesToHere: [
      { uci: 'e2e4', resultingFen: 'after-e4' },
      { uci: 'e7e5', resultingFen: 'after-e5' }
    ],
    movesToHereSan: ['e4', 'e5']
  }
  expect(decodeOpeningSession({ ...fakeOpeningSession, current })).toMatchObject({
    current: { movesToHereSan: ['e4', 'e5'] }
  })
})

test('opening activity decoder rejects route SAN with a different length', () => {
  expect(() => decodeOpeningSession({
    ...fakeOpeningSession,
    current: { ...fakeOpeningSession.current, movesToHereSan: ['e4'] }
  })).toThrow('movesToHereSan must align with movesToHere')
})

test('opening decoder keeps authoritative comparison frames', () => {
  const comparison = [{
    label: 'Active centre',
    moves: ['c3', 'd4'],
    frames: [
      { uci: 'c2c3', resultingFen: 'after-c3' },
      { uci: 'd2d4', resultingFen: 'after-d4' }
    ]
  }]
  expect(decodeOpeningSession({
    ...fakeOpeningSession,
    current: { ...fakeOpeningSession.current, kind: 'comparison', comparison }
  })).toMatchObject({ current: { comparison } })
})

test('opening decoder rejects mismatched comparison frames', () => {
  expect(() => decodeOpeningSession({
    ...fakeOpeningSession,
    current: {
      ...fakeOpeningSession.current,
      kind: 'comparison',
      comparison: [{
        label: 'Mismatch',
        moves: ['c3', 'd4'],
        frames: [{ uci: 'c2c3', resultingFen: 'after-c3' }]
      }]
    }
  })).toThrow('frames must align with moves')
})

test('opening decoder requires authoritative comparison frames', () => {
  expect(() => decodeOpeningSession({
    ...fakeOpeningSession,
    current: {
      ...fakeOpeningSession.current,
      kind: 'comparison',
      comparison: [{ label: 'Missing', moves: ['c3'], frames: [] }]
    }
  })).toThrow('authoritative move frames')
})
