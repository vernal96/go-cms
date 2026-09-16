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

  it('owns module field editors and rejects another module namespace', () => {
    const registry = new AdminPluginRegistry([{ code: 'forms', fieldEditors: { 'forms.form-picker': view } }])
    expect(registry.fieldEditor('forms.form-picker')).toBe(view)
    expect(registry.fieldEditor('missing.editor')).toBeUndefined()
    expect(() => new AdminPluginRegistry([{ code: 'forms', fieldEditors: { 'mail.editor': view } }])).toThrow(/invalid field editor/i)
  })

  it('registers every Mail navigation target and icon', () => {
    expect(adminPluginRegistry.icon('mail')).toBeDefined()
    expect(adminPluginRegistry.icon('forms')).toBeDefined()
    expect([
      'mail.templates',
      'mail.templates.create',
      'mail.templates.edit',
      'mail.send',
      'mail.history',
      'mail.history.detail',
      'forms.list',
      'forms.edit',
      'forms.results',
      'forms.results.detail',
    ].every((name) => adminPluginRegistry.hasRoute(name))).toBe(true)
  })
})

it('rejects equivalent parameter routes and shell routes', () => {
  for (const paths of [
    ['/admin/items/:id', '/admin/items/:slug'],
    ['/admin/items/:id(\\d+)', '/admin/items/:slug(\\d+)'],
    ['/admin/Items/:id/', '/admin/items/:slug'],
  ]) {
    expect(() => new AdminPluginRegistry([
      {code:'one',routes:[{name:'one.item',path:paths[0]!,component:view}]},
      {code:'two',routes:[{name:'two.item',path:paths[1]!,component:view}]},
    ])).toThrow(/one.item/)
  }
  expect(() => new AdminPluginRegistry([
    {code:'example',routes:[{name:'example.dashboard',path:'/admin/dashboard',component:view}]},
  ], [{name:'shell.dashboard',path:'/admin/dashboard'}])).toThrow(/shell.dashboard/)
})

it('keeps static routes, different regexp constraints and modifiers distinct', () => {
  const paths = ['/admin/items/new', '/admin/items/:id(\\d+)', '/admin/items/:slug([a-z]+)', '/admin/items/:id?', '/admin/items/:id+']
  const registry = new AdminPluginRegistry([{code:'example',routes:paths.map((path,index)=>({name:`example.route${index}`,path,component:view}))}])
  expect(registry.routeRecords()).toHaveLength(paths.length)
})
