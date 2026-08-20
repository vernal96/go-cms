<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { ElInput } from 'element-plus'

const model = defineModel<unknown>()
const source = ref('[]')
const invalid = ref(false)

watch(model, (value) => {
  source.value = JSON.stringify(value ?? [], null, 2)
  invalid.value = false
}, { immediate: true, deep: true })

const help = computed(() => invalid.value ? 'Некорректный JSON.' : 'JSON-массив или объект')
function update(value: string): void {
  source.value = value
  try {
    const parsed: unknown = JSON.parse(value)
    if (Array.isArray(parsed) || (typeof parsed === 'object' && parsed !== null)) {
      invalid.value = false
      model.value = parsed
      return
    }
  } catch { /* shown below */ }
  invalid.value = true
}
</script>

<template>
  <el-input :model-value="source" type="textarea" :rows="7" @update:model-value="update" />
  <small :class="{ error: invalid }">{{ help }}</small>
</template>

<style scoped>
small { color: var(--el-text-color-secondary); }
small.error { color: var(--el-color-danger); }
</style>
