<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElAlert, ElButton, ElCard, ElInput, ElMessage } from 'element-plus'
import { AdminAPIError, adminRequest } from '../api/admin-api'

const props = defineProps<{ accessToken: string }>()
const emit = defineEmits<{ unauthorized: [] }>()
const count = ref(0)
const loading = ref(true)
const purging = ref(false)
const error = ref<string | null>(null)
const confirmText = ref('')

async function load(): Promise<void> {
	loading.value = true; error.value = null
	try {
		const response = await adminRequest<{ count: number }>('/api/administration/resource-revisions', props.accessToken)
		count.value = response.count
	} catch (reason) {
		if (reason instanceof AdminAPIError && reason.status === 401) emit('unauthorized')
		else error.value = reason instanceof AdminAPIError && reason.status === 403 ? 'Раздел доступен только участникам встроенной группы администраторов.' : 'Не удалось загрузить статистику.'
	} finally { loading.value = false }
}

async function purge(): Promise<void> {
	if (confirmText.value !== 'DELETE' || purging.value) return
	purging.value = true; error.value = null
	try {
		const response = await adminRequest<{ count: number }>('/api/administration/resource-revisions', props.accessToken, { method: 'DELETE' })
		ElMessage.success(`Удалено ревизий: ${response.count}`)
		confirmText.value = ''
		await load()
	} catch (reason) {
		error.value = reason instanceof Error ? reason.message : 'Не удалось очистить историю.'
	} finally { purging.value = false }
}

onMounted(() => void load())
</script>

<template>
	<section class="workspace-page">
		<header class="page-header"><div><h1>Администрирование</h1><p>Глобальные операции обслуживания CMS</p></div></header>
		<el-alert v-if="error" type="error" :closable="false" :title="error" show-icon />
		<el-card v-if="!error || count > 0" v-loading="loading" header="История ресурсов">
			<p>Всего ревизий: <strong>{{ count }}</strong></p>
			<el-alert type="warning" :closable="false" show-icon title="История ресурсов будет удалена без возможности восстановления. Текущие ресурсы не удаляются и не изменяются." />
			<p>Для удаления {{ count }} ревизий введите <code>DELETE</code>.</p>
			<el-input v-model="confirmText" class="administration-confirm" placeholder="DELETE" :disabled="purging || count === 0" />
			<el-button type="danger" :loading="purging" :disabled="confirmText !== 'DELETE' || count === 0" @click="purge">Очистить всю историю</el-button>
		</el-card>
	</section>
</template>

<style scoped>.administration-confirm{max-width:20rem;margin:0 1rem 1rem 0}</style>
