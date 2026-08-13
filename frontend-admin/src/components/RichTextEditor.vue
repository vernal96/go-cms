<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import Editor from '@tinymce/tinymce-vue'

defineProps<{ modelValue: string; disabled?: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const dark = ref(document.documentElement.dataset.theme === 'dark')
const ready = ref(false)
const lightSkin = ref('')
const darkSkin = ref('')
const lightContent = ref('')
const darkContent = ref('')
let observer: MutationObserver | null = null
let skinElement: HTMLStyleElement | null = null
let editorInstance: {
  getDoc(): Document
} | null = null

const init = computed(() => ({
  license_key: 'gpl',
  height: 480,
  menubar: false,
  branding: false,
  promotion: false,
  skin: false,
  content_css: false,
  content_style: dark.value
    ? `${darkContent.value}\nbody { background: #1d1e1f; color: #e5eaf3; }`
    : lightContent.value,
  plugins: 'advlist autolink code fullscreen link lists searchreplace table wordcount',
  toolbar:
    'undo redo | blocks | bold italic underline | bullist numlist | link table | searchreplace code fullscreen',
}))

function applySkin(): void {
  dark.value = document.documentElement.dataset.theme === 'dark'
  if (!skinElement) {
    skinElement = document.createElement('style')
    skinElement.dataset.adminTinyMceSkin = 'true'
    document.head.appendChild(skinElement)
  }
  skinElement.textContent = dark.value ? darkSkin.value : lightSkin.value
  applyContentTheme()
}

function applyContentTheme(): void {
  const document = editorInstance?.getDoc()
  if (!document) return
  let style = document.querySelector<HTMLStyleElement>('style[data-admin-content-skin]')
  if (!style) {
    style = document.createElement('style')
    style.dataset.adminContentSkin = 'true'
    document.head.appendChild(style)
  }
  style.textContent = dark.value
    ? `${darkContent.value}\nbody { background: #1d1e1f; color: #e5eaf3; }`
    : `${lightContent.value}\nbody { background: #fff; color: #303133; }`
}

function handleInit(_event: unknown, editor: { getDoc(): Document }): void {
  editorInstance = editor
  applyContentTheme()
}

async function loadTinyMCE(): Promise<void> {
	await import('tinymce/tinymce')
  const modules = await Promise.all([
    import('tinymce/icons/default'),
    import('tinymce/themes/silver'),
    import('tinymce/models/dom'),
    import('tinymce/plugins/advlist'),
    import('tinymce/plugins/autolink'),
    import('tinymce/plugins/code'),
    import('tinymce/plugins/fullscreen'),
    import('tinymce/plugins/link'),
    import('tinymce/plugins/lists'),
    import('tinymce/plugins/searchreplace'),
    import('tinymce/plugins/table'),
    import('tinymce/plugins/wordcount'),
    import('tinymce/skins/ui/oxide/skin.css?inline'),
    import('tinymce/skins/ui/oxide-dark/skin.css?inline'),
    import('tinymce/skins/content/default/content.css?inline'),
    import('tinymce/skins/content/dark/content.css?inline'),
  ])
  lightSkin.value = modules[12].default
  darkSkin.value = modules[13].default
  lightContent.value = modules[14].default
  darkContent.value = modules[15].default
  ready.value = true
  applySkin()
}

onMounted(() => {
  void loadTinyMCE()
  observer = new MutationObserver(applySkin)
  observer.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['data-theme'],
  })
})

onBeforeUnmount(() => {
  observer?.disconnect()
  skinElement?.remove()
  editorInstance = null
})
</script>

<template>
  <Editor
    v-if="ready"
    license-key="gpl"
    :model-value="modelValue"
    :disabled="disabled"
    :init="init"
    @init="handleInit"
    @update:model-value="emit('update:modelValue', $event)"
  />
  <div v-else class="rich-text-editor-loading">Загрузка редактора…</div>
</template>
