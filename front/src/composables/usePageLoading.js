import { ref } from 'vue'
const pending = ref(false),
  visible = ref(false),
  navigationError = ref('')
let delay
export function usePageLoading() {
  function start() {
    pending.value = true
    navigationError.value = ''
    clearTimeout(delay)
    delay = setTimeout(() => {
      visible.value = true
    }, 150)
  }
  function finish() {
    clearTimeout(delay)
    pending.value = false
    visible.value = false
  }
  function fail() {
    finish()
    navigationError.value = 'This page could not be loaded. Check your connection and retry.'
  }
  return { pending, visible, navigationError, start, finish, fail }
}
