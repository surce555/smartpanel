<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Icon } from '@iconify/vue'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { useTheme } from '@/composables/useTheme'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()
const theme = useTheme()

const email = ref<string>('admin')
const password = ref<string>('admin123')
const errorMsg = ref<string>('')

// Force change password modal state
const showChangeModal = ref<boolean>(false)
const oldPassword = ref<string>('admin123')
const newPassword = ref<string>('')
const confirmPassword = ref<string>('')
const changeMsg = ref<string>('')

// Theme state
const isDarkMode = ref<boolean>(document.documentElement.classList.contains('dark'))

function toggleTheme() {
  const next = isDarkMode.value ? 'light' : 'dark'
  theme.applyTheme(next)
  isDarkMode.value = next === 'dark'
}

const activeWallpaperUrl = computed(() => {
  if (settingsStore.loginWallpaperType === 'upload' && settingsStore.loginWallpaperUrl) {
    return settingsStore.loginWallpaperUrl
  }
  if (settingsStore.loginWallpaperUrl) {
    return settingsStore.loginWallpaperUrl
  }
  if (settingsStore.wallpaperUrl) {
    return settingsStore.wallpaperUrl
  }
  const cachedLogin = localStorage.getItem('smartpanel_login_wallpaper')
  if (cachedLogin) return cachedLogin
  const cachedHome = localStorage.getItem('smartpanel_wallpaper')
  if (cachedHome) return cachedHome
  return ''
})

const wallpaperStyle = computed(() => {
  const url = activeWallpaperUrl.value
  const blur = settingsStore.loginWallpaperBlur || 0
  return {
    backgroundImage: url ? `url(${url})` : 'none',
    filter: blur > 0 ? `blur(${blur}px)` : 'none',
    transform: blur > 0 ? 'scale(1.05)' : 'none',
  }
})

const maskStyle = computed(() => {
  const mask = settingsStore.loginWallpaperMask ?? 0
  const opacity = mask / 100
  return {
    backgroundColor: opacity > 0 ? `rgba(0, 0, 0, ${opacity})` : 'transparent',
  }
})

async function handleLogin() {
  errorMsg.value = ''
  const res = await authStore.login(email.value, password.value)
  if (res.success) {
    if (authStore.mustChangePassword) {
      showChangeModal.value = true
      oldPassword.value = password.value
    } else {
      const redirect = (route.query.redirect as string) || (settingsStore.requireLogin ? '/' : '/admin')
      router.push(redirect)
    }
  } else {
    errorMsg.value = res.error || '登录失败'
  }
}

async function handleChangePassword() {
  changeMsg.value = ''
  if (newPassword.value.length < 6) {
    changeMsg.value = '新密码长度至少需 6 位'
  } else if (newPassword.value !== confirmPassword.value) {
    changeMsg.value = '两次输入的新密码不一致'
  } else {
    const res = await authStore.changePassword(oldPassword.value, newPassword.value)
    if (res.success) {
      showChangeModal.value = false
      const redirect = (route.query.redirect as string) || '/admin'
      router.push(redirect)
    } else {
      changeMsg.value = res.error || '修改密码失败'
    }
  }
}

onMounted(async () => {
  await settingsStore.fetchSettings()
  if (settingsStore.loginWallpaperUrl) {
    localStorage.setItem('smartpanel_login_wallpaper', settingsStore.loginWallpaperUrl)
  }
  if (settingsStore.wallpaperUrl) {
    localStorage.setItem('smartpanel_wallpaper', settingsStore.wallpaperUrl)
  }
})
</script>

