<script setup lang="ts">
import { Delete, Plus } from '@element-plus/icons-vue'
import { ElButton } from 'element-plus'
import MailAddressFields from './MailAddressFields.vue'
import type { MailAddressTemplate } from './types'

const props = defineProps<{ modelValue: MailAddressTemplate[]; addLabel?: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: MailAddressTemplate[]] }>()

function add(): void {
  emit('update:modelValue', [...props.modelValue, { name: '', email: '' }])
}
function update(index: number, value: MailAddressTemplate): void {
  const result = [...props.modelValue]
  result[index] = value
  emit('update:modelValue', result)
}
function remove(index: number): void {
  emit('update:modelValue', props.modelValue.filter((_, current) => current !== index))
}
</script>

<template>
  <div class="mail-address-list">
    <div v-for="(address, index) in modelValue" :key="index" class="mail-address-row">
      <mail-address-fields :model-value="address" @update:model-value="update(index, $event)" />
      <el-button :icon="Delete" circle plain aria-label="Удалить адрес" @click="remove(index)" />
    </div>
    <el-button plain :icon="Plus" @click="add">{{ addLabel ?? 'Добавить адрес' }}</el-button>
  </div>
</template>

<style scoped>
.mail-address-list { display: grid; gap: 10px; width: 100%; }
.mail-address-row { display: flex; align-items: center; gap: 8px; }
</style>
