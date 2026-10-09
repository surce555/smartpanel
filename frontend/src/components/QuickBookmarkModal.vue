<script setup lang="ts">
import { ref, watch } from 'vue'
import { Icon } from '@iconify/vue'
import { useBookmarksStore, Bookmark, Tag } from '@/stores/bookmarks'

const props = defineProps<{
  show: boolean
  bookmark?: Partial<Bookmark> | null
  defaultGroupId?: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'saved', bookmark: Bookmark): void
}>()

const bookmarksStore = useBookmarksStore()

const form = ref({
  id: '',
  name: '',
  icon: 'tabler:link',
  url_internal: '',
  url_public_template: '',
  url_fallback: '',
  group_id: '',
  description: '',
  open_in_new_tab: true,
  is_private: false,
  tag_ids: [] as string[],
})

// Inline create group
const showInlineGroup = ref(false)
const inlineGroupName = ref('')

// Inline create tag
const showInlineTag = ref(false)
const inlineTagName = ref('')
const inlineTagColor = ref('#6366f1')

const iconPresets = [
  { icon: 'tabler:server', label: 'NAS' },
  { icon: 'tabler:router', label: '路由器' },
  { icon: 'tabler:brand-docker', label: 'Docker' },
  { icon: 'tabler:movie', label: '影音' },
  { icon: 'tabler:download', label: '下载' },
  { icon: 'tabler:cloud', label: '网盘' },
  { icon: 'tabler:shield-check', label: '防护' },
  { icon: 'tabler:key', label: '密码库' },
  { icon: 'tabler:terminal-2', label: '终端' },
  { icon: 'tabler:layout-dashboard', label: '面板' },
]

watch(
  () => props.show,
  (val) => {
    if (val) {
      if (props.bookmark) {
        form.value = {
          id: props.bookmark.id || '',
          name: props.bookmark.name || '',
          icon: props.bookmark.icon || 'tabler:link',
          url_internal: props.bookmark.url_internal || '',
          url_public_template: props.bookmark.url_public_template || '',
          url_fallback: props.bookmark.url_fallback || '',
          group_id: props.bookmark.group_id || props.defaultGroupId || bookmarksStore.groups[0]?.id || '',
          description: props.bookmark.description || '',
          open_in_new_tab: props.bookmark.open_in_new_tab ?? true,
          is_private: props.bookmark.is_private ?? false,
          tag_ids: props.bookmark.tags ? props.bookmark.tags.map((t: Tag) => t.id) : (props.bookmark.tag_ids || []),
        }
      } else {
        form.value = {
          id: '',
          name: '',
          icon: 'tabler:link',
          url_internal: '',
          url_public_template: '',
          url_fallback: '',
          group_id: props.defaultGroupId || bookmarksStore.groups[0]?.id || '',
          description: '',
          open_in_new_tab: true,
          is_private: false,
          tag_ids: [],
        }
      }
      showInlineGroup.value = false
      showInlineTag.value = false
    }
  },
  { immediate: true }
)

async function handleSaveInlineGroup() {
  if (!inlineGroupName.value.trim()) return
  const g = await bookmarksStore.createGroup(inlineGroupName.value.trim(), 'tabler:folder')
  if (g && g.id) {
    form.value.group_id = g.id
  }
  inlineGroupName.value = ''
  showInlineGroup.value = false
}

async function handleSaveInlineTag() {
  if (!inlineTagName.value.trim()) return
  const t = await bookmarksStore.createTag(inlineTagName.value.trim(), inlineTagColor.value)
  if (t && t.id) {
    if (!form.value.tag_ids.includes(t.id)) {
      form.value.tag_ids.push(t.id)
    }
  }
  inlineTagName.value = ''
  showInlineTag.value = false
}

async function handleSave() {
  if (!form.value.name.trim()) return
  let res: Bookmark
  if (form.value.id) {
    res = await bookmarksStore.updateBookmark(form.value.id, form.value)
  } else {
    res = await bookmarksStore.createBookmark(form.value)
  }
  emit('saved', res)
  emit('close')
}

function fillTemplate(tmpl: string) {
  form.value.url_public_template = tmpl
}
</script>

