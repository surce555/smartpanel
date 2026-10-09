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
const deletingTagId = ref<string | null>(null)

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
}

async function handleDeleteTag(id: string, tagName: string) {
  if (confirm(`确认删除标签「${tagName}」吗？关联此标签的书签不会被删除。`)) {
    deletingTagId.value = id
    try {
      await bookmarksStore.deleteTag(id)
      if (bookmarksStore.selectedTagId === id) {
        bookmarksStore.selectedTagId = null
      }
    } finally {
      deletingTagId.value = null
    }
  }
}
</script>

<template>
  <div
    v-if="show"
    class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-md"
    @click.self="emit('close')"
  >
    <div
      class="w-full max-w-md p-6 rounded-3xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-2xl space-y-4"
    >
      <!-- Header -->
      <div class="flex items-center justify-between border-b border-slate-100 dark:border-slate-800/80 pb-3">
        <div class="flex items-center gap-2">
          <div class="flex items-center justify-center w-8 h-8 rounded-xl bg-indigo-50 dark:bg-indigo-950/60 text-indigo-600 dark:text-indigo-400">
            <Icon icon="tabler:tags" class="w-5 h-5" />
          </div>
          <div>
            <h3 class="text-base font-bold text-slate-800 dark:text-slate-100">
              标签系统管理
            </h3>
            <p class="text-[11px] text-slate-400">管理分类标签，支持新建或删除不需要的标签</p>
          </div>
        </div>
        <button
          @click="emit('close')"
          class="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 cursor-pointer"
        >
          ✕
        </button>
      </div>

      <!-- 1. Existing Tags List (with delete button) -->
      <div class="space-y-2 pb-3 border-b border-slate-100 dark:border-slate-800">
        <div class="flex items-center justify-between text-xs font-semibold text-slate-600 dark:text-slate-300">
          <span>当前已有标签 ({{ bookmarksStore.tags.length }})</span>
          <span class="text-[11px] text-slate-400 font-normal">点击垃圾桶图标即可删除</span>
        </div>

        <div v-if="bookmarksStore.tags.length > 0" class="flex flex-wrap gap-2 max-h-44 overflow-y-auto pr-1 py-1">
          <div
            v-for="t in bookmarksStore.tags"
            :key="t.id"
            class="group flex items-center gap-1.5 pl-2.5 pr-1 py-1 rounded-xl text-xs font-medium border transition-all"
            :style="{
              backgroundColor: `${t.color}15`,
              borderColor: `${t.color}40`,
              color: t.color
            }"
          >
            <span class="w-2.5 h-2.5 rounded-full shrink-0" :style="{ backgroundColor: t.color }"></span>
            <span class="text-slate-800 dark:text-slate-200 max-w-[120px] truncate">{{ t.name }}</span>
            <button
              type="button"
              @click="handleDeleteTag(t.id, t.name)"
              :disabled="deletingTagId === t.id"
              class="ml-1 p-1 rounded-lg text-slate-400 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/40 transition-colors cursor-pointer"
              title="删除此标签"
            >
              <Icon icon="tabler:trash" class="w-3.5 h-3.5" />
            </button>
          </div>
        </div>

        <div v-else class="text-xs text-slate-400 py-3 text-center bg-slate-50 dark:bg-slate-800/40 rounded-xl">
          暂无任何标签，请在下方表单创建新标签
        </div>
      </div>

      <!-- 2. Create Tag Form -->
      <form @submit.prevent="handleSave" class="space-y-3.5 text-xs">
        <h4 class="font-semibold text-slate-700 dark:text-slate-300">新建分类标签</h4>

        <div>
          <label class="block font-medium text-slate-700 dark:text-slate-300 mb-1">标签名称 *</label>
          <input
            v-model="name"
            required
            placeholder="例如：影音、下载、开发、家庭、监控"
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
            class="px-4 py-2 text-xs text-slate-600 dark:text-slate-400 rounded-xl hover:bg-slate-100 dark:hover:bg-slate-800 cursor-pointer"
          >
            完成
          </button>
          <button
            type="submit"
            class="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl text-xs font-semibold shadow-md shadow-indigo-600/20 cursor-pointer"
          >
            添加新标签
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
