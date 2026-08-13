<script setup lang="ts">
import AccessDeniedView from '../components/AccessDeniedView.vue'
import FileExplorer from '../components/files/FileExplorer.vue'

const props = defineProps<{
  accessToken: string
  permissions: ReadonlySet<string>
}>()
</script>

<template>
  <section class="workspace-page filesystem-page">
    <header class="page-header">
      <div>
        <h1>Файловая система</h1>
        <p>Файлы публичного и приватного хранилищ</p>
      </div>
    </header>
    <access-denied-view v-if="!permissions.has('core.file.read')" />
    <file-explorer
      v-else
      :access-token="props.accessToken"
      :permissions="props.permissions"
    />
  </section>
</template>
