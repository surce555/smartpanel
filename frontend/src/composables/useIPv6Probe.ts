import { ref } from 'vue'

const isIPv6Available = ref<boolean | null>(null)
const isLanAvailable = ref<boolean | null>(null)
const isProbing = ref<boolean>(false)
const routePreference = ref<'auto' | 'lan' | 'ipv6' | 'domain'>(
  (localStorage.getItem('smartpanel_route_preference') as any) || 'auto'
)

export function isLanHost(hostname: string): boolean {
  if (!hostname) return false
  if (hostname === 'localhost' || hostname === '127.0.0.1' || hostname === '::1') return true
  if (hostname.endsWith('.local') || hostname.endsWith('.lan') || hostname.endsWith('.home.arpa')) return true

  if (/^10\.\d{1,3}\.\d{1,3}\.\d{1,3}$/.test(hostname)) return true
  if (/^172\.(1[6-9]|2\d|3[01])\.\d{1,3}\.\d{1,3}$/.test(hostname)) return true
  if (/^192\.168\.\d{1,3}\.\d{1,3}$/.test(hostname)) return true

  return false
}

export function isIPv6Host(hostname: string): boolean {
  if (!hostname) return false
  return hostname.includes(':') || hostname.startsWith('[') || /^([0-9a-fA-F]{1,4}:){2,}/.test(hostname)
}

/**
 * Whether the page is currently opened through a LAN entry.
 *
 * Only the *access origin* matters here. Reachability hints such as "the client
 * can also reach the NAS over LAN" must NOT be treated as "in LAN", otherwise
 * opening the panel via the public domain would force every card link onto the
 * internal URL and silently disable IPv6 / domain dual-stack routing.
 */
export function isLanEntry(hostname: string, lanDomain?: string): boolean {
  if (!hostname) return false
  if (isLanHost(hostname)) return true
  const cleanDomain = (lanDomain || '').trim().replace(/^\*?\./, '')
  if (cleanDomain && (hostname === cleanDomain || hostname.endsWith(`.${cleanDomain}`))) {
    return true
  }
  return false
}

/**
 * Cached IPv6 probe result.
 *
 * Stored as `<0|1>:<epoch-ms>` so the answer expires. A bare boolean kept for
 * the entire session meant a laptop that probed on Wi-Fi kept using that result
 * after switching to an IPv4-only mobile hotspot, silently breaking routing.
 */
const IPV6_CACHE_KEY = 'smartpanel_ipv6_probe'
const IPV6_CACHE_TTL_MS = 5 * 60 * 1000

function readIPv6Cache(): boolean | null {
  try {
    const raw = sessionStorage.getItem(IPV6_CACHE_KEY)
    if (!raw) return null
    const [flag, tsRaw] = raw.split(':')
    if (flag !== '0' && flag !== '1') return null
    const ts = Number(tsRaw)
    if (!Number.isFinite(ts) || Date.now() - ts > IPV6_CACHE_TTL_MS) {
      sessionStorage.removeItem(IPV6_CACHE_KEY)
      return null
    }
    return flag === '1'
  } catch {
    return null
  }
}

function writeIPv6Cache(available: boolean) {
  try {
    sessionStorage.setItem(IPV6_CACHE_KEY, `${available ? '1' : '0'}:${Date.now()}`)
  } catch {
    // sessionStorage may be unavailable (private mode / quota); probing still works.
  }
}

