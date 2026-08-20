import { computed, ref } from 'vue'

import { adminPluginRegistry } from '../admin-plugins'
import type { AdminPluginRegistry } from '../admin-plugins/registry'
import { adminRequest } from '../api/admin-api'
import type { AdminNavigationItem, AdminNavigationResponse } from '../types/admin'

export interface ResolvedAdminNavigationItem extends AdminNavigationItem {
  route?: string
  children: ResolvedAdminNavigationItem[]
}

export function resolveAdminNavigation(
  items: readonly AdminNavigationItem[],
  registry: AdminPluginRegistry = adminPluginRegistry,
  warn: (message: string) => void = console.warn,
): ResolvedAdminNavigationItem[] {
  const result: ResolvedAdminNavigationItem[] = []
  for (const item of items) {
    const children = resolveAdminNavigation(item.children ?? [], registry, warn)
    if ((item.children?.length ?? 0) > 0) {
      if (children.length === 0) continue
      result.push({ ...item, route: undefined, children })
      continue
    }
    if (!item.route || !registry.hasRoute(item.route)) {
      warn(
        `[GO CMS admin] Navigation item "${item.code}" references unavailable semantic route "${item.route ?? ''}". ` +
        'Install/register the matching frontend admin plugin and rebuild the application.',
      )
      continue
    }
    result.push({ ...item, children: [] })
  }
  return result
}

export function useAdminNavigation(
  registry: AdminPluginRegistry = adminPluginRegistry,
) {
  const items = ref<ResolvedAdminNavigationItem[]>([])
  const loading = ref(false)
  let controller: AbortController | null = null
  let requestSequence = 0
  let currentSiteID: number | null = null
  let currentToken: string | null = null

  async function refresh(accessToken: string, siteID: number | null): Promise<void> {
    if (accessToken !== currentToken) {
      currentToken = accessToken
      items.value = []
    } else if (siteID !== currentSiteID) {
      items.value = items.value.filter((item) => item.scope === 'global')
    }
    currentSiteID = siteID

    controller?.abort()
    controller = new AbortController()
    const sequence = ++requestSequence
    loading.value = true
    const path = siteID === null
      ? '/api/admin/navigation'
      : `/api/admin/navigation?site_id=${encodeURIComponent(String(siteID))}`

    try {
      const response = await adminRequest<AdminNavigationResponse>(
        path,
        accessToken,
        { signal: controller.signal },
      )
      if (sequence !== requestSequence) return
      items.value = resolveAdminNavigation(
        Array.isArray(response.items) ? response.items : [],
        registry,
      )
    } catch (error) {
      if (error instanceof DOMException && error.name === 'AbortError') return
      throw error
    } finally {
      if (sequence === requestSequence) loading.value = false
    }
  }

  function dispose(): void {
    controller?.abort()
    controller = null
    requestSequence += 1
    loading.value = false
  }

  return {
    items: computed(() => items.value),
    loading: computed(() => loading.value),
    refresh,
    dispose,
  }
}
