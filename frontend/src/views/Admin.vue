<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { Icon } from '@iconify/vue'
import { useBookmarksStore, Bookmark, Group, Tag } from '@/stores/bookmarks'
import { useSettingsStore } from '@/stores/settings'
import { useAuthStore } from '@/stores/auth'
import { useApi } from '@/composables/useApi'
import CardStyleSettings from '@/components/CardStyleSettings.vue'
import WallpaperManager from '@/components/WallpaperManager.vue'
import FontSettings from '@/components/FontSettings.vue'
import CustomCode from '@/components/CustomCode.vue'
import NetworkSettings from '@/components/NetworkSettings.vue'
import SystemStatus from '@/components/SystemStatus.vue'
import DockerStatus from '@/components/DockerStatus.vue'

const router = useRouter()
const bookmarksStore = useBookmarksStore()
const settingsStore = useSettingsStore()
const authStore = useAuthStore()
const api = useApi()

const activeTab = ref<string>('bookmarks')

const tabs = [
  { id: 'bookmarks', name: '书签与分组', icon: 'tabler:bookmarks' },
  { id: 'tags', name: '标签系统', icon: 'tabler:tags' },
  { id: 'appearance', name: '外观个性化', icon: 'tabler:palette' },
  { id: 'network', name: '网络与 DDNS', icon: 'tabler:network' },
  { id: 'system', name: '系统与容器', icon: 'tabler:cpu' },
  { id: 'backup', name: '备份与还原', icon: 'tabler:database-export' },
  { id: 'account', name: '安全与账户', icon: 'tabler:lock' },
]

// Modal states: Bookmark
const showBookmarkModal = ref<boolean>(false)
const editingBookmark = ref<Partial<Bookmark> | null>(null)
const bookmarkForm = ref({
  id: '',
  name: '',
  icon: '',
  url_internal: '',
  url_public_template: '',
  url_fallback: '',
  group_id: '',
  description: '',
  open_in_new_tab: true,
  is_private: false,
  tag_ids: [] as string[],
})

// Modal states: Group
const showGroupModal = ref<boolean>(false)
const groupForm = ref({
  id: '',
  name: '',
  icon: '',
})

// Modal states: Tag
const showTagModal = ref<boolean>(false)
const tagForm = ref({
  name: '',
  color: '#6366f1',
})

// Account & Security
const accountName = ref('')
const accountMsg = ref('')
const securityMsg = ref('')

// Password change
const oldPwd = ref('')
const newPwd = ref('')
const pwdMsg = ref('')

// Backup & Import
const backupMsg = ref('')
const importing = ref(false)

onMounted(async () => {
  await bookmarksStore.fetchAll()
  await settingsStore.fetchSettings()
  if (!authStore.user) {
    await authStore.fetchMe()
  }
  if (authStore.user?.email) {
    accountName.value = authStore.user.email
  }
})

// Bookmark actions
function openCreateBookmark() {
  editingBookmark.value = null
  bookmarkForm.value = {
    id: '',
    name: '',
    icon: 'tabler:link',
    url_internal: '',
    url_public_template: '',
    url_fallback: '',
    group_id: bookmarksStore.groups[0]?.id || '',
    description: '',
    open_in_new_tab: true,
    is_private: false,
    tag_ids: [],
  }
  showBookmarkModal.value = true
}

function openEditBookmark(b: Bookmark) {
  editingBookmark.value = b
  bookmarkForm.value = {
    id: b.id,
    name: b.name,
    icon: b.icon || '',
    url_internal: b.url_internal || '',
    url_public_template: b.url_public_template || '',
    url_fallback: b.url_fallback || '',
    group_id: b.group_id || '',
    description: b.description || '',
    open_in_new_tab: b.open_in_new_tab,
    is_private: b.is_private,
    tag_ids: b.tags ? b.tags.map(t => t.id) : [],
  }
  showBookmarkModal.value = true
}

async function saveBookmark() {
  if (!bookmarkForm.value.name) return
  if (editingBookmark.value && editingBookmark.value.id) {
    await bookmarksStore.updateBookmark(editingBookmark.value.id, bookmarkForm.value)
  } else {
    await bookmarksStore.createBookmark(bookmarkForm.value)
  }
  showBookmarkModal.value = false
}

async function removeBookmark(id: string) {
  if (confirm('确认删除此书签吗？')) {
    await bookmarksStore.deleteBookmark(id)
  }
}

const fetchingAdminFavicon = ref(false)

async function autoFetchAdminIcon() {
  const targetUrl = bookmarkForm.value.url_internal || bookmarkForm.value.url_fallback || bookmarkForm.value.url_public_template
  if (!targetUrl) {
    alert('请先输入内网直连地址或中继域名')
    return
  }
  fetchingAdminFavicon.value = true
  try {
    const res = await api.get(`/bookmarks/fetch-favicon?url=${encodeURIComponent(targetUrl)}`)
    const data = res.data
    if (data.title && !bookmarkForm.value.name) {
      bookmarkForm.value.name = data.title
    }
    if (data.google_favicon) {
      bookmarkForm.value.icon = data.google_favicon
    } else if (data.icon_url) {
      bookmarkForm.value.icon = data.icon_url
    } else if (data.suggested_icon) {
      bookmarkForm.value.icon = data.suggested_icon
    }
  } catch (e) {
    alert('抓取失败，请手动输入图标')
  } finally {
    fetchingAdminFavicon.value = false
  }
}

