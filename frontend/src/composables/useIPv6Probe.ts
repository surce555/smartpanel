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

function expandIPv6(ip: string): string[] | null {
  const clean = ip.trim().toLowerCase().replace(/^\[|\]$/g, '')
  if (!clean.includes(':')) return null
  const parts = clean.split('::')
  if (parts.length > 2) return null
  const left = parts[0] ? parts[0].split(':') : []
  const right = parts.length === 2 && parts[1] ? parts[1].split(':') : []
  const missing = 8 - (left.length + right.length)
  if (missing < 0) return null
  const middle = new Array(missing).fill('0000')
  const full = [...left, ...middle, ...right].map((p) => p.padStart(4, '0'))
  return full.length === 8 ? full : null
}

export function isSameSubnetIPv6Str(ip1: string, ip2: string): boolean {
  const exp1 = expandIPv6(ip1)
  const exp2 = expandIPv6(ip2)
  if (!exp1 || !exp2) return false
  return exp1[0] === exp2[0] && exp1[1] === exp2[1] && exp1[2] === exp2[2] && exp1[3] === exp2[3]
}

/**
 * Queries local WebRTC ICE candidates to detect client's real physical IPs without sending STUN requests
 * Completely immune to proxy/VPN redirection and works on HTTPS
 */
export function probeLocalNetworkViaWebRTC(
  nasLanIps?: string[],
  nasIpv6s?: string[],
  timeoutMs = 1200
): Promise<{ isLan: boolean; hasIPv6: boolean; matchedIp?: string; reason?: string }> {
  return new Promise((resolve) => {
    if (typeof window === 'undefined') {
      resolve({ isLan: false, hasIPv6: false })
      return
    }

    const RTCPC = window.RTCPeerConnection || (window as any).webkitRTCPeerConnection
    if (!RTCPC) {
      resolve({ isLan: false, hasIPv6: false })
      return
    }

    let pc: RTCPeerConnection | null = null
    let finished = false
    let detectedIPv6 = false

    const cleanNasV6s = (nasIpv6s || [])
      .map((ip) => (ip || '').trim().toLowerCase().replace(/^\[|\]$/g, ''))
      .filter((ip) => ip && ip.includes(':'))

    const cleanNasLanV4s = (nasLanIps || [])
      .map((ip) => (ip || '').trim())
      .filter((ip) => /^(\d{1,3}\.){3}\d{1,3}$/.test(ip))

    const finish = (result: { isLan: boolean; hasIPv6: boolean; matchedIp?: string; reason?: string }) => {
      if (finished) return
      finished = true
      clearTimeout(timer)
      if (pc) {
        try {
          pc.close()
        } catch {}
        pc = null
      }
      resolve(result)
    }

    const timer = setTimeout(() => {
      finish({ isLan: false, hasIPv6: detectedIPv6 })
    }, timeoutMs)

    try {
      pc = new RTCPC({ iceServers: [] })
      pc.createDataChannel('smartpanel_pna')
      pc.createOffer()
        .then((offer) => pc?.setLocalDescription(offer))
        .catch(() => finish({ isLan: false, hasIPv6: false }))

      pc.onicecandidate = (event) => {
        if (finished || !event || !event.candidate || !event.candidate.candidate) {
          return
        }

        const cand = event.candidate.candidate

        // 1. Look for IPv6 address in candidate
        const v6Match = cand.match(/([0-9a-fA-F]{1,4}:[0-9a-fA-F:]+)/)
        if (v6Match) {
          const clientV6 = v6Match[1].toLowerCase()
          // Global unicast IPv6 (starts with 2xxx or 3xxx)
          if (/^[23][0-9a-f]{3}:/i.test(clientV6)) {
            detectedIPv6 = true
            for (const nasV6 of cleanNasV6s) {
              if (isSameSubnetIPv6Str(clientV6, nasV6)) {
                finish({
                  isLan: true,
                  hasIPv6: true,
                  matchedIp: clientV6,
                  reason: `客户端与 NAS 共享相同 IPv6 /64 子网 (${clientV6})`,
                })
                return
              }
            }
          }
        }

        // 2. Look for IPv4 address in candidate
        const v4Match = cand.match(/\b(192\.168\.\d{1,3}\.\d{1,3}|10\.\d{1,3}\.\d{1,3}\.\d{1,3}|172\.(1[6-9]|2\d|3[01])\.\d{1,3}\.\d{1,3})\b/)
        if (v4Match) {
          const clientV4 = v4Match[1]
          for (const nasV4 of cleanNasLanV4s) {
            const clientSubnet = clientV4.substring(0, clientV4.lastIndexOf('.'))
            const nasSubnet = nasV4.substring(0, nasV4.lastIndexOf('.'))
            if (clientSubnet === nasSubnet) {
              finish({
                isLan: true,
                hasIPv6: detectedIPv6,
                matchedIp: clientV4,
                reason: `客户端与 NAS 处于同一局域网网段 (${clientV4})`,
              })
              return
            }
          }
        }
      }
    } catch {
      finish({ isLan: false, hasIPv6: false })
    }
  })
}

