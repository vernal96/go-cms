<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import {
  ElAlert,
  ElButton,
  ElCard,
  ElCheckbox,
  ElForm,
  ElFormItem,
  ElInput,
  ElMessage,
	ElPagination,
  ElSkeleton,
  ElTable,
  ElTableColumn,
	ElTabPane,
	ElTabs,
  ElTag,
} from 'element-plus'
import { useRoute, useRouter } from 'vue-router'

import { AdminAPIError, adminRequest } from '../api/admin-api'
import AccessDeniedView from '../components/AccessDeniedView.vue'
import type { GroupDetailsResponse, GroupSiteAccess, PermissionCatalogResponse, PermissionDefinition, Site, SiteListResponse } from '../types/admin'

const props = defineProps<{ accessToken: string; permissions: ReadonlySet<string> }>()
const emit = defineEmits<{ unauthorized: [] }>()
const route = useRoute()
const router = useRouter()
const isCreate = computed(() => route.params.groupId === undefined)
const groupId = computed(() => Number(route.params.groupId))
const hasAccess = computed(() => isCreate.value ? props.permissions.has('core.group.create') : props.permissions.has('core.group.read'))
const loading = ref(true)
const saving = ref(false)
const error = ref<string | null>(null)
const canManage = ref(false)
const system = ref(false)
const selectedCodes = ref<string[]>([])
const catalog = ref<PermissionDefinition[]>([])
const activeTab = ref('general')
const sites = ref<Site[]>([])
const siteAccess = ref(new Map<number, GroupSiteAccess>())
const siteSearch = ref('')
const sitePage = ref(1)
const siteTotal = ref(0)
const sitesLoading = ref(false)
const sitePerPage = 10
const form = reactive({ code: '', name: '' })
const actions = ['read', 'create', 'update', 'delete'] as const

const permissionRows = computed(() => {
  const result = new Map<string, { key: string; module: string; entity: string; codes: Partial<Record<(typeof actions)[number], string>> }>()
  for (const item of catalog.value.filter((definition) => definition.code !== 'core.site.create')) {
    const key = `${item.module}.${item.entity}`
    const row = result.get(key) ?? { key, module: item.module, entity: item.entity, codes: {} }
    row.codes[item.action] = item.code
    result.set(key, row)
  }
  return [...result.values()]
})

onMounted(() => {
  if (hasAccess.value) void initialize()
})

async function initialize(): Promise<void> {
  loading.value = true
  try {
    const catalogResponse = await adminRequest<PermissionCatalogResponse>('/api/admin/permission-catalog', props.accessToken)
    catalog.value = catalogResponse.items
    canManage.value = catalogResponse.can_manage
    if (!isCreate.value) {
      const response = await adminRequest<GroupDetailsResponse>(`/api/admin/groups/${groupId.value}`, props.accessToken)
      form.code = response.group.code
      form.name = response.group.name
      system.value = response.group.system
      canManage.value = response.group.can_manage_permissions
      selectedCodes.value = response.permission_codes
		siteAccess.value = new Map(response.site_access.map((item) => [item.site_id, item]))
    }
	if (canManage.value && !system.value) await loadSites()
  } catch (caught) {
    handleError(caught)
  } finally {
    loading.value = false
  }
}

async function loadSites(): Promise<void> {
	sitesLoading.value = true
	try {
		const query = new URLSearchParams({ search: siteSearch.value.trim(), page: String(sitePage.value), per_page: String(sitePerPage) })
		const response = await adminRequest<SiteListResponse>(`/api/sites?${query}`, props.accessToken)
		sites.value = response.items
		siteTotal.value = response.pagination.total
	} catch (caught) {
		handleError(caught)
	} finally {
		sitesLoading.value = false
	}
}

function checked(code?: string): boolean {
  return code !== undefined && selectedCodes.value.includes(code)
}

function updatePermission(code: string | undefined, enabled: boolean): void {
  if (!code || !canManage.value || system.value) return
  const next = new Set(selectedCodes.value)
  if (enabled) next.add(code)
  else next.delete(code)
  selectedCodes.value = [...next]
}

