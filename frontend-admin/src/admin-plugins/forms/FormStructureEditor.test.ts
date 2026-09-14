// @vitest-environment jsdom
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ElButton, ElInput, ElSelect, ElTree } from 'element-plus'
import FormStructureEditor from './FormStructureEditor.vue'
import FormFieldEditor from './FormFieldEditor.vue'
import * as api from './api'
import type { FormEditorResponse } from './types'

vi.mock('./api')
vi.mock('vue-router', () => ({ onBeforeRouteLeave: vi.fn(), onBeforeRouteUpdate: vi.fn() }))
enableAutoUnmount(afterEach)
afterEach(() => vi.resetAllMocks())
function fixture(): FormEditorResponse {
  return {
    form: { id: 9, site_id: 5, code: 'test', name: 'Test', description: '', enabled: true, created_at: '', updated_at: '' },
    fields: [{ id: 1, form_id: 9, code: 'email', type: 'email', label: 'Email', required: false, rules: [], result_label: '', show_in_results: true, show_on_site: false, result_position: 0, created_at: '', updated_at: '' }],
    elements: [], statuses: [], actions: [],
    layout: [{ id: 10, form_id: 9, kind: 'field', field_id: 1, position: 0 }, { id: 11, form_id: 9, kind: 'container', container_type: 'group', position: 1, config: { label: 'Контакты' } }],
    available_field_types: ['email', 'string'].map(code => ({code,label:code,options:[]})), available_element_types: [{ code: 'text', label: 'Текст', fields: [] }], available_container_types: [{ code: 'group', label: 'Группа' }, { code: 'slide', label: 'Слайд' }], available_action_types: [],
  }
}
function setup(permissions = new Set(['forms.form.update'])) {
  const detail = fixture()
  const wrapper = mount(FormStructureEditor, {
    props: { detail, accessToken: 'token', permissions, onChanged: (value: FormEditorResponse) => { void wrapper.setProps({ detail: value }) } },
    global: { stubs: { ElDialog: { props: ['modelValue', 'title'], template: '<section v-if="modelValue" class="dialog"><h3>{{ title }}</h3><slot /><slot name="footer" /></section>' } } },
  })
  vi.mocked(api.getFormEditor).mockResolvedValue(detail)
  return { wrapper, detail }
}
function button(wrapper: ReturnType<typeof setup>['wrapper'], label: string) {
  return wrapper.findAllComponents(ElButton).find(item => item.text() === label)!
}
async function choose(wrapper: ReturnType<typeof setup>['wrapper'], id: number) {
  await wrapper.get(`[data-node-id="${id}"]`).trigger('click'); await flushPromises()
}

