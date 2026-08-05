<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import {
  ElAlert,
  ElButton,
  ElDialog,
  ElForm,
  ElFormItem,
  ElInput,
  ElOption,
  ElSelect,
} from 'element-plus'

import { adminRequest } from '../api/admin-api'
import type {
  ResourceCreatePayload,
  ResourceMetadata,
  ResourceTreeItem,
} from '../types/admin'

const props = defineProps<{ accessToken: string; siteId: number }>()
const emit = defineEmits<{
  created: [item: ResourceTreeItem, parentId: number | null]
  error: [error: unknown]
}>()

const visible = ref(false)
const loading = ref(false)
const metadataLoading = ref(false)
const errorMessage = ref<string | null>(null)
const metadata = ref<ResourceMetadata>({ types: [], templates: [] })
const parent = ref<ResourceTreeItem | null>(null)
const form = reactive({
  type: 'page' as 'page' | 'link',
  template_code: '',
  title: '',
  menu_title: '',
  slug: '',
  external_url: '',
})
const title = computed(() =>
  parent.value ? `Создать ресурс в «${parent.value.display_title}»` : 'Создать корневой ресурс',
)

async function open(parentItem: ResourceTreeItem | null): Promise<void> {
  parent.value = parentItem
  Object.assign(form, {
    type: 'page', template_code: '', title: '', menu_title: '', slug: '', external_url: '',
  })
  errorMessage.value = null
  visible.value = true
  metadataLoading.value = true
  try {
    metadata.value = await adminRequest<ResourceMetadata>(
      `/api/admin/sites/${props.siteId}/resource-metadata`,
      props.accessToken,
    )
    form.type = metadata.value.types[0]?.code ?? 'link'
    form.template_code = metadata.value.templates[0]?.code ?? ''
  } catch (error) {
    errorMessage.value = 'Не удалось загрузить типы ресурсов.'
    emit('error', error)
  } finally {
    metadataLoading.value = false
  }
}

async function submit(): Promise<void> {
  errorMessage.value = null
  if (!form.title.trim() || (parent.value && !form.slug.trim())) {
    errorMessage.value = 'Заполните название и slug дочернего ресурса.'
    return
  }
  if (form.type === 'page' && !form.template_code) {
    errorMessage.value = 'Выберите шаблон страницы.'
    return
  }
  if (form.type === 'link' && !form.external_url.trim()) {
    errorMessage.value = 'Укажите адрес ссылки.'
    return
  }
  const payload: ResourceCreatePayload = {
    parent_id: parent.value?.id ?? null,
    type: form.type,
    title: form.title.trim(),
    menu_title: form.menu_title.trim(),
    slug: form.slug.trim(),
  }
  if (form.type === 'page') payload.template_code = form.template_code
  else payload.external_url = form.external_url.trim()

  loading.value = true
  try {
    const created = await adminRequest<ResourceTreeItem>(
      `/api/admin/sites/${props.siteId}/resources`,
      props.accessToken,
      { method: 'POST', body: JSON.stringify(payload) },
    )
    visible.value = false
    emit('created', created, parent.value?.id ?? null)
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : 'Не удалось создать ресурс.'
    emit('error', error)
  } finally {
    loading.value = false
  }
}

defineExpose({ open })
</script>

<template>
  <el-dialog v-model="visible" :title="title" width="520px" destroy-on-close>
    <el-alert v-if="errorMessage" class="dialog-alert" type="error" :closable="false" :title="errorMessage" />
    <el-form label-position="top" :model="form" :disabled="metadataLoading">
      <el-form-item label="Тип">
        <el-select v-model="form.type" class="full-width">
          <el-option v-for="item in metadata.types" :key="item.code" :label="item.label" :value="item.code" />
        </el-select>
      </el-form-item>
      <el-form-item v-if="form.type === 'page'" label="Шаблон">
        <el-select v-model="form.template_code" class="full-width">
          <el-option v-for="item in metadata.templates" :key="item.code" :label="item.label" :value="item.code" />
        </el-select>
      </el-form-item>
      <el-form-item label="Название"><el-input v-model="form.title" /></el-form-item>
      <el-form-item label="Название в меню"><el-input v-model="form.menu_title" /></el-form-item>
      <el-form-item label="Slug"><el-input v-model="form.slug" placeholder="Можно оставить пустым только для корневой страницы" /></el-form-item>
      <el-form-item v-if="form.type === 'link'" label="Адрес ссылки">
        <el-input v-model="form.external_url" placeholder="https://example.com" />
      </el-form-item>
    </el-form>
    <template #footer>
      <el-button @click="visible = false">Отмена</el-button>
      <el-button type="primary" :loading="loading" :disabled="metadataLoading" @click="submit">Создать</el-button>
    </template>
  </el-dialog>
</template>
