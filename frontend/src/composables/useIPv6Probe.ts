import { ref } from 'vue'

const isIPv6Available = ref<boolean | null>(null)
const isLanAvailable = ref<boolean | null>(null)
const isProbing = ref<boolean>(false)
const lastProbeDetail = ref<string>('')
const routePreference = ref<'auto' | 'lan' | 'ipv6' | 'domain'>(
  (localStorage.getItem('smartpanel_route_preference') as any) || 'auto'
)

/**
 * Check if the given hostname is a local/private host (IP or LAN TLD)
 */
export function isLanHost(hostname: string): boolean {
  if (!hostname) return false
  const clean = hostname.trim().toLowerCase()
  if (clean === 'localhost' || clean === '127.0.0.1' || clean === '::1') return true
  if (clean.endsWith('.local') || clean.endsWith('.lan') || clean.endsWith('.home.arpa')) return true

  // Private IPv4 ranges
  if (/^10\.\d{1,3}\.\d{1,3}\.\d{1,3}$/.test(clean)) return true
  if (/^172\.(1[6-9]|2\d|3[01])\.\d{1,3}\.\d{1,3}$/.test(clean)) return true
  if (/^192\.168\.\d{1,3}\.\d{1,3}$/.test(clean)) return true

  // IPv6 ULA (fc00::/7) or Link-Local (fe80::/10)
  if (/^(fc|fd)[0-9a-f]{2}:/i.test(clean) || /^fe[89ab][0-9a-f]:/i.test(clean)) return true

  return false
}

/**
 * Check if the hostname matches the user-configured LAN domain (e.g. nas.lan, *.home)
 */
export function isLanDomain(hostname: string, lanDomain?: string): boolean {
  if (!hostname || !lanDomain) return false
  const cleanHost = hostname.trim().toLowerCase()
  const cleanLan = lanDomain.trim().toLowerCase().replace(/^\*\.?/, '')
  if (!cleanLan) return false
  return cleanHost === cleanLan || cleanHost.endsWith(`.${cleanLan}`)
}

/**
 * Convenience function to test if current page entry is LAN
 */
export function isLanEntry(hostname: string, lanDomain?: string): boolean {
  return isLanHost(hostname) || isLanDomain(hostname, lanDomain)
}

/**
 * Check if the given hostname is an IPv6 address
 */
export function isIPv6Host(hostname: string): boolean {
  if (!hostname) return false
  return hostname.includes(':') || hostname.startsWith('[') || /^([0-9a-fA-F]{1,4}:){2,}/.test(hostname)
}

/**
 * Check if an IP address string is IPv6 (not IPv4-mapped)
 */
export function isIPv6Address(ip?: string): boolean {
  if (!ip) return false
  const v = ip.trim().replace(/^\[|\]$/g, '')
  if (!v.includes(':')) return false
  if (/^::ffff:\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$/i.test(v)) return false
  return /^[0-9a-fA-F:]+$/.test(v)
}

/**
 * Checks if a visitor IP matches user-configured LAN IP whitelist
 * Supports exact IP match, /64 IPv6 prefix match, and /24 IPv4 subnet match
 */
export function isIpInWhitelist(clientIp?: string, whitelist?: string): boolean {
  if (!clientIp || !whitelist) return false
  const cleanClient = clientIp.trim().toLowerCase().replace(/^\[|\]$/g, '')
  if (!cleanClient) return false

  const entries = whitelist
    .split(/[,;\n\r]+/)
    .map((s) => s.trim().toLowerCase().replace(/^\[|\]$/g, ''))
    .filter(Boolean)

  for (const entry of entries) {
    if (entry === cleanClient) return true

    // IPv6 prefix match (e.g. 2409:8a00:1234:5678 or with ::/64)
    if (entry.includes(':') && cleanClient.includes(':')) {
      const cleanEntry = entry.replace(/\/64$/, '').replace(/::.*$/, '')
      if (cleanEntry && cleanClient.startsWith(cleanEntry)) {
        return true
      }
    }

    // IPv4 subnet match (e.g. 192.168.1.0/24 or 192.168.1.)
    if (entry.includes('.') && cleanClient.includes('.')) {
      const cleanEntry = entry.replace(/\/24$/, '').replace(/\.0$/, '')
      if (cleanEntry && cleanClient.startsWith(cleanEntry)) {
        return true
      }
    }
  }
  return false
}

const IPV6_CACHE_KEY = 'smartpanel_ipv6_probe'
const TTL_POSITIVE_MS = 10 * 60 * 1000 // 10 minutes for confirmed positive IPv6
const TTL_NEGATIVE_MS = 30 * 1000      // 30 seconds for negative/unsupported

function readIPv6Cache(): boolean | null {
  try {
    const raw = sessionStorage.getItem(IPV6_CACHE_KEY)
    if (!raw) return null
    const [flag, tsStr] = raw.split(':')
    if (flag !== '0' && flag !== '1') return null
    const ts = Number(tsStr)
    if (!Number.isFinite(ts)) return null
    const ttl = flag === '1' ? TTL_POSITIVE_MS : TTL_NEGATIVE_MS
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
    // sessionStorage might be restricted
  }
}

/**
 * Probes a single endpoint via fetch with CORS.
 * Returns true ONLY if request succeeded and returned a valid non-5xx response.
 */
