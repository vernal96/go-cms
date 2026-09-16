import { readFile, stat } from 'node:fs/promises'
import { fileURLToPath } from 'node:url'

const root = new URL('../', import.meta.url)
const manifest = JSON.parse(await readFile(new URL('package.json', root), 'utf8'))

async function verify(value) {
  if (typeof value === 'string') {
    if (!value.startsWith('./dist-sdk/')) {
      throw new Error(`SDK export must belong to dist-sdk: ${value}`)
    }
    const path = new URL(value, root)
    const info = await stat(path)
    if (!info.isFile() || info.size === 0) {
      throw new Error(`Missing or empty SDK export: ${fileURLToPath(path)}`)
    }
    return
  }
  for (const target of Object.values(value)) await verify(target)
}

await verify(manifest.exports)
console.log('SDK JavaScript, declarations and styles are ready for packaging')
