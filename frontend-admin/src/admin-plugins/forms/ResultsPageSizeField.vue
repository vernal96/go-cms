<script setup lang="ts">
import { watch } from 'vue'
import { ElInputNumber } from 'element-plus'
const model = defineModel<number | null>()
watch(model, (value) => { if (value == null) model.value = 20 }, { immediate: true })
</script>

<template>
  <el-input-number v-model="model" :min="1" :max="100" :step="1" :precision="0" />
</template>