async function probeSingleFetch(url: string, timeoutMs: number): Promise<boolean> {
  const controller = new AbortController()
  const timer = window.setTimeout(() => controller.abort(), timeoutMs)
  try {
    const res = await fetch(url, {
      method: 'GET',
      mode: 'cors',
      cache: 'no-store',
      credentials: 'omit',
      signal: controller.signal,
    })
    // 502/503/504 indicates a reverse proxy error page, not a valid IPv6 backend
    if (res.status >= 500) {
      return false
    }
    return res.ok || res.status === 401 || res.status === 403
  } catch {
    return false
  } finally {
    clearTimeout(timer)
  }
}

export function useIPv6Probe() {
  function setRoutePreference(mode: 'auto' | 'lan' | 'ipv6' | 'domain') {
    routePreference.value = mode
    localStorage.setItem('smartpanel_route_preference', mode)
  }

  function setIPv6Result(available: boolean): boolean {
    isIPv6Available.value = available
    writeIPv6Cache(available)
    return available
  }

  /**
   * Probe client IPv6 capability.
   * Priority:
   * 1. Direct IPv6 access (hostname is IPv6 or v6 domain)
   * 2. Backend verified direct IPv6 connection (directIPv6)
   * 3. Backend observed visitor address is IPv6 (clientHasIPv6 from CF-Connecting-IP)
   * 4. Session cache
   * 5. Active browser probe using fetch
   */
  async function probeIPv6(
    customProbeUrl?: string,
    timeoutMs: number = 2500,
    opts: {
      clientIP?: string
      directIPv6?: boolean
      clientHasIPv6?: boolean
      viaCloudflare?: boolean
      v6domain?: string
      force?: boolean
    } = {}
  ): Promise<boolean> {
    // 1. Current URL is already IPv6
    if (isIPv6Host(window.location.hostname)) {
      lastProbeDetail.value = '当前直接通过 IPv6 地址访问，IPv6 可用'
      return setIPv6Result(true)
    }

    // 2. Direct IPv6 connection confirmed by backend
    if (opts.directIPv6) {
      lastProbeDetail.value = '页面经由 IPv6 直连服务到达，IPv6 可用'
      return setIPv6Result(true)
    }

    // 3. Visitor public address is confirmed IPv6
    if (opts.clientHasIPv6) {
      lastProbeDetail.value = '后端检测到客户端公网地址为 IPv6，确认支持 IPv6 直连'
      return setIPv6Result(true)
    }

    // 4. Session cache
    if (!opts.force) {
      const cached = readIPv6Cache()
      if (cached !== null) {
        isIPv6Available.value = cached
        lastProbeDetail.value = `读取客户端 IPv6 探活缓存: ${cached ? '支持 IPv6' : '不支持 IPv6'}`
        return cached
      }
    }

    isProbing.value = true

    // 5. Active browser probe targets
    const targets: string[] = []

    // If NAS v6domain is configured (e.g. v6.yourdomain.com:5666)
    if (opts.v6domain) {
      const clean = opts.v6domain.trim()
      if (clean) {
        const proto = window.location.protocol === 'https:' ? 'https:' : 'http:'
        const v6Host = clean.startsWith('http://') || clean.startsWith('https://') ? clean : `${proto}//${clean}`
        targets.push(`${v6Host.replace(/\/+$/, '')}/api/health/ping`)
      }
    }

    if (customProbeUrl) {
      targets.push(customProbeUrl)
    }

    // Public IPv6-only CORS-enabled endpoints
    targets.push('https://api6.ipify.org?format=json')
    targets.push('https://v6.ident.me')
    targets.push('https://ipv6.icanhazip.com')

    try {
      // Race: whichever endpoint succeeds first wins; if all fail, return false
      const ok = await new Promise<boolean>((resolve) => {
        let resolved = false
        let remaining = targets.length

        for (const target of targets) {
          probeSingleFetch(target, timeoutMs).then((result) => {
            if (resolved) return
            if (result) {
              resolved = true
              resolve(true)
            } else {
              remaining--
              if (remaining <= 0) {
                resolved = true
                resolve(false)
              }
            }
          })
        }
      })

      lastProbeDetail.value = ok
        ? '客户端 IPv6 端点探测成功，确认支持 IPv6 直连'
        : '客户端 IPv6 端点均无法连通，判定当前网络不支持 IPv6 直连'
      return setIPv6Result(ok)
    } catch (err: any) {
      lastProbeDetail.value = '客户端 IPv6 探针执行异常: ' + (err?.message || '不可达')
      return setIPv6Result(false)
    } finally {
      isProbing.value = false
    }
  }

  /**
   * Probe LAN connectivity
   */
  async function probeLanConnectivity(hostLanIps?: string[], port: string = '5050', timeoutMs: number = 400): Promise<boolean> {
    if (isLanHost(window.location.hostname)) {
      isLanAvailable.value = true
      return true
    }

    if (!hostLanIps || hostLanIps.length === 0) {
      return false
    }

    // Modern browsers block HTTP fetch from HTTPS pages (Mixed Content)
    if (window.location.protocol === 'https:') {
      return false
    }

    const testIp = hostLanIps[0]
    const testUrl = `http://${testIp}:${port}/api/health/ping`

    try {
      const controller = new AbortController()
      const timer = window.setTimeout(() => controller.abort(), timeoutMs)
      const res = await fetch(testUrl, { signal: controller.signal })
      clearTimeout(timer)
      if (res.ok) {
        isLanAvailable.value = true
        return true
      }
      return false
    } catch {
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
    lastProbeDetail,
    routePreference,
    setRoutePreference,
    probeIPv6,
    probeLanConnectivity,
    clearProbeCache,
  }
}
