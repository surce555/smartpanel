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
/**
 * Human-readable trace of the last IPv6 probe. Surfaced in the route badge menu
 * so a mis-routed client can be diagnosed from the UI instead of guessing.
 */
const lastProbeDetail = ref<string>('')
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
 * Stored as `<0|1>:<epoch-ms>`. A bare boolean kept for the whole session meant
 * a laptop that probed on Wi-Fi kept that answer after switching to an IPv4-only
 * hotspot, silently breaking routing.
 *
 * A NEGATIVE result expires quickly: it is the harmful answer, and one transient
 * timeout must not pin the client to the domain fallback for minutes.
 */
const IPV6_CACHE_KEY = 'smartpanel_ipv6_probe'
const IPV6_CACHE_TTL_POSITIVE_MS = 5 * 60 * 1000
const IPV6_CACHE_TTL_NEGATIVE_MS = 30 * 1000

function readIPv6Cache(): boolean | null {
  try {
    const raw = sessionStorage.getItem(IPV6_CACHE_KEY)
    if (!raw) return null
    const [flag, tsRaw] = raw.split(':')
    if (flag !== '0' && flag !== '1') return null
    const ts = Number(tsRaw)
    if (!Number.isFinite(ts)) return null
    const ttl = flag === '1' ? IPV6_CACHE_TTL_POSITIVE_MS : IPV6_CACHE_TTL_NEGATIVE_MS
    if (Date.now() - ts > ttl) {
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

/**
 * Build the probe URL for an IPv6 direct-connect host.
 *
 * Accepted inputs (as stored by the admin UI / bookmark templates):
 *   v6.example.com
 *   v6.example.com:5666
 *   https://v6.example.com:5666
 *   [2408:8214::1]:5666          (bracketed literal with port)
 *   2408:8214::1
 *
 * Two things must survive, because bookmark templates rely on them:
 *
 *  - the PORT. Templates point at `...:5666`; forcing an implicit :443 probes a
 *    port nothing listens on and reports "IPv6 unavailable".
 *  - the SCHEME. Templates commonly use plain `http://[{ipv6}]:5666`. Probing an
 *    HTTP endpoint over HTTPS fails the TLS handshake, which likewise reads as
 *    "unreachable" even though the host is perfectly reachable.
 *
 * @param fallbackScheme scheme to use when the input carries none. Callers pass
 *        the scheme observed in the bookmark template.
 */
export function buildV6ProbeUrl(
  v6domain?: string,
  fallbackPort?: string,
  fallbackScheme: 'http' | 'https' = 'http'
): string {
  let raw = (v6domain || '').trim()
  if (!raw) return ''

  let scheme = `${fallbackScheme}://`
  if (/^http:\/\//i.test(raw)) {
    scheme = 'http://'
    raw = raw.replace(/^http:\/\//i, '')
  } else if (/^https:\/\//i.test(raw)) {
    scheme = 'https://'
    raw = raw.replace(/^https:\/\//i, '')
  }

  // Drop any path/query; only the authority matters for a reachability probe.
  raw = raw.split('/')[0]

  const port = (fallbackPort || '').trim()
  const isBracketedV6 = raw.startsWith('[')

  if (isBracketedV6) {
    // Already bracketed: `[addr]` or `[addr]:port`.
    if (/^\[[^\]]+\]:\d+$/.test(raw)) return `${scheme}${raw}`
    if (port) return `${scheme}${raw}:${port}`
    return `${scheme}${raw}`
  }

  const colonCount = (raw.match(/:/g) || []).length
  const isBareV6 = colonCount > 1

  if (isBareV6) {
    // A bare IPv6 literal cannot be told apart from `host:port` by colon count
    // alone, so it must always be bracketed. The port goes OUTSIDE the brackets:
    //   http://[2408:8214::1]:5666   (correct)
    //   http://[2408:8214::1:5666]   (invalid - port swallowed into the literal)
    return port ? `${scheme}[${raw}]:${port}` : `${scheme}[${raw}]`
  }

  const hasPort = /:\d+$/.test(raw)
  if (!hasPort && port) return `${scheme}${raw}:${port}`
  return `${scheme}${raw}`
}

/**
 * Extract the scheme from a bookmark URL template, defaulting to http.
 * `http://[{ipv6}]:5666` -> 'http'.
 */
export function schemeFromTemplate(template?: string): 'http' | 'https' {
  const t = (template || '').trim().toLowerCase()
  if (t.startsWith('https://')) return 'https'
  if (t.startsWith('http://')) return 'http'
  // No template: prefer http on the typical NAS port, https otherwise.
  return 'http'
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
   * Probe IPv6 dual-stack reachability.
   *
   * @param opts.v6domain    the user's IPv6 direct-connect host, exactly as it
   *                         appears in bookmark templates (may include a port,
   *                         e.g. `v6.example.com:5666`)
   * @param opts.v6port      fallback port when `v6domain` has none
   * @param opts.directIPv6  backend proved this page arrived over direct IPv6
   * @param opts.viaCloudflare request came through the Cloudflare edge
   *
   * Evidence is ranked, strongest first. Only an active probe may report
   * "unavailable".
   *
   *   1. backend `direct_ipv6`  -> this page was served over direct IPv6.
   *   2. page opened via a bare IPv6 host.
   *   3. probe the user's own v6domain. This is the single most relevant test:
   *      it is the exact host the cards will link to, it proves the client can
   *      reach the NAS over IPv6, and it does not depend on any external
   *      third-party endpoint being reachable.
   *   4. probe public IPv6-only endpoints (weakest: often blocked in CN).
   *
   * When `v6domain` is configured we deliberately IGNORE a public-endpoint
   * failure. Being unable to reach `6.ipw.cn` says nothing about whether the
   * client can reach the user's own NAS; earlier versions conflated the two and
   * downgraded working IPv6 clients to the Cloudflare domain.
   */
  async function probeIPv6(
    customProbeUrl?: string,
    timeoutMs: number = 2000,
    opts: {
      clientIP?: string
      directIPv6?: boolean
      viaCloudflare?: boolean
      v6domain?: string
      v6port?: string
      currentIPv6?: string
      /** scheme used by the bookmark template, e.g. 'http' for http://[{ipv6}]:5666 */
      scheme?: 'http' | 'https'
      probeSelfUrl?: string
      force?: boolean
    } = {}
  ): Promise<boolean> {
    isProxiedClient.value = Boolean(opts.viaCloudflare)

    // 1. Backend proved this page arrived over direct IPv6.
    if (opts.directIPv6) {
      lastProbeDetail.value = '后端已确认本次页面经由 IPv6 直连'
      return setIPv6Result(true)
    }

    // 2. Page opened through a bare IPv6 host: IPv6 is proven.
    if (isIPv6Host(window.location.hostname)) {
      lastProbeDetail.value = '当前页面本身即通过 IPv6 地址访问'
      return setIPv6Result(true)
    }

    // 3. Reuse a recent probe result (negative results expire in 30s).
    if (!opts.force) {
      const cached = readIPv6Cache()
      if (cached !== null) {
        isIPv6Available.value = cached
        lastProbeDetail.value = `使用缓存结果: ${cached ? 'IPv6 可用' : 'IPv6 不可用'}`
        return cached
      }
    }

    isProbing.value = true
    try {
      // Build the v6 targets. Bookmark templates can use either `{v6domain}`
      // (`v6.example.com:5666`) or `{ipv6}` (`[2408:...]:5666`), so both forms
      // are probed: whichever form the user's cards actually use is the one that
      // matters. The scheme comes from the bookmark template (`http://` in the
      // common NAS setup), never an assumed https.
      const scheme = opts.scheme || 'http'
      const candidates: string[] = []
      const v6ByDomain = buildV6ProbeUrl(opts.v6domain, opts.v6port, scheme)
      if (v6ByDomain) candidates.push(v6ByDomain)
      const v6ByAddress = buildV6ProbeUrl(opts.currentIPv6, opts.v6port, scheme)
      if (v6ByAddress && !candidates.includes(v6ByAddress)) candidates.push(v6ByAddress)

      const parts: string[] = []

      // 3. Probe the user's own IPv6 endpoints first — most relevant evidence.
      if (candidates.length > 0) {
        const reachable = await Promise.all(
          candidates.map(async (url) => ({ url, ok: await probeSingleIPv6Endpoint(url, timeoutMs) }))
        )
        for (const r of reachable) {
          parts.push(`${r.url} -> ${r.ok ? '可达' : '不可达'}`)
        }
        if (reachable.some((r) => r.ok)) {
          lastProbeDetail.value = parts.join('; ')
          return setIPv6Result(true)
        }
        // None answered. Fall through to the generic probes: a failure here can
        // be a certificate/hostname problem rather than a missing IPv6 stack.
      } else {
        parts.push('未配置 IPv6 直连域名/地址，跳过自测')
      }

      const endpoints: string[] = []
      if (customProbeUrl) endpoints.push(customProbeUrl)
      if (opts.probeSelfUrl) endpoints.push(opts.probeSelfUrl)
      endpoints.push('https://6.ipw.cn', 'https://speed.neu6.edu.cn')

      const results = await Promise.all(
        endpoints.map((url) => probeSingleIPv6Endpoint(url, timeoutMs))
      )
      const available = results.some(Boolean)
      parts.push(`公网探测 ${available ? '可达' : '全部不可达'}`)
      lastProbeDetail.value = parts.join('; ')

      // A configured-but-unreachable v6domain combined with unreachable public
      // probes means IPv6 genuinely is not usable from this client.
      return setIPv6Result(available)
    } catch {
      lastProbeDetail.value = (lastProbeDetail.value || '') + '; 探测异常'
      return setIPv6Result(false)
    } finally {
      isProbing.value = false
    }
  }

  function setIPv6Result(available: boolean): boolean {
    isIPv6Available.value = available
    writeIPv6Cache(available)
    return available
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
    lastProbeDetail,
    routePreference,
    setRoutePreference,
    probeIPv6,
    probeLanConnectivity,
    clearProbeCache,
  }
}