function siteGrant(siteId: number): GroupSiteAccess {
	return siteAccess.value.get(siteId) ?? { site_id: siteId, can_view: false, can_edit: false, can_delete: false }
}

function updateSiteAccess(siteId: number, capability: 'view' | 'edit' | 'delete', enabled: boolean): void {
	if (!canManage.value || system.value) return
	const next = { ...siteGrant(siteId) }
	if (capability === 'view') {
		next.can_view = enabled
		if (!enabled) {
			next.can_edit = false
			next.can_delete = false
		}
	} else if (capability === 'edit') {
		next.can_edit = enabled
		if (enabled) next.can_view = true
		else next.can_delete = false
	} else {
		next.can_delete = enabled
		if (enabled) {
			next.can_edit = true
			next.can_view = true
		}
	}
	const grants = new Map(siteAccess.value)
	if (next.can_view) grants.set(siteId, next)
	else grants.delete(siteId)
	siteAccess.value = grants
}

function updateSiteSearch(): void {
	sitePage.value = 1
	void loadSites()
}

async function save(): Promise<void> {
  const code = form.code.trim().toLowerCase()
  const name = form.name.trim()
  if (!name || (isCreate.value && !/^[a-z][a-z0-9_-]{1,63}$/.test(code))) {
    ElMessage.error('Заполните название и корректный код группы')
    return
  }
  saving.value = true
  try {
    if (isCreate.value) {
	  const grants = [...siteAccess.value.values()]
      await adminRequest<GroupDetailsResponse>('/api/admin/groups', props.accessToken, {
        method: 'POST',
        body: JSON.stringify({ code, name, permission_codes: canManage.value ? selectedCodes.value : [], site_access: canManage.value ? grants : [] }),
      })
      ElMessage.success('Группа создана')
    } else {
	  const payload: { name: string; permission_codes?: string[]; site_access?: GroupSiteAccess[] } = { name }
      if (canManage.value && !system.value) {
		payload.permission_codes = selectedCodes.value
		payload.site_access = [...siteAccess.value.values()]
	  }
      await adminRequest<GroupDetailsResponse>(`/api/admin/groups/${groupId.value}`, props.accessToken, { method: 'PATCH', body: JSON.stringify(payload) })
      ElMessage.success('Группа сохранена')
    }
    await router.push('/admin/groups')
  } catch (caught) {
    handleError(caught)
  } finally {
    saving.value = false
  }
}

function entityLabel(value: unknown): string {
	const row = value as { key: string; module: string; entity: string }
  const labels: Record<string, string> = {
    'admin.panel': 'Административная панель',
    'core.site': 'Сайты',
    'core.resource': 'Ресурсы',
    'core.file': 'Файлы',
    'core.media': 'Медиа',
    'core.user': 'Пользователи',
    'core.group': 'Группы',
  }
  return labels[row.key] ?? row.key
}

function actionLabel(action: string): string {
  return ({ read: 'Просмотр', create: 'Создание', update: 'Изменение', delete: 'Удаление / блокировка' } as Record<string, string>)[action]
}

function handleError(caught: unknown): void {
  if (caught instanceof AdminAPIError && caught.status === 401) {
    emit('unauthorized')
    return
  }
  error.value = caught instanceof Error ? caught.message : 'Операция не выполнена.'
  ElMessage.error(error.value)
}
</script>

