<script setup lang="ts">
import { ref, watch } from 'vue'
import { Icon } from '@iconify/vue'
import { useApi } from '@/composables/useApi'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const api = useApi()
const content = ref('')
const lastUpdated = ref('')
const saving = ref(false)
const copied = ref(false)
let saveTimeout: any = null

async function loadMemo() {
  try {
    const res = await api.get('/memo')
    content.value = res.data.content || ''
    lastUpdated.value = res.data.updated_at || ''
  } catch (e) {
    console.warn('Failed to load memo:', e)
  }
}

async function saveMemo() {
  saving.value = true
  try {
    const res = await api.put('/memo', { content: content.value })
    if (res.data.updated_at) {
      lastUpdated.value = res.data.updated_at
    }
  } catch (e) {
    console.error('Failed to save memo:', e)
  } finally {
    saving.value = false
  }
}

function onContentChange() {
  if (saveTimeout) clearTimeout(saveTimeout)
  saving.value = true
  saveTimeout = setTimeout(() => {
    saveMemo()
  }, 600)
}

async function copyContent() {
  if (!content.value) return
  try {
    await navigator.clipboard.writeText(content.value)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 2000)
  } catch (e) {
    console.warn('Clipboard copy failed:', e)
  }
}

function clearContent() {
  if (confirm('确认清空便签内容吗？')) {
    content.value = ''
    saveMemo()
  }
}

watch(
  () => props.show,
  (val) => {
    if (val) {
      loadMemo()
    }
  }
)
</script>

<template>
  <div
    v-if="show"
    class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-md"
    @click.self="emit('close')"
  >
    <div
      class="w-full max-w-lg rounded-3xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-2xl p-6 space-y-4"
    >
      <!-- Header -->
      <div class="flex items-center justify-between border-b border-slate-100 dark:border-slate-800/80 pb-3">
        <div class="flex items-center gap-2.5">
          <div class="flex items-center justify-center w-8 h-8 rounded-xl bg-indigo-50 dark:bg-indigo-950/60 text-indigo-600 dark:text-indigo-400">
            <Icon icon="tabler:notes" class="w-5 h-5" />
          </div>
          <div>
            <h3 class="text-base font-bold text-slate-800 dark:text-slate-100">
              随手记 / 跨设备剪贴板
            </h3>
            <p class="text-[11px] text-slate-400">在手机与电脑之间秒级互传文字、链接与配置代码</p>
          </div>
        </div>
        <button
          @click="emit('close')"
          class="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 cursor-pointer"
        >
          ✕
        </button>
      </div>

      <!-- Editor Textarea -->
      <div class="relative">
        <textarea
          v-model="content"
          @input="onContentChange"
          rows="9"
          placeholder="在此粘贴或输入任何临时文本、下载磁力链接、代码片段，所有设备打开即见..."
          class="w-full p-3.5 bg-slate-50 dark:bg-slate-800/70 rounded-2xl border border-slate-200 dark:border-slate-700/80 focus:outline-none focus:border-indigo-500 font-mono text-xs leading-relaxed resize-none text-slate-800 dark:text-slate-100"
        ></textarea>

        <div class="absolute bottom-3 right-3 flex items-center gap-2">
          <span v-if="saving" class="text-[10px] text-indigo-500 flex items-center gap-1 font-mono">
            <Icon icon="tabler:loader-2" class="w-3 h-3 animate-spin" />
            正在同步...
          </span>
          <span v-else-if="lastUpdated" class="text-[10px] text-slate-400 font-mono">
            ✓ 已保存 {{ lastUpdated.slice(11) }}
          </span>
        </div>
      </div>

      <!-- Actions -->
      <div class="flex items-center justify-between pt-1">
        <div class="flex items-center gap-2">
          <button
            type="button"
            @click="copyContent"
            :disabled="!content"
            class="flex items-center gap-1 px-3 py-1.5 rounded-xl bg-indigo-50 dark:bg-indigo-950/50 text-indigo-600 dark:text-indigo-400 hover:bg-indigo-100 text-xs font-semibold transition-colors disabled:opacity-40 cursor-pointer"
          >
            <Icon :icon="copied ? 'tabler:check' : 'tabler:copy'" class="w-3.5 h-3.5" />
            <span>{{ copied ? '已复制！' : '复制全部' }}</span>
          </button>

          <button
            type="button"
            @click="clearContent"
            :disabled="!content"
            class="flex items-center gap-1 px-2.5 py-1.5 rounded-xl text-slate-400 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/40 text-xs transition-colors disabled:opacity-40 cursor-pointer"
            title="清空内容"
          >
            <Icon icon="tabler:trash" class="w-3.5 h-3.5" />
            <span>清空</span>
          </button>
        </div>

        <button
          type="button"
          @click="emit('close')"
          class="px-4 py-1.5 bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl text-xs font-semibold shadow-xs cursor-pointer"
        >
          关闭
        </button>
      </div>
    </div>
  </div>
</template>
