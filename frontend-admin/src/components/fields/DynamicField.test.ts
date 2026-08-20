// @vitest-environment jsdom

import { shallowMount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'

import type { FieldDefinition } from '../../types/admin'
import DynamicField from './DynamicField.vue'

function definition(type: string): FieldDefinition {
  return {
    key: type,
    type,
    label: type,
    required: false,
    rules: [],
    options: {
      step: 0.1,
      choices: [{ value: 'one', label: 'One' }],
      multiple: type === 'select',
    },
  }
}

describe('DynamicField', () => {
  it.each([
    ['string', 'TextField'],
    ['email', 'TextField'],
    ['phone', 'TextField'],
    ['int', 'NumberField'],
    ['float', 'NumberField'],
    ['checkbox', 'CheckboxField'],
    ['radio', 'RadioField'],
    ['select', 'SelectField'],
    ['textarea', 'TextareaField'],
  ])(
    'dispatches %s to %s and preserves v-model values',
    async (type, component) => {
      const wrapper = shallowMount(DynamicField, {
        props: {
          field: definition(type),
          modelValue: type === 'select' ? [] : '',
			siteId: 0,
			accessToken: '',
        },
      })
      const rendered = wrapper.findComponent({ name: component })
      expect(rendered.exists()).toBe(true)
      const nextValue =
        type === 'int'
          ? 3
          : type === 'float'
            ? 1.2
            : type === 'select'
              ? ['one']
              : 'one'
      rendered.vm.$emit('update:modelValue', nextValue)
      expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([nextValue])
    },
  )

  it('renders an explicit error instead of a fallback input', () => {
    const wrapper = shallowMount(DynamicField, {
		props: { field: definition('future'), siteId: 0, accessToken: '' },
    })
    expect(wrapper.findComponent({ name: 'ElAlert' }).exists()).toBe(true)
    expect(wrapper.findComponent({ name: 'ElInput' }).exists()).toBe(false)
  })
})
