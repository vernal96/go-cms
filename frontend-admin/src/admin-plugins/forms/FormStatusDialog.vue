<script setup lang="ts">
import { reactive, watch } from 'vue'
import { ElButton, ElColorPicker, ElDialog, ElForm, ElFormItem, ElInput, ElInputNumber, ElSwitch } from 'element-plus'
import type { FormStatus } from './types'

const props = defineProps<{ modelValue: boolean; status?: FormStatus | null; nextPosition: number }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; save: [payload: Pick<FormStatus, 'code' | 'name' | 'color' | 'position' | 'is_default'>] }>()
const state = reactive({ code: '', name: '', color: '#409eff', position: 0, is_default: false })
watch(() => [props.modelValue, props.status] as const, ([open]) => { if (open) Object.assign(state, props.status ? { code: props.status.code, name: props.status.name, color: props.status.color, position: props.status.position, is_default: props.status.is_default } : { code: '', name: '', color: '#409eff', position: props.nextPosition, is_default: false }) }, { deep: true })
function save(): void { emit('save', { code: state.code.trim(), name: state.name.trim(), color: state.color, position: state.position, is_default: state.is_default }) }
</script>
<template><el-dialog :model-value="modelValue" :title="status ? 'Статус' : 'Новый статус'" width="min(520px, 94vw)" @update:model-value="emit('update:modelValue', $event)"><el-form label-position="top" @submit.prevent="save"><el-form-item label="Название" required><el-input v-model="state.name" /></el-form-item><el-form-item label="Код" required><el-input v-model="state.code" /></el-form-item><el-form-item label="Цвет"><el-color-picker v-model="state.color" /></el-form-item><el-form-item label="Позиция"><el-input-number v-model="state.position" :min="0" /></el-form-item><el-form-item label="Статус по умолчанию"><el-switch v-model="state.is_default" :disabled="status?.is_default" /></el-form-item></el-form><template #footer><el-button @click="emit('update:modelValue', false)">Отмена</el-button><el-button type="primary" @click="save">Сохранить</el-button></template></el-dialog></template>
