<script setup lang="ts">
import { computed } from 'vue'
import { ElInput } from 'element-plus'
const model = defineModel<unknown>()
const text = computed(() => Array.isArray(model.value) ? model.value.join('\n') : '')
function update(value: string): void { model.value = value.split('\n').map(item => item.trim()).filter(Boolean) }
</script>
<template><el-input :model-value="text" type="textarea" :rows="3" placeholder="По одному значению в строке" @update:model-value="update" /></template>
