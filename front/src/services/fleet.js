import { request, useMock } from './http'
import { authBaseURL } from './auth'
import { vehicles } from '../mocks/fleet'
import { demoVehicles, demoOrders } from '../mocks/runtime'
import { isModuleActive } from '../modules/state'
// Service contract: list endpoints return arrays; mutation endpoints return the saved entity.
// Demo records live in memory and reset on reload. No operational commands are sent.
const copy = (value) => structuredClone(value)
let sampleSequence = 0
const demoWait = () => new Promise((resolve) => setTimeout(resolve, 280))
function enrich(asset) {
  return {
    heading: 42,
    altitude: 48,
    temperature: 31.4,
    burnRate: 27.8,
    csq: 27,
    satellites: 14,
    hdop: 0.82,
    backupBattery: 4.12,
    drift: 0.2,
    updatedAt: new Date().toISOString(),
    ...asset,
  }
}
for (const asset of demoVehicles) Object.assign(asset, enrich(asset))
export const fleetService = {
  sampleTelemetry() {
    sampleSequence++
    const now = new Date().toISOString(),
      notifications = []
    for (const [index, asset] of demoVehicles.entries()) {
      if (!asset.imei) continue
      const drift = Math.sin(sampleSequence * 0.7 + index)
      if (asset.state === 'moving') {
        asset.speed = Math.max(12, Math.round(asset.speed + drift * 3))
        asset.fuel = Number(Math.max(0, asset.fuel - 0.06 - Math.abs(drift) * 0.04).toFixed(2))
        asset.odo = Number((asset.odo + asset.speed / 900).toFixed(2))
        asset.lat = Number((asset.lat + 0.00002).toFixed(6))
        asset.lng = Number((asset.lng + 0.00003).toFixed(6))
        asset.x = Math.min(85, Math.max(15, asset.x + drift * 0.2))
        asset.y = Math.min(80, Math.max(20, asset.y + drift * 0.15))
      }
      asset.battery = Number(
        (asset.ignition ? (asset.type === 'Van' ? 13.8 : 27.8) + drift * 0.12 : 27.8).toFixed(2),
      )
      asset.rssi = Math.round(-66 + drift * 3)
      asset.temperature = Number((31.4 + drift * 0.2).toFixed(1))
      asset.heading = Math.round((asset.heading + (asset.speed ? 1 : 0)) % 360)
      asset.burnRate = Number((27.8 + drift * 0.3).toFixed(1))
      asset.csq = Math.round(27 + drift * 2)
      asset.hdop = Number((0.82 + drift * 0.03).toFixed(2))
      asset.updatedAt = now
    }
    if (sampleSequence === 5) {
      const probe = demoVehicles.find((v) => v.id === 'FLATBED-11')
      if (probe) probe.rssi = -86
      notifications.push({
        id: 'demo:ble-warning',
        title: 'BLE signal weakened · FLATBED-11',
        message: 'Signal changed in the demo feed. Inspect the probe link before dispatch.',
        severity: 'warning',
        vehicleId: 'FLATBED-11',
        target: '/tracking?unit=FLATBED-11',
        createdAt: now,
      })
    }
    for (const order of isModuleActive('ext.fleet.maintenance') ? demoOrders : []) {
      const asset = demoVehicles.find((v) => v.id === order.vehicleId)
      if (asset && order.unit === 'km' && order.status !== 'Completed')
        order.current = Math.round(asset.odo)
    }
    return {
      type: 'snapshot',
      data: {
        vehicles: copy(demoVehicles),
        orders: isModuleActive('ext.fleet.maintenance') ? copy(demoOrders) : undefined,
        notifications,
      },
    }
  },
  async listVehicles() {
    if (!useMock) {
      const response = await request('/vehicles', { baseURL: authBaseURL })
      const rows = Array.isArray(response) ? response : response?.data
      if (!Array.isArray(rows)) throw new Error('Invalid vehicle response from backend.')
      return rows.map((row) => ({
        id: String(row.id),
        plate: row.plate_number || row.plate || '',
        model: row.vehicle_name || row.model || '',
        type: row.vehicle_type || row.type || 'Vehicle',
        year: row.production_year || row.year || null,
        vin: row.vin || '',
        capacity: Number(row.tank_capacity_l ?? row.capacity ?? 0),
        fuelType: row.fuel_type || 'Diesel B35',
        tankShape: row.tank_shape || 'Cylindrical',
        imei: row.imei || row.device_identifier || null,
        state: String(row.movement_status || row.state || 'unknown').toLowerCase(),
        status: String(row.connectivity || row.movement_status || 'UNKNOWN'),
        tone: row.connectivity === 'ONLINE' ? 'green' : 'slate',
        csq: row.csq ?? null,
        rssi: row.rssi ?? null,
        fuel: Number(row.fuel_liters ?? 0),
        lat: row.latitude ?? null,
        lng: row.longitude ?? null,
        speed: Number(row.speed_kmh ?? 0),
        battery: row.battery_voltage ?? row.main_battery_voltage ?? null,
        ignition: row.ignition_status === 'ON',
        probeBattery: row.sensor_battery_voltage ?? null,
        updatedAt: row.received_at || row.device_last_received_at || null,
      }))
    }
    await demoWait()
    return copy(demoVehicles)
  },
  async createVehicle(payload) {
    if (!useMock)
      return request('/vehicles', { baseURL: authBaseURL, method: 'POST', body: payload })
    if (demoVehicles.some((v) => v.id.toLowerCase() === payload.id.toLowerCase()))
      throw new Error('Asset ID already exists.')
    await demoWait()
    const asset = enrich({ ...copy(vehicles[5]), ...payload })
    demoVehicles.push(asset)
    return copy(asset)
  },
  async updateVehicle(id, payload) {
    if (!useMock) throw new Error('Vehicle inventory updates require the live inventory endpoint.')
    const asset = demoVehicles.find((vehicle) => vehicle.id === id)
    if (!asset) throw new Error('Asset not found.')
    await demoWait()
    Object.assign(asset, {
      id: payload.code,
      plate: payload.plate,
      model: payload.model,
      fleet_code: payload.fleetCode,
      capacity: Number(payload.capacity),
    })
    return copy(asset)
  },
  async deleteVehicle(id) {
    const index = demoVehicles.findIndex((vehicle) => vehicle.id === id)
    if (index < 0) throw new Error('Asset not found.')
    await demoWait()
    return copy(demoVehicles.splice(index, 1)[0])
  },
  async pairDevice(id, imei) {
    if (!useMock)
      return request(`/vehicles/${encodeURIComponent(id)}/device`, {
        baseURL: authBaseURL,
        method: 'PUT',
        body: { imei },
      })
    await demoWait()
    if (demoVehicles.some((v) => v.imei === imei && v.id !== id))
      throw new Error('This IMEI is already paired to another asset.')
    const asset = demoVehicles.find((v) => v.id === id)
    Object.assign(asset, { imei, status: 'Device Paired · Awaiting Telemetry', tone: 'blue' })
    return copy(asset)
  },
}
