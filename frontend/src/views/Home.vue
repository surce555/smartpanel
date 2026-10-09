<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Icon } from '@iconify/vue'
import { useBookmarksStore } from '@/stores/bookmarks'
import { useSettingsStore } from '@/stores/settings'
import { useAuthStore } from '@/stores/auth'
import { useIPv6Probe } from '@/composables/useIPv6Probe'
import { Bookmark } from '@/stores/bookmarks'
import SearchBar from '@/components/SearchBar.vue'
import GroupSection from '@/components/GroupSection.vue'
import ThemeSwitcher from '@/components/ThemeSwitcher.vue'
import CardSizeSwitcher from '@/components/CardSizeSwitcher.vue'
import QuickBookmarkModal from '@/components/QuickBookmarkModal.vue'
import QuickTagModal from '@/components/QuickTagModal.vue'
import NetworkRouteBadge from '@/components/NetworkRouteBadge.vue'
import SpotlightSearch from '@/components/SpotlightSearch.vue'
import QuickMemoModal from '@/components/QuickMemoModal.vue'

const router = useRouter()
const bookmarksStore = useBookmarksStore()
const settingsStore = useSettingsStore()
const authStore = useAuthStore()
const { probeIPv6 } = useIPv6Probe()

// Modals state
const showBookmarkModal = ref<boolean>(false)
const showTagModal = ref<boolean>(false)
const showSpotlight = ref<boolean>(false)
const showMemoModal = ref<boolean>(false)
const isZenMode = ref<boolean>(false)
const zenTime = ref<string>('')
const zenDate = ref<string>('')
let zenTimer: any = null
let idleTimer: any = null
const editingBookmark = ref<Bookmark | null>(null)
const targetGroupId = ref<string>('')
const showLauncherMenu = ref<boolean>(false)

function isCardStyle(style: string): boolean {
  return settingsStore.cardStyle === style
}

function handleAddBookmark(groupId?: string) {
  if (!authStore.isAuthenticated) {
    router.push({ name: 'Login', query: { redirect: '/' } })
    return
  }
  editingBookmark.value = null
  targetGroupId.value = groupId || bookmarksStore.groups[0]?.id || ''
  showBookmarkModal.value = true
  showLauncherMenu.value = false
}

function handleEditBookmark(bookmark: Bookmark) {
  if (!authStore.isAuthenticated) {
    router.push({ name: 'Login', query: { redirect: '/' } })
    return
  }
  editingBookmark.value = bookmark
  targetGroupId.value = bookmark.group_id
  showBookmarkModal.value = true
}

function handleAddTag() {
  if (!authStore.isAuthenticated) {
    router.push({ name: 'Login', query: { redirect: '/' } })
    return
  }
  showTagModal.value = true
  showLauncherMenu.value = false
}

async function handleSelectThemePreset(preset: 'crystal' | 'glass' | 'solid' | 'minimal') {
  await settingsStore.applyPresetTheme(preset)
  showLauncherMenu.value = false
}

async function persistQuickAppearance() {
  localStorage.setItem('smartpanel_text_opacity', String(settingsStore.textOpacity))
  localStorage.setItem('smartpanel_content_top_offset', String(settingsStore.contentTopOffset))
  localStorage.setItem('smartpanel_show_group_titles', String(settingsStore.showGroupTitles))
  if (authStore.isAuthenticated) {
    try {
      await settingsStore.saveSettings({
        text_opacity: String(settingsStore.textOpacity),
        content_top_offset: String(settingsStore.contentTopOffset),
        show_group_titles: String(settingsStore.showGroupTitles),
      })
    } catch (e) {
      // ignore
    }
  }
}

async function toggleGroupTitles() {
  settingsStore.showGroupTitles = !settingsStore.showGroupTitles
  await persistQuickAppearance()
}

