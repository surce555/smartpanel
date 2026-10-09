<script setup lang="ts">
import { ref } from 'vue'
import { Icon } from '@iconify/vue'
import { useSettingsStore } from '@/stores/settings'
import { useApi } from '@/composables/useApi'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'changed', url: string): void
}>()

const settingsStore = useSettingsStore()
const api = useApi()

const currentUrl = ref<string>(settingsStore.loginWallpaperUrl || '/wallpapers/login_anime_girl.png')
const currentType = ref<string>(settingsStore.loginWallpaperType || 'upload')
const currentBlur = ref<number>(settingsStore.loginWallpaperBlur || 0)
const currentMask = ref<number>(settingsStore.loginWallpaperMask || 15)
const customUrlInput = ref<string>('')
const uploading = ref<boolean>(false)
const msg = ref<string>('')

const presets = [
  {
    name: '动漫山色少女 (预制默认)',
    url: '/wallpapers/login_anime_girl.png',
    thumb: '/wallpapers/login_anime_girl.png'
  },
  {
    name: '咖啡厅微笑少女 (全透明预制)',
    url: '/wallpapers/home_cafe_girl.png',
    thumb: '/wallpapers/home_cafe_girl.png'
  },
  {
    name: '深邃星空银河',
    url: 'https://images.unsplash.com/photo-1506703719100-a0f3a48c0f86?auto=format&fit=crop&w=1920&q=80',
    thumb: 'https://images.unsplash.com/photo-1506703719100-a0f3a48c0f86?auto=format&fit=crop&w=300&q=80'
  },
  {
    name: '雪山晨曦湖畔',
    url: 'https://images.unsplash.com/photo-1464822759023-fed622ff2c3b?auto=format&fit=crop&w=1920&q=80',
    thumb: 'https://images.unsplash.com/photo-1464822759023-fed622ff2c3b?auto=format&fit=crop&w=300&q=80'
  }
]

function selectPreset(url: string) {
  currentUrl.value = url
  currentType.value = 'upload'
  emit('changed', url)
}

function selectFollowHome() {
  currentType.value = 'follow'
  const homeUrl = settingsStore.wallpaperUrl || '/wallpapers/home_cafe_girl.png'
  currentUrl.value = homeUrl
  emit('changed', homeUrl)
}

function handleApplyCustomUrl() {
  if (customUrlInput.value.trim()) {
    currentUrl.value = customUrlInput.value.trim()
    currentType.value = 'upload'
    emit('changed', currentUrl.value)
    customUrlInput.value = ''
  }
}

async function handleFileUpload(e: Event) {
  const target = e.target as HTMLInputElement
  if (!target.files || target.files.length === 0) return

  const file = target.files[0]
  uploading.value = true
  msg.value = '正在上传壁纸...'

  try {
    const formData = new FormData()
    formData.append('file', file)
    const res = await api.post('/upload', formData)
    const uploadedUrl = res.data.url
    currentUrl.value = uploadedUrl
    currentType.value = 'upload'
    emit('changed', uploadedUrl)
    msg.value = '上传成功！'
  } catch {
    // If not logged in yet or upload fails, read as local base64 preview
    const reader = new FileReader()
    reader.onload = (re) => {
      const b64 = re.target?.result as string
      currentUrl.value = b64
      currentType.value = 'upload'
      localStorage.setItem('smartpanel_login_wallpaper_local', b64)
      emit('changed', b64)
      msg.value = '本地壁纸已应用！'
    }
    reader.readAsDataURL(file)
  } finally {
    uploading.value = false
  }
}

async function handleSave() {
  try {
    await settingsStore.saveSettings({
      login_wallpaper_type: currentType.value,
      login_wallpaper_url: currentUrl.value,
      login_wallpaper_blur: String(currentBlur.value),
      login_wallpaper_mask: String(currentMask.value),
    })
  } catch {
    // Guest fallback
    localStorage.setItem('smartpanel_login_wallpaper_url', currentUrl.value)
    localStorage.setItem('smartpanel_login_wallpaper_type', currentType.value)
  }
  emit('close')
}
</script>

