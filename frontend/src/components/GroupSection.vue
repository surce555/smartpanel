<script setup lang="ts">
import { ref } from 'vue'
import { Icon } from '@iconify/vue'
import { Group } from '@/stores/bookmarks'
import BookmarkGrid from './BookmarkGrid.vue'

const props = defineProps<{
  group: Group
}>()

const isCollapsed = ref<boolean>(false)
</script>

<template>
  <section class="mb-8 last:mb-2">
    <!-- Group Header -->
    <div
      @click="isCollapsed = !isCollapsed"
      class="flex items-center justify-between mb-3 px-1 py-1 cursor-pointer select-none group"
    >
      <div class="flex items-center gap-2.5">
        <div class="flex items-center justify-center w-7 h-7 rounded-lg bg-indigo-50 dark:bg-indigo-950/60 text-indigo-600 dark:text-indigo-400 group-hover:scale-105 transition-transform">
          <Icon v-if="group.icon" :icon="group.icon" class="w-4 h-4" />
          <Icon v-else icon="tabler:folder" class="w-4 h-4" />
        </div>
        <h2 class="text-base font-bold text-slate-800 dark:text-slate-100 tracking-tight group-hover:text-indigo-600 dark:group-hover:text-indigo-400 transition-colors">
          {{ group.name }}
        </h2>
        <span class="text-xs px-2 py-0.5 rounded-full bg-slate-200/60 dark:bg-slate-800/60 text-slate-500 dark:text-slate-400 font-medium">
          {{ group.bookmarks?.length || 0 }}
        </span>
      </div>

      <!-- Collapse Icon -->
      <button
        type="button"
        class="p-1 rounded-md text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 transition-transform duration-200"
        :class="{ 'rotate-180': isCollapsed }"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
        </svg>
      </button>
    </div>

    <!-- Bookmarks Grid -->
    <div v-show="!isCollapsed" class="transition-all duration-300">
      <BookmarkGrid
        v-if="group.bookmarks && group.bookmarks.length > 0"
        :bookmarks="group.bookmarks"
      />
      <div
        v-else
        class="p-8 rounded-2xl border border-dashed border-slate-300 dark:border-slate-800 text-center text-xs text-slate-400"
      >
        此分组暂无书签
      </div>
    </div>
  </section>
</template>