export function useIPv6Probe() {
  function setRoutePreference(mode: 'auto' | 'lan' | 'ipv6' | 'domain') {
    routePreference.value = mode
    localStorage.setItem('smartpanel_route_preference', mode)
  }

  /**
   * Probe IPv6 connectivity using endpoints that only resolve over IPv6.
   *
   * Uses `fetch(..., { mode: 'no-cors' })` instead of an <img> probe. The old
   * implementation resolved `true` from `img.onerror`, on the assumption that
   * "an error on an IPv6-only host means DNS/TCP succeeded". That assumption is
   * wrong: on a network without IPv6 the AAAA lookup fails and the request
   * errors out *immediately*, which also fires `onerror` — so every IPv4-only
   * client was reported as "IPv6 available" and every card link was routed to
   * an unreachable IPv6 address.
   *
   * `fetch` gives a real discriminator: it RESOLVES once the response headers
   * arrive (an opaque response for no-cors counts), and only REJECTS when the
   * network layer itself failed. So:
   *   resolve -> IPv6 really works
   *   reject  -> IPv6 is unusable (or too slow to be worth using)
   */
  async function probeSingleIPv6Endpoint(url: string, timeout: number): Promise<boolean> {
    const controller = new AbortController()
    const timer = window.setTimeout(() => controller.abort(), timeout)
    try {
      await fetch(`${url}${url.includes('?') ? '&' : '?'}_t=${Date.now()}`, {
        mode: 'no-cors',
        cache: 'no-store',
        credentials: 'omit',
        redirect: 'follow',
        signal: controller.signal,
      })
      return true
    } catch {
      // Network error, DNS failure, TLS failure or timeout.
      return false
    } finally {
      clearTimeout(timer)
    }
  }

  async function probeIPv6(customProbeUrl?: string, timeoutMs: number = 1500): Promise<boolean> {
    // 1. Already browsing over IPv6: no probe needed.
    if (isIPv6Host(window.location.hostname)) {
      isIPv6Available.value = true
      writeIPv6Cache(true)
      return true
    }

    // 2. Reuse a recent result. Cached with a timestamp: the old code stored a
    //    bare flag for the whole session, so moving between networks (Wi-Fi to
    //    cellular) kept the stale answer until the tab was closed.
    const cached = readIPv6Cache()
    if (cached !== null) {
      isIPv6Available.value = cached
      return cached
    }

    isProbing.value = true

    // Only IPv6-reachable hosts: they either have no A record at all, or resolve
    // to something unusable over IPv4, so a successful fetch implies IPv6.
    const endpoints = [
      customProbeUrl,
      'https://6.ipw.cn',
      'https://speed.neu6.edu.cn',
    ].filter(Boolean) as string[]

    try {
      // Race them in parallel: one success is enough, and this is faster than
      // walking the list one by one.
      const results = await Promise.all(
        endpoints.map((url) => probeSingleIPv6Endpoint(url, timeoutMs))
      )
      const available = results.some(Boolean)

      isIPv6Available.value = available
      writeIPv6Cache(available)
      return available
    } catch {
      isIPv6Available.value = false
      writeIPv6Cache(false)
      return false
    } finally {
      isProbing.value = false
    }
  }

  /**
   * Probe LAN reachability by checking local host IPs with short timeout.
   */
  async function probeLanConnectivity(hostLanIps?: string[], port: string = '5050', timeoutMs: number = 500): Promise<boolean> {
    if (isLanHost(window.location.hostname)) {
      isLanAvailable.value = true
      return true
    }

    if (!hostLanIps || hostLanIps.length === 0) {
      return false
    }

    // If current page is HTTPS, browsers block mixed content fetch to HTTP, skip active probe
    if (window.location.protocol === 'https:') {
      return false
    }

    const testIp = hostLanIps[0]
    const testUrl = `http://${testIp}:${port}/api/system/network`

    try {
      const controller = new AbortController()
      const timer = setTimeout(() => controller.abort(), timeoutMs)
      await fetch(testUrl, { mode: 'no-cors', signal: controller.signal })
      clearTimeout(timer)
      isLanAvailable.value = true
      return true
    } catch (err: any) {
      if (err.name !== 'AbortError') {
        // Non-timeout error means TCP connection reached host
        isLanAvailable.value = true
        return true
      }
      return false
    }
  }

  function clearProbeCache() {
    try {
      sessionStorage.removeItem(IPV6_CACHE_KEY)
    } catch {
      // ignore
    }
    isIPv6Available.value = null
    isLanAvailable.value = null
  }

  return {
    isIPv6Available,
    isLanAvailable,
    isProbing,
    routePreference,
    setRoutePreference,
    probeIPv6,
    probeLanConnectivity,
    clearProbeCache,
  }
}
