<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, provide, ref, toRef, watch } from 'vue'
import {
  ElAside,
  ElAvatar,
  ElButton,
  ElContainer,
  ElDropdown,
  ElDropdownItem,
  ElDropdownMenu,
  ElHeader,
  ElIcon,
  ElMain,
  ElMessage,
  ElPopover,
  ElScrollbar,
  ElTooltip,
} from 'element-plus'
import { ArrowDown, ArrowRightBold, Brush, Check, Moon, Platform, Sunny, UserFilled } from '@element-plus/icons-vue'
import { RouterLink, RouterView, useRouter } from 'vue-router'

import { AdminAPIError, adminBlob, adminRequest } from '../api/admin-api'
import { useSelectedSite } from '../composables/use-selected-site'
import { useAdminNavigation } from '../composables/use-admin-navigation'
import AdminNavigation from './AdminNavigation.vue'
import ResourceTree from './ResourceTree.vue'
import SiteSelector from './SiteSelector.vue'
import { adminAccessTokenKey, adminPermissionsKey } from '../admin-context'
import { accentColorOptions, applyAccentColor, applyAppearance, applyColorScheme } from '../theme'
import type { AccentColor, AdminUser, ColorScheme } from '../types/auth'

const props = defineProps<{
  user: AdminUser
  accessToken: string
  permissions: ReadonlySet<string>
}>()

const emit = defineEmits<{ logout: []; profileUpdated: [] }>()
const selected = useSelectedSite()
const navigation = useAdminNavigation()
const router = useRouter()
provide(adminAccessTokenKey, toRef(props, 'accessToken'))
provide(adminPermissionsKey, toRef(props, 'permissions'))
const avatarURL = ref('')
const darkTheme = ref(document.documentElement.dataset.theme === 'dark')
const selectedScheme = ref<ColorScheme>(props.user.color_scheme)
const selectedAccent = ref<AccentColor>(props.user.accent_color)
const savingPreference = ref<'theme' | 'accent' | null>(null)
const savingPreferences = computed(() => savingPreference.value !== null)
const sidebarWidth = ref(320)
const sidebarCollapsed = ref(false)
const resizingSidebar = ref(false)
let themeObserver: MutationObserver | null = null
let sidebarResizeStartWidth = 320
const sidebarStorageKey = 'admin.resource-sidebar'
const sidebarMin = 260
const sidebarMax = 520
const sidebarCollapseThreshold = 220

function loadSidebarState(): void {
  try {
    const value = JSON.parse(localStorage.getItem(sidebarStorageKey) ?? '{}') as { width?: number; collapsed?: boolean }
    if (typeof value.width === 'number') sidebarWidth.value = Math.min(sidebarMax, Math.max(sidebarMin, value.width))
    sidebarCollapsed.value = value.collapsed === true
  } catch {
    sidebarWidth.value = 320
    sidebarCollapsed.value = false
  }
}

function saveSidebarState(): void {
  localStorage.setItem(sidebarStorageKey, JSON.stringify({ width: sidebarWidth.value, collapsed: sidebarCollapsed.value }))
}

function startSidebarResize(event: PointerEvent): void {
  if (sidebarCollapsed.value) return
  sidebarResizeStartWidth = sidebarWidth.value
  resizingSidebar.value = true
  ;(event.currentTarget as HTMLElement).setPointerCapture(event.pointerId)
}

function resizeSidebar(event: PointerEvent): void {
  if (!resizingSidebar.value) return
  const bodyLeft = document.querySelector<HTMLElement>('.admin-body')?.getBoundingClientRect().left ?? 0
  const requested = event.clientX - bodyLeft
  if (requested < sidebarCollapseThreshold) {
    sidebarWidth.value = sidebarResizeStartWidth
    sidebarCollapsed.value = true
    resizingSidebar.value = false
    saveSidebarState()
    return
  }
  sidebarWidth.value = Math.min(sidebarMax, Math.max(sidebarMin, requested))
}

function finishSidebarResize(): void {
  if (!resizingSidebar.value) return
  resizingSidebar.value = false
  saveSidebarState()
}

function resizeSidebarWithKeyboard(event: KeyboardEvent): void {
  if (event.key !== 'ArrowLeft' && event.key !== 'ArrowRight' && event.key !== 'Home' && event.key !== 'End') return
  event.preventDefault()
  if (event.key === 'Home') {
    sidebarCollapsed.value = true
  } else if (event.key === 'End') {
    sidebarCollapsed.value = false
    sidebarWidth.value = sidebarMax
  } else {
    const next = sidebarWidth.value + (event.key === 'ArrowRight' ? 20 : -20)
    if (next < sidebarMin) sidebarCollapsed.value = true
    else {
      sidebarCollapsed.value = false
      sidebarWidth.value = Math.min(sidebarMax, next)
    }
  }
  saveSidebarState()
}

