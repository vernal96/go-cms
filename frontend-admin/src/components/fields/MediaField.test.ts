// @vitest-environment jsdom
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import DynamicField from './DynamicField.vue'
import FilePickerDialog from '../files/FilePickerDialog.vue'
import ImageEditor from '../images/ImageEditor.vue'
import { adminAccessTokenKey, adminPermissionsKey } from '../../admin-context'
import { createFieldValues, validateFieldValues } from './model'
const field = { key: 'page_media', label: 'Медиа', type: 'media', editor: 'media', required: false, rules: [] }
afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals() })
it('uses the shared editor and stores the created Media ID instead of File ID', async () => {
 const fetcher = vi.fn(async (url: string) => {
  if (url === '/api/media') return new Response(JSON.stringify({ id: 77 }))
  if (url.includes('/image')) return new Response(JSON.stringify({ editable: true, current_file: { id: 9 } }))
  return new Response('image')
 })
 vi.stubGlobal('fetch', fetcher)
 vi.stubGlobal('URL', { ...URL, createObjectURL: vi.fn(() => 'blob:media'), revokeObjectURL: vi.fn() })
 const wrapper = mount(DynamicField, { props: { field, modelValue: 31 }, global: { provide: { [adminAccessTokenKey as symbol]: ref('token'), [adminPermissionsKey as symbol]: ref(new Set(['core.media.create','core.media.update','core.file.read','core.file.create'])) }, stubs: { FilePickerDialog: true, ImageEditor: true } } })
 await flushPromises()
 expect(fetcher.mock.calls.some(([url]) => url === '/api/media/31/image')).toBe(true)
 await wrapper.findAll('button').find(b => b.text() === 'Редактировать')!.trigger('click')
 expect(wrapper.findComponent(ImageEditor).props('modelValue')).toBe(true)
 wrapper.findComponent(FilePickerDialog).vm.$emit('select', { id: 9 })
 await flushPromises()
 expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([77])
 wrapper.unmount()
})
it('allows an empty optional Media field and rejects non-reference values', () => {
 expect(createFieldValues([field])).toEqual({ page_media: null })
 expect(validateFieldValues([field], { page_media: null })).toEqual({})
 expect(validateFieldValues([field], { page_media: { file_id: 9 } })).toHaveProperty('page_media')
})
