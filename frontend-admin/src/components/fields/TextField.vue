<script setup lang="ts">
import { computed } from 'vue'
import { ElInput } from 'element-plus'

const props = defineProps<{ kind: 'string' | 'email' | 'phone' }>()
const model = defineModel<any>()
const inputType = computed(() => (props.kind === 'email' ? 'email' : 'text'))
</script>

<template>
  <el-input
    v-model="model"
    :type="inputType"
    :placeholder="kind === 'phone' ? '+79991234567' : undefined"
  />
</template>
