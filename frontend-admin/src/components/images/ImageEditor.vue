<script setup lang="ts">
import { nextTick, onBeforeUnmount, reactive, ref, watch } from 'vue'
import { ElAlert, ElButton, ElCheckbox, ElDialog, ElInputNumber } from 'element-plus'
import Cropper from 'cropperjs'
import 'cropperjs/dist/cropper.css'
import { adminBlob } from '../../api/admin-api'
import { defaultTransform, imageState, restoreImage, saveImage, type ImageState } from './image-api'

const props = defineProps<{ accessToken: string; baseUrl: string; sourceUrl?: string; currentUrl?: string }>()
const visible = defineModel<boolean>({ required: true })
const emit = defineEmits<{ saved: [state: ImageState] }>()
const state = ref<ImageState | null>(null)
const source = ref('')
const current = ref('')
const img = ref<HTMLImageElement>()
const error = ref('')
const busy = ref(false)
const preserve = ref(true)
const positions = [{v:'center',l:'Центр'},{v:'top',l:'Сверху'},{v:'bottom',l:'Снизу'},{v:'left',l:'Слева'},{v:'right',l:'Справа'},{v:'top-left',l:'Слева сверху'},{v:'top-right',l:'Справа сверху'},{v:'bottom-left',l:'Слева снизу'},{v:'bottom-right',l:'Справа снизу'}]
const options = reactive(defaultTransform())
let cropper: Cropper | null = null
let generation = 0

function dispose() {
  generation++
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
  } catch (e) { error.value = e instanceof Error ? e.message : 'Не удалось открыть редактор.' }
  finally { if (generation === version) busy.value = false }
}
function initializeCropper() {
  if (!img.value || cropper) return
  cropper = new Cropper(img.value, {
    viewMode: 1, autoCropArea: 1, checkOrientation: true,
    ready() {
      const t = state.value?.transform
      if (t) cropper?.setData({ ...(t.crop ? { x: t.crop.x, y: t.crop.y, width: t.crop.width, height: t.crop.height } : {}), rotate: t.rotate, scaleX: t.scale_x, scaleY: t.scale_y })
    },
  })
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
function reset() { cropper?.reset(); Object.assign(options, defaultTransform()); error.value = '' }
async function save() {
  if (!state.value || !cropper || busy.value) return
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
      <div class="image-editor-canvas"><img v-if="source" ref="img" :src="source" alt="Оригинал для редактирования" /></div>
      <aside class="image-editor-settings">
        <img v-if="current" :src="current" alt="Сохранённое изображение" class="image-editor-current" />
        <div class="image-editor-tools">
          <el-button @click="cropper?.rotate(-90)">↶ 90°</el-button><el-button @click="cropper?.rotate(90)">↷ 90°</el-button>
          <el-button @click="cropper?.zoom(0.1)">Масштаб +</el-button><el-button @click="cropper?.zoom(-0.1)">Масштаб −</el-button>
          <el-button @click="flip('x')">Отразить ↔</el-button><el-button @click="flip('y')">Отразить ↕</el-button>
        </div>
        <label>Ширина (0 — автоматически)<el-input-number v-model="options.width" :min="0" :max="state?.limits.output_dimension ?? 4096" @change="(v) => resize('width', v)" /></label>
        <label>Высота (0 — автоматически)<el-input-number v-model="options.height" :min="0" :max="state?.limits.output_dimension ?? 4096" @change="(v) => resize('height', v)" /></label>
        <el-checkbox v-model="preserve">Сохранять пропорции размеров</el-checkbox>
        <label>Размещение<select class="image-editor-select" v-model="options.fit"><option value="contain">Вместить целиком</option><option value="cover">Заполнить с обрезкой</option><option value="stretch">Растянуть</option></select></label>
        <label>Позиция<select class="image-editor-select" v-model="options.position"><option v-for="p in positions" :key="p.v" :value="p.v">{{ p.l }}</option></select></label>
        <label>Качество JPEG<el-input-number v-model="options.quality" :min="state?.limits.min_quality ?? 1" :max="state?.limits.max_quality ?? 100" /></label>
      </aside>
    </div>
    <template #footer><el-button v-if="state?.can_restore" :disabled="busy" @click="restore">Восстановить оригинал</el-button><el-button :disabled="busy" @click="reset">Сбросить изменения</el-button><el-button :disabled="busy" @click="visible = false">Отмена</el-button><el-button type="primary" :loading="busy" :disabled="!state?.editable || !source" @click="save">Сохранить</el-button></template>
  </el-dialog>
</template>
<style scoped>
.image-editor-layout { display:grid; grid-template-columns:minmax(0,1fr) 255px; gap:20px; margin-top:12px; }
.image-editor-canvas { height:min(55vh,520px); min-height:220px; overflow:hidden; background:#eee; }
.image-editor-canvas img { display:block; max-width:100%; }
.image-editor-settings { display:flex; flex-direction:column; gap:10px; max-height:55vh; overflow:auto; }
.image-editor-settings label { display:flex; flex-direction:column; gap:4px; }
.image-editor-tools { display:flex; flex-wrap:wrap; gap:4px; }
.image-editor-tools .el-button { margin:0; }
.image-editor-select { padding:8px; border:1px solid var(--el-border-color); border-radius:4px; color:var(--el-text-color-primary); background:var(--el-bg-color); }
.image-editor-current { height:90px; max-width:100%; object-fit:contain; }
@media(max-width:700px) { .image-editor-layout { grid-template-columns:1fr; } .image-editor-settings { max-height:240px; } }
</style>
