<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  status?: 'online' | 'offline' | 'unknown'
  responseTime?: number
}>()

const badgeClass = computed(() => {
  switch (props.status) {
    case 'online':
      return 'bg-emerald-500 shadow-emerald-500/50'
    case 'offline':
      return 'bg-rose-500 shadow-rose-500/50'
    default:
      return 'bg-slate-400 shadow-slate-400/50'
  }
})

const tooltipText = computed(() => {
  if (props.status === 'online') {
    return `在线 (${props.responseTime || 0}ms)`
  }
  if (props.status === 'offline') {
    return '服务离线或超时'
  }
  return '未检测'
})
</script>

<template>
  <div class="relative group/badge flex items-center" :title="tooltipText">
    <span
      class="w-2.5 h-2.5 rounded-full shadow-sm transition-all duration-300"
      :class="[badgeClass, status === 'online' ? 'animate-pulse' : '']"
    ></span>
  </div>
</template>
