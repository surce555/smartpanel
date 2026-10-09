<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Icon } from '@iconify/vue'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'
import { useTheme } from '@/composables/useTheme'
import LoginWallpaperModal from '@/components/LoginWallpaperModal.vue'

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

// Login Wallpaper Modal state
const showWallpaperModal = ref<boolean>(false)
const localWallpaperOverride = ref<string>('')

// Theme state
const isDarkMode = ref<boolean>(document.documentElement.classList.contains('dark'))

function toggleTheme() {
  const next = isDarkMode.value ? 'light' : 'dark'
  theme.applyTheme(next)
  isDarkMode.value = next === 'dark'
}

const activeWallpaperUrl = computed(() => {
  if (localWallpaperOverride.value) return localWallpaperOverride.value

  const cachedLocal = localStorage.getItem('smartpanel_login_wallpaper_local')
  if (cachedLocal) return cachedLocal

  const cachedUrl = localStorage.getItem('smartpanel_login_wallpaper_url')
  if (cachedUrl) return cachedUrl

  if (settingsStore.loginWallpaperType === 'follow') {
    return settingsStore.wallpaperUrl || '/wallpapers/home_cafe_girl.png'
  }
  return settingsStore.loginWallpaperUrl || '/wallpapers/login_anime_girl.png'
})

const wallpaperStyle = computed(() => {
  const blur = settingsStore.loginWallpaperBlur || 0
  return {
    backgroundImage: `url(${activeWallpaperUrl.value})`,
    filter: blur > 0 ? `blur(${blur}px)` : 'none',
    transform: blur > 0 ? 'scale(1.05)' : 'none',
  }
})

const maskStyle = computed(() => {
  const mask = settingsStore.loginWallpaperMask ?? 15
  const opacity = mask / 100
  return {
    backgroundColor: `rgba(0, 0, 0, ${opacity})`,
  }
})

function onWallpaperChanged(url: string) {
  localWallpaperOverride.value = url
}

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
    return
  }
  if (newPassword.value !== confirmPassword.value) {
    changeMsg.value = '两次输入的新密码不一致'
    return
  }

  const res = await authStore.changePassword(oldPassword.value, newPassword.value)
  if (res.success) {
    showChangeModal.value = false
    const redirect = (route.query.redirect as string) || '/admin'
    router.push(redirect)
  } else {
    changeMsg.value = res.error || '修改密码失败'
  }
}

onMounted(async () => {
  await settingsStore.fetchSettings()
})
</script>