// Group actions
function openCreateGroup() {
  groupForm.value = { id: '', name: '', icon: 'tabler:folder' }
  showGroupModal.value = true
}

function openEditGroup(g: Group) {
  groupForm.value = { id: g.id, name: g.name, icon: g.icon || '' }
  showGroupModal.value = true
}

async function saveGroup() {
  if (!groupForm.value.name) return
  if (groupForm.value.id) {
    await bookmarksStore.updateGroup(groupForm.value.id, groupForm.value.name, groupForm.value.icon)
  } else {
    await bookmarksStore.createGroup(groupForm.value.name, groupForm.value.icon)
  }
  showGroupModal.value = false
}

async function removeGroup(id: string) {
  if (confirm('确认删除此分组吗？分组下的书签不会被删除')) {
    await bookmarksStore.deleteGroup(id)
  }
}

// Tag actions
async function saveTag() {
  if (!tagForm.value.name) return
  await bookmarksStore.createTag(tagForm.value.name, tagForm.value.color)
  tagForm.value = { name: '', color: '#6366f1' }
  showTagModal.value = false
}

async function removeTag(id: string) {
  if (confirm('确认删除此标签？')) {
    await bookmarksStore.deleteTag(id)
  }
}

// Account & Security actions
async function handleUpdateAccount() {
  accountMsg.value = ''
  if (!accountName.value.trim()) {
    accountMsg.value = '账号名称不能为空'
    return
  }
  const res = await authStore.updateProfile(accountName.value.trim())
  if (res.success) {
    accountMsg.value = '管理员账号名称修改成功！下次请使用新账号登录。'
  } else {
    accountMsg.value = res.error || '修改失败'
  }
}

async function handleToggleRequireLogin() {
  securityMsg.value = ''
  try {
    await settingsStore.saveSettings({
      require_login: settingsStore.requireLogin ? 'true' : 'false',
    })
    securityMsg.value = settingsStore.requireLogin
      ? '已开启私密模式：未登录访客将直接跳转至登录页'
      : '已关闭私密模式：公共书签对所有访客公开可见'
  } catch (err: any) {
    securityMsg.value = '保存失败: ' + (err.message || '未知错误')
  }
}

// Change Password
async function handleChangePassword() {
  pwdMsg.value = ''
  if (newPwd.value.length < 6) {
    pwdMsg.value = '新密码长度至少 6 位'
    return
  }
  const res = await authStore.changePassword(oldPwd.value, newPwd.value)
  if (res.success) {
    pwdMsg.value = '密码修改成功！'
    oldPwd.value = ''
    newPwd.value = ''
  } else {
    pwdMsg.value = res.error || '修改失败'
  }
}

// Backup & Restore
function exportZip() {
  window.open('/api/backup/export', '_blank')
}

async function handleRestoreZip(e: Event) {
  const target = e.target as HTMLInputElement
  if (!target.files || target.files.length === 0) return

  const file = target.files[0]
  const formData = new FormData()
  formData.append('file', file)

  importing.value = true
  backupMsg.value = '正在还原数据...'
  try {
    const res = await api.post('/backup/import', formData)
    backupMsg.value = res.data.message || '还原成功！'
    await bookmarksStore.fetchAll()
    await settingsStore.fetchSettings()
  } catch (err: any) {
    backupMsg.value = '还原失败: ' + (err.response?.data?.error || err.message)
  } finally {
    importing.value = false
  }
}

function exportHTML() {
  window.open('/api/bookmarks/export?format=html', '_blank')
}

function exportJSON() {
  window.open('/api/bookmarks/export?format=json', '_blank')
}

async function handleImportBookmarks(e: Event) {
  const target = e.target as HTMLInputElement
  if (!target.files || target.files.length === 0) return

  const file = target.files[0]
  const formData = new FormData()
  formData.append('file', file)

  importing.value = true
  backupMsg.value = '正在导入书签...'
  try {
    const res = await api.post('/bookmarks/import', formData)
    backupMsg.value = res.data.message || '导入成功！'
    await bookmarksStore.fetchAll()
  } catch (err: any) {
    backupMsg.value = '导入失败: ' + (err.response?.data?.error || err.message)
  } finally {
    importing.value = false
  }
}

function handleLogout() {
  authStore.logout()
  router.push('/login')
}
</script>

