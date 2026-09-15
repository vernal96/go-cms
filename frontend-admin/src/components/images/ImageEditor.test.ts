// @vitest-environment jsdom
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ElDialog } from 'element-plus'
import ImageEditor from './ImageEditor.vue'
const crop = vi.hoisted(() => ({ x: 2, y: 3, width: 80, height: 60, rotate: 90, scaleX: -1, scaleY: 1 }))
vi.mock('cropperjs', () => ({ default: class { getData() { return crop } setData() {} destroy() {} reset() {} rotate() {} zoom() {} scaleX() {} scaleY() {} } }))
const file = { id: 1, kind: 'file', folder_id: null, source_file_id: null, mime_type: 'image/png', name: 'photo.png' }
const original = { media_id: 5, current_file: file, original_file: file, expected_updated_at: '2026-09-15T10:00:00Z', transform: null, editable: true, can_restore: false, limits: { output_dimension: 4096, min_quality: 1, max_quality: 100 } }
const derived = { ...original, current_file: { ...file, id: 2, source_file_id: 1 }, can_restore: true }
function response(data: unknown) { return new Response(JSON.stringify(data), { headers: { 'Content-Type': 'application/json' } }) }
async function setup(isDerived = false) {
 let state = isDerived ? derived : original
 const fetcher = vi.fn(async (url: string, _init?: RequestInit) => {
  if (url.endsWith('/restore')) { state = original; return response(state) }
  if (url === '/api/media/5/image') return response(state)
  return new Response('image')
 })
 vi.stubGlobal('fetch', fetcher)
 vi.stubGlobal('URL', { ...URL, createObjectURL: vi.fn(() => 'blob:image'), revokeObjectURL: vi.fn() })
 const wrapper = mount(ImageEditor, { props: { modelValue: false, accessToken: 'token', baseUrl: '/api/media/5/image' }, global: { stubs: { teleport: true } } })
 await wrapper.setProps({ modelValue: true }); await flushPromises()
 wrapper.findComponent(ElDialog).vm.$emit('opened'); await flushPromises()
 return { wrapper, fetcher }
}
describe('ImageEditor', () => {
 afterEach(() => { vi.restoreAllMocks(); vi.unstubAllGlobals() })
 it('opens the saved image alongside the root source and submits transform data', async () => {
  const { wrapper, fetcher } = await setup(true)
  expect(fetcher.mock.calls.map(c => c[0])).toContain('/api/files/1/preview')
  expect(fetcher.mock.calls.map(c => c[0])).toContain('/api/files/2/thumbnail?width=256&height=256&fit=contain')
  const save = wrapper.findAll('button').find(b => b.text() === 'Сохранить')!
  await save.trigger('click'); await flushPromises()
  const call = fetcher.mock.calls.find(([u, i]) => u === '/api/media/5/image' && i?.method === 'POST')
  expect(JSON.parse(String(call?.[1]?.body))).toMatchObject({ expected_updated_at: derived.expected_updated_at, transform: { crop: { x: 2, y: 3, width: 80, height: 60 }, rotate: 90, scale_x: -1 } })
  expect(wrapper.emitted('saved')).toHaveLength(1); wrapper.unmount()
 })
 it('shows restore only for derivatives and refreshes state after restore', async () => {
  const { wrapper, fetcher } = await setup(true)
  await wrapper.findAll('button').find(b => b.text() === 'Восстановить оригинал')!.trigger('click'); await flushPromises()
  expect(fetcher.mock.calls.some(([u]) => u.endsWith('/restore'))).toBe(true)
  expect(wrapper.findAll('button').some(b => b.text() === 'Восстановить оригинал')).toBe(false)
  expect(wrapper.emitted('saved')?.[0]?.[0]).toMatchObject({ can_restore: false, current_file: { id: 1 } }); wrapper.unmount()
 })
 it('does not offer restore for an original', async () => { const { wrapper } = await setup(); expect(wrapper.text()).not.toContain('Восстановить оригинал'); wrapper.unmount() })
})
