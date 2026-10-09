<script setup lang="ts">
import { useSettingsStore } from '@/stores/settings'

const settingsStore = useSettingsStore()

const styles = [
  { id: 'glass', name: '玻璃拟态 (Glass)', desc: '高斯模糊毛玻璃，透光微边框' },
  { id: 'solid', name: '纯色实体 (Solid)', desc: '规整卡片底色，高对比度' },
  { id: 'transparent', name: '轻透磨砂 (Transparent)', desc: '微透轻盈，契合壁纸' },
  { id: 'minimal', name: '极简悬浮 (Minimal)', desc: '无底色边框，纯图标与文本' },
]

const shadows = [
  { id: 'none', name: '无阴影' },
  { id: 'sm', name: '轻微' },
  { id: 'md', name: '中等' },
  { id: 'lg', name: '深邃' },
]

const iconSizes = [
  { id: 'small', name: '紧凑 (小)' },
  { id: 'medium', name: '标准 (中)' },
  { id: 'large', name: '显眼 (大)' },
  { id: 'detail', name: '特大 (详情)' },
]
</script>

<template>
  <div class="space-y-6">
    <!-- Card Style Selector -->
    <div>
      <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">卡片风格预设</label>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <button
          type="button"
          v-for="s in styles"
          :key="s.id"
          @click="settingsStore.cardStyle = s.id"
          class="p-3 text-left rounded-xl border transition-all"
          :class="[
            settingsStore.cardStyle === s.id
              ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400 font-semibold'
              : 'border-slate-200 dark:border-slate-800 hover:bg-slate-50 dark:hover:bg-slate-800/50 text-slate-700 dark:text-slate-300'
          ]"
        >
          <div class="font-medium text-sm">{{ s.name }}</div>
          <div class="text-xs text-slate-400 mt-0.5">{{ s.desc }}</div>
        </button>
      </div>
    </div>

    <!-- Sliders: Border radius -->
    <div>
      <div class="flex justify-between text-sm mb-1.5">
        <span class="text-slate-700 dark:text-slate-300 font-medium">卡片圆角</span>
        <span class="text-slate-500 font-mono text-xs">{{ settingsStore.cardBorderRadius }}px</span>
      </div>
      <input
        type="range"
        min="0"
        max="24"
        step="2"
        v-model.number="settingsStore.cardBorderRadius"
        class="w-full accent-indigo-600"
      />
    </div>

    <!-- Shadow & Icon Size -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-6">
      <div>
        <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">阴影强度</label>
        <div class="grid grid-cols-4 gap-2">
          <button
            type="button"
            v-for="sh in shadows"
            :key="sh.id"
            @click="settingsStore.cardShadow = sh.id"
            class="p-2 text-center rounded-xl border text-xs transition-all"
            :class="[
              settingsStore.cardShadow === sh.id
                ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400 font-semibold'
                : 'border-slate-200 dark:border-slate-800 text-slate-700 dark:text-slate-300'
            ]"
          >
            {{ sh.name }}
          </button>
        </div>
      </div>

      <div>
        <label class="block text-sm font-medium text-slate-700 dark:text-slate-300 mb-2">图标尺寸</label>
        <div class="grid grid-cols-4 gap-2">
          <button
            type="button"
            v-for="is in iconSizes"
            :key="is.id"
            @click="settingsStore.iconSize = is.id"
            class="p-2 text-center rounded-xl border text-xs transition-all"
            :class="[
              settingsStore.iconSize === is.id
                ? 'border-indigo-500 bg-indigo-50 dark:bg-indigo-950/40 text-indigo-600 dark:text-indigo-400 font-semibold'
                : 'border-slate-200 dark:border-slate-800 text-slate-700 dark:text-slate-300'
            ]"
          >
            {{ is.name }}
          </button>
        </div>
      </div>
    </div>

    <!-- Grid Columns -->
    <div class="pt-2 border-t border-slate-200/60 dark:border-slate-800/60">
      <h4 class="text-sm font-semibold text-slate-800 dark:text-slate-200 mb-3">栅格列数适配 (自适应排版)</h4>
      <div class="grid grid-cols-3 gap-4">
        <div>
          <label class="block text-xs text-slate-500 mb-1">桌面端 (列)</label>
          <select
            v-model.number="settingsStore.gridColsDesktop"
            class="w-full px-3 py-2 bg-slate-50 dark:bg-slate-800/80 rounded-xl border border-slate-200 dark:border-slate-800 text-xs font-medium"
          >
            <option :value="4">4 列</option>
            <option :value="5">5 列</option>
            <option :value="6">6 列 (推荐)</option>
            <option :value="7">7 列</option>
            <option :value="8">8 列 (超宽)</option>
          </select>
        </div>

        <div>
          <label class="block text-xs text-slate-500 mb-1">平板端 (列)</label>
          <select
            v-model.number="settingsStore.gridColsTablet"
            class="w-full px-3 py-2 bg-slate-50 dark:bg-slate-800/80 rounded-xl border border-slate-200 dark:border-slate-800 text-xs font-medium"
          >
            <option :value="2">2 列</option>
            <option :value="3">3 列 (推荐)</option>
            <option :value="4">4 列</option>
          </select>
        </div>

        <div>
          <label class="block text-xs text-slate-500 mb-1">移动端 (列)</label>
          <select
            v-model.number="settingsStore.gridColsMobile"
            class="w-full px-3 py-2 bg-slate-50 dark:bg-slate-800/80 rounded-xl border border-slate-200 dark:border-slate-800 text-xs font-medium"
          >
            <option :value="1">1 列</option>
            <option :value="2">2 列 (推荐)</option>
          </select>
        </div>
      </div>
    </div>
  </div>
</template>
