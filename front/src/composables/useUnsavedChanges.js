import { onBeforeUnmount, watch } from 'vue'
import { onBeforeRouteLeave } from 'vue-router'
import { useConfirm } from './useConfirm'
import { useAuth } from './useAuth'
export function useUnsavedChanges(active) {
  const { ask } = useConfirm()
  const beforeUnload = (event) => {
    event.preventDefault()
    event.returnValue = ''
  }
  watch(
    active,
    (dirty) => {
      if (dirty) window.addEventListener('beforeunload', beforeUnload)
      else window.removeEventListener('beforeunload', beforeUnload)
    },
    { immediate: true },
  )
  onBeforeRouteLeave(
    () =>
      (!useAuth().authDemo && !useAuth().user.value) ||
      !active.value ||
      ask({
        title: 'Leave without saving?',
        message: 'Your form has unsaved changes. Leave this page and discard them?',
      }),
  )
  onBeforeUnmount(() => window.removeEventListener('beforeunload', beforeUnload))
}
