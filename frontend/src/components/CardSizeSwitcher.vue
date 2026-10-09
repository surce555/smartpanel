<script setup lang="ts">
import { computed } from 'vue'
import { Icon } from '@iconify/vue'
import { useSettingsStore } from '@/stores/settings'
import { useAuthStore } from '@/stores/auth'

const settingsStore = useSettingsStore()
const authStore = useAuthStore()

const currentSize = computed(() => {
  if (settingsStore.iconSize === 'small' || settingsStore.gridColsDesktop >= 7) {
    return 'small'
  }
  if (settingsStore.iconSize === 'large' || settingsStore.gridColsDesktop <= 4) {
    return 'large'
  }
  return 'medium'
})

const options = [
  { id: 'small', label: '小', title: '紧凑小号 (密集小网格)', icon: 'tabler:grid-dots', cols: 8 },
  { id: 'medium', label: '中', title: '标准中号 (默认平衡)', icon: 'tabler:layout-grid', cols: 6 },
  { id: 'large', label: '大', title: '舒适大号 (大卡片宽排版)', icon: 'tabler:layout-cards', cols: 4 },
]

async function selectSize(sizeId: string, cols: number) {
  settingsStore.iconSize = sizeId
  settingsStore.gridColsDesktop = cols
  if (sizeId === 'small') {
    settingsStore.gridColsTablet = 4
    settingsStore.gridColsMobile = 2
  } else if (sizeId === 'large') {
    settingsStore.gridColsTablet = 2
    settingsStore.gridColsMobile = 1
  } else {
    settingsStore.gridColsTablet = 3
    settingsStore.gridColsMobile = 2
  }

  // Save to local storage for guests
  localStorage.setItem('smartpanel_icon_size', sizeId)
  localStorage.setItem('smartpanel_grid_cols', String(cols))

  // If authenticated, persist to server settings
  if (authStore.isAuthenticated) {
    try {
      await settingsStore.saveSettings({
        icon_size: sizeId,
        grid_cols_desktop: String(cols),
        grid_cols_tablet: String(settingsStore.gridColsTablet),
        grid_cols_mobile: String(settingsStore.gridColsMobile),
      })
    } catch (e) {
      // ignore
    }
  }
}
</script>

<template>
  <div
    class="flex items-center p-1 rounded-full bg-white/70 dark:bg-slate-800/70 backdrop-blur-md border border-slate-200/80 dark:border-slate-700/80 shadow-xs"
    title="调整卡片标签大小与网格密集度"
  >
    <button
      v-for="opt in options"
      :key="opt.id"
      type="button"
      @click="selectSize(opt.id, opt.cols)"
      :title="opt.title"
      class="flex items-center gap-1 px-2.5 py-1 rounded-full text-xs font-semibold transition-all duration-200"
      :class="[
        currentSize === opt.id
          ? 'bg-indigo-600 text-white shadow-xs scale-105'
          : 'text-slate-500 hover:text-slate-800 dark:hover:text-slate-200 hover:bg-slate-100/60 dark:hover:bg-slate-700/60'
      ]"
    >
      <Icon :icon="opt.icon" class="w-3.5 h-3.5" />
      <span class="text-[11px]">{{ opt.label }}</span>
    </button>
  </div>
</template>
