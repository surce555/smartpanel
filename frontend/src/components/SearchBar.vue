<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useBookmarksStore } from '@/stores/bookmarks'
import { useSettingsStore } from '@/stores/settings'

const bookmarksStore = useBookmarksStore()
const settingsStore = useSettingsStore()

const inputRef = ref<HTMLInputElement | null>(null)
const showEngines = ref<boolean>(false)

interface SearchEngine {
  id: string
  name: string
  icon: string
  url: string
}

const engines: SearchEngine[] = [
  {
    id: 'bing',
    name: 'Bing',
    icon: 'logos:bing',
    url: 'https://www.bing.com/search?q=%s',
  },
  {
    id: 'google',
    name: 'Google',
    icon: 'logos:google-icon',
    url: 'https://www.google.com/search?q=%s',
  },
  {
    id: 'baidu',
    name: '百度',
    icon: 'simple-icons:baidu',
    url: 'https://www.baidu.com/s?wd=%s',
  },
  {
    id: 'custom',
    name: '自定义',
    icon: 'tabler:world-search',
    url: settingsStore.searchCustomUrl,
  },
]

const currentEngine = computed(() => {
  return engines.find((e) => e.id === settingsStore.searchEngine) || engines[0]
})

function selectEngine(id: string) {
  settingsStore.searchEngine = id
  showEngines.value = false
}

function handleEnter() {
  const query = bookmarksStore.searchQuery.trim()
  if (!query) return

  // If there are filtered bookmarks, don't necessarily jump to search engine unless intended
  if (bookmarksStore.filteredBookmarks.length === 0) {
    executeWebSearch(query)
  }
}

function executeWebSearch(query?: string) {
  const q = query || bookmarksStore.searchQuery.trim()
  if (!q) return

  let template = currentEngine.value.url
  if (currentEngine.value.id === 'custom' && settingsStore.searchCustomUrl) {
    template = settingsStore.searchCustomUrl
  }

  const target = template.replace('%s', encodeURIComponent(q))
  window.open(target, '_blank')
}

// Global Hotkeys: '/' and 'Ctrl+K'
function handleGlobalKeydown(e: KeyboardEvent) {
  // If user is typing in another input/textarea, ignore
  const activeEl = document.activeElement
  const isInput = activeEl instanceof HTMLInputElement || activeEl instanceof HTMLTextAreaElement

  if ((e.key === '/' && !isInput) || ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k')) {
    e.preventDefault()
    inputRef.value?.focus()
    inputRef.value?.select()
  } else if (e.key === 'Escape' && document.activeElement === inputRef.value) {
    inputRef.value?.blur()
    showEngines.value = false
  }
}

onMounted(() => {
  window.addEventListener('keydown', handleGlobalKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleGlobalKeydown)
})
</script>

<template>
  <div class="relative w-full max-w-2xl mx-auto z-30">
    <div
      class="flex items-center backdrop-blur-xl rounded-2xl transition-all duration-300 focus-within:ring-2 focus-within:ring-indigo-500/50"
      :class="[
        settingsStore.cardStyle === 'transparent'
          ? 'bg-black/35 backdrop-blur-md border border-white/20 text-white shadow-xl focus-within:border-white/50'
          : 'bg-white/70 dark:bg-slate-900/70 border border-slate-200/80 dark:border-slate-800/80 shadow-lg shadow-slate-200/20 dark:shadow-black/20 focus-within:border-indigo-500'
      ]"
    >
      <!-- Engine Selector Button -->
      <div class="relative">
        <button
          type="button"
          @click="showEngines = !showEngines"
          class="flex items-center gap-1.5 px-3.5 py-3 text-xs font-medium border-r transition-colors"
          :class="[
            settingsStore.cardStyle === 'transparent'
              ? 'text-white border-white/20 hover:text-white/80'
              : 'text-slate-700 dark:text-slate-300 hover:text-indigo-600 dark:hover:text-indigo-400 border-slate-200/60 dark:border-slate-800/60'
          ]"
        >
          <span>{{ currentEngine.name }}</span>
          <svg class="w-3.5 h-3.5 opacity-60" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
          </svg>
        </button>

        <!-- Dropdown Menu -->
        <div
          v-if="showEngines"
          class="absolute left-0 top-full mt-2 w-36 bg-white dark:bg-slate-900 rounded-xl shadow-xl border border-slate-200 dark:border-slate-800 p-1.5 z-50 text-xs backdrop-blur-lg"
        >
          <button
            v-for="e in engines"
            :key="e.id"
            @click="selectEngine(e.id)"
            class="w-full flex items-center justify-between px-3 py-2 rounded-lg text-left transition-colors"
            :class="[
              settingsStore.searchEngine === e.id
                ? 'bg-indigo-50 dark:bg-indigo-950/60 text-indigo-600 dark:text-indigo-400 font-medium'
                : 'text-slate-700 dark:text-slate-300 hover:bg-slate-100 dark:hover:bg-slate-800'
            ]"
          >
            <span>{{ e.name }}</span>
            <span v-if="settingsStore.searchEngine === e.id" class="text-indigo-600">✓</span>
          </button>
        </div>
      </div>

      <!-- Search Input (text-base on mobile to avoid iOS Safari zoom) -->
      <div class="relative flex-1 flex items-center">
        <input
          ref="inputRef"
          type="text"
          v-model="bookmarksStore.searchQuery"
          @keydown.enter="handleEnter"
          placeholder="搜索书签或回车网页检索... (按 / 聚焦)"
          class="w-full px-4 py-3 bg-transparent text-base sm:text-sm focus:outline-none"
          :class="[
            settingsStore.cardStyle === 'transparent'
              ? 'text-white placeholder-white/50'
              : 'text-slate-800 dark:text-slate-100 placeholder-slate-400 dark:placeholder-slate-500'
          ]"
        />
        <button
          v-if="bookmarksStore.searchQuery"
          @click="bookmarksStore.searchQuery = ''"
          class="p-1 mr-2 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 rounded-full"
          title="清除"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"/>
          </svg>
        </button>
      </div>

      <!-- Search Action Button -->
      <button
        type="button"
        @click="() => executeWebSearch()"
        class="px-4 py-3 text-slate-500 hover:text-indigo-600 dark:text-slate-400 dark:hover:text-indigo-400 transition-colors"
        title="在网络中搜索"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <circle cx="11" cy="11" r="8" stroke-width="2"/>
          <path stroke-linecap="round" stroke-width="2" d="M21 21l-4.35-4.35"/>
        </svg>
      </button>
    </div>
  </div>
</template>
