// @vitest-environment jsdom
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { ElInput, ElOption, ElPagination, ElSelect, ElSwitch } from 'element-plus'
import * as api from './api'
import FormPickerField from './FormPickerField.vue'
import ResultsPageSizeField from './ResultsPageSizeField.vue'
import FormFieldEditor from './FormFieldEditor.vue'
import DynamicField from '../../components/fields/DynamicField.vue'
import { AdminPluginRegistry } from '../registry'
import { adminPluginRegistryKey } from '../context'
import type { FormRecord, FormsListResponse } from './types'

vi.mock('./api', () => ({ listForms: vi.fn(), getForm: vi.fn() }))
const form = (id: number, enabled = true): FormRecord => ({
  id, site_id: 5, code: `form_${id}`, name: `Форма ${id}`, enabled, description: '', created_at: '', updated_at: '',
})
const page = (items: FormRecord[], total = items.length): FormsListResponse => ({ items, pagination: { page: 1, per_page: 20, total } })
afterEach(() => { vi.clearAllMocks(); vi.useRealTimers() })

describe('Forms widget editors', () => {
  it('loads site forms, labels disabled choices, searches and changes pages', async () => {
    vi.mocked(api.listForms).mockResolvedValue(page([form(1), form(2, false)], 41))
    const wrapper = mount(FormPickerField, { props: { siteId: 5, accessToken: 'token' } })
    await flushPromises()
    expect(api.listForms).toHaveBeenLastCalledWith('token', 5, 1, 20, '')
    expect(wrapper.findAllComponents(ElOption).map((item) => item.props('label'))).toContain('Форма 2 (form_2) — отключена')
    wrapper.findComponent(ElPagination).vm.$emit('current-change', 2)
    await flushPromises()
    expect(api.listForms).toHaveBeenLastCalledWith('token', 5, 2, 20, '')
    vi.useFakeTimers()
    wrapper.findComponent(ElInput).vm.$emit('update:modelValue', 'test')
    await vi.advanceTimersByTimeAsync(250)
    await flushPromises()
    expect(api.listForms).toHaveBeenLastCalledWith('token', 5, 1, 20, 'test')
    wrapper.findComponent(ElSelect).vm.$emit('update:modelValue', 2)
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([2])
    wrapper.unmount()
  })

  it('resolves a saved choice outside the current page and discards stale site responses', async () => {
    let oldResponse!: (value: FormsListResponse) => void
    vi.mocked(api.listForms).mockImplementationOnce(() => new Promise((resolve) => { oldResponse = resolve }))
      .mockResolvedValue(page([form(20)]))
    vi.mocked(api.getForm).mockResolvedValue(form(99, false))
    const wrapper = mount(FormPickerField, { props: { siteId: 5, accessToken: 'token', modelValue: 99 } })
    await flushPromises()
    expect(wrapper.findComponent(ElOption).props('label')).toContain('отключена')
    await wrapper.setProps({ siteId: 6 })
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([null])
    await wrapper.setProps({ modelValue: null })
    await flushPromises()
    oldResponse(page([form(1)]))
    await flushPromises()
    expect(wrapper.findAllComponents(ElOption).map((item) => item.props('value'))).toEqual([20])
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([null])
    wrapper.unmount()
  })

  it('shows API permission errors without offering arbitrary IDs', async () => {
    vi.mocked(api.listForms).mockRejectedValue(new Error('Нет доступа'))
    const wrapper = mount(FormPickerField, { props: { siteId: 5, accessToken: 'token' } })
    await flushPromises()
    expect(wrapper.text()).toContain('Нет доступа')
    expect(wrapper.findAllComponents(ElOption)).toHaveLength(0)
    wrapper.unmount()
  })

  it('resolves module editors generically and supplies the page size default', async () => {
    const registry = new AdminPluginRegistry([{ code: 'forms', fieldEditors: { 'forms.results-page-size': ResultsPageSizeField } }])
    const wrapper = mount(DynamicField, {
      props: { field: { key: 'per_page', type: 'int', label: 'На странице', required: false, rules: [], editor: 'forms.results-page-size' }, modelValue: null },
      global: { provide: { [adminPluginRegistryKey as symbol]: registry } },
    })
    await flushPromises()
    expect(wrapper.findComponent(ResultsPageSizeField).exists()).toBe(true)
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([20])
    wrapper.unmount()
  })

  it('keeps public visibility independent of admin columns and locks transient fields', async () => {
    const wrapper = mount(FormFieldEditor, { props: { disabled: false, initialType: 'email', fields: [], availableTypes: ['email'].map(code => ({code,label:code,options:[]})) } })
    await flushPromises()
    const row = wrapper.findAll('.el-form-item').find((item) => item.text().includes('Показывать на сайте'))!
    row.findComponent(ElSwitch).vm.$emit('update:modelValue', true)
    await flushPromises()
    expect((wrapper.vm as unknown as { payload: () => unknown }).payload()).toMatchObject({ show_on_site: true, show_in_results: false })
    wrapper.unmount()
    const captcha = mount(FormFieldEditor, { props: { disabled: false, initialType: 'forms.captcha', fields: [], availableTypes: ['forms.captcha'].map(code => ({code,label:code,options:[]})) } })
    await flushPromises()
    const publicRow = captcha.findAll('.el-form-item').find((item) => item.text().includes('Показывать на сайте'))!
    expect(publicRow.findComponent(ElSwitch).props('disabled')).toBe(true)
    captcha.unmount()
  })
})
