import { describe, expect, it } from 'vitest'

import { AdminPluginRegistry } from './registry'
import { adminPluginRegistry } from '../admin-plugins'

const view = { template: '<div />' }

describe('AdminPluginRegistry', () => {
  it('composes semantic routes and icons from installed plugins', () => {
    const icon = { template: '<svg />' }
    const registry = new AdminPluginRegistry([
      {
        code: 'forms',
        routes: [{ name: 'forms.list', path: '/admin/forms', component: view }],
        icons: { forms: icon },
      },
    ])

    expect(registry.hasRoute('forms.list')).toBe(true)
    expect(registry.route('forms.list')?.path).toBe('/admin/forms')
    expect(registry.icon('forms')).toBe(icon)
    expect(registry.routeRecords()).toEqual([
      expect.objectContaining({ name: 'forms.list', path: '/admin/forms' }),
    ])
  })

  it('rejects duplicate plugin, route and icon registrations', () => {
    expect(() => new AdminPluginRegistry([
      { code: 'forms' },
      { code: 'forms' },
    ])).toThrow(/plugin is registered more than once/i)

    expect(() => new AdminPluginRegistry([
      { code: 'forms', icons: { feature: view } },
      { code: 'seo', icons: { feature: view } },
    ])).toThrow(/icon is registered more than once/i)
  })

  it('registers every Mail navigation target and icon', () => {
    expect(adminPluginRegistry.icon('mail')).toBeDefined()
    expect([
      'mail.templates',
      'mail.templates.create',
      'mail.templates.edit',
      'mail.send',
      'mail.history',
      'mail.history.detail',
    ].every((name) => adminPluginRegistry.hasRoute(name))).toBe(true)
  })
})
