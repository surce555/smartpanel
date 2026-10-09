<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Icon } from '@iconify/vue'
import { useApi } from '@/composables/useApi'

const api = useApi()

interface DockerContainer {
  id: string
  names: string[]
  image: string
  state: string
  status: string
  created: number
  cpu_percent: number
  ram_usage_str: string
  ram_bytes: number
}

const available = ref<boolean>(false)
const message = ref<string>('')
const containers = ref<DockerContainer[]>([])
const loading = ref<boolean>(false)
const operating = ref<Record<string, string>>({}) // container id -> 'starting' | 'stopping' | 'restarting'
const showInfo = ref<boolean>(false)
const searchQuery = ref<string>('')
const sortField = ref<'name' | 'status'>('name')
const sortOrder = ref<'asc' | 'desc'>('asc')

// Detail / Log modal
const inspectContainer = ref<DockerContainer | null>(null)
const inspectLogs = ref<string>('')
const logsLoading = ref<boolean>(false)
const copiedLog = ref<boolean>(false)

let autoRefreshTimer: any = null

async function fetchContainers(silent = false) {
  if (!silent) loading.value = true
  try {
    const resp = await api.get('/system/docker')
    available.value = resp.data.available
    message.value = resp.data.message || ''
    containers.value = resp.data.containers || []
  } catch (e: any) {
    available.value = false
    message.value = e.response?.data?.error || '请求 Docker API 失败'
  } finally {
    if (!silent) loading.value = false
  }
}

function toggleSort(field: 'name' | 'status') {
  if (sortField.value === field) {
    sortOrder.value = sortOrder.value === 'asc' ? 'desc' : 'asc'
  } else {
    sortField.value = field
    sortOrder.value = 'asc'
  }
}

const filteredAndSortedContainers = computed(() => {
  let list = [...containers.value]
  if (searchQuery.value.trim()) {
    const q = searchQuery.value.trim().toLowerCase()
    list = list.filter(c => {
      const name = (c.names[0] || c.id).toLowerCase()
      const img = c.image.toLowerCase()
      return name.includes(q) || img.includes(q)
    })
  }

  list.sort((a, b) => {
    if (sortField.value === 'name') {
      const nameA = (a.names[0] || a.id).toLowerCase()
      const nameB = (b.names[0] || b.id).toLowerCase()
      const cmp = nameA.localeCompare(nameB)
      return sortOrder.value === 'asc' ? cmp : -cmp
    } else {
      // Running first or stopped first
      const valA = a.state === 'running' ? 1 : 0
      const valB = b.state === 'running' ? 1 : 0
      return sortOrder.value === 'asc' ? valB - valA : valA - valB
    }
  })

  return list
})

async function handleToggleContainer(c: DockerContainer) {
  const isRunning = c.state === 'running'
  const action = isRunning ? 'stop' : 'start'

  if (isRunning) {
    const ok = window.confirm(`确定要停止容器「${c.names[0] || c.id}」吗？`)
    if (!ok) return
  }

  operating.value[c.id] = isRunning ? 'stopping' : 'starting'
  try {
    await api.post(`/system/docker/${c.id}/${action}`)
    await fetchContainers(true)
  } catch (e: any) {
    alert(`操作失败: ${e.response?.data?.error || e.message}`)
  } finally {
    delete operating.value[c.id]
  }
}

async function handleRestart(c: DockerContainer) {
  const ok = window.confirm(`确定要重启容器「${c.names[0] || c.id}」吗？`)
  if (!ok) return

  operating.value[c.id] = 'restarting'
  try {
    await api.post(`/system/docker/${c.id}/restart`)
    await fetchContainers(true)
  } catch (e: any) {
    alert(`重启失败: ${e.response?.data?.error || e.message}`)
  } finally {
    delete operating.value[c.id]
  }
}

async function handleOpenInspect(c: DockerContainer) {
  inspectContainer.value = c
  inspectLogs.value = ''
  logsLoading.value = true
  try {
    const resp = await api.get(`/system/docker/${c.id}/logs?tail=150`)
    inspectLogs.value = resp.data.logs || '暂无控制台日志'
  } catch (e: any) {
    inspectLogs.value = `获取日志失败: ${e.response?.data?.error || e.message}`
  } finally {
    logsLoading.value = false
  }
}

function copyLogs() {
  if (!inspectLogs.value) return
  navigator.clipboard.writeText(inspectLogs.value)
  copiedLog.value = true
  setTimeout(() => {
    copiedLog.value = false
  }, 2000)
}

