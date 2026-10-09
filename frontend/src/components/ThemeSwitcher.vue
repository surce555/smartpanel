<script setup lang="ts">
import { useTheme, ThemeMode } from '@/composables/useTheme'
import { useSettingsStore } from '@/stores/settings'

const { currentTheme, applyTheme } = useTheme()
const settingsStore = useSettingsStore()

function selectTheme(mode: ThemeMode) {
  applyTheme(mode)
  settingsStore.themeMode = mode
}
</script>

<template>
  <div class="inline-flex items-center p-1 bg-slate-200/60 dark:bg-slate-800/60 backdrop-blur-md rounded-full border border-slate-300/40 dark:border-slate-700/40 text-xs">
    <!-- Light -->
    <button
      @click="selectTheme('light')"
      :class="[
        'p-1.5 rounded-full transition-all duration-200 flex items-center justify-center',
        currentTheme === 'light'
          ? 'bg-white text-amber-500 shadow-sm'
          : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
      ]"
      title="亮色模式"
    >
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <circle cx="12" cy="12" r="5" stroke-width="2"/>
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 1v2m0 18v2M4.22 4.22l1.42 1.42m12.72 12.72l1.42 1.42M1 12h2m18 0h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42"/>
      </svg>
    </button>

    <!-- Dark -->
    <button
      @click="selectTheme('dark')"
      :class="[
        'p-1.5 rounded-full transition-all duration-200 flex items-center justify-center',
        currentTheme === 'dark'
          ? 'bg-slate-700 text-indigo-400 shadow-sm'
          : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
      ]"
      title="深色模式"
    >
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20.354 15.354A9 9 0 018.646 3.646 9.003 9.003 0 0012 21a9.03 9.03 0 008.354-5.646z"/>
      </svg>
    </button>

    <!-- System -->
    <button
      @click="selectTheme('system')"
      :class="[
        'p-1.5 rounded-full transition-all duration-200 flex items-center justify-center',
        currentTheme === 'system'
          ? 'bg-white dark:bg-slate-700 text-indigo-600 dark:text-indigo-400 shadow-sm'
          : 'text-slate-600 dark:text-slate-400 hover:text-slate-900 dark:hover:text-white'
      ]"
      title="跟随系统"
    >
      <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
        <rect x="2" y="3" width="20" height="14" rx="2" stroke-width="2"/>
        <path stroke-linecap="round" stroke-width="2" d="M8 21h8m-4-4v4"/>
      </svg>
    </button>
  </div>
</template>
