import { coreAdminPlugin } from './admin-plugins/core'
import { AdminPluginRegistry } from './admin-plugins/registry'

// Installed frontend admin packages have one declarative composition point.
export const adminPlugins = [coreAdminPlugin] as const
export const adminPluginRegistry = new AdminPluginRegistry(adminPlugins)
