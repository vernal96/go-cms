<script setup lang="ts">
import { ElOption, ElSelect } from 'element-plus'
import type { FieldChoice } from '../../types/admin'

defineProps<{ choices: FieldChoice[]; multiple: boolean }>()
const model = defineModel<any>()
</script>

<template>
  <el-select v-model="model" class="full-width" :multiple="multiple" clearable>
    <el-option
      v-for="choice in choices"
      :key="choice.value"
      :label="choice.label"
      :value="choice.value"
    />
  </el-select>
</template>
