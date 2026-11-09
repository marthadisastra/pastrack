import { createApp } from 'vue'
import App from './App.vue'
import router from './router'
import './assets/main.css'
import { watch } from 'vue'
import { settingsState } from './modules/state'
import { applyBranding } from './modules/branding'
import { useAuth } from './composables/useAuth'
window.addEventListener('pastrack:session-expired', () => {
  const auth = useAuth()
  if (auth.authDemo) return
  auth.clearSession()
  if (router.currentRoute.value.meta.requiresAuth)
    router.replace({ name: 'login', query: { redirect: router.currentRoute.value.fullPath } })
})
watch(() => settingsState.value.branding, applyBranding, { immediate: true })
createApp(App).use(router).mount('#app')
