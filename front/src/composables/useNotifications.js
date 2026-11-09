import { computed, ref } from 'vue'
import { useMock } from '../services/http'
import { useToast } from './useToast'
import { notifyAudio } from '../modules/audio'
import { notificationService } from '../services/notifications'
// Read preferences are scoped to this browser in demo mode. They are not incident acknowledgement.
const storageKey = 'pastrack.demo.notification-read.v1'
function readSaved() {
  if (!useMock) return []
  try {
    return JSON.parse(localStorage.getItem(storageKey) || '[]')
      .filter((id) => typeof id === 'string')
      .slice(-200)
  } catch {
    return []
  }
}
const readIds = new Set(readSaved()),
  items = ref([]),
  busy = ref(false),
  error = ref('')
let seeded = false
const isRead = (record) =>
  Boolean(record.readAt ?? record.read_at ?? record.read)
function persist() {
  if (!useMock) return
  try {
    localStorage.setItem(storageKey, JSON.stringify([...readIds].slice(-200)))
  } catch {
    /* Private browsing can disable storage. */
  }
}
export function useNotifications() {
  const unread = computed(() => items.value.filter((item) => !item.read).length)
  function push(payload) {
    const id = String(payload.id || crypto.randomUUID())
    const existing = items.value.find((item) => item.id === id)
    if (existing) {
      Object.assign(existing, payload, { id, read: existing.read || Boolean(payload.read) })
      return existing
    }
    const item = {
      severity: 'info',
      createdAt: new Date().toISOString(),
      read: readIds.has(id) || isRead(payload),
      ...payload,
      id,
    }
    if (
      item.severity === 'critical' &&
      !item.read &&
      Date.now() - new Date(item.createdAt).getTime() < 30000
    ) {
      useToast().notify(`${item.title}. ${item.message}`, 'error')
    }
    if (Date.now() - new Date(item.createdAt).getTime() < 30000) notifyAudio(item)
    items.value.unshift(item)
    items.value = items.value.slice(0, 200)
    return item
  }
  function replace(records) {
    items.value = records.map((record) => ({
      id: String(record.id),
      title: record.title,
      message: record.body ?? record.message ?? '',
      target: record.target || '/alerts',
      severity: record.severity || 'info',
      createdAt: record.createdAt,
      read: isRead(record) || readIds.has(String(record.id)),
    }))
  }
  function applyDelta(upserts = [], removed = []) {
    for (const id of removed) remove(id)
    for (const record of upserts) {
      const normalized = {
        id: String(record.id),
        title: record.title,
        message: record.body ?? record.message ?? '',
        target: record.target || '/alerts',
        severity: record.severity || 'info',
        createdAt: record.createdAt,
        read: isRead(record) || readIds.has(String(record.id)),
      }
      const existing = items.value.find((item) => item.id === normalized.id)
      if (existing) Object.assign(existing, normalized)
      else items.value.unshift(normalized)
    }
    items.value = items.value.slice(0, 200)
  }
  async function markRead(id) {
    const item = items.value.find((item) => item.id === id)
    if (!item) return
    const prior = item.read
    item.read = true
    if (useMock) {
      readIds.add(id)
      persist()
      return
    }
    busy.value = true
    error.value = ''
    try {
      await notificationService.markRead(id)
    } catch (cause) {
      item.read = prior
      error.value = cause.message
      throw cause
    } finally {
      busy.value = false
    }
  }
  async function markAllRead() {
    const prior = new Map(items.value.map((item) => [item.id, item.read]))
    items.value.forEach((item) => {
      item.read = true
      if (useMock) readIds.add(item.id)
    })
    if (useMock) {
      persist()
      return
    }
    busy.value = true
    error.value = ''
    try {
      await notificationService.markAllRead()
    } catch (cause) {
      items.value.forEach((item) => {
        item.read = prior.get(item.id) ?? item.read
      })
      error.value = cause.message
      throw cause
    } finally {
      busy.value = false
    }
  }
  function remove(id) {
    items.value = items.value.filter((item) => item.id !== String(id))
  }
  function seed(vehicles, orders) {
    if (seeded || !useMock) return
    seeded = true
    const at = Date.now()
    orders
      .filter((order) => order.status === 'Overdue')
      .forEach((order, index) =>
        push({
          id: `seed:${order.id}`,
          severity: 'warning',
          title: `Maintenance overdue · ${order.vehicleId}`,
          message: order.title,
          target: '/maintenance',
          createdAt: new Date(at - (index + 2) * 60000).toISOString(),
        }),
      )
    vehicles
      .filter((v) => v.state === 'alert')
      .forEach((v) =>
        push({
          id: `seed:${v.id}:siphon`,
          severity: 'critical',
          title: `Fuel siphon detected · ${v.id}`,
          message: '48.2 L drop while ignition is off. Inspect the unit and verify the incident.',
          vehicleId: v.id,
          target: `/tracking?unit=${encodeURIComponent(v.id)}`,
          createdAt: new Date(at - 60000).toISOString(),
        }),
      )
  }
  return {
    items,
    unread,
    busy,
    error,
    push,
    replace,
    applyDelta,
    markRead,
    markAllRead,
    remove,
    seed,
  }
}
