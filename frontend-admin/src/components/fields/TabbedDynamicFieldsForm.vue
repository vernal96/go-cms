<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElTabPane, ElTabs } from 'element-plus'

import type { FieldDefinition, FieldEditorTab } from '../../types/admin'
import DynamicFieldsForm from './DynamicFieldsForm.vue'
import type { DynamicFieldErrors, DynamicValues } from './model'

const props = withDefaults(
  defineProps<{
    fields: FieldDefinition[]
    editorTabs?: FieldEditorTab[]
    modelValue: DynamicValues
    errors?: DynamicFieldErrors
    siteId?: number
    accessToken?: string
    resourceTemplates?: Array<{ code: string; label: string }>
  }>(),
  {
    editorTabs: () => [],
    errors: () => ({}),
  },
)
const emit = defineEmits<{ 'update:modelValue': [value: DynamicValues] }>()
const activeTab = ref('')
const tabs = computed(() => props.editorTabs)

function selectAvailableTab(): void {
  const codes = tabs.value.map((tab) => tab.code)
  if (!codes.includes(activeTab.value)) activeTab.value = codes[0] ?? ''
}

function fieldsForTab(tab: FieldEditorTab): FieldDefinition[] {
  const selected = new Set(tab.fields)
  return props.fields.filter((field) => selected.has(field.key))
}

function selectFirstErrorTab(): void {
  if (tabs.value.length === 0) return
  const errorFields = new Set(Object.keys(props.errors))
  const tab = tabs.value.find((candidate) =>
    candidate.fields.some((field) => errorFields.has(field)),
  )
  if (tab) activeTab.value = tab.code
}

watch(
  () => props.editorTabs,
  () => {
    activeTab.value = tabs.value[0]?.code ?? ''
  },
  { immediate: true },
)
watch(() => tabs.value.map((tab) => tab.code), selectAvailableTab)
watch(() => props.errors, selectFirstErrorTab, { deep: true })
</script>

<template>
  <el-tabs
    v-if="tabs.length"
    v-model="activeTab"
    class="tabbed-dynamic-fields"
    tab-position="left"
  >
    <el-tab-pane
      v-for="tab in tabs"
      :key="tab.code"
      :label="tab.label"
      :name="tab.code"
    >
      <dynamic-fields-form
        :fields="fieldsForTab(tab)"
        :model-value="modelValue"
        :errors="errors"
        :site-id="siteId"
        :access-token="accessToken"
        :resource-templates="resourceTemplates"
        @update:model-value="emit('update:modelValue', $event)"
      />
    </el-tab-pane>
  </el-tabs>
  <dynamic-fields-form
    v-else
    :fields="fields"
    :model-value="modelValue"
    :errors="errors"
    :site-id="siteId"
    :access-token="accessToken"
    :resource-templates="resourceTemplates"
    @update:model-value="emit('update:modelValue', $event)"
  />
</template>

<style scoped>
.tabbed-dynamic-fields {
  width: 100%;
}

.tabbed-dynamic-fields :deep(.el-tabs__header.is-left) {
  min-width: 140px;
  max-width: 220px;
}

.tabbed-dynamic-fields :deep(.el-tabs__item.is-left) {
  justify-content: flex-start;
  text-align: left;
}

.tabbed-dynamic-fields :deep(.el-tabs__content) {
  min-width: 0;
  padding-left: 20px;
}
</style>
