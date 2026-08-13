// @vitest-environment jsdom

import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'

import FilePickerDialog from '../components/files/FilePickerDialog.vue'
import ProfileView from './ProfileView.vue'

const profile = {
  user: {
    id: 1,
    login: 'admin',
    email: 'admin@example.test',
    name: 'Администратор',
    last_name: null,
    middle_name: null,
    phone: null,
    color_scheme: 'system',
    accent_color: 'blue',
    avatar: null,
  },
}

function response(payload: unknown): Response {
  return new Response(JSON.stringify(payload), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  })
}

describe('ProfileView', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('loads self-service tabs and keeps login and email read-only', async () => {
    const fetchMock = vi.fn().mockResolvedValue(response(profile))
    vi.stubGlobal('fetch', fetchMock)
    const wrapper = mount(ProfileView, {
      props: { accessToken: 'token', permissions: new Set<string>() },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Личные данные')
    expect(wrapper.text()).toContain('Безопасность')
    expect(wrapper.text()).not.toContain('Оформление')
    const disabledInputs = wrapper.findAll('input:disabled').map((input) => (input.element as HTMLInputElement).value)
    expect(disabledInputs).toEqual(expect.arrayContaining(['admin', 'admin@example.test']))
    expect(wrapper.text()).not.toContain('Выбрать файл')
    expect(fetchMock).toHaveBeenCalledWith('/api/admin/profile', expect.anything())
  })

  it('offers the filesystem picker only with file read access', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(response(profile)))
    const wrapper = mount(ProfileView, {
      props: { accessToken: 'token', permissions: new Set(['core.file.read']) },
    })
    await flushPromises()

    expect(wrapper.text()).toContain('Выбрать файл')
  })

  it('selects an existing image and sends only its file id', async () => {
    const updated = {
      user: {
        ...profile.user,
        avatar: {
          file_id: 7,
          name: 'avatar.png',
          mime_type: 'image/png',
          size: 128,
          updated_at: '2026-08-13T10:00:00Z',
        },
      },
    }
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(response(profile))
      .mockResolvedValueOnce(response(updated))
      .mockResolvedValueOnce(new Response('avatar', { status: 200 }))
    vi.stubGlobal('fetch', fetchMock)
    vi.stubGlobal('URL', {
      ...URL,
      createObjectURL: vi.fn(() => 'blob:avatar'),
      revokeObjectURL: vi.fn(),
    })
    const wrapper = mount(ProfileView, {
      props: { accessToken: 'token', permissions: new Set(['core.file.read']) },
    })
    await flushPromises()

    wrapper.findComponent(FilePickerDialog).vm.$emit('select', {
      kind: 'file', id: 7, parent_id: null, storage: 'private', name: 'avatar.png',
      mime_type: 'image/png', size: 128,
      created_at: '2026-08-13T10:00:00Z', updated_at: '2026-08-13T10:00:00Z',
    })
    await flushPromises()

    expect(fetchMock).toHaveBeenCalledWith('/api/admin/profile/avatar', expect.objectContaining({
      method: 'PUT',
      body: JSON.stringify({ file_id: 7 }),
    }))
    expect(wrapper.emitted('updated')).toHaveLength(1)
  })

})
