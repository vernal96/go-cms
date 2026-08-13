import { describe, expect, it } from 'vitest'

import type { FieldDefinition } from '../../types/admin'
import {
  createFieldValues,
  fieldErrorMessage,
  unsupportedFieldTypes,
  validateFieldValues,
} from './model'

const fields: FieldDefinition[] = [
  {
    key: 'text',
    type: 'string',
    label: 'Text',
    required: true,
    rules: ['min=2', 'max=5'],
  },
  {
    key: 'integer',
    type: 'int',
    label: 'Integer',
    required: true,
    rules: ['min=1', 'max=4'],
    options: { step: 1 },
  },
  {
    key: 'float',
    type: 'float',
    label: 'Float',
    required: true,
    rules: [],
    options: { step: 0.1 },
  },
  {
    key: 'checked',
    type: 'checkbox',
    label: 'Checked',
    required: true,
    rules: [],
  },
  {
    key: 'radio',
    type: 'radio',
    label: 'Radio',
    required: true,
    rules: [],
    options: { choices: [{ value: 'a', label: 'A' }] },
  },
  {
    key: 'single',
    type: 'select',
    label: 'Single',
    required: true,
    rules: [],
    options: { choices: [{ value: 'a', label: 'A' }], multiple: false },
  },
  {
    key: 'multiple',
    type: 'select',
    label: 'Multiple',
    required: false,
    rules: [],
    options: { choices: [{ value: 'a', label: 'A' }], multiple: true },
  },
  {
    key: 'area',
    type: 'textarea',
    label: 'Area',
    required: false,
    rules: ['max=5'],
  },
  { key: 'email', type: 'email', label: 'Email', required: false, rules: [] },
  {
    key: 'phone',
    type: 'phone',
    label: 'Phone',
    required: false,
    rules: [],
    options: { pattern: '^\\+[1-9][0-9]{1,14}$' },
  },
  {
    key: 'asset',
    type: 'file',
    label: 'Asset',
    required: true,
    rules: [],
    options: { storages: ['public'], mime_types: ['image/*'] },
  },
]

describe('dynamic field model', () => {
  it('initializes checkbox and multiple select without pre-filling other required values', () => {
    expect(createFieldValues(fields)).toEqual({
      text: '',
      integer: null,
      float: null,
      checked: false,
      radio: '',
      single: '',
      multiple: [],
      area: '',
      email: '',
      phone: '',
      asset: null,
    })
  })

  it('validates required, numeric, text, email and E.164 rules', () => {
    expect(
      validateFieldValues(fields, createFieldValues(fields)),
    ).toMatchObject({
      text: 'Поле обязательно.',
      integer: 'Поле обязательно.',
      float: 'Поле обязательно.',
      radio: 'Поле обязательно.',
      single: 'Поле обязательно.',
      asset: 'Поле обязательно.',
    })
    const errors = validateFieldValues(fields, {
      text: 'x',
      integer: 7,
      float: 1.5,
      checked: false,
      radio: 'a',
      single: 'a',
      multiple: ['a'],
      area: 'too long',
      email: 'bad',
      phone: '8999',
      asset: -1,
    })
    expect(errors.text).toContain('Минимальное')
    expect(errors.integer).toContain('Максимальное')
    expect(errors.area).toContain('Максимальное')
    expect(errors.email).toContain('электронной')
    expect(errors.phone).toContain('E.164')
    expect(errors.asset).toContain('Выберите файл')
  })

  it('blocks unknown types and maps backend validation rules', () => {
    expect(unsupportedFieldTypes([{ ...fields[0]!, type: 'future' }])).toEqual([
      'text (future)',
    ])
    expect(fieldErrorMessage('oneof', '')).toContain('недопустимое')
    expect(fieldErrorMessage('required', '')).toBe('Поле обязательно.')
  })
})
