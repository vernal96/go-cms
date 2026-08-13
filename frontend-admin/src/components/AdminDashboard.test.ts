// @vitest-environment jsdom

import { flushPromises, shallowMount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import { useSelectedSite } from '../composables/use-selected-site'
import { applyAppearance, disposeColorScheme } from '../theme'
import AdminDashboard from './AdminDashboard.vue'

async function mountDashboard(permissions: string[], path = '/admin/dashboard') {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/admin/dashboard', component: { template: '<div />' } },
      { path: '/admin/users', component: { template: '<div />' } },
      { path: '/admin/groups', component: { template: '<div />' } },
      { path: '/admin/profile', component: { template: '<div />' } },
      { path: '/admin/files', component: { template: '<div />' } },
    ],
  })
  await router.push(path)
  await router.isReady()
  const wrapper = shallowMount(AdminDashboard, {
    props: {
      user: {
        id: 1, login: 'admin', email: 'admin@example.test',
        display_name: 'Администратор', color_scheme: 'system',
        accent_color: 'blue',
        has_avatar: false, avatar_updated_at: null,
      },
      accessToken: 'token',
      permissions: new Set(permissions),
    },
    global: {
      plugins: [router],
      renderStubDefaultSlot: true,
      stubs: {
        ElPopover: { template: '<div class="popover-stub"><slot name="reference" /><slot /></div>' },
        ElTooltip: { template: '<span class="tooltip-stub"><slot /></span>' },
      },
    },
  })
  await flushPromises()
  return wrapper
}

