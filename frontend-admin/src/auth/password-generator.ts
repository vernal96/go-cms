const CHARACTER_SETS = [
  'ABCDEFGHJKLMNPQRSTUVWXYZ',
  'abcdefghijkmnopqrstuvwxyz',
  '23456789',
  '!@#$%&*+-_=',
] as const

export function generatePassword(length = 20): string {
  if (length < CHARACTER_SETS.length) throw new Error('password length is too small')
  const characters = CHARACTER_SETS.join('')
  const result = CHARACTER_SETS.map((set) => set[randomIndex(set.length)])
  while (result.length < length) result.push(characters[randomIndex(characters.length)])
  for (let index = result.length - 1; index > 0; index--) {
    const swap = randomIndex(index + 1)
    ;[result[index], result[swap]] = [result[swap], result[index]]
  }
  return result.join('')
}

function randomIndex(length: number): number {
  const limit = Math.floor(256 / length) * length
  const buffer = new Uint8Array(1)
  do crypto.getRandomValues(buffer); while (buffer[0] >= limit)
  return buffer[0] % length
}
