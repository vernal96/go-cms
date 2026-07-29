<script setup lang="ts">
import { ElButton, ElCard, ElResult } from 'element-plus'

defineEmits<{
  switchUser: []
}>()
</script>

<template>
  <section class="app-state auth-page">
    <el-card class="access-card" shadow="never">
      <el-result
        icon="warning"
        title="Доступ запрещён"
        sub-title="У вашей группы нет права на вход в панель администратора."
      >
        <template #extra>
          <el-button type="primary" @click="$emit('switchUser')">
            Войти другим пользователем
          </el-button>
        </template>
      </el-result>
    </el-card>
  </section>
</template>