function handleOutsideClick(e: MouseEvent) {
  const target = e.target as HTMLElement
  if (!target.closest('.launcher-menu-container')) {
    showLauncherMenu.value = false
  }
}

watch(
  () => [settingsStore.requireLogin, authStore.isAuthenticated],
  ([reqLogin, isAuth]) => {
    if (reqLogin && !isAuth) {
      router.replace({ name: 'Login', query: { redirect: '/' } })
    }
  },
  { immediate: true }
)

function resetIdleTimer() {
  if (isZenMode.value) {
    isZenMode.value = false
  }
  if (idleTimer) clearTimeout(idleTimer)
  idleTimer = setTimeout(() => {
    if (!showBookmarkModal.value && !showTagModal.value && !showSpotlight.value && !showMemoModal.value) {
      isZenMode.value = true
    }
  }, 150000)
}

function updateZenClock() {
  const now = new Date()
  zenTime.value = now.toLocaleTimeString('zh-CN', { hour12: false })
  zenDate.value = now.toLocaleDateString('zh-CN', { month: 'long', day: 'numeric', weekday: 'long' })
}

function handleGlobalKeydown(e: KeyboardEvent) {
  const target = e.target as HTMLElement
  const isInput = target && (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable)

  if (isZenMode.value) {
    isZenMode.value = false
    return
  }

  if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    showSpotlight.value = !showSpotlight.value
    return
  }

  if (e.key === '/' && !isInput) {
    e.preventDefault()
    showSpotlight.value = true
    return
  }

  if ((e.key === 'z' || e.key === 'Z') && !isInput) {
    e.preventDefault()
    isZenMode.value = !isZenMode.value
    return
  }

  resetIdleTimer()
}

onMounted(async () => {
  if (settingsStore.requireLogin && !authStore.isAuthenticated) {
    router.replace({ name: 'Login', query: { redirect: '/' } })
    return
  }
  probeIPv6()
  window.addEventListener('click', handleOutsideClick)
  window.addEventListener('keydown', handleGlobalKeydown)
  window.addEventListener('mousemove', resetIdleTimer)
  window.addEventListener('mousedown', resetIdleTimer)
  window.addEventListener('touchstart', resetIdleTimer)
  window.addEventListener('scroll', resetIdleTimer)
  zenTimer = setInterval(updateZenClock, 1000)
  updateZenClock()
  resetIdleTimer()
  await bookmarksStore.fetchAll()
})

onUnmounted(() => {
  window.removeEventListener('click', handleOutsideClick)
  window.removeEventListener('keydown', handleGlobalKeydown)
  window.removeEventListener('mousemove', resetIdleTimer)
  window.removeEventListener('mousedown', resetIdleTimer)
  window.removeEventListener('touchstart', resetIdleTimer)
  window.removeEventListener('scroll', resetIdleTimer)
  if (idleTimer) clearTimeout(idleTimer)
  if (zenTimer) clearInterval(zenTimer)
})

function toggleTag(tagId: string) {
  if (bookmarksStore.selectedTagId === tagId) {
    bookmarksStore.selectedTagId = null
  } else {
    bookmarksStore.selectedTagId = tagId
  }
}
</script>

