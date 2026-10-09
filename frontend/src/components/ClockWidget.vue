<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

const timeStr = ref<string>('')
const dateStr = ref<string>('')
const lunarStr = ref<string>('')
let timer: number | null = null

const weekDays = ['星期日', '星期一', '星期二', '星期三', '星期四', '星期五', '星期六']

function getLunarDate(d: Date): string {
  try {
    const formatter = new Intl.DateTimeFormat('zh-Hans-u-ca-chinese', {
      month: 'long',
      day: 'numeric'
    })
    const parts = formatter.formatToParts(d)
    let month = ''
    let day = ''
    for (const p of parts) {
      if (p.type === 'month') month = p.value
      if (p.type === 'day') day = p.value
    }
    if (month && day) {
      return `农历 ${month}${day}日`
    }
    return `农历 ${formatter.format(d)}`
  } catch {
    return ''
  }
}

function updateClock() {
  const now = new Date()
  const h = String(now.getHours()).padStart(2, '0')
  const m = String(now.getMinutes()).padStart(2, '0')
  const s = String(now.getSeconds()).padStart(2, '0')
  timeStr.value = `${h}:${m}:${s}`

  const year = now.getFullYear()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  const date = String(now.getDate()).padStart(2, '0')
  const weekday = weekDays[now.getDay()]
  dateStr.value = `${year}年${month}月${date}日 ${weekday}`

  lunarStr.value = getLunarDate(now)
}

onMounted(() => {
  updateClock()
  timer = window.setInterval(updateClock, 1000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<template>
  <div
    class="flex flex-col items-end justify-center px-4 py-2 rounded-2xl bg-black/40 hover:bg-black/55 backdrop-blur-md border border-white/10 shadow-lg text-right select-none transition-all duration-200"
    title="当前系统时间与农历日期"
  >
    <div class="text-xl sm:text-2xl font-bold font-mono tracking-wider text-white drop-shadow-sm">
      {{ timeStr }}
    </div>
    <div class="text-[11px] sm:text-xs text-white/80 font-normal tracking-tight mt-0.5">
      <span>{{ dateStr }}</span>
      <span v-if="lunarStr" class="ml-1.5 opacity-90">| {{ lunarStr }}</span>
    </div>
  </div>
</template>
