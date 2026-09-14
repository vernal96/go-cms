import type { LayoutNode } from './types'

export interface LayoutTreeNode extends LayoutNode { children: LayoutTreeNode[] }
export type DropPosition = 'before' | 'after' | 'inner'
export function siblings(nodes: LayoutNode[], parent: number | null): LayoutNode[] {
  return nodes.filter(node => (node.parent_id ?? null) === parent).sort((a, b) => a.position - b.position || a.id - b.id)
}
export function treeNodes(nodes: LayoutNode[], parent: number | null = null): LayoutTreeNode[] {
  return siblings(nodes, parent).map(node => ({ ...node, children: treeNodes(nodes, node.id) }))
}
export function canPlace(nodes: LayoutNode[], id: number, parent: number | null): boolean {
  const visited = new Set([id])
  while (parent !== null) {
    if (visited.has(parent)) return false
    visited.add(parent)
    const node = nodes.find(node => node.id === parent)
    if (!node || node.kind !== 'container') return false
    parent = node.parent_id ?? null
  }
  return true
}
export function moveNode(nodes: LayoutNode[], id: number, parent: number | null, position: number): LayoutNode[] {
  if (!canPlace(nodes, id, parent)) throw new Error('Недопустимый родитель узла')
  const draft = nodes.map(node => ({ ...node }))
  const moving = draft.find(node => node.id === id)
  if (!moving) throw new Error('Узел не найден')
  const source = siblings(draft, moving.parent_id ?? null).filter(node => node.id !== id)
  source.forEach((node, index) => { node.position = index })
  const target = siblings(draft, parent).filter(node => node.id !== id)
  if (position < 0 || position > target.length) throw new Error('Недопустимая позиция узла')
  moving.parent_id = parent
  target.splice(position, 0, moving)
  target.forEach((node, index) => { node.position = index })
  return draft
}
export function dropNode(nodes: LayoutNode[], id: number, targetID: number, drop: DropPosition): LayoutNode[] {
  const target = nodes.find(node => node.id === targetID)
  if (!target || id === targetID) throw new Error('Недопустимая цель переноса')
  const parent = drop === 'inner' ? target.id : target.parent_id ?? null
  const children = siblings(nodes, parent).filter(node => node.id !== id)
  const index = drop === 'inner' ? children.length : children.findIndex(node => node.id === targetID) + (drop === 'after' ? 1 : 0)
  return moveNode(nodes, id, parent, index)
}
