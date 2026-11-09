import { computed, ref, watch } from 'vue'
import {
  dashboardService,
  dashboardSnapshotTransport,
  dashboardSocketURL,
  normalizeDashboardSnapshot,
  normalizeDashboardVehicle,
} from '../services/dashboard'
import { useAuth } from './useAuth'
import { useNotifications } from './useNotifications'
const snapshot = ref(null),
  fuelSamples = ref([]),
  loading = ref(false),
  error = ref(''),
  connection = ref({ status: 'idle', detail: 'Realtime connection is not active.', attempt: 0 })
let socket,
  pending,
  controller,
  retryTimer,
  generation = 0,
  cursor = 0,
  attempt = 0,
  running = false
const notificationStore = useNotifications()
const networkChanged = () => {
  if (!running) return
  if (navigator.onLine) reconnectSocket('Network restored. Loading a fresh snapshot.')
  else socket?.close()
}
watch(
  () => useAuth().user.value?.id,
  () => {
    generation++
    running = false
    clearTimeout(retryTimer)
    socket?.close()
    socket = null
    controller?.abort()
    pending = null
    snapshot.value = null
    fuelSamples.value = []
    loading.value = false
    error.value = ''
  },
)
export function useDashboard() {
  async function loadSnapshot() {
    if (pending) return pending
    const revision = generation
    controller = new AbortController()
    loading.value = true
    error.value = ''
    pending = dashboardService
      .snapshot(controller.signal)
      .then((result) => {
        if (revision === generation) snapshot.value = result
        return true
      })
      .catch((cause) => {
        if (revision === generation) error.value = cause.message
        return false
      })
      .finally(() => {
        if (revision === generation) {
          loading.value = false
          pending = null
        }
      })
    return pending
  }
  function setConnection(status, detail) {
    connection.value = { status, detail, attempt }
  }
  function applySnapshot(message) {
    const normalized = normalizeDashboardSnapshot({
      data: { vehicles: message.vehicles, summary: message.summary },
      meta: {
        schemaVersion: message.schemaVersion,
        asOf: message.asOf,
        mode: 'websocket',
        realtime: true,
      },
    })
    snapshot.value = normalized
    notificationStore.replace(message.notifications || [])
    fuelSamples.value = []
    cursor = message.cursor
    loading.value = false
    error.value = ''
    attempt = 0
    setConnection('live', 'WebSocket connected · event-driven updates')
  }
  function applyDelta(message) {
    if (!snapshot.value) {
      reconnect('Realtime delta arrived before snapshot. Loading a fresh snapshot.')
      return
    }
    // A retried delivery can legitimately replay an already-applied cursor.
    // Ignore stale duplicates; only a future gap requires a full resync.
    if (message.cursor <= cursor) return
    if (message.cursor !== cursor + 1) {
      reconnect('Realtime cursor gap detected. Loading a fresh snapshot.')
      return
    }
    const byID = new Map(snapshot.value.data.vehicles.map((row) => [row.id, row]))
    for (const id of message.removed || []) byID.delete(String(id))
    for (const raw of message.upserts || []) {
      const row = normalizeDashboardVehicle(raw)
      const prior = byID.get(row.id)
      if (prior) Object.assign(prior, row)
      else byID.set(row.id, row)
    }
    snapshot.value.data.vehicles = [...byID.values()]
    if (message.summary) snapshot.value.data.summary = message.summary
    snapshot.value.meta.asOf = message.asOf
    notificationStore.applyDelta(message.notificationUpserts, message.notificationRemoved)
    for (const raw of message.fuelSamples || []) {
      if (!raw?.vehicleId || !raw?.sampleId || !raw?.at) continue
      const sample = {
        ...raw,
        vehicleId: String(raw.vehicleId),
        sampleId: String(raw.sampleId),
        liters: raw.liters === null || raw.liters === undefined ? null : Number(raw.liters),
        percent: raw.percent === null || raw.percent === undefined ? null : Number(raw.percent),
      }
      const index = fuelSamples.value.findIndex(
        (item) => item.vehicleId === sample.vehicleId && item.sampleId === sample.sampleId,
      )
      if (index >= 0) fuelSamples.value.splice(index, 1, sample)
      else fuelSamples.value.push(sample)
    }
    if (fuelSamples.value.length > 1000)
      fuelSamples.value.splice(0, fuelSamples.value.length - 1000)
    cursor = message.cursor
    error.value = ''
  }
  function scheduleRetry(detail) {
    if (!running || !navigator.onLine) {
      setConnection('offline', 'Network unavailable. Last values are retained as stale.')
      return
    }
    attempt++
    setConnection('reconnecting', detail)
    const delay = Math.min(30000, 750 * 2 ** Math.min(attempt - 1, 5))
    clearTimeout(retryTimer)
    retryTimer = setTimeout(open, delay + Math.floor(Math.random() * 250))
  }
  function open() {
    if (!running || socket) return
    const revision = generation
    loading.value = !snapshot.value
    setConnection(attempt ? 'reconnecting' : 'connecting', 'Connecting to dashboard WebSocket')
    const connectionSocket = new WebSocket(dashboardSocketURL())
    socket = connectionSocket
    connectionSocket.onmessage = (event) => {
      if (revision !== generation) return
      try {
        const message = JSON.parse(event.data)
        if (message.schemaVersion !== 1) throw new Error('Unsupported realtime schema.')
        if (message.type === 'snapshot') applySnapshot(message)
        else if (message.type === 'delta') applyDelta(message)
        else if (message.type === 'heartbeat')
          setConnection('live', 'WebSocket connected · event-driven updates')
        else if (message.type === 'session_revoked') {
          window.dispatchEvent(new Event('pastrack:session-expired'))
          stop()
        } else if (message.type === 'error') {
          error.value = 'Realtime synchronization failed. Last values are retained as stale.'
          setConnection('error', message.reason || 'Realtime synchronization failed')
        }
      } catch (cause) {
        error.value = cause.message
        reconnect('Invalid realtime message. Loading a fresh snapshot.')
      }
    }
    connectionSocket.onclose = () => {
      if (revision !== generation) return
      if (socket === connectionSocket) socket = null
      scheduleRetry('Connection closed. Reconnecting without polling.')
    }
    connectionSocket.onerror = () => connectionSocket.close()
  }
  function start() {
    if (running) return
    running = true
    generation++
    if (dashboardSnapshotTransport) {
      setConnection('snapshot', 'Snapshot test transport · automatic updates disabled')
      loadSnapshot()
      return
    }
    window.addEventListener('online', networkChanged)
    window.addEventListener('offline', networkChanged)
    open()
  }
  function stop() {
    running = false
    generation++
    clearTimeout(retryTimer)
    controller?.abort()
    pending = null
    const active = socket
    socket = null
    active?.close()
    window.removeEventListener('online', networkChanged)
    window.removeEventListener('offline', networkChanged)
    setConnection('idle', 'Realtime connection is not active.')
    loading.value = false
  }
  function reconnect(detail = 'Resynchronizing from the server.') {
    if (dashboardSnapshotTransport) return loadSnapshot()
    if (!running) {
      start()
      return true
    }
    generation++
    clearTimeout(retryTimer)
    const active = socket
    socket = null
    active?.close()
    cursor = 0
    setConnection('reconnecting', detail)
    open()
    return true
  }

  function reconnectSocket(detail) {
    return reconnect(detail)
  }
  return {
    snapshot,
    loading,
    error,
    load: reconnect,
    start,
    stop,
    connection,
    vehicles: computed(() => snapshot.value?.data.vehicles || []),
    summary: computed(() => snapshot.value?.data.summary || null),
    fuelSamples,
    asOf: computed(() => snapshot.value?.meta.asOf || null),
  }
}
