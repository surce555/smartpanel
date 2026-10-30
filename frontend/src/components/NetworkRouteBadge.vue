<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Icon } from '@iconify/vue'
import { useIPv6Probe, isLanEntry, isIPv6Host, isIpInWhitelist } from '@/composables/useIPv6Probe'
import { useSettingsStore } from '@/stores/settings'

const {
  isIPv6Available,
  isLanAvailable,
  isProbing,
  lastProbeDetail,
  routePreference,
  setRoutePreference,
  probeIPv6,
  clearProbeCache,
} = useIPv6Probe()

const settingsStore = useSettingsStore()
const showMenu = ref(false)
const isWhitelisting = ref(false)
const whitelistSuccess = ref(false)

const isClientInLan = computed(() => {
  const net = settingsStore.networkInfo
  const hostname = window.location.hostname
  return (
    isLanEntry(hostname, net.lan_domain) ||
    net.is_client_lan === true ||
    isIpInWhitelist(net.client_ip, net.lan_ip_whitelist) ||
    isLanAvailable.value === true
  )
})

const currentActiveMode = computed(() => {
  if (routePreference.value !== 'auto') {
    return routePreference.value
  }

  // 1. LAN Environment
  if (isClientInLan.value) {
    return 'lan'
  }

  const hostname = window.location.hostname

  // 2. Direct IPv6 Host
  if (isIPv6Host(hostname)) {
    return 'ipv6'
  }

  // 3. IPv6 confirmed available
  if (isIPv6Available.value === true) {
    return 'ipv6'
  }

  // 4. IPv6 confirmed not available -> Domain
  if (isIPv6Available.value === false) {
    return 'domain'
  }

  // 5. While probing or undetermined
  if (isProbing.value || isIPv6Available.value === null) {
    return 'probing'
  }

  return 'domain'
})

const badgeConfig = computed(() => {
  switch (currentActiveMode.value) {
    case 'lan':
      return {
        label: routePreference.value === 'auto' ? '内网直连' : '强制内网',
        dotColor: 'bg-emerald-500',
        textColor: 'text-emerald-700 dark:text-emerald-300',
        bgColor: 'bg-emerald-100/80 dark:bg-emerald-950/60',
        borderColor: 'border-emerald-200 dark:border-emerald-800',
        icon: 'tabler:home',
        desc: '当前走局域网内网直连通道',
      }
    case 'ipv6':
      return {
        label: routePreference.value === 'auto' ? 'IPv6 直连' : '强制 IPv6',
        dotColor: 'bg-blue-500',
        textColor: 'text-blue-700 dark:text-blue-300',
        bgColor: 'bg-blue-100/80 dark:bg-blue-950/60',
        borderColor: 'border-blue-200 dark:border-blue-800',
        icon: 'tabler:bolt',
        desc: '当前通过公网 IPv6 高速直连 NAS',
      }
    case 'domain':
      return {
        label: routePreference.value === 'auto' ? 'CF 代理' : '强制域名',
        dotColor: 'bg-amber-500',
        textColor: 'text-amber-700 dark:text-amber-300',
        bgColor: 'bg-amber-100/80 dark:bg-amber-950/60',
        borderColor: 'border-amber-200 dark:border-amber-800',
        icon: 'tabler:cloud',
        desc: '当前走 Cloudflare 代理回退回源通道',
      }
    case 'probing':
    default:
      return {
        label: '网络检测中...',
        dotColor: 'bg-slate-400 animate-pulse',
        textColor: 'text-slate-500 dark:text-slate-400',
        bgColor: 'bg-slate-100 dark:bg-slate-800',
        borderColor: 'border-slate-200 dark:border-slate-700',
        icon: 'tabler:loader-2',
        desc: '正在检测网络环境与 IPv6 连通性',
      }
  }
})

const options = [
  {
    id: 'auto',
    name: '⚡ 智能分流 (默认)',
    desc: '按 内网 ➔ IPv6 ➔ 域名 优先级自动匹配最佳路由',
  },
  {
    id: 'lan',
    name: '🏠 强制内网直连',
    desc: '所有书签优先使用局域网地址 (如 192.168.x.x)',
  },
  {
    id: 'ipv6',
    name: '🚀 强制 IPv6 直连',
    desc: '所有书签优先使用公网 IPv6 动态模板直连',
  },
  {
    id: 'domain',
    name: '🌐 强制域名代理',
    desc: '所有书签优先使用 Cloudflare Tunnel 回退域名',
  },
]

function selectMode(mode: 'auto' | 'lan' | 'ipv6' | 'domain') {
  setRoutePreference(mode)
  showMenu.value = false
}

async function handleReProbe() {
  clearProbeCache()
  const net = settingsStore.networkInfo
  await probeIPv6(undefined, 2500, {
    clientIP: net.client_ip,
    directIPv6: net.direct_ipv6,
    clientHasIPv6: net.client_has_ipv6,
    viaCloudflare: net.via_cloudflare,
    v6domain: net.v6domain,
    force: true,
  })
}

async function handleQuickWhitelist() {
  const net = settingsStore.networkInfo
  if (!net.client_ip) return
  const ip = net.client_ip.trim()
  const existing = (net.lan_ip_whitelist || '').trim()
  const updated = existing ? `${existing}, ${ip}` : ip
  isWhitelisting.value = true
  try {
    await settingsStore.saveSettings({ lan_ip_whitelist: updated })
    settingsStore.networkInfo.lan_ip_whitelist = updated
    settingsStore.networkInfo.is_client_lan = true
    settingsStore.networkInfo.lan_reason = `访客 IP (${ip}) 已加入家庭白名单`
    whitelistSuccess.value = true
    setTimeout(() => {
      whitelistSuccess.value = false
    }, 3000)
  } catch (e: any) {
    console.error('Failed to whitelist IP:', e)
  } finally {
    isWhitelisting.value = false
  }
}

