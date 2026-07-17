export type Piece = {
  color: 'white' | 'black'
  role: 'pawn' | 'knight' | 'bishop' | 'rook' | 'queen' | 'king'
}

export type BoardMap = Record<string, Piece>

const roles: Record<string, Piece['role']> = {
  p: 'pawn',
  n: 'knight',
  b: 'bishop',
  r: 'rook',
  q: 'queen',
  k: 'king'
}

export function parseFEN(fen: string): BoardMap {
  const placement = fen.trim().split(/\s+/)[0]
  const ranks = placement?.split('/') ?? []
  if (ranks.length !== 8) throw new Error(`FEN has ${ranks.length} ranks, want 8`)

  const board: BoardMap = {}
  ranks.forEach((rank, rankIndex) => {
    let fileIndex = 0
    for (const token of rank) {
      if (/^[1-8]$/.test(token)) {
        fileIndex += Number(token)
        continue
      }
      const role = roles[token.toLowerCase()]
      if (!role) throw new Error(`FEN rank ${8 - rankIndex} contains invalid piece ${token}`)
      if (fileIndex >= 8) throw new Error(`FEN rank ${8 - rankIndex} exceeds eight files`)
      const square = `${String.fromCharCode(97 + fileIndex)}${8 - rankIndex}`
      board[square] = {
        color: token === token.toUpperCase() ? 'white' : 'black',
        role
      }
      fileIndex++
    }
    if (fileIndex !== 8) throw new Error(`FEN rank ${8 - rankIndex} totals ${fileIndex} files, want 8`)
  })
  return board
}

export function orientSquares(color: 'white' | 'black'): string[] {
  const ranks = color === 'white' ? [8, 7, 6, 5, 4, 3, 2, 1] : [1, 2, 3, 4, 5, 6, 7, 8]
  const files = color === 'white'
    ? ['a', 'b', 'c', 'd', 'e', 'f', 'g', 'h']
    : ['h', 'g', 'f', 'e', 'd', 'c', 'b', 'a']
  return ranks.flatMap((rank) => files.map((file) => `${file}${rank}`))
}

export function describeSquare(square: string, piece: Piece | undefined): string {
  if (!piece) return `Empty ${square}`
  const color = piece.color === 'white' ? 'White' : 'Black'
  return `${color} ${piece.role} on ${square}`
}