<template>
  <div
    v-if="show"
    class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-md"
    @click.self="emit('close')"
  >
    <div
      class="w-full max-w-lg p-6 rounded-3xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-2xl space-y-4 max-h-[90vh] overflow-y-auto"
    >
      <div class="flex items-center justify-between border-b border-slate-100 dark:border-slate-800/80 pb-3">
        <div class="flex items-center gap-2">
          <div class="flex items-center justify-center w-8 h-8 rounded-xl bg-indigo-50 dark:bg-indigo-950/60 text-indigo-600 dark:text-indigo-400">
            <Icon icon="tabler:bookmark-plus" class="w-5 h-5" />
          </div>
          <div>
            <h3 class="text-base font-bold text-slate-800 dark:text-slate-100">
              {{ form.id ? '编辑书签卡片' : '快捷添加书签' }}
            </h3>
            <p class="text-[11px] text-slate-400">在首页直接管理您的服务入口</p>
          </div>
        </div>
        <button
          @click="emit('close')"
          class="p-1.5 rounded-lg text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800"
        >
          ✕
        </button>
      </div>

      <form @submit.prevent="handleSave" class="space-y-3.5 text-xs">
        <!-- Bookmark Name -->
        <div>
          <label class="block font-medium text-slate-700 dark:text-slate-300 mb-1">书签名称 *</label>
          <input
            v-model="form.name"
            required
            placeholder="例如：群晖 DSM、Jellyfin 影视、路由器"
            class="w-full px-3.5 py-2.5 bg-slate-50 dark:bg-slate-800/80 rounded-xl border border-slate-200 dark:border-slate-700 focus:outline-none focus:border-indigo-500 font-medium"
          />
        </div>

        <!-- Icon Picker & Presets -->
        <div>
          <label class="block font-medium text-slate-700 dark:text-slate-300 mb-1">
            图标 (Iconify 名称如 tabler:server，或图片/SVG 地址)
          </label>
          <div class="flex items-center gap-2">
            <input
              v-model="form.icon"
              placeholder="tabler:link"
              class="flex-1 px-3.5 py-2 bg-slate-50 dark:bg-slate-800/80 rounded-xl border border-slate-200 dark:border-slate-700 focus:outline-none focus:border-indigo-500"
            />
            <div class="flex items-center justify-center w-9 h-9 rounded-xl bg-indigo-50 dark:bg-indigo-950/60 text-indigo-600 dark:text-indigo-400 shrink-0">
              <Icon :icon="form.icon || 'tabler:link'" class="w-5 h-5" />
            </div>
          </div>
          <!-- Quick Presets -->
          <div class="flex items-center flex-wrap gap-1.5 mt-2">
            <span class="text-[10px] text-slate-400">快速推荐：</span>
            <button
              v-for="p in iconPresets"
              :key="p.icon"
              type="button"
              @click="form.icon = p.icon"
              class="px-2 py-0.5 rounded-md bg-slate-100 dark:bg-slate-800 hover:bg-indigo-50 dark:hover:bg-indigo-950 text-slate-600 dark:text-slate-300 hover:text-indigo-600 text-[11px] transition-colors"
            >
              {{ p.label }}
            </button>
          </div>
        </div>

        <!-- Public Dynamic Template URL -->
        <div>
          <div class="flex items-center justify-between mb-1">
            <label class="font-medium text-slate-700 dark:text-slate-300">
              外网动态模板地址 (自动替换 IPv6 / 域名)
            </label>
            <div class="flex items-center gap-1.5">
              <button
                type="button"
                @click="fillTemplate('https://[{ipv6}]:5666')"
                class="text-[10px] text-indigo-600 hover:underline"
              >
                + 填入 {ipv6}
              </button>
              <button
                type="button"
                @click="fillTemplate('https://{v6domain}:5666')"
                class="text-[10px] text-indigo-600 hover:underline"
              >
                + 填入 {v6domain}
              </button>
            </div>
          </div>
          <input
            v-model="form.url_public_template"
            placeholder="https://[{ipv6}]:5666 或 https://{v6domain}:5666"
            class="w-full px-3.5 py-2 bg-slate-50 dark:bg-slate-800/80 rounded-xl border border-slate-200 dark:border-slate-700 focus:outline-none focus:border-indigo-500 font-mono text-xs"
          />
        </div>

        <!-- Internal LAN URL -->
        <div>
          <label class="block font-medium text-slate-700 dark:text-slate-300 mb-1">内网直连地址 (局域网环境)</label>
          <input
            v-model="form.url_internal"
            placeholder="http://192.168.1.100:5666"
            class="w-full px-3.5 py-2 bg-slate-50 dark:bg-slate-800/80 rounded-xl border border-slate-200 dark:border-slate-700 focus:outline-none focus:border-indigo-500 font-mono text-xs"
          />
        </div>

        <!-- Fallback URL -->
        <div>
          <label class="block font-medium text-slate-700 dark:text-slate-300 mb-1">Cloudflare 代理回退地址 (IPv4 无法直连时)</label>
          <input
            v-model="form.url_fallback"
            placeholder="https://{domain}"
            class="w-full px-3.5 py-2 bg-slate-50 dark:bg-slate-800/80 rounded-xl border border-slate-200 dark:border-slate-700 focus:outline-none focus:border-indigo-500 font-mono text-xs"
          />
        </div>

        <!-- Group Selector + Inline Add -->
        <div>
          <div class="flex items-center justify-between mb-1">
            <label class="font-medium text-slate-700 dark:text-slate-300">所属分组</label>
            <button
              type="button"
              @click="showInlineGroup = !showInlineGroup"
              class="text-[11px] text-indigo-600 dark:text-indigo-400 hover:underline font-medium"
            >
              {{ showInlineGroup ? '取消新增' : '+ 新建分组' }}
            </button>
          </div>

          <div v-if="showInlineGroup" class="flex items-center gap-2 mb-2 p-2 rounded-xl bg-indigo-50/50 dark:bg-indigo-950/40 border border-indigo-100 dark:border-indigo-900/40">
            <input
              v-model="inlineGroupName"
              placeholder="新分组名称（如：媒体服务）"
              class="flex-1 px-3 py-1.5 text-xs bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 focus:outline-none"
            />
            <button
              type="button"
              @click="handleSaveInlineGroup"
              class="px-3 py-1.5 bg-indigo-600 text-white rounded-lg text-xs font-semibold hover:bg-indigo-700"
            >
              创建
            </button>
          </div>

          <select
            v-model="form.group_id"
            class="w-full px-3.5 py-2 bg-slate-50 dark:bg-slate-800/80 rounded-xl border border-slate-200 dark:border-slate-700 focus:outline-none focus:border-indigo-500 font-medium"
          >
            <option v-for="g in bookmarksStore.groups" :key="g.id" :value="g.id">
              {{ g.name }}
            </option>
          </select>
        </div>

        <!-- Tags Row + Inline Add -->
        <div>
          <div class="flex items-center justify-between mb-1">
            <label class="font-medium text-slate-700 dark:text-slate-300">关联标签</label>
            <button
              type="button"
              @click="showInlineTag = !showInlineTag"
              class="text-[11px] text-indigo-600 dark:text-indigo-400 hover:underline font-medium"
            >
              {{ showInlineTag ? '取消新增' : '+ 新建标签' }}
            </button>
          </div>

          <div v-if="showInlineTag" class="flex items-center gap-2 mb-2 p-2 rounded-xl bg-indigo-50/50 dark:bg-indigo-950/40 border border-indigo-100 dark:border-indigo-900/40">
            <input
              v-model="inlineTagName"
              placeholder="新标签名称（如：影音、下载）"
              class="flex-1 px-3 py-1.5 text-xs bg-white dark:bg-slate-800 rounded-lg border border-slate-200 dark:border-slate-700 focus:outline-none"
            />
            <input type="color" v-model="inlineTagColor" class="w-8 h-7 rounded cursor-pointer" />
            <button
              type="button"
              @click="handleSaveInlineTag"
              class="px-3 py-1.5 bg-indigo-600 text-white rounded-lg text-xs font-semibold hover:bg-indigo-700"
            >
              创建
            </button>
          </div>

          <div class="flex flex-wrap gap-2 pt-1">
            <label
              v-for="t in bookmarksStore.tags"
              :key="t.id"
              class="flex items-center gap-1.5 px-2.5 py-1 rounded-lg border text-xs cursor-pointer select-none transition-all"
              :class="[
                form.tag_ids.includes(t.id)
                  ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 font-semibold shadow-xs'
                  : 'border-slate-200 dark:border-slate-800 text-slate-600 dark:text-slate-400 hover:border-slate-300'
              ]"
            >
              <input type="checkbox" :value="t.id" v-model="form.tag_ids" class="hidden" />
              <span class="w-2 h-2 rounded-full" :style="{ backgroundColor: t.color }"></span>
              <span>{{ t.name }}</span>
            </label>
            <span v-if="bookmarksStore.tags.length === 0" class="text-slate-400 text-[11px]">
              暂无标签，可点击上方「+ 新建标签」创建
            </span>
          </div>
        </div>

        <!-- Description -->
        <div>
          <label class="block font-medium text-slate-700 dark:text-slate-300 mb-1">描述信息 (可选)</label>
          <textarea
            v-model="form.description"
            rows="2"
            placeholder="简要说明服务功能与用途"
            class="w-full px-3.5 py-2 bg-slate-50 dark:bg-slate-800/80 rounded-xl border border-slate-200 dark:border-slate-700 focus:outline-none focus:border-indigo-500"
          ></textarea>
        </div>

        <!-- Checkboxes -->
        <div class="flex items-center gap-4 pt-1">
          <label class="flex items-center gap-2 cursor-pointer select-none">
            <input type="checkbox" v-model="form.open_in_new_tab" class="rounded accent-indigo-600 w-4 h-4" />
            <span class="text-slate-700 dark:text-slate-300">新标签页打开</span>
          </label>
          <label class="flex items-center gap-2 cursor-pointer select-none">
            <input type="checkbox" v-model="form.is_private" class="rounded accent-indigo-600 w-4 h-4" />
            <span class="text-slate-700 dark:text-slate-300">私有书签 (仅登录可见)</span>
          </label>
        </div>

        <div class="flex justify-end gap-2.5 pt-3 border-t border-slate-100 dark:border-slate-800/80">
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
            保存书签
          </button>
        </div>
      </form>
    </div>
  </div>
</template>
