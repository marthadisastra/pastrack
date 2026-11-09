import { request } from '../../services/http'
import { authBaseURL, authDemo } from '../../services/auth'
import { demoOrders } from '../../mocks/runtime'
import { isModuleActive } from '../state'
const clone = (value) => structuredClone(value)
const latency = () => new Promise((resolve) => setTimeout(resolve, 280))
const maintenanceDemoMode = import.meta.env.VITE_MAINTENANCE_MODE === 'demo' || authDemo
const normalize = (row) => ({
  ...row,
  id: String(row.id),
  vehicleId: String(row.vehicleId ?? row.vehicle_id ?? row.vehicle_code ?? ''),
  trigger: Number(row.trigger ?? row.trigger_value ?? 0),
  current: Number(row.current ?? row.current_value ?? 0),
  budget: Number(row.budget ?? 0),
  unit: row.unit || row.trigger_unit || 'km',
  date: row.date || row.scheduled_date,
})
function requireActive() {
  if (!isModuleActive('ext.fleet.maintenance'))
    throw new Error('The maintenance module is disabled.')
}
export const maintenanceService = {
  async list() {
    requireActive()
    if (!maintenanceDemoMode) {
      const response = await request('/maintenance', { baseURL: authBaseURL })
      return (Array.isArray(response) ? response : response?.data || []).map(normalize)
    }
    await latency()
    return clone(demoOrders)
  },
  async create(payload) {
    requireActive()
    if (!maintenanceDemoMode) {
      const response = await request('/maintenance', { baseURL: authBaseURL, method: 'POST', body: payload })
      const id = response?.data?.id
      const rows = await this.list()
      return rows.find((row) => row.id === String(id)) || { ...normalize(payload), id: String(id) }
    }
    await latency()
    const order = {
      ...payload,
      id: `WO-${crypto.randomUUID()}`,
      status: 'Scheduled',
      current: 0,
      bay: 'Pending allocation',
    }
    demoOrders.unshift(order)
    return clone(order)
  },
  async update(id, payload) {
    requireActive()
    if (!maintenanceDemoMode) {
      await request(`/maintenance/${encodeURIComponent(id)}`, { baseURL: authBaseURL, method: 'PATCH', body: payload })
      const rows = await this.list()
      return rows.find((row) => row.id === String(id)) || { id: String(id), ...payload }
    }
    await latency()
    const order = demoOrders.find((order) => order.id === id)
    if (!order) throw new Error('Work order not found.')
    Object.assign(order, payload)
    return clone(order)
  },
  async remove(id) {
    requireActive()
    if (!maintenanceDemoMode) {
      await request(`/maintenance/${encodeURIComponent(id)}`, { baseURL: authBaseURL, method: 'DELETE' })
      return true
    }
    const index = demoOrders.findIndex((order) => order.id === id)
    if (index < 0) throw new Error('Work order not found.')
    demoOrders.splice(index, 1)
    return true
  },
}
