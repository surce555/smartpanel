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

export function useIPv6Probe() {
  function setRoutePreference(mode: 'auto' | 'lan' | 'ipv6' | 'domain') {
    routePreference.value = mode
    localStorage.setItem('smartpanel_route_preference', mode)
  }

  /**
   * Probe IPv6 connectivity using high-speed domestic IPv6 endpoints
   * (e.g. 6.ipw.cn, speed.neu6.edu.cn).
   */
  async function probeIPv6(customProbeUrl?: string, timeoutMs: number = 800): Promise<boolean> {
    // 1. If currently visiting via IPv6 directly, IPv6 is 100% available!
    if (isIPv6Host(window.location.hostname)) {
      isIPv6Available.value = true
      sessionStorage.setItem('smartpanel_ipv6_probe', 'true')
      return true
    }

    // 2. Check sessionStorage cache
    const cached = sessionStorage.getItem('smartpanel_ipv6_probe')
    if (cached !== null) {
      isIPv6Available.value = cached === 'true'
      return isIPv6Available.value
    }

    isProbing.value = true

    // Candidate endpoints that only resolve over IPv6
    const endpoints = [
      customProbeUrl,
      'https://6.ipw.cn/favicon.ico',
      'https://speed.neu6.edu.cn/favicon.ico',
    ].filter(Boolean) as string[]

    const probeSingle = (url: string, timeout: number): Promise<boolean> => {
      return new Promise<boolean>((resolve) => {
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
        }, timeout)

        img.onload = () => {
          cleanup()
          resolve(true)
        }

        img.onerror = () => {
          cleanup()
          // An onerror from an IPv6-only domain means DNS and TCP handshake succeeded
          resolve(true)
        }

        img.src = `${url}?_t=${Date.now()}`
      })
    }

    try {
      // Race or probe first reliable endpoint
      let result = await probeSingle(endpoints[0], timeoutMs)
      if (!result && endpoints.length > 1) {
        result = await probeSingle(endpoints[1], timeoutMs)
      }

      isIPv6Available.value = result
      sessionStorage.setItem('smartpanel_ipv6_probe', String(result))
      return result
    } catch {
      isIPv6Available.value = false
      sessionStorage.setItem('smartpanel_ipv6_probe', 'false')
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
    sessionStorage.removeItem('smartpanel_ipv6_probe')
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
