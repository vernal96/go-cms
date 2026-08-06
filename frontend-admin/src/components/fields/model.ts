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
])

export function unsupportedFieldTypes(fields: FieldDefinition[]): string[] {
  return fields
    .filter((field) => !supportedTypes.has(field.type))
    .map((field) => `${field.key} (${field.type})`)
}

export function createFieldValues(
  fields: FieldDefinition[],
  source: DynamicValues = {},
): DynamicValues {
  const result: DynamicValues = {}
  for (const field of fields) {
    if (Object.hasOwn(source, field.key)) {
      result[field.key] = source[field.key]
    } else if (field.type === 'checkbox') {
      result[field.key] = false
    } else if (field.type === 'select' && field.options?.multiple) {
      result[field.key] = []
    } else if (field.type === 'int' || field.type === 'float') {
      result[field.key] = null
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
    if (!supportedTypes.has(field.type)) {
      errors[field.key] = `Тип поля «${field.type}» не поддерживается.`
      continue
    }

    const value = values[field.key]
    const empty = isEmpty(value)
    if (field.required && empty) {
      errors[field.key] = 'Поле обязательно.'
      continue
    }
    if (empty) continue

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
      typeof value !== 'number'
    ) {
      errors[field.key] = 'Введите число.'
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
