// @vitest-environment jsdom
import { shallowMount } from '@vue/test-utils'
import { expect, it } from 'vitest'
import WidgetParamFields from './WidgetParamFields.vue'

it('offers only matching semantic types and cardinality and clears the literal when binding', async () => {
  const wrapper = shallowMount(WidgetParamFields, {
    props: {
      fields: [{ key: 'title', label: 'Title', type: 'string', required: true, rules: [] }],
      modelValue: { title: 'Manual' }, bindings: {}, errors: {}, siteId: 7, accessToken: 'token',
      paramTypes: { title: { type: 'string', multiple: false } },
      sources: [
        { kind: 'resource_property', key: 'title', label: 'Название', type: 'string', multiple: false },
        { kind: 'resource_property', key: 'content', label: 'Контент', type: 'textarea', multiple: false },
        { kind: 'resource_field', key: 'tags', label: 'Метки', type: 'string', multiple: true },
      ],
    }, global: { renderStubDefaultSlot: true },
  })
  wrapper.getComponent({ name: 'ElRadioGroup' }).vm.$emit('update:modelValue', 'source')
  await wrapper.vm.$nextTick()
  expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([{}])
  await wrapper.setProps({ modelValue: {}, bindings: { title: { kind: 'resource_field', key: '' } } })
  expect(wrapper.findAllComponents({ name: 'ElOption' }).map((item) => item.props('value'))).toEqual(['resource_property:title'])
  wrapper.getComponent({ name: 'ElSelect' }).vm.$emit('update:modelValue', 'resource_property:title')
  expect(wrapper.emitted('update:bindings')?.at(-1)).toEqual([{ title: { kind: 'resource_property', key: 'title' } }])
})

it('keeps dependent editors available when the visibility controller is resolved at render time', () => {
  const wrapper = shallowMount(WidgetParamFields, {
    props: {
      fields: [{ key: 'detail', label: 'Detail', type: 'string', required: false, rules: [], visible_when: { field: 'enabled', value: true } }],
      modelValue: {}, bindings: { enabled: { kind: 'resource_field', key: 'enabled' } },
      errors: {}, siteId: 7, accessToken: 'token', paramTypes: {}, sources: [],
    }, global: { renderStubDefaultSlot: true },
  })
  const editor = wrapper.getComponent({ name: 'DynamicFieldsForm' })
  expect(editor.props('fields')[0].key).toBe('detail')
  expect(editor.props('fields')[0].visible_when).toBeUndefined()
})
