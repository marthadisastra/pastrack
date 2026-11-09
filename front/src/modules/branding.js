import { settingsState } from './state'
export function applyBranding() {
  const value = settingsState.value.branding
  document.documentElement.style.setProperty('--brand-accent', value.color)
  const theme = document.querySelector('meta[name="theme-color"]')
  if (theme) theme.content = value.color
  let favicon = document.querySelector('link[rel="icon"]')
  if (value.favicon) {
    if (!favicon) {
      favicon = document.createElement('link')
      favicon.rel = 'icon'
      favicon.dataset.settings = 'true'
      document.head.append(favicon)
    }
    favicon.href = value.favicon
  } else if (favicon?.dataset.settings) favicon.remove()
  let touchIcon = document.querySelector('link[rel="apple-touch-icon"]')
  if (value.touchIcon) {
    if (!touchIcon) {
      touchIcon = document.createElement('link')
      touchIcon.rel = 'apple-touch-icon'
      touchIcon.dataset.settings = 'true'
      document.head.append(touchIcon)
    }
    touchIcon.href = value.touchIcon
  } else if (touchIcon?.dataset.settings) touchIcon.remove()
}
export function makeWebManifest(branding) {
  return {
    name: `${branding.name} ${branding.tagline}`,
    short_name: branding.name,
    display: 'standalone',
    background_color: '#f8fafc',
    theme_color: branding.color,
    scope: import.meta.env.BASE_URL,
    start_url: `${import.meta.env.BASE_URL}#/dashboard`,
    icons: branding.favicon
      ? [
          {
            src: branding.favicon,
            sizes: 'any',
            type: branding.favicon.match(/^data:([^;]+);/)?.[1] || 'image/png',
          },
        ]
      : [],
  }
}
export function downloadJson(filename, value) {
  const url = URL.createObjectURL(
      new Blob([JSON.stringify(value, null, 2)], { type: 'application/json' }),
    ),
    anchor = document.createElement('a')
  anchor.href = url
  anchor.download = filename
  anchor.click()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}