function handleOutsideClick(e: MouseEvent) {
  const target = e.target as HTMLElement
  if (!target.closest('.network-route-container')) {
    showMenu.value = false
  }
}

onMounted(() => {
  window.addEventListener('click', handleOutsideClick)
})

onUnmounted(() => {
  window.removeEventListener('click', handleOutsideClick)
})
</script>

<template>
  <div class="relative network-route-container inline-block">
    <!-- Clickable Badge Pill -->
    <button
      type="button"
      @click.stop="showMenu = !showMenu"
      :class="[
        'inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full text-[11px] font-semibold border shadow-2xs transition-all hover:scale-105 select-none cursor-pointer',
        badgeConfig.bgColor,
        badgeConfig.textColor,
        badgeConfig.borderColor
      ]"
      :title="`${badgeConfig.desc} (点击可自由切换网络分流优先级)`"
    >
      <span class="w-1.5 h-1.5 rounded-full" :class="badgeConfig.dotColor"></span>
      <span>{{ badgeConfig.label }}</span>
      <Icon icon="tabler:chevron-down" class="w-3 h-3 opacity-60" />
    </button>

    <!-- Dropdown Menu -->
    <div
      v-if="showMenu"
      class="absolute left-0 mt-2 w-72 p-2 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xl z-50 text-left space-y-1 animate-in fade-in zoom-in-95 duration-150"
    >
      <div class="px-2.5 py-1.5 border-b border-slate-100 dark:border-slate-800 flex items-center justify-between">
        <div>
          <div class="text-[11px] font-bold text-slate-700 dark:text-slate-300">网络智能分流</div>
          <div class="text-[10px] text-slate-400">点击书签卡片时优先匹配的连接路线</div>
        </div>
        <button
          type="button"
          @click.stop="handleReProbe"
          class="p-1 rounded-md text-slate-400 hover:text-indigo-600 hover:bg-slate-100 dark:hover:bg-slate-800 transition-colors"
          title="重新检测网络连通性"
        >
          <Icon icon="tabler:refresh" class="w-3.5 h-3.5" :class="{ 'animate-spin': isProbing }" />
        </button>
      </div>

      <button
        v-for="opt in options"
        :key="opt.id"
        type="button"
        @click="selectMode(opt.id as any)"
        class="w-full px-2.5 py-2 rounded-xl text-xs text-left transition-colors flex flex-col gap-0.5"
        :class="[
          routePreference === opt.id
            ? 'bg-indigo-50 dark:bg-indigo-950/60 text-indigo-600 dark:text-indigo-400 font-semibold'
            : 'hover:bg-slate-50 dark:hover:bg-slate-800/80 text-slate-700 dark:text-slate-300'
        ]"
      >
        <div class="flex items-center justify-between">
          <span>{{ opt.name }}</span>
          <Icon v-if="routePreference === opt.id" icon="tabler:check" class="w-4 h-4 text-indigo-600" />
        </div>
        <span class="text-[10px] text-slate-400 font-normal leading-tight">{{ opt.desc }}</span>
      </button>

      <!-- Diagnostic Status Footer -->
      <div v-if="lastProbeDetail || settingsStore.networkInfo.client_ip" class="pt-1 mt-1 border-t border-slate-100 dark:border-slate-800/60 px-2 py-1 text-[10px] text-slate-400 space-y-1 leading-relaxed">
        <div v-if="settingsStore.networkInfo.client_ip" class="truncate flex items-center justify-between">
          <span>访客 IP: <span class="font-mono text-slate-600 dark:text-slate-300">{{ settingsStore.networkInfo.client_ip }}</span></span>
        </div>

        <div v-if="settingsStore.networkInfo.lan_reason" class="text-emerald-600 dark:text-emerald-400 truncate flex items-center gap-1 font-medium">
          <Icon icon="tabler:shield-check" class="w-3.5 h-3.5 shrink-0" />
          <span>{{ settingsStore.networkInfo.lan_reason }}</span>
        </div>

        <!-- Quick 1-click button if user is in home network but not auto-detected -->
        <div v-if="!isClientInLan && settingsStore.networkInfo.client_ip" class="pt-0.5">
          <button
            type="button"
            @click.stop="handleQuickWhitelist"
            :disabled="isWhitelisting"
            class="w-full py-1 px-2 text-[10px] bg-emerald-50 hover:bg-emerald-100 text-emerald-700 dark:bg-emerald-950/60 dark:hover:bg-emerald-900/80 dark:text-emerald-300 rounded-lg border border-emerald-200 dark:border-emerald-800 flex items-center justify-center gap-1 transition-all cursor-pointer font-medium"
            title="将当前访客 IP 记录为内网，之后无论通过任何域名访问都自动直连"
          >
            <Icon icon="tabler:home-plus" class="w-3.5 h-3.5 shrink-0" />
            <span>{{ isWhitelisting ? '正在配置...' : whitelistSuccess ? '已生效为家庭内网！' : '当前是家庭网络？一键设为内网' }}</span>
          </button>
        </div>

        <div v-if="lastProbeDetail" class="text-slate-500 dark:text-slate-400">
          {{ lastProbeDetail }}
        </div>
      </div>
    </div>
  </div>
</template>