<template>
  <div
    class="relative min-h-[100dvh] flex flex-col justify-start p-3.5 sm:p-6 lg:p-8 pt-[max(1rem,env(safe-area-inset-top))] pb-[max(1rem,env(safe-area-inset-bottom))] max-w-7xl mx-auto transition-all duration-700"
    :class="{ 'opacity-0 pointer-events-none scale-95': isZenMode }"
  >
    
    <!-- ========================================== -->
    <!-- 1. FLOATING LAUNCHER (Transparent Mode)    -->
    <!-- Matching Image 4: Only top-right 4-grid ⊞  -->
    <!-- ========================================== -->
    <div
      v-if="settingsStore.cardStyle === 'transparent'"
      class="fixed top-4 sm:top-5 right-4 sm:right-5 z-40 launcher-menu-container"
    >
      <button
        type="button"
        @click.stop="showLauncherMenu = !showLauncherMenu"
        class="flex items-center justify-center p-2.5 rounded-xl bg-black/40 hover:bg-black/60 text-white/90 hover:text-white border border-white/10 shadow-lg cursor-pointer transition-all hover:scale-105 active:scale-95"
        title="快捷控制中心与系统管理"
      >
        <Icon icon="tabler:apps" class="w-5 h-5" />
      </button>
    </div>

    <!-- ========================================== -->
    <!-- 2. STANDARD HEADER (Non-Transparent Modes) -->
    <!-- ========================================== -->
    <header
      v-else
      class="flex items-center justify-between gap-3 mb-6 sm:mb-8"
    >
      <!-- Left: Logo & Network Indicator -->
      <div class="flex items-center gap-2.5 sm:gap-3">
        <div
          class="flex items-center justify-center w-9 h-9 sm:w-10 sm:h-10 rounded-2xl shadow-lg transition-transform hover:scale-105 bg-indigo-600 text-white shadow-indigo-600/30"
        >
          <Icon icon="tabler:layout-dashboard" class="w-5 h-5" />
        </div>
        <div>
          <div class="flex items-center gap-2">
            <h1 class="text-base sm:text-lg font-bold tracking-tight text-slate-900 dark:text-white">
              SmartPanel
            </h1>
            <NetworkRouteBadge />
          </div>
          <p class="text-[11px] hidden sm:block text-slate-500 dark:text-slate-400">
            智能双栈 NAS 个人仪表盘
          </p>
        </div>
      </div>

      <!-- Right: Controls & Launcher -->
      <div class="flex items-center gap-2 sm:gap-3">
        <!-- Desktop Controls -->
        <div class="hidden md:flex items-center gap-2.5">
          <CardSizeSwitcher />
          <ThemeSwitcher />

          <button
            v-if="authStore.isAuthenticated"
            type="button"
            @click="handleAddBookmark()"
            class="flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold shadow-md shadow-indigo-600/20 hover:scale-105 transition-all cursor-pointer"
            title="在首页快速添加卡片/书签"
          >
            <Icon icon="tabler:plus" class="w-4 h-4" />
            <span>添加卡片</span>
          </button>

          <router-link
            v-if="authStore.isAuthenticated"
            to="/admin"
            class="flex items-center gap-1.5 px-3.5 py-1.5 rounded-full bg-white/70 dark:bg-slate-800/70 backdrop-blur-md border border-slate-200/80 dark:border-slate-700/80 text-xs font-semibold text-slate-700 dark:text-slate-200 hover:text-indigo-600 dark:hover:text-indigo-400 shadow-xs transition-colors"
          >
            <Icon icon="tabler:settings" class="w-4 h-4" />
            <span>管理面板</span>
          </router-link>

          <router-link
            v-else
            to="/login"
            class="flex items-center gap-1.5 px-3.5 py-1.5 rounded-full bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold shadow-md shadow-indigo-600/20 transition-colors"
          >
            <Icon icon="tabler:login" class="w-4 h-4" />
            <span>登录</span>
          </router-link>
        </div>

        <!-- 4-Square Grid Launcher Button -->
        <div class="relative launcher-menu-container">
          <button
            type="button"
            @click.stop="showLauncherMenu = !showLauncherMenu"
            class="flex items-center justify-center p-2 sm:p-2.5 rounded-xl transition-all shadow-lg cursor-pointer bg-white/80 dark:bg-slate-800/80 text-slate-700 dark:text-slate-200 hover:text-indigo-600 border border-slate-200 dark:border-slate-700"
            title="快捷控制中心与主题设置"
          >
            <Icon icon="tabler:apps" class="w-5 h-5" />
          </button>
        </div>
      </div>
    </header>

    <!-- ========================================== -->
    <!-- 3. UNIFIED LAUNCHER POPOVER MENU           -->
    <!-- Positioned cleanly at top right            -->
    <!-- ========================================== -->
    <div
      v-if="showLauncherMenu"
      @click.stop
      class="launcher-menu-container fixed top-16 right-4 sm:right-6 w-72 sm:w-80 p-4 rounded-3xl bg-slate-900/95 backdrop-blur-2xl border border-white/20 shadow-2xl text-white text-xs z-50 space-y-3.5 animate-in fade-in zoom-in-95"
    >
      <!-- Popover Header -->
      <div class="flex items-center justify-between border-b border-white/10 pb-2.5">
        <div class="flex items-center gap-2">
          <span class="font-bold text-white text-sm flex items-center gap-1.5">
            <Icon icon="tabler:apps" class="w-4 h-4 text-indigo-400" />
            快捷控制中心
          </span>
          <NetworkRouteBadge />
        </div>
        <button @click="showLauncherMenu = false" class="text-white/60 hover:text-white p-1">✕</button>
      </div>

      <!-- Quick Search inside Popover -->
      <div class="relative">
        <input
          type="text"
          v-model="bookmarksStore.searchQuery"
          placeholder="搜索当前书签..."
          class="w-full px-3 py-2 pl-8 rounded-xl bg-white/10 border border-white/15 text-white placeholder-white/40 text-xs focus:outline-none focus:border-indigo-400"
        />
        <Icon icon="tabler:search" class="w-4 h-4 text-white/50 absolute left-2.5 top-2.5" />
      </div>

      <!-- Theme Presets Switcher -->
      <div class="space-y-1.5">
        <span class="text-[11px] font-semibold text-white/70">预制主题样式</span>
        <div class="grid grid-cols-2 gap-1.5">
          <button
            type="button"
            @click="handleSelectThemePreset('crystal')"
            class="px-2.5 py-2 rounded-xl text-left border transition-all flex flex-col gap-0.5"
            :class="[
              isCardStyle('transparent')
                ? 'border-indigo-500 bg-indigo-600/30 text-white font-semibold'
                : 'border-white/10 hover:border-white/30 bg-white/5 text-white/90'
            ]"
          >
            <span class="text-xs">✨ 全透明模式</span>
            <span class="text-[10px] text-white/60">原图纯净文字</span>
          </button>

          <button
            type="button"
            @click="handleSelectThemePreset('glass')"
            class="px-2.5 py-2 rounded-xl text-left border transition-all flex flex-col gap-0.5"
            :class="[
              isCardStyle('glass')
                ? 'border-indigo-500 bg-indigo-600/30 text-white font-semibold'
                : 'border-white/10 hover:border-white/30 bg-white/5 text-white/90'
            ]"
          >
            <span class="text-xs">🧊 经典毛玻璃</span>
            <span class="text-[10px] text-white/60">半透亚克力卡片</span>
          </button>

          <button
            type="button"
            @click="handleSelectThemePreset('solid')"
            class="px-2.5 py-2 rounded-xl text-left border transition-all flex flex-col gap-0.5"
            :class="[
              isCardStyle('solid')
                ? 'border-indigo-500 bg-indigo-600/30 text-white font-semibold'
                : 'border-white/10 hover:border-white/30 bg-white/5 text-white/90'
            ]"
          >
            <span class="text-xs">◻️ 纯色卡片</span>
            <span class="text-[10px] text-white/60">清晰实体质感</span>
          </button>

          <button
            type="button"
            @click="handleSelectThemePreset('minimal')"
            class="px-2.5 py-2 rounded-xl text-left border transition-all flex flex-col gap-0.5"
            :class="[
              isCardStyle('minimal')
                ? 'border-indigo-500 bg-indigo-600/30 text-white font-semibold'
                : 'border-white/10 hover:border-white/30 bg-white/5 text-white/90'
            ]"
          >
            <span class="text-xs">▫️ 极简线条</span>
            <span class="text-[10px] text-white/60">无边框平铺</span>
          </button>
        </div>
      </div>

      <!-- Card Size Toggle -->
      <div class="pt-2 border-t border-white/10 flex items-center justify-between">
        <span class="text-[11px] font-semibold text-white/70">排列密度</span>
        <CardSizeSwitcher />
      </div>

      <!-- Appearance Fine-tuning: Font Opacity & Label Vertical Position Offset -->
      <div class="pt-2.5 border-t border-white/10 space-y-2.5">
        <!-- Font Opacity Slider -->
        <div class="space-y-1">
          <div class="flex items-center justify-between text-[11px] font-semibold text-white/70">
            <span>字体透明度</span>
            <span class="font-mono text-indigo-400">{{ settingsStore.textOpacity }}%</span>
          </div>
          <input
            type="range"
            min="20"
            max="100"
            step="5"
            v-model.number="settingsStore.textOpacity"
            @change="persistQuickAppearance"
            class="w-full accent-indigo-500 cursor-pointer"
          />
        </div>

        <!-- Vertical Offset Slider -->
        <div class="space-y-1">
          <div class="flex items-center justify-between text-[11px] font-semibold text-white/70">
            <span>标签上下位置 (整体垂直偏移)</span>
            <span class="font-mono text-indigo-400">{{ settingsStore.contentTopOffset }}vh</span>
          </div>
          <input
            type="range"
            min="0"
            max="50"
            step="1"
            v-model.number="settingsStore.contentTopOffset"
            @change="persistQuickAppearance"
            class="w-full accent-indigo-500 cursor-pointer"
          />
        </div>

        <!-- Group Titles Switch (Default hidden, keeps only + icon) -->
        <div class="flex items-center justify-between text-[11px] font-semibold text-white/70 pt-1">
          <span>显示分组文字</span>
          <button
            type="button"
            @click="toggleGroupTitles"
            class="relative inline-flex h-5 w-9 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none"
            :class="settingsStore.showGroupTitles ? 'bg-indigo-600' : 'bg-white/20'"
            title="开启/关闭分组标题文字 (关闭时仅显示加号+)"
          >
            <span
              class="pointer-events-none inline-block h-4 w-4 transform rounded-full bg-white shadow ring-0 transition duration-200 ease-in-out"
              :class="settingsStore.showGroupTitles ? 'translate-x-4' : 'translate-x-0'"
            />
          </button>
        </div>
      </div>

      <!-- Quick Actions -->
        <!-- Spotlight Search -->
        <button
          type="button"
          @click="showSpotlight = true; showLauncherMenu = false"
          class="w-full px-3 py-2 rounded-xl bg-white/10 hover:bg-white/20 text-white font-medium flex items-center justify-between text-xs transition-colors cursor-pointer"
        >
          <div class="flex items-center gap-1.5">
            <Icon icon="tabler:search" class="w-4 h-4 text-indigo-400" />
            <span>全局快捷搜索</span>
          </div>
          <kbd class="px-1.5 py-0.2 rounded bg-black/30 text-[10px] text-white/50 font-mono">Ctrl+K</kbd>
        </button>

        <!-- Quick Memo / Clipboard -->
        <button
          type="button"
          @click="showMemoModal = true; showLauncherMenu = false"
          class="w-full px-3 py-2 rounded-xl bg-white/10 hover:bg-white/20 text-white font-medium flex items-center justify-between text-xs transition-colors cursor-pointer"
        >
          <div class="flex items-center gap-1.5">
            <Icon icon="tabler:notes" class="w-4 h-4 text-emerald-400" />
            <span>随手记 / 剪贴板</span>
          </div>
          <span class="text-[10px] text-white/50">云同步</span>
        </button>

        <!-- Zen Screensaver Mode -->
        <button
          type="button"
          @click="isZenMode = true; showLauncherMenu = false"
          class="w-full px-3 py-2 rounded-xl bg-white/10 hover:bg-white/20 text-white font-medium flex items-center justify-between text-xs transition-colors cursor-pointer"
        >
          <div class="flex items-center gap-1.5">
            <Icon icon="tabler:sparkles" class="w-4 h-4 text-amber-400" />
            <span>Zen 沉浸屏保</span>
          </div>
          <kbd class="px-1.5 py-0.2 rounded bg-black/30 text-[10px] text-white/50 font-mono">按 Z</kbd>
        </button>

        <button
          v-if="authStore.isAuthenticated"
          type="button"
          @click="handleAddBookmark()"
          class="w-full px-3 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-semibold flex items-center justify-center gap-1.5 transition-colors cursor-pointer"
        >
          <Icon icon="tabler:plus" class="w-4 h-4" />
          <span>添加书签卡片</span>
        </button>

        <button
          v-if="authStore.isAuthenticated"
          type="button"
          @click="handleAddTag()"
          class="w-full px-3 py-2 rounded-xl bg-white/10 hover:bg-white/20 text-white font-medium flex items-center justify-center gap-1.5 transition-colors cursor-pointer"
        >
          <Icon icon="tabler:tags" class="w-4 h-4" />
          <span>标签管理 / 新建</span>
        </button>

        <router-link
          v-if="authStore.isAuthenticated"
          to="/admin"
          class="w-full px-3 py-2 rounded-xl bg-white/10 hover:bg-white/20 text-white font-medium flex items-center justify-center gap-1.5 transition-colors"
        >
          <Icon icon="tabler:settings" class="w-4 h-4" />
          <span>进入管理后台</span>
        </router-link>

        <router-link
          v-else
          to="/login"
          class="w-full px-3 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-semibold flex items-center justify-center gap-1.5 transition-colors"
        >
          <Icon icon="tabler:login" class="w-4 h-4" />
          <span>管理员登录</span>
        </router-link>
      </div>
    </div>

    <!-- ========================================== -->
    <!-- 4. MAIN CONTENT                            -->
    <!-- ========================================== -->
    <main class="flex-1 flex flex-col items-center w-full">
      <!-- Search Bar (Hidden in transparent mode) -->
      <div v-if="settingsStore.cardStyle !== 'transparent'" class="w-full mb-6">
        <SearchBar />
      </div>

      <!-- Tags Quick Filter Bar (Hidden in transparent mode) -->
      <div
        v-if="settingsStore.cardStyle !== 'transparent' && (bookmarksStore.tags.length > 0 || authStore.isAuthenticated)"
        class="w-full max-w-4xl flex items-center justify-center flex-wrap gap-2 mb-8 px-2"
      >
        <button
          type="button"
          @click="bookmarksStore.selectedTagId = null"
          class="px-3 py-1 rounded-full text-xs font-medium transition-all"
          :class="[
            bookmarksStore.selectedTagId === null
              ? 'bg-indigo-600 text-white shadow-xs'
              : 'bg-white/60 dark:bg-slate-800/60 backdrop-blur-md text-slate-600 dark:text-slate-400 hover:bg-white dark:hover:bg-slate-800'
          ]"
        >
          全部标签
        </button>

        <button
          v-for="t in bookmarksStore.tags"
          :key="t.id"
          type="button"
          @click="toggleTag(t.id)"
          class="px-3 py-1 rounded-full text-xs font-medium transition-all"
          :style="{
            backgroundColor: bookmarksStore.selectedTagId === t.id ? t.color : `${t.color}20`,
            color: bookmarksStore.selectedTagId === t.id ? '#ffffff' : t.color,
            border: `1px solid ${t.color}40`
          }"
        >
          {{ t.name }}
        </button>

        <button
          v-if="authStore.isAuthenticated"
          type="button"
          @click="handleAddTag"
          class="flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-medium transition-all text-slate-500 hover:text-indigo-600 dark:text-slate-400 dark:hover:text-indigo-400 bg-white/50 dark:bg-slate-800/50 hover:bg-white dark:hover:bg-slate-800 border border-dashed border-slate-300 dark:border-slate-700 cursor-pointer"
          title="管理或新建分类标签"
        >
          <Icon icon="tabler:tags" class="w-3.5 h-3.5" />
          <span>管理 / 新建标签</span>
        </button>
      </div>

      <!-- Groups and Bookmarks Display with Adjustable Top Offset -->
      <div
        class="w-full transition-all duration-300"
        :style="{
          paddingTop: `${settingsStore.contentTopOffset}vh`
        }"
      >
        <!-- If groups exist -->
        <template v-if="bookmarksStore.groupedBookmarks.length > 0">
          <GroupSection
            v-for="g in bookmarksStore.groupedBookmarks"
            :key="g.id"
            :group="g"
            @add-bookmark="handleAddBookmark($event)"
            @edit-bookmark="handleEditBookmark($event)"
          />
        </template>

        <!-- Empty State -->
        <div
          v-else-if="!bookmarksStore.loading"
          class="w-full py-16 text-center text-slate-400"
        >
          <Icon icon="tabler:bookmark-off" class="w-12 h-12 mx-auto mb-3 opacity-40" />
          <p class="text-sm">没有找到匹配的书签</p>
          <button
            v-if="authStore.isAuthenticated"
            type="button"
            @click="handleAddBookmark()"
            class="mt-3 px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl text-xs font-semibold"
          >
            + 立即添加首个卡片
          </button>
        </div>
      </div>
    </main>

    <!-- Quick Modals -->
    <QuickBookmarkModal
      :show="showBookmarkModal"
      :bookmark="editingBookmark"
      :default-group-id="targetGroupId"
      @close="showBookmarkModal = false"
    />

    <QuickTagModal
      :show="showTagModal"
      @close="showTagModal = false"
    />

    <!-- Global Spotlight / Raycast Search -->
    <SpotlightSearch
      :show="showSpotlight"
      @close="showSpotlight = false"
    />

    <!-- Cross-device Quick Memo / Scratchpad -->
    <QuickMemoModal
      :show="showMemoModal"
      @close="showMemoModal = false"
    />
  </div>

  <!-- Zen Screensaver Ambient Clock Overlay -->
  <transition
    enter-active-class="transition duration-500 ease-out"
    enter-from-class="opacity-0"
    enter-to-class="opacity-100"
    leave-active-class="transition duration-300 ease-in"
    leave-from-class="opacity-100"
    leave-to-class="opacity-0"
  >
    <div
      v-if="isZenMode"
      @click="isZenMode = false"
      class="fixed inset-0 z-50 flex flex-col items-center justify-center cursor-pointer select-none bg-black/25 backdrop-blur-[2px]"
    >
      <div class="text-center space-y-3 p-8 sm:p-12 rounded-3xl bg-black/30 backdrop-blur-md border border-white/10 shadow-2xl animate-pulse">
        <div class="text-6xl sm:text-8xl md:text-9xl font-extralight tracking-tight text-white drop-shadow-[0_4px_24px_rgba(0,0,0,0.95)] font-mono">
          {{ zenTime }}
        </div>
        <div class="text-base sm:text-xl font-light text-white/80 drop-shadow-[0_2px_8px_rgba(0,0,0,0.95)] tracking-widest">
          {{ zenDate }}
        </div>
        <div class="pt-4 text-xs text-white/50 font-light flex items-center justify-center gap-1.5">
          <Icon icon="tabler:sparkles" class="w-4 h-4 text-amber-400" />
          <span>Zen 沉浸屏保模式 · 晃动鼠标或触控任意处退出</span>
        </div>
      </div>
    </div>
  </transition>
</template>
