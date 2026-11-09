import { ref } from 'vue'
const confirmation = ref(null)
let resolver
export function useConfirm() {
  function ask(options) {
    // Only one navigation confirmation can be active at once.
    if (resolver) resolver(false)
    confirmation.value = options
    return new Promise((resolve) => {
      resolver = resolve
    })
  }
  function answer(value) {
    const resolve = resolver
    resolver = null
    confirmation.value = null
    resolve?.(value)
  }
  return { confirmation, ask, answer }
}
