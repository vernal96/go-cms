<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import {
  ElAlert,
  ElButton,
  ElDialog,
  ElEmpty,
  ElMessage,
  ElMessageBox,
  ElOption,
  ElSelect,
  ElSkeleton,
} from 'element-plus'
import {
  ArrowLeft,
  Back,
  Delete,
  Document,
  Download,
  Folder,
  FolderAdd,
  HomeFilled,
  Picture,
  Refresh,
  Rank,
  Upload,
} from '@element-plus/icons-vue'

import {
  AdminAPIError,
  adminBlob,
  adminRequest,
  adminRequestVoid,
  adminUpload,
} from '../../api/admin-api'
import type {
  FilesystemDisksResponse,
  FilesystemItem,
  FilesystemListingResponse,
} from '../../types/admin'
import FolderMoveDialog from './FolderMoveDialog.vue'

const props = withDefaults(defineProps<{
  accessToken: string
  permissions: ReadonlySet<string>
  picker?: boolean
  allowedStorages?: string[]
  allowedMIMETypes?: string[]
}>(), {
  picker: false,
  allowedStorages: () => [],
  allowedMIMETypes: () => [],
})
const emit = defineEmits<{ select: [item: FilesystemItem] }>()

const disks = ref<FilesystemDisksResponse['items']>([])
const permissions = ref({ read: false, create: false, update: false, delete: false })
const listing = ref<FilesystemListingResponse | null>(null)
const disk = ref('')
const loading = ref(true)
const error = ref('')
const selected = ref(new Set<string>())
const anchorIndex = ref<number | null>(null)
const sortBy = ref<'name' | 'created_at' | 'updated_at' | 'size'>('name')
const sortDirection = ref<'asc' | 'desc'>('asc')
const history = ref<Array<number | null>>([])
const previewURL = ref('')
const previewName = ref('')
const previewVisible = ref(false)
const uploadInput = ref<HTMLInputElement | null>(null)
const folderInput = ref<HTMLInputElement | null>(null)
const contextMenu = ref<{ item: FilesystemItem; x: number; y: number } | null>(null)
const dropActive = ref(false)
const moveVisible = ref(false)
const moveItems = ref<FilesystemItem[]>([])

const visibleDisks = computed(() => props.allowedStorages.length
  ? disks.value.filter((item) => props.allowedStorages.includes(item.code))
  : disks.value)
const sortedItems = computed(() => [...(listing.value?.items ?? [])].sort((left, right) => {
  if (left.kind !== right.kind) return left.kind === 'folder' ? -1 : 1
  let result = 0
  if (sortBy.value === 'name') result = left.name.localeCompare(right.name, 'ru', { numeric: true })
  else if (sortBy.value === 'size') result = (left.size ?? -1) - (right.size ?? -1)
  else result = new Date(left[sortBy.value]).getTime() - new Date(right[sortBy.value]).getTime()
  if (result === 0) result = left.id - right.id
  return sortDirection.value === 'asc' ? result : -result
}))
const selectedItems = computed(() => sortedItems.value.filter((item) => selected.value.has(itemKey(item))))
const selectableItem = computed(() => selectedItems.value.length === 1 && selectedItems.value[0]?.kind === 'file'
  ? selectedItems.value[0]
  : null)
const selectedSummary = computed(() => {
  const items = selectedItems.value
  if (items.length === 0) return 'Ничего не выбрано'
  if (items.length > 1) {
    const folders = items.filter((item) => item.kind === 'folder').length
    const files = items.length - folders
    const size = items.reduce((sum, item) => sum + (item.size ?? 0), 0)
    return `Выбрано: ${items.length} · папок: ${folders} · файлов: ${files} · ${formatSize(size)}`
  }
  const item = items[0]!
  const details = item.kind === 'folder'
    ? `${item.item_count ?? 0} элементов`
    : `${item.mime_type ?? 'файл'} · ${formatSize(item.size ?? 0)}`
  return `${item.name} · создан: ${formatDate(item.created_at)} · изменён: ${formatDate(item.updated_at)} · ${details}`
})

