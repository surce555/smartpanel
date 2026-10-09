<script setup lang="ts">
import { ref } from 'vue'
import { Icon } from '@iconify/vue'
import { Group, Bookmark, useBookmarksStore } from '@/stores/bookmarks'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import BookmarkGrid from './BookmarkGrid.vue'

const props = defineProps<{
  group: Group
}>()

const emit = defineEmits<{
  (e: 'add-bookmark', groupId: string): void
  (e: 'edit-bookmark', bookmark: Bookmark): void
}>()

const bookmarksStore = useBookmarksStore()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const isCollapsed = ref<boolean>(false)
const isDragOverGroup = ref<boolean>(false)

async function handleEmptyDrop(e: DragEvent) {
  isDragOverGroup.value = false
  const sourceId = bookmarksStore.draggingBookmarkId || e.dataTransfer?.getData('text/plain')
  if (sourceId && props.group.id) {
    await bookmarksStore.moveBookmark(sourceId, undefined, props.group.id)
  }
}
</script>

<template>
  <section class="mb-8 last:mb-2">
    <!-- Group Header -->
    <div
      class="flex items-center justify-between mb-3 px-1 py-1 select-none group/hdr"
    >
      <div
        @click="isCollapsed = !isCollapsed"
        class="flex items-center gap-2.5 cursor-pointer"
      >
        <div
          class="flex items-center justify-center w-7 h-7 rounded-lg group-hover/hdr:scale-105 transition-transform"
          :class="[
            settingsStore.cardStyle === 'transparent'
              ? 'bg-black/35 backdrop-blur-md text-white border border-white/20'
              : 'bg-indigo-50 dark:bg-indigo-950/60 text-indigo-600 dark:text-indigo-400'
          ]"
        >
          <Icon v-if="group.icon" :icon="group.icon" class="w-4 h-4" />
          <Icon v-else icon="tabler:folder" class="w-4 h-4" />
        </div>
        <h2
          class="text-base font-bold tracking-tight transition-colors"
          :class="[
            settingsStore.cardStyle === 'transparent'
              ? 'text-white drop-shadow-[0_2px_4px_rgba(0,0,0,0.95)]'
              : 'text-slate-800 dark:text-slate-100 group-hover/hdr:text-indigo-600 dark:group-hover/hdr:text-indigo-400'
          ]"
        >
          {{ group.name }}
        </h2>
        <span
          class="text-xs px-2 py-0.5 rounded-full font-medium"
          :class="[
            settingsStore.cardStyle === 'transparent'
              ? 'bg-black/35 backdrop-blur-md text-white/90 border border-white/10'
              : 'bg-slate-200/60 dark:bg-slate-800/60 text-slate-500 dark:text-slate-400'
          ]"
        >
          {{ group.bookmarks?.length || 0 }}
        </span>
      </div>

      <!-- Actions on Header -->
      <div class="flex items-center gap-1.5">
        <!-- Quick add button for this group -->
        <button
          v-if="authStore.isAuthenticated"
          type="button"
          @click.stop="emit('add-bookmark', group.id)"
          class="flex items-center gap-1 px-2.5 py-1 rounded-lg text-xs font-medium transition-all opacity-85 hover:opacity-100"
          :class="[
            settingsStore.cardStyle === 'transparent'
              ? 'text-white bg-black/35 hover:bg-black/55 backdrop-blur-md border border-white/20'
              : 'text-slate-500 hover:text-indigo-600 dark:text-slate-400 dark:hover:text-indigo-400 hover:bg-slate-100 dark:hover:bg-slate-800'
          ]"
          title="在此分组添加新卡片"
        >
          <Icon icon="tabler:plus" class="w-3.5 h-3.5" />
          <span class="hidden sm:inline text-[11px]">添加卡片</span>
        </button>

        <!-- Collapse Icon -->
        <button
          type="button"
          @click="isCollapsed = !isCollapsed"
          class="p-1 rounded-md text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 transition-transform duration-200"
          :class="{ 'rotate-180': isCollapsed }"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
          </svg>
        </button>
      </div>
    </div>

    <!-- Bookmarks Grid -->
    <div v-show="!isCollapsed" class="transition-all duration-300">
      <BookmarkGrid
        v-if="group.bookmarks && group.bookmarks.length > 0"
        :bookmarks="group.bookmarks"
        :group-id="group.id"
        @edit="emit('edit-bookmark', $event)"
      />
      <!-- Empty dropzone -->
      <div
        v-else
        @dragover.prevent="isDragOverGroup = true"
        @dragleave="isDragOverGroup = false"
        @drop.prevent="handleEmptyDrop"
        class="p-8 rounded-2xl border-2 border-dashed text-center text-xs transition-all duration-200"
        :class="[
          isDragOverGroup
            ? 'border-indigo-500 bg-indigo-50/50 dark:bg-indigo-950/30 text-indigo-600 dark:text-indigo-400 scale-[1.01]'
            : 'border-slate-300 dark:border-slate-800 text-slate-400 hover:border-slate-400'
        ]"
      >
        <p class="font-medium">此分组暂无卡片</p>
        <p class="text-[11px] text-slate-400 mt-1">
          可直接拖拽其他卡片至此处，或点击右上角「+ 添加卡片」
        </p>
      </div>
    </div>
  </section>
</template>
