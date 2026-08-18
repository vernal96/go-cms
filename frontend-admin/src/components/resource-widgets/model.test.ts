import { describe, expect, it } from 'vitest'
import type { ResourceWidget } from '../../types/admin'
import { moveWidget, widgetOrder } from './model'

const widget = (id: number, area: 'body' | 'sidebar', position: number): ResourceWidget => ({
  id, code: 'core_content', area, position, view: 'default', columns: 12,
  margin_top: 0, margin_bottom: 0, enabled: true, params: {},
})

describe('resource widget ordering', () => {
  it('reorders inside one area without changing stable ids', () => {
    const moved = moveWidget([
      widget(41, 'body', 0), widget(42, 'body', 1), widget(43, 'body', 2),
    ], 41, 'body', 3)
    expect(widgetOrder(moved)).toEqual([
      { id: 42, area: 'body', position: 0 },
      { id: 43, area: 'body', position: 1 },
      { id: 41, area: 'body', position: 2 },
    ])
  })

  it('keeps stable ids while reordering and moving across areas', () => {
    const moved = moveWidget([
      widget(41, 'body', 0), widget(42, 'body', 1), widget(77, 'sidebar', 0),
    ], 42, 'sidebar', 0)
    expect(widgetOrder(moved)).toEqual([
      { id: 41, area: 'body', position: 0 },
      { id: 42, area: 'sidebar', position: 0 },
      { id: 77, area: 'sidebar', position: 1 },
    ])
  })
})
