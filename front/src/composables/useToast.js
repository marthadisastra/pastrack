import { ref } from 'vue'
const messages = ref([]),
  timers = new Map()
export function useToast() {
  function dismiss(id) {
    clearTimeout(timers.get(id))
    timers.delete(id)
    messages.value = messages.value.filter((item) => item.id !== id)
  }
  function resume(id) {
    const item = messages.value.find((item) => item.id === id)
    if (!item || item.tone === 'error') return
    clearTimeout(timers.get(id))
    timers.set(
      id,
      setTimeout(() => dismiss(id), 6000),
    )
  }
  function pause(id) {
    clearTimeout(timers.get(id))
    timers.delete(id)
  }
  function notify(message, tone = 'success') {
    const existing = messages.value.find((item) => item.message === message && item.tone === tone)
    if (existing) {
      resume(existing.id)
      return existing.id
    }
    const id = crypto.randomUUID()
    // Bound visual interruptions; errors remain until explicitly dismissed.
    if (messages.value.length >= 3)
      dismiss(messages.value.find((item) => item.tone !== 'error')?.id || messages.value[0].id)
    messages.value.push({ id, message, tone })
    resume(id)
    return id
  }
  return { messages, notify, dismiss, pause, resume }
}
