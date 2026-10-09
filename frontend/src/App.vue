<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { useSettingsStore } from '@/stores/settings'
import { useTheme } from '@/composables/useTheme'
import { useIPv6Probe } from '@/composables/useIPv6Probe'
import { useAuthStore } from '@/stores/auth'

const settingsStore = useSettingsStore()
const { initTheme } = useTheme()
const { probeIPv6 } = useIPv6Probe()
const authStore = useAuthStore()

const wallpaperStyle = computed(() => {
  if (settingsStore.wallpaperType === 'upload' && settingsStore.wallpaperUrl) {
    return {
      backgroundImage: `url(${settingsStore.wallpaperUrl})`,
      backgroundSize: 'cover',
      backgroundPosition: 'center',
      filter: settingsStore.wallpaperBlur > 0 ? `blur(${settingsStore.wallpaperBlur}px)` : 'none',
      transform: settingsStore.wallpaperBlur > 0 ? 'scale(1.05)' : 'none',
    }
  }
  if (settingsStore.wallpaperType === 'unsplash') {
    return {
      backgroundImage: `url(https://images.unsplash.com/photo-1507525428034-b723cf961d3e?auto=format&fit=crop&w=1920&q=80)`,
      backgroundSize: 'cover',
      backgroundPosition: 'center',
      filter: settingsStore.wallpaperBlur > 0 ? `blur(${settingsStore.wallpaperBlur}px)` : 'none',
      transform: settingsStore.wallpaperBlur > 0 ? 'scale(1.05)' : 'none',
    }
  }
  return {}
})

const maskStyle = computed(() => {
  const opacity = (settingsStore.wallpaperMask || 0) / 100
  return {
    backgroundColor: `rgba(0, 0, 0, ${opacity})`,
  }
})

onMounted(async () => {
  initTheme()
  await settingsStore.fetchSettings()
  await settingsStore.fetchNetworkInfo()
  settingsStore.setupSSE()
  // Probe IPv6 in background
  probeIPv6()
  // Check auth
  authStore.fetchMe()
})
</script>

<template>
  <div class="relative min-h-screen font-sans selection:bg-indigo-500 selection:text-white overflow-x-hidden">
    <!-- Dynamic Wallpaper Background Layer -->
    <div
      v-if="settingsStore.wallpaperType === 'upload' || settingsStore.wallpaperType === 'unsplash'"
      class="fixed inset-0 -z-20 transition-all duration-700 pointer-events-none"
      :style="wallpaperStyle"
    ></div>

    <!-- Wallpaper Dark/Light Overlay Mask -->
    <div
      v-if="settingsStore.wallpaperType === 'upload' || settingsStore.wallpaperType === 'unsplash'"
      class="fixed inset-0 -z-10 pointer-events-none transition-all duration-300"
      :style="maskStyle"
    ></div>

    <!-- Gradient Background (if selected) -->
    <div
      v-else-if="settingsStore.wallpaperType === 'gradient'"
      class="fixed inset-0 -z-20 bg-gradient-to-br from-indigo-50/50 via-white to-slate-100 dark:from-slate-950 dark:via-indigo-950/20 dark:to-slate-900 pointer-events-none"
    ></div>

    <!-- App Router Content -->
    <router-view />
  </div>
</template>
