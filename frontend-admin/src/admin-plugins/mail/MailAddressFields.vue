<script setup lang="ts">
import { ElInput } from 'element-plus'
import type { MailAddressTemplate } from './types'

const props = defineProps<{
  modelValue: MailAddressTemplate
  sender?: boolean
  emailRequired?: boolean
  error?: string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: MailAddressTemplate] }>()

function update(key: keyof MailAddressTemplate, value: string): void {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}
</script>

<template>
  <div class="mail-address-fields">
    <label class="mail-address-field">
      <span>{{ sender ? 'Имя отправителя (необязательно)' : 'Имя (необязательно)' }}</span>
      <el-input
        :model-value="modelValue.name"
        :aria-label="sender ? 'Имя отправителя (необязательно)' : 'Имя (необязательно)'"
        @update:model-value="update('name', $event)"
      />
      <small v-pre>Допустимы {{site.*}} и {{data.*}}</small>
    </label>
    <label class="mail-address-field" :class="{ 'has-error': error }">
      <span>{{ sender ? 'Email отправителя' : 'Email' }} <b v-if="emailRequired" aria-hidden="true">*</b></span>
      <el-input
        :model-value="modelValue.email"
        :aria-label="sender ? 'Email отправителя' : 'Email'"
        :aria-invalid="Boolean(error)"
        placeholder="email@example.com"
        @update:model-value="update('email', $event)"
      />
      <small v-pre>Допустимы {{site.*}} и {{data.*}}</small>
      <small v-if="error" class="mail-address-error">{{ error }}</small>
    </label>
  </div>
</template>

<style scoped>
.mail-address-fields { display: grid; grid-template-columns: minmax(180px, 1fr) minmax(240px, 1.3fr); gap: 10px; width: 100%; }
.mail-address-field { display: grid; align-content: start; gap: 6px; min-width: 0; color: var(--el-text-color-primary); }
.mail-address-field > span { font-size: 14px; line-height: 1.35; }
.mail-address-field small { color: var(--el-text-color-secondary); font-weight: 400; }
.mail-address-field b, .mail-address-error { color: var(--el-color-danger) !important; }
.mail-address-field.has-error :deep(.el-input__wrapper) { box-shadow: 0 0 0 1px var(--el-color-danger) inset; }
@media (max-width: 720px) { .mail-address-fields { grid-template-columns: 1fr; } }
</style>
