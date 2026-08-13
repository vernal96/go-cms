// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import FileExplorer from './FileExplorer.vue'
import FolderMoveDialog from './FolderMoveDialog.vue'

function json(payload: unknown): Response {
  return new Response(JSON.stringify(payload), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('FileExplorer', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('loads disks and renders folders before files with one-line metadata', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(json({
        items: [{ code: 'public', visibility: 'public' }],
        permissions: { read: true, create: true, update: true, delete: true },
      }))
      .mockResolvedValueOnce(json({
        disk: { code: 'public', visibility: 'public' },
        folder: null,
        breadcrumbs: [],
        permissions: { read: true, create: true, update: true, delete: true },
        items: [
          { kind: 'file', id: 2, parent_id: null, storage: 'public', name: 'a.txt', mime_type: 'text/plain', size: 2048, created_at: '2026-01-01T10:00:00Z', updated_at: '2026-01-02T10:00:00Z' },
          { kind: 'folder', id: 1, parent_id: null, storage: 'public', name: 'Каталог', item_count: 3, created_at: '2026-01-01T10:00:00Z', updated_at: '2026-01-02T10:00:00Z' },
        ],
      }))
    vi.stubGlobal('fetch', fetchMock)

    const wrapper = mount(FileExplorer, {
      props: { accessToken: 'token', permissions: new Set(['core.file.read']) },
    })
    await flushPromises()

    const tiles = wrapper.findAll('.file-tile')
    expect(tiles).toHaveLength(2)
    expect(tiles[0]?.text()).toContain('Каталог')
    expect(tiles[0]?.text()).toContain('3 эл.')
    expect(tiles[1]?.text()).toContain('2 КБ')

    await tiles[1]!.trigger('click')
    expect(wrapper.find('.file-status-text').text()).toContain('создан:')
    expect(wrapper.find('.file-status-text').text()).toContain('text/plain')
  })

  it('moves the right-clicked selection through the folder dialog', async () => {
    const listing = {
      disk: { code: 'public', visibility: 'public' }, folder: null, breadcrumbs: [],
      permissions: { read: true, create: true, update: true, delete: true },
      items: [
        { kind: 'file', id: 2, parent_id: null, storage: 'public', name: 'a.txt', mime_type: 'text/plain', size: 10, created_at: '2026-01-01T10:00:00Z', updated_at: '2026-01-02T10:00:00Z' },
        { kind: 'file', id: 3, parent_id: null, storage: 'public', name: 'b.txt', mime_type: 'text/plain', size: 20, created_at: '2026-01-01T10:00:00Z', updated_at: '2026-01-02T10:00:00Z' },
      ],
    }
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(json({ items: [{ code: 'public', visibility: 'public' }], permissions: listing.permissions }))
      .mockResolvedValueOnce(json(listing))
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
      .mockResolvedValueOnce(json(listing))
    vi.stubGlobal('fetch', fetchMock)
    const wrapper = mount(FileExplorer, {
      props: { accessToken: 'token', permissions: new Set(['core.file.read', 'core.file.update']) },
    })
    await flushPromises()

    const tiles = wrapper.findAll('.file-tile')
    await tiles[0]!.trigger('click')
    await tiles[1]!.trigger('click', { ctrlKey: true })
    await tiles[0]!.trigger('contextmenu')
    expect(wrapper.find('.file-context-menu').text()).toContain('Переместить')
    const moveButton = wrapper.findAll('.file-context-menu button').find((button) => button.text().includes('Переместить'))!
    await moveButton.trigger('click')
    wrapper.findComponent(FolderMoveDialog).vm.$emit('confirm', 9)
    await flushPromises()

    const moveCall = fetchMock.mock.calls.find(([url]) => url === '/api/admin/filesystem/move')
    expect(moveCall?.[1]).toEqual(expect.objectContaining({ method: 'POST' }))
    expect(JSON.parse(String(moveCall?.[1]?.body))).toEqual({
      disk: 'public', folder_id: 9,
      items: [{ kind: 'file', id: 2 }, { kind: 'file', id: 3 }],
    })
  })
})
