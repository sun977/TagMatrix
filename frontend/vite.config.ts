import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'
import * as fs from 'fs'
import * as path from 'path'

const cwd = process.cwd()
const wailsJsonPath1 = path.join(cwd, 'wails.json')
const wailsJsonPath2 = path.join(cwd, '../wails.json')
const wailsJsonPath = fs.existsSync(wailsJsonPath1) ? wailsJsonPath1 : wailsJsonPath2
let appVersion = '1.0.0'
let authorName = 'Author'

try {
  const wailsJson = JSON.parse(fs.readFileSync(wailsJsonPath, 'utf-8'))
  if (wailsJson.info && wailsJson.info.productVersion) {
    appVersion = wailsJson.info.productVersion
  }
  if (wailsJson.author && wailsJson.author.name) {
    authorName = wailsJson.author.name
  }
} catch (e) {
  console.error('Failed to read wails.json', e)
}

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],
  define: {
    __APP_VERSION__: JSON.stringify(appVersion),
    __APP_AUTHOR__: JSON.stringify(authorName)
  }
})
