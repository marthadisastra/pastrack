import { request } from './http'
import { authBaseURL, authDemo } from './auth'
// Dashboard integration is independent of other modules' demo adapters.
export const dashboardDemo = import.meta.env.VITE_DASHBOARD_MODE
  ? import.meta.env.VITE_DASHBOARD_MODE === 'demo'
  : authDemo
export const dashboardSnapshotTransport = import.meta.env.VITE_DASHBOARD_TRANSPORT === 'snapshot'
const nullableNumber = (value) => {
  if (value === null || value === undefined || value === '') return null
  const number = Number(value)
  if (!Number.isFinite(number)) throw new Error('Invalid dashboard numeric measurement.')
  return number
}
export function normalizeDashboardVehicle(row) {
  if (!row?.id || !row.gps || !row.fuel || !row.ignition)
    throw new Error('Invalid dashboard vehicle identity or measurement contract.')
  const gps = { ...row.gps },
    fuel = { ...row.fuel }
  for (const key of ['latitude', 'longitude', 'speedKmh', 'headingDeg', 'satellites', 'hdop'])
    gps[key] = nullableNumber(gps[key])
  for (const key of ['liters', 'percent', 'capacityLiters']) fuel[key] = nullableNumber(fuel[key])
  return { ...row, id: String(row.id), gps, fuel }
}
export function normalizeDashboardSnapshot(response) {
  if (
    response?.meta?.schemaVersion !== 1 ||
    !Array.isArray(response?.data?.vehicles) ||
    !response.data.summary
  )
    throw new Error('Unsupported dashboard data contract. Retry or contact your administrator.')
  const ids = new Set()
  const vehicles = response.data.vehicles.map((row) => {
    if (!row.id || ids.has(String(row.id)))
      throw new Error('Invalid dashboard vehicle identity or measurement contract.')
    ids.add(String(row.id))
    return normalizeDashboardVehicle(row)
  })
  return { ...response, data: { ...response.data, vehicles } }
}
export function dashboardSocketURL() {
  if (import.meta.env.VITE_DASHBOARD_WS_URL) return import.meta.env.VITE_DASHBOARD_WS_URL
  const base = new URL(authBaseURL, window.location.origin)
  base.protocol = base.protocol === 'https:' ? 'wss:' : 'ws:'
  base.pathname = `${base.pathname.replace(/\/$/, '')}/ws`
  base.search = ''
  base.hash = ''
  return base.toString()
}
export const dashboardService = {
  async snapshot(signal) {
    return normalizeDashboardSnapshot(
      await request('/dashboard/snapshot', { baseURL: authBaseURL, signal }),
    )
  },
  async fuelHistory(vehicleId, range, signal) {
    const response = await request(
      `/dashboard/fuel-history?vehicleId=${encodeURIComponent(vehicleId)}&range=${encodeURIComponent(range)}`,
      { baseURL: authBaseURL, signal },
    )
    if (response?.meta?.schemaVersion !== 1 || !Array.isArray(response?.data?.points))
      throw new Error('Unsupported fuel history contract.')
    return {
      ...response,
      data: {
        ...response.data,
        points: response.data.points.map((point) => ({
          ...point,
          sampleId: String(point.sampleId),
          liters: nullableNumber(point.liters),
          percent: nullableNumber(point.percent),
        })),
      },
    }
  },
}
