import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const scssPath = path.resolve(__dirname, '../src/css/app.scss')
const content = fs.readFileSync(scssPath, 'utf8')

function extractTokens(mode: 'light' | 'dark'): Record<string, string> {
  const tokens: Record<string, string> = {}
  const regex = new RegExp(`body\\.body--${mode}\\s*\\{[^}]+\\}`, 's')
  const match = content.match(regex)
  if (match) {
    const block = match[0]
    const varRegex = /(--[a-zA-Z0-9-]+):\s*(#[a-fA-F0-9]{3,8});/g
    let m
    while ((m = varRegex.exec(block)) !== null) {
      tokens[m[1]] = m[2]
    }
  }
  return tokens
}

const themes = {
  light: extractTokens('light'),
  dark: extractTokens('dark'),
}

const pairs = [
  ['--app-text', '--app-bg', 4.5],
  ['--app-text', '--app-surface', 4.5],
  ['--app-text-muted', '--app-bg', 4.5],
  ['--app-text-muted', '--app-surface', 4.5],
  ['--app-primary-text', '--q-primary', 4.5],
  ['--app-border', '--app-bg', 3.0]
] as const

function hexToRgb(hex: string) {
  const raw = hex.replace('#', '')
  const bigint = Number.parseInt(raw, 16)

  return {
    r: (bigint >> 16) & 255,
    g: (bigint >> 8) & 255,
    b: bigint & 255
  }
}

function channel(value: number) {
  const s = value / 255
  return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
}

function luminance(hex: string) {
  const { r, g, b } = hexToRgb(hex)

  return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b)
}

function contrast(fg: string, bg: string) {
  const l1 = luminance(fg)
  const l2 = luminance(bg)
  const lighter = Math.max(l1, l2)
  const darker = Math.min(l1, l2)

  return (lighter + 0.05) / (darker + 0.05)
}

let failed = false

for (const [themeName, tokens] of Object.entries(themes)) {
  console.log(`\nChecking ${themeName} theme:`)
  for (const [fgKey, bgKey, min] of pairs) {
    const fgColor = tokens[fgKey]
    const bgColor = tokens[bgKey]
    
    if (!fgColor || !bgColor) {
        failed = true
        console.error(`  ❌ Missing token! ${fgKey} or ${bgKey} is not defined.`)
        continue
    }

    const ratio = contrast(fgColor, bgColor)

    if (ratio < min) {
      failed = true
      console.error(
        `  ❌ ${fgKey} (${fgColor}) on ${bgKey} (${bgColor}) failed: ${ratio.toFixed(2)} < ${min}`
      )
    } else {
      console.log(
        `  ✅ ${fgKey} on ${bgKey} passed: ${ratio.toFixed(2)}`
      )
    }
  }
}

process.exit(failed ? 1 : 0)
