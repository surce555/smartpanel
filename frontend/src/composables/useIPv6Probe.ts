import { ref } from 'vue'

const isIPv6Available = ref<boolean | null>(null)
const isLanAvailable = ref<boolean | null>(null)
const isProbing = ref<boolean>(false)
/**
 * True when the page itself arrived through the Cloudflare edge. In that case
 * `client_ip` is the visitor's public address and says nothing about which
 * protocol reached the NAS, so it cannot be used as IPv6 proof.
 */
const isProxiedClient = ref<boolean>(false)
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
 * Whether a plain IP string is IPv6 (used for the client IP the backend reports).
 * `client_ip` arrives as `2408:8207:...` or `::ffff:1.2.3.4` style values.
 */
export function isIPv6Address(ip?: string): boolean {
  if (!ip) return false
  const v = ip.trim().replace(/^\[|\]$/g, '')
  if (!v.includes(':')) return false
  // IPv4-mapped IPv6 (::ffff:192.168.1.1) is really an IPv4 connection.
  if (/^::ffff:\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$/i.test(v)) return false
  return /^[0-9a-fA-F:]+$/.test(v)
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
   * Probe IPv6 connectivity.
   *
   * Evidence is ranked; earlier sources are strictly more reliable than later:
   *
   *   1. Page opened via a bare IPv6 host    -> IPv6 proven, no probe needed.
   *   2. Backend saw the client connect over IPv6 (`client_ip` is v6)
   *      -> IPv6 proven. The request that delivered this very page arrived over
   *         IPv6, so the client demonstrably has working IPv6 connectivity.
   *   3. Active probe against IPv6-only public endpoints.
   *
   * Step 2 is what fixes the "IPv6 client routed to the CF domain" regression:
   * the external probe can fail for reasons unrelated to the client (endpoint
   * blocked by a firewall / proxy / slow region), which used to be treated as
   * "client has no IPv6" and downgraded everyone to the domain fallback.
   *
   * The probe distinguishes real reachability from mere failure: `fetch` only
   * rejects when the network layer failed; a no-cors opaque response resolving
   * means DNS + TCP + TLS all worked.
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

  /**
   * @param opts.clientIP       backend-reported client IP for this request
   * @param opts.directIPv6     backend proved the page arrived over a direct
   *                            (non-Cloudflare) IPv6 connection
   * @param opts.v6domain       the user's own IPv6-only host (e.g. v6.example.com)
   * @param opts.probeSelfUrl   same-origin URL to probe when nothing better exists
   * @param opts.force          ignore the cache and re-probe (network change)
   *
   * Evidence is ranked, strongest first. Only the active external probe can
   * return "unavailable" — every other source is positive proof.
   *
   *   1. `direct_ipv6` from the backend: this very page was served over a direct
   *      IPv6 connection, so the client demonstrably has IPv6.
   *   2. Page opened via a bare IPv6 host.
   *   3. Probe the user's own v6domain — the exact host the cards link to, so a
   *      success here is the most relevant possible signal.
   *   4. Probe public IPv6-only endpoints. These are frequently blocked or slow
   *      on mainland networks, so a failure here is the *weakest* evidence.
   */
  async function probeIPv6(
    customProbeUrl?: string,
    timeoutMs: number = 1500,
    opts: {
      clientIP?: string
      directIPv6?: boolean
      viaCloudflare?: boolean
      v6domain?: string
      probeSelfUrl?: string
      force?: boolean
    } = {}
  ): Promise<boolean> {
    isProxiedClient.value = Boolean(opts.viaCloudflare)

    // 1. Backend proved this page arrived over direct IPv6.
    if (opts.directIPv6) {
      isIPv6Available.value = true
      writeIPv6Cache(true)
      return true
    }

    // 2. Page opened through a bare IPv6 host: IPv6 is proven.
    if (isIPv6Host(window.location.hostname)) {
      isIPv6Available.value = true
      writeIPv6Cache(true)
      return true
    }

    // 3. Direct (non-proxied) connection whose peer address is IPv6.
    //    `directIPv6` already covers this, but some deployments hide RemoteAddr;
    //    treat a non-proxied IPv6 client_ip as equivalent proof.
    if (!opts.v6domain && !opts.probeSelfUrl && isIPv6Address(opts.clientIP) && !isProxiedClient.value) {
      isIPv6Available.value = true
      writeIPv6Cache(true)
      return true
    }

    // 4. Reuse a recent probe result (5 min TTL, reset on network change).
    if (!opts.force) {
      const cached = readIPv6Cache()
      if (cached !== null) {
        isIPv6Available.value = cached
        return cached
      }
    }

    isProbing.value = true

    // Ordered by usefulness: the user's own v6 host first, then generic probes.
    const endpoints: string[] = []
    if (opts.v6domain) {
      endpoints.push(`https://${opts.v6domain.replace(/^https?:\/\//, '').replace(/\/$/, '')}`)
    }
    if (customProbeUrl) endpoints.push(customProbeUrl)
    if (opts.probeSelfUrl) endpoints.push(opts.probeSelfUrl)
    endpoints.push('https://6.ipw.cn', 'https://speed.neu6.edu.cn')

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
    isProxiedClient,
    routePreference,
    setRoutePreference,
    probeIPv6,
    probeLanConnectivity,
    clearProbeCache,
  }
}
