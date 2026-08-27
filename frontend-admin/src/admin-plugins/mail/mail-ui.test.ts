// @vitest-environment jsdom

import { config, flushPromises, mount, shallowMount } from '@vue/test-utils'
import { createMemoryHistory, createRouter } from 'vue-router'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useSelectedSite } from '../../composables/use-selected-site'
import MailAttachmentsEditor from './MailAttachmentsEditor.vue'
import MailHtmlPreview from './MailHtmlPreview.vue'
import MailHistoryView from './MailHistoryView.vue'
import MailMessageDetailView from './MailMessageDetailView.vue'
import MailSendView from './MailSendView.vue'
import MailTemplateFormView from './MailTemplateFormView.vue'
import MailVariablesEditor from './MailVariablesEditor.vue'
import { setMailTemplateEnabled } from './api'

const permissions = new Set([
  'mail.template.read', 'mail.template.create', 'mail.template.update', 'mail.template.delete',
  'mail.message.read', 'mail.message.create', 'mail.message.delete', 'core.file.read',
])

function router() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { name: 'mail.templates', path: '/admin/mail/templates', component: { template: '<div />' } },
      { name: 'mail.templates.create', path: '/admin/mail/templates/new', component: { template: '<div />' } },
      { name: 'mail.history', path: '/admin/mail/history', component: { template: '<div />' } },
      { name: 'mail.history.detail', path: '/admin/mail/history/:messageId', component: { template: '<div />' } },
    ],
  })
}

function response(value: unknown, status = 200): Response {
  return { ok: status >= 200 && status < 300, status, json: async () => value } as Response
}

beforeEach(() => {
  config.global.renderStubDefaultSlot = true
  useSelectedSite().setSelected({ id: 5, domain: 'example.test' })
})

afterEach(() => {
  config.global.renderStubDefaultSlot = false
  useSelectedSite().reset()
  vi.restoreAllMocks()
})

