// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { defineComponent, h, ref } from 'vue'
import { describe, expect, it } from 'vitest'
import { ElFormItem } from 'element-plus'
import type { FieldDefinition } from '../../types/admin'
import { AdminPluginRegistry } from '../../admin-plugins/registry'
import { adminPluginRegistryKey } from '../../admin-plugins/context'
import { adminAccessTokenKey } from '../../admin-context'
import DynamicField from './DynamicField.vue'
import RepeaterField from './RepeaterField.vue'
import { createFieldValues } from './model'

const title: FieldDefinition = { key: 'title', type: 'string', label: 'Заголовок', required: true, rules: [] }
const active: FieldDefinition = { key: 'active', type: 'checkbox', label: 'Активен', required: false, rules: [] }
const field: FieldDefinition = { key: 'slides', type: 'repeater', label: 'Слайды', required: false, rules: [], options: { fields: [title, active], min_items: 1, max_items: 3 } }
function setup(value: unknown = [{ title: 'One' }, { title: 'Two' }], definition = field) {
  const model = ref(value)
  const wrapper = mount(defineComponent({ setup: () => () => h(DynamicField, {
    field: definition, modelValue: model.value, siteId: 7, accessToken: 'token', resourceTemplates: [{ code: 'page', label: 'Page' }],
    'onUpdate:modelValue': (value: unknown) => { model.value = value },
  }) }))
  return { wrapper, model }
}
describe('RepeaterField', () => {
  it('renders ordered rows using DynamicField and passes context', () => {
    const { wrapper } = setup()
    expect(wrapper.findAll('.repeater-row').map(row => row.find('strong').text())).toEqual(['Строка 1', 'Строка 2'])
    const nested = wrapper.findComponent(RepeaterField).findAllComponents(DynamicField)
    expect(nested).toHaveLength(4)
    expect(nested[0]!.props()).toMatchObject({ field: title, siteId: 7, accessToken: 'token', resourceTemplates: [{ code: 'page', label: 'Page' }] })
    wrapper.unmount()
  })
  it('adds with shared defaults, edits without mutating input and observes MaxItems', async () => {
    const input = [{ title: 'One' }, { title: 'Two' }]
    const { wrapper, model } = setup(input)
    const add = () => wrapper.findAll('button').find(button => button.text() === 'Добавить')!
    await add().trigger('click')
    expect(model.value).toEqual([...input, createFieldValues(field.options!.fields!)])
    expect(add().attributes('disabled')).toBeDefined()
    await add().trigger('click')
    expect(model.value).toHaveLength(3)
    const nested = wrapper.findComponent(RepeaterField).findAllComponents(DynamicField)
    nested[0]!.vm.$emit('update:modelValue', 'Edited')
    expect((model.value as Array<Record<string, unknown>>)[0]).toEqual({ title: 'Edited' })
    expect(input[0]!.title).toBe('One')
    wrapper.unmount()
  })
  it('reorders rows, preserves editor identity and enforces MinItems on delete', async () => {
    const { wrapper, model } = setup()
    const firstInput = wrapper.find('input').element
    await wrapper.get('[aria-label="Переместить строку 1 вниз"]').trigger('click')
    expect(model.value).toEqual([{ title: 'Two' }, { title: 'One' }])
    expect(wrapper.findAll('.repeater-row')[1]!.find('input').element).toBe(firstInput)
    await wrapper.get('[aria-label="Переместить строку 2 вверх"]').trigger('click')
    expect(model.value).toEqual([{ title: 'One' }, { title: 'Two' }])
    await wrapper.get('[aria-label="Удалить строку 1"]').trigger('click')
    expect(model.value).toEqual([{ title: 'Two' }])
    expect(wrapper.get('[aria-label="Удалить строку 1"]').attributes('disabled')).toBeDefined()
    await wrapper.get('[aria-label="Удалить строку 1"]').trigger('click')
    expect(model.value).toHaveLength(1)
    expect(wrapper.get('[aria-label="Переместить строку 1 вверх"]').attributes('disabled')).toBeDefined()
    expect(wrapper.get('[aria-label="Переместить строку 1 вниз"]').attributes('disabled')).toBeDefined()
    wrapper.unmount()
  })
  it('starts empty and emits a plain array of objects', async () => {
    const { wrapper, model } = setup(undefined, { ...field, options: { ...field.options, min_items: 0 } })
    // Explicit undefined argument uses setup defaults; replace the model to model an omitted field.
    model.value = undefined
    await wrapper.vm.$nextTick()
    expect(wrapper.findAll('.repeater-row')).toHaveLength(0)
    await wrapper.findAll('button').find(button => button.text() === 'Добавить')!.trigger('click')
    expect(JSON.stringify(model.value)).toBe('[{"title":"","active":false}]')
    wrapper.unmount()
  })
  it('renders a contributed editor and forwards nested backend errors', () => {
    const editor = defineComponent({ props: ['modelValue', 'siteId', 'accessToken', 'resourceTemplates'], emits: ['update:modelValue'], setup: () => () => h('span', 'Custom editor') })
    const custom = { ...title, type: 'example.custom', editor: 'example.custom' }
    const registry = new AdminPluginRegistry([{ code: 'example', fieldEditors: { 'example.custom': editor } }])
    const wrapper = mount(DynamicField, { props: { field: { ...field, options: { fields: [custom] } }, modelValue: [{ title: 'Custom' }], siteId: 9, resourceTemplates: [{ code: 'page', label: 'Page' }], errors: { 'slides[0].title': 'Ошибка вложенного поля' } }, global: { provide: { [adminPluginRegistryKey as symbol]: registry, [adminAccessTokenKey as symbol]: ref('injected-token') } } })
    const rendered = wrapper.findComponent(editor)
    expect(rendered.exists()).toBe(true)
    expect(rendered.props()).toMatchObject({ modelValue: 'Custom', siteId: 9, accessToken: 'injected-token', resourceTemplates: [{ code: 'page', label: 'Page' }] })
    expect(wrapper.findComponent(ElFormItem).props('error')).toBe('Ошибка вложенного поля')
    rendered.vm.$emit('update:modelValue', 'Updated')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([[{ title: 'Updated' }]])
    wrapper.unmount()
  })
})
