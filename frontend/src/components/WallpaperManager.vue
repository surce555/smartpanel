<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Icon } from '@iconify/vue'
import { useSettingsStore } from '@/stores/settings'
import { useApi } from '@/composables/useApi'

const settingsStore = useSettingsStore()
const api = useApi()

const uploadedWallpapers = ref<any[]>([])
const uploadingHome = ref<boolean>(false)
const uploadingLogin = ref<boolean>(false)
const uploadMsg = ref<string>('')
const saveMsg = ref<string>('')

async function fetchUploadedList() {
  try {
    const resp = await api.get('/upload/list?category=wallpapers')
    uploadedWallpapers.value = resp.data
  } catch (e) {
    console.error('Failed to fetch uploaded wallpapers:', e)
  }
}

async function handleHomeUpload(e: Event) {
  const target = e.target as HTMLInputElement
  if (!target.files || target.files.length === 0) return

  const file = target.files[0]
  const formData = new FormData()
  formData.append('file', file)

  uploadingHome.value = true
  uploadMsg.value = ''
  try {
    const resp = await api.post('/upload/wallpaper', formData)
    settingsStore.wallpaperUrl = resp.data.url
    settingsStore.wallpaperType = 'upload'
    uploadMsg.value = '首页壁纸上传成功！'
    await fetchUploadedList()
    await handleSave()
  } catch (err: any) {
    uploadMsg.value = err.response?.data?.error || '上传失败'
  } finally {
    uploadingHome.value = false
    target.value = ''
  }
}

async function handleLoginUpload(e: Event) {
  const target = e.target as HTMLInputElement
  if (!target.files || target.files.length === 0) return

  const file = target.files[0]
  const formData = new FormData()
  formData.append('file', file)

  uploadingLogin.value = true
  uploadMsg.value = ''
  try {
    const resp = await api.post('/upload/wallpaper', formData)
    settingsStore.loginWallpaperUrl = resp.data.url
    settingsStore.loginWallpaperType = 'upload'
    uploadMsg.value = '登录页壁纸上传成功！'
    await fetchUploadedList()
    await handleSave()
  } catch (err: any) {
    uploadMsg.value = err.response?.data?.error || '上传失败'
  } finally {
    uploadingLogin.value = false
    target.value = ''
  }
}

async function deleteWallpaper(filename: string) {
  if (!confirm('确认从服务器删除此壁纸文件？')) return
  try {
    await api.delete(`/upload/${filename}?category=wallpapers`)
    await fetchUploadedList()
  } catch (e) {
    console.error('Delete wallpaper failed:', e)
  }
}

function selectAsHomeWallpaper(url: string) {
  settingsStore.wallpaperUrl = url
  settingsStore.wallpaperType = 'upload'
}

function selectAsLoginWallpaper(url: string) {
  settingsStore.loginWallpaperUrl = url
  settingsStore.loginWallpaperType = 'upload'
}

async function handleSave() {
  saveMsg.value = ''
  try {
    await settingsStore.saveSettings({
      wallpaper_type: settingsStore.wallpaperType,
      wallpaper_url: settingsStore.wallpaperUrl,
      wallpaper_blur: String(settingsStore.wallpaperBlur),
      wallpaper_mask: String(settingsStore.wallpaperMask),
      login_wallpaper_type: settingsStore.loginWallpaperType,
      login_wallpaper_url: settingsStore.loginWallpaperUrl,
      login_wallpaper_blur: String(settingsStore.loginWallpaperBlur),
      login_wallpaper_mask: String(settingsStore.loginWallpaperMask),
    })
    saveMsg.value = '壁纸与背景设置已成功保存！'
    setTimeout(() => {
      saveMsg.value = ''
    }, 3000)
  } catch (err: any) {
    saveMsg.value = '保存失败，请检查网络连接'
  }
}

onMounted(() => {
  fetchUploadedList()
})
</script>

