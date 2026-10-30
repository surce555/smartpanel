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

// Tailwind only emits utilities it can find as complete literal class names while
// scanning the source. Interpolating like `sm:${'grid-cols-3'}` produces
// `sm:grid-cols-3` at runtime but Tailwind never sees it, so every sm:/lg:
// column setting was silently dropped. Keep the full class strings literal here.
const MOBILE_COLS: Record<number, string> = {
  1: 'grid-cols-1',
  2: 'grid-cols-2',
}

const TABLET_COLS: Record<number, string> = {
  2: 'sm:grid-cols-2',
  3: 'sm:grid-cols-3',
  4: 'sm:grid-cols-4',
}

const DESKTOP_COLS: Record<number, string> = {
  4: 'lg:grid-cols-4',
  5: 'lg:grid-cols-5',
  6: 'lg:grid-cols-6',
  7: 'lg:grid-cols-7',
  8: 'lg:grid-cols-8',
}

const gridClass = computed(() => {
  const mobile = MOBILE_COLS[settingsStore.gridColsMobile] || MOBILE_COLS[2]
  const tablet = TABLET_COLS[settingsStore.gridColsTablet] || TABLET_COLS[3]
  const desktop = DESKTOP_COLS[settingsStore.gridColsDesktop] || DESKTOP_COLS[6]
  return `${mobile} ${tablet} ${desktop}`
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
