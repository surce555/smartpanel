import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useApi } from '@/composables/useApi'

export interface User {
  id: string
  email: string
  must_change_password: boolean
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string | null>(localStorage.getItem('smartpanel_token'))
  const user = ref<User | null>(null)
  const loading = ref<boolean>(false)

  const isAuthenticated = computed(() => !!token.value)
  const mustChangePassword = computed(() => user.value?.must_change_password ?? false)

  async function login(email: string, password: string):Promise<{ success: boolean; error?: string }> {
    loading.value = true
    const api = useApi()
    try {
      const resp = await api.post('/auth/login', { email, password })
      token.value = resp.data.token
      user.value = resp.data.user
      localStorage.setItem('smartpanel_token', resp.data.token)
      return { success: true }
    } catch (err: any) {
      return {
        success: false,
        error: err.response?.data?.error || '登录失败，请检查邮箱和密码',
      }
    } finally {
      loading.value = false
    }
  }

  async function fetchMe() {
    if (!token.value) return
    const api = useApi()
    try {
      const resp = await api.get('/auth/me')
      user.value = resp.data
    } catch (err) {
      logout()
    }
  }

  async function changePassword(oldPassword: string, newPassword: string): Promise<{ success: boolean; error?: string }> {
    const api = useApi()
    try {
      await api.put('/auth/password', {
        old_password: oldPassword,
        new_password: newPassword,
      })
      if (user.value) {
        user.value.must_change_password = false
      }
      return { success: true }
    } catch (err: any) {
      return {
        success: false,
        error: err.response?.data?.error || '修改密码失败',
      }
    }
  }

  function logout() {
    token.value = null
    user.value = null
    localStorage.removeItem('smartpanel_token')
  }

  return {
    token,
    user,
    loading,
    isAuthenticated,
    mustChangePassword,
    login,
    fetchMe,
    changePassword,
    logout,
  }
})
