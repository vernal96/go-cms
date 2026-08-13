// @vitest-environment jsdom

import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import FilesystemView from './FilesystemView.vue'

describe('FilesystemView', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('shows access denied and does not mount the explorer without file read', () => {
    const fetchMock = vi.fn()
    vi.stubGlobal('fetch', fetchMock)
    const wrapper = mount(FilesystemView, {
      props: { accessToken: 'token', permissions: new Set(['core.file.create']) },
      global: {
        stubs: {
          AccessDeniedView: { template: '<div data-testid="denied">Нет доступа</div>' },
          FileExplorer: { template: '<div data-testid="explorer" />' },
        },
      },
    })
    expect(wrapper.find('[data-testid="denied"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="explorer"]').exists()).toBe(false)
    expect(fetchMock).not.toHaveBeenCalled()
  })
})
