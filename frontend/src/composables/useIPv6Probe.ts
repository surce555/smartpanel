import { ref } from 'vue'

const isIPv6Available = ref<boolean | null>(null)
const isProbing = ref<boolean>(false)

export function useIPv6Probe() {
  /**
   * Probe IPv6 connectivity with a strict timeout (default 250ms).
   * Result is cached in sessionStorage.
   */
  async function probeIPv6(customProbeUrl?: string, timeoutMs: number = 250): Promise<boolean> {
    // Check sessionStorage cache first
    const cached = sessionStorage.getItem('smartpanel_ipv6_probe')
    if (cached !== null) {
      isIPv6Available.value = cached === 'true'
      return isIPv6Available.value
    }

    isProbing.value = true
    const probeUrl = customProbeUrl || 'https://ipv6.google.com/favicon.ico'

    const probePromise = new Promise<boolean>((resolve) => {
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

      // In some networks, an image error from ipv6-only endpoint still indicates IPv6 handshake succeeded,
      // but timeout definitely indicates unreachable. To be conservative:
      img.onerror = () => {
        cleanup()
        // If the server answered with 404/etc., IPv6 was reached!
        resolve(true)
      }

      // Add cache buster
      img.src = `${probeUrl}?_t=${Date.now()}`
    })

    try {
      const result = await probePromise
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

  function clearProbeCache() {
    sessionStorage.removeItem('smartpanel_ipv6_probe')
    isIPv6Available.value = null
  }

  return {
    isIPv6Available,
    isProbing,
    probeIPv6,
    clearProbeCache,
  }
}
