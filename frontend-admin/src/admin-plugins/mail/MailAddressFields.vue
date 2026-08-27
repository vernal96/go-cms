<script setup lang="ts">
import { ElInput } from 'element-plus'
import type { MailAddressTemplate } from './types'

const props = defineProps<{
  modelValue: MailAddressTemplate
  namePlaceholder?: string
  emailPlaceholder?: string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: MailAddressTemplate] }>()

function update(key: keyof MailAddressTemplate, value: string): void {
  emit('update:modelValue', { ...props.modelValue, [key]: value })
}
</script>

<template>
  <div class="mail-address-fields">
    <el-input
      :model-value="modelValue.name"
      :placeholder="namePlaceholder ?? 'Имя (можно с {{data.*}})'"
      @update:model-value="update('name', $event)"
    />
    <el-input
      :model-value="modelValue.email"
      :placeholder="emailPlaceholder ?? 'email@example.com'"
      @update:model-value="update('email', $event)"
    />
  </div>
</template>

<style scoped>
.mail-address-fields { display: grid; grid-template-columns: minmax(180px, 1fr) minmax(240px, 1.3fr); gap: 10px; width: 100%; }
@media (max-width: 720px) { .mail-address-fields { grid-template-columns: 1fr; } }
</style>