function expandSidebar(): void {
  sidebarCollapsed.value = false
  saveSidebarState()
}

function can(code: string): boolean {
  return props.permissions.has(code)
}

function handleUserCommand(command: string): void {
  if (command === 'logout') emit('logout')
  if (command === 'profile') void router.push('/admin/profile')
}

function handleAPIError(error: unknown): void {
  if (!(error instanceof AdminAPIError)) return
  if (error.status === 401) emit('logout')
  else if (error.status === 403 || error.status === 404) selected.clearSelected()
}

watch(
  () => [props.accessToken, selected.selectedSite.value?.id ?? null] as const,
  ([accessToken, siteID]) => {
    void navigation.refresh(accessToken, siteID).catch(handleAPIError)
  },
  { immediate: true },
)

onMounted(() => {
  loadSidebarState()
  syncTheme()
  themeObserver = new MutationObserver(syncTheme)
  themeObserver.observe(document.documentElement, {
    attributes: true,
    attributeFilter: ['data-theme'],
  })
  window.addEventListener('pointermove', resizeSidebar)
  window.addEventListener('pointerup', finishSidebarResize)
  if (can('core.site.read')) {
    void selected.initialize(props.accessToken).catch(handleAPIError)
  } else {
    selected.clearSelected()
  }
})
watch(
  () => [props.user.has_avatar, props.user.avatar_updated_at, props.accessToken] as const,
  () => void loadAvatar(),
  { immediate: true },
)
watch(
  () => [props.user.color_scheme, props.user.accent_color] as const,
  ([scheme, accent]) => {
    selectedScheme.value = scheme
    selectedAccent.value = accent
  },
)
onBeforeUnmount(() => {
  themeObserver?.disconnect()
  themeObserver = null
  window.removeEventListener('pointermove', resizeSidebar)
  window.removeEventListener('pointerup', finishSidebarResize)
  selected.reset()
  navigation.dispose()
  revokeAvatar()
})

async function loadAvatar(): Promise<void> {
  revokeAvatar()
  if (!props.user.has_avatar) return
  try {
    const blob = await adminBlob('/api/admin/profile/avatar/preview', props.accessToken)
    avatarURL.value = URL.createObjectURL(blob)
  } catch (error) {
    handleAPIError(error)
  }
}

function revokeAvatar(): void {
  if (avatarURL.value) URL.revokeObjectURL(avatarURL.value)
  avatarURL.value = ''
}

function syncTheme(): void {
  darkTheme.value = document.documentElement.dataset.theme === 'dark'
}

async function toggleTheme(): Promise<void> {
  if (savingPreferences.value) return
  const previous = selectedScheme.value
  const next: ColorScheme = darkTheme.value ? 'light' : 'dark'
  selectedScheme.value = next
  savingPreference.value = 'theme'
  applyColorScheme(next)
  try {
    await persistPreferences(next, selectedAccent.value)
    emit('profileUpdated')
  } catch (error) {
    selectedScheme.value = previous
    applyAppearance(previous, selectedAccent.value)
    ElMessage.error(error instanceof Error ? error.message : 'Не удалось сохранить тему.')
  } finally {
    savingPreference.value = null
  }
}

async function selectAccent(accent: AccentColor): Promise<void> {
  if (savingPreferences.value || accent === selectedAccent.value) return
  const previous = selectedAccent.value
  selectedAccent.value = accent
  savingPreference.value = 'accent'
  applyAccentColor(accent)
  try {
    await persistPreferences(selectedScheme.value, accent)
    emit('profileUpdated')
  } catch (error) {
    selectedAccent.value = previous
    applyAccentColor(previous)
    ElMessage.error(error instanceof Error ? error.message : 'Не удалось сохранить акцентный цвет.')
  } finally {
    savingPreference.value = null
  }
}

async function persistPreferences(colorScheme: ColorScheme, accentColor: AccentColor): Promise<void> {
  await adminRequest('/api/admin/profile/preferences', props.accessToken, {
    method: 'PUT',
    body: JSON.stringify({ color_scheme: colorScheme, accent_color: accentColor }),
  })
}
</script>

