// @vitest-environment jsdom

import { shallowMount } from '@vue/test-utils'
import { ElTree } from 'element-plus'
import { afterEach, describe, expect, it, vi } from 'vitest'

import FolderMoveDialog from './FolderMoveDialog.vue'

function json(payload: unknown): Response {
  return new Response(JSON.stringify(payload), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('FolderMoveDialog', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('loads only folders lazily and disables a source folder and its descendants', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(json({
        items: [
          { kind: 'folder', id: 2, parent_id: null, storage: 'public', name: 'Источник' },
          { kind: 'folder', id: 3, parent_id: null, storage: 'public', name: 'Назначение' },
          { kind: 'file', id: 8, parent_id: null, storage: 'public', name: 'file.txt' },
        ],
      }))
      .mockResolvedValueOnce(json({
        items: [
          { kind: 'folder', id: 4, parent_id: 2, storage: 'public', name: 'Потомок' },
        ],
      }))
    vi.stubGlobal('fetch', fetchMock)
    const selected = [{
      kind: 'folder' as const,
      id: 2,
      parent_id: null,
      storage: 'public',
      name: 'Источник',
      item_count: 1,
      created_at: '2026-08-13T10:00:00Z',
      updated_at: '2026-08-13T10:00:00Z',
    }]
    const wrapper = shallowMount(FolderMoveDialog, {
      props: {
        modelValue: true,
        'onUpdate:modelValue': () => {},
        accessToken: 'token',
        disk: 'public',
        items: selected,
      },
      global: { renderStubDefaultSlot: true },
    })
    const load = wrapper.findComponent(ElTree).props('load') as (
      node: { level: number; data: unknown },
      resolve: (items: Array<Record<string, unknown>>) => void,
    ) => Promise<void>
    let sentinelResult: Array<Record<string, unknown>> = []
    await load(
      { level: 0, data: {} },
      (items) => { sentinelResult = items },
    )
    expect(sentinelResult).toEqual([expect.objectContaining({ id: null, name: 'Корень' })])
    expect(fetchMock).not.toHaveBeenCalled()

    let roots: Array<Record<string, unknown>> = []
    await load(
      { level: 1, data: { id: null, blocked: false } },
      (items) => { roots = items },
    )

    expect(fetchMock).toHaveBeenNthCalledWith(
      1,
      '/api/files/items?disk=public',
      expect.anything(),
    )
    expect(roots.map(({ id }) => id)).toEqual([2, 3])
    expect(roots[0]?.disabled).toBe(true)
    expect(roots[1]?.disabled).toBe(false)

    let children: Array<Record<string, unknown>> = []
    await load(
      { level: 2, data: roots[0] },
      (items) => { children = items },
    )
    expect(fetchMock).toHaveBeenNthCalledWith(
      2,
      '/api/files/items?disk=public&folder_id=2',
      expect.anything(),
    )
    expect(children).toEqual([expect.objectContaining({ id: 4, disabled: true, blocked: true })])
  })
})
