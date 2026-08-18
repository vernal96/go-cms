<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import {
  ElAlert,
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElOption,
  ElRadioButton,
  ElRadioGroup,
  ElSelect,
  ElSlider,
  ElSwitch,
  ElTabPane,
  ElTabs,
} from 'element-plus'
import DynamicFieldsForm from '../fields/DynamicFieldsForm.vue'
import { createFieldValues, unsupportedFieldTypes, validateFieldValues, type DynamicFieldErrors } from '../fields/model'
import type { ResourceWidget, WidgetDefinition } from '../../types/admin'
import type { WidgetSettingsValue } from './model'

const props = defineProps<{
  modelValue: boolean
  definition: WidgetDefinition | null
  widget: ResourceWidget | null
  saving?: boolean
}>()
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  save: [value: WidgetSettingsValue]
}>()

const form = reactive<WidgetSettingsValue>({
  view: 'default', columns: 12, margin_top: 0, margin_bottom: 0,
  enabled: true, params: {},
})
const errors = ref<DynamicFieldErrors>({})
const unsupported = computed(() => unsupportedFieldTypes(props.definition?.fields ?? []))
const tabs = computed(() => props.definition?.editor_tabs ?? [])

watch(() => [props.modelValue, props.definition, props.widget] as const, ([open]) => {
  if (!open || !props.definition) return
  const widget = props.widget
  Object.assign(form, {
    view: widget?.view ?? 'default',
    columns: widget?.columns ?? 12,
    margin_top: widget?.margin_top ?? 0,
    margin_bottom: widget?.margin_bottom ?? 0,
    enabled: widget?.enabled ?? true,
    params: createFieldValues(props.definition.fields, widget?.params ?? {}),
  })
  errors.value = {}
})

function fieldsForTab(codes: string[]) {
  const selected = new Set(codes)
  return props.definition?.fields.filter((field) => selected.has(field.key)) ?? []
}

function save(): void {
  if (!props.definition || unsupported.value.length) return
  errors.value = validateFieldValues(props.definition.fields, form.params)
  if (Object.keys(errors.value).length) return
  emit('save', {
    view: form.view,
    columns: form.columns,
    margin_top: form.margin_top,
    margin_bottom: form.margin_bottom,
    enabled: form.enabled,
    params: { ...form.params },
  })
}
</script>

<template>
  <el-dialog
    :model-value="modelValue"
    :title="widget ? `Настройки: ${definition?.label ?? ''}` : `Новый виджет: ${definition?.label ?? ''}`"
    width="680px"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <el-alert v-if="unsupported.length" type="error" :closable="false" :title="`Неизвестные типы полей: ${unsupported.join(', ')}`" />
    <el-form v-if="definition" label-position="top">
      <div class="widget-presentation-grid">
        <el-form-item label="Вид">
          <el-select v-model="form.view" class="full-width">
            <el-option label="Default" value="default" />
            <el-option v-for="view in definition.views" :key="view.code" :label="view.label" :value="view.code" />
          </el-select>
        </el-form-item>
        <el-form-item label="Ширина (колонки)">
          <el-slider v-model="form.columns" :min="1" :max="12" :step="1" show-stops show-input />
        </el-form-item>
        <el-form-item label="Отступ сверху">
          <el-radio-group v-model="form.margin_top">
            <el-radio-button v-for="value in [0, 1, 2, 3]" :key="value" :value="value">{{ value }}</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="Отступ снизу">
          <el-radio-group v-model="form.margin_bottom">
            <el-radio-button v-for="value in [0, 1, 2, 3]" :key="value" :value="value">{{ value }}</el-radio-button>
          </el-radio-group>
        </el-form-item>
      </div>
      <el-form-item label="Включён"><el-switch v-model="form.enabled" /></el-form-item>

      <el-tabs v-if="tabs.length">
        <el-tab-pane v-for="tab in tabs" :key="tab.code" :label="tab.label" :name="tab.code">
          <dynamic-fields-form v-model="form.params" :fields="fieldsForTab(tab.fields)" :errors="errors" />
        </el-tab-pane>
      </el-tabs>
      <dynamic-fields-form v-else v-model="form.params" :fields="definition.fields" :errors="errors" />
    </el-form>
    <template #footer>
      <el-button @click="emit('update:modelValue', false)">Отмена</el-button>
      <el-button type="primary" :loading="saving" :disabled="unsupported.length > 0" @click="save">Сохранить</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.widget-presentation-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 0 20px; }
.full-width { width: 100%; }
@media (max-width: 650px) { .widget-presentation-grid { grid-template-columns: 1fr; } }
</style>