<template>
  <el-container class="admin-shell">
    <el-header class="topbar" height="64px">
      <router-link
        to="/admin/dashboard"
        class="brand-mark"
        aria-label="На главную"
      >
        <el-icon :size="24"><Platform /></el-icon>
      </router-link>

      <div class="topbar-main">
        <admin-navigation :items="navigation.items.value" />
        <el-dropdown placement="bottom-end" trigger="click" @command="handleUserCommand">
          <el-button class="user-control" text aria-label="Открыть меню пользователя">
            <el-avatar :size="34" :src="avatarURL" :icon="avatarURL ? undefined : UserFilled" />
            <span class="user-name">{{ user.display_name }}</span>
            <el-icon class="user-chevron"><ArrowDown /></el-icon>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item command="profile">Профиль</el-dropdown-item>
              <el-dropdown-item command="logout" divided>Выйти</el-dropdown-item>
            </el-dropdown-menu>
          </template>
        </el-dropdown>
      </div>
    </el-header>

    <el-container class="admin-body" :class="{ 'sidebar-is-collapsed': sidebarCollapsed, 'sidebar-is-resizing': resizingSidebar }">
      <el-aside
        class="resource-sidebar"
        :class="{ 'is-collapsed': sidebarCollapsed }"
        :width="sidebarCollapsed ? '0px' : `${sidebarWidth}px`"
      >
        <div class="sidebar-site-selector">
          <div class="sidebar-heading">Сайт</div>
          <site-selector
            v-if="can('core.site.read')"
            :access-token="accessToken"
            :can-create="can('core.site.create')"
            @error="handleAPIError"
          />
          <div v-else class="sidebar-empty sidebar-site-empty">Нет доступа к сайтам</div>
        </div>
        <el-scrollbar class="resource-scrollbar">
          <resource-tree
            v-if="can('core.site.read') && can('core.resource.read')"
            :access-token="accessToken"
            :can-create="can('core.resource.create')"
            :can-update="can('core.resource.update')"
            :can-delete="can('core.resource.delete')"
            @error="handleAPIError"
          />
          <div v-else class="sidebar-empty">
            {{ can('core.site.read') ? 'Нет доступа к ресурсам' : 'Нет доступа к сайтам' }}
          </div>
        </el-scrollbar>
        <footer class="sidebar-theme-footer">
          <el-popover placement="top-start" :width="228" trigger="click">
            <template #reference>
              <el-button
                class="sidebar-accent-toggle"
                circle
                :disabled="savingPreferences"
                :loading="savingPreference === 'accent'"
                :icon="Brush"
                aria-label="Выбрать акцентный цвет"
                title="Выбрать акцентный цвет"
              />
            </template>
            <div class="accent-palette" role="listbox" aria-label="Акцентный цвет">
              <el-tooltip
                v-for="option in accentColorOptions"
                :key="option.code"
                :content="option.label"
                placement="top"
              >
                <button
                  type="button"
                  class="accent-swatch"
                  :class="{ 'is-active': selectedAccent === option.code }"
                  :style="{ backgroundColor: option.color }"
                  role="option"
                  :aria-label="option.label"
                  :aria-selected="selectedAccent === option.code"
                  :disabled="savingPreferences"
                  @click="selectAccent(option.code)"
                >
                  <el-icon v-if="selectedAccent === option.code"><Check /></el-icon>
                </button>
              </el-tooltip>
            </div>
          </el-popover>
          <el-button
            class="sidebar-theme-toggle"
            circle
            :disabled="savingPreferences"
            :loading="savingPreference === 'theme'"
            :icon="darkTheme ? Sunny : Moon"
            :aria-label="darkTheme ? 'Включить светлую тему' : 'Включить тёмную тему'"
            :title="darkTheme ? 'Включить светлую тему' : 'Включить тёмную тему'"
            @click="toggleTheme"
          />
        </footer>
        <div
          class="sidebar-resize-handle"
          role="separator"
          aria-label="Изменить ширину сайдбара"
          aria-orientation="vertical"
          :aria-valuemin="sidebarMin"
          :aria-valuemax="sidebarMax"
          :aria-valuenow="sidebarWidth"
          tabindex="0"
          @pointerdown.prevent="startSidebarResize"
          @keydown="resizeSidebarWithKeyboard"
        />
      </el-aside>

      <el-button
        v-if="sidebarCollapsed"
        class="sidebar-expand-button"
        circle
        :icon="ArrowRightBold"
        aria-label="Развернуть сайдбар"
        title="Развернуть сайдбар"
        @click="expandSidebar"
      />

      <el-main class="workspace" aria-label="Рабочая область">
        <router-view v-slot="{ Component }">
          <component
            :is="Component"
            :access-token="accessToken"
            :permissions="permissions"
            @unauthorized="emit('logout')"
            @updated="emit('profileUpdated')"
          />
        </router-view>
      </el-main>
    </el-container>
  </el-container>
</template>
