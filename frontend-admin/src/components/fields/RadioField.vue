<script setup lang="ts">
import { ElRadio, ElRadioGroup } from 'element-plus'
import type { FieldChoice } from '../../types/admin'

defineProps<{ choices: FieldChoice[] }>()
const model = defineModel<any>()
</script>

<template>
  <el-radio-group v-model="model">
    <el-radio
      v-for="choice in choices"
      :key="choice.value"
      :value="choice.value"
    >
      {{ choice.label }}
    </el-radio>
  </el-radio-group>
</template>
