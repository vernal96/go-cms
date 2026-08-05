// @vitest-environment jsdom

import { shallowMount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import AdminDataTable from './AdminDataTable.vue'

const baseProps = {
  columns: [{ prop: 'domain', label: 'Домен' }],
  rows: [] as Record<string, unknown>[],
  loading: false,
  error: null,
  page: 1,
  perPage: 10,
  total: 0,
  search: '',
  actions: undefined,
  emptyText: undefined,
}

describe('AdminDataTable', () => {
  it('renders mutually exclusive loading, error and empty states', async () => {
    const wrapper = shallowMount(AdminDataTable, { props: { ...baseProps, loading: true } })
    expect(wrapper.findComponent({ name: 'ElSkeleton' }).exists()).toBe(true)

    await wrapper.setProps({ loading: false, error: 'network error' })
    expect(wrapper.findComponent({ name: 'ElResult' }).exists()).toBe(true)

    await wrapper.setProps({ error: null })
    expect(wrapper.findComponent({ name: 'ElEmpty' }).exists()).toBe(true)
  })

  it('renders a controlled table and pagination for loaded rows', () => {
    const wrapper = shallowMount(AdminDataTable, {
      props: {
        ...baseProps,
        rows: [{ domain: 'example.com' }],
        total: 25,
      },
    })
    expect(wrapper.findComponent({ name: 'ElTable' }).exists()).toBe(true)
    expect(wrapper.findComponent({ name: 'ElPagination' }).exists()).toBe(true)
  })
})
