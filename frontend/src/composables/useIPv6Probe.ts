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

export function useIPv6Probe() {
  function setRoutePreference(mode: 'auto' | 'lan' | 'ipv6' | 'domain') {
    routePreference.value = mode
    localStorage.setItem('smartpanel_route_preference', mode)
  }



  /**
   * Probe whether THIS CLIENT (the phone/PC in front of the screen) can use
   * IPv6 at all.
   *
   * This is deliberately NOT a test of the NAS. Whether the NAS has IPv6 is
   * already known from the backend (`current_ipv6` / `detected_host_ipv6`); the
   * only open question at click time is whether the viewer's own network can
   * reach IPv6 addresses.
   *
   * Two earlier approaches both failed on a proxy setup (Clash with IPv6 turned
   * off), and the reason matters:
   *
   *  - `<img>` + elapsed-time threshold: an unreachable address can take hundreds
   *    of ms to fail, exceeding the threshold, so everything looked IPv6-capable.
   *  - `fetch(..., { mode: 'no-cors' })` + literal IPv6: a proxy answers with its
   *    own error page. `no-cors` cannot read the status, so that error page
   *    RESOLVES the fetch and again looks like a working IPv6 connection.
   *
   * This version uses CORS mode against an IPv6-ONLY host. That combination is
   * what makes it sound:
   *
   *   - `api6.ipify.org` has an AAAA record and no usable A record, so it can
   *     only be reached over IPv6. Reaching it proves IPv6 works.
   *   - CORS mode exposes the real HTTP status, so a proxy's 502 error page is
   *     rejected instead of being mistaken for success.
   *   - The proxy error page additionally carries no `Access-Control-Allow-Origin`,
   *     so the request fails the CORS check. Either way it cannot pass.
   */
  async function probeClientIPv6(timeoutMs: number): Promise<boolean> {
    // IPv6-only endpoints that send permissive CORS headers.
    const targets = [
      'https://api6.ipify.org?format=json',
      'https://v6.ident.me',
      'https://ipv6.icanhazip.com',
    ]

    const probeOne = async (url: string): Promise<boolean> => {
      const controller = new AbortController()
      const timer = window.setTimeout(() => controller.abort(), timeoutMs)
      try {
        const res = await fetch(url, {
          mode: 'cors',
          cache: 'no-store',
          credentials: 'omit',
          redirect: 'follow',
          referrerPolicy: 'no-referrer',
          signal: controller.signal,
        })
        // A proxy/gateway error page must not count as a working IPv6 path.
        if (res.status === 502 || res.status === 503 || res.status === 504) {
          return false
        }
        // Any real answer from an IPv6-only host proves IPv6 connectivity.
        // 200 with JSON is the expected case; auth/console quirks that return
        // 401/403 still required a working IPv6 connection, so accept them.
        return res.ok || res.status === 401 || res.status === 403
      } catch {
        // DNS/route/TLS failure, CORS rejection, or abort on timeout.
        return false
      } finally {
        clearTimeout(timer)
      }
    }

    const results = await Promise.all(targets.map((u) => probeOne(u)))
    // One confirmed IPv6 connection is enough; all failing means no IPv6.
    return results.some(Boolean)
  }

  /**
   * Probe IPv6 availability for the current client.
   *
   * @param opts.clientIP       backend-reported client IP for this request
   * @param opts.directIPv6     backend proved the page arrived over direct IPv6
   * @param opts.viaCloudflare  request came through the Cloudflare edge
   * @param opts.force          ignore the cached answer
   *
   * Evidence, strongest first:
   *   1. `client_has_ipv6` - backend saw the visitor's real address (through
   *      CF-Connecting-IP / XFF when proxied) and it is an IPv6 address, so the
   *      browser demonstrably has a working IPv6 stack.
   *   2. `direct_ipv6` - this page itself was served over IPv6.
   *   3. page opened via a bare IPv6 host.
   *   4. active client-side IPv6 test (see `probeClientIPv6`).
   *
   * Evidence 1 outranks the browser probe on purpose. The probe talks to
   * third-party hosts, so when the user is behind a proxy/VPN or Clash with
   * IPv6 disabled it fails, even though the NAS is perfectly reachable over
   * IPv6 from that same machine. The backend's view of the visitor address has
   * no such blind spot.
   */
  async function probeIPv6(
    _customProbeUrl?: string,
    timeoutMs: number = 4000,
    opts: {
      clientIP?: string
      directIPv6?: boolean
      clientHasIPv6?: boolean
      viaCloudflare?: boolean
      force?: boolean
    } = {}
  ): Promise<boolean> {
    isProxiedClient.value = Boolean(opts.viaCloudflare)

    // 1. Backend proved the client itself is IPv6-capable.
    if (opts.clientHasIPv6) {
      lastProbeDetail.value = '后端观测到访客地址为 IPv6，客户端支持 IPv6'
      return setIPv6Result(true)
    }

    // 2. Backend proved this page arrived over direct IPv6.
    if (opts.directIPv6) {
      lastProbeDetail.value = '此页面本身经由 IPv6 直连到达，客户端支持 IPv6'
      return setIPv6Result(true)
    }

    // 3. Page opened through a bare IPv6 host.
    if (isIPv6Host(window.location.hostname)) {
      lastProbeDetail.value = '当前通过 IPv6 地址访问，客户端支持 IPv6'
      return setIPv6Result(true)
    }

    // 4. Reuse a recent answer.
    if (!opts.force) {
      const cached = readIPv6Cache()
      if (cached !== null) {
        isIPv6Available.value = cached
        lastProbeDetail.value = `客户端 IPv6 探测(缓存): ${cached ? '支持' : '不支持'}`
        return cached
      }
    }

    isProbing.value = true
    try {
      const ok = await probeClientIPv6(timeoutMs)
      lastProbeDetail.value = ok
        ? '客户端可访问 IPv6 地址，判定支持 IPv6'
        : '客户端无法访问任何 IPv6 地址，判定不支持 IPv6'
      return setIPv6Result(ok)
    } catch {
      lastProbeDetail.value = '客户端 IPv6 探测异常，保持未知'
      return setIPv6Result(null)
    } finally {
      isProbing.value = false
    }
  }

  /**
   * Record the probe outcome.
   *
   * `null` means "undetermined" — the probe could not run (e.g. blocked as mixed
   * content) or hit an unexpected error. It must NOT be cached and must NOT be
   * treated as a failure: the resolver only downgrades on an explicit `false`.
   */
  function setIPv6Result(available: boolean | null): boolean {
    isIPv6Available.value = available
    if (available === null) {
      try {
        sessionStorage.removeItem(IPV6_CACHE_KEY)
      } catch {
        // ignore
      }
      return false
    }
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
