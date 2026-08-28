// @vitest-environment jsdom

import { config, enableAutoUnmount, flushPromises, mount, shallowMount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useSelectedSite } from '../../composables/use-selected-site'
import FormActionDialog from './FormActionDialog.vue'
import FormBuilderView from './FormBuilderView.vue'
import FormFieldDialog from './FormFieldDialog.vue'
import FormsListView from './FormsListView.vue'
import FormsResultsView from './FormsResultsView.vue'
import type { FormEditorResponse } from './types'

enableAutoUnmount(afterEach)
const permissions = new Set([
  'forms.form.read', 'forms.form.create', 'forms.form.update', 'forms.form.delete',
  'forms.result.read', 'forms.result.update', 'forms.result.delete',
  'forms.status.read', 'forms.status.create', 'forms.status.update', 'forms.status.delete',
  'forms.action.read', 'forms.action.create', 'forms.action.update', 'forms.action.delete',
  'mail.template.read', 'core.file.read',
])
function response(value: unknown, status = 200): Response { return { ok: status >= 200 && status < 300, status, json: async () => value } as Response }
function router() { return createRouter({ history: createMemoryHistory(), routes: [
  { name: 'forms.list', path: '/admin/forms', component: { template: '<div />' } },
  { name: 'forms.edit', path: '/admin/forms/:formId', component: { template: '<div />' } },
  { name: 'forms.results', path: '/admin/forms-results', component: { template: '<div />' } },
  { name: 'forms.results.detail', path: '/admin/forms-results/:resultId', component: { template: '<div />' } },
] }) }

const editor: FormEditorResponse = {
  form: { id: 9, site_id: 5, code: 'feedback', name: 'Обратная связь', description: '', enabled: true, created_at: '', updated_at: '' },
  fields: [
    { id: 1, form_id: 9, code: 'privacy_consent', type: 'forms.consent', label: 'Согласие', required: true, rules: [], options: {}, result_label: 'Согласие', show_in_results: true, result_position: 0, created_at: '', updated_at: '' },
    { id: 2, form_id: 9, code: 'captcha', type: 'forms.captcha', label: 'CAPTCHA', required: true, rules: [], options: {}, result_label: '', show_in_results: false, result_position: 1, created_at: '', updated_at: '' },
    { id: 3, form_id: 9, code: 'email', type: 'email', label: 'Email', required: true, rules: [], result_label: 'Контакт', show_in_results: true, result_position: 2, created_at: '', updated_at: '' },
  ],
  elements: [{ id: 4, form_id: 9, code: 'submit', type: 'submit_button', config: { label: 'Отправить' }, created_at: '', updated_at: '' }],
  layout: [
    { id: 10, form_id: 9, kind: 'field', field_id: 1, position: 0 },
    { id: 11, form_id: 9, kind: 'field', field_id: 2, position: 1 },
    { id: 12, form_id: 9, kind: 'field', field_id: 3, position: 2 },
    { id: 13, form_id: 9, kind: 'element', element_id: 4, position: 3 },
  ],
  statuses: [{ id: 5, form_id: 9, code: 'new', name: 'Новый', color: '#409eff', position: 0, is_default: true, created_at: '', updated_at: '' }],
  actions: [], available_field_types: ['string', 'email', 'forms.captcha', 'forms.consent', 'forms.upload'],
  available_element_types: [{ code: 'text', label: 'Текст', fields: [] }, { code: 'submit_button', label: 'Кнопка', fields: [] }],
  available_action_types: [{ code: 'mail', label: 'Письмо', available: true, editor_code: 'forms.mail', fields: [] }],
}

const tableStubs = {
  ElTable: { name: 'ElTable', props: ['data'], template: '<div><slot /></div>' },
  ElTableColumn: { name: 'ElTableColumn', props: ['label', 'prop'], template: '<div />' },
  ElButton: { name: 'ElButton', template: '<button><slot /></button>' },
}
const builderStubs = {
  ...tableStubs,
  ElTabs: { name: 'ElTabs', template: '<div><slot /></div>' },
  ElTabPane: { name: 'ElTabPane', props: ['label', 'name'], template: '<section><slot /></section>' },
  ElCard: { name: 'ElCard', template: '<div><slot name="header" /><slot /></div>' },
}

beforeEach(() => { config.global.renderStubDefaultSlot = false; useSelectedSite().setSelected({ id: 5, domain: 'example.test' }) })
afterEach(() => { config.global.renderStubDefaultSlot = false; useSelectedSite().reset(); vi.restoreAllMocks() })