onMounted(() => {
  document.addEventListener('click', closeContextMenu)
  void initialize()
})
onBeforeUnmount(() => {
  document.removeEventListener('click', closeContextMenu)
  revokePreview()
})
watch(() => props.allowedStorages, () => {
  if (disk.value && !visibleDisks.value.some((item) => item.code === disk.value)) {
    disk.value = visibleDisks.value[0]?.code ?? ''
    void loadFolder(null, false)
  }
}, { deep: true })

async function initialize(): Promise<void> {
  loading.value = true
  error.value = ''
  try {
    const response = await adminRequest<FilesystemDisksResponse>('/api/admin/filesystem/disks', props.accessToken)
    disks.value = response.items
    permissions.value = response.permissions
    disk.value = visibleDisks.value[0]?.code ?? ''
    if (disk.value) await loadFolder(null, false)
  } catch (caught) {
    error.value = message(caught, 'Не удалось загрузить диски.')
  } finally {
    loading.value = false
  }
}

async function loadFolder(folderID: number | null, remember = true): Promise<void> {
  if (!disk.value) return
  const previous = listing.value?.folder?.id ?? null
  loading.value = true
  error.value = ''
  try {
    const query = new URLSearchParams({ disk: disk.value })
    if (folderID !== null) query.set('folder_id', String(folderID))
    listing.value = await adminRequest<FilesystemListingResponse>(`/api/admin/filesystem/items?${query}`, props.accessToken)
    if (remember && previous !== folderID) history.value.push(previous)
    selected.value = new Set()
    anchorIndex.value = null
  } catch (caught) {
    error.value = message(caught, 'Не удалось открыть папку.')
  } finally {
    loading.value = false
  }
}

function changeDisk(): void {
  history.value = []
  listing.value = null
  void loadFolder(null, false)
}
function goBack(): void {
  const target = history.value.pop()
  if (target !== undefined) void loadFolder(target, false)
}
function goUp(): void {
  void loadFolder(listing.value?.folder?.parent_id ?? null)
}
function activate(item: FilesystemItem): void {
  if (item.kind === 'folder') void loadFolder(item.id)
  else if (isImage(item)) void preview(item)
  else void confirmDownload(item)
}

function choose(item: FilesystemItem, event: MouseEvent, index: number): void {
  const next = new Set(event.ctrlKey || event.metaKey ? selected.value : [])
  if (event.shiftKey && anchorIndex.value !== null) {
    const [start, end] = [anchorIndex.value, index].sort((a, b) => a - b)
    for (let current = start; current <= end; current++) next.add(itemKey(sortedItems.value[current]!))
  } else if (next.has(itemKey(item)) && (event.ctrlKey || event.metaKey)) {
    next.delete(itemKey(item))
  } else {
    next.add(itemKey(item))
    anchorIndex.value = index
  }
  selected.value = next
}

async function createFolder(parentID = listing.value?.folder?.id ?? null, suggested = ''): Promise<FilesystemItem | null> {
  if (!permissions.value.create) return null
  try {
    const { value } = await ElMessageBox.prompt('Название папки', 'Новая папка', {
      inputValue: suggested,
      confirmButtonText: 'Создать', cancelButtonText: 'Отмена',
      inputValidator: (name) => name.trim().length > 0 || 'Введите название.',
    })
    const created = await adminRequest<FilesystemItem>('/api/admin/filesystem/folders', props.accessToken, {
      method: 'POST', body: JSON.stringify({ disk: disk.value, parent_id: parentID, name: value.trim() }),
    })
    await loadFolder(listing.value?.folder?.id ?? null, false)
    return created
  } catch { return null }
}

