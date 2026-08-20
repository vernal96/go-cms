// @vitest-environment jsdom

import { mount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { describe, expect, it } from 'vitest'

import { adminPluginRegistry } from '../admin-plugins'
import type { ResolvedAdminNavigationItem } from '../composables/use-admin-navigation'
import AdminNavigation from './AdminNavigation.vue'

const items: ResolvedAdminNavigationItem[] = [
  {
    code: 'arbitrary', label: 'Название из API', route: 'core.sites', icon: 'sites',
    order: 100, scope: 'global', children: [],
  },
  {
    code: 'identity', label: 'Управление доступом', icon: 'users',
    order: 200, scope: 'global', children: [
      {
        code: 'users', label: 'Аккаунты', route: 'core.users',
        order: 100, scope: 'global', children: [],
      },
      {
        code: 'nested', label: 'Вложенная группа',
        order: 200, scope: 'global', children: [
          {
            code: 'groups', label: 'Роли', route: 'core.groups',
            order: 100, scope: 'global', children: [],
          },
        ],
      },
    ],
  },
]

const stubs = {
  ElDropdown: { template: '<div class="dropdown-stub"><slot /><slot name="dropdown" /></div>' },
  ElDropdownMenu: { template: '<div class="dropdown-menu-stub"><slot /></div>' },
  ElDropdownItem: {
    props: ['command'],
    emits: ['click'],
    template: '<button class="dropdown-item-stub" @click="$emit(\'click\')"><slot /></button>',
  },
  ElButton: { template: '<button><slot /></button>' },
  ElIcon: { template: '<span class="icon-stub"><slot /></span>' },
}

async function mountNavigation(path: string) {
  const router = createRouter({
    history: createMemoryHistory(),
    routes: adminPluginRegistry.routeRecords(),
  })
  await router.push(path)
  await router.isReady()
  const wrapper = mount(AdminNavigation, {
    props: { items },
    global: { plugins: [router], stubs },
  })
  return { wrapper, router }
}

describe('AdminNavigation', () => {
  it('renders arbitrary backend labels and recursively nested items', async () => {
    const { wrapper } = await mountNavigation('/admin/sites')

    expect(wrapper.text()).toContain('Название из API')
    expect(wrapper.text()).toContain('Управление доступом')
    expect(wrapper.text()).toContain('Аккаунты')
    expect(wrapper.text()).toContain('Вложенная группа')
    expect(wrapper.text()).toContain('Роли')
    expect(wrapper.find('[data-navigation-code="arbitrary"]').exists()).toBe(true)
    expect(wrapper.find('[data-navigation-code="groups"]').exists()).toBe(true)
  })

  it('resolves semantic routes and preserves active state for descendant screens', async () => {
    const direct = await mountNavigation('/admin/sites')
    const link = direct.wrapper.find('[data-navigation-code="arbitrary"]')
    expect(link.attributes('href')).toBe('/admin/sites')
    await link.trigger('click')
    expect(direct.router.currentRoute.value.name).toBe('core.sites')

    const nested = await mountNavigation('/admin/users/42/edit')
    expect(nested.wrapper.find('.admin-navigation-trigger').classes()).toContain('is-active')
    expect(nested.wrapper.find('[data-navigation-code="users"]').classes()).toContain('is-active')
  })
})
