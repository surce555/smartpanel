<script setup lang="ts">
import { computed } from 'vue'
import { Bookmark } from '@/stores/bookmarks'
import { useSettingsStore } from '@/stores/settings'
import BookmarkCard from './BookmarkCard.vue'

defineProps<{
  bookmarks: Bookmark[]
}>()

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
</script>

<template>
  <div class="grid gap-3.5 sm:gap-4 transition-all duration-300" :class="gridClass">
    <BookmarkCard
      v-for="b in bookmarks"
      :key="b.id"
      :bookmark="b"
    />
  </div>
</template>
