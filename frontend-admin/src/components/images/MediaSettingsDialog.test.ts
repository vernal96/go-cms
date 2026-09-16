// @vitest-environment jsdom
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import MediaSettingsDialog from './MediaSettingsDialog.vue'
import DynamicFieldsForm from '../fields/DynamicFieldsForm.vue'
import DynamicField from '../fields/DynamicField.vue'
import { adminPermissionsKey } from '../../admin-context'
const fields = [{ key: 'alt', label: 'Alt', type: 'string', required: false, rules: [] }]
const state = { code: 'image', values: { alt: 'original' }, fields, expected_updated_at: '2026-09-16T00:00:00Z' }
const props = { modelValue: true, mediaId: 7, siteId: 2, settingsCode: 'image', accessToken: 'token' }
const stubs = { ElDialog: { template: '<div><slot /><slot name="footer" /></div>' } }
afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals() })
it('loads current settings, saves a versioned draft and closes only after success', async () => {
 const fetcher = vi.fn(async (_url: string, _init?: RequestInit) => new Response(JSON.stringify(state)))
 vi.stubGlobal('fetch', fetcher)
 const wrapper = mount(MediaSettingsDialog, { props, global: { stubs } })
 await flushPromises()
 expect(fetcher.mock.calls[0]?.[0]).toBe('/api/sites/2/media/7/settings?code=image')
 await wrapper.find('input').setValue('changed')
 await wrapper.findAll('button').find(b => b.text() === 'Сохранить')!.trigger('click')
 await flushPromises()
 const init = fetcher.mock.calls[1]?.[1] as RequestInit
 expect(JSON.parse(init.body as string)).toEqual({ code: 'image', values: { alt: 'changed' }, expected_updated_at: state.expected_updated_at })
 expect(wrapper.emitted('update:modelValue')).toEqual([[false]])
 wrapper.unmount()
})
it('cancel never writes the draft', async () => {
 const fetcher = vi.fn(async (_url: string, _init?: RequestInit) => new Response(JSON.stringify(state)))
 vi.stubGlobal('fetch', fetcher)
 const wrapper = mount(MediaSettingsDialog, { props, global: { stubs } })
 await flushPromises()
 await wrapper.find('input').setValue('discard')
 await wrapper.findAll('button').find(b => b.text() === 'Отмена')!.trigger('click')
 expect(fetcher).toHaveBeenCalledTimes(1)
 expect(wrapper.emitted('update:modelValue')).toEqual([[false]])
 wrapper.unmount()
})
it.each([422, 409])('preserves the draft on HTTP %s and does not close', async status => {
 vi.stubGlobal('fetch', vi.fn(async (_url, init) => init?.method === 'PUT'
  ? new Response(JSON.stringify({ error: { code: 'validation_error', message: 'Invalid', details: { fields: [{ key: 'alt', rule: 'max', param: '3' }] } } }), { status })
  : new Response(JSON.stringify(state))))
 const wrapper = mount(MediaSettingsDialog, { props, global: { stubs } })
 await flushPromises()
 await wrapper.find('input').setValue('draft')
 await wrapper.findAll('button').find(b => b.text() === 'Сохранить')!.trigger('click')
 await flushPromises()
 expect(wrapper.find('input').element.value).toBe('draft')
 expect(wrapper.emitted('update:modelValue')).toBeUndefined()
 if (status === 409) expect(wrapper.text()).toContain('Загрузить заново')
 else expect(wrapper.findComponent(DynamicFieldsForm).props('errors')).toHaveProperty('alt')
 wrapper.unmount()
})
it('ignores a late response after switching Media', async () => {
 let resolve!: (value: Response) => void
 vi.stubGlobal('fetch', vi.fn(async (url: string) => url.includes('/7/') ? new Promise<Response>(r => { resolve = r }) : new Response(JSON.stringify({ ...state, values: { alt: 'second' } }))))
 const wrapper = mount(MediaSettingsDialog, { props, global: { stubs } })
 await wrapper.setProps({ mediaId: 8 }); await flushPromises()
 resolve(new Response(JSON.stringify(state))); await flushPromises()
 expect(wrapper.findComponent(DynamicFieldsForm).props('modelValue')).toEqual({ alt: 'second' })
 wrapper.unmount()
})
it.each([false, true])('passes the named settings and site through media lists (multiple=%s)', async multiple => {
 vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify({ editable: false, current_file: { id: 1 } }))))
 vi.stubGlobal('URL', { createObjectURL: () => 'blob:test', revokeObjectURL: vi.fn() })
 const wrapper = mount(DynamicField, { props: { field: { key: 'photo', label: 'Photo', type: 'media', required: false, rules: [], options: { settings_code: 'image', multiple } }, siteId: 2, accessToken: 'token', modelValue: multiple ? [7, 8] : 7 }, global: { stubs: { FilePickerDialog: true, ImageEditor: true, MediaSettingsDialog: true }, provide: { [adminPermissionsKey as symbol]: ref(new Set(['core.media.read', 'core.media.update'])) } } })
 await flushPromises()
 const buttons = wrapper.findAll('button').filter(b => b.text() === 'Настройки')
 expect(buttons).toHaveLength(multiple ? 2 : 1)
 await buttons.at(-1)!.trigger('click')
 const dialog = wrapper.findComponent(MediaSettingsDialog)
 expect(dialog.props('mediaId')).toBe(multiple ? 8 : 7)
 expect(dialog.props('siteId')).toBe(2)
 expect(dialog.props('settingsCode')).toBe('image')
 wrapper.unmount()
})
