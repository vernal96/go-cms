<script setup lang="ts">
import { ElCheckbox } from 'element-plus'

defineProps<{ label: string }>()
const model = defineModel<any>()
</script>

<template>
  <el-checkbox v-model="model">{{ label }}</el-checkbox>
</template>
