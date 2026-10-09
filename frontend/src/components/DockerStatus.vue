<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useApi } from '@/composables/useApi'

const api = useApi()

interface DockerContainer {
  id: string
  names: string[]
  image: string
  state: string
  status: string
  created: number
}

const available = ref<boolean>(false)
const message = ref<string>('')
const containers = ref<DockerContainer[]>([])
const loading = ref<boolean>(false)

async function fetchContainers() {
  loading.value = true
  try {
    const resp = await api.get('/system/docker')
    available.value = resp.data.available
    message.value = resp.data.message || ''
    containers.value = resp.data.containers || []
  } catch (e: any) {
    available.value = false
    message.value = '请求 Docker API 失败'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  fetchContainers()
})
</script>

<template>
  <div class="space-y-4">
    <!-- Header with Refresh -->
    <div class="flex items-center justify-between">
      <div>
        <h4 class="text-sm font-semibold text-slate-800 dark:text-slate-200">Docker 容器监控</h4>
        <p class="text-xs text-slate-500">通过挂载 /var/run/docker.sock 实时读取宿主机容器清单与运行状态</p>
      </div>
      <button
        type="button"
        @click="fetchContainers"
        class="px-3 py-1.5 bg-slate-100 hover:bg-slate-200 dark:bg-slate-800 dark:hover:bg-slate-700 text-slate-700 dark:text-slate-300 text-xs font-medium rounded-xl transition-colors"
      >
        刷新
      </button>
    </div>

    <!-- Socket Unavailable Warning -->
    <div
      v-if="!available"
      class="p-4 rounded-2xl bg-amber-50 dark:bg-amber-950/30 border border-amber-200 dark:border-amber-900/40 text-xs text-amber-800 dark:text-amber-300 space-y-1.5"
    >
      <div class="font-semibold flex items-center gap-1.5">
        <span>⚠️</span>
        <span>{{ message || 'Docker Socket 未挂载' }}</span>
      </div>
      <p>若需在面板中查看容器状态，请在 docker-compose.yml 的 volumes 中加入：</p>
      <code class="block p-2 bg-amber-100/60 dark:bg-amber-900/60 rounded-lg font-mono text-[11px]">
        - /var/run/docker.sock:/var/run/docker.sock:ro
      </code>
    </div>

    <!-- Containers List -->
    <div v-else class="space-y-2">
      <div
        v-for="c in containers"
        :key="c.id"
        class="flex items-center justify-between p-3 rounded-xl bg-slate-50 dark:bg-slate-800/40 border border-slate-200/60 dark:border-slate-800/60 text-xs"
      >
        <div class="flex items-center gap-3">
          <span
            class="w-2.5 h-2.5 rounded-full"
            :class="c.state === 'running' ? 'bg-emerald-500' : 'bg-slate-400'"
          ></span>
          <div>
            <div class="font-semibold text-slate-800 dark:text-slate-200">
              {{ c.names.join(', ') || c.id }}
            </div>
            <div class="text-[11px] text-slate-500 font-mono">
              {{ c.image }}
            </div>
          </div>
        </div>

        <div class="text-right">
          <span
            class="px-2 py-0.5 rounded-md text-[10px] font-medium"
            :class="c.state === 'running' ? 'bg-emerald-50 dark:bg-emerald-950/50 text-emerald-600' : 'bg-slate-200 dark:bg-slate-800 text-slate-500'"
          >
            {{ c.status }}
          </span>
        </div>
      </div>
    </div>
  </div>
</template>
