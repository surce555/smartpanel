import { defineStore } from 'pinia'
import { ref } from 'vue'
import { useApi } from '@/composables/useApi'
import { useTheme } from '@/composables/useTheme'

export interface NetworkInfo {
  current_ipv6: string
  detected_host_ipv6?: string
  domain: string
  v6domain: string
  ddns_enabled: boolean
  ddns_interval_minutes: number
  ddns_status: string
  last_ipv6_check: string
  has_cf_tunnel?: boolean
  cf_zone_id?: string
  cf_record_id?: string
  has_api_token: boolean
}

export const useSettingsStore = defineStore('settings', () => {
  const theme = useTheme()

  const requireLogin = ref<boolean>(false)
  const themeMode = ref<string>('system')
  const cardStyle = ref<string>('glass')
  const cardBorderRadius = ref<number>(12)
  const cardShadow = ref<string>('md')
  const iconSize = ref<string>('medium')
  const gridColsDesktop = ref<number>(6)
  const gridColsTablet = ref<number>(3)
  const gridColsMobile = ref<number>(2)

  const wallpaperType = ref<string>('upload')
  const wallpaperBlur = ref<number>(0)
  const wallpaperMask = ref<number>(0)
  const wallpaperInterval = ref<number>(60)
  const wallpaperUrl = ref<string>('')

  const loginWallpaperType = ref<string>('follow')
  const loginWallpaperUrl = ref<string>('')
  const loginWallpaperBlur = ref<number>(0)
  const loginWallpaperMask = ref<number>(0)

  const textOpacity = ref<number>(
    parseInt(localStorage.getItem('smartpanel_text_opacity') || '100', 10) || 100
  )
  const contentTopOffset = ref<number>(
    parseInt(localStorage.getItem('smartpanel_content_top_offset') || '6', 10) || 6
  )

  const fontFamily = ref<string>('')
  const fontSize = ref<string>('16px')

  const searchEngine = ref<string>('bing')
  const searchCustomUrl = ref<string>('https://www.google.com/search?q=%s')

  const footerText = ref<string>('SmartPanel - NAS Navigation Dashboard')
  const showClock = ref<boolean>(true)
  const clockFormat24 = ref<boolean>(true)

  const customCss = ref<string>('')
  const customJs = ref<string>('')

  const networkInfo = ref<NetworkInfo>({
    current_ipv6: '',
    domain: '',
    v6domain: '',
    ddns_enabled: false,
    ddns_interval_minutes: 5,
    ddns_status: 'idle',
    last_ipv6_check: '',
    has_api_token: false,
  })

  let sseEventSource: EventSource | null = null

  async function fetchSettings() {
    const api = useApi()
    try {
      const resp = await api.get('/settings')
      const s = resp.data

      if (s.theme_mode) themeMode.value = s.theme_mode
      if (s.card_style) cardStyle.value = s.card_style
      if (s.card_border_radius) cardBorderRadius.value = parseInt(s.card_border_radius, 10) || 12
      if (s.card_shadow) cardShadow.value = s.card_shadow
      if (s.icon_size) iconSize.value = s.icon_size
      if (s.grid_cols_desktop) gridColsDesktop.value = parseInt(s.grid_cols_desktop, 10) || 6
      if (s.grid_cols_tablet) gridColsTablet.value = parseInt(s.grid_cols_tablet, 10) || 3
      if (s.grid_cols_mobile) gridColsMobile.value = parseInt(s.grid_cols_mobile, 10) || 2

      if (s.wallpaper_type) wallpaperType.value = s.wallpaper_type
      if (s.wallpaper_blur !== undefined) wallpaperBlur.value = parseInt(s.wallpaper_blur, 10) || 0
      if (s.wallpaper_mask !== undefined) {
        const val = parseInt(s.wallpaper_mask, 10)
        wallpaperMask.value = isNaN(val) ? 0 : val
      }
      if (s.wallpaper_interval) wallpaperInterval.value = parseInt(s.wallpaper_interval, 10) || 60
      if (s.wallpaper_url !== undefined) {
        wallpaperUrl.value = s.wallpaper_url
        if (s.wallpaper_url) localStorage.setItem('smartpanel_wallpaper', s.wallpaper_url)
      }

      if (s.login_wallpaper_type) loginWallpaperType.value = s.login_wallpaper_type
      if (s.login_wallpaper_url !== undefined) {
        loginWallpaperUrl.value = s.login_wallpaper_url
        if (s.login_wallpaper_url) localStorage.setItem('smartpanel_login_wallpaper', s.login_wallpaper_url)
      }
      if (s.login_wallpaper_blur !== undefined) loginWallpaperBlur.value = parseInt(s.login_wallpaper_blur, 10) || 0
      if (s.login_wallpaper_mask !== undefined) {
        const val = parseInt(s.login_wallpaper_mask, 10)
        loginWallpaperMask.value = isNaN(val) ? 0 : val
      }

      if (s.text_opacity !== undefined) {
        const val = parseInt(s.text_opacity, 10)
        textOpacity.value = isNaN(val) ? 100 : val
        localStorage.setItem('smartpanel_text_opacity', String(textOpacity.value))
      }
      if (s.content_top_offset !== undefined) {
        const val = parseInt(s.content_top_offset, 10)
        contentTopOffset.value = isNaN(val) ? 6 : val
        localStorage.setItem('smartpanel_content_top_offset', String(contentTopOffset.value))
      }

      if (s.font_family) fontFamily.value = s.font_family
      if (s.font_size) fontSize.value = s.font_size

      if (s.search_default_engine) searchEngine.value = s.search_default_engine
      if (s.search_custom_url) searchCustomUrl.value = s.search_custom_url

      if (s.footer_text) footerText.value = s.footer_text
      if (s.show_clock) showClock.value = s.show_clock === 'true'
      if (s.clock_format_24) clockFormat24.value = s.clock_format_24 === 'true'
      if (s.require_login !== undefined) requireLogin.value = s.require_login === 'true'

      if (s.custom_css) {
        customCss.value = s.custom_css
        theme.injectCustomCSS(s.custom_css)
      }
      if (s.custom_js) {
        customJs.value = s.custom_js
        theme.injectCustomJS(s.custom_js)
      }

      // Apply root styles
      document.documentElement.style.setProperty('--card-radius', `${cardBorderRadius.value}px`)
      theme.applyTheme(themeMode.value as any)
      theme.applyFontSettings(fontFamily.value, fontSize.value)
    } catch (e) {
      console.warn('Could not fetch settings from backend:', e)
    }
  }

  async function fetchNetworkInfo() {
    const api = useApi()
    try {
      const resp = await api.get('/system/network')
      networkInfo.value = resp.data
    } catch (e) {
      console.warn('Could not fetch network info:', e)
    }
  }

  function setupSSE() {
    if (sseEventSource) {
      sseEventSource.close()
    }

    try {
      sseEventSource = new EventSource('/api/system/network/stream')
      sseEventSource.addEventListener('network_update', (e: MessageEvent) => {
        try {
          const data = JSON.parse(e.data)
          networkInfo.value = { ...networkInfo.value, ...data }
        } catch (err) {
          console.error('Failed to parse SSE network update:', err)
        }
      })
      sseEventSource.onerror = () => {
        // EventSource will auto-reconnect
      }
    } catch (err) {
      console.warn('SSE not supported or failed to connect:', err)
    }
  }

  async function saveSettings(payload: Record<string, any>) {
    const api = useApi()
    await api.put('/settings', payload)
    await fetchSettings()
    await fetchNetworkInfo()
  }

  async function triggerNetworkCheck(manualIPv6?: string) {
    const api = useApi()
    await api.post('/system/network/check', { manual_ipv6: manualIPv6 || '' })
  }

  async function applyPresetTheme(preset: 'crystal' | 'glass' | 'solid' | 'minimal') {
    if (preset === 'crystal') {
      await saveSettings({
        card_style: 'transparent',
        card_shadow: 'none',
        wallpaper_blur: '0',
        wallpaper_mask: '0',
        login_wallpaper_blur: '0',
        login_wallpaper_mask: '0',
      })
    } else if (preset === 'glass') {
      await saveSettings({
        card_style: 'glass',
        card_shadow: 'md',
        wallpaper_blur: '0',
        wallpaper_mask: '15',
        login_wallpaper_type: 'follow',
      })
    } else if (preset === 'solid') {
      await saveSettings({
        card_style: 'solid',
        card_shadow: 'sm',
        login_wallpaper_type: 'follow',
      })
    } else if (preset === 'minimal') {
      await saveSettings({
        card_style: 'minimal',
        card_shadow: 'none',
        login_wallpaper_type: 'follow',
      })
    }
  }

  return {
    requireLogin,
    themeMode,
    cardStyle,
    cardBorderRadius,
    cardShadow,
    iconSize,
    gridColsDesktop,
    gridColsTablet,
    gridColsMobile,
    wallpaperType,
    wallpaperBlur,
    wallpaperMask,
    wallpaperInterval,
    wallpaperUrl,
    loginWallpaperType,
    loginWallpaperUrl,
    loginWallpaperBlur,
    loginWallpaperMask,
    textOpacity,
    contentTopOffset,
    fontFamily,
    fontSize,
    searchEngine,
    searchCustomUrl,
    footerText,
    showClock,
    clockFormat24,
    customCss,
    customJs,
    networkInfo,
    fetchSettings,
    fetchNetworkInfo,
    setupSSE,
    saveSettings,
    applyPresetTheme,
    triggerNetworkCheck,
  }
})