async function rename(item: FilesystemItem): Promise<void> {
  if (!permissions.value.update) return
  try {
    const { value } = await ElMessageBox.prompt('Новое название', 'Переименование', {
      inputValue: item.name, confirmButtonText: 'Сохранить', cancelButtonText: 'Отмена',
      inputValidator: (name) => name.trim().length > 0 || 'Введите название.',
    })
    const path = item.kind === 'folder' ? `/api/admin/filesystem/folders/${item.id}` : `/api/admin/filesystem/files/${item.id}`
    await adminRequest<FilesystemItem>(path, props.accessToken, { method: 'PATCH', body: JSON.stringify({ name: value.trim() }) })
    await loadFolder(listing.value?.folder?.id ?? null, false)
  } catch { /* cancelled */ }
}

async function remove(items = selectedItems.value): Promise<void> {
  if (!permissions.value.delete || items.length === 0) return
  try {
    await ElMessageBox.confirm(`Удалить выбранные объекты (${items.length}) без возможности восстановления?`, 'Удаление', {
      type: 'warning', confirmButtonText: 'Удалить', cancelButtonText: 'Отмена',
    })
    await adminRequestVoid('/api/admin/filesystem/delete', props.accessToken, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ items: items.map(reference) }),
    })
    await loadFolder(listing.value?.folder?.id ?? null, false)
  } catch (caught) {
    if (caught instanceof AdminAPIError) ElMessage.error(message(caught, 'Не удалось удалить объекты.'))
  }
}

async function move(items: FilesystemItem[], folderID: number | null): Promise<void> {
  if (!permissions.value.update || items.length === 0 || items.some((item) => item.kind === 'folder' && item.id === folderID)) return
  try {
    await adminRequestVoid('/api/admin/filesystem/move', props.accessToken, {
      method: 'POST', headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ disk: disk.value, folder_id: folderID, items: items.map(reference) }),
    })
    await loadFolder(listing.value?.folder?.id ?? null, false)
  } catch (caught) { ElMessage.error(message(caught, 'Не удалось переместить объекты.')) }
}

function startInternalDrag(event: DragEvent, item: FilesystemItem): void {
  if (!permissions.value.update || !event.dataTransfer) return
  const items = selected.value.has(itemKey(item)) ? selectedItems.value : [item]
  event.dataTransfer.setData('application/x-cms-files', JSON.stringify(items.map(reference)))
  event.dataTransfer.effectAllowed = 'move'
}

async function handleDrop(event: DragEvent, targetFolderID = listing.value?.folder?.id ?? null): Promise<void> {
  event.preventDefault()
  dropActive.value = false
  const internal = event.dataTransfer?.getData('application/x-cms-files')
  if (internal) {
    const refs = JSON.parse(internal) as Array<{ kind: 'file' | 'folder'; id: number }>
    const items = sortedItems.value.filter((item) => refs.some((ref) => ref.kind === item.kind && ref.id === item.id))
    await move(items, targetFolderID)
    return
  }
  if (!permissions.value.create || !event.dataTransfer) return
  await uploadTransfer(event.dataTransfer, targetFolderID)
}

async function uploadTransfer(transfer: DataTransfer, parentID: number | null): Promise<void> {
  const entries = [...transfer.items].map((item) => (item as DataTransferItem & { webkitGetAsEntry?: () => FileSystemEntry | null }).webkitGetAsEntry?.()).filter(Boolean) as FileSystemEntry[]
  if (entries.length) {
    for (const entry of entries) await uploadEntry(entry, parentID)
  } else {
    for (const uploaded of transfer.files) await uploadFile(uploaded, parentID)
  }
  await loadFolder(listing.value?.folder?.id ?? null, false)
}

async function uploadEntry(entry: FileSystemEntry, parentID: number | null): Promise<void> {
  if (entry.isFile) {
    const uploaded = await entryFile(entry as FileSystemFileEntry)
    await uploadFile(uploaded, parentID)
    return
  }
  if (!('createReader' in entry)) {
    ElMessage.warning('Загрузка папок поддерживается в Chrome и Edge.')
    return
  }
  const created = await createFolderDirect(entry.name, parentID)
  if (!created) return
  for (const child of await readEntries(entry as FileSystemDirectoryEntry)) await uploadEntry(child, created.id)
}