<template>
  <div class="min-h-screen bg-slate-50/50 dark:bg-slate-950/50 flex flex-col">
    <!-- Top Bar -->
    <header class="h-16 border-b border-slate-200/80 dark:border-slate-800/80 bg-white/70 dark:bg-slate-900/70 backdrop-blur-xl px-4 sm:px-8 flex items-center justify-between">
      <div class="flex items-center gap-3">
        <router-link to="/" class="flex items-center gap-2 text-indigo-600 dark:text-indigo-400 font-bold text-sm hover:opacity-80 transition-opacity">
          <Icon icon="tabler:arrow-left" class="w-4 h-4" />
          <span>返回首页</span>
        </router-link>
        <span class="text-slate-300 dark:text-slate-700">|</span>
        <h2 class="text-sm font-semibold text-slate-800 dark:text-slate-200">SmartPanel 管理控制台</h2>
      </div>

      <div class="flex items-center gap-3">
        <span class="text-xs text-slate-500 hidden sm:inline">{{ authStore.user?.email }}</span>
        <button
          type="button"
          @click="handleLogout"
          class="px-3 py-1.5 text-xs text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/40 rounded-xl transition-colors font-medium"
        >
          安全退出
        </button>
      </div>
    </header>

    <!-- Content Layout -->
    <div class="flex-1 flex max-w-7xl w-full mx-auto p-4 sm:p-6 lg:p-8 gap-6 flex-col md:flex-row">
      <!-- Sidebar Navigation Tabs -->
      <aside class="w-full md:w-56 shrink-0 flex md:flex-col gap-1 overflow-x-auto pb-2 md:pb-0">
        <button
          v-for="t in tabs"
          :key="t.id"
          type="button"
          @click="activeTab = t.id"
          class="flex items-center gap-2.5 px-3.5 py-2.5 rounded-xl text-xs font-semibold whitespace-nowrap transition-all duration-200"
          :class="[
            activeTab === t.id
              ? 'bg-indigo-600 text-white shadow-md shadow-indigo-600/20'
              : 'text-slate-600 dark:text-slate-400 hover:bg-white dark:hover:bg-slate-900'
          ]"
        >
          <Icon :icon="t.icon" class="w-4 h-4" />
          <span>{{ t.name }}</span>
        </button>
      </aside>

      <!-- Main Tab Body -->
      <main class="flex-1 bg-white/70 dark:bg-slate-900/70 backdrop-blur-xl rounded-3xl border border-slate-200/80 dark:border-slate-800/80 p-6 shadow-sm overflow-hidden">
        <!-- 1. Bookmarks & Groups Tab -->
        <div v-if="activeTab === 'bookmarks'" class="space-y-8">
          <div class="flex items-center justify-between pb-4 border-b border-slate-200/60 dark:border-slate-800/60">
            <div>
              <h3 class="text-base font-bold text-slate-800 dark:text-slate-100">书签与分组管理</h3>
              <p class="text-xs text-slate-500">创建个性化服务卡片、配置内外网自适应地址</p>
            </div>
            <div class="flex gap-2">
              <button
                @click="openCreateGroup"
                class="px-3 py-1.5 bg-slate-100 dark:bg-slate-800 hover:bg-slate-200 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 rounded-xl text-xs font-medium transition-colors"
              >
                + 添加分组
              </button>
              <button
                @click="openCreateBookmark"
                class="px-3.5 py-1.5 bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl text-xs font-semibold shadow-xs transition-colors"
              >
                + 添加书签
              </button>
            </div>
          </div>

          <!-- Groups List -->
          <div class="space-y-6">
            <div
              v-for="g in bookmarksStore.groupedBookmarks"
              :key="g.id"
              class="p-4 rounded-2xl bg-slate-50/60 dark:bg-slate-800/30 border border-slate-200/60 dark:border-slate-800/60 space-y-3"
            >
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <Icon v-if="g.icon" :icon="g.icon" class="w-4 h-4 text-indigo-600" />
                  <span class="font-bold text-sm text-slate-800 dark:text-slate-200">{{ g.name }}</span>
                  <span class="text-xs text-slate-400">({{ g.bookmarks?.length || 0 }})</span>
                </div>
                <div class="flex items-center gap-1.5">
                  <button
                    @click="openEditGroup(g)"
                    class="p-1 text-slate-400 hover:text-indigo-600 rounded-lg text-xs"
                    title="编辑分组"
                  >
                    <Icon icon="tabler:edit" class="w-4 h-4" />
                  </button>
                  <button
                    @click="removeGroup(g.id)"
                    class="p-1 text-slate-400 hover:text-rose-600 rounded-lg text-xs"
                    title="删除分组"
                  >
                    <Icon icon="tabler:trash" class="w-4 h-4" />
                  </button>
                </div>
              </div>

              <!-- Bookmarks table inside group -->
              <div v-if="g.bookmarks && g.bookmarks.length > 0" class="divide-y divide-slate-200/40 dark:divide-slate-800/40">
                <div
                  v-for="b in g.bookmarks"
                  :key="b.id"
                  class="py-2.5 flex items-center justify-between gap-4 text-xs group"
                >
                  <div class="flex items-center gap-2.5 min-w-0">
                    <div class="w-6 h-6 rounded-md bg-white dark:bg-slate-800 flex items-center justify-center text-indigo-600 shrink-0 shadow-2xs">
                      <Icon v-if="b.icon" :icon="b.icon" class="w-3.5 h-3.5" />
                      <Icon v-else icon="tabler:link" class="w-3.5 h-3.5" />
                    </div>
                    <div class="min-w-0">
                      <div class="font-semibold text-slate-800 dark:text-slate-200 truncate">
                        {{ b.name }}
                        <span v-if="b.is_private" class="ml-1 text-[10px] text-amber-600 font-normal">🔒仅私有</span>
                      </div>
                      <div class="text-[11px] text-slate-400 font-mono truncate">
                        {{ b.url_public_template || b.url_internal || b.url_fallback }}
                      </div>
                    </div>
                  </div>

                  <div class="flex items-center gap-1 shrink-0">
                    <button
                      @click="openEditBookmark(b)"
                      class="px-2 py-1 text-slate-500 hover:text-indigo-600 font-medium rounded-md hover:bg-slate-200/50 dark:hover:bg-slate-700/50"
                    >
                      编辑
                    </button>
                    <button
                      @click="removeBookmark(b.id)"
                      class="px-2 py-1 text-slate-400 hover:text-rose-600 rounded-md hover:bg-rose-50 dark:hover:bg-rose-950/40"
                    >
                      删除
                    </button>
                  </div>
                </div>
              </div>

              <div v-else class="text-xs text-slate-400 py-2 italic">
                暂无书签
              </div>
            </div>
          </div>
        </div>

        <!-- 2. Tags Tab -->
        <div v-else-if="activeTab === 'tags'" class="space-y-6">
          <div class="flex items-center justify-between pb-4 border-b border-slate-200/60 dark:border-slate-800/60">
            <div>
              <h3 class="text-base font-bold text-slate-800 dark:text-slate-100">标签系统管理</h3>
              <p class="text-xs text-slate-500">为书签打上多维度分类标签（如 媒体、下载、网络管理）</p>
            </div>
            <button
              @click="showTagModal = true"
              class="px-3.5 py-1.5 bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl text-xs font-semibold shadow-xs"
            >
              + 新增标签
            </button>
          </div>

          <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
            <div
              v-for="t in bookmarksStore.tags"
              :key="t.id"
              class="flex items-center justify-between p-3 rounded-xl border border-slate-200/80 dark:border-slate-800/80 bg-slate-50/50 dark:bg-slate-800/30 hover:border-slate-300 dark:hover:border-slate-700 transition-all"
            >
              <div class="flex items-center gap-2 truncate pr-2">
                <span class="w-3 h-3 rounded-full shrink-0" :style="{ backgroundColor: t.color }"></span>
                <span class="text-xs font-semibold text-slate-800 dark:text-slate-200 truncate">{{ t.name }}</span>
              </div>
              <button
                type="button"
                @click="removeTag(t.id)"
                class="flex items-center gap-1 text-slate-400 hover:text-rose-600 hover:bg-rose-50 dark:hover:bg-rose-950/40 p-1.5 rounded-lg text-xs transition-colors cursor-pointer shrink-0"
                title="删除此标签"
              >
                <Icon icon="tabler:trash" class="w-4 h-4" />
              </button>
            </div>
          </div>
        </div>

        <!-- 3. Appearance Tab -->
        <div v-else-if="activeTab === 'appearance'" class="space-y-10">
          <!-- Preset Themes Quick Switcher -->
          <div class="p-5 rounded-2xl bg-indigo-50/70 dark:bg-indigo-950/40 border border-indigo-200/80 dark:border-indigo-800/80 space-y-3">
            <div class="flex items-center justify-between">
              <div>
                <h3 class="text-base font-bold text-slate-900 dark:text-white flex items-center gap-2">
                  <span>🎨 预置主题一键应用</span>
                  <span class="text-xs px-2 py-0.5 rounded-full bg-indigo-600 text-white font-medium">推荐</span>
                </h3>
                <p class="text-xs text-slate-500 dark:text-slate-400 mt-0.5">一键配置卡片风格、高质壁纸与登录页视觉体系</p>
              </div>
            </div>

            <div class="grid grid-cols-2 sm:grid-cols-4 gap-3 pt-1">
              <button
                type="button"
                @click="settingsStore.applyPresetTheme('crystal')"
                class="p-3.5 rounded-xl border text-left transition-all flex flex-col gap-1 cursor-pointer"
                :class="[
                  settingsStore.cardStyle === 'transparent'
                    ? 'border-indigo-500 bg-white dark:bg-slate-900 text-indigo-600 dark:text-indigo-400 font-semibold shadow-md ring-2 ring-indigo-500/30'
                    : 'border-slate-200 dark:border-slate-800 bg-white/60 dark:bg-slate-900/60 hover:bg-white text-slate-700 dark:text-slate-300'
                ]"
              >
                <div class="flex items-center justify-between">
                  <span class="text-sm font-bold">✨ 全透明模式</span>
                  <span v-if="settingsStore.cardStyle === 'transparent'" class="text-xs text-indigo-600 font-bold">当前</span>
                </div>
                <span class="text-[11px] text-slate-400 font-normal leading-tight">超清无框卡片，原图透光质感</span>
              </button>

              <button
                type="button"
                @click="settingsStore.applyPresetTheme('glass')"
                class="p-3.5 rounded-xl border text-left transition-all flex flex-col gap-1 cursor-pointer"
                :class="[
                  settingsStore.cardStyle === 'glass'
                    ? 'border-indigo-500 bg-white dark:bg-slate-900 text-indigo-600 dark:text-indigo-400 font-semibold shadow-md ring-2 ring-indigo-500/30'
                    : 'border-slate-200 dark:border-slate-800 bg-white/60 dark:bg-slate-900/60 hover:bg-white text-slate-700 dark:text-slate-300'
                ]"
              >
                <div class="flex items-center justify-between">
                  <span class="text-sm font-bold">🧊 经典毛玻璃</span>
                  <span v-if="settingsStore.cardStyle === 'glass'" class="text-xs text-indigo-600 font-bold">当前</span>
                </div>
                <span class="text-[11px] text-slate-400 font-normal leading-tight">半透明亚克力模糊毛玻璃</span>
              </button>

              <button
                type="button"
                @click="settingsStore.applyPresetTheme('solid')"
                class="p-3.5 rounded-xl border text-left transition-all flex flex-col gap-1 cursor-pointer"
                :class="[
                  settingsStore.cardStyle === 'solid'
                    ? 'border-indigo-500 bg-white dark:bg-slate-900 text-indigo-600 dark:text-indigo-400 font-semibold shadow-md ring-2 ring-indigo-500/30'
                    : 'border-slate-200 dark:border-slate-800 bg-white/60 dark:bg-slate-900/60 hover:bg-white text-slate-700 dark:text-slate-300'
                ]"
              >
                <div class="flex items-center justify-between">
                  <span class="text-sm font-bold">◻️ 纯色卡片</span>
                  <span v-if="settingsStore.cardStyle === 'solid'" class="text-xs text-indigo-600 font-bold">当前</span>
                </div>
                <span class="text-[11px] text-slate-400 font-normal leading-tight">经典实体白/深色卡片</span>
              </button>

              <button
                type="button"
                @click="settingsStore.applyPresetTheme('minimal')"
                class="p-3.5 rounded-xl border text-left transition-all flex flex-col gap-1 cursor-pointer"
                :class="[
                  settingsStore.cardStyle === 'minimal'
                    ? 'border-indigo-500 bg-white dark:bg-slate-900 text-indigo-600 dark:text-indigo-400 font-semibold shadow-md ring-2 ring-indigo-500/30'
                    : 'border-slate-200 dark:border-slate-800 bg-white/60 dark:bg-slate-900/60 hover:bg-white text-slate-700 dark:text-slate-300'
                ]"
              >
                <div class="flex items-center justify-between">
                  <span class="text-sm font-bold">▫️ 极简线条</span>
                  <span v-if="settingsStore.cardStyle === 'minimal'" class="text-xs text-indigo-600 font-bold">当前</span>
                </div>
                <span class="text-[11px] text-slate-400 font-normal leading-tight">无背景阴影简洁平铺</span>
              </button>
            </div>
          </div>

          <div>
            <h3 class="text-base font-bold text-slate-800 dark:text-slate-100 mb-1">卡片风格与排版</h3>
            <CardStyleSettings />
          </div>

          <div class="pt-6 border-t border-slate-200/60 dark:border-slate-800/60">
            <h3 class="text-base font-bold text-slate-800 dark:text-slate-100 mb-1">背景与壁纸管理</h3>
            <WallpaperManager />
          </div>

          <div class="pt-6 border-t border-slate-200/60 dark:border-slate-800/60">
            <h3 class="text-base font-bold text-slate-800 dark:text-slate-100 mb-1">字体与排版定制</h3>
            <FontSettings />
          </div>

          <div class="pt-6 border-t border-slate-200/60 dark:border-slate-800/60">
            <h3 class="text-base font-bold text-slate-800 dark:text-slate-100 mb-1">自定义 CSS / JS 代码注入</h3>
            <CustomCode />
          </div>

          <div class="pt-6 border-t border-slate-200/60 dark:border-slate-800/60 flex justify-end">
            <button
              @click="settingsStore.saveSettings({
                card_style: settingsStore.cardStyle,
                card_border_radius: String(settingsStore.cardBorderRadius),
                card_shadow: settingsStore.cardShadow,
                icon_size: settingsStore.iconSize,
                grid_cols_desktop: String(settingsStore.gridColsDesktop),
                grid_cols_tablet: String(settingsStore.gridColsTablet),
                grid_cols_mobile: String(settingsStore.gridColsMobile),
                wallpaper_type: settingsStore.wallpaperType,
                wallpaper_blur: String(settingsStore.wallpaperBlur),
                wallpaper_mask: String(settingsStore.wallpaperMask),
                wallpaper_url: settingsStore.wallpaperUrl,
                login_wallpaper_type: settingsStore.loginWallpaperType,
                login_wallpaper_url: settingsStore.loginWallpaperUrl,
                login_wallpaper_blur: String(settingsStore.loginWallpaperBlur),
                login_wallpaper_mask: String(settingsStore.loginWallpaperMask),
                text_opacity: String(settingsStore.textOpacity),
                content_top_offset: String(settingsStore.contentTopOffset),
                show_group_titles: String(settingsStore.showGroupTitles),
                font_family: settingsStore.fontFamily,
                font_size: settingsStore.fontSize,
                custom_css: settingsStore.customCss,
                custom_js: settingsStore.customJs,
                footer_text: settingsStore.footerText,
              })"
              class="px-6 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl text-xs font-semibold shadow-md transition-colors"
            >
              保存全部外观配置
            </button>
          </div>
        </div>

        <!-- 4. Network & DDNS Tab -->
        <div v-else-if="activeTab === 'network'">
          <NetworkSettings />
        </div>

        <!-- 5. System & Docker Tab -->
        <div v-else-if="activeTab === 'system'" class="space-y-8">
          <div>
            <h3 class="text-base font-bold text-slate-800 dark:text-slate-100 mb-2">NAS 主机硬件状态</h3>
            <SystemStatus />
          </div>
          <div class="pt-6 border-t border-slate-200/60 dark:border-slate-800/60">
            <DockerStatus />
          </div>
        </div>

        <!-- 6. Backup & Restore Tab -->
        <div v-else-if="activeTab === 'backup'" class="space-y-8">
          <div>
            <h3 class="text-base font-bold text-slate-800 dark:text-slate-100 mb-1">全量灾备与数据迁移</h3>
            <p class="text-xs text-slate-500 mb-4">打包 SQLite 数据库文件与所有上传图标/壁纸生成 ZIP 压缩包</p>

            <div class="flex flex-wrap gap-3">
              <button
                @click="exportZip"
                class="px-4 py-2 bg-indigo-600 hover:bg-indigo-700 text-white text-xs font-semibold rounded-xl shadow-xs transition-colors"
              >
                📦 下载全量备份 ZIP
              </button>

              <label class="px-4 py-2 bg-slate-100 hover:bg-slate-200 dark:bg-slate-800 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 text-xs font-medium rounded-xl cursor-pointer transition-colors">
                <span>{{ importing ? '还原中...' : '📥 上传 ZIP 恢复备份' }}</span>
                <input type="file" accept=".zip" class="hidden" @change="handleRestoreZip" :disabled="importing" />
              </label>
            </div>
          </div>

          <div class="pt-6 border-t border-slate-200/60 dark:border-slate-800/60">
            <h3 class="text-base font-bold text-slate-800 dark:text-slate-100 mb-1">书签导入与导出</h3>
            <p class="text-xs text-slate-500 mb-4">兼容浏览器 Netscape HTML 书签格式与 JSON 标准配置文件</p>

            <div class="flex flex-wrap gap-3">
              <button
                @click="exportHTML"
                class="px-3.5 py-2 bg-slate-100 hover:bg-slate-200 dark:bg-slate-800 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 text-xs font-medium rounded-xl transition-colors"
              >
                导出 HTML 书签
              </button>
              <button
                @click="exportJSON"
                class="px-3.5 py-2 bg-slate-100 hover:bg-slate-200 dark:bg-slate-800 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-200 text-xs font-medium rounded-xl transition-colors"
              >
                导出 JSON 配置
              </button>
              <label class="px-3.5 py-2 bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400 border border-indigo-200 dark:border-indigo-800 text-xs font-semibold rounded-xl cursor-pointer hover:bg-indigo-100 transition-colors">
                <span>{{ importing ? '导入中...' : '导入 HTML / JSON 书签' }}</span>
                <input type="file" accept=".html,.htm,.json" class="hidden" @change="handleImportBookmarks" :disabled="importing" />
              </label>
            </div>
          </div>

          <div v-if="backupMsg" class="p-3 rounded-xl bg-indigo-50 dark:bg-indigo-950/40 text-xs text-indigo-600 font-medium">
            {{ backupMsg }}
          </div>
        </div>

        <!-- 7. Account Tab -->
        <div v-else-if="activeTab === 'account'" class="space-y-6 max-w-lg">
          <!-- 1. 访问控制与私密模式 -->
          <div class="p-5 rounded-2xl bg-slate-50/80 dark:bg-slate-800/50 border border-slate-200/80 dark:border-slate-800 space-y-4">
            <div>
              <h3 class="text-base font-bold text-slate-800 dark:text-slate-100 mb-1">访问权限与私密模式</h3>
              <p class="text-xs text-slate-500">控制是否需要登录管理员账号才能查看面板书签内容</p>
            </div>

            <div class="flex items-center justify-between py-2">
              <div>
                <div class="text-xs font-semibold text-slate-700 dark:text-slate-300">强制登录后才显示导航页 (私密模式)</div>
                <div class="text-[11px] text-slate-500 mt-0.5">开启后，未登录访客访问首页将自动跳转至登录页，完全保护书签隐私</div>
              </div>
              <label class="relative inline-flex items-center cursor-pointer">
                <input
                  type="checkbox"
                  v-model="settingsStore.requireLogin"
                  @change="handleToggleRequireLogin"
                  class="sr-only peer"
                />
                <div class="w-11 h-6 bg-slate-200 peer-focus:outline-none rounded-full peer dark:bg-slate-700 peer-checked:after:translate-x-full peer-checked:after:border-white border border-transparent after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all dark:border-slate-600 peer-checked:bg-indigo-600"></div>
              </label>
            </div>

            <div v-if="securityMsg" class="p-2.5 rounded-xl bg-indigo-50 dark:bg-indigo-950/40 text-xs font-medium text-indigo-600 dark:text-indigo-400">
              {{ securityMsg }}
            </div>
          </div>

          <!-- 2. 管理员账号名称修改 -->
          <div class="p-5 rounded-2xl bg-slate-50/80 dark:bg-slate-800/50 border border-slate-200/80 dark:border-slate-800 space-y-4">
            <div>
              <h3 class="text-base font-bold text-slate-800 dark:text-slate-100 mb-1">管理员账号名称</h3>
              <p class="text-xs text-slate-500">自定义登录时使用的管理员名称（可改为简短易记的用户名，如 admin、nas 或个人邮箱）</p>
            </div>

            <div class="space-y-3">
              <div>
                <label class="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">管理员账号 / 用户名</label>
                <input
                  type="text"
                  v-model="accountName"
                  placeholder="例如：admin 或 mynas"
                  class="w-full px-3 py-2 text-xs bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 focus:outline-none focus:border-indigo-500"
                />
              </div>

              <div v-if="accountMsg" class="p-2.5 rounded-xl text-xs font-medium" :class="accountMsg.includes('成功') ? 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400' : 'bg-rose-50 dark:bg-rose-950/40 text-rose-500'">
                {{ accountMsg }}
              </div>

              <button
                @click="handleUpdateAccount"
                class="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl text-xs font-semibold transition-colors"
              >
                保存账号名称
              </button>
            </div>
          </div>

          <!-- 3. 管理员密码修改 -->
          <div class="p-5 rounded-2xl bg-slate-50/80 dark:bg-slate-800/50 border border-slate-200/80 dark:border-slate-800 space-y-4">
            <div>
              <h3 class="text-base font-bold text-slate-800 dark:text-slate-100 mb-1">管理员密码修改</h3>
              <p class="text-xs text-slate-500">修改登录密码，保护面板管理权限</p>
            </div>

            <div class="space-y-3">
              <div>
                <label class="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">当前原密码</label>
                <input
                  type="password"
                  v-model="oldPwd"
                  placeholder="请输入原密码"
                  class="w-full px-3 py-2 text-xs bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 focus:outline-none focus:border-indigo-500"
                />
              </div>
              <div>
                <label class="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">设定新密码</label>
                <input
                  type="password"
                  v-model="newPwd"
                  placeholder="至少 6 位密码"
                  class="w-full px-3 py-2 text-xs bg-white dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700 focus:outline-none focus:border-indigo-500"
                />
              </div>
              <div v-if="pwdMsg" class="p-2.5 rounded-xl text-xs font-medium" :class="pwdMsg.includes('成功') ? 'bg-emerald-50 dark:bg-emerald-950/40 text-emerald-600 dark:text-emerald-400' : 'bg-rose-50 dark:bg-rose-950/40 text-rose-500'">
                {{ pwdMsg }}
              </div>
              <button
                @click="handleChangePassword"
                class="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl text-xs font-semibold transition-colors"
              >
                更新密码
              </button>
            </div>
          </div>
        </div>
      </main>
    </div>

    <!-- Bookmark Edit Modal -->
    <div v-if="showBookmarkModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-md">
      <div class="w-full max-w-lg p-6 rounded-3xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-2xl space-y-4 max-h-[90vh] overflow-y-auto">
        <div class="flex items-center justify-between">
          <h3 class="text-base font-bold text-slate-800 dark:text-slate-100">
            {{ editingBookmark ? '编辑书签卡片' : '添加新书签' }}
          </h3>
          <button @click="showBookmarkModal = false" class="text-slate-400 hover:text-slate-600">✕</button>
        </div>

        <div class="space-y-3 text-xs">
          <div>
            <label class="block font-medium mb-1">书签名称 *</label>
            <input v-model="bookmarkForm.name" required class="w-full px-3 py-2 bg-slate-50 dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700" />
          </div>

          <div>
            <div class="flex items-center justify-between mb-1">
              <label class="font-medium">图标 (Iconify 名称如 tabler:server，或图片 URL)</label>
              <button
                type="button"
                @click="autoFetchAdminIcon"
                :disabled="fetchingAdminFavicon"
                class="px-2 py-0.5 rounded-lg bg-indigo-50 dark:bg-indigo-950/60 text-indigo-600 dark:text-indigo-400 hover:bg-indigo-100 text-[11px] font-semibold flex items-center gap-1 transition-all cursor-pointer"
                title="根据输入的网址自动识别并抓取高清图标和标题"
              >
                <Icon :icon="fetchingAdminFavicon ? 'tabler:loader-2' : 'tabler:sparkles'" class="w-3.5 h-3.5" :class="{ 'animate-spin': fetchingAdminFavicon }" />
                <span>{{ fetchingAdminFavicon ? '抓取中...' : '智能抓取' }}</span>
              </button>
            </div>
            <input v-model="bookmarkForm.icon" placeholder="tabler:link" class="w-full px-3 py-2 bg-slate-50 dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700" />
          </div>

          <div>
            <label class="block font-medium mb-1">外网动态模板地址 (支持 {ipv6}, {domain}, {v6domain})</label>
            <input v-model="bookmarkForm.url_public_template" placeholder="https://[{ipv6}]:5666" class="w-full px-3 py-2 bg-slate-50 dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700" />
            <p class="text-[11px] text-slate-400 mt-0.5">例如：https://[{ipv6}]:5666 或 https://{v6domain}:5666</p>
          </div>

          <div>
            <label class="block font-medium mb-1">内网直连地址 (局域网环境)</label>
            <input v-model="bookmarkForm.url_internal" placeholder="http://192.168.1.100:5666" class="w-full px-3 py-2 bg-slate-50 dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700" />
          </div>

          <div>
            <label class="block font-medium mb-1">Cloudflare 代理回退地址 (IPv4 无法直连时使用)</label>
            <input v-model="bookmarkForm.url_fallback" placeholder="https://{domain}" class="w-full px-3 py-2 bg-slate-50 dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700" />
          </div>

          <div>
            <label class="block font-medium mb-1">所属分组</label>
            <select v-model="bookmarkForm.group_id" class="w-full px-3 py-2 bg-slate-50 dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700">
              <option v-for="g in bookmarksStore.groups" :key="g.id" :value="g.id">{{ g.name }}</option>
            </select>
          </div>

          <div>
            <label class="block font-medium mb-1">描述信息</label>
            <textarea v-model="bookmarkForm.description" rows="2" class="w-full px-3 py-2 bg-slate-50 dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700"></textarea>
          </div>

          <!-- Tags selection -->
          <div>
            <label class="block font-medium mb-1">关联标签</label>
            <div class="flex flex-wrap gap-2 pt-1">
              <label
                v-for="t in bookmarksStore.tags"
                :key="t.id"
                class="flex items-center gap-1.5 px-2.5 py-1 rounded-lg border text-xs cursor-pointer select-none"
                :class="[
                  bookmarkForm.tag_ids.includes(t.id)
                    ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 font-semibold'
                    : 'border-slate-200 dark:border-slate-800 text-slate-600 dark:text-slate-400'
                ]"
              >
                <input
                  type="checkbox"
                  :value="t.id"
                  v-model="bookmarkForm.tag_ids"
                  class="hidden"
                />
                <span class="w-2 h-2 rounded-full" :style="{ backgroundColor: t.color }"></span>
                <span>{{ t.name }}</span>
              </label>
            </div>
          </div>

          <div class="flex items-center gap-4 pt-2">
            <label class="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" v-model="bookmarkForm.open_in_new_tab" class="rounded accent-indigo-600" />
              <span>新标签页打开</span>
            </label>
            <label class="flex items-center gap-2 cursor-pointer">
              <input type="checkbox" v-model="bookmarkForm.is_private" class="rounded accent-indigo-600" />
              <span>仅登录可见 (私有)</span>
            </label>
          </div>
        </div>

        <div class="flex justify-end gap-2 pt-2 border-t border-slate-200 dark:border-slate-800">
          <button @click="showBookmarkModal = false" class="px-4 py-2 text-xs text-slate-600 rounded-xl hover:bg-slate-100 dark:hover:bg-slate-800">取消</button>
          <button @click="saveBookmark" class="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl text-xs font-semibold">保存</button>
        </div>
      </div>
    </div>

    <!-- Group Modal -->
    <div v-if="showGroupModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-md">
      <div class="w-full max-w-sm p-6 rounded-3xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-2xl space-y-4">
        <h3 class="text-base font-bold text-slate-800 dark:text-slate-100">
          {{ groupForm.id ? '编辑分组' : '新增分组' }}
        </h3>
        <div class="space-y-3 text-xs">
          <div>
            <label class="block font-medium mb-1">分组名称 *</label>
            <input v-model="groupForm.name" required class="w-full px-3 py-2 bg-slate-50 dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700" />
          </div>
          <div>
            <label class="block font-medium mb-1">图标 (Iconify)</label>
            <input v-model="groupForm.icon" placeholder="tabler:folder" class="w-full px-3 py-2 bg-slate-50 dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700" />
          </div>
        </div>
        <div class="flex justify-end gap-2 pt-2">
          <button @click="showGroupModal = false" class="px-4 py-2 text-xs text-slate-600 rounded-xl">取消</button>
          <button @click="saveGroup" class="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl text-xs font-semibold">确定</button>
        </div>
      </div>
    </div>

    <!-- Tag Modal -->
    <div v-if="showTagModal" class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-md">
      <div class="w-full max-w-sm p-6 rounded-3xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-2xl space-y-4">
        <h3 class="text-base font-bold text-slate-800 dark:text-slate-100">新增标签</h3>
        <div class="space-y-3 text-xs">
          <div>
            <label class="block font-medium mb-1">标签名称 *</label>
            <input v-model="tagForm.name" required placeholder="如 媒体、下载" class="w-full px-3 py-2 bg-slate-50 dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700" />
          </div>
          <div>
            <label class="block font-medium mb-1">标签颜色</label>
            <div class="flex items-center gap-3">
              <input type="color" v-model="tagForm.color" class="w-10 h-8 rounded-lg cursor-pointer" />
              <input type="text" v-model="tagForm.color" class="flex-1 px-3 py-1.5 font-mono text-xs bg-slate-50 dark:bg-slate-800 rounded-xl border border-slate-200 dark:border-slate-700" />
            </div>
          </div>
        </div>
        <div class="flex justify-end gap-2 pt-2">
          <button @click="showTagModal = false" class="px-4 py-2 text-xs text-slate-600 rounded-xl">取消</button>
          <button @click="saveTag" class="px-5 py-2 bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl text-xs font-semibold">确定</button>
        </div>
      </div>
    </div>
  </div>
</template>
