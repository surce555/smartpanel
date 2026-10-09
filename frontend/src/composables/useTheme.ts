import { ref, watch, onMounted } from 'vue'

export type ThemeMode = 'light' | 'dark' | 'system'

const currentTheme = ref<ThemeMode>('system')
const isDark = ref<boolean>(false)

export function useTheme() {
  function applyTheme(mode: ThemeMode) {
    currentTheme.value = mode
    localStorage.setItem('smartpanel_theme', mode)

    const isSystemDark = window.matchMedia('(prefers-color-scheme: dark)').matches
    const shouldBeDark = mode === 'dark' || (mode === 'system' && isSystemDark)

    isDark.value = shouldBeDark
    if (shouldBeDark) {
      document.documentElement.classList.add('dark')
    } else {
      document.documentElement.classList.remove('dark')
    }
  }

  function initTheme() {
    const saved = (localStorage.getItem('smartpanel_theme') as ThemeMode) || 'system'
    applyTheme(saved)

    // Listen to OS scheme changes
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', (e) => {
      if (currentTheme.value === 'system') {
        applyTheme('system')
      }
    })
  }

  function injectCustomCSS(css: string) {
    let styleEl = document.getElementById('smartpanel-custom-css')
    if (!styleEl) {
      styleEl = document.createElement('style')
      styleEl.id = 'smartpanel-custom-css'
      document.head.appendChild(styleEl)
    }
    styleEl.textContent = css || ''
  }

  function injectCustomJS(jsCode: string) {
    const existing = document.getElementById('smartpanel-custom-js')
    if (existing) {
      existing.remove()
    }

    if (!jsCode || !jsCode.trim()) return

    try {
      const scriptEl = document.createElement('script')
      scriptEl.id = 'smartpanel-custom-js'
      scriptEl.type = 'text/javascript'
      scriptEl.textContent = jsCode
      document.body.appendChild(scriptEl)
    } catch (err) {
      console.error('Failed to execute custom JS:', err)
    }
  }

  function applyFontSettings(fontFamily?: string, fontSize?: string) {
    if (fontFamily) {
      document.documentElement.style.setProperty('--font-main', fontFamily)
      document.body.style.fontFamily = `var(--font-main), system-ui, -apple-system, sans-serif`
    }
    if (fontSize) {
      document.documentElement.style.fontSize = fontSize
    }
  }

  return {
    currentTheme,
    isDark,
    applyTheme,
    initTheme,
    injectCustomCSS,
    injectCustomJS,
    applyFontSettings,
  }
}
