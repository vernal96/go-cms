<script setup lang="ts">
import { nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { ElAlert, ElButton, ElCheckbox, ElDialog, ElInputNumber } from 'element-plus'
import Cropper from 'cropperjs'
import 'cropperjs/dist/cropper.css'
import { adminBlob } from '../../api/admin-api'
import { defaultTransform, imageState, restoreImage, saveImage, type ImageState } from './image-api'
import { editorView, fitScale, imageSize, rotateSelection, type Size } from './image-editor-geometry'

const props = defineProps<{ accessToken: string; baseUrl: string; sourceUrl?: string; currentUrl?: string }>()
const visible = defineModel<boolean>({ required: true })
const emit = defineEmits<{ saved: [state: ImageState] }>()
const state = ref<ImageState | null>(null)
const source = ref('')
const current = ref('')
const img = ref<HTMLImageElement>()
const viewport = ref<HTMLDivElement>()
const stageSize = ref<Size | null>(null)
const editorReady = ref(false)
const error = ref('')
const busy = ref(false)
const preserve = ref(true)
const positions = [{v:'center',l:'Центр'},{v:'top',l:'Сверху'},{v:'bottom',l:'Снизу'},{v:'left',l:'Слева'},{v:'right',l:'Справа'},{v:'top-left',l:'Слева сверху'},{v:'top-right',l:'Справа сверху'},{v:'bottom-left',l:'Слева снизу'},{v:'bottom-right',l:'Справа снизу'}]
const options = reactive(defaultTransform())
let cropper: Cropper | null = null
let generation = 0
let pendingView: { data: Cropper.Data; zoom: number } | null = null
let viewportSize: Size | null = null
let resizeObserver: ResizeObserver | null = null

function dispose() {
  generation++
  resizeObserver?.disconnect(); resizeObserver = null
  editorReady.value = false; pendingView = null; stageSize.value = null; viewportSize = null
  cropper?.destroy(); cropper = null
  for (const url of [source.value, current.value]) if (url) URL.revokeObjectURL(url)
  source.value = ''; current.value = ''
}
onBeforeUnmount(dispose)
watch(visible, (value) => { if (value) void open(); else dispose() })
async function open() {
  dispose(); const version = generation
  error.value = ''; busy.value = true
  try {
    const loaded = await imageState(props.baseUrl, props.accessToken)
    if (generation !== version) return
    state.value = loaded
    if (!loaded.editable) { error.value = 'Редактор поддерживает JPEG и статические PNG.'; return }
    Object.assign(options, defaultTransform(), loaded.transform ?? {})
    const [original, saved] = await Promise.all([
      adminBlob(props.sourceUrl ?? `/api/files/${loaded.original_file.id}/preview`, props.accessToken),
      adminBlob(props.currentUrl ?? `/api/files/${loaded.current_file.id}/thumbnail?width=256&height=256&fit=contain`, props.accessToken),
    ])
    if (generation !== version) return
    source.value = URL.createObjectURL(original); current.value = URL.createObjectURL(saved)
    await nextTick()
    initializeCropper()
    if (viewport.value) {
      resizeObserver = new ResizeObserver(() => {
        const size = readViewport()
        if (!size || !viewportSize || !editorReady.value || !cropper) return
        if (size.width === viewportSize.width && size.height === viewportSize.height) return
        const data = cropper.getData()
        const zoom = relativeZoom()
        void applyView(data, zoom)
      })
      resizeObserver.observe(viewport.value)
    }
  } catch (e) { error.value = e instanceof Error ? e.message : 'Не удалось открыть редактор.' }
  finally { if (generation === version) busy.value = false }
}
function initializeCropper() {
  if (!img.value || cropper) return
  const version = generation
  cropper = new Cropper(img.value, {
    viewMode: 1, autoCropArea: 1, checkOrientation: true, responsive: false,
    minContainerWidth: 1, minContainerHeight: 1,
    ready() {
      if (generation !== version || !cropper) return
      if (pendingView) {
        void applyView(pendingView.data, pendingView.zoom)
        return
      }
      const t = state.value?.transform
      const data = { ...cropper.getData(), rotate: t?.rotate ?? 0, scaleX: t?.scale_x ?? 1, scaleY: t?.scale_y ?? 1 }
      const size = transformedSize(data)
      Object.assign(data, t?.crop ?? { x: 0, y: 0, ...size })
      void applyView(data, 1)
    },
  })
}
function readViewport(): Size | null {
  if (!viewport.value?.clientWidth || !viewport.value.clientHeight) return null
  return { width: viewport.value.clientWidth, height: viewport.value.clientHeight }
}
function transformedSize(data: Cropper.Data): Size {
  const image = cropper!.getImageData()
  return imageSize({ width: image.naturalWidth, height: image.naturalHeight }, data)
}
function relativeZoom() {
  if (!cropper || !viewportSize) return 1
  const canvas = cropper.getCanvasData()
  return (canvas.width / canvas.naturalWidth) / fitScale({ width: canvas.naturalWidth, height: canvas.naturalHeight }, viewportSize)
}
async function applyView(data: Cropper.Data, zoom: number) {
  const size = readViewport()
  if (!cropper || !size) return
  editorReady.value = false
  viewportSize = size
  const view = editorView(data, transformedSize(data), size, zoom)
  const container = cropper.getContainerData()
  if (container.width !== view.stage.width || container.height !== view.stage.height) {
    // Rebuild only when the internal stage needs a different size. The blob is
    // already loaded; no API request or image re-encoding is involved.
    pendingView = { data, zoom }
    stageSize.value = view.stage
    cropper.destroy(); cropper = null
    const version = generation
    await nextTick()
    if (generation === version) initializeCropper()
    return
  }
  cropper.clear()
  cropper.setData({ rotate: data.rotate, scaleX: data.scaleX, scaleY: data.scaleY })
  cropper.setCanvasData(view.canvas)
  cropper.crop()
  cropper.setData(data)
  pendingView = null
  editorReady.value = true
}
function rotate(angle: 90 | -90) {
  if (!cropper || !editorReady.value) return
  const data = cropper.getData()
  const zoom = relativeZoom()
  const rotated = rotateSelection(data, transformedSize(data), angle)
  const width = options.width
  options.width = options.height
  options.height = width
  void applyView(rotated, zoom)
}
function resize(axis: 'width' | 'height', value: number | undefined) {
  if (!value || !preserve.value || !cropper) return
  const data = cropper.getData()
  if (data.width <= 0 || data.height <= 0) return
  if (axis === 'width') options.height = Math.max(1, Math.round(value * data.height / data.width))
  else options.width = Math.max(1, Math.round(value * data.width / data.height))
}
function flip(axis: 'x' | 'y') {
  const data = cropper?.getData(); if (!data) return
  if (axis === 'x') cropper?.scaleX(-(data.scaleX ?? 1)); else cropper?.scaleY(-(data.scaleY ?? 1))
}
function reset() {
  if (!cropper || !editorReady.value) return
  const data = { ...cropper.getData(), x: 0, y: 0, rotate: 0, scaleX: 1, scaleY: 1 }
  Object.assign(data, transformedSize(data))
  Object.assign(options, defaultTransform()); error.value = ''
  void applyView(data, 1)
}
async function save() {
  if (!state.value || !cropper || !editorReady.value || busy.value) return
  busy.value = true; error.value = ''
  try {
    const data = cropper.getData(true)
    const transform = { ...options, rotate: ((Math.round(data.rotate ?? 0) % 360) + 360) % 360, scale_x: data.scaleX ?? 1, scale_y: data.scaleY ?? 1, crop: { x: Math.round(data.x), y: Math.round(data.y), width: Math.round(data.width), height: Math.round(data.height) } }
    const updated = await saveImage(props.baseUrl, props.accessToken, state.value, transform)
    emit('saved', updated); visible.value = false
  } catch (e) { error.value = e instanceof Error ? e.message : 'Не удалось сохранить изображение.' }
  finally { busy.value = false }
}
async function restore() {
  if (!state.value || busy.value) return
  busy.value = true; error.value = ''
  try { const updated = await restoreImage(props.baseUrl, props.accessToken, state.value); state.value = updated; emit('saved', updated); await open() }
  catch (e) { error.value = e instanceof Error ? e.message : 'Не удалось восстановить оригинал.' }
  finally { busy.value = false }
}
</script>
<template>
  <el-dialog v-model="visible" title="Редактор изображения" width="min(1080px, 95vw)" :close-on-click-modal="false" :close-on-press-escape="!busy" :show-close="!busy" @opened="initializeCropper">
    <el-alert v-if="error" type="error" :title="error" :closable="false" />
    <div class="image-editor-layout">
      <div ref="viewport" class="image-editor-canvas"><div class="image-editor-stage" :style="stageSize ? { width: `${stageSize.width}px`, height: `${stageSize.height}px` } : undefined"><img v-if="source" ref="img" :src="source" alt="Оригинал для редактирования" /></div></div>
      <aside class="image-editor-settings">
        <img v-if="current" :src="current" alt="Сохранённое изображение" class="image-editor-current" />
        <div class="image-editor-tools">
          <el-button :disabled="!editorReady" @click="rotate(-90)">↶ 90°</el-button><el-button :disabled="!editorReady" @click="rotate(90)">↷ 90°</el-button>
          <el-button :disabled="!editorReady" @click="cropper?.zoom(0.1)">Масштаб +</el-button><el-button :disabled="!editorReady" @click="cropper?.zoom(-0.1)">Масштаб −</el-button>
          <el-button :disabled="!editorReady" @click="flip('x')">Отразить ↔</el-button><el-button :disabled="!editorReady" @click="flip('y')">Отразить ↕</el-button>
        </div>
        <label>Ширина (0 — автоматически)<el-input-number v-model="options.width" :min="0" :max="state?.limits.output_dimension ?? 4096" @change="(v) => resize('width', v)" /></label>
        <label>Высота (0 — автоматически)<el-input-number v-model="options.height" :min="0" :max="state?.limits.output_dimension ?? 4096" @change="(v) => resize('height', v)" /></label>
        <el-checkbox v-model="preserve">Сохранять пропорции размеров</el-checkbox>
        <label>Размещение<select class="image-editor-select" v-model="options.fit"><option value="contain">Вместить целиком</option><option value="cover">Заполнить с обрезкой</option><option value="stretch">Растянуть</option></select></label>
        <label>Позиция<select class="image-editor-select" v-model="options.position"><option v-for="p in positions" :key="p.v" :value="p.v">{{ p.l }}</option></select></label>
        <label>Качество JPEG<el-input-number v-model="options.quality" :min="state?.limits.min_quality ?? 1" :max="state?.limits.max_quality ?? 100" /></label>
      </aside>
    </div>
    <template #footer><el-button v-if="state?.can_restore" :disabled="busy" @click="restore">Восстановить оригинал</el-button><el-button :disabled="busy || !editorReady" @click="reset">Сбросить изменения</el-button><el-button :disabled="busy" @click="visible = false">Отмена</el-button><el-button type="primary" :loading="busy" :disabled="!state?.editable || !editorReady" @click="save">Сохранить</el-button></template>
  </el-dialog>
</template>
<style scoped>
.image-editor-layout { display:grid; grid-template-columns:minmax(0,1fr) 255px; gap:20px; margin-top:12px; }
.image-editor-canvas { position:relative; height:min(55vh,520px); min-height:220px; overflow:hidden; background:#eee; }
.image-editor-stage { position:absolute; width:100%; height:100%; left:50%; top:50%; transform:translate(-50%,-50%); }
.image-editor-canvas img { display:block; max-width:100%; }
.image-editor-settings { display:flex; flex-direction:column; gap:10px; max-height:55vh; overflow:auto; }
.image-editor-settings label { display:flex; flex-direction:column; gap:4px; }
.image-editor-tools { display:flex; flex-wrap:wrap; gap:4px; }
.image-editor-tools .el-button { margin:0; }
.image-editor-select { padding:8px; border:1px solid var(--el-border-color); border-radius:4px; color:var(--el-text-color-primary); background:var(--el-bg-color); }
.image-editor-current { height:90px; max-width:100%; object-fit:contain; }
@media(max-width:700px) { .image-editor-layout { grid-template-columns:1fr; } .image-editor-settings { max-height:240px; } }
</style>
