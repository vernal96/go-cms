import { describe, expect, it } from 'vitest'
import { canPlace, dropNode, moveNode, siblings, treeNodes } from './layout-tree'
import type { LayoutNode } from './types'
const nodes: LayoutNode[] = [
  { id: 1, form_id: 1, kind: 'container', container_type: 'group', position: 0 },
  { id: 2, form_id: 1, kind: 'field', field_id: 2, parent_id: 1, position: 0 },
  { id: 3, form_id: 1, kind: 'container', container_type: 'slide', parent_id: 1, position: 1 },
  { id: 4, form_id: 1, kind: 'element', element_id: 1, parent_id: 3, position: 0 },
  { id: 5, form_id: 1, kind: 'field', field_id: 3, position: 1 },
]
describe('Forms layout moves', () => {
  it('moves a complete branch and preserves its descendants and source', () => {
    const result = dropNode(nodes, 3, 5, 'after')
    expect(siblings(result, null).map(node => node.id)).toEqual([1, 5, 3])
    expect(treeNodes(result)[2]?.children.map(node => node.id)).toEqual([4])
    expect(nodes[2]?.parent_id).toBe(1)
  })
  it('reorders forward and backward without duplicate positions', () => {
    const result = moveNode(nodes, 2, 1, 1)
    expect(siblings(result, 1).map(node => [node.id, node.position])).toEqual([[3, 0], [2, 1]])
    expect(siblings(moveNode(result, 2, 1, 0), 1).map(node => node.id)).toEqual([2, 3])
  })
  it('appends to a container and accepts an empty root', () => {
    expect(siblings(dropNode(nodes, 5, 3, 'inner'), 3).map(node => node.id)).toEqual([4, 5])
    expect(treeNodes([])).toEqual([])
  })
  it('rejects cycles, self-parent, leaves and missing parents', () => {
    for (const parent of [1, 2, 3, 4, 99]) expect(canPlace(nodes, 1, parent)).toBe(false)
    expect(() => dropNode(nodes, 1, 4, 'after')).toThrow()
    expect(() => dropNode(nodes, 2, 5, 'inner')).toThrow()
    expect(() => moveNode(nodes, 2, null, 99)).toThrow()
  })
})
