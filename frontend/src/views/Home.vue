<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { useRouter } from 'vue-router'
import { Icon } from '@iconify/vue'
import { useBookmarksStore } from '@/stores/bookmarks'
import { useSettingsStore } from '@/stores/settings'
import { useAuthStore } from '@/stores/auth'
import { useIPv6Probe } from '@/composables/useIPv6Probe'
import SearchBar from '@/components/SearchBar.vue'
import GroupSection from '@/components/GroupSection.vue'
import ThemeSwitcher from '@/components/ThemeSwitcher.vue'

const router = useRouter()
const bookmarksStore = useBookmarksStore()
const settingsStore = useSettingsStore()
const authStore = useAuthStore()
const { isIPv6Available, probeIPv6 } = useIPv6Probe()

// Digital Clock
const currentTime = ref<string>('')
const currentDate = ref<string>('')
const currentWeekday = ref<string>('')
let clockTimer: number | null = null

function updateClock() {
  const now = new Date()
  const is24 = settingsStore.clockFormat24

  let hours = now.getHours()
  const minutes = String(now.getMinutes()).padStart(2, '0')
  const seconds = String(now.getSeconds()).padStart(2, '0')

  let timeStr = ''
  if (is24) {
    timeStr = `${String(hours).padStart(2, '0')}:${minutes}:${seconds}`
  } else {
    const ampm = hours >= 12 ? 'PM' : 'AM'
    hours = hours % 12 || 12
    timeStr = `${hours}:${minutes}:${seconds} ${ampm}`
  }
  currentTime.value = timeStr

  currentDate.value = `${now.getFullYear()}年${now.getMonth() + 1}月${now.getDate()}日`
  const weekdays = ['星期日', '星期一', '星期二', '星期三', '星期四', '星期五', '星期六']
  currentWeekday.value = weekdays[now.getDay()]
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
  updateClock()
  clockTimer = window.setInterval(updateClock, 1000)
  await bookmarksStore.fetchAll()
})

onUnmounted(() => {
  if (clockTimer) clearInterval(clockTimer)
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
  <div class="min-h-screen flex flex-col justify-between p-4 sm:p-6 lg:p-8 max-w-7xl mx-auto">
    <!-- Top Header -->
    <header class="flex items-center justify-between gap-4 mb-6 sm:mb-8">
      <!-- Left: Logo & Network Indicator -->
      <div class="flex items-center gap-3">
        <div class="flex items-center justify-center w-10 h-10 rounded-2xl bg-indigo-600 text-white shadow-lg shadow-indigo-600/30">
          <Icon icon="tabler:layout-dashboard" class="w-5 h-5" />
        </div>
        <div>
          <div class="flex items-center gap-2">
            <h1 class="text-lg font-bold text-slate-900 dark:text-white tracking-tight">
              SmartPanel
            </h1>
            <!-- Smart Routing Mode Badge -->
            <span
              v-if="isIPv6Available === true"
              class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold bg-emerald-100/80 dark:bg-emerald-950/60 text-emerald-700 dark:text-emerald-300 border border-emerald-200 dark:border-emerald-800"
              title="当前网络已通过 IPv6 高速直连 NAS"
            >
              <span class="w-1.5 h-1.5 rounded-full bg-emerald-500"></span>
              IPv6 直连
            </span>
            <span
              v-else-if="isIPv6Available === false"
              class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold bg-sky-100/80 dark:bg-sky-950/60 text-sky-700 dark:text-sky-300 border border-sky-200 dark:border-sky-800"
              title="当前无 IPv6 环境，自动走 Cloudflare 代理通道"
            >
              <span class="w-1.5 h-1.5 rounded-full bg-sky-500"></span>
              CF 代理
            </span>
            <span
              v-else
              class="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-semibold bg-slate-100 dark:bg-slate-800 text-slate-500 border border-slate-200 dark:border-slate-700"
            >
              探测中...
            </span>
          </div>
          <p class="text-[11px] text-slate-500 dark:text-slate-400 hidden sm:block">
            智能双栈 NAS 个人仪表盘
          </p>
        </div>
      </div>

      <!-- Center: Clock (Desktop) -->
      <div v-if="settingsStore.showClock" class="hidden md:flex flex-col items-center">
        <div class="text-2xl font-extrabold font-mono text-slate-800 dark:text-slate-100 tracking-wider">
          {{ currentTime }}
        </div>
        <div class="text-xs text-slate-500 dark:text-slate-400 font-medium">
          {{ currentDate }} {{ currentWeekday }}
        </div>
      </div>

      <!-- Right: Controls -->
      <div class="flex items-center gap-2 sm:gap-3">
        <ThemeSwitcher />

        <!-- Admin / Login Link -->
        <router-link
          v-if="authStore.isAuthenticated"
          to="/admin"
          class="flex items-center gap-1.5 px-3.5 py-1.5 rounded-full bg-white/70 dark:bg-slate-800/70 backdrop-blur-md border border-slate-200/80 dark:border-slate-700/80 text-xs font-semibold text-slate-700 dark:text-slate-200 hover:text-indigo-600 dark:hover:text-indigo-400 shadow-xs transition-colors"
        >
          <Icon icon="tabler:settings" class="w-4 h-4" />
          <span class="hidden sm:inline">管理面板</span>
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
    </header>

    <!-- Main Content Area -->
    <main class="flex-1 flex flex-col items-center w-full">
      <!-- Search Bar -->
      <div class="w-full mb-6">
        <SearchBar />
      </div>

      <!-- Tags Quick Filter Bar -->
      <div
        v-if="bookmarksStore.tags.length > 0"
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
      </div>

      <!-- Groups and Bookmarks Display -->
      <div class="w-full">
        <!-- If groups exist -->
        <template v-if="bookmarksStore.groupedBookmarks.length > 0">
          <GroupSection
            v-for="g in bookmarksStore.groupedBookmarks"
            :key="g.id"
            :group="g"
          />
        </template>

        <!-- Empty State -->
        <div
          v-else-if="!bookmarksStore.loading"
          class="w-full py-16 text-center text-slate-400"
        >
          <Icon icon="tabler:bookmark-off" class="w-12 h-12 mx-auto mb-3 opacity-40" />
          <p class="text-sm">没有找到匹配的书签</p>
        </div>
      </div>
    </main>

    <!-- Footer -->
    <footer class="mt-12 pt-6 border-t border-slate-200/50 dark:border-slate-800/50 text-center text-xs text-slate-400 dark:text-slate-500 space-y-1">
      <p>{{ settingsStore.footerText }}</p>
      <p v-if="settingsStore.networkInfo.current_ipv6" class="text-[11px] font-mono text-slate-400/80">
        IPv6: {{ settingsStore.networkInfo.current_ipv6 }}
      </p>
    </footer>
  </div>
</template>
