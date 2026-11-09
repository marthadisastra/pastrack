import { request } from './http'
import { authBaseURL } from './auth'

export const notificationService = {
  markRead(id) {
    return request(`/notifications/${encodeURIComponent(id)}/read`, {
      baseURL: authBaseURL,
      method: 'POST',
      body: {},
    })
  },
  markAllRead() {
    return request('/notifications/read-all', { baseURL: authBaseURL, method: 'POST', body: {} })
  },
}