<template>
  <div
    v-if="show"
    class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-md"
    @click.self="emit('close')"
  >
    <div
      class="w-full max-w-md p-6 rounded-3xl bg-slate-900/95 border border-white/20 shadow-2xl space-y-4 text-white max-h-[90vh] overflow-y-auto animate-in fade-in zoom-in-95 duration-150"
    >
      <div class="flex items-center justify-between border-b border-white/10 pb-3">
        <div class="flex items-center gap-2">
          <div class="flex items-center justify-center w-8 h-8 rounded-xl bg-white/10 text-white">
            <Icon icon="tabler:photo" class="w-5 h-5" />
          </div>
          <div>
            <h3 class="text-base font-bold text-white">登录界面壁纸设置</h3>
            <p class="text-[11px] text-white/60">自由更换登录界面的背景视觉效果</p>
          </div>
        </div>
        <button
          @click="emit('close')"
          class="p-1 rounded-lg text-white/60 hover:text-white hover:bg-white/10"
        >
          ✕
        </button>
      </div>

      <!-- Presets Grid -->
      <div class="space-y-2">
        <label class="block text-xs font-semibold text-white/90">精选预设壁纸</label>
        <div class="grid grid-cols-2 gap-2.5">
          <div
            v-for="p in presets"
            :key="p.name"
            @click="selectPreset(p.url)"
            class="group relative h-20 rounded-xl overflow-hidden border-2 cursor-pointer transition-all"
            :class="[
              currentUrl === p.url && currentType !== 'follow'
                ? 'border-indigo-500 ring-2 ring-indigo-500/50'
                : 'border-white/10 hover:border-white/40'
            ]"
          >
            <img :src="p.thumb" class="w-full h-full object-cover group-hover:scale-105 transition-transform" />
            <div class="absolute inset-0 bg-gradient-to-t from-black/80 via-black/20 to-transparent flex items-end p-1.5">
              <span class="text-[11px] font-medium text-white truncate">{{ p.name }}</span>
            </div>
            <div
              v-if="currentUrl === p.url && currentType !== 'follow'"
              class="absolute top-1 right-1 w-4 h-4 rounded-full bg-indigo-600 flex items-center justify-center"
            >
              <Icon icon="tabler:check" class="w-3 h-3 text-white" />
            </div>
          </div>
        </div>

        <button
          type="button"
          @click="selectFollowHome"
          class="w-full mt-1 px-3 py-2 rounded-xl border border-white/20 hover:bg-white/10 text-xs flex items-center justify-between text-white/90 transition-colors"
          :class="{ 'border-indigo-500 bg-indigo-600/20 text-white font-semibold': currentType === 'follow' }"
        >
          <div class="flex items-center gap-2">
            <Icon icon="tabler:link" class="w-4 h-4 text-white/70" />
            <span>跟随主站首页壁纸</span>
          </div>
          <Icon v-if="currentType === 'follow'" icon="tabler:check" class="w-4 h-4 text-indigo-400" />
        </button>
      </div>

      <!-- Custom Upload or URL -->
      <div class="space-y-3 pt-2 border-t border-white/10 text-xs">
        <div>
          <label class="block font-semibold text-white/90 mb-1.5">上传自定义壁纸</label>
          <label class="flex items-center justify-center gap-2 w-full py-2.5 rounded-xl border border-dashed border-white/30 hover:border-white/60 bg-white/5 hover:bg-white/10 cursor-pointer transition-all">
            <Icon icon="tabler:upload" class="w-4 h-4 text-white/80" />
            <span class="text-white/80 font-medium">{{ uploading ? '正在上传...' : '选择本地图片上传' }}</span>
            <input type="file" accept="image/*" class="hidden" @change="handleFileUpload" :disabled="uploading" />
          </label>
        </div>

        <div>
          <label class="block font-semibold text-white/90 mb-1.5">或者输入网络图片 URL</label>
          <div class="flex gap-2">
            <input
              v-model="customUrlInput"
              placeholder="https://example.com/wallpaper.jpg"
              class="flex-1 px-3 py-2 rounded-xl bg-white/10 border border-white/20 text-white placeholder-white/40 focus:outline-none focus:border-white/50 text-xs"
            />
            <button
              type="button"
              @click="handleApplyCustomUrl"
              class="px-3 py-2 rounded-xl bg-white/20 hover:bg-white/30 text-white font-medium text-xs whitespace-nowrap"
            >
              应用
            </button>
          </div>
        </div>

        <!-- Blur & Mask Sliders -->
        <div class="grid grid-cols-2 gap-3 pt-1">
          <div>
            <div class="flex items-center justify-between text-[11px] text-white/70 mb-1">
              <span>背景模糊度</span>
              <span>{{ currentBlur }}px</span>
            </div>
            <input
              type="range"
              min="0"
              max="20"
              v-model.number="currentBlur"
              class="w-full accent-indigo-500"
            />
          </div>
          <div>
            <div class="flex items-center justify-between text-[11px] text-white/70 mb-1">
              <span>暗色遮罩</span>
              <span>{{ currentMask }}%</span>
            </div>
            <input
              type="range"
              min="0"
              max="80"
              v-model.number="currentMask"
              class="w-full accent-indigo-500"
            />
          </div>
        </div>

        <div v-if="msg" class="text-xs text-indigo-400 font-medium">
          {{ msg }}
        </div>
      </div>

      <!-- Action Buttons -->
      <div class="flex justify-end gap-2 pt-2 border-t border-white/10">
        <button
          type="button"
          @click="emit('close')"
          class="px-4 py-2 rounded-xl bg-white/10 hover:bg-white/20 text-white text-xs"
        >
          取消
        </button>
        <button
          type="button"
          @click="handleSave"
          class="px-5 py-2 rounded-xl bg-indigo-600 hover:bg-indigo-500 text-white text-xs font-semibold shadow-lg shadow-indigo-600/30"
        >
          保存生效
        </button>
      </div>
    </div>
  </div>
</template>
