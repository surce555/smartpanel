<script setup lang="ts">
import { ref, computed } from 'vue'
import { Icon } from '@iconify/vue'
import { Bookmark, useBookmarksStore } from '@/stores/bookmarks'
import { useSettingsStore } from '@/stores/settings'
import { useAuthStore } from '@/stores/auth'
import HealthBadge from './HealthBadge.vue'

const props = defineProps<{
  bookmark: Bookmark
}>()

const emit = defineEmits<{
  (e: 'edit', bookmark: Bookmark): void
}>()

const bookmarksStore = useBookmarksStore()
const settingsStore = useSettingsStore()
const authStore = useAuthStore()

const isDragging = ref(false)
const isDragOver = ref(false)
let didDrag = false

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

// Dynamic sizing styles based on iconSize / density
const paddingClass = computed(() => {
  switch (settingsStore.iconSize) {
    case 'small':
      return 'p-3'
    case 'large':
    case 'detail':
      return 'p-5'
    case 'medium':
    default:
      return 'p-4'
  }
})

const iconSizeClass = computed(() => {
  switch (settingsStore.iconSize) {
    case 'small':
      return 'w-8 h-8 text-lg p-1.5'
    case 'large':
      return 'w-12 h-12 text-3xl p-2.5'
    case 'detail':
      return 'w-14 h-14 text-4xl p-3'
    case 'medium':
    default:
      return 'w-10 h-10 text-2xl p-2'
  }
})

const titleSizeClass = computed(() => {
  switch (settingsStore.iconSize) {
    case 'small':
      return 'text-xs font-semibold'
    case 'large':
    case 'detail':
      return 'text-base font-bold'
    case 'medium':
    default:
      return 'text-sm font-semibold'
  }
})

const descLinesClass = computed(() => {
  switch (settingsStore.iconSize) {
    case 'small':
      return 'text-[11px] line-clamp-1 mt-0.5'
    case 'large':
    case 'detail':
      return 'text-xs line-clamp-3 mt-1.5 leading-relaxed'
    case 'medium':
    default:
      return 'text-xs line-clamp-2 mt-1 leading-relaxed'
  }
})

function handleDragStart(e: DragEvent) {
  didDrag = true
  isDragging.value = true
  bookmarksStore.draggingBookmarkId = props.bookmark.id
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.setData('text/plain', props.bookmark.id)
  }
}

function handleDragEnd() {
  isDragging.value = false
  bookmarksStore.draggingBookmarkId = null
  setTimeout(() => {
    didDrag = false
  }, 120)
}

function handleDragOver(e: DragEvent) {
  if (bookmarksStore.draggingBookmarkId && bookmarksStore.draggingBookmarkId !== props.bookmark.id) {
    if (e.dataTransfer) {
      e.dataTransfer.dropEffect = 'move'
    }
    isDragOver.value = true
  }
}

function handleDragLeave() {
  isDragOver.value = false
}

async function handleDrop(e: DragEvent) {
  isDragOver.value = false
  const sourceId = bookmarksStore.draggingBookmarkId || e.dataTransfer?.getData('text/plain')
  if (sourceId && sourceId !== props.bookmark.id) {
    await bookmarksStore.moveBookmark(sourceId, props.bookmark.id, props.bookmark.group_id)
  }
}

