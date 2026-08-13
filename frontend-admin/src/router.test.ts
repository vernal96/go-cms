// @vitest-environment jsdom

import { describe, expect, it } from 'vitest'

import { router } from './router'

describe('admin router', () => {
  it('uses the dashboard as the main and fallback route', () => {
    const routes = router.getRoutes()
    expect(routes.find((route) => route.path === '/')?.redirect).toBe('/admin/dashboard')
    expect(routes.find((route) => route.path === '/admin')?.redirect).toBe('/admin/dashboard')
    expect(routes.find((route) => route.path === '/admin/dashboard')?.components?.default).toBeTruthy()
    expect(routes.find((route) => route.path === '/:pathMatch(.*)*')?.redirect).toBe('/admin/dashboard')
  })
})