describe('Mail admin UI', () => {
  it('isolates arbitrary HTML previews in a sandboxed iframe', () => {
    const wrapper = mount(MailHtmlPreview, { props: { html: '<script>parent.hacked=true</script>', title: undefined } })
    const iframe = wrapper.get('iframe')
    expect(iframe.attributes('sandbox')).toBe('')
    expect(iframe.attributes('srcdoc')).toContain('<script>')
    expect(wrapper.html()).not.toContain('v-html')
  })

  it('switches the template body editor and exposes compatible variable types', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => response({ items: [
      { variable: 'site.id', label: 'ID сайта', type: 'int', source: 'site' },
      { variable: 'site.field.contract', label: 'Договор', type: 'file', source: 'site' },
    ], upload_storage: 'private', upload_path: 'mail/uploads' })))
    const instanceRouter = router()
    await instanceRouter.push({ name: 'mail.templates.create' })
    await instanceRouter.isReady()
    const wrapper = shallowMount(MailTemplateFormView, {
      props: { accessToken: 'token', permissions },
      global: { plugins: [instanceRouter] },
    })
    await flushPromises()
    expect(wrapper.findComponent({ name: 'RichTextEditor' }).exists()).toBe(false)
    expect(wrapper.findComponent(MailAttachmentsEditor).props()).toEqual(expect.objectContaining({ uploadStorage: 'private', uploadPath: 'mail/uploads' }))
    const contentType = wrapper.findAllComponents({ name: 'ElSelect' }).at(-1)
    contentType?.vm.$emit('update:modelValue', 'html')
    await wrapper.vm.$nextTick()
    expect(wrapper.findComponent({ name: 'RichTextEditor' }).exists()).toBe(true)

    const variables = shallowMount(MailVariablesEditor, { props: { modelValue: [] } })
    variables.getComponent({ name: 'ElButton' }).vm.$emit('click')
    expect(variables.emitted('update:modelValue')?.[0]?.[0]).toEqual([
      expect.objectContaining({ type: 'string', required: false }),
    ])
  })

  it('uses the semantic enable endpoint', async () => {
    const fetchMock = vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      if (url.endsWith('/templates/3/enabled')) return response({ id: 3, enabled: false, updated_at: '2026-01-02T00:00:00Z' })
      throw new Error(`unexpected request ${url} ${init?.method ?? 'GET'}`)
    })
    vi.stubGlobal('fetch', fetchMock)
    await expect(setMailTemplateEnabled('token', 5, 3, false)).resolves.toEqual({ id: 3, enabled: false, updated_at: '2026-01-02T00:00:00Z' })
    const call = fetchMock.mock.calls.find(([url]) => String(url).endsWith('/templates/3/enabled'))
    expect(call?.[1]).toEqual(expect.objectContaining({ method: 'PATCH', body: JSON.stringify({ enabled: false }) }))
  })

  it('renders an access-denied route instead of an unauthorized template form', async () => {
    const instanceRouter = router()
    await instanceRouter.push({ name: 'mail.templates.create' })
    await instanceRouter.isReady()
    const wrapper = shallowMount(MailTemplateFormView, {
      props: { accessToken: 'token', permissions: new Set<string>() },
      global: { plugins: [instanceRouter] },
    })
    expect(wrapper.findComponent({ name: 'AccessDeniedView' }).exists()).toBe(true)
    expect(wrapper.findComponent({ name: 'ElForm' }).exists()).toBe(false)
  })

  it('reuses the file picker and offers only file variables for variable attachments', () => {
    const wrapper = shallowMount(MailAttachmentsEditor, {
      props: {
        modelValue: [{ source: 'variable', variable: '', filename_template: '' }, { source: 'site', variable: '', filename_template: '' }],
        variables: [
          { key: 'document', type: 'file', label: 'Документ', required: false, rules: [] },
          { key: 'name', type: 'string', label: 'Имя', required: false, rules: [] },
        ],
        siteVariables: [{ variable: 'site.field.contract', label: 'Договор', type: 'file', source: 'site' }],
        accessToken: 'token', permissions,
      },
    })
    expect(wrapper.findComponent({ name: 'FilePickerDialog' }).exists()).toBe(true)
    const options = wrapper.findAllComponents({ name: 'ElOption' }).map((item) => item.props('value'))
    expect(options).toContain('data.document')
    expect(options).toContain('site.field.contract')
    expect(options).not.toContain('data.name')
  })

  it('previews backend-rendered content, shows warnings, and queues once', async () => {
    const calls: Array<{ url: string; method: string }> = []
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input)
      calls.push({ url, method: init?.method ?? 'GET' })
      if (url.includes('/send/templates')) return response({ items: [{
        id: 3, site_id: 5, code: 'welcome', name: 'Welcome', enabled: true, transport: 'default',
        from: { name: '', email: 'noreply@example.test' }, to: [{ name: '', email: '{{data.email}}' }], cc: [], bcc: [], reply_to: null,
        subject: 'Hello', content_type: 'html', text_body: '', html_body: '<p>{{data.name}}</p>', attachments: [],
        variables: [{ key: 'email', type: 'email', label: 'Email', required: false, rules: [] }], created_at: '', updated_at: '',
      }], pagination: { page: 1, per_page: 100, total: 1 } })
      if (url.endsWith('/preview')) return response({
        from: { name: '', email: 'noreply@example.test' }, to: [{ name: '', email: 'person@example.test' }], cc: [], bcc: [],
        subject: 'Hello', content_type: 'html', text_body: '', html_body: '<p></p>', attachments: [],
        warnings: [{ field: 'html_body', variable: 'data.name', message: 'missing value' }],
      })
      if (url.endsWith('/send')) return response({ id: 91 }, 202)
      throw new Error(`unexpected request ${url}`)
    }))

    const wrapper = shallowMount(MailSendView, { props: { accessToken: 'token', permissions } })
    await flushPromises()
    wrapper.getComponent({ name: 'ElSelect' }).vm.$emit('update:modelValue', 3)
    await wrapper.vm.$nextTick()
    const dynamicForm = wrapper.getComponent({ name: 'DynamicFieldsForm' })
    dynamicForm.vm.$emit('update:modelValue', { email: 'person@example.test' })
    await wrapper.vm.$nextTick()
    const buttons = () => wrapper.findAllComponents({ name: 'ElButton' })
    buttons().find((item) => item.text() === 'Предпросмотр')?.vm.$emit('click')
    await flushPromises()
    expect(wrapper.findComponent(MailHtmlPreview).props('html')).toBe('<p></p>')
    expect(wrapper.text()).toContain('data.name')
    buttons().find((item) => item.text() === 'Поставить в очередь')?.vm.$emit('click')
    await flushPromises()
    expect(calls.filter((item) => item.url.endsWith('/send') && item.method === 'POST')).toHaveLength(1)
  })

  it('loads server-paginated history and message attempts by site', async () => {
    const message = {
      id: 91, site_id: 5, template_id: null, template_code: 'welcome', template_name: 'Welcome', transport: 'default', rfc_message_id: '<91@example.test>',
      from: { name: '', email: 'noreply@example.test' }, to: [{ name: '', email: 'person@example.test' }], cc: [], bcc: [], reply_to: null,
      subject: 'Hello', content_type: 'text', text_body: 'Body', html_body: '', attachments: [], status: 'accepted', origin: 'manual', origin_source: '', origin_event: '', origin_reference: '', recipients: ['person@example.test'],
      requested_at: '2026-08-27T10:00:00Z', requested_by: 7, requested_by_name: 'Editor', accepted_at: '2026-08-27T10:00:01Z',
      created_at: '2026-08-27T10:00:00Z', updated_at: '2026-08-27T10:00:01Z', attempt_count: 1,
      latest_attempt: { id: 1, message_id: 91, attempt_number: 1, transport: 'default', driver: 'smtp', started_at: '2026-08-27T10:00:00Z', finished_at: '2026-08-27T10:00:01Z', status: 'accepted', remote_message_id: 'remote', response_code: '250', safe_error: '', created_at: '2026-08-27T10:00:00Z' },
    }
    vi.stubGlobal('fetch', vi.fn(async (input: RequestInfo | URL) => {
      const url = String(input)
      if (url.includes('/messages?')) return response({ items: [message], pagination: { page: 1, per_page: 20, total: 1 } })
      if (url.endsWith('/messages/91')) return response({ message, attempts: [message.latest_attempt] })
      throw new Error(`unexpected request ${url}`)
    }))
    const instanceRouter = router()
    await instanceRouter.push({ name: 'mail.history' })
    await instanceRouter.isReady()
    config.global.renderStubDefaultSlot = false
    const cardStub = { template: '<div><slot /></div>' }
    const history = shallowMount(MailHistoryView, { props: { accessToken: 'token', permissions }, global: { plugins: [instanceRouter], stubs: { ElCard: cardStub } } })
    await flushPromises()
    expect(history.getComponent({ name: 'ElTable' }).props('data')).toEqual([message])
    await instanceRouter.push({ name: 'mail.history.detail', params: { messageId: 91 } })
    const detail = shallowMount(MailMessageDetailView, { props: { accessToken: 'token', permissions }, global: { plugins: [instanceRouter], stubs: { ElCard: cardStub } } })
    await flushPromises()
    const tables = detail.findAllComponents({ name: 'ElTable' })
    expect(tables.at(-1)?.props('data')).toEqual([message.latest_attempt])
    expect(detail.findComponent(MailHtmlPreview).exists()).toBe(false)
  })
})