<template>
  <div class="relative min-h-[100dvh] flex items-center justify-center p-4 overflow-hidden select-none">
    <!-- Dynamic Fullscreen Wallpaper Background Layer -->
    <div
      class="fixed inset-0 -z-20 bg-cover bg-center transition-all duration-700 pointer-events-none"
      :style="wallpaperStyle"
    ></div>

    <!-- Wallpaper Mask Layer -->
    <div
      class="fixed inset-0 -z-10 transition-all duration-700 pointer-events-none"
      :style="maskStyle"
    ></div>

    <!-- Top Floating Controls: Quick Wallpaper Switcher -->
    <div class="fixed top-4 right-4 z-20 flex items-center gap-2">
      <button
        type="button"
        @click="showWallpaperModal = true"
        class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full bg-black/40 hover:bg-black/60 text-white/90 hover:text-white backdrop-blur-md border border-white/20 text-xs font-medium shadow-lg hover:scale-105 transition-all cursor-pointer"
        title="更换登录页壁纸"
      >
        <Icon icon="tabler:photo" class="w-3.5 h-3.5" />
        <span>更换壁纸</span>
      </button>
    </div>

    <!-- Crystal Transparent Login Card (Matching Image 1) -->
    <div
      class="w-full max-w-sm sm:max-w-md p-7 sm:p-8 rounded-3xl border border-white/30 bg-black/20 dark:bg-black/35 backdrop-blur-md shadow-2xl space-y-6 transition-all animate-in fade-in zoom-in-95 duration-200"
    >
      <!-- Card Top Bar: Theme Switcher + Language Selector (as shown in Image 1) -->
      <div class="flex items-center justify-between pt-1">
        <!-- Theme Toggle Switch Pill -->
        <button
          type="button"
          @click="toggleTheme"
          class="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full bg-white/10 hover:bg-white/20 border border-white/20 text-white text-xs cursor-pointer transition-all"
          :title="isDarkMode ? '切换为浅色模式' : '切换为深色模式'"
        >
          <Icon :icon="isDarkMode ? 'tabler:moon' : 'tabler:sun'" class="w-4 h-4 text-white" />
        </button>

        <!-- Language Pill -->
        <div class="inline-flex items-center gap-1.5 px-3 py-1 rounded-xl bg-white/10 border border-white/25 text-white/90 text-xs font-normal">
          <Icon icon="tabler:language" class="w-3.5 h-3.5" />
          <span>简体中文</span>
          <Icon icon="tabler:chevron-down" class="w-3 h-3 opacity-70" />
        </div>
      </div>

      <!-- Login Form -->
      <form @submit.prevent="handleLogin" class="space-y-4">
        <!-- Account Input with User Icon (Matching Image 1) -->
        <div class="relative">
          <div class="flex items-center w-full px-3.5 py-2.5 rounded-xl bg-white/10 dark:bg-black/25 border border-white/20 focus-within:border-white/50 focus-within:bg-white/15 transition-all">
            <Icon icon="tabler:user" class="w-4 h-4 text-white/70 mr-2.5 shrink-0" />
            <input
              type="text"
              v-model="email"
              placeholder="sunas (管理员账号)"
              required
              autocomplete="username"
              class="w-full bg-transparent text-white placeholder-white/45 text-base sm:text-sm focus:outline-none"
            />
          </div>
        </div>

        <!-- Password Input with Lock Icon (Matching Image 1) -->
        <div class="relative">
          <div class="flex items-center w-full px-3.5 py-2.5 rounded-xl bg-white/10 dark:bg-black/25 border border-white/20 focus-within:border-white/50 focus-within:bg-white/15 transition-all">
            <Icon icon="tabler:lock" class="w-4 h-4 text-white/70 mr-2.5 shrink-0" />
            <input
              type="password"
              v-model="password"
              placeholder="••••••••••••"
              required
              autocomplete="current-password"
              class="w-full bg-transparent text-white placeholder-white/45 text-base sm:text-sm tracking-widest focus:outline-none"
            />
          </div>
        </div>

        <div v-if="errorMsg" class="p-2.5 rounded-xl bg-rose-500/20 border border-rose-400/40 text-xs text-rose-200">
          {{ errorMsg }}
        </div>

        <!-- Submit Button (Matching Image 1: Thin border, translucent background, white text) -->
        <button
          type="submit"
          :disabled="authStore.loading"
          class="w-full py-2.5 mt-2 rounded-xl border border-white/40 hover:border-white/70 bg-white/10 hover:bg-white/25 active:scale-[0.99] text-white font-medium text-sm tracking-widest transition-all duration-200 flex items-center justify-center cursor-pointer shadow-sm"
        >
          <span>{{ authStore.loading ? '正在验证...' : '登 录' }}</span>
        </button>
      </form>

      <!-- Footer navigation -->
      <div class="pt-3 border-t border-white/10 text-center">
        <router-link
          to="/"
          class="inline-flex items-center gap-1 text-xs text-white/70 hover:text-white transition-colors"
        >
          <Icon icon="tabler:arrow-left" class="w-3.5 h-3.5" />
          <span>返回导航首页</span>
        </router-link>
      </div>
    </div>

    <!-- Wallpaper Modal -->
    <LoginWallpaperModal
      :show="showWallpaperModal"
      @close="showWallpaperModal = false"
      @changed="onWallpaperChanged"
    />

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