onMounted(() => {
  fetchContainers()
  // Poll every 12 seconds for live CPU and RAM
  autoRefreshTimer = setInterval(() => {
    if (available.value && !loading.value) {
      fetchContainers(true)
    }
  }, 12000)
})

onUnmounted(() => {
  if (autoRefreshTimer) clearInterval(autoRefreshTimer)
})
</script>

<template>
  <div class="w-full bg-white dark:bg-slate-900 rounded-2xl shadow-sm border border-slate-200/80 dark:border-slate-800 p-4 sm:p-6 space-y-4">
    <!-- Top Action Bar (Matching Screenshot: ℹ 说明 | ↻ 刷新) -->
    <div class="flex items-center justify-between flex-wrap gap-3">
      <div class="flex items-center gap-2">
        <div class="inline-flex items-center rounded-full bg-sky-50 dark:bg-sky-950/40 border border-sky-100 dark:border-sky-900/50 px-3 py-1 text-xs text-slate-700 dark:text-slate-300 font-medium select-none shadow-xs">
          <button
            type="button"
            @click="showInfo = !showInfo"
            class="flex items-center gap-1 hover:text-sky-600 dark:hover:text-sky-400 transition-colors cursor-pointer"
            title="查看挂载与使用说明"
          >
            <Icon icon="tabler:info-circle" class="w-4 h-4 text-sky-500" />
            <span>说明</span>
          </button>
          <span class="mx-2 text-slate-300 dark:text-slate-600">|</span>
          <button
            type="button"
            @click="fetchContainers(false)"
            :disabled="loading"
            class="flex items-center gap-1 hover:text-sky-600 dark:hover:text-sky-400 transition-colors cursor-pointer disabled:opacity-50"
            title="重新获取容器状态与资源占用"
          >
            <Icon icon="tabler:refresh" class="w-4 h-4 text-sky-500" :class="{ 'animate-spin': loading }" />
            <span>刷新</span>
          </button>
        </div>

        <span v-if="available" class="text-xs text-slate-400 dark:text-slate-500 hidden sm:inline">
          共 {{ containers.length }} 个容器（{{ containers.filter(c => c.state === 'running').length }} 运行中）
        </span>
      </div>

      <!-- Quick Search Filter -->
      <div v-if="available && containers.length > 0" class="relative">
        <Icon icon="tabler:search" class="w-3.5 h-3.5 text-slate-400 absolute left-2.5 top-1/2 -translate-y-1/2" />
        <input
          v-model="searchQuery"
          type="text"
          placeholder="搜索容器或镜像..."
          class="pl-8 pr-3 py-1 text-xs rounded-xl bg-slate-50 dark:bg-slate-800/80 border border-slate-200 dark:border-slate-700 text-slate-700 dark:text-slate-200 focus:outline-none focus:ring-1 focus:ring-indigo-500 w-36 sm:w-48 transition-all"
        />
      </div>
    </div>

    <!-- Info / Guide Accordion Banner -->
    <transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0 -translate-y-2"
      enter-to-class="opacity-100 translate-y-0"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="opacity-100 translate-y-0"
      leave-to-class="opacity-0 -translate-y-2"
    >
      <div
        v-if="showInfo"
        class="p-4 rounded-xl bg-sky-50/80 dark:bg-sky-950/30 border border-sky-100 dark:border-sky-900/40 text-xs text-slate-700 dark:text-slate-300 space-y-2"
      >
        <div class="font-semibold text-sky-800 dark:text-sky-300 flex items-center gap-1.5">
          <Icon icon="tabler:bulb" class="w-4 h-4 text-sky-500" />
          <span>Docker 容器监控与管理说明</span>
        </div>
        <ul class="list-disc list-inside space-y-1 text-slate-600 dark:text-slate-400 leading-relaxed text-[11px]">
          <li><strong>挂载要求：</strong>需要在 <code class="px-1 py-0.5 rounded bg-sky-100/70 dark:bg-sky-900/60 font-mono">docker-compose.yml</code> 中挂载套接字：<code class="px-1 py-0.5 rounded bg-sky-100/70 dark:bg-sky-900/60 font-mono">- /var/run/docker.sock:/var/run/docker.sock</code>。</li>
          <li><strong>启停控制：</strong>点击右侧滑动开关，可一键启动或停止对应容器（停止会有确认保护）。</li>
          <li><strong>资源概览：</strong>实时显示运行中容器的 CPU 占用率与内存（RAM）使用量。</li>
          <li><strong>重启与日志：</strong>点击操作列右侧的卡片详情图标，可一键重启容器并查看最新控制台日志。</li>
        </ul>
      </div>
    </transition>

    <!-- Socket Unavailable Warning -->
    <div
      v-if="!available && !loading"
      class="p-5 rounded-2xl bg-amber-50 dark:bg-amber-950/30 border border-amber-200/80 dark:border-amber-900/50 text-xs text-amber-900 dark:text-amber-300 space-y-2.5"
    >
      <div class="font-semibold flex items-center gap-2 text-sm text-amber-800 dark:text-amber-200">
        <Icon icon="tabler:alert-triangle" class="w-5 h-5 text-amber-500 shrink-0" />
        <span>{{ message || 'Docker Socket 未挂载或不可访问' }}</span>
      </div>
      <p class="leading-relaxed">
        若需在此面板监控和管理宿主机上的 Docker 容器，请在 SmartPanel 的 <code class="font-mono bg-amber-100/80 dark:bg-amber-900/60 px-1.5 py-0.5 rounded">docker-compose.yml</code> 中配置 volumes 挂载：
      </p>
      <pre class="p-3 bg-amber-100/60 dark:bg-amber-900/60 rounded-xl font-mono text-xs overflow-x-auto text-amber-950 dark:text-amber-100">volumes:
  - /var/run/docker.sock:/var/run/docker.sock</pre>
      <p class="text-[11px] text-amber-700 dark:text-amber-400">
        （若仅需监控只读状态，可加上 <code class="font-mono">:ro</code> ；若需在网页端启停容器，请勿加 :ro）
      </p>
    </div>

    <!-- Main Containers Table (Matching Screenshot Table Header & Rows) -->
    <div v-else class="overflow-x-auto -mx-4 sm:mx-0">
      <table class="w-full text-left text-xs border-collapse">
        <!-- Table Header -->
        <thead>
          <tr class="border-b border-slate-100 dark:border-slate-800 text-slate-600 dark:text-slate-400 bg-slate-50/60 dark:bg-slate-800/40">
            <!-- 容器名称 -->
            <th
              @click="toggleSort('name')"
              class="py-3 px-4 font-medium cursor-pointer hover:text-slate-900 dark:hover:text-slate-200 select-none min-w-[200px]"
            >
              <div class="flex items-center gap-1">
                <span>容器名称</span>
                <Icon
                  :icon="sortField === 'name' && sortOrder === 'desc' ? 'tabler:arrow-up' : 'tabler:arrow-down'"
                  class="w-3.5 h-3.5 text-slate-400"
                  :class="{ 'text-indigo-600 font-bold': sortField === 'name' }"
                />
              </div>
            </th>

            <!-- 状态 -->
            <th
              @click="toggleSort('status')"
              class="py-3 px-3 font-medium cursor-pointer hover:text-slate-900 dark:hover:text-slate-200 select-none w-28"
            >
              <div class="flex items-center gap-1">
                <span>状态</span>
                <Icon
                  :icon="sortField === 'status' && sortOrder === 'desc' ? 'tabler:arrow-up' : 'tabler:arrow-down'"
                  class="w-3.5 h-3.5 text-slate-400"
                  :class="{ 'text-indigo-600 font-bold': sortField === 'status' }"
                />
              </div>
            </th>

            <!-- 资源概览 -->
            <th class="py-3 px-3 font-medium w-36">
              <span>资源概览</span>
            </th>

            <!-- 操作 PRO -->
            <th class="py-3 px-4 font-medium text-right w-32">
              <div class="flex items-center justify-end gap-1.5">
                <span>操作</span>
                <span class="px-1.5 py-0.2 rounded bg-amber-100 dark:bg-amber-900/60 text-amber-800 dark:text-amber-300 font-bold text-[10px] tracking-wider">
                  PRO
                </span>
              </div>
            </th>
          </tr>
        </thead>

        <!-- Table Body -->
        <tbody class="divide-y divide-slate-100 dark:divide-slate-800/70">
          <tr
            v-for="c in filteredAndSortedContainers"
            :key="c.id"
            class="hover:bg-slate-50/70 dark:hover:bg-slate-800/30 transition-colors"
          >
            <!-- 容器名称 & 镜像 -->
            <td class="py-3 px-4">
              <div class="font-medium text-slate-800 dark:text-slate-100 text-sm">
                {{ c.names[0] || c.id }}
              </div>
              <div class="text-[11px] text-slate-400 dark:text-slate-500 font-mono truncate max-w-[220px] sm:max-w-xs md:max-w-md" :title="c.image">
                {{ c.image }}
              </div>
            </td>

            <!-- 状态 Badge -->
            <td class="py-3 px-3 align-middle">
              <span
                v-if="c.state === 'running'"
                class="px-2.5 py-0.5 rounded-full text-[11px] font-medium bg-emerald-50 dark:bg-emerald-950/50 text-emerald-600 dark:text-emerald-400 border border-emerald-200/50 dark:border-emerald-800/40 inline-flex items-center gap-1"
              >
                运行中
              </span>
              <span
                v-else
                class="px-2.5 py-0.5 rounded-full text-[11px] font-medium bg-slate-100 dark:bg-slate-800 text-slate-500 dark:text-slate-400 border border-slate-200/60 dark:border-slate-700/60 inline-flex items-center gap-1"
              >
                已停止
              </span>
            </td>

            <!-- 资源概览 (CPU: x% / RAM: xxMB) -->
            <td class="py-3 px-3 align-middle font-mono text-[11px]">
              <div v-if="c.state === 'running'" class="space-y-0.5 text-slate-700 dark:text-slate-300">
                <div>CPU: {{ c.cpu_percent }}%</div>
                <div>RAM: {{ c.ram_usage_str || '0MB' }}</div>
              </div>
              <div v-else class="text-slate-300 dark:text-slate-600">
                -
              </div>
            </td>

            <!-- 操作 (iOS Toggle Switch + Detail Action) -->
            <td class="py-3 px-4 align-middle text-right">
              <div class="flex items-center justify-end gap-2.5">
                <!-- Toggle Switch -->
                <button
                  type="button"
                  @click="handleToggleContainer(c)"
                  :disabled="!!operating[c.id]"
                  class="relative inline-flex h-6 w-11 shrink-0 cursor-pointer rounded-full border-2 border-transparent transition-colors duration-200 ease-in-out focus:outline-none disabled:opacity-50"
                  :class="c.state === 'running' ? 'bg-emerald-500' : 'bg-slate-200 dark:bg-slate-700'"
                  :title="c.state === 'running' ? '点击停止容器' : '点击启动容器'"
                >
                  <span
                    aria-hidden="true"
                    class="pointer-events-none inline-block h-5 w-5 transform rounded-full bg-white shadow-md ring-0 transition duration-200 ease-in-out flex items-center justify-center"
                    :class="c.state === 'running' ? 'translate-x-5' : 'translate-x-0'"
                  >
                    <Icon
                      v-if="operating[c.id]"
                      icon="tabler:loader-2"
                      class="w-3 h-3 text-slate-500 animate-spin"
                    />
                  </span>
                </button>

                <!-- Inspect / Restart / Log Button -->
                <button
                  type="button"
                  @click="handleOpenInspect(c)"
                  class="p-1.5 rounded-lg text-slate-500 dark:text-slate-400 hover:text-indigo-600 dark:hover:text-indigo-400 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors cursor-pointer"
                  title="查看容器详情与日志"
                >
                  <Icon icon="tabler:square-plus" class="w-4 h-4" />
                </button>
              </div>
            </td>
          </tr>

          <!-- Empty state -->
          <tr v-if="filteredAndSortedContainers.length === 0">
            <td colspan="4" class="py-8 text-center text-slate-400 text-xs">
              {{ searchQuery ? '未找到匹配的 Docker 容器' : '宿主机暂无容器' }}
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Container Inspect & Logs Modal -->
    <transition
      enter-active-class="transition duration-200 ease-out"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition duration-150 ease-in"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="inspectContainer"
        class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-xs"
        @click.self="inspectContainer = null"
      >
        <div class="w-full max-w-2xl bg-white dark:bg-slate-900 rounded-2xl shadow-2xl border border-slate-200 dark:border-slate-800 overflow-hidden flex flex-col max-h-[85vh]">
          <!-- Modal Header -->
          <div class="px-5 py-4 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between">
            <div class="flex items-center gap-2">
              <Icon icon="tabler:brand-docker" class="w-5 h-5 text-sky-500" />
              <h3 class="text-sm font-semibold text-slate-800 dark:text-slate-100 truncate">
                {{ inspectContainer.names[0] || inspectContainer.id }}
              </h3>
              <span
                class="px-2 py-0.5 rounded-full text-[10px] font-medium"
                :class="inspectContainer.state === 'running' ? 'bg-emerald-50 text-emerald-600 dark:bg-emerald-950/60 dark:text-emerald-400' : 'bg-slate-100 text-slate-500 dark:bg-slate-800'"
              >
                {{ inspectContainer.status }}
              </span>
            </div>

            <div class="flex items-center gap-2">
              <button
                type="button"
                @click="handleRestart(inspectContainer)"
                :disabled="!!operating[inspectContainer.id]"
                class="px-2.5 py-1 rounded-lg bg-amber-50 hover:bg-amber-100 dark:bg-amber-950/40 dark:hover:bg-amber-900/50 text-amber-700 dark:text-amber-300 text-xs font-medium flex items-center gap-1 transition-colors cursor-pointer"
                title="重启此容器"
              >
                <Icon icon="tabler:rotate-clockwise" class="w-3.5 h-3.5" :class="{ 'animate-spin': operating[inspectContainer.id] === 'restarting' }" />
                <span>重启容器</span>
              </button>

              <button
                type="button"
                @click="inspectContainer = null"
                class="p-1 text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 transition-colors rounded-lg"
              >
                <Icon icon="tabler:x" class="w-5 h-5" />
              </button>
            </div>
          </div>

          <!-- Modal Body: Logs & Info -->
          <div class="p-5 overflow-y-auto space-y-3.5 flex-1 text-xs">
            <!-- Metadata summary -->
            <div class="grid grid-cols-2 sm:grid-cols-3 gap-2.5 p-3 rounded-xl bg-slate-50 dark:bg-slate-800/50 text-[11px]">
              <div>
                <span class="text-slate-400 block">容器 ID</span>
                <span class="font-mono text-slate-700 dark:text-slate-200">{{ inspectContainer.id }}</span>
              </div>
              <div class="col-span-2 truncate">
                <span class="text-slate-400 block">镜像名称</span>
                <span class="font-mono text-slate-700 dark:text-slate-200 truncate">{{ inspectContainer.image }}</span>
              </div>
              <div v-if="inspectContainer.cpu_percent !== undefined">
                <span class="text-slate-400 block">当前 CPU</span>
                <span class="font-mono font-semibold text-emerald-600 dark:text-emerald-400">{{ inspectContainer.cpu_percent }}%</span>
              </div>
              <div v-if="inspectContainer.ram_usage_str">
                <span class="text-slate-400 block">内存占用</span>
                <span class="font-mono font-semibold text-sky-600 dark:text-sky-400">{{ inspectContainer.ram_usage_str }}</span>
              </div>
            </div>

            <!-- Terminal Logs -->
            <div class="space-y-1.5">
              <div class="flex items-center justify-between text-slate-600 dark:text-slate-400">
                <span class="font-semibold text-xs flex items-center gap-1.5">
                  <Icon icon="tabler:terminal-2" class="w-4 h-4 text-indigo-500" />
                  <span>控制台日志 (最新 150 行)</span>
                </span>
                <div class="flex items-center gap-2">
                  <button
                    type="button"
                    @click="copyLogs"
                    class="flex items-center gap-1 text-[11px] text-slate-500 hover:text-indigo-600 transition-colors"
                  >
                    <Icon :icon="copiedLog ? 'tabler:check' : 'tabler:copy'" class="w-3.5 h-3.5" :class="{ 'text-emerald-500': copiedLog }" />
                    <span>{{ copiedLog ? '已复制' : '复制日志' }}</span>
                  </button>
                  <button
                    type="button"
                    @click="handleOpenInspect(inspectContainer)"
                    :disabled="logsLoading"
                    class="flex items-center gap-1 text-[11px] text-slate-500 hover:text-indigo-600 transition-colors"
                  >
                    <Icon icon="tabler:refresh" class="w-3.5 h-3.5" :class="{ 'animate-spin': logsLoading }" />
                    <span>刷新日志</span>
                  </button>
                </div>
              </div>

              <div class="relative bg-slate-950 text-slate-200 rounded-xl p-3 font-mono text-[11px] h-64 overflow-y-auto leading-relaxed border border-slate-800 shadow-inner">
                <div v-if="logsLoading" class="flex items-center justify-center h-full text-slate-500 gap-2">
                  <Icon icon="tabler:loader-2" class="w-5 h-5 animate-spin text-indigo-400" />
                  <span>正在获取容器日志...</span>
                </div>
                <pre v-else class="whitespace-pre-wrap break-all">{{ inspectLogs }}</pre>
              </div>
            </div>
          </div>
        </div>
      </div>
    </transition>
  </div>
</template>
