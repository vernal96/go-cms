import type { InjectionKey, Ref } from 'vue'

export const adminAccessTokenKey: InjectionKey<Ref<string>> = Symbol('admin-access-token')
export const adminPermissionsKey: InjectionKey<Ref<ReadonlySet<string>>> = Symbol('admin-permissions')
