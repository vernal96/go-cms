<script setup lang="ts">
import { ElInputNumber } from 'element-plus'

defineProps<{ kind: 'int' | 'float'; step?: number }>()
const model = defineModel<any>()
</script>

<template>
  <el-input-number
    v-model="model"
    class="full-width"
    :step="step ?? 1"
    :precision="kind === 'int' ? 0 : undefined"
    controls-position="right"
  />
</template>
