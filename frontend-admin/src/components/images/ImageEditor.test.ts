// @vitest-environment jsdom
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { ElDialog } from 'element-plus'
import ImageEditor from './ImageEditor.vue'
const crop = vi.hoisted(() => ({ x: 2, y: 3, width: 80, height: 60, rotate: 90, scaleX: -1, scaleY: 1 }))
vi.mock('cropperjs', () => ({ default: class {
 data = { x: 0, y: 0, width: 1200, height: 800, rotate: 0, scaleX: 1, scaleY: 1 }
 canvas = { width: 740, naturalWidth: 1200, naturalHeight: 800 }
 constructor(private element: HTMLImageElement, options: { ready: () => void }) { queueMicrotask(options.ready) }
 getData() { return { ...this.data } }
 setData(data: Partial<typeof this.data>) {
  Object.assign(this.data, data)
  const rotated = this.data.rotate % 180 !== 0
  this.canvas.naturalWidth = rotated ? 800 : 1200
  this.canvas.naturalHeight = rotated ? 1200 : 800
 }
 getImageData() { return { naturalWidth: 1200, naturalHeight: 800 } }
 getContainerData() { return { width: parseFloat(this.element.parentElement!.style.width) || 740, height: parseFloat(this.element.parentElement!.style.height) || 520 } }
 getCanvasData() { return { ...this.canvas } }
 setCanvasData(data: { width: number }) { this.canvas.width = data.width }
 clear() {} crop() {} destroy() {}
 zoom(ratio: number) { this.canvas.width *= 1 + ratio }
 scaleX(scale: number) { this.data.scaleX = scale }
 scaleY(scale: number) { this.data.scaleY = scale }
} }))
const file = { id: 1, kind: 'file', folder_id: null, source_file_id: null, mime_type: 'image/png', name: 'photo.png' }
const original = { media_id: 5, current_file: file, original_file: file, expected_updated_at: '2026-09-15T10:00:00Z', transform: null, editable: true, can_restore: false, limits: { output_dimension: 4096, min_quality: 1, max_quality: 100 } }
const derived = { ...original, current_file: { ...file, id: 2, source_file_id: 1 }, can_restore: true, transform: { crop: { x: crop.x, y: crop.y, width: crop.width, height: crop.height }, rotate: crop.rotate, scale_x: crop.scaleX, scale_y: crop.scaleY, width: 320, height: 0, fit: 'contain', position: 'center', quality: 85 } }
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
 beforeEach(() => {
  vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockReturnValue(740)
  vi.spyOn(HTMLElement.prototype, 'clientHeight', 'get').mockReturnValue(520)
  vi.stubGlobal('ResizeObserver', class { observe() {} disconnect() {} unobserve() {} })
 })
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
 it('rotates the complete image without changing its pixel dimensions', async () => {
  const { wrapper, fetcher } = await setup()
  await wrapper.findAll('button').find(b => b.text() === '↷ 90°')!.trigger('click'); await flushPromises()
  await wrapper.findAll('button').find(b => b.text() === 'Сохранить')!.trigger('click'); await flushPromises()
  const call = fetcher.mock.calls.find(([, i]) => i?.method === 'POST')
  expect(JSON.parse(String(call?.[1]?.body)).transform).toMatchObject({ rotate: 90, crop: { x: 0, y: 0, width: 800, height: 1200 }, width: 0, height: 0 })
  wrapper.unmount()
 })
 it('rotates the saved selection and swaps explicit and automatic output dimensions', async () => {
  const { wrapper, fetcher } = await setup(true)
  await wrapper.findAll('button').find(b => b.text() === '↶ 90°')!.trigger('click'); await flushPromises()
  await wrapper.findAll('button').find(b => b.text() === 'Сохранить')!.trigger('click'); await flushPromises()
  const call = fetcher.mock.calls.find(([, i]) => i?.method === 'POST')
  expect(JSON.parse(String(call?.[1]?.body)).transform).toMatchObject({ rotate: 0, crop: { x: 3, y: 718, width: 60, height: 80 }, scale_x: -1, width: 0, height: 320 })
  wrapper.unmount()
 })
 it('preserves a zoomed selection through rotation and inverse rotation', async () => {
  const { wrapper, fetcher } = await setup(true)
  for (const label of ['Масштаб +', '↷ 90°', '↶ 90°']) {
   await wrapper.findAll('button').find(b => b.text() === label)!.trigger('click'); await flushPromises()
  }
  // A 1200px portrait at fit occupies 520px; the retained 1.1x zoom needs 572px.
  expect(parseFloat((wrapper.find('.image-editor-stage').element as HTMLElement).style.height)).toBeGreaterThanOrEqual(572)
  expect(parseFloat((wrapper.find('.image-editor-stage').element as HTMLElement).style.height)).toBeLessThanOrEqual(573)
  await wrapper.findAll('button').find(b => b.text() === 'Сохранить')!.trigger('click'); await flushPromises()
  const call = fetcher.mock.calls.find(([, i]) => i?.method === 'POST')
  expect(JSON.parse(String(call?.[1]?.body)).transform).toMatchObject({ crop: { x: 2, y: 3, width: 80, height: 60 }, rotate: 90, width: 320, height: 0 })
  wrapper.unmount()
 })
})
