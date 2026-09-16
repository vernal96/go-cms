import type { Component } from 'vue'
import type { RouteRecordRaw } from 'vue-router'

import type { AdminPlugin, AdminRouteDefinition } from './plugin'
import { routeShape } from './route-shape'

const semanticCodePattern = /^[a-z][a-z0-9]*(?:[._-][a-z0-9]+)*$/

export class AdminPluginRegistry {
  readonly #routes = new Map<string, AdminRouteDefinition>()
  readonly #configEditors = new Map<string, Component>()
  readonly #fieldEditors = new Map<string, Component>()
  readonly #icons = new Map<string, Component>()
  readonly #plugins: AdminPlugin[]

  constructor(plugins: readonly AdminPlugin[], reservedRoutes: readonly Pick<AdminRouteDefinition, 'name' | 'path'>[] = []) {
    const pluginCodes = new Set<string>()
    const routePaths = new Map<string, string>()
    for (const route of reservedRoutes) routePaths.set(routeShape(route.path), route.name)
    this.#plugins = [...plugins]

    for (const plugin of plugins) {
      if (!semanticCodePattern.test(plugin.code)) {
        throw new Error(`Invalid admin plugin code: ${plugin.code}`)
      }
      if (pluginCodes.has(plugin.code)) {
        throw new Error(`Admin plugin is registered more than once: ${plugin.code}`)
      }
      pluginCodes.add(plugin.code)

      for (const route of plugin.routes ?? []) {
        if (!semanticCodePattern.test(route.name) || !route.name.startsWith(`${plugin.code}.`)) {
          throw new Error(`Invalid route ${route.name} for admin plugin ${plugin.code}`)
        }
        if (!route.path.startsWith('/admin/')) {
          throw new Error(`Admin plugin route must use an /admin/ path: ${route.path}`)
        }
        if (this.#routes.has(route.name) || reservedRoutes.some(item => item.name === route.name)) {
          throw new Error(`Admin route is registered more than once: ${route.name}`)
        }
        const shape = routeShape(route.path)
        const owner = routePaths.get(shape)
        if (owner) {
          throw new Error(`Admin route ${route.name} (${route.path}) conflicts with ${owner}`)
        }
        this.#routes.set(route.name, route)
        routePaths.set(shape, route.name)
      }

      for (const [code, editor] of Object.entries(plugin.fieldEditors ?? {})) {
        if (!semanticCodePattern.test(code) || !code.startsWith(`${plugin.code}.`)) {
          throw new Error(`Invalid field editor ${code} for admin plugin ${plugin.code}`)
        }
        if (this.#fieldEditors.has(code)) throw new Error(`Admin field editor is registered more than once: ${code}`)
        this.#fieldEditors.set(code, editor)
      }

      for (const [code, editor] of Object.entries(plugin.configEditors ?? {})) {
        if (!semanticCodePattern.test(code) || !code.startsWith(`${plugin.code}.`)) throw new Error(`Invalid config editor ${code} for admin plugin ${plugin.code}`)
        if (this.#configEditors.has(code)) throw new Error(`Admin config editor is registered more than once: ${code}`)
        this.#configEditors.set(code, editor)
      }

      for (const [code, icon] of Object.entries(plugin.icons ?? {})) {
        if (!semanticCodePattern.test(code)) {
          throw new Error(`Invalid admin icon code: ${code}`)
        }
        if (this.#icons.has(code)) {
          throw new Error(`Admin icon is registered more than once: ${code}`)
        }
        this.#icons.set(code, icon)
      }
    }
  }

  plugins(): readonly AdminPlugin[] {
    return this.#plugins
  }

  routeRecords(): RouteRecordRaw[] {
    return [...this.#routes.values()].map((route) => ({
      name: route.name,
      path: route.path,
      component: route.component,
      props: route.props,
    })) as RouteRecordRaw[]
  }

  hasRoute(name: string): boolean {
    return this.#routes.has(name)
  }

  route(name: string): AdminRouteDefinition | undefined {
    return this.#routes.get(name)
  }

  fieldEditor(code: string): Component | undefined {
    return this.#fieldEditors.get(code)
  }

  configEditor(code: string): Component | undefined {
    return this.#configEditors.get(code)
  }

  icon(code: string): Component | undefined {
    return this.#icons.get(code)
  }
}
