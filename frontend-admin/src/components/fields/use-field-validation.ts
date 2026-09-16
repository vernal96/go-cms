import { inject } from 'vue'
import { adminPluginRegistryKey } from '../../admin-plugins/context'
import type { FieldDefinition } from '../../types/admin'
import { unsupportedFieldTypes, validateFieldValues, type DynamicValues } from './model'

// Bind validation to the exact registry used by DynamicField in this app.
export function useFieldValidation() {
  const registry = inject(adminPluginRegistryKey, undefined)
  return {
    unsupportedFieldTypes: (fields: FieldDefinition[]) => unsupportedFieldTypes(fields, registry),
    validateFieldValues: (fields: FieldDefinition[], values: DynamicValues) => validateFieldValues(fields, values, registry),
  }
}
