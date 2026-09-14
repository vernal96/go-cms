<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { ElForm, ElFormItem, ElInput, ElOption, ElSelect } from 'element-plus'
import ConfigurationEditor from '../../components/fields/ConfigurationEditor.vue'
import type { ElementType, ElementTypeMetadata, FormElement } from './types'
const props = defineProps<{disabled:boolean;initialType:ElementType;element?:FormElement|null;availableTypes:ElementTypeMetadata[];accessToken:string;permissions:ReadonlySet<string>;siteId?:number}>()
const emit = defineEmits<{dirty:[value:boolean]}>()
const state=reactive({code:'',type:props.initialType})
const values=ref<Record<string,unknown>>({})
const editor=ref<{ validate(): void }>()
const selectedType=computed(() => props.availableTypes.find(item => item.code === state.type))
const locked=computed(() => props.element?.type === 'submit_button')
let baseline=''
watch(() => state.type, () => {values.value={}}, {flush:'sync'})
onMounted(() => {
 state.code=props.element?.code ?? '';state.type=props.element?.type ?? props.initialType
 values.value=JSON.parse(JSON.stringify(props.element?.config ?? {}))
 baseline=JSON.stringify([state,values.value]);emit('dirty',false)
})
watch([state,values],() => emit('dirty',JSON.stringify([state,values.value]) !== baseline),{deep:true,flush:'sync'})
function payload():Pick<FormElement,'code'|'type'|'config'> {
 if (!selectedType.value) throw new Error('Тип элемента недоступен.')
 editor.value?.validate()
 return {code:state.code.trim(),type:state.type,config:JSON.parse(JSON.stringify(values.value))}
}
defineExpose({payload})
</script>
<template>
 <el-form label-position="top" :disabled="disabled" @submit.prevent>
  <el-form-item v-if="element" label="Тип" required><el-select v-model="state.type" :disabled="locked"><el-option v-for="type in availableTypes" :key="type.code" :value="type.code" :label="type.label" /></el-select></el-form-item>
  <el-form-item label="Код" required><el-input v-model="state.code" :disabled="locked" /></el-form-item>
  <configuration-editor v-if="selectedType" ref="editor" v-model="values" :fields="selectedType.fields" :editor="selectedType.editor_code" :site-id="siteId" :access-token="accessToken" :context="{permissions}" />
 </el-form>
</template>
<style scoped>.el-select{width:100%}</style>
