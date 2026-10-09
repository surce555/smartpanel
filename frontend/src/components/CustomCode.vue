<script setup lang="ts">
import { useSettingsStore } from '@/stores/settings'
import { useTheme } from '@/composables/useTheme'

const settingsStore = useSettingsStore()
const { injectCustomCSS, injectCustomJS } = useTheme()

function handleCSSChange() {
  injectCustomCSS(settingsStore.customCss)
}

function handleJSChange() {
  injectCustomJS(settingsStore.customJs)
}
</script>

<template>
  <div class="space-y-6">
    <!-- Custom CSS -->
    <div>
      <div class="flex items-center justify-between mb-2">
        <div>
          <label class="text-sm font-semibold text-slate-800 dark:text-slate-200">自定义 CSS 样式注入</label>
          <p class="text-xs text-slate-500">动态注入页面 &lt;style&gt; 标签，可自由定制任何选择器与视觉动效</p>
        </div>
      </div>
      <textarea
        v-model="settingsStore.customCss"
        @input="handleCSSChange"
        rows="6"
        placeholder="/* 示例：覆盖卡片背景或阴影 */&#10;.card-glass { backdrop-filter: blur(24px) !important; }"
        class="w-full p-3 font-mono text-xs bg-slate-900 text-slate-100 rounded-xl border border-slate-700 focus:outline-none focus:ring-2 focus:ring-indigo-500"
      ></textarea>
    </div>

    <!-- Custom JS -->
    <div>
      <div class="flex items-center justify-between mb-2">
        <div>
          <label class="text-sm font-semibold text-slate-800 dark:text-slate-200">自定义 JavaScript 脚本执行</label>
          <p class="text-xs text-slate-500">页面载入后安全执行，可用于接入个性化统计代码或额外动态功能</p>
        </div>
      </div>
      <textarea
        v-model="settingsStore.customJs"
        @input="handleJSChange"
        rows="6"
        placeholder="// 示例：打印欢迎日志&#10;console.log('SmartPanel loaded smoothly!');"
        class="w-full p-3 font-mono text-xs bg-slate-900 text-slate-100 rounded-xl border border-slate-700 focus:outline-none focus:ring-2 focus:ring-indigo-500"
      ></textarea>
    </div>
  </div>
</template>