async function createFolderDirect(name: string, parentID: number | null): Promise<FilesystemItem | null> {
  try {
    return await adminRequest<FilesystemItem>('/api/admin/filesystem/folders', props.accessToken, {
      method: 'POST', body: JSON.stringify({ disk: disk.value, parent_id: parentID, name }),
    })
  } catch (caught) { ElMessage.error(`${name}: ${message(caught, 'не удалось создать папку')}`); return null }
}

async function uploadFile(uploaded: File, parentID: number | null): Promise<void> {
  const data = new FormData()
  data.set('disk', disk.value)
  if (parentID !== null) data.set('folder_id', String(parentID))
  data.set('file', uploaded, uploaded.name)
  try { await adminUpload<FilesystemItem>('/api/admin/filesystem/uploads', props.accessToken, data) }
  catch (caught) { ElMessage.error(`${uploaded.name}: ${message(caught, 'ошибка загрузки')}`) }
}

async function uploadSelected(files: FileList | null): Promise<void> {
  if (!files) return
  const rootID = listing.value?.folder?.id ?? null
  const folders = new Map<string, number | null>([['', rootID]])
  for (const uploaded of files) {
    const relative = uploaded.webkitRelativePath || uploaded.name
    const parts = relative.split('/').filter(Boolean)
    let path = ''
    let parentID = rootID
    for (const segment of parts.slice(0, -1)) {
      path = path ? `${path}/${segment}` : segment
      if (!folders.has(path)) {
        const created = await createFolderDirect(segment, parentID)
        if (!created) break
        folders.set(path, created.id)
      }
      parentID = folders.get(path) ?? rootID
    }
    await uploadFile(uploaded, parentID)
  }
  await loadFolder(listing.value?.folder?.id ?? null, false)
  if (uploadInput.value) uploadInput.value.value = ''
  if (folderInput.value) folderInput.value.value = ''
}

async function preview(item: FilesystemItem): Promise<void> {
  revokePreview()
  try {
    const blob = await adminBlob(`/api/admin/filesystem/files/${item.id}/preview`, props.accessToken)
    previewURL.value = URL.createObjectURL(blob)
    previewName.value = item.name
    previewVisible.value = true
  } catch (caught) { ElMessage.error(message(caught, 'Не удалось открыть изображение.')) }
}
async function confirmDownload(item: FilesystemItem): Promise<void> {
  try {
    await ElMessageBox.confirm(`Скачать файл «${item.name}»?`, 'Скачивание', { confirmButtonText: 'Скачать', cancelButtonText: 'Отмена' })
    await download(item)
  } catch { /* cancelled */ }
}
async function download(item: FilesystemItem): Promise<void> {
  const blob = await adminBlob(`/api/admin/filesystem/files/${item.id}/download`, props.accessToken)
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url; link.download = item.name; link.click()
  URL.revokeObjectURL(url)
}
function revokePreview(): void {
  if (previewURL.value) URL.revokeObjectURL(previewURL.value)
  previewURL.value = ''
}
function showContext(event: MouseEvent, item: FilesystemItem): void {
  event.preventDefault(); event.stopPropagation()
  if (!selected.value.has(itemKey(item))) selected.value = new Set([itemKey(item)])
  contextMenu.value = { item, x: event.clientX, y: event.clientY }
}
function closeContextMenu(): void { contextMenu.value = null }
function openMoveDialog(): void {
  moveItems.value = [...selectedItems.value]
  moveVisible.value = moveItems.value.length > 0
  closeContextMenu()
}
async function confirmMove(folderID: number | null): Promise<void> {
  await move(moveItems.value, folderID)
  moveVisible.value = false
  moveItems.value = []
}
function confirmSelection(): void {
  if (selectableItem.value && matchesPicker(selectableItem.value)) emit('select', selectableItem.value)
}
function matchesPicker(item: FilesystemItem): boolean {
  if (item.kind !== 'file') return false
  if (props.allowedStorages.length && !props.allowedStorages.includes(item.storage)) return false
  return !props.allowedMIMETypes.length || props.allowedMIMETypes.some((allowed) => allowed === item.mime_type || (allowed.endsWith('/*') && item.mime_type?.startsWith(allowed.slice(0, -1))))
}
function itemKey(item: FilesystemItem): string { return `${item.kind}:${item.id}` }
function reference(item: FilesystemItem): { kind: 'file' | 'folder'; id: number } { return { kind: item.kind, id: item.id } }
function isImage(item: FilesystemItem): boolean { return item.mime_type?.startsWith('image/') ?? false }
function formatDate(value: string): string { return new Intl.DateTimeFormat('ru-RU', { dateStyle: 'short', timeStyle: 'short' }).format(new Date(value)) }
function formatSize(value: number): string {
  if (value < 1024) return `${value} Б`
  const units = ['КБ', 'МБ', 'ГБ', 'ТБ']; let result = value / 1024; let unit = 0
  while (result >= 1024 && unit < units.length - 1) { result /= 1024; unit++ }
  return `${result.toLocaleString('ru-RU', { maximumFractionDigits: 1 })} ${units[unit]}`
}
function message(errorValue: unknown, fallback: string): string { return errorValue instanceof Error ? errorValue.message : fallback }

