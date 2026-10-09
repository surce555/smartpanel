<script setup lang="ts">
import { computed } from 'vue'
import { Icon } from '@iconify/vue'
import { Bookmark, useBookmarksStore } from '@/stores/bookmarks'
import { useSettingsStore } from '@/stores/settings'
import HealthBadge from './HealthBadge.vue'

const props = defineProps<{
  bookmark: Bookmark
}>()

const bookmarksStore = useBookmarksStore()
const settingsStore = useSettingsStore()

const resolvedUrl = computed(() => {
  return bookmarksStore.resolveBookmarkUrl(props.bookmark)
})

const isExternalOrUploadedIcon = computed(() => {
  const icon = props.bookmark.icon
  if (!icon) return false
  return icon.startsWith('http://') || icon.startsWith('https://') || icon.startsWith('/uploads/') || icon.startsWith('data:')
})

const cardStyleClass = computed(() => {
  switch (settingsStore.cardStyle) {
    case 'solid':
      return 'card-solid'
    case 'transparent':
      return 'card-transparent'
    case 'minimal':
      return 'card-minimal'
    case 'glass':
    default:
      return 'card-glass'
  }
})

const shadowClass = computed(() => {
  switch (settingsStore.cardShadow) {
    case 'none':
      return ''
    case 'sm':
      return 'shadow-sm'
    case 'lg':
      return 'shadow-lg'
    case 'md':
    default:
      return 'shadow'
  }
})

const iconSizeClass = computed(() => {
  switch (settingsStore.iconSize) {
    case 'small':
      return 'w-8 h-8 text-xl'
    case 'large':
      return 'w-12 h-12 text-3xl'
    case 'detail':
      return 'w-14 h-14 text-4xl'
    case 'medium':
    default:
      return 'w-10 h-10 text-2xl'
  }
})

function handleClick(e: MouseEvent) {
  // If user clicks, open URL
  const url = resolvedUrl.value
  if (!url || url === '#') {
    e.preventDefault()
    return
  }
}
</script>

<template>
  <a
    :href="resolvedUrl"
    :target="bookmark.open_in_new_tab ? '_blank' : '_self'"
    rel="noopener noreferrer"
    @click="handleClick"
    class="group relative flex flex-col p-4 transition-all duration-300 hover:-translate-y-1 hover:shadow-xl cursor-pointer overflow-hidden"
    :class="[cardStyleClass, shadowClass]"
    :style="{ borderRadius: `${settingsStore.cardBorderRadius}px` }"
  >
    <!-- Top row: Icon + Health Badge -->
    <div class="flex items-start justify-between gap-3 mb-2.5">
      <!-- Icon Container -->
      <div
        class="flex items-center justify-center rounded-xl bg-slate-100/80 dark:bg-slate-800/80 p-2 text-indigo-600 dark:text-indigo-400 group-hover:scale-105 transition-transform duration-300 overflow-hidden shadow-xs"
        :class="iconSizeClass"
      >
        <img
          v-if="isExternalOrUploadedIcon"
          :src="bookmark.icon"
          :alt="bookmark.name"
          class="w-full h-full object-contain"
          loading="lazy"
          onerror="this.style.display='none'"
        />
        <Icon
          v-else-if="bookmark.icon"
          :icon="bookmark.icon"
          class="w-full h-full"
        />
        <Icon
          v-else
          icon="tabler:link"
          class="w-full h-full"
        />
      </div>

      <!-- Health Status Badge -->
      <div class="flex items-center gap-1.5 pt-1">
        <HealthBadge
          v-if="bookmark.health_check"
          :status="bookmark.health_check.status"
          :response-time="bookmark.health_check.response_time_ms"
        />
        <!-- Private lock indicator -->
        <span
          v-if="bookmark.is_private"
          title="私有书签（仅登录可见）"
          class="text-slate-400 dark:text-slate-500"
        >
          <svg class="w-3 h-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <rect x="3" y="11" width="18" height="11" rx="2" ry="2" stroke-width="2"/>
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 11V7a5 5 0 0110 0v4"/>
          </svg>
        </span>
      </div>
    </div>

    <!-- Title -->
    <h3 class="text-sm font-semibold text-slate-800 dark:text-slate-100 group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors truncate">
      {{ bookmark.name }}
    </h3>

    <!-- Description (if any) -->
    <p
      v-if="bookmark.description"
      class="text-xs text-slate-500 dark:text-slate-400 mt-1 line-clamp-2 leading-relaxed"
    >
      {{ bookmark.description }}
    </p>

    <!-- Tags Row (if any) -->
    <div
      v-if="bookmark.tags && bookmark.tags.length > 0"
      class="flex flex-wrap gap-1.5 mt-3 pt-2 border-t border-slate-200/40 dark:border-slate-800/40"
    >
      <span
        v-for="tag in bookmark.tags"
        :key="tag.id"
        class="text-[10px] px-1.5 py-0.5 rounded-md font-medium"
        :style="{
          backgroundColor: `${tag.color}15`,
          color: tag.color,
          border: `1px solid ${tag.color}30`
        }"
      >
        {{ tag.name }}
      </span>
    </div>
  </a>
</template>
