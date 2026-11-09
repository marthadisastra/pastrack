import { request } from './http'
// Authentication is independently enabled while fleet/module integration proceeds in later stages.
export const authDemo = import.meta.env.VITE_AUTH_MODE === 'demo'
export const authBaseURL = (import.meta.env.VITE_AUTH_API_BASE_URL || '/api/v1').replace(/\/$/, '')
const authRequest = (path, options = {}) =>
  request(`/auth${path}`, { ...options, baseURL: authBaseURL, unauthorized: false })
export const authService = {
  async login(payload) {
    if (!authDemo) {
      await authRequest('/login', { method: 'POST', body: payload })
      return this.me()
    }
    if (payload.password === 'fail')
      throw new Error('Demo invalid credentials. Use another password to preview the dashboard.')
    return { name: 'NOC Dispatcher', email: payload.email, role: 'admin', demo: true }
  },
  async me() {
    if (authDemo) return null
    const response = await authRequest('/me')
    return { ...response.data, name: response.data.full_name }
  },
  async logout() {
    if (!authDemo) await authRequest('/logout', { method: 'POST', body: {} })
  },
  async forgotPassword(email) {
    return authDemo
      ? { demo: true }
      : authRequest('/forgot-password', { method: 'POST', body: { email } })
  },
  async resetPassword(payload) {
    return authDemo
      ? { demo: true }
      : authRequest('/reset-password', { method: 'POST', body: payload })
  },
}
