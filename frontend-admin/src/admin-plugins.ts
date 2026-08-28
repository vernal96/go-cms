import { coreAdminPlugin } from './admin-plugins/core'
import { formsAdminPlugin } from './admin-plugins/forms'
import { mailAdminPlugin } from './admin-plugins/mail'
import { AdminPluginRegistry } from './admin-plugins/registry'

// Installed frontend admin packages have one declarative composition point.
export const adminPlugins = [coreAdminPlugin, mailAdminPlugin, formsAdminPlugin] as const
export const adminPluginRegistry = new AdminPluginRegistry(adminPlugins)
