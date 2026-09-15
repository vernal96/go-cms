// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ElMessageBox } from 'element-plus'
import FileExplorer from './FileExplorer.vue'
import FolderMoveDialog from './FolderMoveDialog.vue'

function json(payload: unknown): Response {
  return new Response(JSON.stringify(payload), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
}

function jsonError(status: number): Response {
  return new Response(JSON.stringify({ error: { code: 'not_found', message: 'not found' } }), {
    status,
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
    { kind: 'folder', source_file_id: null, id: 1, folder_id: null, storage: 'public', name: 'Каталог', item_count: 0, created_at: '2026-01-01T10:00:00Z', updated_at: '2026-01-02T10:00:00Z' },
    { kind: 'file', source_file_id: null, id: 2, folder_id: null, storage: 'public', name: 'a.txt', mime_type: 'text/plain', size: 10, created_at: '2026-01-01T10:00:00Z', updated_at: '2026-01-02T10:00:00Z' },
    { kind: 'file', source_file_id: null, id: 3, folder_id: null, storage: 'public', name: 'b.txt', mime_type: 'text/plain', size: 20, created_at: '2026-01-01T10:00:00Z', updated_at: '2026-01-02T10:00:00Z' },
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
          { kind: 'file', source_file_id: null, id: 2, folder_id: null, storage: 'public', name: 'a.txt', mime_type: 'text/plain', size: 2048, created_at: '2026-01-01T10:00:00Z', updated_at: '2026-01-02T10:00:00Z' },
          { kind: 'folder', source_file_id: null, id: 1, folder_id: null, storage: 'public', name: 'Каталог', item_count: 3, created_at: '2026-01-01T10:00:00Z', updated_at: '2026-01-02T10:00:00Z' },
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

  it('shows arbitrary disk labels and switches requests by code', async () => {
    const disks = [
      { code: 'media', label: 'Медиатека', visibility: 'public' },
      { code: 'archive', label: 'Архив', visibility: 'private' },
    ]
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(json({ items: disks, permissions: listing.permissions }))
      .mockResolvedValueOnce(json({ ...listing, disk: disks[0], items: [] }))
      .mockResolvedValueOnce(json({ ...listing, disk: disks[1], items: [] }))
    const wrapper = mountExplorer(fetchMock)
    await flushPromises()
    const select = wrapper.findComponent({ name: 'ElSelect' })
    expect(select.findAllComponents({ name: 'ElOption' }).map(option => option.props('label')))
      .toEqual(['Медиатека · публичный', 'Архив · приватный'])
    expect(String(fetchMock.mock.calls[1]?.[0])).toContain('disk=media')
    select.vm.$emit('update:modelValue', 'archive')
    select.vm.$emit('change', 'archive')
    await flushPromises()
    expect(String(fetchMock.mock.calls[2]?.[0])).toContain('disk=archive')
    wrapper.unmount()
  })

  it('limits picker disks by code even when labels match', async () => {
    const archive = { code: 'archive', label: 'Файлы', visibility: 'private' }
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(json({
        items: [{ code: 'media', label: 'Файлы', visibility: 'public' }, archive],
        permissions: listing.permissions,
      }))
      .mockResolvedValueOnce(json({ ...listing, disk: archive, items: [] }))
    vi.stubGlobal('fetch', fetchMock)
    const wrapper = mount(FileExplorer, {
      props: { accessToken: 'token', permissions: new Set(['core.file.read']), picker: true, allowedStorages: ['archive'] },
    })
    await flushPromises()
    const options = wrapper.findComponent({ name: 'ElSelect' }).findAllComponents({ name: 'ElOption' })
    expect(options.map(option => option.props('value'))).toEqual(['archive'])
    expect(String(fetchMock.mock.calls[1]?.[0])).toContain('disk=archive')
    wrapper.unmount()
  })

  it('opens the configured initial storage and resolved folder', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(json({
        items: [{ code: 'public', visibility: 'public' }, { code: 'private', visibility: 'private' }],
        permissions: { read: true, create: true, update: true, delete: true },
      }))
      .mockResolvedValueOnce(json({ kind: 'folder', source_file_id: null, id: 9, folder_id: null, storage: 'private', name: 'mail', created_at: '', updated_at: '' }))
      .mockResolvedValueOnce(json({ ...listing, disk: { code: 'private', visibility: 'private' }, folder: { kind: 'folder', source_file_id: null, id: 9, folder_id: null, storage: 'private', name: 'mail', created_at: '', updated_at: '' } }))
    vi.stubGlobal('fetch', fetchMock)
    mount(FileExplorer, {
      props: { accessToken: 'token', permissions: new Set(['core.file.read']), initialStorage: 'private', initialPath: 'mail/uploads' },
    })
    await flushPromises()
    expect(fetchMock.mock.calls[1]?.[0]).toBe('/api/files/folders/resolve?disk=private&path=mail%2Fuploads')
    expect(fetchMock.mock.calls[2]?.[0]).toBe('/api/files/items?disk=private&folder_id=9')
  })

  it('ensures and opens a missing configured folder when create is allowed', async () => {
    const folder = { kind: 'folder', source_file_id: null, id: 9, folder_id: 4, storage: 'private', name: 'uploads', created_at: '', updated_at: '' }
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(json({
        items: [{ code: 'private', visibility: 'private' }],
        permissions: { read: true, create: true, update: false, delete: false },
      }))
      .mockResolvedValueOnce(jsonError(404))
      .mockResolvedValueOnce(json(folder))
      .mockResolvedValueOnce(json({ ...listing, disk: { code: 'private', visibility: 'private' }, folder }))
    vi.stubGlobal('fetch', fetchMock)
    mount(FileExplorer, {
      props: { accessToken: 'token', permissions: new Set(['core.file.read', 'core.file.create']), initialStorage: 'private', initialPath: 'mail/uploads' },
    })
    await flushPromises()
    expect(fetchMock.mock.calls[2]?.[0]).toBe('/api/files/folders/ensure')
    expect(JSON.parse(String(fetchMock.mock.calls[2]?.[1]?.body))).toEqual({ disk: 'private', path: 'mail/uploads' })
    expect(fetchMock.mock.calls[3]?.[0]).toBe('/api/files/items?disk=private&folder_id=9')
  })

  it('does not ensure a missing configured folder without create permission', async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(json({
        items: [{ code: 'private', visibility: 'private' }],
        permissions: { read: true, create: false, update: false, delete: false },
      }))
      .mockResolvedValueOnce(jsonError(404))
      .mockResolvedValueOnce(json({ ...listing, disk: { code: 'private', visibility: 'private' }, folder: null }))
    vi.stubGlobal('fetch', fetchMock)
    mount(FileExplorer, {
      props: { accessToken: 'token', permissions: new Set(['core.file.read']), initialStorage: 'private', initialPath: 'mail/uploads' },
    })
    await flushPromises()
    expect(fetchMock.mock.calls.some(([url]) => url === '/api/files/folders/ensure')).toBe(false)
    expect(fetchMock.mock.calls[2]?.[0]).toBe('/api/files/items?disk=private')
  })

  it('moves the right-clicked selection through the folder dialog', async () => {
    const listing = {
      disk: { code: 'public', visibility: 'public' }, folder: null, breadcrumbs: [],
      permissions: { read: true, create: true, update: true, delete: true },
      items: [
        { kind: 'file', source_file_id: null, id: 2, folder_id: null, storage: 'public', name: 'a.txt', mime_type: 'text/plain', size: 10, created_at: '2026-01-01T10:00:00Z', updated_at: '2026-01-02T10:00:00Z' },
        { kind: 'file', source_file_id: null, id: 3, folder_id: null, storage: 'public', name: 'b.txt', mime_type: 'text/plain', size: 20, created_at: '2026-01-01T10:00:00Z', updated_at: '2026-01-02T10:00:00Z' },
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
      .mockResolvedValueOnce(json({ kind: 'file', source_file_id: null, id: 4, name: 'upload.txt' }))
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


describe('FileExplorer image delivery and deletion', () => {
 afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals() })
 const raster = { ...listing.items[1], name: 'photo.png', mime_type: 'image/png' }
 function mockFiles(extra?: (url: string) => Response | undefined) {
   const fetcher = vi.fn(async (url: string) => extra?.(url) ?? (url === '/api/files/disks'
     ? json({ items: [{ code: 'public', label: 'Files', visibility: 'public' }], permissions: listing.permissions })
     : url.startsWith('/api/files/items') ? json({ ...listing, items: [raster] }) : new Response('image')))
   vi.stubGlobal('fetch', fetcher)
   vi.stubGlobal('URL', { ...URL, createObjectURL: vi.fn(() => 'blob:test'), revokeObjectURL: vi.fn() })
   return fetcher
 }
 it('uses a bounded thumbnail for tiles and loads full content only when opened', async () => {
   const fetcher = mockFiles()
   const wrapper = mount(FileExplorer, { props: { accessToken: 'token', permissions: new Set(['core.file.read']) } })
   await flushPromises()
   expect(fetcher.mock.calls.some(([u]) => u.includes('/2/thumbnail?'))).toBe(true)
   expect(fetcher.mock.calls.some(([u]) => u.endsWith('/preview'))).toBe(false)
   await wrapper.find('.file-tile').trigger('dblclick'); await flushPromises()
   expect(fetcher.mock.calls.some(([u]) => u === '/api/files/2/preview')).toBe(true)
   wrapper.unmount()
 })
 it('requests impact and warns about derivatives before confirmed cascade', async () => {
   const fetcher = mockFiles((url) => url === '/api/files/delete-impact' ? json({ total_files: 3, derived_files: 2, media_references: 1, file_field_references: 0, token: 'snapshot' }) : url === '/api/files/delete' ? new Response(null, { status: 204 }) : undefined)
   const confirm = vi.spyOn(ElMessageBox, 'confirm').mockResolvedValue('confirm' as Awaited<ReturnType<typeof ElMessageBox.confirm>>)
   const wrapper = mount(FileExplorer, { props: { accessToken: 'token', permissions: new Set(['core.file.read','core.file.delete']) } })
   await flushPromises(); await wrapper.find('.file-tile').trigger('click')
   const button = wrapper.findAll('button').find(b => b.text().includes('Удалить'))
   expect(button).toBeTruthy(); await button!.trigger('click'); await flushPromises()
   expect(String(confirm.mock.calls[0]?.[0])).toContain('производные изображения: 2')
   expect(String(confirm.mock.calls[0]?.[0])).toContain('медиа: 1')
   const call = fetcher.mock.calls.find(([u]) => u === '/api/files/delete') as unknown as [string, RequestInit]
   expect(JSON.parse(String(call[1].body))).toMatchObject({ policy: 'confirmed_media_cascade', impact_token: 'snapshot' })
   wrapper.unmount()
 })
})