<template>
  <div class="relative min-h-[100dvh] flex items-center justify-center p-4 overflow-hidden select-none">
    <!-- Dynamic Fullscreen Wallpaper Background Layer -->
    <div
      class="fixed inset-0 z-0 bg-cover bg-center transition-all duration-700 pointer-events-none bg-slate-950"
      :style="wallpaperStyle"
    ></div>

    <!-- Wallpaper Mask Layer (only when opacity > 0) -->
    <div
      v-if="maskStyle.backgroundColor !== 'transparent'"
      class="fixed inset-0 z-0 transition-all duration-700 pointer-events-none"
      :style="maskStyle"
    ></div>

    <!-- Top Left Discreet Return to Home Link -->
    <router-link
      to="/"
      class="fixed top-4 sm:top-5 left-4 sm:left-5 z-20 p-2 sm:px-3 sm:py-1.5 rounded-xl bg-black/30 hover:bg-black/55 text-white/70 hover:text-white border border-white/15 transition-all text-xs flex items-center gap-1.5"
      title="返回首页"
    >
      <Icon icon="tabler:arrow-left" class="w-4 h-4" />
      <span class="hidden sm:inline">首页</span>
    </router-link>

    <!-- Crystal 100% Pure Transparent Login Card (Matching Image 1) -->
    <div
      class="relative z-10 w-full max-w-[380px] sm:max-w-[400px] p-7 sm:p-8 rounded-[28px] border border-white/40 shadow-2xl space-y-6 transition-all animate-in fade-in zoom-in-95 duration-200"
      style="background: transparent !important; backdrop-filter: none !important; -webkit-backdrop-filter: none !important;"
    >
      <!-- Card Top Bar: Theme Switcher Pill (Left) + Language Selector (Right) -->
      <div class="flex items-center justify-between pt-0.5">
        <!-- Theme Toggle Switch Pill -->
        <button
          type="button"
          @click="toggleTheme"
          class="w-12 h-6 rounded-full bg-black/35 border border-white/20 p-0.5 flex items-center transition-all cursor-pointer"
          :class="isDarkMode ? 'justify-end' : 'justify-start'"
          :title="isDarkMode ? '切换为浅色' : '切换为深色'"
        >
          <span class="w-5 h-5 rounded-full bg-white/90 shadow flex items-center justify-center text-slate-800">
            <Icon :icon="isDarkMode ? 'tabler:moon' : 'tabler:sun'" class="w-3.5 h-3.5" />
          </span>
        </button>

        <!-- Language Pill -->
        <div class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-black/25 border border-white/25 text-white/90 text-xs font-normal">
          <Icon icon="tabler:language" class="w-3.5 h-3.5 opacity-80" />
          <span class="opacity-90">简体中文</span>
          <Icon icon="tabler:chevron-down" class="w-3.5 h-3.5 opacity-60 ml-0.5" />
        </div>
      </div>

      <!-- Login Form -->
      <form @submit.prevent="handleLogin" class="space-y-4 pt-1">
        <!-- Account Input with User Icon (Matching Image 1) -->
        <div class="relative">
          <div class="flex items-center w-full px-3.5 py-2.5 rounded-lg bg-black/25 border border-white/25 focus-within:border-white/60 focus-within:bg-black/35 transition-all">
            <Icon icon="tabler:user" class="w-4 h-4 text-white/70 mr-2.5 shrink-0" />
            <input
              type="text"
              v-model="email"
              placeholder="sunas"
              required
              autocomplete="username"
              class="w-full bg-transparent text-white placeholder-white/40 text-sm focus:outline-none"
            />
          </div>
        </div>

        <!-- Password Input with Lock Icon (Matching Image 1) -->
        <div class="relative">
          <div class="flex items-center w-full px-3.5 py-2.5 rounded-lg bg-black/25 border border-white/25 focus-within:border-white/60 focus-within:bg-black/35 transition-all">
            <Icon icon="tabler:lock" class="w-4 h-4 text-white/70 mr-2.5 shrink-0" />
            <input
              type="password"
              v-model="password"
              placeholder="••••••••••••"
              required
              autocomplete="current-password"
              class="w-full bg-transparent text-white placeholder-white/40 text-sm tracking-widest focus:outline-none"
            />
          </div>
        </div>

        <div v-if="errorMsg" class="p-2.5 rounded-lg bg-rose-500/30 border border-rose-400/40 text-xs text-rose-100">
          {{ errorMsg }}
        </div>

        <!-- Submit Button (Matching Image 1: Thin border, translucent bg, white text) -->
        <button
          type="submit"
          :disabled="authStore.loading"
          class="w-full py-2.5 mt-2 rounded-lg border border-white/35 hover:border-white/60 bg-black/25 hover:bg-black/40 active:scale-[0.99] text-white font-medium text-sm tracking-widest transition-all duration-200 flex items-center justify-center cursor-pointer shadow-sm"
        >
          <span>{{ authStore.loading ? '正在验证...' : '登录' }}</span>
        </button>
      </form>
    </div>

    <!-- Forced Password Reset Modal -->
    <div
      v-if="showChangeModal"
      class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-md"
    >
      <div class="w-full max-w-md p-6 rounded-3xl bg-slate-900 border border-white/20 shadow-2xl space-y-4 text-white">
        <div>
          <h3 class="text-lg font-bold text-white">🔒 首次登录强制修改密码</h3>
          <p class="text-xs text-white/60 mt-1">为了保障 NAS 安全，请设定全新的管理员密码</p>
        </div>

        <div class="space-y-3">
          <div>
            <label class="block text-xs font-medium text-white/80 mb-1">新密码 (至少 6 位)</label>
            <input
              type="password"
              v-model="newPassword"
              class="w-full px-3 py-2 text-sm bg-white/10 rounded-xl border border-white/20 text-white focus:outline-none focus:border-white/50"
            />
          </div>

          <div>
            <label class="block text-xs font-medium text-white/80 mb-1">确认新密码</label>
            <input
              type="password"
              v-model="confirmPassword"
              class="w-full px-3 py-2 text-sm bg-white/10 rounded-xl border border-white/20 text-white focus:outline-none focus:border-white/50"
            />
          </div>

          <div v-if="changeMsg" class="p-2.5 rounded-xl bg-rose-500/20 text-xs text-rose-200">
            {{ changeMsg }}
          </div>
        </div>

        <div class="flex justify-end gap-3 pt-2">
          <button
            type="button"
            @click="handleChangePassword"
            class="px-5 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold rounded-xl transition-colors"
          >
            确认更新密码
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
