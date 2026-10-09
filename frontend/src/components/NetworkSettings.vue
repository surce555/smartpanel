<script setup lang="ts">
import { ref, computed } from 'vue'
import { useSettingsStore } from '@/stores/settings'

const settingsStore = useSettingsStore()

const cfApiToken = ref<string>('')
const cfZoneId = ref<string>(settingsStore.networkInfo.cf_zone_id || '')
const cfRecordId = ref<string>(settingsStore.networkInfo.cf_record_id || '')
const domain = ref<string>(settingsStore.networkInfo.domain || '')
const v6domain = ref<string>(settingsStore.networkInfo.v6domain || '')
const ddnsEnabled = ref<boolean>(settingsStore.networkInfo.ddns_enabled || false)
const ddnsInterval = ref<number>(settingsStore.networkInfo.ddns_interval_minutes || 5)
const manualIPv6 = ref<string>('')

const isChecking = ref<boolean>(false)
const saveMsg = ref<string>('')

// Detected client/host IPv6
const browserHostIPv6 = computed(() => {
  if (settingsStore.networkInfo.detected_host_ipv6) {
    return settingsStore.networkInfo.detected_host_ipv6
  }
  let h = window.location.hostname || ''
  h = h.replace(/^\[|\]$/g, '')
  if (h.includes(':')) {
    return h
  }
  return ''
})

const ddnsStatusBadge = computed(() => {
  const s = settingsStore.networkInfo.ddns_status || 'idle'
  if (s.startsWith('success')) {
    return { text: '同步正常', color: 'bg-emerald-50 text-emerald-600 border-emerald-200 dark:bg-emerald-950/40 dark:text-emerald-400' }
  }
  if (s.startsWith('detect_failed') || s.startsWith('cf_error')) {
    return { text: '同步失败: ' + s, color: 'bg-rose-50 text-rose-600 border-rose-200 dark:bg-rose-950/40 dark:text-rose-400' }
  }
  if (s === 'ddns_disabled') {
    return { text: 'DDNS 未开启', color: 'bg-slate-100 text-slate-500 border-slate-200 dark:bg-slate-800' }
  }
  return { text: s, color: 'bg-amber-50 text-amber-600 border-amber-200 dark:bg-amber-950/40 dark:text-amber-400' }
})

function applyDetectedIPv6() {
  if (browserHostIPv6.value) {
    manualIPv6.value = browserHostIPv6.value
    handleManualCheck()
  }
}

async function handleSave() {
  saveMsg.value = ''
  try {
    const payload: Record<string, any> = {
      cf_zone_id: cfZoneId.value,
      cf_record_id: cfRecordId.value,
      domain: domain.value,
      v6domain: v6domain.value,
      ddns_enabled: ddnsEnabled.value ? 'true' : 'false',
      ddns_interval_minutes: String(ddnsInterval.value),
    }
    if (cfApiToken.value) {
      payload.cf_api_token = cfApiToken.value
    }
    if (manualIPv6.value.trim()) {
      payload.current_ipv6 = manualIPv6.value.trim()
    }
    await settingsStore.saveSettings(payload)
    saveMsg.value = '网络与 DDNS 设置保存成功！'
    cfApiToken.value = ''
  } catch (err: any) {
    saveMsg.value = '保存失败: ' + (err.response?.data?.error || err.message)
  }
}

async function handleManualCheck() {
  isChecking.value = true
  saveMsg.value = ''
  try {
    await settingsStore.triggerNetworkCheck(manualIPv6.value.trim())
    saveMsg.value = '已发送检测指令'
  } finally {
    setTimeout(() => {
      isChecking.value = false
    }, 1200)
  }
}
</script>

