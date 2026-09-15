import type { FieldDefinition } from '../../types/admin'

export type DynamicValues = Record<string, unknown>
export type DynamicFieldErrors = Record<string, string>

const supportedTypes = new Set([
  'string',
  'int',
  'float',
  'checkbox',
  'radio',
  'select',
  'textarea',
  'email',
  'phone',
  'file',
  'media',
  'repeater',
	'json',
])

const multipleTypes = new Set(['string', 'textarea', 'email', 'phone', 'int', 'float', 'file', 'media', 'select'])

export function isMultipleField(field: FieldDefinition): boolean {
  return multipleTypes.has(field.type) && field.options?.multiple === true
}

export function singleValueField(field: FieldDefinition): FieldDefinition {
  return { ...field, required: true, options: { ...field.options, multiple: false, min_items: 0, max_items: 0 } }
}

export function unsupportedFieldTypes(fields: FieldDefinition[]): string[] {
  return fields
    .filter((field) => !supportedTypes.has(field.type) && !field.editor)
    .map((field) => `${field.key} (${field.type})`)
}

export function createFieldValues(
  fields: FieldDefinition[],
  source: DynamicValues = {},
): DynamicValues {
  const result: DynamicValues = {}
  for (const field of fields) {
    const type = supportedTypes.has(field.editor ?? '') ? field.editor : field.type
    if (Object.hasOwn(source, field.key)) {
      result[field.key] = source[field.key]
    } else if (isMultipleField(field)) {
      result[field.key] = []
    } else if (type === 'checkbox') {
      result[field.key] = false
    } else if (type === 'select' && field.options?.multiple) {
      result[field.key] = []
    } else if (type === 'int' || type === 'float') {
      result[field.key] = null
    } else if (type === 'file' || type === 'media') {
      result[field.key] = null
		} else if (type === 'json' || type === 'repeater') {
			result[field.key] = []
    } else {
      result[field.key] = ''
    }
  }
  return result
}

export function validateFieldValues(
  fields: FieldDefinition[],
  values: DynamicValues,
): DynamicFieldErrors {
  const errors: DynamicFieldErrors = {}
  for (const field of fields) {
    if (!supportedTypes.has(field.type) && !field.editor) {
      errors[field.key] = `Тип поля «${field.type}» не поддерживается.`
      continue
    }

    const value = values[field.key]
    if (isMultipleField(field)) {
      const minimum = Math.max(field.required ? 1 : 0, field.options?.min_items ?? 0)
      const maximum = field.options?.max_items ?? 0
      if (value !== null && value !== undefined && !Array.isArray(value)) {
        errors[field.key] = 'Ожидается список значений.'
        continue
      }
      const items: unknown[] = Array.isArray(value) ? value : []
      if (items.length < minimum) errors[field.key] = minimum === 1 && field.required ? 'Поле обязательно.' : fieldErrorMessage('min_items', String(minimum))
      if (maximum && items.length > maximum) errors[field.key] = fieldErrorMessage('max_items', String(maximum))
      const seen = new Set<unknown>()
      for (const [index, item] of items.entries()) {
        const key = `${field.key}[${index}]`
        Object.assign(errors, validateFieldValues([{ ...singleValueField(field), key }], { [key]: item }))
        if (field.type === 'select' && seen.has(item)) errors[key] = fieldErrorMessage('unique', '')
        seen.add(item)
      }
      continue
    }
    if (field.type === 'repeater' && Array.isArray(value)) {
      for (const [index, row] of value.entries()) {
        if (!row || typeof row !== 'object' || Array.isArray(row)) {
          errors[`${field.key}[${index}]`] = 'Ожидается запись.'
          continue
        }
        for (const [key, error] of Object.entries(validateFieldValues(field.options?.fields ?? [], row as DynamicValues))) errors[`${field.key}[${index}].${key}`] = error
      }
    }
    const empty = isEmpty(value)
    if (field.required && empty) {
      errors[field.key] = 'Поле обязательно.'
      continue
    }
    if (empty) continue

    if ((field.type === 'select' || field.type === 'radio') && !field.options?.choices?.some(choice => choice.value === value)) {
      errors[field.key] = fieldErrorMessage('oneof', '')
      continue
    }
    if (
      field.type === 'email' &&
      !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(String(value))
    ) {
      errors[field.key] = 'Введите корректный адрес электронной почты.'
      continue
    }
    if (field.type === 'phone' && field.options?.pattern) {
      try {
        if (!new RegExp(field.options.pattern).test(String(value))) {
          errors[field.key] =
            'Введите телефон в формате E.164, например +79991234567.'
          continue
        }
      } catch {
        errors[field.key] = 'Backend передал некорректный шаблон телефона.'
        continue
      }
    }
    if (
      (field.type === 'int' || field.type === 'float') &&
      (typeof value !== 'number' || !Number.isFinite(value) || (field.type === 'int' && !Number.isInteger(value)))
    ) {
      errors[field.key] = 'Введите число.'
      continue
    }
    if ((field.type === 'file' || field.type === 'media') && (typeof value !== 'number' || !Number.isInteger(value) || value <= 0)) {
      errors[field.key] = field.type === 'media' ? 'Выберите изображение.' : 'Выберите файл.'
      continue
    }
		if (field.type === 'json' && !Array.isArray(value) && (typeof value !== 'object' || value === null)) {
			errors[field.key] = 'Введите JSON-массив или объект.'
			continue
		}

    for (const rule of field.rules) {
      const [name, param = ''] = rule.split('=', 2)
      const limit = Number(param)
      if (name === 'min' && violatesMin(value, limit)) {
        errors[field.key] = `Минимальное значение: ${param}.`
        break
      }
      if (name === 'max' && violatesMax(value, limit)) {
        errors[field.key] = `Максимальное значение: ${param}.`
        break
      }
    }
  }
  return errors
}

export function fieldErrorMessage(rule: string, param: string): string {
  switch (rule) {
    case 'required':
      return 'Поле обязательно.'
    case 'defined':
      return 'Поле отсутствует в актуальной схеме.'
    case 'type':
      return 'Значение имеет неверный тип.'
    case 'email':
      return 'Введите корректный адрес электронной почты.'
    case 'e164':
      return 'Введите телефон в формате E.164, например +79991234567.'
    case 'pattern':
      return 'Значение не соответствует требуемому формату.'
    case 'oneof':
      return 'Выбрано недопустимое значение.'
    case 'min_items':
      return `Минимум значений: ${param}.`
    case 'max_items':
      return `Максимум значений: ${param}.`
    case 'unique':
      return 'Значение уже выбрано.'
    case 'min':
      return `Минимальное значение: ${param}.`
    case 'max':
      return `Максимальное значение: ${param}.`
    default:
      return `Значение не прошло проверку «${rule}»${param ? ` (${param})` : ''}.`
  }
}

function isEmpty(value: unknown): boolean {
  return (
    value === null ||
    value === undefined ||
    value === '' ||
    (Array.isArray(value) && value.length === 0)
  )
}

function violatesMin(value: unknown, limit: number): boolean {
  if (!Number.isFinite(limit)) return false
  return typeof value === 'number'
    ? value < limit
    : String(value).length < limit
}

function violatesMax(value: unknown, limit: number): boolean {
  if (!Number.isFinite(limit)) return false
  return typeof value === 'number'
    ? value > limit
    : String(value).length > limit
}
