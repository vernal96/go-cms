<script setup lang="ts">
import type { FieldDefinition } from '@go-cms/admin/sdk'
defineOptions({ inheritAttrs: false })
defineProps<{field: FieldDefinition}>()
const model = defineModel<unknown>()
</script>
<template><input :aria-label="field.label" :value="model" @input="model = ($event.target as HTMLInputElement).value" /></template>
