// @vitest-environment jsdom

import { afterEach, describe, expect, it, vi } from 'vitest'

import { applyAccentColor, applyAppearance, applyColorScheme, disposeColorScheme } from './theme'

describe('admin color scheme', () => {
  afterEach(() => {
    disposeColorScheme()
    delete document.documentElement.dataset.theme
    delete document.documentElement.dataset.accent
    document.documentElement.classList.remove('dark')
    document.documentElement.style.removeProperty('color-scheme')
    for (const property of [
      '--admin-primary',
      '--el-color-primary',
      '--el-color-primary-rgb',
      '--el-color-primary-light-3',
      '--el-color-primary-light-5',
      '--el-color-primary-light-7',
      '--el-color-primary-light-8',
      '--el-color-primary-light-9',
      '--el-color-primary-dark-2',
    ]) document.documentElement.style.removeProperty(property)
    vi.unstubAllGlobals()
  })

  it('applies explicit and system themes and follows system changes', () => {
    let trigger = () => {}
    let dark = true
    vi.stubGlobal('matchMedia', vi.fn(() => ({
      get matches() { return dark },
      addEventListener: vi.fn((_event: string, callback: (event: Event) => void) => {
        trigger = () => callback(new Event('change'))
      }),
      removeEventListener: vi.fn(),
    })))

    applyAppearance('light', 'violet')
    expect(document.documentElement.dataset.theme).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
    expect(document.documentElement.dataset.accent).toBe('violet')
    expect(document.documentElement.style.getPropertyValue('--el-color-primary')).toBe('#8B5CF6')
    expect(document.documentElement.style.getPropertyValue('--el-color-primary-light-3')).toBe('#ae8df9')
    applyColorScheme('system')
    expect(document.documentElement.dataset.theme).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(document.documentElement.style.getPropertyValue('--el-color-primary-light-3')).toBe('#6746b2')
    dark = false
    trigger()
    expect(document.documentElement.dataset.theme).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
    expect(document.documentElement.style.getPropertyValue('--el-color-primary-light-3')).toBe('#ae8df9')
  })

  it('updates the complete Element Plus primary palette', () => {
    vi.stubGlobal('matchMedia', vi.fn(() => ({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })))

    applyAppearance('light', 'blue')
    applyAccentColor('amber')

    expect(document.documentElement.dataset.accent).toBe('amber')
    expect(document.documentElement.style.getPropertyValue('--admin-primary')).toBe('#F59E0B')
    expect(document.documentElement.style.getPropertyValue('--el-color-primary-rgb')).toBe('245, 158, 11')
    for (const property of [
      '--el-color-primary-light-3',
      '--el-color-primary-light-5',
      '--el-color-primary-light-7',
      '--el-color-primary-light-8',
      '--el-color-primary-light-9',
      '--el-color-primary-dark-2',
    ]) expect(document.documentElement.style.getPropertyValue(property)).not.toBe('')
  })
})
