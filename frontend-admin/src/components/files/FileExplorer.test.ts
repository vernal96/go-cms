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

function dragTransfer(initialTypes: string[] = [], files: File[] = []): DataTransfer {
  const data = new Map<string, string>()
  const types = [...initialTypes]
  return {
    dropEffect: 'none',
    effectAllowed: 'uninitialized',
    files,
    items: [],
    types,
    getData: (type: string) => data.get(type) ?? '',
    setData: (type: string, value: string) => {
      data.set(type, value)
      if (!types.includes(type)) types.push(type)
    },
  } as unknown as DataTransfer
}

const listing = {
  disk: { code: 'public', visibility: 'public' },
  folder: null,
  breadcrumbs: [],
  permissions: { read: true, create: true, update: true, delete: true },
  items: [
    { kind: 'folder', id: 1, parent_id: null, storage: 'public', name: 'Каталог', item_count: 0, created_at: '2026-01-01T10:00:00Z', updated_at: '2026-01-02T10:00:00Z' },
    { kind: 'file', id: 2, parent_id: null, storage: 'public', name: 'a.txt', mime_type: 'text/plain', size: 10, created_at: '2026-01-01T10:00:00Z', updated_at: '2026-01-02T10:00:00Z' },
    { kind: 'file', id: 3, parent_id: null, storage: 'public', name: 'b.txt', mime_type: 'text/plain', size: 20, created_at: '2026-01-01T10:00:00Z', updated_at: '2026-01-02T10:00:00Z' },
  ],
} as const

function mountExplorer(fetchMock: ReturnType<typeof vi.fn>) {
  vi.stubGlobal('fetch', fetchMock)
  return mount(FileExplorer, {
    props: { accessToken: 'token', permissions: new Set(['core.file.read', 'core.file.update']) },
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

    const moveCall = fetchMock.mock.calls.find(([url]) => url === '/api/files/move')
    expect(moveCall?.[1]).toEqual(expect.objectContaining({ method: 'POST' }))
    expect(JSON.parse(String(moveCall?.[1]?.body))).toEqual({
      disk: 'public', folder_id: 9,
      items: [{ kind: 'file', id: 2 }, { kind: 'file', id: 3 }],
    })
  })

  it('drags the selected group by a handle and highlights only the destination folder', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(json({ items: [{ code: 'public', visibility: 'public' }], permissions: listing.permissions }))
      .mockResolvedValueOnce(json(listing))
      .mockResolvedValueOnce(new Response(null, { status: 204 }))
      .mockResolvedValueOnce(json(listing))
    const wrapper = mountExplorer(fetchMock)
    await flushPromises()

    const tiles = wrapper.findAll('.file-tile')
    await tiles[1]!.trigger('click')
    await tiles[2]!.trigger('click', { ctrlKey: true })
    const transfer = dragTransfer()
    await wrapper.findAll('.file-drag-handle')[1]!.trigger('dragstart', { dataTransfer: transfer })

    expect(wrapper.findAll('.file-tile.is-drag-source')).toHaveLength(2)
    expect(wrapper.find('.file-tile').attributes('draggable')).toBeUndefined()
    expect(wrapper.find('.file-drag-handle').attributes('draggable')).toBe('true')

    await tiles[0]!.trigger('dragenter', { dataTransfer: transfer })
    expect(tiles[0]!.classes()).toContain('is-drop-target')
    expect(wrapper.find('.file-grid').classes()).not.toContain('is-drop-target')

    await tiles[0]!.trigger('drop', { dataTransfer: transfer })
    await flushPromises()
    const moveCall = fetchMock.mock.calls.find(([url]) => url === '/api/files/move')
    expect(JSON.parse(String(moveCall?.[1]?.body))).toEqual({
      disk: 'public', folder_id: 1,
      items: [{ kind: 'file', id: 2 }, { kind: 'file', id: 3 }],
    })
    expect(wrapper.findAll('.file-tile.is-drag-source')).toHaveLength(0)
    expect(wrapper.findAll('.file-tile.is-drop-target')).toHaveLength(0)
  })

  it('rejects a folder as its own destination and clears drag state on dragend', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(json({ items: [{ code: 'public', visibility: 'public' }], permissions: listing.permissions }))
      .mockResolvedValueOnce(json(listing))
    const wrapper = mountExplorer(fetchMock)
    await flushPromises()

    const transfer = dragTransfer()
    const folder = wrapper.findAll('.file-tile')[0]!
    const handle = wrapper.findAll('.file-drag-handle')[0]!
    await handle.trigger('dragstart', { dataTransfer: transfer })
    await folder.trigger('dragenter', { dataTransfer: transfer })
    expect(folder.classes()).not.toContain('is-drop-target')

    await handle.trigger('dragend', { dataTransfer: transfer })
    expect(folder.classes()).not.toContain('is-drag-source')
    expect(fetchMock).toHaveBeenCalledTimes(2)
  })

  it('distinguishes an external upload and highlights the current directory', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(json({ items: [{ code: 'public', visibility: 'public' }], permissions: listing.permissions }))
      .mockResolvedValueOnce(json(listing))
      .mockResolvedValueOnce(json({ kind: 'file', id: 4, name: 'upload.txt' }))
      .mockResolvedValueOnce(json(listing))
    const wrapper = mountExplorer(fetchMock)
    await flushPromises()

    const transfer = dragTransfer(['Files'], [new File(['content'], 'upload.txt', { type: 'text/plain' })])
    const grid = wrapper.find('.file-grid')
    await grid.trigger('dragenter', { dataTransfer: transfer })
    expect(grid.classes()).toContain('is-drop-target')
    expect(transfer.dropEffect).toBe('copy')

    await grid.trigger('dragleave', { dataTransfer: transfer, relatedTarget: null })
    expect(wrapper.find('.file-grid').classes()).not.toContain('has-active-drag')
    await wrapper.find('.file-grid').trigger('dragenter', { dataTransfer: transfer })

    await wrapper.find('.file-grid').trigger('drop', { dataTransfer: transfer })
    await flushPromises()
    expect(fetchMock.mock.calls.some(([url]) => url === '/api/files/uploads')).toBe(true)
    expect(wrapper.find('.file-grid').classes()).not.toContain('is-drop-target')
  })

  it('does not expose drag handles without update permission', async () => {
    const denied = {
      ...listing,
      permissions: { ...listing.permissions, update: false },
    }
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(json({ items: [{ code: 'public', visibility: 'public' }], permissions: denied.permissions }))
      .mockResolvedValueOnce(json(denied))
    const wrapper = mountExplorer(fetchMock)
    await flushPromises()

    expect(wrapper.find('.file-drag-handle').exists()).toBe(false)
  })
})
