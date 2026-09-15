// @vitest-environment jsdom
import { mount } from '@vue/test-utils'
import { expect, it } from 'vitest'
import DynamicFieldsForm from './DynamicFieldsForm.vue'

it('shows privacy only for site metadata and keeps both values editable', () => {
  const wrapper = mount(DynamicFieldsForm, {
    props: {
      fields: [
        { key: 'title', type: 'string', label: 'Название', required: false, rules: [], public: true },
        { key: 'secret', type: 'string', label: 'Секрет', required: false, rules: [], public: false },
        { key: 'ordinary', type: 'string', label: 'Обычное поле', required: false, rules: [] },
      ],
      modelValue: { title: 'Студия', secret: 'hidden', ordinary: 'value' },
    },
    global: { stubs: { DynamicField: { props: ['modelValue'], template: '<input :value="modelValue">' } } },
  })
  expect(wrapper.findAll('.el-tag').map(tag => tag.text())).toEqual(['Публичный', 'Приватный'])
  expect(wrapper.findAll('input').map(input => input.element.value)).toEqual(['Студия', 'hidden', 'value'])
})
