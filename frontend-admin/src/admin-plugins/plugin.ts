import type { Component } from 'vue'

export interface AdminRouteDefinition {
  name: string
  path: string
  component: Component
  props?: boolean
}

export interface AdminPlugin {
  code: string
  routes?: AdminRouteDefinition[]
  fieldEditors?: Record<string, Component>
  icons?: Record<string, Component>
}
