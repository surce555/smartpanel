import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useApi } from '@/composables/useApi'
import { useSettingsStore, NetworkInfo } from './settings'
import { useIPv6Probe, isIPv6Host, isLanEntry } from '@/composables/useIPv6Probe'

export interface Tag {
  id: string
  name: string
  color: string
}

export interface HealthCheck {
  bookmark_id: string
  status: 'online' | 'offline' | 'unknown'
  last_checked?: string
  response_time_ms: number
}

export interface Bookmark {
  id: string
  name: string
  icon: string
  url_internal: string
  url_public_template: string
  url_fallback: string
  group_id: string
  description: string
  open_in_new_tab: boolean
  is_private: boolean
  sort_order: number
  created_at?: string
  tags?: Tag[]
  tag_ids?: string[]
  health_check?: HealthCheck
}

export interface Group {
  id: string
  name: string
  icon?: string
  sort_order: number
  bookmarks?: Bookmark[]
}

export const useBookmarksStore = defineStore('bookmarks', () => {
  const groups = ref<Group[]>([])
  const bookmarks = ref<Bookmark[]>([])
  const tags = ref<Tag[]>([])
  const selectedTagId = ref<string | null>(null)
  const searchQuery = ref<string>('')
  const loading = ref<boolean>(false)
  const draggingBookmarkId = ref<string | null>(null)

  const settingsStore = useSettingsStore()
  const { isIPv6Available, routePreference } = useIPv6Probe()

  function getInternalUrl(b: Bookmark): string {
    return b.url_internal ? b.url_internal.trim() : ''
  }

  function getV6Url(b: Bookmark, net: NetworkInfo): string {
    if (!b.url_public_template) return ''
    let cleanV6 = (net.current_ipv6 || net.detected_host_ipv6 || '').trim()
    if (!cleanV6 && isIPv6Host(window.location.hostname)) {
      cleanV6 = window.location.hostname.replace(/^\[|\]$/g, '')
    }
    const tmpl = b.url_public_template.trim()
    if (tmpl.includes('{ipv6}') && !cleanV6) {
      return ''
    }
    if (tmpl.includes('{v6domain}') && !net.v6domain) {
      return ''
    }
    return tmpl
      .replace(/{ipv6}/g, cleanV6)
      .replace(/{domain}/g, net.domain || window.location.hostname)
      .replace(/{v6domain}/g, net.v6domain || '')
  }

  function getFallbackUrl(b: Bookmark, net: NetworkInfo): string {
    if (!b.url_fallback) return ''
    return b.url_fallback.trim()
      .replace(/{domain}/g, net.domain || window.location.hostname)
      .replace(/{v6domain}/g, net.v6domain || '')
      .replace(/{ipv6}/g, net.current_ipv6 || net.detected_host_ipv6 || '')
  }

  /**
   * Whether the current page is genuinely being visited through a LAN address
   * (private IPv4 / localhost / *.local / *.lan / configured lan_domain).
   * See `isLanEntry` for why reachability signals must not be used here.
   */
  function isAccessingViaLanEntry(): boolean {
    return isLanEntry(window.location.hostname, settingsStore.networkInfo.lan_domain)
  }

  /**
   * Resolve final URL for a bookmark dynamically based on the current entry point
   *
   * Auto priority hierarchy:
   * 1. 通过内网地址 / 内网域名访问 -> 内网直连 (url_internal)
   * 2. 通过 IPv6 地址访问         -> IPv6 直连 (url_public_template)
   * 3. 通过公网域名访问           -> IPv6 可用则直连, 否则域名回退 (url_fallback)
   */
  function resolveBookmarkUrl(b: Bookmark): string {
    const net = settingsStore.networkInfo
    const pref = routePreference.value
    const internalUrl = getInternalUrl(b)
    const v6Url = getV6Url(b, net)
    const fallbackUrl = getFallbackUrl(b, net)

    // 1. Explicit preference overrides (user forced a mode via the route badge)
    if (pref === 'lan') {
      return internalUrl || v6Url || fallbackUrl || '#'
    }
    if (pref === 'ipv6') {
      return v6Url || internalUrl || fallbackUrl || '#'
    }
    if (pref === 'domain') {
      return fallbackUrl || v6Url || internalUrl || '#'
    }

    // 2. Auto Routing
    const hostname = window.location.hostname

    // (a) Opened through the LAN entry (private IP / *.lan / lan_domain).
    //     The browser proved it can reach the NAS internally, so prefer it.
    if (isAccessingViaLanEntry()) {
      return internalUrl || v6Url || fallbackUrl || '#'
    }

    // (b) Opened through a bare IPv6 address: the client is IPv6-capable.
    if (isIPv6Host(hostname)) {
      return v6Url || internalUrl || fallbackUrl || '#'
    }

    // (c) Opened through the public domain: use the Cloudflare/Tunnel domain in
    //     hand unless IPv6 has been POSITIVELY confirmed working.
    //
    //     Fail-safe, not fail-open. The previous version routed to the IPv6 URL
    //     whenever `isIPv6Available !== false`, which includes `null` (never
    //     probed / probe inconclusive). On an IPv4-only network that sent every
    //     card to an unreachable IPv6 address — the page simply did not open.
    //     Now IPv6 requires a confirmed `true`; anything else falls back to the
    //     domain the user is already successfully browsing through.
    if (isIPv6Available.value === true && v6Url) {
      return v6Url
    }
    if (fallbackUrl) {
      return fallbackUrl
    }
    // No fallback configured: last resort, without a known-good domain there is
    // nothing better than the IPv6 template.
    if (v6Url) {
      return v6Url
    }
    return internalUrl || '#'
  }

  // Filtered bookmarks
  const filteredBookmarks = computed(() => {
    return bookmarks.value.filter((b) => {
      // Tag filter
      if (selectedTagId.value) {
        const hasTag = b.tags?.some((t) => t.id === selectedTagId.value)
        if (!hasTag) return false
      }
      // Search query filter
      if (searchQuery.value.trim()) {
        const q = searchQuery.value.toLowerCase().trim()
        const matchName = b.name.toLowerCase().includes(q)
        const matchDesc = b.description?.toLowerCase().includes(q)
        const matchTag = b.tags?.some((t) => t.name.toLowerCase().includes(q))
        if (!matchName && !matchDesc && !matchTag) return false
      }
      return true
    })
  })

  // Grouped bookmarks for display
  const groupedBookmarks = computed(() => {
    const map = new Map<string, Bookmark[]>()
    // Initialize groups
    for (const g of groups.value) {
      map.set(g.id, [])
    }
    // Ungrouped
    map.set('ungrouped', [])

    for (const b of filteredBookmarks.value) {
      if (b.group_id && map.has(b.group_id)) {
        map.get(b.group_id)!.push(b)
      } else {
        map.get('ungrouped')!.push(b)
      }
    }

    return groups.value.map((g) => ({
      ...g,
      bookmarks: map.get(g.id) || [],
    })).filter((g) => {
      // If searching, hide empty groups
      if (searchQuery.value.trim() || selectedTagId.value) {
        return (g.bookmarks?.length ?? 0) > 0
      }
      return true
    })
  })

  async function fetchAll() {
    loading.value = true
    const api = useApi()
    try {
      const [gRes, bRes, tRes] = await Promise.all([
        api.get('/groups'),
        api.get('/bookmarks'),
        api.get('/tags'),
      ])
      groups.value = gRes.data
      bookmarks.value = bRes.data
      tags.value = tRes.data
    } catch (e) {
      console.error('Failed to fetch bookmark data:', e)
    } finally {
      loading.value = false
    }
  }

  /**
   * Derive the IPv6 probe parameters from the bookmarks' actual URL templates.
   *
   * The NAS IPv6 address changes on every boot, so it must never be hardcoded —
   * the template is the source of truth. Templates look like
   * `http://[{ipv6}]:5666` or `https://{v6domain}:5666`, and the scheme/port in
   * them decide how the probe must be built. Probing an HTTP endpoint over HTTPS
   * (or the wrong port) fails the handshake and reads as "IPv6 unavailable".
   */
  function deriveIPv6ProbeParams(): {
    scheme: 'http' | 'https'
    port: string
    hostTemplate: string
  } {
    const withTemplate = bookmarks.value.find((b) => (b.url_public_template || '').trim())
    const template = (withTemplate?.url_public_template || '').trim()

    let scheme: 'http' | 'https' = 'http'
    if (/^https:\/\//i.test(template)) scheme = 'https'
    else if (/^http:\/\//i.test(template)) scheme = 'http'

    // Port from the template (e.g. `:5666`), else the panel's own port.
    let port = ''
    const portMatch = template.match(/\]:(\d+)/) || template.match(/:(\d+)(?:\/|$)/)
    if (portMatch) port = portMatch[1]
    if (!port) port = settingsStore.networkInfo.server_port || ''

    return { scheme, port, hostTemplate: template }
  }

  async function createBookmark(data: Partial<Bookmark>) {
    const api = useApi()
    const resp = await api.post('/bookmarks', data)
    await fetchAll()
    return resp.data
  }

  async function updateBookmark(id: string, data: Partial<Bookmark>) {
    const api = useApi()
    const resp = await api.put(`/bookmarks/${id}`, data)
    await fetchAll()
    return resp.data
  }

  async function deleteBookmark(id: string) {
    const api = useApi()
    await api.delete(`/bookmarks/${id}`)
    await fetchAll()
  }

  async function reorderBookmarks(items: { id: string; sort_order: number; group_id?: string }[]) {
    const api = useApi()
    await api.put('/bookmarks/reorder', items)
    await fetchAll()
  }

  async function moveBookmark(sourceId: string, targetId?: string, targetGroupId?: string) {
    if (sourceId === targetId) return

    const sourceIndex = bookmarks.value.findIndex((b) => b.id === sourceId)
    if (sourceIndex === -1) return

    const sourceItem = { ...bookmarks.value[sourceIndex] }
    const newGroupId = targetGroupId || (targetId ? bookmarks.value.find((b) => b.id === targetId)?.group_id : sourceItem.group_id) || sourceItem.group_id
    sourceItem.group_id = newGroupId

    const list = [...bookmarks.value]
    list.splice(sourceIndex, 1)

    if (targetId) {
      const targetIndex = list.findIndex((b) => b.id === targetId)
      if (targetIndex !== -1) {
        list.splice(targetIndex, 0, sourceItem)
      } else {
        list.push(sourceItem)
      }
    } else {
      list.push(sourceItem)
    }

    const counters: Record<string, number> = {}
    const itemsToUpdate: { id: string; sort_order: number; group_id: string }[] = []

    for (const b of list) {
      const gid = b.group_id || 'ungrouped'
      const order = counters[gid] || 0
      b.sort_order = order
      counters[gid] = order + 1
      itemsToUpdate.push({ id: b.id, sort_order: order, group_id: b.group_id })
    }

    // Optimistic local update
    bookmarks.value = list

    const api = useApi()
    try {
      await api.put('/bookmarks/reorder', itemsToUpdate)
    } catch (err) {
      console.error('Failed to save bookmark reordering:', err)
      await fetchAll()
    }
  }

  async function createGroup(name: string, icon?: string) {
    const api = useApi()
    const resp = await api.post('/groups', { name, icon })
    await fetchAll()
    return resp.data
  }

  async function updateGroup(id: string, name: string, icon?: string) {
    const api = useApi()
    const resp = await api.put(`/groups/${id}`, { name, icon })
    await fetchAll()
    return resp.data
  }

  async function deleteGroup(id: string) {
    const api = useApi()
    await api.delete(`/groups/${id}`)
    await fetchAll()
  }

  async function reorderGroups(items: { id: string; sort_order: number }[]) {
    const api = useApi()
    await api.put('/groups/reorder', items)
    await fetchAll()
  }

  async function createTag(name: string, color: string) {
    const api = useApi()
    const resp = await api.post('/tags', { name, color })
    await fetchAll()
    return resp.data
  }

  async function deleteTag(id: string) {
    const api = useApi()
    await api.delete(`/tags/${id}`)
    await fetchAll()
  }

  return {
    groups,
    bookmarks,
    tags,
    selectedTagId,
    searchQuery,
    loading,
    filteredBookmarks,
    groupedBookmarks,
    resolveBookmarkUrl,
    deriveIPv6ProbeParams,
    fetchAll,
    createBookmark,
    updateBookmark,
    deleteBookmark,
    reorderBookmarks,
    moveBookmark,
    draggingBookmarkId,
    createGroup,
    updateGroup,
    deleteGroup,
    reorderGroups,
    createTag,
    deleteTag,
  }
})