function handleClick(e: MouseEvent) {
  if (didDrag) {
    e.preventDefault()
    return
  }
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
    draggable="true"
    @dragstart="handleDragStart"
    @dragend="handleDragEnd"
    @dragover.prevent="handleDragOver"
    @dragleave="handleDragLeave"
    @drop.prevent="handleDrop"
    @click="handleClick"
    class="group relative flex flex-col transition-all duration-300 cursor-pointer overflow-hidden select-none touch-manipulation"
    :class="[
      cardStyleClass,
      shadowClass,
      paddingClass,
      isDragging ? 'opacity-35 scale-95 border-2 border-dashed border-indigo-400' : 'hover:-translate-y-1 hover:shadow-xl',
      isDragOver ? 'ring-2 ring-indigo-500 bg-indigo-50/70 dark:bg-indigo-950/40 scale-[1.02] shadow-xl' : ''
    ]"
    :style="{ borderRadius: `${settingsStore.cardBorderRadius}px` }"
  >
    <!-- Hover Action Handle (Drag grip + Quick edit) -->
    <div
      class="absolute top-2 right-2 flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity duration-200 z-10"
    >
      <button
        v-if="authStore.isAuthenticated"
        type="button"
        @click.prevent.stop="emit('edit', bookmark)"
        class="p-1 rounded-md bg-white/90 dark:bg-slate-800/90 text-slate-400 hover:text-indigo-600 dark:hover:text-indigo-400 shadow-xs hover:scale-110 transition-all"
        title="编辑此书签卡片"
      >
        <Icon icon="tabler:pencil" class="w-3.5 h-3.5" />
      </button>

      <span
        class="p-1 rounded-md bg-white/90 dark:bg-slate-800/90 text-slate-400 dark:text-slate-400 cursor-grab active:cursor-grabbing hover:text-indigo-600 shadow-xs"
        title="长按拖拽以调整位置"
      >
        <Icon icon="tabler:grip-vertical" class="w-3.5 h-3.5" />
      </span>
    </div>

    <!-- Top row: Icon + Health Badge -->
    <div class="flex items-start justify-between gap-3 mb-2">
      <!-- Icon Container -->
      <div
        class="flex items-center justify-center rounded-xl group-hover:scale-105 transition-transform duration-300 overflow-hidden shadow-xs shrink-0"
        :class="[
          iconSizeClass,
          settingsStore.cardStyle === 'transparent'
            ? 'bg-black/35 backdrop-blur-md text-white border border-white/20'
            : 'bg-slate-100/80 dark:bg-slate-800/80 text-indigo-600 dark:text-indigo-400'
        ]"
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

      <!-- Health Status Badge & Private Lock Indicator -->
      <div class="flex items-center gap-1.5 pt-0.5">
        <HealthBadge
          v-if="bookmark.health_check"
          :status="bookmark.health_check.status"
          :response-time="bookmark.health_check.response_time_ms"
        />
        <span
          v-if="bookmark.is_private"
          title="私有书签（仅登录可见）"
          :class="settingsStore.cardStyle === 'transparent' ? 'text-white/80' : 'text-slate-400 dark:text-slate-500'"
        >
          <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <rect x="3" y="11" width="18" height="11" rx="2" ry="2" stroke-width="2"/>
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 11V7a5 5 0 0110 0v4"/>
          </svg>
        </span>
      </div>
    </div>

    <!-- Title -->
    <h3
      class="transition-colors truncate tracking-wide"
      :class="[
        titleSizeClass,
        settingsStore.cardStyle === 'transparent'
          ? 'text-white font-semibold drop-shadow-[0_2px_4px_rgba(0,0,0,0.95)]'
          : 'text-slate-800 dark:text-slate-100 group-hover:text-indigo-600 dark:group-hover:text-indigo-400'
      ]"
    >
      {{ bookmark.name }}
    </h3>

    <!-- Description (if any) -->
    <p
      v-if="bookmark.description"
      class="transition-colors"
      :class="[
        descLinesClass,
        settingsStore.cardStyle === 'transparent'
          ? 'text-white/85 drop-shadow-[0_1px_2px_rgba(0,0,0,0.85)]'
          : 'text-slate-500 dark:text-slate-400'
      ]"
    >
      {{ bookmark.description }}
    </p>

    <!-- Tags Row (if any) -->
    <div
      v-if="bookmark.tags && bookmark.tags.length > 0 && settingsStore.iconSize !== 'small'"
      class="flex flex-wrap gap-1.5 mt-2.5 pt-2 border-t border-slate-200/40 dark:border-slate-800/40"
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
