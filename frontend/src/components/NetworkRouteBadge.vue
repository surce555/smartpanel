<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { Icon } from '@iconify/vue'
import { useIPv6Probe, isIPv6Host, isLanEntry } from '@/composables/useIPv6Probe'
import { useSettingsStore } from '@/stores/settings'

const { isIPv6Available, lastProbeDetail, routePreference, setRoutePreference } = useIPv6Probe()
const settingsStore = useSettingsStore()

const showMenu = ref(false)

const currentActiveMode = computed(() => {
  if (routePreference.value !== 'auto') {
    return routePreference.value
  }

  // Auto evaluation — must mirror `resolveBookmarkUrl` exactly, otherwise the
  // badge would claim one mode while the cards actually open another.
  const hostname = window.location.hostname
  const net = settingsStore.networkInfo

  // Same LAN rule as the resolver: access origin OR the backend's view of the
  // visitor address.
  if (isLanEntry(hostname, net.lan_domain) || net.is_client_lan === true) {
    return 'lan'
  }
  if (isIPv6Host(hostname)) {
    return 'ipv6'
  }
  // Mirrors the resolver's veto semantics: IPv6 unless explicitly disproved.
  return isIPv6Available.value !== false ? 'ipv6' : 'domain'
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
        desc: '当前优先走局域网内网 IP 直连',
      }
    case 'ipv6':
      return {
        label: routePreference.value === 'auto' ? 'IPv6 直连' : '强制 IPv6',
        dotColor: 'bg-emerald-500',
        textColor: 'text-emerald-700 dark:text-emerald-300',
        bgColor: 'bg-emerald-100/80 dark:bg-emerald-950/60',
        borderColor: 'border-emerald-200 dark:border-emerald-800',
        icon: 'tabler:bolt',
        desc: '当前通过公网 IPv6 高速直连 NAS',
      }
    case 'domain':
      return {
        label: routePreference.value === 'auto' ? 'CF 代理' : '强制域名',
        dotColor: 'bg-sky-500',
        textColor: 'text-sky-700 dark:text-sky-300',
        bgColor: 'bg-sky-100/80 dark:bg-sky-950/60',
        borderColor: 'border-sky-200 dark:border-sky-800',
        icon: 'tabler:world',
        desc: '当前走 Cloudflare 域名代理回源通道',
      }
    default:
      return {
        label: routePreference.value === 'auto' ? 'IPv6 直连' : '强制 IPv6',
        dotColor: 'bg-emerald-500',
        textColor: 'text-emerald-700 dark:text-emerald-300',
        bgColor: 'bg-emerald-100/80 dark:bg-emerald-950/60',
        borderColor: 'border-emerald-200 dark:border-emerald-800',
        icon: 'tabler:bolt',
        desc: '当前通过公网 IPv6 高速直连 NAS',
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
      class="absolute left-0 mt-2 w-64 p-2 rounded-2xl bg-white dark:bg-slate-900 border border-slate-200 dark:border-slate-800 shadow-xl z-50 text-left space-y-1 animate-in fade-in zoom-in-95 duration-150"
    >
      <div class="px-2.5 py-1.5 border-b border-slate-100 dark:border-slate-800">
        <div class="text-[11px] font-bold text-slate-700 dark:text-slate-300">网络分流优先级</div>
        <div class="text-[10px] text-slate-400">决定点击书签卡片时优先跳转的连接模式</div>
      </div>

      <!-- Live diagnosis: shows what the IPv6 probe actually observed, so a
           mis-routed client can be debugged from the UI. -->
      <div
        v-if="lastProbeDetail"
        class="px-2.5 py-1.5 rounded-xl bg-slate-50 dark:bg-slate-800/60 border border-slate-100 dark:border-slate-800"
      >
        <div class="text-[10px] font-semibold text-slate-500 dark:text-slate-400 mb-0.5">IPv6 探测结果</div>
        <div class="text-[10px] leading-relaxed text-slate-600 dark:text-slate-300 break-all font-mono">
          {{ lastProbeDetail }}
        </div>
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
    </div>
  </div>
</template>
