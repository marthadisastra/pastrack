import { useMock } from './http'
import { authBaseURL } from './auth'
import { fleetService } from './fleet'
// Single connection per mounted workspace; cleanup removes listeners, retries and timers.
export function connectRealtime({ onPacket, onState }) {
  let source,
    timer,
    retryTimer,
    deadline,
    disposed = false,
    paused = false,
    attempt = 0
  const hidden = () => document.visibilityState === 'hidden'
  function clear() {
    source?.close()
    source = null
    clearInterval(timer)
    clearTimeout(retryTimer)
    clearTimeout(deadline)
  }
  function state(status, detail = '') {
    if (!disposed) onState({ status, detail, attempt })
  }
  function packet(payload) {
    if (!disposed && !paused && !hidden() && navigator.onLine) onPacket(payload)
  }
  function retry(message) {
    if (disposed || paused || hidden() || !navigator.onLine) return
    clear()
    attempt++
    state('reconnecting', message)
    retryTimer = setTimeout(start, Math.min(30000, 1000 * 2 ** Math.min(attempt - 1, 5)))
  }
  function start() {
    clear()
    if (disposed) return
    if (paused) return state('paused', 'Updates paused by you')
    if (!navigator.onLine)
      return state('offline', 'Network unavailable. Last received values are retained.')
    if (hidden()) return state('background', 'Updates suspended while this tab is hidden')
    state('connecting', 'Connecting to telemetry')
    if (useMock) {
      attempt = 0
      state('demo', 'Simulated telemetry · updates every 4 seconds')
      timer = setInterval(() => packet(fleetService.sampleTelemetry()), 4000)
      return
    }
    const url =
      import.meta.env.VITE_REALTIME_URL ||
      `${authBaseURL}/realtime?watch=fleet/live&watch=notifications%3Flimit%3D100`
    source = new EventSource(url, { withCredentials: true })
    deadline = setTimeout(() => retry('The telemetry connection timed out.'), 15000)
    source.addEventListener('open', () => {
      clearTimeout(deadline)
      attempt = 0
      state('live', 'Stream connected')
    })
    source.addEventListener('ready', () => {
      clearTimeout(deadline)
      state('live', 'Waiting for telemetry changes')
    })
    for (const type of [
      'snapshot',
      'vehicle',
      'work-order',
      'notification',
      'heartbeat',
      'values',
    ]) {
      source.addEventListener(type, (event) => {
        try {
          packet({ type, data: JSON.parse(event.data) })
        } catch {
          retry('Received an invalid telemetry packet. Reconnecting for a fresh snapshot.')
        }
      })
    }
    source.addEventListener('live-error', () =>
      retry('The server could not synchronize telemetry.'),
    )
    source.onerror = () => retry('Connection interrupted. Retrying automatically.')
  }
  function changed() {
    start()
  }
  window.addEventListener('online', changed)
  window.addEventListener('offline', changed)
  document.addEventListener('visibilitychange', changed)
  start()
  return {
    setPaused(value) {
      paused = value
      start()
    },
    reconnect() {
      attempt = 0
      start()
    },
    close() {
      disposed = true
      clear()
      window.removeEventListener('online', changed)
      window.removeEventListener('offline', changed)
      document.removeEventListener('visibilitychange', changed)
    },
  }
}
// Apply the Go server's per-field delta format without discarding fields absent from the packet.
export function mergeSection(previous, patch) {
  const rows = { ...previous.rows },
    order = patch.order ? patch.order.map(String) : [...previous.order]
  for (const id of patch.removed || []) delete rows[String(id)]
  for (const [id, fields] of Object.entries(patch.upserts || {})) {
    rows[id] = { ...rows[id], ...fields, id: fields.id ?? rows[id]?.id ?? id }
    if (!order.includes(id)) order.push(id)
  }
  return { rows, order: order.filter((id) => rows[id]) }
}