<template>
  <div class="space-y-7">
    <!-- Top Action notice -->
    <div v-if="saveMsg" class="p-3 rounded-xl bg-emerald-500/10 border border-emerald-500/20 text-emerald-600 dark:text-emerald-400 text-xs font-medium flex items-center gap-2">
      <Icon icon="tabler:check" class="w-4 h-4" />
      <span>{{ saveMsg }}</span>
    </div>

    <!-- 1. Home Wallpaper Configuration -->
    <div class="p-5 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200/80 dark:border-slate-800/80 space-y-5">
      <div class="flex items-center justify-between border-b border-slate-100 dark:border-slate-800 pb-3">
        <div>
          <h3 class="text-sm font-bold text-slate-800 dark:text-slate-100 flex items-center gap-2">
            <Icon icon="tabler:photo" class="w-4 h-4 text-indigo-600" />
            <span>首页壁纸背景设置</span>
          </h3>
          <p class="text-xs text-slate-500 mt-0.5">配置主导航界面的全局背景壁纸及显示效果</p>
        </div>
      </div>

      <!-- Mode Selection -->
      <div>
        <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-2">背景呈现模式</label>
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-2.5">
          <button
            type="button"
            v-for="mode in [
              { id: 'upload', name: '自定义/本地壁纸', icon: 'tabler:photo' },
              { id: 'gradient', name: '系统精致渐变', icon: 'tabler:color-swatch' },
              { id: 'none', name: '极简纯色背景', icon: 'tabler:box' },
              { id: 'unsplash', name: '网络随机风景', icon: 'tabler:world' }
            ]"
            :key="mode.id"
            @click="settingsStore.wallpaperType = mode.id"
            class="p-2.5 rounded-xl border text-center transition-all cursor-pointer flex flex-col items-center gap-1"
            :class="[
              settingsStore.wallpaperType === mode.id
                ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400 font-semibold shadow-xs'
                : 'border-slate-200 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-800/50 text-slate-600 dark:text-slate-400'
            ]"
          >
            <Icon :icon="mode.icon" class="w-4 h-4" />
            <span class="text-xs">{{ mode.name }}</span>
          </button>
        </div>
      </div>

      <!-- Upload / Custom URL inputs -->
      <div v-if="settingsStore.wallpaperType === 'upload'" class="space-y-3 pt-2">
        <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-3">
          <div class="flex-1">
            <label class="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">壁纸图片 URL (可直接输入网络链接或本地上传)</label>
            <input
              type="text"
              v-model="settingsStore.wallpaperUrl"
              placeholder="https://example.com/wallpaper.jpg 或选择右侧上传"
              class="w-full px-3 py-2 text-xs bg-slate-50 dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 text-slate-800 dark:text-slate-200 focus:outline-none focus:border-indigo-500"
            />
          </div>
          <div class="sm:self-end">
            <label class="inline-flex items-center justify-center gap-1.5 px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold rounded-xl cursor-pointer transition-colors shadow-xs">
              <Icon icon="tabler:upload" class="w-4 h-4" />
              <span>{{ uploadingHome ? '上传中...' : '上传新壁纸' }}</span>
              <input type="file" accept="image/*" class="hidden" @change="handleHomeUpload" :disabled="uploadingHome" />
            </label>
          </div>
        </div>

        <!-- Sliders: Blur & Mask -->
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-5 pt-3 border-t border-slate-100 dark:border-slate-800">
          <div>
            <div class="flex justify-between text-xs mb-1.5 font-medium text-slate-600 dark:text-slate-400">
              <span>首页壁纸模糊度</span>
              <span class="font-mono text-indigo-600">{{ settingsStore.wallpaperBlur }}px</span>
            </div>
            <input
              type="range"
              min="0"
              max="20"
              step="1"
              v-model.number="settingsStore.wallpaperBlur"
              class="w-full accent-indigo-600 cursor-pointer"
            />
          </div>

          <div>
            <div class="flex justify-between text-xs mb-1.5 font-medium text-slate-600 dark:text-slate-400">
              <span>首页遮罩暗度</span>
              <span class="font-mono text-indigo-600">{{ settingsStore.wallpaperMask }}%</span>
            </div>
            <input
              type="range"
              min="0"
              max="80"
              step="5"
              v-model.number="settingsStore.wallpaperMask"
              class="w-full accent-indigo-600 cursor-pointer"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- 2. Login Page Wallpaper Settings (Merged into Settings as requested) -->
    <div class="p-5 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200/80 dark:border-slate-800/80 space-y-5">
      <div class="flex items-center justify-between border-b border-slate-100 dark:border-slate-800 pb-3">
        <div>
          <h3 class="text-sm font-bold text-slate-800 dark:text-slate-100 flex items-center gap-2">
            <Icon icon="tabler:lock" class="w-4 h-4 text-indigo-600" />
            <span>登录界面壁纸独立设置</span>
          </h3>
          <p class="text-xs text-slate-500 mt-0.5">管理后台认证界面的专属壁纸（登录页不允许外部直接修改，仅在此处统一配置）</p>
        </div>
      </div>

      <!-- Mode selection -->
      <div class="grid grid-cols-2 gap-3">
        <button
          type="button"
          @click="settingsStore.loginWallpaperType = 'follow'"
          class="p-3 rounded-xl border text-left transition-all cursor-pointer flex flex-col gap-0.5"
          :class="[
            settingsStore.loginWallpaperType === 'follow'
              ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400 font-semibold'
              : 'border-slate-200 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-800/50 text-slate-600 dark:text-slate-400'
          ]"
        >
          <span class="text-xs font-bold">跟随主站首页壁纸</span>
          <span class="text-[11px] text-slate-400">自动与主导航页面保持统一壁纸</span>
        </button>

        <button
          type="button"
          @click="settingsStore.loginWallpaperType = 'upload'"
          class="p-3 rounded-xl border text-left transition-all cursor-pointer flex flex-col gap-0.5"
          :class="[
            settingsStore.loginWallpaperType === 'upload'
              ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400 font-semibold'
              : 'border-slate-200 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-800/50 text-slate-600 dark:text-slate-400'
          ]"
        >
          <span class="text-xs font-bold">独立登录壁纸</span>
          <span class="text-[11px] text-slate-400">为登录界面指定单独的高清图片</span>
        </button>
      </div>

      <!-- Independent Login inputs -->
      <div v-if="settingsStore.loginWallpaperType === 'upload'" class="space-y-4 pt-1">
        <div class="flex flex-col sm:flex-row items-stretch sm:items-center gap-3">
          <div class="flex-1">
            <label class="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">登录页壁纸 URL</label>
            <input
              type="text"
              v-model="settingsStore.loginWallpaperUrl"
              placeholder="https://example.com/login_bg.jpg 或选择右侧上传"
              class="w-full px-3 py-2 text-xs bg-slate-50 dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 text-slate-800 dark:text-slate-200 focus:outline-none focus:border-indigo-500"
            />
          </div>
          <div class="sm:self-end">
            <label class="inline-flex items-center justify-center gap-1.5 px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold rounded-xl cursor-pointer transition-colors shadow-xs">
              <Icon icon="tabler:upload" class="w-4 h-4" />
              <span>{{ uploadingLogin ? '上传中...' : '上传登录专属壁纸' }}</span>
              <input type="file" accept="image/*" class="hidden" @change="handleLoginUpload" :disabled="uploadingLogin" />
            </label>
          </div>
        </div>

        <!-- Sliders: Login Blur & Mask -->
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-5 pt-3 border-t border-slate-100 dark:border-slate-800">
          <div>
            <div class="flex justify-between text-xs mb-1.5 font-medium text-slate-600 dark:text-slate-400">
              <span>登录页壁纸模糊度</span>
              <span class="font-mono text-indigo-600">{{ settingsStore.loginWallpaperBlur }}px</span>
            </div>
            <input
              type="range"
              min="0"
              max="20"
              step="1"
              v-model.number="settingsStore.loginWallpaperBlur"
              class="w-full accent-indigo-600 cursor-pointer"
            />
          </div>

          <div>
            <div class="flex justify-between text-xs mb-1.5 font-medium text-slate-600 dark:text-slate-400">
              <span>登录页遮罩暗度</span>
              <span class="font-mono text-indigo-600">{{ settingsStore.loginWallpaperMask }}%</span>
            </div>
            <input
              type="range"
              min="0"
              max="80"
              step="5"
              v-model.number="settingsStore.loginWallpaperMask"
              class="w-full accent-indigo-600 cursor-pointer"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- 3. Uploaded Wallpaper Gallery (User's Private Media Assets) -->
    <div class="p-5 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200/80 dark:border-slate-800/80 space-y-4">
      <div class="flex items-center justify-between border-b border-slate-100 dark:border-slate-800 pb-3">
        <div>
          <h3 class="text-sm font-bold text-slate-800 dark:text-slate-100 flex items-center gap-2">
            <Icon icon="tabler:folder" class="w-4 h-4 text-indigo-600" />
            <span>已上传壁纸图库 ({{ uploadedWallpapers.length }})</span>
          </h3>
          <p class="text-xs text-slate-500 mt-0.5">您上传的所有自定义壁纸素材，点击一键设为首页或登录页壁纸</p>
        </div>
        <button
          type="button"
          @click="fetchUploadedList"
          class="p-1.5 rounded-lg text-slate-400 hover:text-indigo-600 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
          title="刷新列表"
        >
          <Icon icon="tabler:refresh" class="w-4 h-4" />
        </button>
      </div>

      <div v-if="uploadedWallpapers.length > 0" class="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 gap-3.5 pt-1">
        <div
          v-for="wp in uploadedWallpapers"
          :key="wp.name"
          class="group relative rounded-xl overflow-hidden border aspect-video bg-slate-950 transition-all shadow-xs"
          :class="[
            settingsStore.wallpaperUrl === wp.url || settingsStore.loginWallpaperUrl === wp.url
              ? 'ring-2 ring-indigo-500 border-indigo-400'
              : 'border-slate-200 dark:border-slate-800 hover:border-slate-400'
          ]"
        >
          <img :src="wp.url" class="w-full h-full object-cover group-hover:scale-105 transition-transform" />

          <!-- Current status badges -->
          <div class="absolute top-1.5 left-1.5 flex flex-col gap-1 z-10">
            <span
              v-if="settingsStore.wallpaperUrl === wp.url"
              class="px-1.5 py-0.5 rounded-md bg-indigo-600/90 text-white text-[10px] font-bold shadow-xs backdrop-blur-xs"
            >
              首页壁纸
            </span>
            <span
              v-if="settingsStore.loginWallpaperUrl === wp.url"
              class="px-1.5 py-0.5 rounded-md bg-emerald-600/90 text-white text-[10px] font-bold shadow-xs backdrop-blur-xs"
            >
              登录壁纸
            </span>
          </div>

          <!-- Hover Action Overlay -->
          <div class="absolute inset-0 bg-black/65 opacity-0 group-hover:opacity-100 transition-opacity flex flex-col justify-between p-2">
            <div class="flex justify-end">
              <button
                type="button"
                @click.stop="deleteWallpaper(wp.name)"
                class="p-1 rounded-md bg-rose-600 hover:bg-rose-700 text-white text-xs cursor-pointer"
                title="从服务器删除此图片"
              >
                <Icon icon="tabler:trash" class="w-3.5 h-3.5" />
              </button>
            </div>

            <div class="flex items-center gap-1.5">
              <button
                type="button"
                @click.stop="selectAsHomeWallpaper(wp.url)"
                class="flex-1 py-1 rounded-md bg-white/20 hover:bg-indigo-600 text-white text-[10px] font-medium transition-colors"
                title="设为首页壁纸"
              >
                设为首页
              </button>
              <button
                type="button"
                @click.stop="selectAsLoginWallpaper(wp.url)"
                class="flex-1 py-1 rounded-md bg-white/20 hover:bg-emerald-600 text-white text-[10px] font-medium transition-colors"
                title="设为登录壁纸"
              >
                设为登录
              </button>
            </div>
          </div>
        </div>
      </div>

      <div v-else class="py-8 text-center text-slate-400 text-xs">
        <Icon icon="tabler:photo-off" class="w-8 h-8 mx-auto mb-2 opacity-50" />
        <p>暂无已上传壁纸，可点击上方「上传新壁纸」添加您的专属壁纸</p>
      </div>
    </div>

    <!-- Bottom Save Button Bar -->
    <div class="pt-2 flex justify-end">
      <button
        type="button"
        @click="handleSave"
        class="px-6 py-2.5 bg-indigo-600 hover:bg-indigo-700 active:scale-95 text-white text-xs font-semibold rounded-xl shadow-md shadow-indigo-600/20 transition-all cursor-pointer flex items-center gap-1.5"
      >
        <Icon icon="tabler:check" class="w-4 h-4" />
        <span>保存并应用所有壁纸配置</span>
      </button>
    </div>
  </div>
</template>
