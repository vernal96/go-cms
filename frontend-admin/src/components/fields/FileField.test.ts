// @vitest-environment jsdom
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import FileField from './FileField.vue'
import FilePickerDialog from '../files/FilePickerDialog.vue'
import { adminAccessTokenKey, adminPermissionsKey } from '../../admin-context'
afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals() })
it('keeps a generic File ID and uses a thumbnail without creating Media', async () => {
 const file = { id: 3, kind: 'file', folder_id: null, source_file_id: null, storage: 'public', name: 'photo.png', mime_type: 'image/png' }
 const fetcher = vi.fn(async (url: string) => url === '/api/files/3' ? new Response(JSON.stringify(file)) : new Response('image'))
 vi.stubGlobal('fetch', fetcher); vi.stubGlobal('URL', { ...URL, createObjectURL: vi.fn(() => 'blob:file'), revokeObjectURL: vi.fn() })
 const wrapper = mount(FileField, { props: { modelValue: 3 }, global: { provide: { [adminAccessTokenKey as symbol]: ref('token'), [adminPermissionsKey as symbol]: ref(new Set(['core.file.read'])) } } })
 await flushPromises()
 expect(fetcher.mock.calls.some(([u]) => u.includes('/3/thumbnail?'))).toBe(true)
 wrapper.findComponent(FilePickerDialog).vm.$emit('select', { ...file, id: 7 }); await flushPromises()
 expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([7])
 expect(fetcher.mock.calls.some(([u]) => u.includes('/media') || u.endsWith('/preview'))).toBe(false)
 wrapper.unmount()
})
