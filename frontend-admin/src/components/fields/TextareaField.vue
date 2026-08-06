<script setup lang="ts">
import { ElInput } from 'element-plus'

const model = defineModel<any>()
</script>

<template>
  <el-input v-model="model" type="textarea" :rows="5" resize="vertical" />
</template>
