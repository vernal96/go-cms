// Match Vue Router's parameter spelling, retaining modifiers and constraints.
// Route names and parameter names do not distinguish URL patterns.
export function routeShape(path: string): string {
  let shape = ''
  for (let index = 0; index < path.length; index++) {
    const char = path[index]!
    if (char === '\\') {
      shape += char + (path[++index] ?? '')
      continue
    }
    if (char !== ':') { shape += char.toLowerCase(); continue }
    shape += ':'
    while (/[a-zA-Z0-9_]/.test(path[index + 1] ?? '') && index + 1 < path.length) index++
    if (path[index + 1] === '(') {
      // Vue Router terminates custom regexps at the first unescaped ')'.
      do {
        const current = path[++index]!
        shape += current
        if (current === '\\') shape += path[++index] ?? ''
        else if (current === ')') break
      } while (index + 1 < path.length)
    }
  }
  // The application uses Vue Router's default insensitive, non-strict mode.
  // Keep regexp text case intact; callers should use the same spelling for
  // constraints. Static path case is normalized while scanning.
  return shape.replace(/\/$/, '') || '/'
}
