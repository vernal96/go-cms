<script setup lang="ts">
import { computed } from 'vue'
import { ElButton, ElInput } from 'element-plus'
const model = defineModel<unknown>()
const rows = computed<Array<{value:string;label:string}>>(() => Array.isArray(model.value) ? model.value : [])
function update(index:number, key:'value'|'label', value:string):void { model.value = rows.value.map((row,i) => i === index ? {...row,[key]:value} : row) }
function remove(index:number):void { model.value = rows.value.filter((_,i) => i !== index) }
function add():void { model.value = [...rows.value,{value:'',label:''}] }
</script>
<template>
 <div class="choices"><div v-for="(row,index) in rows" :key="index" class="choice"><el-input :model-value="row.value" placeholder="Значение" @update:model-value="update(index,'value',$event)" /><el-input :model-value="row.label" placeholder="Подпись" @update:model-value="update(index,'label',$event)" /><el-button @click="remove(index)">Удалить</el-button></div><el-button @click="add">Добавить вариант</el-button></div>
</template>
<style scoped>.choices{width:100%}.choice{display:flex;gap:8px;margin-bottom:8px}</style>
