<script setup lang="ts">
import { ref } from 'vue'
import { Icon } from '@iconify/vue'
import { useBookmarksStore } from '@/stores/bookmarks'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'saved'): void
}>()

const bookmarksStore = useBookmarksStore()
const name = ref('')
const color = ref('#6366f1')

const presetColors = [
  '#6366f1', // Indigo
  '#3b82f6', // Blue
  '#06b6d4', // Cyan
  '#10b981', // Emerald
  '#84cc16', // Lime
  '#f59e0b', // Amber
  '#f97316', // Orange
  '#ef4444', // Red
  '#ec4899', // Pink
  '#8b5cf6', // Violet
]

async function handleSave() {
  if (!name.value.trim()) return
  await bookmarksStore.createTag(name.value.trim(), color.value)
  name.value = ''
  color.value = '#6366f1'
  emit('saved')
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
      class="w-full max-w-sm p-6 rounded-3xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-2xl space-y-4"
    >
      <div class="flex items-center justify-between border-b border-slate-100 dark:border-slate-800/80 pb-3">
        <div class="flex items-center gap-2">
          <div class="flex items-center justify-center w-8 h-8 rounded-xl bg-indigo-50 dark:bg-indigo-950/60 text-indigo-600 dark:text-indigo-400">
            <Icon icon="tabler:tag" class="w-5 h-5" />
          </div>
          <h3 class="text-base font-bold text-slate-800 dark:text-slate-100">
            新增分类标签
          </h3>
        </div>
        <button
          @click="emit('close')"
          class="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200"
        >
          ✕
        </button>
      </div>

      <form @submit.prevent="handleSave" class="space-y-4 text-xs">
        <div>
          <label class="block font-medium text-slate-700 dark:text-slate-300 mb-1">标签名称 *</label>
          <input
            v-model="name"
            required
            placeholder="例如：影音、下载、开发、家庭"
            class="w-full px-3.5 py-2 bg-slate-50 dark:bg-slate-800/80 rounded-xl border border-slate-200 dark:border-slate-700 focus:outline-none focus:border-indigo-500 font-medium"
          />
        </div>

        <div>
          <label class="block font-medium text-slate-700 dark:text-slate-300 mb-1.5">标签主题色</label>
          <div class="flex items-center gap-2.5 mb-2.5">
            <input type="color" v-model="color" class="w-9 h-8 rounded-lg cursor-pointer" />
            <span class="font-mono text-xs text-slate-500">{{ color }}</span>
          </div>

          <div class="flex items-center flex-wrap gap-2">
            <button
              v-for="c in presetColors"
              :key="c"
              type="button"
              @click="color = c"
              class="w-6 h-6 rounded-full transition-transform hover:scale-110"
              :class="color === c ? 'ring-2 ring-offset-2 ring-indigo-500 scale-110' : ''"
              :style="{ backgroundColor: c }"
            ></button>
          </div>
        </div>

        <div class="flex justify-end gap-2 pt-2 border-t border-slate-100 dark:border-slate-800/80">
          <button
            type="button"
            @click="emit('close')"
            class="px-4 py-2 text-xs text-slate-600 dark:text-slate-400 rounded-xl hover:bg-slate-100 dark:hover:bg-slate-800"
          >
            取消
          </button>
          <button
            type="submit"
            class="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl text-xs font-semibold shadow-md shadow-indigo-600/20"
          >
            确定创建
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