interface FileSystemEntry { isFile: boolean; isDirectory: boolean; name: string }
interface FileSystemFileEntry extends FileSystemEntry { file: (success: (file: File) => void, error?: (error: DOMException) => void) => void }
interface FileSystemDirectoryReader { readEntries: (success: (entries: FileSystemEntry[]) => void, error?: (error: DOMException) => void) => void }
interface FileSystemDirectoryEntry extends FileSystemEntry { createReader: () => FileSystemDirectoryReader }
function entryFile(entry: FileSystemFileEntry): Promise<File> { return new Promise((resolve, reject) => entry.file(resolve, reject)) }
async function readEntries(entry: FileSystemDirectoryEntry): Promise<FileSystemEntry[]> {
  const reader = entry.createReader(); const result: FileSystemEntry[] = []
  for (;;) {
    const batch = await new Promise<FileSystemEntry[]>((resolve, reject) => reader.readEntries(resolve, reject))
    if (!batch.length) return result
    result.push(...batch)
  }
}
</script>

<template>
  <div class="file-explorer" @dragover.prevent="dropActive = true" @dragleave.self="dropActive = false" @drop="handleDrop($event)">
    <div class="file-toolbar">
      <el-select v-model="disk" aria-label="Диск" class="disk-select" @change="changeDisk">
        <el-option v-for="item in visibleDisks" :key="item.code" :label="`${item.code} · ${item.visibility === 'public' ? 'публичный' : 'приватный'}`" :value="item.code" />
      </el-select>
      <el-button :icon="ArrowLeft" :disabled="!history.length" title="Назад" @click="goBack" />
      <el-button :icon="Back" :disabled="!listing?.folder" title="Вверх" @click="goUp" />
      <el-button :icon="HomeFilled" :disabled="!listing?.folder" title="В корень" @click="loadFolder(null)" />
      <el-button :icon="Refresh" title="Обновить" @click="loadFolder(listing?.folder?.id ?? null, false)" />
      <el-button v-if="permissions.create" :icon="FolderAdd" @click="createFolder()">Папка</el-button>
      <el-button v-if="permissions.create" :icon="Upload" @click="uploadInput?.click()">Файлы</el-button>
      <el-button v-if="permissions.create" :icon="Upload" @click="folderInput?.click()">Папка с компьютера</el-button>
      <input ref="uploadInput" hidden type="file" multiple @change="uploadSelected(($event.target as HTMLInputElement).files)" />
      <input ref="folderInput" hidden type="file" webkitdirectory multiple @change="uploadSelected(($event.target as HTMLInputElement).files)" />
      <span class="toolbar-spacer" />
      <el-select v-model="sortBy" class="sort-select" aria-label="Сортировка">
        <el-option label="По имени" value="name" /><el-option label="По созданию" value="created_at" />
        <el-option label="По изменению" value="updated_at" /><el-option label="По размеру" value="size" />
      </el-select>
      <el-button @click="sortDirection = sortDirection === 'asc' ? 'desc' : 'asc'">{{ sortDirection === 'asc' ? '↑' : '↓' }}</el-button>
    </div>

    <nav class="file-breadcrumbs" aria-label="Путь">
      <button type="button" @click="loadFolder(null)">Корень</button><span>/</span>
      <template v-for="crumb in listing?.breadcrumbs ?? []" :key="crumb.id">
        <button type="button" @click="loadFolder(crumb.id)">{{ crumb.name }}</button><span>/</span>
      </template>
    </nav>

    <el-alert v-if="error" type="error" :closable="false" :title="error" />
    <el-skeleton v-else-if="loading" :rows="6" animated />
    <el-empty v-else-if="!sortedItems.length" description="Папка пуста" />
    <div v-else class="file-grid" :class="{ 'is-drop-target': dropActive }">
      <button
        v-for="(item, index) in sortedItems" :key="itemKey(item)" type="button"
        class="file-tile" :class="{ 'is-selected': selected.has(itemKey(item)), 'is-disabled': picker && item.kind === 'file' && !matchesPicker(item) }"
        draggable="true" @click="choose(item, $event, index)" @dblclick="activate(item)"
        @contextmenu="showContext($event, item)" @dragstart="startInternalDrag($event, item)"
        @dragover.prevent @drop.stop="handleDrop($event, item.kind === 'folder' ? item.id : (listing?.folder?.id ?? null))"
      >
        <component :is="item.kind === 'folder' ? Folder : isImage(item) ? Picture : Document" class="file-tile-icon" />
        <span class="file-tile-name" :title="item.name">{{ item.name }}</span>
        <span class="file-tile-extra">{{ item.kind === 'folder' ? `${item.item_count ?? 0} эл.` : formatSize(item.size ?? 0) }}</span>
      </button>
    </div>

    <footer class="file-statusbar">
      <span class="file-status-text" :title="selectedSummary">{{ selectedSummary }}</span>
      <el-button v-if="permissions.delete && selectedItems.length" text type="danger" :icon="Delete" @click="remove()">Удалить</el-button>
      <el-button v-if="picker" type="primary" :disabled="!selectableItem || !matchesPicker(selectableItem)" @click="confirmSelection">Выбрать</el-button>
    </footer>

    <div v-if="contextMenu" class="file-context-menu" :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }" @click.stop>
      <button v-if="contextMenu.item.kind === 'folder'" type="button" @click="activate(contextMenu.item); closeContextMenu()">Открыть</button>
      <button v-if="contextMenu.item.kind === 'file'" type="button" @click="download(contextMenu.item); closeContextMenu()"><Download /> Скачать</button>
      <button v-if="permissions.update" type="button" @click="rename(contextMenu.item); closeContextMenu()">Переименовать</button>
      <button v-if="permissions.update" type="button" @click="openMoveDialog"><Rank /> Переместить</button>
      <button v-if="permissions.delete" type="button" class="danger" @click="remove([contextMenu.item]); closeContextMenu()">Удалить</button>
    </div>

    <folder-move-dialog
      v-if="disk && moveVisible"
      v-model="moveVisible"
      :access-token="accessToken"
      :disk="disk"
      :items="moveItems"
      @confirm="confirmMove"
    />

    <el-dialog v-model="previewVisible" :title="previewName" width="min(900px, 90vw)" @closed="revokePreview">
      <img :src="previewURL" :alt="previewName" class="file-preview-image" />
    </el-dialog>
  </div>
</template>
