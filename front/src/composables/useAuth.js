import { ref } from 'vue'
import { authService, authDemo } from '../services/auth'
const user = ref(null),
  ready = ref(authDemo),
  sessionError = ref('')
let pending
export function useAuth() {
  async function restore(force = false) {
    if (authDemo) return true
    if (pending) return pending
    if (ready.value && !force) return Boolean(user.value)
    pending = authService
      .me()
      .then((identity) => {
        user.value = identity
        ready.value = true
        sessionError.value = ''
        return true
      })
      .catch((error) => {
        if (error.status === 401) {
          user.value = null
          ready.value = true
          sessionError.value = ''
        } else {
          sessionError.value = 'Unable to verify your session. Check the connection and try again.'
        }
        return Boolean(user.value)
      })
      .finally(() => {
        pending = null
      })
    return pending
  }
  async function signIn(payload) {
    user.value = await authService.login(payload)
    ready.value = true
    sessionError.value = ''
    return user.value
  }
  async function signOut() {
    await authService.logout()
    clearSession()
  }
  function clearSession() {
    user.value = null
    ready.value = true
  }
  function watchSession(onExpired) {
    if (authDemo) return () => {}
    let disposed = false
    const check = async () => {
      if (disposed || document.hidden || !navigator.onLine) return
      if (!(await restore(true)) && !disposed && ready.value) onExpired()
    }
    const timer = setInterval(check, 60000)
    window.addEventListener('focus', check)
    document.addEventListener('visibilitychange', check)
    return () => {
      disposed = true
      clearInterval(timer)
      window.removeEventListener('focus', check)
      document.removeEventListener('visibilitychange', check)
    }
  }
  return {
    user,
    ready,
    sessionError,
    restore,
    signIn,
    signOut,
    clearSession,
    watchSession,
    authDemo,
  }
}
