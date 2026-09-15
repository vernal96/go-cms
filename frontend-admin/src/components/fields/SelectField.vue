<script setup lang="ts">
import { ElOption, ElSelect } from 'element-plus'
import { computed } from 'vue'
import type { DynamicFieldErrors } from './model'
import type { FieldChoice } from '../../types/admin'

const props = withDefaults(defineProps<{ choices: FieldChoice[]; multiple: boolean; minItems?: number; maxItems?: number; errors?: DynamicFieldErrors; fieldPath?: string }>(), { minItems: 0, maxItems: 0 })
const itemErrors = computed(() => Object.entries(props.errors ?? {}).filter(([key]) => props.fieldPath && key.startsWith(`${props.fieldPath}[`)))
const model = defineModel<any>()
</script>

<template>
  <el-select v-model="model" class="full-width" :multiple="multiple" :multiple-limit="maxItems" clearable>
    <el-option
      v-for="choice in choices"
      :key="choice.value"
      :label="choice.label"
      :value="choice.value"
    />
  </el-select>
  <div v-if="multiple && (minItems || maxItems)" class="select-limits">Минимум: {{ minItems }}<template v-if="maxItems">, максимум: {{ maxItems }}</template>.</div>
  <div v-for="[key, error] in itemErrors" :key="key" class="el-form-item__error" style="position:static">{{ error }}</div>
</template>
