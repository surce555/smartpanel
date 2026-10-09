<script setup lang="ts">
import { ref } from 'vue'
import { useSettingsStore } from '@/stores/settings'
import { useApi } from '@/composables/useApi'
import { useTheme } from '@/composables/useTheme'

const settingsStore = useSettingsStore()
const { applyFontSettings } = useTheme()
const api = useApi()

const uploading = ref<boolean>(false)
const uploadMsg = ref<string>('')

const fontPresets = [
  { name: '默认系统字体', value: 'system-ui, -apple-system, sans-serif' },
  { name: '优雅无衬线 (Inter)', value: 'Inter, system-ui, sans-serif' },
  { name: '极客等宽 (JetBrains Mono)', value: '"JetBrains Mono", monospace' },
  { name: '代码手写 (Fira Code)', value: '"Fira Code", monospace' },
  { name: '经典黑体 (思源/雅黑)', value: '"PingFang SC", "Microsoft YaHei", sans-serif' },
]

const fontSizes = [
  { name: '标准 (16px)', value: '16px' },
  { name: '紧凑 (14px)', value: '14px' },
  { name: '宽松 (18px)', value: '18px' },
  { name: '超大 (20px)', value: '20px' },
]

function selectFont(fontValue: string) {
  settingsStore.fontFamily = fontValue
  applyFontSettings(fontValue, settingsStore.fontSize)
}

function selectFontSize(sizeValue: string) {
  settingsStore.fontSize = sizeValue
  applyFontSettings(settingsStore.fontFamily, sizeValue)
}

async function handleFontUpload(e: Event) {
  const target = e.target as HTMLInputElement
  if (!target.files || target.files.length === 0) return

  const file = target.files[0]
  const formData = new FormData()
  formData.append('file', file)

  uploading.value = true
  uploadMsg.value = ''

  try {
    const resp = await api.post('/upload/font', formData)
    const fontName = resp.data.font_family
    const fontUrl = resp.data.url

    // Inject @font-face dynamically
    const style = document.createElement('style')
    style.textContent = `
      @font-face {
        font-family: '${fontName}';
        src: url('${fontUrl}');
      }
    `
    document.head.appendChild(style)

    settingsStore.fontFamily = fontName
    applyFontSettings(fontName, settingsStore.fontSize)
    uploadMsg.value = `自定义字体 "${fontName}" 上传并应用成功！`
  } catch (err: any) {
    uploadMsg.value = err.response?.data?.error || '上传字体失败'
  } finally {
    uploading.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <!-- Font Presets -->
    <div>
      <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">字体预设</label>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <button
          type="button"
          v-for="fp in fontPresets"
          :key="fp.name"
          @click="selectFont(fp.value)"
          class="p-3 text-left rounded-xl border text-sm transition-all"
          :class="[
            settingsStore.fontFamily === fp.value
              ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400 font-semibold'
              : 'border-slate-200 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-800/50 text-slate-700 dark:text-slate-300'
          ]"
        >
          {{ fp.name }}
        </button>
      </div>
    </div>

    <!-- Font Size -->
    <div>
      <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">基准字号</label>
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <button
          type="button"
          v-for="fs in fontSizes"
          :key="fs.name"
          @click="selectFontSize(fs.value)"
          class="p-2.5 text-center rounded-xl border text-xs transition-all"
          :class="[
            settingsStore.fontSize === fs.value
              ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400 font-semibold'
              : 'border-slate-200 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-800/50 text-slate-700 dark:text-slate-300'
          ]"
        >
          {{ fs.name }}
        </button>
      </div>
    </div>

    <!-- Custom Font Upload -->
    <div class="p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/40 border border-slate-200/60 dark:border-slate-800/60 flex items-center justify-between">
      <div>
        <h4 class="text-sm font-semibold text-slate-800 dark:text-slate-200">上传个性化字体文件</h4>
        <p class="text-xs text-slate-500">支持 TTF, WOFF2, WOFF 格式，不超过 15MB</p>
        <p v-if="uploadMsg" class="text-xs text-indigo-600 dark:text-indigo-400 mt-1 font-medium">{{ uploadMsg }}</p>
      </div>
      <label class="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-medium rounded-xl cursor-pointer transition-colors shadow-sm">
        <span>{{ uploading ? '上传中...' : '选择字体' }}</span>
        <input type="file" accept=".ttf,.woff,.woff2,.otf" class="hidden" @change="handleFontUpload" :disabled="uploading" />
      </label>
    </div>
  </div>
</template>