describe('AdminDashboard', () => {
  beforeEach(() => {
    localStorage.clear()
    useSelectedSite().reset()
    document.documentElement.dataset.theme = 'light'
    document.documentElement.classList.remove('dark')
    vi.stubGlobal('matchMedia', vi.fn(() => ({
      matches: false,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })))
    applyAppearance('system', 'blue')
  })

  afterEach(() => {
    vi.restoreAllMocks()
    disposeColorScheme()
    vi.unstubAllGlobals()
    delete document.documentElement.dataset.theme
    document.documentElement.classList.remove('dark')
  })

  it('does not restore a stored site or mount resource navigation without site read access', async () => {
    localStorage.setItem('go-cms.admin.selected-site', JSON.stringify({ id: 7, domain: 'stored.test' }))
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = await mountDashboard(['admin.panel.read', 'core.resource.read'])

    expect(fetchMock).not.toHaveBeenCalled()
    expect(wrapper.findComponent({ name: 'SiteSelector' }).exists()).toBe(false)
    expect(wrapper.findComponent({ name: 'ResourceTree' }).exists()).toBe(false)
    expect(wrapper.find('.resource-sidebar').exists()).toBe(true)
    expect(wrapper.find('.global-search').attributes('readonly')).toBeDefined()
    expect(wrapper.text()).toContain('Нет доступа к сайтам')
    const home = wrapper.findAllComponents({ name: 'RouterLink' })
      .find((link) => link.props('to') === '/admin/dashboard')
    expect(home?.classes()).toContain('brand-mark')
    expect(home?.attributes('aria-label')).toBe('На главную')
    expect(wrapper.text()).not.toContain('Главная')
  })

  it('keeps the site selector and resource tree on the dashboard when permitted', async () => {
    const wrapper = await mountDashboard([
      'admin.panel.read',
      'core.site.read',
      'core.resource.read',
    ])

    expect(wrapper.findComponent({ name: 'SiteSelector' }).exists()).toBe(true)
    expect(wrapper.findComponent({ name: 'ResourceTree' }).exists()).toBe(true)
  })

  it('restores, collapses and expands the resizable sidebar without losing its width', async () => {
    localStorage.setItem('admin.resource-sidebar', JSON.stringify({ width: 480, collapsed: false }))
    const wrapper = await mountDashboard(['admin.panel.read'])
    const aside = wrapper.findComponent({ name: 'ElAside' })

    expect(aside.props('width')).toBe('480px')
    await wrapper.find('.sidebar-resize-handle').trigger('keydown', { key: 'Home' })
    expect(aside.props('width')).toBe('0px')

    await wrapper.find('.sidebar-expand-button').trigger('click')
    expect(aside.props('width')).toBe('480px')
    expect(JSON.parse(localStorage.getItem('admin.resource-sidebar') ?? '{}')).toEqual({
      width: 480,
      collapsed: false,
    })
  })

  it.each(['/admin/dashboard', '/admin/files', '/admin/profile', '/admin/users', '/admin/groups'])(
    'keeps the shared sidebar on %s',
    async (path) => {
      const wrapper = await mountDashboard(
        ['admin.panel.read', 'core.site.read', 'core.resource.read'],
        path,
      )

      expect(wrapper.find('.resource-sidebar').exists()).toBe(true)
      expect(wrapper.findComponent({ name: 'SiteSelector' }).exists()).toBe(true)
      expect(wrapper.findComponent({ name: 'ResourceTree' }).exists()).toBe(true)
    },
  )

  it('shows the filesystem menu only with file read permission', async () => {
    const allowed = await mountDashboard(['admin.panel.read', 'core.file.read'])
    expect(allowed.text()).toContain('Файловая система')
    const denied = await mountDashboard(['admin.panel.read', 'core.file.create'])
    expect(denied.text()).not.toContain('Файловая система')
  })

  it('persists the icon-only sidebar theme toggle', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      user: { color_scheme: 'dark', accent_color: 'blue' },
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    vi.stubGlobal('fetch', fetchMock)
    const wrapper = await mountDashboard(['admin.panel.read'])
    const toggle = wrapper.find('.sidebar-theme-toggle')

    expect(toggle.exists()).toBe(true)
    expect(toggle.attributes('aria-label')).toBe('Включить тёмную тему')
    expect(toggle.text()).toBe('')
    await toggle.trigger('click')
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledWith('/api/admin/profile/preferences', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ color_scheme: 'dark', accent_color: 'blue' }),
    }))
    expect(document.documentElement.dataset.theme).toBe('dark')
    expect(document.documentElement.classList.contains('dark')).toBe(true)
    expect(wrapper.emitted('profileUpdated')).toHaveLength(1)
  })

  it('restores the system theme when persistence fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      error: { code: 'request_failed', message: 'Ошибка сохранения' },
    }), { status: 500, headers: { 'Content-Type': 'application/json' } })))
    const wrapper = await mountDashboard(['admin.panel.read'])

    await wrapper.find('.sidebar-theme-toggle').trigger('click')
    await flushPromises()

    expect(document.documentElement.dataset.theme).toBe('light')
    expect(document.documentElement.classList.contains('dark')).toBe(false)
    expect(wrapper.emitted('profileUpdated')).toBeUndefined()
  })

  it('shows the accent picker before the theme toggle and persists a selected color', async () => {
    const fetchMock = vi.fn().mockResolvedValue(new Response(JSON.stringify({
      user: { color_scheme: 'system', accent_color: 'violet' },
    }), { status: 200, headers: { 'Content-Type': 'application/json' } }))
    vi.stubGlobal('fetch', fetchMock)
    const wrapper = await mountDashboard(['admin.panel.read'])
    const footerHTML = wrapper.find('.sidebar-theme-footer').html()

    expect(footerHTML.indexOf('sidebar-accent-toggle')).toBeLessThan(
      footerHTML.indexOf('sidebar-theme-toggle'),
    )
    expect(wrapper.find('.sidebar-accent-toggle').attributes('aria-label')).toBe('Выбрать акцентный цвет')
    const violet = wrapper.find('button[aria-label="Фиолетовый"]')
    expect(violet.exists()).toBe(true)
    await violet.trigger('click')
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledWith('/api/admin/profile/preferences', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ color_scheme: 'system', accent_color: 'violet' }),
    }))
    expect(document.documentElement.dataset.accent).toBe('violet')
    expect(document.documentElement.style.getPropertyValue('--el-color-primary')).toBe('#8B5CF6')
    expect(wrapper.find('button[aria-label="Фиолетовый"]').attributes('aria-selected')).toBe('true')
    expect(wrapper.emitted('profileUpdated')).toHaveLength(1)
  })

  it('restores the previous accent when persistence fails', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response(JSON.stringify({
      error: { code: 'request_failed', message: 'Ошибка сохранения' },
    }), { status: 500, headers: { 'Content-Type': 'application/json' } })))
    const wrapper = await mountDashboard(['admin.panel.read'])

    await wrapper.find('button[aria-label="Розовый"]').trigger('click')
    await flushPromises()

    expect(document.documentElement.dataset.accent).toBe('blue')
    expect(document.documentElement.style.getPropertyValue('--el-color-primary')).toBe('#409EFF')
    expect(wrapper.find('button[aria-label="Синий"]').attributes('aria-selected')).toBe('true')
    expect(wrapper.emitted('profileUpdated')).toBeUndefined()
  })
})
