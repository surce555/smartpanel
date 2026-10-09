<script setup lang="ts">
import { ref, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { Icon } from '@iconify/vue'
import { Bookmark, useBookmarksStore } from '@/stores/bookmarks'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'select', bookmark: Bookmark): void
}>()

const bookmarksStore = useBookmarksStore()
const query = ref('')
const selectedIndex = ref(0)
const inputRef = ref<HTMLInputElement | null>(null)

const searchResults = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) {
    // Return first 8 bookmarks as suggestions
    return bookmarksStore.bookmarks.slice(0, 8)
  }
  return bookmarksStore.bookmarks.filter((b) => {
    const matchName = b.name.toLowerCase().includes(q)
    const matchDesc = b.description?.toLowerCase().includes(q)
    const matchUrl = (b.url_internal || '').toLowerCase().includes(q) || (b.url_public_template || '').toLowerCase().includes(q)
    const matchTag = b.tags?.some((t) => t.name.toLowerCase().includes(q))
    return matchName || matchDesc || matchUrl || matchTag
  }).slice(0, 10)
})

watch(
  () => props.show,
  (val) => {
    if (val) {
      query.value = ''
      selectedIndex.value = 0
      nextTick(() => {
        inputRef.value?.focus()
      })
    }
  }
)

watch(searchResults, () => {
  selectedIndex.value = 0
})

function handleKeyDown(e: KeyboardEvent) {
  if (!props.show) return

  if (e.key === 'ArrowDown') {
    e.preventDefault()
    if (searchResults.value.length > 0) {
      selectedIndex.value = (selectedIndex.value + 1) % searchResults.value.length
    }
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    if (searchResults.value.length > 0) {
      selectedIndex.value = (selectedIndex.value - 1 + searchResults.value.length) % searchResults.value.length
    }
  } else if (e.key === 'Enter') {
    e.preventDefault()
    if (searchResults.value[selectedIndex.value]) {
      openBookmark(searchResults.value[selectedIndex.value])
    }
  } else if (e.key === 'Escape') {
    emit('close')
  }
}

function openBookmark(b: Bookmark) {
  const url = bookmarksStore.resolveBookmarkUrl(b)
  if (url && url !== '#') {
    window.open(url, b.open_in_new_tab ? '_blank' : '_self')
  }
  emit('close')
}

onMounted(() => {
  window.addEventListener('keydown', handleKeyDown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeyDown)
})
</script>

<template>
  <div
    v-if="show"
    class="fixed inset-0 z-50 flex items-start justify-center pt-[15vh] px-4 bg-black/60 backdrop-blur-md"
    @click.self="emit('close')"
  >
    <div
      class="w-full max-w-xl rounded-2xl bg-slate-900/90 border border-white/15 shadow-2xl overflow-hidden backdrop-blur-2xl flex flex-col"
    >
      <!-- Search Input Header -->
      <div class="flex items-center gap-3 px-4 py-3.5 border-b border-white/10 bg-white/5">
        <Icon icon="tabler:search" class="w-5 h-5 text-indigo-400 shrink-0" />
        <input
          ref="inputRef"
          v-model="query"
          type="text"
          placeholder="快速搜索服务、书签、或按回车直达..."
          class="flex-1 bg-transparent text-white placeholder-white/40 text-sm focus:outline-none font-medium"
        />
        <div class="flex items-center gap-1.5 shrink-0">
          <kbd class="px-2 py-0.5 rounded bg-white/10 text-[10px] text-white/50 border border-white/10 font-mono">ESC 退出</kbd>
        </div>
      </div>

      <!-- Results List -->
      <div class="max-h-[50vh] overflow-y-auto p-2 divide-y divide-white/5">
        <div
          v-if="searchResults.length === 0"
          class="py-10 text-center text-xs text-white/40 flex flex-col items-center justify-center gap-2"
        >
          <Icon icon="tabler:mood-empty" class="w-8 h-8 opacity-40" />
          <span>没有找到相关服务，可尝试搜索其他关键词</span>
        </div>

        <div
          v-for="(b, idx) in searchResults"
          :key="b.id"
          @click="openBookmark(b)"
          @mouseenter="selectedIndex = idx"
          class="group flex items-center justify-between px-3 py-2.5 rounded-xl cursor-pointer transition-all"
          :class="[
            selectedIndex === idx
              ? 'bg-indigo-600 text-white shadow-md'
              : 'text-white/80 hover:bg-white/5'
          ]"
        >
          <!-- Left: Icon & Title -->
          <div class="flex items-center gap-3 min-w-0">
            <div
              class="w-8 h-8 rounded-lg flex items-center justify-center shrink-0 transition-colors"
              :class="selectedIndex === idx ? 'bg-white/20 text-white' : 'bg-white/10 text-indigo-300'"
            >
              <img
                v-if="b.icon && (b.icon.startsWith('http') || b.icon.startsWith('/') || b.icon.startsWith('data:'))"
                :src="b.icon"
                class="w-5 h-5 object-contain rounded"
              />
              <Icon v-else-if="b.icon" :icon="b.icon" class="w-5 h-5" />
              <Icon v-else icon="tabler:link" class="w-5 h-5" />
            </div>

            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="text-xs font-semibold truncate">{{ b.name }}</span>
                <span
                  v-if="b.is_private"
                  class="text-[9px] px-1.5 py-0.2 rounded bg-amber-500/30 text-amber-300 font-mono"
                >
                  私密
                </span>
              </div>
              <p
                class="text-[11px] truncate opacity-70"
                :class="selectedIndex === idx ? 'text-white/80' : 'text-white/50'"
              >
                {{ b.description || bookmarksStore.resolveBookmarkUrl(b) }}
              </p>
            </div>
          </div>

          <!-- Right: Tags or Enter key hint -->
          <div class="flex items-center gap-2 shrink-0 ml-3">
            <div v-if="b.tags && b.tags.length > 0" class="hidden sm:flex items-center gap-1">
              <span
                v-for="t in b.tags"
                :key="t.id"
                class="text-[10px] px-1.5 py-0.5 rounded-md bg-white/10 text-white/70"
              >
                {{ t.name }}
              </span>
            </div>
            <kbd
              v-if="selectedIndex === idx"
              class="px-1.5 py-0.5 rounded bg-black/20 text-[10px] text-white/90 border border-white/20 font-mono flex items-center gap-1"
            >
              <span>回车打开</span>
              <Icon icon="tabler:corner-down-left" class="w-3 h-3" />
            </kbd>
          </div>
        </div>
      </div>

      <!-- Footer Help -->
      <div class="px-4 py-2 border-t border-white/10 bg-white/5 flex items-center justify-between text-[11px] text-white/40">
        <div class="flex items-center gap-3">
          <span><kbd class="font-mono bg-white/10 px-1 py-0.5 rounded">↑</kbd> <kbd class="font-mono bg-white/10 px-1 py-0.5 rounded">↓</kbd> 切换选项</span>
          <span><kbd class="font-mono bg-white/10 px-1 py-0.5 rounded">↵</kbd> 直达打开</span>
        </div>
        <span>全局快捷键 <kbd class="font-mono bg-white/10 px-1 py-0.5 rounded">Ctrl + K</kbd></span>
      </div>
    </div>
  </div>
</template>
