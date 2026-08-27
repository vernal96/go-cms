<script setup lang="ts">
import { Delete, Plus } from '@element-plus/icons-vue'
import { ElButton, ElCheckbox, ElInput, ElOption, ElSelect } from 'element-plus'
import type { FieldChoice, FieldDefinition, FieldOptions } from '../../types/admin'

const props = defineProps<{ modelValue: FieldDefinition[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: FieldDefinition[]] }>()

const types = [
  ['string', 'Строка'], ['email', 'Email'], ['textarea', 'Многострочный текст'],
  ['phone', 'Телефон'], ['int', 'Целое число'], ['float', 'Число'],
  ['checkbox', 'Флаг'], ['radio', 'Один вариант'], ['select', 'Список'], ['file', 'Файл'],
] as const

function add(): void {
  emit('update:modelValue', [...props.modelValue, { key: '', type: 'string', label: '', required: false, rules: [] }])
}
function remove(index: number): void {
  emit('update:modelValue', props.modelValue.filter((_, current) => current !== index))
}
function update(index: number, patch: Partial<FieldDefinition>): void {
  const result = [...props.modelValue]
  const current = { ...result[index], ...patch, required: false }
  if (patch.type !== undefined) current.options = defaultOptions(String(patch.type))
  result[index] = current
  emit('update:modelValue', result)
}
function updateOptions(index: number, patch: Partial<FieldOptions>): void {
  update(index, { options: { ...(props.modelValue[index]?.options ?? {}), ...patch } })
}
function defaultOptions(type: string): FieldOptions | undefined {
  if (type === 'radio') return { choices: [] }
  if (type === 'select') return { choices: [], multiple: false }
  if (type === 'file') return { storages: [], mime_types: [] }
  if (type === 'int' || type === 'float' || type === 'phone') return {}
  return undefined
}
function choicesText(choices: FieldChoice[] | undefined): string {
  return (choices ?? []).map((choice) => `${choice.value}|${choice.label}`).join('\n')
}
function parseChoices(source: string): FieldChoice[] {
  return source.split(/\r?\n/).map((line) => line.trim()).filter(Boolean).map((line) => {
    const separator = line.indexOf('|')
    return separator < 0
      ? { value: line, label: line }
      : { value: line.slice(0, separator).trim(), label: line.slice(separator + 1).trim() }
  }).filter((choice) => choice.value !== '' && choice.label !== '')
}
function csv(value: string): string[] {
  return value.split(',').map((item) => item.trim()).filter(Boolean)
}
</script>

<template>
  <div class="mail-variables-editor">
    <article v-for="(variable, index) in modelValue" :key="index" class="mail-variable-card">
      <div class="mail-variable-main">
        <el-input :model-value="variable.key" placeholder="name" @update:model-value="update(index, { key: $event })">
          <template #prepend>data.</template>
        </el-input>
        <el-input :model-value="variable.label" placeholder="Название поля" @update:model-value="update(index, { label: $event })" />
        <el-select :model-value="variable.type" @update:model-value="update(index, { type: $event })">
          <el-option v-for="item in types" :key="item[0]" :value="item[0]" :label="item[1]" />
        </el-select>
        <el-button :icon="Delete" circle plain aria-label="Удалить переменную" @click="remove(index)" />
      </div>
      <el-input
        :model-value="variable.rules.join(', ')"
        placeholder="Правила через запятую, например min=2, max=100"
        @update:model-value="update(index, { rules: csv($event) })"
      />
      <el-input
        v-if="variable.type === 'radio' || variable.type === 'select'"
        type="textarea"
        :rows="3"
        :model-value="choicesText(variable.options?.choices)"
        placeholder="Варианты: значение|Подпись, по одному в строке"
        @update:model-value="updateOptions(index, { choices: parseChoices($event) })"
      />
      <el-checkbox
        v-if="variable.type === 'select'"
        :model-value="Boolean(variable.options?.multiple)"
        @update:model-value="updateOptions(index, { multiple: Boolean($event) })"
      >Множественный выбор</el-checkbox>
      <div v-if="variable.type === 'file'" class="mail-variable-file-options">
        <el-input
          :model-value="(variable.options?.storages ?? []).join(', ')"
          placeholder="Допустимые хранилища через запятую (необязательно)"
          @update:model-value="updateOptions(index, { storages: csv($event) })"
        />
        <el-input
          :model-value="(variable.options?.mime_types ?? []).join(', ')"
          placeholder="MIME-типы через запятую, например application/pdf,image/*"
          @update:model-value="updateOptions(index, { mime_types: csv($event) })"
        />
      </div>
    </article>
    <el-button plain :icon="Plus" @click="add">Добавить переменную</el-button>
  </div>
</template>

<style scoped>
.mail-variables-editor { display: grid; gap: 12px; }
.mail-variable-card { display: grid; gap: 10px; padding: 12px; border: 1px solid var(--el-border-color); border-radius: 8px; }
.mail-variable-main { display: grid; grid-template-columns: 1fr 1fr 190px auto; gap: 8px; }
.mail-variable-file-options { display: grid; grid-template-columns: 1fr 1fr; gap: 8px; }
@media (max-width: 900px) { .mail-variable-main, .mail-variable-file-options { grid-template-columns: 1fr; } }
</style>