const IPV6_CACHE_KEY = 'smartpanel_ipv6_probe'
const TTL_POSITIVE_MS = 10 * 60 * 1000 // 10 minutes for confirmed positive IPv6
const TTL_NEGATIVE_MS = 15 * 1000      // 15 seconds for negative/unsupported

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
 * Probes an image endpoint.
 * Returns true ONLY if onload fires (proving the image bytes were actually loaded).
 */
function probeImageEndpoint(url: string, timeoutMs: number): Promise<boolean> {
  return new Promise((resolve) => {
    const img = new Image()
    let timer: number | null = null
    const cleanup = () => {
      if (timer) clearTimeout(timer)
      img.onload = null
      img.onerror = null
    }
    timer = window.setTimeout(() => {
      cleanup()
      resolve(false)
    }, timeoutMs)
    img.onload = () => {
      cleanup()
      resolve(true)
    }
    img.onerror = () => {
      cleanup()
      resolve(false)
    }
    img.src = `${url}${url.includes('?') ? '&' : '?'}_t=${Date.now()}`
  })
}

/**
 * Probes a single endpoint via fetch with CORS.
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
   * 4. WebRTC local network detection
   * 5. Session cache
   * 6. Active browser probe (Image onload & CORS fetch across IPv6 endpoints)
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
      currentIPv6?: string
      nasLanIps?: string[]
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

    // 4. Session cache (if not forced)
    if (!opts.force) {
      const cached = readIPv6Cache()
      if (cached !== null) {
        isIPv6Available.value = cached
        lastProbeDetail.value = `读取客户端 IPv6 探活缓存: ${cached ? '支持 IPv6' : '不支持 IPv6'}`
        return cached
      }
    }

    isProbing.value = true

    // 5. Query WebRTC for local network & physical IPv6 stack
    try {
      const webrtc = await probeLocalNetworkViaWebRTC(opts.nasLanIps, [opts.currentIPv6 || ''])
      if (webrtc.isLan) {
        isLanAvailable.value = true
      }
      if (webrtc.hasIPv6) {
        lastProbeDetail.value = webrtc.reason || '客户端已分配公网单播 IPv6 地址，确认支持 IPv6 直连'
        return setIPv6Result(true)
      }
    } catch {}

    // 6. Active probe targets
    const imageTargets: string[] = [
      'http://test6.ustc.edu.cn/favicon.ico',
      'https://speed.neu6.edu.cn/favicon.ico',
      'https://6.ipw.cn/favicon.ico',
    ]

    if (opts.v6domain) {
      const clean = opts.v6domain.trim()
      if (clean) {
        const proto = clean.startsWith('http://') || clean.startsWith('https://') ? '' : `${window.location.protocol}//`
        imageTargets.unshift(`${proto}${clean}/api/health/probe.png`)
      }
    }

    if (customProbeUrl) {
      imageTargets.unshift(customProbeUrl)
    }

    const fetchTargets: string[] = [
      'https://ipv6.test-ipv6.com/images/knob_green.png',
      'https://api6.ipify.org?format=json',
    ]

    try {
      const probePromises = [
        ...imageTargets.map((t) => probeImageEndpoint(t, timeoutMs)),
        ...fetchTargets.map((t) => probeSingleFetch(t, timeoutMs)),
      ]

      const ok = await new Promise<boolean>((resolve) => {
        let remaining = probePromises.length
        let resolved = false
        for (const p of probePromises) {
          p.then((success) => {
            if (resolved) return
            if (success) {
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
        : '客户端 IPv6 端点探测超时，判定当前网络不支持 IPv6'
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
  async function probeLanConnectivity(
    hostLanIps?: string[],
    port: string = '5050',
    timeoutMs: number = 800,
    nasIpv6s?: string[]
  ): Promise<boolean> {
    if (isLanHost(window.location.hostname)) {
      isLanAvailable.value = true
      return true
    }

    // 1. WebRTC probe against host LAN IPs and IPv6
    try {
      const webrtc = await probeLocalNetworkViaWebRTC(hostLanIps, nasIpv6s, timeoutMs)
      if (webrtc.isLan) {
        isLanAvailable.value = true
        return true
      }
    } catch {}

    // 2. Active Image probe to LAN IP
    if (hostLanIps && hostLanIps.length > 0) {
      for (const ip of hostLanIps) {
        const ok = await probeImageEndpoint(`http://${ip}:${port}/api/health/probe.png`, 400)
        if (ok) {
          isLanAvailable.value = true
          return true
        }
      }
    }

    return false
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
