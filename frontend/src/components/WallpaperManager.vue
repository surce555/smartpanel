<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useSettingsStore } from '@/stores/settings'
import { useApi } from '@/composables/useApi'

const settingsStore = useSettingsStore()
const api = useApi()

const uploadedWallpapers = ref<any[]>([])
const uploading = ref<boolean>(false)
const uploadMsg = ref<string>('')

async function fetchUploadedList() {
  try {
    const resp = await api.get('/upload/list?category=wallpapers')
    uploadedWallpapers.value = resp.data
  } catch (e) {
    console.error(e)
  }
}

async function handleFileUpload(e: Event) {
  const target = e.target as HTMLInputElement
  if (!target.files || target.files.length === 0) return

  const file = target.files[0]
  const formData = new FormData()
  formData.append('file', file)

  uploading.value = true
  uploadMsg.value = ''
  try {
    const resp = await api.post('/upload/wallpaper', formData)
    settingsStore.wallpaperUrl = resp.data.url
    settingsStore.wallpaperType = 'upload'
    uploadMsg.value = '壁纸上传成功！'
    await fetchUploadedList()
  } catch (err: any) {
    uploadMsg.value = err.response?.data?.error || '上传失败'
  } finally {
    uploading.value = false
  }
}

async function deleteWallpaper(filename: string) {
  try {
    await api.delete(`/upload/${filename}?category=wallpapers`)
    await fetchUploadedList()
  } catch (e) {
    console.error(e)
  }
}

function selectWallpaper(url: string) {
  settingsStore.wallpaperUrl = url
  settingsStore.wallpaperType = 'upload'
}

onMounted(() => {
  fetchUploadedList()
})
</script>

<template>
  <div class="space-y-6">
    <!-- Wallpaper Mode Selection -->
    <div>
      <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">背景模式</label>
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <button
          type="button"
          v-for="mode in [
            { id: 'gradient', name: '精致渐变' },
            { id: 'upload', name: '本地壁纸' },
            { id: 'unsplash', name: '网络随机 (Unsplash)' },
            { id: 'none', name: '纯色背景' }
          ]"
          :key="mode.id"
          @click="settingsStore.wallpaperType = mode.id"
          class="p-3 text-sm rounded-xl border text-center transition-all"
          :class="[
            settingsStore.wallpaperType === mode.id
              ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400 font-semibold shadow-xs'
              : 'border-slate-200 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-800/50 text-slate-700 dark:text-slate-300'
          ]"
        >
          {{ mode.name }}
        </button>
      </div>
    </div>

    <!-- Upload Section -->
    <div v-if="settingsStore.wallpaperType === 'upload'" class="p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/40 border border-slate-200/60 dark:border-slate-800/60 space-y-4">
      <div class="flex items-center justify-between">
        <div>
          <h4 class="text-sm font-semibold text-slate-800 dark:text-slate-200">上传新壁纸</h4>
          <p class="text-xs text-slate-500">支持 JPG, PNG, WebP 格式，单个文件不超过 10MB</p>
        </div>
        <label class="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-medium rounded-xl cursor-pointer transition-colors shadow-sm">
          <span>{{ uploading ? '上传中...' : '选择文件' }}</span>
          <input type="file" accept="image/*" class="hidden" @change="handleFileUpload" :disabled="uploading" />
        </label>
      </div>

      <div v-if="uploadMsg" class="text-xs text-indigo-600 dark:text-indigo-400 font-medium">
        {{ uploadMsg }}
      </div>

      <!-- Uploaded List / Gallery -->
      <div v-if="uploadedWallpapers.length > 0" class="pt-2">
        <label class="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-2">已上传壁纸库 (点击应用)</label>
        <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
          <div
            v-for="wp in uploadedWallpapers"
            :key="wp.name"
            class="relative group rounded-xl overflow-hidden border aspect-video cursor-pointer"
            :class="[settingsStore.wallpaperUrl === wp.url ? 'ring-2 ring-indigo-500' : 'border-slate-200 dark:border-slate-800']"
            @click="selectWallpaper(wp.url)"
          >
            <img :src="wp.url" class="w-full h-full object-cover" />
            <button
              type="button"
              @click.stop="deleteWallpaper(wp.name)"
              class="absolute top-1 right-1 p-1 bg-red-600/80 hover:bg-red-700 text-white rounded-md text-xs opacity-0 group-hover:opacity-100 transition-opacity"
              title="删除"
            >
              ✕
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Sliders: Blur & Mask -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-6">
      <div>
        <div class="flex justify-between text-sm mb-1.5">
          <span class="text-slate-700 dark:text-slate-300 font-medium">壁纸模糊度</span>
          <span class="text-slate-500 font-mono text-xs">{{ settingsStore.wallpaperBlur }}px</span>
        </div>
        <input
          type="range"
          min="0"
          max="20"
          step="1"
          v-model.number="settingsStore.wallpaperBlur"
          class="w-full accent-indigo-600"
        />
      </div>

      <div>
        <div class="flex justify-between text-sm mb-1.5">
          <span class="text-slate-700 dark:text-slate-300 font-medium">遮罩暗度</span>
          <span class="text-slate-500 font-mono text-xs">{{ settingsStore.wallpaperMask }}%</span>
        </div>
        <input
          type="range"
          min="0"
          max="80"
          step="5"
          v-model.number="settingsStore.wallpaperMask"
          class="w-full accent-indigo-600"
        />
      </div>
    </div>
  </div>
</template>
