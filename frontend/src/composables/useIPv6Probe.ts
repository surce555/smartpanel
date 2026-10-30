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
   * reach IPv6 addresses. Conflating the two caused several wrong fixes.
   *
   * Implementation notes:
   *
   *  - Targets are LITERAL IPv6 addresses (`https://[2001:4860:4860::8888]/`).
   *    A literal needs no AAAA record, so the test cannot fail because of DNS or
   *    because a domestic probe domain is blocked - the two failure modes that
   *    made the old `6.ipw.cn` probes unreliable here.
   *  - Uses an <img> probe rather than fetch: an <img> is not subject to CORS,
   *    and on an HTTPS page an https:// target avoids the mixed-content block.
   *  - Distinguishes "answered" from "never connected" primarily by TIME. A
   *    reachable IPv6 host answers quickly (even with an error status); a client
   *    with no IPv6 cannot route the address and stays silent until the timeout.
   *
   * The original implementation relied on the same <img>+timeout idea and worked
   * in practice; it only fell apart because the probe target was a domestic
   * domain that became unreachable, so every check timed out and reported
   * "no IPv6" regardless of the client.
   */
  function probeClientIPv6(timeoutMs: number): Promise<boolean> {
    // IPv6-only endpoints reached by literal address (no DNS involved).
    const targets = [
      'https://[2001:4860:4860::8888]/',
      'https://[2606:4700:4700::1111]/',
      'https://[2620:fe::fe]/',
    ]

    const probeOne = (url: string, timeout: number): Promise<boolean> =>
      new Promise<boolean>((resolve) => {
        const img = new Image()
        const startedAt = Date.now()
        let settled = false

        const finish = (ok: boolean) => {
          if (settled) return
          settled = true
          img.onload = null
          img.onerror = null
          clearTimeout(timer)
          resolve(ok)
        }

        const timer = window.setTimeout(() => finish(false), timeout)

        img.onload = () => finish(true)
        img.onerror = () => {
          // An error alone is ambiguous: it fires both when the host answered
          // with a non-image response AND when the connection could not be made.
          // Time disambiguates - a real connection completes a TCP/TLS round
          // trip, whereas an unroutable address fails almost instantly.
          finish(Date.now() - startedAt > 60)
        }

        img.src = `${url}?_t=${Date.now()}`
      })

    return Promise.all(targets.map((u) => probeOne(u, timeoutMs))).then((r) => r.some(Boolean))
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
   *   1. `direct_ipv6` - this page itself was served over IPv6.
   *   2. page opened via a bare IPv6 host.
   *   3. active client-side IPv6 test (see `probeClientIPv6`).
   */
  async function probeIPv6(
    _customProbeUrl?: string,
    timeoutMs: number = 2000,
    opts: {
      clientIP?: string
      directIPv6?: boolean
      viaCloudflare?: boolean
      force?: boolean
    } = {}
  ): Promise<boolean> {
    isProxiedClient.value = Boolean(opts.viaCloudflare)

    // 1. Backend proved this page arrived over direct IPv6.
    if (opts.directIPv6) {
      lastProbeDetail.value = '此页面本身经由 IPv6 直连到达，客户端支持 IPv6'
      return setIPv6Result(true)
    }

    // 2. Page opened through a bare IPv6 host.
    if (isIPv6Host(window.location.hostname)) {
      lastProbeDetail.value = '当前通过 IPv6 地址访问，客户端支持 IPv6'
      return setIPv6Result(true)
    }

    // 3. Reuse a recent answer.
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
