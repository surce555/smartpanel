<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useApi } from '@/composables/useApi'

const api = useApi()

interface SystemStatus {
  cpu_usage_percent: number
  memory_total_bytes: number
  memory_used_bytes: number
  memory_usage_percent: number
  disk_total_bytes: number
  disk_used_bytes: number
  disk_usage_percent: number
  host_os: string
  host_arch: string
  uptime_seconds: number
}

const status = ref<SystemStatus | null>(null)
const loading = ref<boolean>(false)
let timer: number | null = null

function formatBytes(bytes: number): string {
  if (!bytes || bytes === 0) return '0 B'
  const k = 1024
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(k))
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i]
}

function formatUptime(seconds: number): string {
  if (!seconds) return '0 分钟'
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  let res = ''
  if (days > 0) res += `${days} 天 `
  if (hours > 0) res += `${hours} 小时 `
  res += `${minutes} 分钟`
  return res
}

async function fetchStatus() {
  try {
    const resp = await api.get('/system/status')
    status.value = resp.data
  } catch (e) {
    console.error('Failed to load system status:', e)
  }
}

onMounted(() => {
  fetchStatus()
  timer = window.setInterval(fetchStatus, 5000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <div class="space-y-6">
    <div v-if="status" class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <!-- CPU -->
      <div class="p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/40 border border-slate-200/60 dark:border-slate-800/60">
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs font-medium text-slate-500">CPU 使用率</span>
          <span class="text-sm font-bold font-mono text-indigo-600 dark:text-indigo-400">
            {{ status.cpu_usage_percent.toFixed(1) }}%
          </span>
        </div>
        <div class="w-full h-2 rounded-full bg-slate-200 dark:bg-slate-700 overflow-hidden">
          <div
            class="h-full bg-indigo-500 transition-all duration-500 rounded-full"
            :style="{ width: `${Math.min(status.cpu_usage_percent, 100)}%` }"
          ></div>
        </div>
      </div>

      <!-- Memory -->
      <div class="p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/40 border border-slate-200/60 dark:border-slate-800/60">
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs font-medium text-slate-500">内存占用</span>
          <span class="text-sm font-bold font-mono text-indigo-600 dark:text-indigo-400">
            {{ formatBytes(status.memory_used_bytes) }} / {{ formatBytes(status.memory_total_bytes) }}
          </span>
        </div>
        <div class="w-full h-2 rounded-full bg-slate-200 dark:bg-slate-700 overflow-hidden">
          <div
            class="h-full bg-indigo-500 transition-all duration-500 rounded-full"
            :style="{ width: `${Math.min(status.memory_usage_percent, 100)}%` }"
          ></div>
        </div>
      </div>

      <!-- Disk -->
      <div class="p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/40 border border-slate-200/60 dark:border-slate-800/60">
        <div class="flex items-center justify-between mb-2">
          <span class="text-xs font-medium text-slate-500">磁盘存储</span>
          <span class="text-sm font-bold font-mono text-indigo-600 dark:text-indigo-400">
            {{ formatBytes(status.disk_used_bytes) }} / {{ formatBytes(status.disk_total_bytes) }}
          </span>
        </div>
        <div class="w-full h-2 rounded-full bg-slate-200 dark:bg-slate-700 overflow-hidden">
          <div
            class="h-full bg-indigo-500 transition-all duration-500 rounded-full"
            :style="{ width: `${Math.min(status.disk_usage_percent, 100)}%` }"
          ></div>
        </div>
      </div>
    </div>

    <!-- Host info summary -->
    <div v-if="status" class="flex flex-wrap items-center gap-4 text-xs text-slate-500 px-1">
      <div>架构: <span class="font-mono text-slate-700 dark:text-slate-300">{{ status.host_os }} / {{ status.host_arch }}</span></div>
      <div>运行时间: <span class="font-mono text-slate-700 dark:text-slate-300">{{ formatUptime(status.uptime_seconds) }}</span></div>
    </div>
  </div>
</template>