describe('Forms admin UI', () => {
  it('protects Forms routes with Forms permissions', () => {
    const wrapper = shallowMount(FormsListView, { props: { accessToken: 'token', permissions: new Set<string>() }, global: { plugins: [router()] } })
    expect(wrapper.findComponent({ name: 'AccessDeniedView' }).exists()).toBe(true)
    expect(wrapper.findComponent({ name: 'ElTable' }).exists()).toBe(false)
  })

  it('loads the site-scoped form list and exposes creation', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => response({ items: [editor.form], pagination: { page: 1, per_page: 20, total: 1 } })))
    const wrapper = shallowMount(FormsListView, { props: { accessToken: 'token', permissions }, global: { plugins: [router()], stubs: tableStubs } })
    await flushPromises()
    expect(wrapper.getComponent({ name: 'ElTable' }).props('data')).toEqual([editor.form])
    expect(wrapper.findAllComponents({ name: 'ElButton' }).some((button) => button.text() === 'Создать форму')).toBe(true)
  })

  it('renders a structured builder with protected mandatory items and result metadata editor', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      if (String(input).endsWith('/forms/9/editor')) return response(editor)
      throw new Error(`unexpected request ${input}`)
    }))
    const instanceRouter = router(); await instanceRouter.push({ name: 'forms.edit', params: { formId: 9 } }); await instanceRouter.isReady()
    const wrapper = shallowMount(FormBuilderView, { props: { accessToken: 'token', permissions }, global: { plugins: [instanceRouter], stubs: builderStubs } })
    await flushPromises()
    expect(wrapper.findAllComponents({ name: 'ElTabPane' }).map((item) => item.props('label'))).toEqual(['Поля', 'Элементы', 'Структура', 'Статусы', 'Действия'])
    expect(wrapper.getComponent(FormFieldDialog).props('fields')).toEqual(editor.fields)
    expect(wrapper.getComponent(FormFieldDialog).props('fields')).toEqual(expect.arrayContaining([
      expect.objectContaining({ code: 'privacy_consent', required: true, show_in_results: true, result_label: 'Согласие' }),
      expect.objectContaining({ code: 'captcha', required: true }),
    ]))
  })

  it('shows the dedicated Mail action editor and maps only scalar values plus upload attachments', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => response({ items: [{
      id: 3, site_id: 5, code: 'feedback', name: 'Feedback', enabled: true,
      from: { name: '', email: 'noreply@example.test' }, to: [], cc: [], bcc: [], reply_to: null,
      subject: '', content_type: 'text', text_body: '', html_body: '', attachments: [],
      variables: [{ key: 'email', type: 'email', label: 'Email', required: true, rules: [] }], created_at: '', updated_at: '',
    }], pagination: { page: 1, per_page: 100, total: 1 } })))
    const wrapper = mount(FormActionDialog, { props: {
      modelValue: true, action: { id: 1, form_id: 9, code: 'mail', name: 'Письмо', enabled: true, trigger: { type: 'submitted' }, action_type: 'mail', config: { template_code: 'feedback', values: { email: 'email' }, attachments: [] }, position: 0, created_at: '', updated_at: '' },
      actionTypes: editor.available_action_types, fields: [...editor.fields, { ...editor.fields[2]!, id: 8, code: 'files', type: 'forms.upload', label: 'Файлы' }], statuses: editor.statuses,
      accessToken: 'token', siteID: 5, permissions, nextPosition: 0,
    } })
    await flushPromises()
    expect(wrapper.text()).toContain('Переменные шаблона')
    expect(wrapper.text()).toContain('Вложения из отправки')
    const optionValues = wrapper.findAllComponents({ name: 'ElOption' }).map((item) => item.props('value'))
    expect(optionValues).toContain('email')
    expect(optionValues).toContain('files')
    expect(optionValues).not.toContain('captcha')
  })

  it('uses backend dynamic result columns for a selected form', async () => {
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/forms/forms?')) return response({ items: [editor.form], pagination: { page: 1, per_page: 100, total: 1 } })
      if (url.includes('/forms/results?')) return response({ items: [{ id: 31, form_name: 'Обратная связь', form_code: 'feedback', status_name: 'Новый', values: { email: 'person@example.test' } }], columns: [editor.fields[2]], pagination: { page: 1, per_page: 20, total: 1 } })
      throw new Error(`unexpected request ${url}`)
    }))
    const wrapper = shallowMount(FormsResultsView, { props: { accessToken: 'token', permissions }, global: { plugins: [router()], stubs: tableStubs } })
    await flushPromises()
    expect(wrapper.getComponent({ name: 'ElTable' }).props('data')).toHaveLength(1)
    expect(wrapper.findAllComponents({ name: 'ElTableColumn' }).some((column) => column.props('label') === 'Контакт')).toBe(true)
  })
})
