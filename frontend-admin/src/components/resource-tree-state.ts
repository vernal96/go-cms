import type { ResourceTreeItem } from '../types/admin'

export function resourceTreeNodeClasses(item: Pick<ResourceTreeItem, 'in_menu' | 'deleted' | 'published'>): Record<string, boolean> {
  return {
    'is-hidden-from-menu': !item.in_menu,
    'is-deleted': item.deleted,
    'is-unpublished': !item.published && !item.deleted,
  }
}