<template>
  <div class="space-y-6">
    <!-- Live Status Banner -->
    <div class="p-4 rounded-2xl bg-indigo-50/60 dark:bg-indigo-950/30 border border-indigo-200/60 dark:border-indigo-900/40 space-y-3">
      <div class="flex items-center justify-between">
        <div class="flex items-center gap-2">
          <span class="w-2.5 h-2.5 rounded-full bg-emerald-500 animate-ping"></span>
          <h4 class="text-sm font-bold text-slate-800 dark:text-slate-100">IPv6 双栈网络实时状态</h4>
        </div>
        <button
          type="button"
          @click="handleManualCheck"
          :disabled="isChecking"
          class="px-3.5 py-1.5 bg-indigo-600 hover:bg-indigo-700 text-white rounded-lg text-xs font-medium transition-colors shadow-xs"
        >
          {{ isChecking ? '检测中...' : '立即手动触发检测' }}
        </button>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-3 gap-3 text-xs">
        <div>
          <span class="text-slate-500 block mb-0.5">当前公网 IPv6:</span>
          <span class="font-mono font-semibold text-slate-800 dark:text-slate-200 break-all">
            {{ settingsStore.networkInfo.current_ipv6 || '暂无检测记录' }}
          </span>
        </div>
        <div>
          <span class="text-slate-500 block mb-0.5">上次检测时间:</span>
          <span class="font-mono text-slate-700 dark:text-slate-300">
            {{ settingsStore.networkInfo.last_ipv6_check || '从不' }}
          </span>
        </div>
        <div>
          <span class="text-slate-500 block mb-0.5">DDNS 运行状态:</span>
          <span
            class="inline-block px-2 py-0.5 rounded-md border text-[11px] font-medium"
            :class="ddnsStatusBadge.color"
          >
            {{ ddnsStatusBadge.text }}
          </span>
        </div>
      </div>

      <!-- Quick Auto-detect & Fill Prompt from Browser / Host connection -->
      <div
        v-if="browserHostIPv6"
        class="mt-2 pt-2 border-t border-indigo-200/50 dark:border-indigo-900/50 flex flex-wrap items-center justify-between gap-2 text-xs"
      >
        <div class="flex items-center gap-1.5 text-slate-600 dark:text-slate-400">
          <span>💡 识别到您当前正在通过 IPv6 访问本面板：</span>
          <code class="font-mono text-indigo-600 dark:text-indigo-400 font-bold bg-indigo-100/60 dark:bg-indigo-900/60 px-1.5 py-0.5 rounded">
            {{ browserHostIPv6 }}
          </code>
        </div>
        <button
          type="button"
          @click="applyDetectedIPv6"
          class="px-2.5 py-1 bg-emerald-600 hover:bg-emerald-700 text-white rounded-md text-[11px] font-medium shadow-xs transition-colors"
        >
          一键设为此 IPv6 并同步
        </button>
      </div>
    </div>

    <!-- Manual IPv6 Override (Optional) -->
    <div class="p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/40 border border-slate-200/60 dark:border-slate-800/60">
      <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">
        手动指定公网 IPv6 地址 (选填，留空则使用 6.ipw.cn 等国内极速探针自动探测)
      </label>
      <div class="flex gap-2">
        <input
          type="text"
          v-model="manualIPv6"
          :placeholder="settingsStore.networkInfo.current_ipv6 || '例如: 2408:8214:224a:...'"
          class="flex-1 px-3 py-2 text-xs font-mono bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700 focus:outline-none focus:ring-2 focus:ring-indigo-500"
        />
        <button
          type="button"
          @click="handleManualCheck"
          class="px-4 py-2 bg-slate-200 hover:bg-slate-300 dark:bg-slate-700 dark:hover:bg-slate-600 text-slate-800 dark:text-slate-100 text-xs font-medium rounded-xl transition-colors"
        >
          应用此地址
        </button>
      </div>
      <p class="text-[11px] text-slate-400 mt-1">
        提示：若 Docker 容器运行在 Bridge 默认桥接模式下导致外部 IPv6 请求受阻，推荐在 docker-compose.yml 配置 <code>network_mode: host</code>，或直接在此手动填入。
      </p>
    </div>

    <!-- Domain Settings -->
    <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
      <div>
        <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">主域名 (Cloudflare Tunnel 回源入口)</label>
        <input
          type="text"
          v-model="domain"
          placeholder="pan.yourdomain.com"
          class="w-full px-3 py-2 text-xs bg-slate-50 dark:bg-slate-800/80 rounded-xl border border-slate-200 dark:border-slate-800 focus:outline-none focus:ring-2 focus:ring-indigo-500"
        />
        <p class="text-[11px] text-slate-400 mt-1">对应模板变量 <code class="font-mono">{domain}</code></p>
      </div>

      <div>
        <label class="block text-xs font-semibold text-slate-700 dark:text-slate-300 mb-1">IPv6 直连子域名 (纯 DNS AAAA 记录)</label>
        <input
          type="text"
          v-model="v6domain"
          placeholder="v6.yourdomain.com"
          class="w-full px-3 py-2 text-xs bg-slate-50 dark:bg-slate-800/80 rounded-xl border border-slate-200 dark:border-slate-800 focus:outline-none focus:ring-2 focus:ring-indigo-500"
        />
        <p class="text-[11px] text-slate-400 mt-1">对应模板变量 <code class="font-mono">{v6domain}</code></p>
      </div>
    </div>

    <!-- Cloudflare DDNS Configuration -->
    <div class="p-4 rounded-2xl bg-slate-50 dark:bg-slate-800/40 border border-slate-200/60 dark:border-slate-800/60 space-y-4">
      <div class="flex items-center justify-between">
        <div>
          <h4 class="text-sm font-semibold text-slate-800 dark:text-slate-200">Cloudflare 自动 DDNS 同步</h4>
          <p class="text-xs text-slate-500">当检测到家庭 IPv6 前缀变化时，自动调用 API 修改 AAAA 记录并推送通知</p>
        </div>
        <label class="relative inline-flex items-center cursor-pointer">
          <input type="checkbox" v-model="ddnsEnabled" class="sr-only peer" />
          <div class="w-11 h-6 bg-slate-300 peer-focus:outline-none rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-slate-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-indigo-600"></div>
        </label>
      </div>

      <div class="grid grid-cols-1 sm:grid-cols-3 gap-3">
        <div>
          <label class="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">Cloudflare API Token</label>
          <input
            type="password"
            v-model="cfApiToken"
            :placeholder="settingsStore.networkInfo.has_api_token ? '已加密配置 (留空保持不变)' : '填入具备 DNS 编辑权限的 Token'"
            class="w-full px-3 py-2 text-xs bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700"
          />
        </div>

        <div>
          <label class="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">Zone ID (区域 ID)</label>
          <input
            type="text"
            v-model="cfZoneId"
            placeholder="Cloudflare 仪表盘右侧 Zone ID"
            class="w-full px-3 py-2 text-xs bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700"
          />
        </div>

        <div>
          <label class="block text-xs font-medium text-slate-600 dark:text-slate-400 mb-1">Record ID (AAAA 记录 ID)</label>
          <input
            type="text"
            v-model="cfRecordId"
            placeholder="对应子域名的 DNS 记录 ID"
            class="w-full px-3 py-2 text-xs bg-white dark:bg-slate-900 rounded-xl border border-slate-200 dark:border-slate-700"
          />
        </div>
      </div>

      <div class="flex items-center gap-3">
        <label class="text-xs text-slate-600 dark:text-slate-400">定时检测间隔 (分钟):</label>
        <input
          type="number"
          min="1"
          max="60"
          v-model.number="ddnsInterval"
          class="w-20 px-2 py-1 text-xs bg-white dark:bg-slate-900 rounded-lg border border-slate-200 dark:border-slate-700 text-center"
        />
      </div>
    </div>

    <!-- Action Buttons -->
    <div class="flex items-center justify-between pt-2">
      <span v-if="saveMsg" class="text-xs text-emerald-600 font-medium">{{ saveMsg }}</span>
      <span v-else></span>
      <button
        type="button"
        @click="handleSave"
        class="px-5 py-2.5 bg-indigo-600 hover:bg-indigo-700 text-white rounded-xl text-xs font-semibold shadow-md transition-colors"
      >
        保存配置
      </button>
    </div>
  </div>
</template>
