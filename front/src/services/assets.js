import { request } from './http'
import { authBaseURL, authDemo } from './auth'
import { fleetService } from './fleet'

const demoMode = import.meta.env.VITE_ASSETS_MODE === 'demo' || authDemo
export const assetsDemoMode = demoMode
const deletedAssetKey = 'pastrack.deleted-assets.v1'
const deletedAssetIds = new Set(
  JSON.parse(localStorage.getItem(deletedAssetKey) || '[]').map((id) => Number(id)),
)

function unwrap(response) {
  const rows = Array.isArray(response) ? response : response?.data
  if (!Array.isArray(rows)) throw new Error('Invalid vehicle response from backend.')
  return rows
}

function normalize(row) {
  return {
    ...row,
    id: String(row.vehicle_code || row.id),
    backendId: Number(row.id),
    plate: row.plate_number || '',
    model: row.vehicle_name || '',
    type: row.vehicle_type || row.type || 'Vehicle',
    vin: row.vin || '',
    capacity: Number(row.tank_capacity_l || 0),
    imei: row.device_identifier || '',
    deviceId: row.device_id ? Number(row.device_id) : null,
    state: String(row.movement_status || 'UNKNOWN').toLowerCase(),
    status: row.movement_status || 'UNKNOWN',
    tone: row.movement_status === 'MOVING' ? 'green' : 'slate',
    csq: row.rssi ?? null,
    fuel: row.fuel_liters == null ? null : Number(row.fuel_liters),
    speed: row.speed_kmh == null ? null : Number(row.speed_kmh),
    lat: row.latitude == null ? null : Number(row.latitude),
    lng: row.longitude == null ? null : Number(row.longitude),
    updatedAt: row.device_last_received_at || row.gps_time || null,
    fuelType: '—',
    tankShape: '—',
  }
}

async function listAll(path) {
  const rows = []
  let cursor = ''
  do {
    const query = new URLSearchParams({ limit: '200' })
    if (cursor) query.set('cursor', cursor)
    const response = await request(`${path}?${query}`, { baseURL: authBaseURL })
    rows.push(...unwrap(response))
    cursor = response?.page?.has_more ? String(response.page.next_cursor || '') : ''
    if (response?.page?.has_more && !cursor) throw new Error('Invalid pagination response from backend.')
  } while (cursor)
  return rows
}

export const assetsService = {
  async list() {
    if (demoMode)
      return (await fleetService.listVehicles())
        .map(normalize)
        .filter((asset) => !deletedAssetIds.has(asset.backendId))
    return (await listAll('/vehicles'))
      .filter((row) => !row.deleted_at && !deletedAssetIds.has(Number(row.id)))
      .map(normalize)
  },
  async create(payload) {
    if (demoMode)
      return normalize(
        await fleetService.createVehicle({
          id: payload.code,
          plate: payload.plate,
          model: payload.model,
          capacity: payload.capacity,
          fleet_code: payload.fleetCode,
        }),
      )
    await request('/vehicles/inventory', {
      baseURL: authBaseURL,
      method: 'POST',
      body: {
        vehicle_code: payload.code,
        vehicle_name: payload.model,
        plate_number: payload.plate,
        fleet_code: payload.fleetCode || '',
        tank_capacity_l: Number(payload.capacity),
      },
    })
  },
  async update(asset, payload) {
    if (demoMode) return fleetService.updateVehicle(asset.id, payload)
    return request(`/vehicles/${asset.backendId}/inventory`, {
      baseURL: authBaseURL,
      method: 'PUT',
      body: {
        vehicle_code: payload.code,
        vehicle_name: payload.model,
        plate_number: payload.plate,
        fleet_code: payload.fleetCode || '',
        tank_capacity_l: Number(payload.capacity),
        row_version: Number(asset.row_version),
      },
    })
  },
  async remove(asset) {
    const result = demoMode
      ? await fleetService.deleteVehicle(asset.id)
      : await request(`/vehicles/${asset.backendId}`, {
          baseURL: authBaseURL,
          method: 'DELETE',
          body: {},
        })
    deletedAssetIds.add(asset.backendId)
    localStorage.setItem(deletedAssetKey, JSON.stringify([...deletedAssetIds]))
    return result
  },
  async pair(asset, deviceId, demoIdentifier = '') {
    if (demoMode) return fleetService.pairDevice(asset.id, demoIdentifier)
    return request(`/vehicles/${asset.backendId}/assignments`, {
      baseURL: authBaseURL,
      method: 'POST',
      body: { device_id: deviceId, starts_at: new Date().toISOString() },
    })
  },
  async devices() {
    if (demoMode) return []
    return (await listAll('/devices')).map((device) => ({
      id: Number(device.id),
      identifier: device.identifier || '',
      model: device.model || '',
      vehicleId: device.vehicle_id ? Number(device.vehicle_id) : null,
      active: Boolean(device.is_active),
    }))
  },
}
