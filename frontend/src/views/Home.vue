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

const router = useRouter()
const bookmarksStore = useBookmarksStore()
const settingsStore = useSettingsStore()
const authStore = useAuthStore()
const { probeIPv6 } = useIPv6Probe()

// Modals state
const showBookmarkModal = ref<boolean>(false)
const showTagModal = ref<boolean>(false)
const editingBookmark = ref<Bookmark | null>(null)
const targetGroupId = ref<string>('')
const showLauncherMenu = ref<boolean>(false)

// Quick search in launcher popover
const launcherSearchQuery = ref<string>('')

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

onMounted(async () => {
  if (settingsStore.requireLogin && !authStore.isAuthenticated) {
    router.replace({ name: 'Login', query: { redirect: '/' } })
    return
  }
  probeIPv6()
  window.addEventListener('click', handleOutsideClick)
  await bookmarksStore.fetchAll()
})

onUnmounted(() => {
  window.removeEventListener('click', handleOutsideClick)
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
  <div class="relative min-h-[100dvh] flex flex-col justify-start p-3.5 sm:p-6 lg:p-8 pt-[max(1rem,env(safe-area-inset-top))] pb-[max(1rem,env(safe-area-inset-bottom))] max-w-7xl mx-auto">
    
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

      <!-- Launcher Dropdown Popover in Transparent Mode -->
      <div
        v-if="showLauncherMenu"
        @click.stop
        class="absolute right-0 mt-2 w-72 sm:w-80 p-4 rounded-3xl bg-slate-900/95 backdrop-blur-2xl border border-white/20 shadow-2xl text-white text-xs z-50 space-y-3.5 animate-in fade-in zoom-in-95"
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
                settingsStore.cardStyle === 'transparent'
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
                settingsStore.cardStyle === 'glass'
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
                settingsStore.cardStyle === 'solid'
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
                settingsStore.cardStyle === 'minimal'
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

        <!-- Quick Actions -->
        <div class="pt-2 border-t border-white/10 space-y-1.5">
          <button
            v-if="authStore.isAuthenticated"
            type="button"
            @click="handleAddBookmark()"
            class="w-full px-3 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-semibold flex items-center justify-center gap-1.5 transition-colors"
          >
            <Icon icon="tabler:plus" class="w-4 h-4" />
            <span>添加书签卡片</span>
          </button>

          <button
            v-if="authStore.isAuthenticated"
            type="button"
            @click="handleAddTag()"
            class="w-full px-3 py-2 rounded-xl bg-white/10 hover:bg-white/20 text-white font-medium flex items-center justify-center gap-1.5 transition-colors"
          >
            <Icon icon="tabler:tag" class="w-4 h-4" />
            <span>新建分类标签</span>
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
    </div>

    <!-- ========================================== -->
    <!-- 2. STANDARD HEADER (Non-Transparent Modes) -->
    <!-- ========================================== -->
    <header
      v-if="settingsStore.cardStyle !== 'transparent'"
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

          <!-- Standard Dropdown Popover -->
          <div
            v-if="showLauncherMenu"
            @click.stop
            class="absolute right-0 mt-2 w-72 p-3.5 rounded-3xl bg-slate-900/95 backdrop-blur-2xl border border-white/20 shadow-2xl text-white text-xs z-50 space-y-3 animate-in fade-in zoom-in-95"
          >
            <div class="flex items-center justify-between border-b border-white/10 pb-2">
              <span class="font-bold text-white text-xs flex items-center gap-1.5">
                <Icon icon="tabler:apps" class="w-4 h-4 text-indigo-400" />
                快捷控制中心
              </span>
              <button @click="showLauncherMenu = false" class="text-white/60 hover:text-white">✕</button>
            </div>

            <!-- Theme Presets -->
            <div class="space-y-1.5">
              <span class="text-[11px] font-semibold text-white/80">预制主题样式</span>
              <div class="grid grid-cols-2 gap-1.5">
                <button
                  type="button"
                  @click="handleSelectThemePreset('crystal')"
                  class="px-2.5 py-2 rounded-xl text-left border transition-all flex flex-col gap-0.5"
                  :class="[
                    settingsStore.cardStyle === 'transparent'
                      ? 'border-indigo-500 bg-indigo-600/30 text-white font-semibold'
                      : 'border-white/10 hover:border-white/30 bg-white/5 text-white/90'
                  ]"
                >
                  <span class="text-xs">✨ 全透明模式</span>
                  <span class="text-[10px] text-white/60">原图纯净卡片</span>
                </button>

                <button
                  type="button"
                  @click="handleSelectThemePreset('glass')"
                  class="px-2.5 py-2 rounded-xl text-left border transition-all flex flex-col gap-0.5"
                  :class="[
                    settingsStore.cardStyle === 'glass'
                      ? 'border-indigo-500 bg-indigo-600/30 text-white font-semibold'
                      : 'border-white/10 hover:border-white/30 bg-white/5 text-white/90'
                  ]"
                >
                  <span class="text-xs">🧊 经典毛玻璃</span>
                  <span class="text-[10px] text-white/60">半透亚克力效果</span>
                </button>

                <button
                  type="button"
                  @click="handleSelectThemePreset('solid')"
                  class="px-2.5 py-2 rounded-xl text-left border transition-all flex flex-col gap-0.5"
                  :class="[
                    settingsStore.cardStyle === 'solid'
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
                    settingsStore.cardStyle === 'minimal'
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
              <span class="text-[11px] font-semibold text-white/80">卡片图标尺寸</span>
              <CardSizeSwitcher />
            </div>

            <!-- Actions -->
            <div class="pt-2 border-t border-white/10 space-y-1.5">
              <button
                v-if="authStore.isAuthenticated"
                type="button"
                @click="handleAddBookmark()"
                class="w-full px-3 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white font-semibold flex items-center justify-center gap-1.5 transition-colors"
              >
                <Icon icon="tabler:plus" class="w-4 h-4" />
                <span>快捷添加书签卡片</span>
              </button>

              <button
                v-if="authStore.isAuthenticated"
                type="button"
                @click="handleAddTag()"
                class="w-full px-3 py-2 rounded-xl bg-white/10 hover:bg-white/20 text-white font-medium flex items-center justify-center gap-1.5 transition-colors"
              >
                <Icon icon="tabler:tag" class="w-4 h-4" />
                <span>新建分类标签</span>
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
        </div>
      </div>
    </header>

    <!-- ========================================== -->
    <!-- 3. MAIN CONTENT                            -->
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
          class="flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-medium transition-all text-slate-500 hover:text-indigo-600 dark:text-slate-400 dark:hover:text-indigo-400 bg-white/50 dark:bg-slate-800/50 hover:bg-white dark:hover:bg-slate-800 border border-dashed border-slate-300 dark:border-slate-700"
          title="新增分类标签"
        >
          <Icon icon="tabler:plus" class="w-3.5 h-3.5" />
          <span>新建标签</span>
        </button>
      </div>

      <!-- Groups and Bookmarks Display -->
      <div class="w-full" :class="[settingsStore.cardStyle === 'transparent' ? 'pt-4 sm:pt-6' : '']">
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
  </div>
</template>
