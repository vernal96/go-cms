<script setup lang="ts">
defineProps<{ html: string; title?: string }>()
</script>

<template>
  <iframe
    class="mail-html-preview"
    sandbox=""
    :srcdoc="html"
    :title="title ?? 'Предпросмотр HTML-письма'"
  />
</template>

<style scoped>
.mail-html-preview {
  width: 100%;
  min-height: 360px;
  border: 1px solid var(--el-border-color);
  border-radius: 8px;
  background: white;
}
</style>
