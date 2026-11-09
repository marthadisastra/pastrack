import { computed, ref, watch } from 'vue'
import { isModuleActive } from '../modules/state'
import { maintenanceService } from '../modules/maintenance/service'
import { fleetService } from '../services/fleet'
import { connectRealtime, mergeSection } from '../services/realtime'
import { useNotifications } from './useNotifications'
const vehicles = ref([]),
  orders = ref([]),
  loading = ref(false),
  error = ref('')
const connection = ref({ status: 'idle', detail: 'Not connected', attempt: 0 })
const lastUpdated = ref(null),
  now = ref(Date.now()),
  paused = ref(false),
  samples = ref({})
const staleAfter = Math.max(10000, Number(import.meta.env.VITE_STALE_AFTER_MS) || 60000)
let pending,
  stream,
  running = false,
  generation = 0,
  clock,
  sections = {}
const notifications = useNotifications()
function mergeRecords(target, records, replace = false) {
  const prior = new Map(target.value.map((item) => [String(item.id), item]))
  const next = []
  for (const record of records) {
    if (record.id === undefined) continue
    const id = String(record.id),
      existing = prior.get(id)
    if (existing) {
      Object.assign(existing, record, { id })
      next.push(existing)
    } else next.push({ ...record, id })
  }
  target.value = replace
    ? next
    : [...target.value, ...next.filter((row) => !prior.has(String(row.id)))]
}
function recordSamples(records) {
  for (const vehicle of records) {
    if (!vehicle.imei || !Number.isFinite(vehicle.fuel)) continue
    const history = samples.value[vehicle.id] || []
    const at = vehicle.updatedAt || new Date().toISOString()
    if (history.at(-1)?.at === at) continue
    samples.value[vehicle.id] = [...history, { at, value: vehicle.fuel }].slice(-60)
  }
}
function receive(packet) {
  const data = packet.data
  let telemetryReceived = false
  if (packet.type === 'snapshot') {
    if (Array.isArray(data.vehicles)) {
      mergeRecords(vehicles, data.vehicles, true)
      recordSamples(data.vehicles)
      telemetryReceived = true
    }
    if (isModuleActive('ext.fleet.maintenance') && Array.isArray(data.orders))
      mergeRecords(orders, data.orders, true)
    for (const item of data.notifications || []) notifications.push(item)
  } else if (packet.type === 'vehicle') {
    mergeRecords(vehicles, [data])
    recordSamples(vehicles.value.filter((v) => String(v.id) === String(data.id)))
    telemetryReceived = true
  } else if (packet.type === 'work-order') {
    if (isModuleActive('ext.fleet.maintenance')) mergeRecords(orders, [data])
  } else if (packet.type === 'notification') notifications.push(data)
  else if (packet.type === 'values') {
    // Existing Go SSE format: { source: { upserts, removed, order } }.
    for (const [source, patch] of Object.entries(data)) {
      sections[source] = mergeSection(sections[source] || { rows: {}, order: [] }, patch)
      if (source.split('?')[0] === 'fleet/live') {
        telemetryReceived = true
        if (Array.isArray(patch.order)) {
          const retained = new Set(patch.order.map(String))
          vehicles.value = vehicles.value.filter((v) => retained.has(String(v.sourceId ?? v.id)))
        }
        for (const id of patch.removed || [])
          vehicles.value = vehicles.value.filter((v) => String(v.sourceId ?? v.id) !== String(id))
        const mapped = []
        for (const [id, fields] of Object.entries(patch.upserts || {})) {
          const existing = vehicles.value.find((v) => String(v.sourceId ?? v.id) === String(id))
          // Full entity creation remains the responsibility of the REST normalization adapter.
          if (!existing) continue
          const changes = { id: existing.id, updatedAt: new Date().toISOString() }
          const aliases = {
            speed_kmh: 'speed',
            fuel_liters: 'fuel',
            latitude: 'lat',
            longitude: 'lng',
            heading_deg: 'heading',
            satellite_count: 'satellites',
            hdop: 'hdop',
            plate_number: 'plate',
            vehicle_name: 'model',
          }
          for (const [field, value] of Object.entries(fields)) {
            if (aliases[field] && value !== null) changes[aliases[field]] = value
            if (field in existing && field !== 'id') changes[field] = value
          }
          if (fields.ignition_status) changes.ignition = fields.ignition_status === 'ON'
          if (fields.movement_status)
            changes.state =
              { MOVING: 'moving', IDLE: 'idle', PARKED: 'parked', OFFLINE: 'offline' }[
                fields.movement_status
              ] || existing.state
          mapped.push(changes)
        }
        mergeRecords(vehicles, mapped)
        recordSamples(vehicles.value)
      } else if (source.split('?')[0] === 'notifications') {
        for (const id of patch.removed || []) notifications.remove(id)
        for (const id of Object.keys(patch.upserts || {})) {
          const item = sections[source].rows[id]
          notifications.push({
            id,
            title: item.title,
            message: item.body,
            target: '/alerts',
            severity: item.severity || 'info',
            createdAt: item.created_at,
            read: Boolean(item.read_at),
          })
        }
      }
    }
  }
  if (telemetryReceived) lastUpdated.value = Date.now()
}
export function useFleet() {
  const stale = computed(
    () => lastUpdated.value !== null && now.value - lastUpdated.value > staleAfter,
  )
  const age = computed(() =>
    lastUpdated.value === null
      ? null
      : Math.max(0, Math.floor((now.value - lastUpdated.value) / 1000)),
  )
  async function load(force = false) {
    if (pending) return pending
    if (!force && vehicles.value.length) return true
    loading.value = true
    error.value = ''
    pending = Promise.allSettled([
      fleetService.listVehicles(),
      isModuleActive('ext.fleet.maintenance') ? maintenanceService.list() : Promise.resolve(null),
    ])
      .then(([assets, workOrders]) => {
        const errors = []
        if (assets.status === 'fulfilled') {
          if (!Array.isArray(assets.value))
            errors.push('Unexpected vehicle response. Check the API adapter.')
          else {
            mergeRecords(vehicles, assets.value, true)
            recordSamples(assets.value)
            lastUpdated.value = Date.now()
          }
        } else errors.push(`Vehicles: ${assets.reason.message}`)
        if (workOrders.status === 'fulfilled' && workOrders.value !== null) {
          if (!Array.isArray(workOrders.value))
            errors.push('Unexpected work order response. Check the API adapter.')
          else mergeRecords(orders, workOrders.value, true)
        } else if (workOrders.status === 'rejected') {
          // Never retain stale/demo work orders when the live API fails.
          orders.value = []
          if (workOrders.reason?.status !== 404)
            errors.push(`Maintenance: ${workOrders.reason.message}`)
        }
        error.value = errors.join(' ')
        notifications.seed(vehicles.value, orders.value)
        return errors.length === 0
      })
      .finally(() => {
        loading.value = false
        pending = null
      })
    return pending
  }
  async function startRealtime() {
    if (running) return
    running = true
    const revision = ++generation
    connection.value = { status: 'connecting', detail: 'Loading fleet data', attempt: 0 }
    clock = setInterval(() => {
      now.value = Date.now()
    }, 1000)
    await load()
    if (!running || revision !== generation) return
    if (error.value && !vehicles.value.length) {
      connection.value = {
        status: 'error',
        detail: 'Initial data unavailable. Retry synchronization.',
      }
      return
    }
    stream = connectRealtime({
      onPacket: receive,
      onState: (value) => {
        connection.value = value
      },
    })
    if (paused.value) stream.setPaused(true)
  }
  function stopRealtime() {
    running = false
    generation++
    stream?.close()
    stream = null
    clearInterval(clock)
  }
  function togglePaused() {
    paused.value = !paused.value
    stream?.setPaused(paused.value)
  }
  async function reconnect() {
    const ok = await load(true)
    if (ok && running) {
      if (stream) stream.reconnect()
      else {
        running = false
        clearInterval(clock)
        await startRealtime()
      }
    }
    return ok
  }
  function activity(id, title, message, target) {
    notifications.push({
      id: `action:${id}:${crypto.randomUUID()}`,
      title,
      message,
      target,
      severity: 'success',
    })
  }
  async function addVehicle(payload) {
    const asset = await fleetService.createVehicle(payload)
    mergeRecords(vehicles, [asset])
    activity(
      asset.id,
      `Asset registered · ${asset.id}`,
      'The vehicle is ready for gateway pairing.',
      '/vehicles',
    )
    return asset
  }
  async function pairDevice(id, imei) {
    const asset = await fleetService.pairDevice(id, imei)
    mergeRecords(vehicles, [asset])
    activity(id, `Gateway paired · ${id}`, 'Waiting for the next telemetry packet.', '/vehicles')
  }
  async function addOrder(payload) {
    if (!isModuleActive('ext.fleet.maintenance'))
      throw new Error('The maintenance module is disabled.')
    const order = await maintenanceService.create(payload)
    orders.value.unshift(order)
    activity(order.id, `Maintenance scheduled · ${order.vehicleId}`, order.title, '/maintenance')
  }
  async function updateOrder(id, payload) {
    if (!isModuleActive('ext.fleet.maintenance'))
      throw new Error('The maintenance module is disabled.')
    const order = await maintenanceService.update(id, payload)
    mergeRecords(orders, [order])
    activity(id, `${id} · ${order.status}`, order.title, '/maintenance')
  }
  async function deleteOrder(id) {
    if (!isModuleActive('ext.fleet.maintenance')) throw new Error('The maintenance module is disabled.')
    await maintenanceService.remove(id)
    orders.value = orders.value.filter((order) => String(order.id) !== String(id))
  }
  return {
    vehicles,
    orders,
    loading,
    error,
    connection,
    lastUpdated,
    now,
    paused,
    stale,
    age,
    samples,
    load,
    startRealtime,
    stopRealtime,
    togglePaused,
    reconnect,
    addVehicle,
    pairDevice,
    addOrder,
    updateOrder,
    deleteOrder,
  }
}

watch(
  () => isModuleActive('ext.fleet.maintenance'),
  (enabled) => {
    if (enabled && running) useFleet().load(true)
  },
)
