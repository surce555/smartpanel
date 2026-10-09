<script setup lang="ts">
import { Icon } from '@iconify/vue'
import DockerStatus from './DockerStatus.vue'

defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()
</script>

<template>
  <transition
    enter-active-class="transition duration-200 ease-out"
    enter-from-class="opacity-0 scale-95"
    enter-to-class="opacity-100 scale-100"
    leave-active-class="transition duration-150 ease-in"
    leave-from-class="opacity-100 scale-100"
    leave-to-class="opacity-0 scale-95"
  >
    <div
      v-if="show"
      class="fixed inset-0 z-50 flex items-center justify-center p-3 sm:p-6 bg-black/60 backdrop-blur-xs overflow-y-auto"
      @click.self="emit('close')"
    >
      <div class="w-full max-w-4xl bg-white dark:bg-slate-900 rounded-3xl shadow-2xl border border-slate-200/80 dark:border-slate-800 overflow-hidden flex flex-col my-auto max-h-[90vh]">
        <!-- Top bar with close button -->
        <div class="px-5 py-3.5 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between bg-slate-50/50 dark:bg-slate-800/40">
          <div class="flex items-center gap-2.5">
            <div class="w-8 h-8 rounded-xl bg-sky-500/10 text-sky-500 flex items-center justify-center">
              <Icon icon="tabler:brand-docker" class="w-5 h-5" />
            </div>
            <div>
              <h2 class="text-sm font-bold text-slate-800 dark:text-slate-100">Docker 容器监控与管理</h2>
              <p class="text-[11px] text-slate-400">实时 CPU / 内存占用统计与容器一键启停</p>
            </div>
          </div>
          <button
            type="button"
            @click="emit('close')"
            class="p-1.5 rounded-xl text-slate-400 hover:text-slate-600 dark:hover:text-slate-200 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors cursor-pointer"
          >
            <Icon icon="tabler:x" class="w-5 h-5" />
          </button>
        </div>

        <div class="p-3 sm:p-5 overflow-y-auto flex-1">
          <DockerStatus />
        </div>
      </div>
    </div>
  </transition>
</template>