describe('Forms structure panel', () => {
  it('creates a field in the selected container only after category, type and settings', async () => {
    const { wrapper, detail } = setup()
    await wrapper.get('button[aria-label="Добавить потомка: Контакты"]').trigger('click')
    await flushPromises()
    expect(api.createField).not.toHaveBeenCalled()
    wrapper.findAllComponents(ElSelect)[0]! .vm.$emit('update:modelValue', 'field')
    await flushPromises()
    wrapper.findAllComponents(ElSelect)[1]! .vm.$emit('update:modelValue', 'email')
    await flushPromises()
    expect(wrapper.findComponent(FormFieldEditor).exists()).toBe(true)
    const inputs = wrapper.getComponent(FormFieldEditor).findAllComponents(ElInput).map(item => item.get('input'))
    await inputs[0]!.setValue('contact'); await inputs[1]!.setValue('Контакт')
    const field = { ...detail.fields[0]!, id: 2, code: 'contact', label: 'Контакт' }
    const node = { id: 12, form_id: 9, parent_id: 11, kind: 'field' as const, field_id: 2, position: 0 }
    vi.mocked(api.createField).mockResolvedValue({ field, layout_node: node })
    vi.mocked(api.getFormEditor).mockResolvedValue({ ...detail, fields: [...detail.fields, field], layout: [...detail.layout, node] })
    await button(wrapper, 'Создать').trigger('click'); await flushPromises()
    expect(api.createField).toHaveBeenCalledWith('token', 5, 9, expect.objectContaining({ parent_id: 11, position: 0, type: 'email', code: 'contact', label: 'Контакт' }))
    expect(wrapper.get('.structure-panel h2').text()).toBe('Контакт')
    expect(button(wrapper, 'Сохранить').props('disabled')).toBe(true)
  })
  it('preserves a field draft when a drop refreshes layout and saves placement immediately', async () => {
    const { wrapper, detail } = setup()
    await choose(wrapper, 10)
    const inputs = wrapper.getComponent(FormFieldEditor).findAllComponents(ElInput).map(item => item.get('input'))
    await inputs[1]!.setValue('Черновик')
    vi.mocked(api.replaceLayout).mockImplementation(async (_token, _site, _form, nodes) => ({ nodes }))
    wrapper.getComponent(ElTree).vm.$emit('node-drop', { data: detail.layout[0] }, { data: detail.layout[1] }, 'inner', new MouseEvent('drop'))
    await flushPromises()
    expect(api.replaceLayout).toHaveBeenCalledWith('token', 5, 9, expect.arrayContaining([expect.objectContaining({ id: 10, parent_id: 11, position: 0 })]))
    expect(wrapper.getComponent(FormFieldEditor).findAllComponents(ElInput).map(item => item.get('input'))[1]!.element.value).toBe('Черновик')
    expect(api.updateField).not.toHaveBeenCalled()
  })
  it('keeps selection on stay and discards only on explicit choice', async () => {
    const { wrapper } = setup()
    await choose(wrapper, 10)
    await wrapper.getComponent(FormFieldEditor).findAllComponents(ElInput).map(item => item.get('input'))[1]!.setValue('Черновик')
    await choose(wrapper, 11)
    expect(wrapper.find('.dialog').text()).toContain('Несохранённые изменения')
    await button(wrapper, 'Остаться').trigger('click'); await flushPromises()
    expect(wrapper.findComponent(FormFieldEditor).exists()).toBe(true)
    await choose(wrapper, 11)
    await button(wrapper, 'Отбросить').trigger('click'); await flushPromises()
    expect(wrapper.get('.structure-panel h2').text()).toBe('Контакты')
    expect(api.updateField).not.toHaveBeenCalled()
  })
  it('restores server layout after a failed move', async () => {
    const { wrapper, detail } = setup()
    vi.mocked(api.replaceLayout).mockRejectedValue(new Error('Ошибка перемещения'))
    wrapper.getComponent(ElTree).vm.$emit('node-drop', { data: detail.layout[0] }, { data: detail.layout[1] }, 'inner', new MouseEvent('drop'))
    await flushPromises()
    expect(api.getFormEditor).toHaveBeenCalledWith('token', 5, 9)
    expect(wrapper.getComponent(ElTree).props('data')).toHaveLength(2)
    expect(wrapper.text()).toContain('Ошибка перемещения')
  })
  it('opens persisted fields with null rules without losing their values', async () => {
    const { wrapper, detail } = setup()
    const fromAPI = JSON.parse(JSON.stringify(detail)) as FormEditorResponse
    // Empty Go slices arrive as JSON null in the existing HTTP contract.
    Object.assign(fromAPI.fields[0]!, { rules: null, required: true })
    await wrapper.setProps({ detail: fromAPI })
    await choose(wrapper, 10)
    expect(wrapper.getComponent(FormFieldEditor).findAllComponents(ElInput).map(item => item.get('input'))[0]!.element.value).toBe('email')
    expect(button(wrapper, 'Сохранить').props('disabled')).toBe(true)
  })
  it('retains the draft and guard when saving before leaving fails', async () => {
    const { wrapper } = setup()
    await choose(wrapper, 10)
    await wrapper.getComponent(FormFieldEditor).findAllComponents(ElInput).map(item => item.get('input'))[1]!.setValue('Черновик')
    await choose(wrapper, 11)
    vi.mocked(api.updateField).mockRejectedValue(new Error('Не удалось сохранить'))
    await wrapper.findAllComponents(ElButton).filter(item => item.text() === 'Сохранить').at(-1)!.trigger('click')
    await flushPromises()
    expect(wrapper.get('.dialog').text()).toContain('Не удалось сохранить')
    expect(wrapper.getComponent(FormFieldEditor).findAllComponents(ElInput).map(item => item.get('input'))[1]!.element.value).toBe('Черновик')
    await button(wrapper, 'Остаться').trigger('click')
  })
  it('does not expose mutations in read-only mode', async () => {
    const { wrapper } = setup(new Set())
    expect(wrapper.getComponent(ElTree).props('draggable')).toBe(false)
    expect(button(wrapper, 'Добавить узел')).toBeUndefined()
    await choose(wrapper, 10)
    expect(wrapper.getComponent(FormFieldEditor).props('disabled')).toBe(true)
    expect(button(wrapper, 'Сохранить')).toBeUndefined()
  })
})
