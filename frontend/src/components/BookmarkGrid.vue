<script setup lang="ts">
import { computed } from 'vue'
import { Bookmark, useBookmarksStore } from '@/stores/bookmarks'
import { useSettingsStore } from '@/stores/settings'
import BookmarkCard from './BookmarkCard.vue'

const props = defineProps<{
  bookmarks: Bookmark[]
  groupId?: string
}>()

const emit = defineEmits<{
  (e: 'edit', bookmark: Bookmark): void
}>()

const bookmarksStore = useBookmarksStore()
const settingsStore = useSettingsStore()

const gridClass = computed(() => {
  const desktopCols = settingsStore.gridColsDesktop || 6
  const tabletCols = settingsStore.gridColsTablet || 3
  const mobileCols = settingsStore.gridColsMobile || 2

  const colClasses: Record<number, string> = {
    1: 'grid-cols-1',
    2: 'grid-cols-2',
    3: 'grid-cols-3',
    4: 'grid-cols-4',
    5: 'grid-cols-5',
    6: 'grid-cols-6',
    7: 'grid-cols-7',
    8: 'grid-cols-8',
  }

  const mClass = colClasses[mobileCols] || 'grid-cols-2'
  const tClass = `sm:${colClasses[tabletCols] || 'grid-cols-3'}`
  const dClass = `lg:${colClasses[desktopCols] || 'grid-cols-6'}`

  return `${mClass} ${tClass} ${dClass}`
})

async function handleGridDrop(e: DragEvent) {
  const sourceId = bookmarksStore.draggingBookmarkId || e.dataTransfer?.getData('text/plain')
  if (sourceId && props.groupId) {
    await bookmarksStore.moveBookmark(sourceId, undefined, props.groupId)
  }
}
</script>

<template>
  <div
    @dragover.prevent
    @drop.prevent="handleGridDrop"
    class="grid transition-all duration-300 min-h-[40px] p-1"
    :class="[
      settingsStore.cardStyle === 'transparent'
        ? 'grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-x-6 sm:gap-x-10 gap-y-3 sm:gap-y-4 mb-6'
        : `gap-3.5 sm:gap-4 rounded-2xl ${gridClass}`
    ]"
  >
    <BookmarkCard
      v-for="b in bookmarks"
      :key="b.id"
      :bookmark="b"
      @edit="emit('edit', $event)"
    />
  </div>
</template>
