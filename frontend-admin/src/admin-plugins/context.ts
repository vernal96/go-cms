import type { InjectionKey } from 'vue'
import type { AdminPluginRegistry } from './registry'

export const adminPluginRegistryKey: InjectionKey<AdminPluginRegistry> = Symbol('adminPluginRegistry')