<template>
  <access-denied-view v-if="!hasAccess" @switch-user="emit('unauthorized')" />
  <section v-else class="workspace-page form-page">
    <header class="page-header">
      <div><h1>{{ isCreate ? 'Новая группа' : 'Редактирование группы' }}</h1><p>Название, код и разрешения</p></div>
      <el-button @click="router.push('/admin/groups')">К списку</el-button>
    </header>
    <el-skeleton v-if="loading" animated :rows="8" />
    <el-card v-else shadow="never" class="editor-card">
      <el-alert v-if="error" type="error" :title="error" show-icon :closable="false" />
      <el-form label-position="top" class="group-form">
        <div class="form-grid">
          <el-form-item label="Название" required><el-input v-model="form.name" /></el-form-item>
          <el-form-item label="Код" required>
            <el-input v-model="form.code" :disabled="!isCreate" />
            <div class="field-hint">Стабильный технический идентификатор; после создания не изменяется.</div>
          </el-form-item>
        </div>
      </el-form>
      <div class="permissions-heading">
        <div><h2>Настройки прав</h2><p>Разрешения сгруппированы по сущностям системы.</p></div>
        <el-tag v-if="system" type="warning">Системная группа</el-tag>
        <el-tag v-else-if="!canManage" type="info">Только просмотр</el-tag>
      </div>
      <el-alert v-if="system" type="info" title="Супергруппа имеет неограниченный доступ ко всем сайтам." show-icon :closable="false" />
      <el-tabs v-model="activeTab" class="group-permission-tabs">
        <el-tab-pane label="Общие права" name="general">
          <el-table :data="permissionRows" border class="permission-matrix">
            <el-table-column label="Сущность" min-width="240">
              <template #default="{ row }"><strong>{{ entityLabel(row) }}</strong><div class="permission-code">{{ row.key }}</div></template>
            </el-table-column>
            <el-table-column v-for="action in actions" :key="action" :label="actionLabel(action)" width="160" align="center">
              <template #default="{ row }">
                <el-checkbox
                  v-if="row.codes[action]"
                  :model-value="checked(row.codes[action])"
                  :disabled="system || !canManage"
                  :aria-label="`${row.key} ${action}`"
                  @change="updatePermission(row.codes[action], Boolean($event))"
                />
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
        <el-tab-pane label="Доступ к сайтам" name="sites">
          <div class="site-create-permission">
            <div><strong>Создание сайтов</strong><div class="permission-code">core.site.create</div></div>
            <el-checkbox
              :model-value="checked('core.site.create')"
              :disabled="system || !canManage"
              aria-label="core.site.create"
              @change="updatePermission('core.site.create', Boolean($event))"
            />
          </div>
          <template v-if="!system">
            <el-input
              v-model="siteSearch"
              clearable
              placeholder="Поиск сайта"
              class="site-access-search"
              @keyup.enter="updateSiteSearch"
              @clear="updateSiteSearch"
            />
            <el-table :data="sites" border class="permission-matrix" v-loading="sitesLoading">
              <el-table-column prop="domain" label="Сайт" min-width="280" />
              <el-table-column label="Просмотр" width="150" align="center">
                <template #default="{ row }"><el-checkbox :model-value="siteGrant(row.id).can_view" :disabled="!canManage" :aria-label="`${row.domain} view`" @change="updateSiteAccess(row.id, 'view', Boolean($event))" /></template>
              </el-table-column>
              <el-table-column label="Редактирование" width="170" align="center">
                <template #default="{ row }"><el-checkbox :model-value="siteGrant(row.id).can_edit" :disabled="!canManage" :aria-label="`${row.domain} edit`" @change="updateSiteAccess(row.id, 'edit', Boolean($event))" /></template>
              </el-table-column>
              <el-table-column label="Удаление" width="150" align="center">
                <template #default="{ row }"><el-checkbox :model-value="siteGrant(row.id).can_delete" :disabled="!canManage" :aria-label="`${row.domain} delete`" @change="updateSiteAccess(row.id, 'delete', Boolean($event))" /></template>
              </el-table-column>
            </el-table>
            <el-pagination
              v-if="siteTotal > sitePerPage"
              v-model:current-page="sitePage"
              :page-size="sitePerPage"
              :total="siteTotal"
              layout="prev, pager, next"
              @current-change="loadSites"
            />
          </template>
        </el-tab-pane>
      </el-tabs>
      <div class="form-submit"><el-button v-if="isCreate || permissions.has('core.group.update')" type="primary" :loading="saving" @click="save">{{ isCreate ? 'Создать группу' : 'Сохранить группу' }}</el-button></div>
    </el-card>
  </section>
</template>
