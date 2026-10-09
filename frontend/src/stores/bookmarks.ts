import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { useApi } from '@/composables/useApi'
import { useSettingsStore, NetworkInfo } from './settings'
import { useIPv6Probe } from '@/composables/useIPv6Probe'

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

  const settingsStore = useSettingsStore()
  const { isIPv6Available } = useIPv6Probe()

  /**
   * Resolve final URL for a bookmark dynamically based on network state
   */
  function resolveBookmarkUrl(b: Bookmark): string {
    const net = settingsStore.networkInfo
    const hasV6 = isIPv6Available.value !== false // Default to direct if true or not probed yet

    let template = b.url_public_template
    if (!template) {
      // Fallback to internal or fallback
      return b.url_internal || b.url_fallback || '#'
    }

    // If client is determined to NOT have IPv6 connectivity, fallback to Tunnel domain or url_fallback
    if (!hasV6) {
      if (b.url_fallback) {
        return b.url_fallback
          .replace(/{domain}/g, net.domain || window.location.hostname)
          .replace(/{v6domain}/g, net.v6domain || '')
      }
    }

    // Standard template substitution:
    // Format IPv6 cleanly, ensuring brackets if template is like https://[{ipv6}]:port or https://{ipv6}:port
    let cleanV6 = net.current_ipv6 || ''

    let resolved = template
      .replace(/{ipv6}/g, cleanV6)
      .replace(/{domain}/g, net.domain || window.location.hostname)
      .replace(/{v6domain}/g, net.v6domain || '')

    return resolved
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
    fetchAll,
    createBookmark,
    updateBookmark,
    deleteBookmark,
    reorderBookmarks,
    createGroup,
    updateGroup,
    deleteGroup,
    reorderGroups,
    createTag,
    deleteTag,
  }
})
