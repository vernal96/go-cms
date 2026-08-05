import { readonly, ref } from 'vue'

import { AdminAPIError, adminRequest } from '../api/admin-api'
import type { SiteDetailsResponse, SiteOption } from '../types/admin'

const storageKey = 'go-cms.admin.selected-site'
const selectedSite = ref<SiteOption | null>(null)
const selectorRevision = ref(0)
let initializedToken: string | null = null

export function useSelectedSite() {
  async function initialize(accessToken: string): Promise<void> {
    if (initializedToken === accessToken) return
    initializedToken = accessToken
    const stored = readStored()
    if (stored === null) {
      selectedSite.value = null
      return
    }
    try {
      const response = await adminRequest<SiteDetailsResponse>(
        `/api/admin/sites/${stored.id}`,
        accessToken,
      )
      setSelected({ id: response.site.id, domain: response.site.domain })
    } catch (error) {
      if (error instanceof AdminAPIError && [403, 404].includes(error.status)) {
        clearSelected()
        return
      }
      throw error
    }
  }

  function setSelected(site: SiteOption | null): void {
    selectedSite.value = site
    if (typeof localStorage === 'undefined') return
    if (site === null) {
      localStorage.removeItem(storageKey)
    } else {
      localStorage.setItem(storageKey, JSON.stringify(site))
    }
  }

  function clearSelected(): void {
    setSelected(null)
  }

  function refreshSelector(): void {
    selectorRevision.value += 1
  }

  function reset(): void {
    initializedToken = null
    selectedSite.value = null
  }

  return {
    selectedSite: readonly(selectedSite),
    selectorRevision: readonly(selectorRevision),
    initialize,
    setSelected,
    clearSelected,
    refreshSelector,
    reset,
  }
}

function readStored(): SiteOption | null {
  if (typeof localStorage === 'undefined') return null
  try {
    const value = JSON.parse(localStorage.getItem(storageKey) ?? 'null') as Partial<SiteOption> | null
    return value && Number.isInteger(value.id) && Number(value.id) > 0 && typeof value.domain === 'string'
      ? { id: Number(value.id), domain: value.domain }
      : null
  } catch {
    localStorage.removeItem(storageKey)
    return null
  }
}
