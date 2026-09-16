<script setup lang="ts">
import { inject, ref, watch } from 'vue'
import { adminAccessTokenKey, adminRequest, useSelectedSite, useFieldValidation, DynamicFieldsForm, ConfigurationEditor, type FieldDefinition, type ConfigField } from '@go-cms/admin/sdk'
const token = inject(adminAccessTokenKey)!
const selected = useSelectedSite()
const {validateFieldValues}=useFieldValidation()
const fields=ref<FieldDefinition[]>([])
const values=ref<Record<string,unknown>>({})
const config=ref<Record<string,unknown>>({text:'Initial notice'})
const elementType=ref<{editor_code:string;fields:ConfigField[]}>()
const configEditor=ref<{validate():void}>()
const site=ref<{id:number;profile_code:string;domain:string;locale:string;is_public:boolean;settings:Record<string,unknown>}>()
const formID=ref(0),elementID=ref(0)
const error=ref(''),status=ref(''),loading=ref(false),saving=ref(false),available=ref(false)
let generation=0
const api=<T,>(path:string,method='GET',body?:unknown)=>adminRequest<T>(path,token.value,{method,...(body===undefined?{}:{body:JSON.stringify(body)})})
async function load(){
 const current=++generation
 available.value=false;fields.value=[];error.value='';status.value='';formID.value=0;elementID.value=0
 const id=selected.selectedSite.value?.id;if(!id)return
 loading.value=true
 try{
  const [detail,profiles]=await Promise.all([api<{site:NonNullable<typeof site.value>}>(`/api/sites/${id}`),api<{items:Array<{code:string;fields:FieldDefinition[]}>}>('/api/site-profiles')])
  if(current!==generation)return
  site.value=detail.site
  fields.value=profiles.items.find(profile=>profile.code===detail.site.profile_code)?.fields??[]
  available.value=fields.value.some(field=>field.editor==='example.text')
  if(!available.value)return
  values.value={...detail.site.settings}
  const forms=await api<{items:Array<{id:number;code:string}>}>(`/api/sites/${id}/forms/forms`)
  if(current!==generation)return
  formID.value=forms.items.find(item=>item.code==='extension')?.id??0
  if(formID.value)await loadForm(id,current)
 }catch(caught){if(current===generation)error.value=String(caught)}finally{if(current===generation)loading.value=false}
}
async function loadForm(id:number,current:number){
 const editor=await api<{elements:Array<{id:number;type:string;config:Record<string,unknown>}>;available_element_types:Array<{code:string;editor_code:string;fields:ConfigField[]}>}>(`/api/sites/${id}/forms/forms/${formID.value}/editor`)
 if(current!==generation)return
 elementType.value=editor.available_element_types.find(item=>item.code==='example.notice')
 const element=editor.elements.find(item=>item.type==='example.notice')
 elementID.value=element?.id??0;config.value=element?.config??{text:'Initial notice'}
}
async function createForm(){
 const id=site.value?.id;if(!id)return
 saving.value=true;error.value=''
 try{
  await api(`/api/sites/${id}/forms/forms`,'POST',{code:'extension',name:'External extension',description:'SDK example',enabled:true})
  await load()
 }catch(caught){error.value=String(caught)}finally{saving.value=false}
}
async function save(){
 const current=site.value;if(!current||loading.value||saving.value)return
 error.value='';status.value=''
 const errors=validateFieldValues(fields.value,values.value)
 if(Object.keys(errors).length){error.value=Object.entries(errors).map(([key,value])=>`${key}: ${value}`).join('; ');return}
 try{configEditor.value?.validate()}catch(caught){error.value=String(caught);return}
 saving.value=true
 try{
  await api(`/api/sites/${current.id}`,'PATCH',{profile_code:current.profile_code,domain:current.domain,locale:current.locale,is_public:current.is_public,settings:values.value})
  if(formID.value&&elementType.value){
   const root=`/api/sites/${current.id}/forms/forms/${formID.value}/elements`
   await api(elementID.value?`${root}/${elementID.value}`:root,elementID.value?'PATCH':'POST',{code:'notice',type:'example.notice',config:config.value,...(elementID.value?{}:{position:3})})
  }
  await load();status.value='Сохранено'
 }catch(caught){error.value=String(caught)}finally{saving.value=false}
}
watch(selected.selectedSite,load,{immediate:true})
</script>
<template>
 <main><h1>Внешнее расширение</h1>
  <p v-if="loading">Загрузка…</p>
  <p v-else-if="!available">Модуль недоступен на выбранном сайте.</p>
  <fieldset v-else :disabled="saving">
   <DynamicFieldsForm v-model="values" :fields="fields" :site-id="site?.id" :access-token="token" />
   <button v-if="!formID" @click="createForm">Создать форму примера</button>
   <ConfigurationEditor v-if="elementType" ref="configEditor" v-model="config" :fields="elementType.fields" :editor="elementType.editor_code" :site-id="site?.id" :access-token="token" />
   <button @click="save">Сохранить</button>
  </fieldset>
  <p role="alert">{{error}}</p><p role="status">{{status}}</p>
 </main>
</template>
