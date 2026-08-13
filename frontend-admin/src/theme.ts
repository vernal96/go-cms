import type { AccentColor, ColorScheme } from './types/auth'

type ResolvedTheme = 'light' | 'dark'
type RGB = readonly [number, number, number]

export const accentColorOptions: ReadonlyArray<{
  code: AccentColor
  label: string
  color: string
}> = [
  { code: 'blue', label: 'Синий', color: '#409EFF' },
  { code: 'violet', label: 'Фиолетовый', color: '#8B5CF6' },
  { code: 'indigo', label: 'Индиго', color: '#6366F1' },
  { code: 'emerald', label: 'Зелёный', color: '#10B981' },
  { code: 'amber', label: 'Янтарный', color: '#F59E0B' },
  { code: 'rose', label: 'Розовый', color: '#F43F5E' },
]

let mediaQuery: MediaQueryList | null = null
let currentScheme: ColorScheme = 'system'
let currentAccent: AccentColor = 'blue'

function resolvedTheme(scheme: ColorScheme): ResolvedTheme {
  if (scheme !== 'system') return scheme
  return globalThis.matchMedia?.('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function setAppearance(): void {
  const resolved = resolvedTheme(currentScheme)
  document.documentElement.dataset.theme = resolved
  document.documentElement.classList.toggle('dark', resolved === 'dark')
  document.documentElement.style.colorScheme = resolved
  setAccentVariables(currentAccent, resolved)
}

export function applyAppearance(scheme: ColorScheme, accent: AccentColor): void {
  currentScheme = scheme
  currentAccent = accent
  subscribeToSystemTheme()
  setAppearance()
}

export function applyColorScheme(scheme: ColorScheme): void {
  currentScheme = scheme
  subscribeToSystemTheme()
  setAppearance()
}

export function applyAccentColor(accent: AccentColor): void {
  currentAccent = accent
  setAccentVariables(accent, resolvedTheme(currentScheme))
}

export function disposeColorScheme(): void {
  mediaQuery?.removeEventListener('change', handleSystemTheme)
  mediaQuery = null
}

function subscribeToSystemTheme(): void {
  disposeColorScheme()
  if (currentScheme !== 'system' || !globalThis.matchMedia) return
  mediaQuery = globalThis.matchMedia('(prefers-color-scheme: dark)')
  mediaQuery.addEventListener('change', handleSystemTheme)
}

function handleSystemTheme(): void {
  setAppearance()
}

function setAccentVariables(accent: AccentColor, theme: ResolvedTheme): void {
  const option = accentColorOptions.find((item) => item.code === accent) ?? accentColorOptions[0]
  const primary = hexToRGB(option.color)
  const lightTarget: RGB = theme === 'dark' ? [20, 20, 20] : [255, 255, 255]
  const darkTarget: RGB = theme === 'dark' ? [255, 255, 255] : [0, 0, 0]
  const root = document.documentElement

  root.dataset.accent = option.code
  root.style.setProperty('--admin-primary', option.color)
  root.style.setProperty('--el-color-primary', option.color)
  root.style.setProperty('--el-color-primary-rgb', primary.join(', '))
  for (const level of [3, 5, 7, 8, 9]) {
    root.style.setProperty(
      `--el-color-primary-light-${level}`,
      rgbToHex(mix(primary, lightTarget, level / 10)),
    )
  }
  root.style.setProperty('--el-color-primary-dark-2', rgbToHex(mix(primary, darkTarget, 0.2)))
}

function hexToRGB(hex: string): RGB {
  return [
    Number.parseInt(hex.slice(1, 3), 16),
    Number.parseInt(hex.slice(3, 5), 16),
    Number.parseInt(hex.slice(5, 7), 16),
  ]
}

function mix(source: RGB, target: RGB, amount: number): RGB {
  return source.map((channel, index) => Math.round(
    channel * (1 - amount) + target[index] * amount,
  )) as unknown as RGB
}

function rgbToHex(color: RGB): string {
  return `#${color.map((channel) => channel.toString(16).padStart(2, '0')).join('')}`
}
