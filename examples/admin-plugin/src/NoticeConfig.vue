<script setup lang="ts">
import { computed, ref } from 'vue'
import { DynamicFieldsForm, useFieldValidation, type ConfigField } from '@go-cms/admin/sdk'
const props = defineProps<{fields: ConfigField[];siteId?:number;accessToken?:string}>()
const model = defineModel<Record<string,unknown>>({required:true})
const definitions = computed(()=>props.fields.map(field=>({...field,rules:field.rules??[]})))
const errors=ref<Record<string,string>>({})
const { validateFieldValues }=useFieldValidation()
function validate(){errors.value=validateFieldValues(definitions.value,model.value);if(Object.keys(errors.value).length)throw new Error('Проверьте поля уведомления.')}
defineExpose({validate})
</script>
<template><DynamicFieldsForm v-model="model" :fields="definitions" :errors="errors" :site-id="siteId" :access-token="accessToken" /></template>
