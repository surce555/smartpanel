<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Icon } from '@iconify/vue'

interface WeatherData {
  city: string
  temp: string
  condition: string
  wind: string
}

const defaultWeather: WeatherData = {
  city: '东营区',
  temp: '26℃',
  condition: '晴',
  wind: '西风 3级'
}

const weather = ref<WeatherData>({ ...defaultWeather })
const isEditing = ref(false)
const inputCity = ref('')
const loading = ref(false)

function loadCachedWeather() {
  const cached = localStorage.getItem('smartpanel_weather_cache')
  if (cached) {
    try {
      weather.value = JSON.parse(cached)
    } catch {
      // ignore
    }
  }
}

async function fetchWeather(cityName?: string) {
  loading.value = true
  const query = cityName || weather.value.city || ''
  try {
    // Attempt free wttr.in query
    const res = await fetch(`https://wttr.in/${encodeURIComponent(query)}?format=j1`, {
      signal: AbortSignal.timeout(3000)
    })
    if (res.ok) {
      const data = await res.json()
      const current = data.current_condition?.[0]
      const area = data.nearest_area?.[0]?.areaName?.[0]?.value || query
      if (current) {
        weather.value = {
          city: cityName || area || weather.value.city,
          temp: `${current.temp_C}℃`,
          condition: current.lang_zh?.[0]?.value || current.weatherDesc?.[0]?.value || '晴',
          wind: `${current.winddir16Point || '微风'} ${current.windspeedKmph || 10}km/h`
        }
        localStorage.setItem('smartpanel_weather_cache', JSON.stringify(weather.value))
      }
    }
  } catch {
    // If external query fails or times out, keep default or cached weather
    if (!localStorage.getItem('smartpanel_weather_cache')) {
      weather.value = { ...defaultWeather }
      localStorage.setItem('smartpanel_weather_cache', JSON.stringify(weather.value))
    }
  } finally {
    loading.value = false
  }
}

function handleSaveCity() {
  if (inputCity.value.trim()) {
    weather.value.city = inputCity.value.trim()
    fetchWeather(inputCity.value.trim())
  }
  isEditing.value = false
}

onMounted(() => {
  loadCachedWeather()
  // Fetch fresh weather in background
  fetchWeather()
})
</script>

<template>
  <div class="relative inline-block select-none">
    <!-- Weather Pill (Matching Image 2) -->
    <div
      @click="isEditing = !isEditing"
      class="flex flex-col items-start justify-center px-4 py-2 rounded-2xl bg-black/40 hover:bg-black/55 backdrop-blur-md border border-white/10 shadow-lg text-left text-white cursor-pointer transition-all duration-200"
      title="点击可修改城市或刷新天气"
    >
      <div class="flex items-center gap-2">
        <span class="text-sm sm:text-base font-bold text-white drop-shadow-sm tracking-wide">
          {{ weather.city }} {{ weather.temp }}
        </span>
        <Icon
          v-if="loading"
          icon="tabler:loader"
          class="w-3.5 h-3.5 text-white/60 animate-spin"
        />
      </div>
      <div class="text-[11px] sm:text-xs text-white/80 font-normal tracking-tight mt-0.5">
        <span>{{ weather.condition }}</span>
        <span class="mx-1 opacity-60">|</span>
        <span>{{ weather.wind }}</span>
      </div>
    </div>

    <!-- Quick City Input Popover -->
    <div
      v-if="isEditing"
      class="absolute bottom-full left-0 mb-2 w-56 p-3 rounded-2xl bg-slate-900/90 backdrop-blur-xl border border-white/20 shadow-2xl text-xs space-y-2 z-50 animate-in fade-in zoom-in-95"
      @click.stop
    >
      <div class="flex items-center justify-between text-white font-semibold">
        <span>设定天气城市</span>
        <button @click="isEditing = false" class="text-white/60 hover:text-white">✕</button>
      </div>
      <input
        v-model="inputCity"
        :placeholder="weather.city"
        @keyup.enter="handleSaveCity"
        class="w-full px-3 py-1.5 rounded-xl bg-white/10 border border-white/20 text-white placeholder-white/40 focus:outline-none focus:border-white/50 text-xs"
      />
      <div class="flex justify-end gap-2 pt-1">
        <button
          type="button"
          @click="fetchWeather()"
          class="px-2.5 py-1 rounded-lg bg-white/10 hover:bg-white/20 text-white text-[11px]"
        >
          刷新
        </button>
        <button
          type="button"
          @click="handleSaveCity"
          class="px-2.5 py-1 rounded-lg bg-indigo-600 hover:bg-indigo-500 text-white text-[11px] font-medium"
        >
          确定
        </button>
      </div>
    </div>
  </div>
</template>
