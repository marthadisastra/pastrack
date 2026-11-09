import { request } from './http'
import { authBaseURL } from './auth'
export const scopeService = {
  async load() {
    return (await request('/dashboard/scopes', { baseURL: authBaseURL })).data
  },
  async save(userId, vehicleIds) {
    return request('/dashboard/scopes', {
      baseURL: authBaseURL,
      method: 'PUT',
      body: { userId, vehicleIds },
    })
  },
}
