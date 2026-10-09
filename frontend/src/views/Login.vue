<script setup lang="ts">
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { Icon } from '@iconify/vue'
import { useAuthStore } from '@/stores/auth'
import { useSettingsStore } from '@/stores/settings'

const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const settingsStore = useSettingsStore()

const email = ref<string>('admin')
const password = ref<string>('admin123')
const errorMsg = ref<string>('')

// Force change password modal state
const showChangeModal = ref<boolean>(false)
const oldPassword = ref<string>('admin123')
const newPassword = ref<string>('')
const confirmPassword = ref<string>('')
const changeMsg = ref<string>('')

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
</script>

<template>
  <div class="min-h-screen flex items-center justify-center p-4">
    <div class="w-full max-w-md p-8 rounded-3xl bg-white/80 dark:bg-slate-900/80 backdrop-blur-2xl border border-slate-200/80 dark:border-slate-800/80 shadow-2xl">
      <!-- Header -->
      <div class="text-center mb-8">
        <div class="inline-flex items-center justify-center w-12 h-12 rounded-2xl bg-indigo-600 text-white shadow-lg shadow-indigo-600/30 mb-3">
          <Icon icon="tabler:layout-dashboard" class="w-6 h-6" />
        </div>
        <h2 class="text-2xl font-bold text-slate-800 dark:text-slate-100">SmartPanel</h2>
        <p class="text-xs text-slate-500 mt-1">管理员登录认证</p>
      </div>

      <!-- Login Form -->
      <form @submit.prevent="handleLogin" class="space-y-4">
        <div>
          <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">管理员账号 (用户名或邮箱)</label>
          <input
            type="text"
            v-model="email"
            placeholder="请输入管理员账号 (如 admin)"
            required
            class="w-full px-4 py-2.5 text-sm bg-slate-50 dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 text-slate-800 dark:text-slate-100"
          />
        </div>

        <div>
          <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1.5">账户密码</label>
          <input
            type="password"
            v-model="password"
            placeholder="请输入管理员密码"
            required
            class="w-full px-4 py-2.5 text-sm bg-slate-50 dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 focus:outline-none focus:ring-2 focus:ring-indigo-500 text-slate-800 dark:text-slate-100"
          />
        </div>

        <div v-if="errorMsg" class="p-3 rounded-xl bg-rose-50 dark:bg-rose-950/40 border border-rose-200 dark:border-rose-900/50 text-xs text-rose-600 dark:text-rose-400">
          {{ errorMsg }}
        </div>

        <button
          type="submit"
          :disabled="authStore.loading"
          class="w-full py-3 bg-indigo-600 hover:bg-indigo-700 text-white font-semibold text-sm rounded-xl shadow-lg shadow-indigo-600/25 transition-all duration-200 flex items-center justify-center"
        >
          <span>{{ authStore.loading ? '正在验证...' : '登 录' }}</span>
        </button>
      </form>

      <!-- Default tips -->
      <div class="mt-6 pt-4 border-t border-slate-100 dark:border-slate-800/80 text-center">
        <p class="text-[11px] text-slate-400">默认初始化账号：admin / admin123 (或 admin@smartpanel.local)</p>
        <p class="text-[11px] text-slate-400 mt-0.5">首次登录将强制重置管理员密码，登录后可在安全设置中自定义账号名</p>
      </div>

      <div class="mt-4 text-center">
        <router-link to="/" class="text-xs text-indigo-600 dark:text-indigo-400 hover:underline">
          ← 返回导航首页
        </router-link>
      </div>
    </div>

    <!-- Forced Password Reset Modal -->
    <div
      v-if="showChangeModal"
      class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-md"
    >
      <div class="w-full max-w-md p-6 rounded-3xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-2xl space-y-4">
        <div>
          <h3 class="text-lg font-bold text-slate-800 dark:text-slate-100">🔒 首次登录强制修改密码</h3>
          <p class="text-xs text-slate-500 mt-1">为了保障 NAS 安全，请设定全新的管理员密码</p>
        </div>

        <div class="space-y-3">
          <div>
            <label class="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">新密码 (至少 6 位)</label>
            <input
              type="password"
              v-model="newPassword"
              class="w-full px-3 py-2 text-sm bg-slate-50 dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700"
            />
          </div>

          <div>
            <label class="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">确认新密码</label>
            <input
              type="password"
              v-model="confirmPassword"
              class="w-full px-3 py-2 text-sm bg-slate-50 dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700"
            />
          </div>

          <div v-if="changeMsg" class="text-xs text-rose-600">
            {{ changeMsg }}
          </div>
        </div>

        <button
          type="button"
          @click="handleChangePassword"
          class="w-full py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl text-xs font-semibold transition-colors"
        >
          确认修改并进入面板
        </button>
      </div>
    </div>
  </div>
</template>
