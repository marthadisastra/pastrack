<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { settingsState, defaultSettings, commitSettings } from '../../modules/state'
import { moduleCatalog, coreVersion } from '../../modules/registry'
import { applyBranding } from '../../modules/branding'
import { settingsService } from '../../services/settings'
import { useMock } from '../../services/http'
import { useToast } from '../../composables/useToast'
import { useConfirm } from '../../composables/useConfirm'
import { useUnsavedChanges } from '../../composables/useUnsavedChanges'
import AppIcon from '../../components/ui/AppIcon.vue'
import StatusBadge from '../../components/ui/StatusBadge.vue'
import ModuleManager from '../../components/settings/ModuleManager.vue'
import BrandingSettings from '../../components/settings/BrandingSettings.vue'
import AudioSettings from '../../components/settings/AudioSettings.vue'
import AccessSettings from '../../components/settings/AccessSettings.vue'
import SmtpSettings from '../../components/settings/SmtpSettings.vue'
import TelemetryScopeSettings from '../../components/settings/TelemetryScopeSettings.vue'
const smtpDirty = ref(false)
const clone = (value) => JSON.parse(JSON.stringify(value)),
  route = useRoute(),
  router = useRouter()
const { notify } = useToast(),
  { ask } = useConfirm(),
  draft = reactive(clone(settingsState.value))
const draftProxy = computed({ get: () => draft, set: (value) => Object.assign(draft, value) })
const baseline = ref(JSON.stringify(draft)),
  busy = ref(false),
  loading = ref(false),
  error = ref(''),
  savedAt = ref(null)
const tabs = [
  { id: 'modules', name: 'Module Manager', icon: 'extension' },
  { id: 'branding', name: 'Branding & Web Assets', icon: 'branding_watermark' },
  { id: 'sound', name: 'Sound & Audio Alerts', icon: 'volume_up' },
  { id: 'access', name: 'Access & RBAC', icon: 'admin_panel_settings' },
  { id: 'smtp', name: 'SMTP & Email', icon: 'mail' },
  { id: 'scope', name: 'Telemetry Scope', icon: 'shield' },
]
const tab = computed(() =>
  tabs.some((tab) => tab.id === route.query.tab) ? route.query.tab : 'modules',
)
const active = computed(
  () =>
    moduleCatalog.filter(
      (module) => draft.modules[module.id].installed && draft.modules[module.id].enabled,
    ).length,
)
const dirty = computed(() => JSON.stringify(draft) !== baseline.value)
useUnsavedChanges(computed(() => dirty.value || smtpDirty.value))
async function tabKey(event, id) {
  const keys = ['ArrowLeft', 'ArrowRight', 'Home', 'End']
  if (!keys.includes(event.key)) return
  event.preventDefault()
  const index = tabs.findIndex((tab) => tab.id === id)
  const target =
    event.key === 'Home'
      ? tabs[0]
      : event.key === 'End'
        ? tabs.at(-1)
        : tabs[(index + (event.key === 'ArrowRight' ? 1 : -1) + tabs.length) % tabs.length]
  await changeTab(target.id)
  await nextTick()
  document.getElementById(`settings-tab-${target.id}`)?.focus()
}
async function changeTab(id) {
  if (
    tab.value === 'smtp' &&
    smtpDirty.value &&
    !(await ask({
      title: 'Discard SMTP changes?',
      message: 'SMTP settings have not been saved.',
      confirmLabel: 'Discard changes',
      cancelLabel: 'Keep editing',
    }))
  )
    return
  smtpDirty.value = false
  await router.replace({ query: { ...route.query, tab: id } })
}
async function save() {
  if (busy.value || loading.value || !dirty.value) return
  busy.value = true
  error.value = ''
  try {
    const saved = await settingsService.save(draft)
    commitSettings(saved)
    Object.assign(draft, clone(saved))
    baseline.value = JSON.stringify(draft)
    applyBranding()
    document.title = `Settings · ${saved.branding.name}`
    savedAt.value = new Date()
    notify(useMock ? 'Settings saved in this browser. Module menus updated.' : 'Settings saved.')
  } catch (cause) {
    error.value =
      cause.name === 'QuotaExceededError'
        ? 'Browser storage is full. Remove large brand assets and retry.'
        : cause.message
    notify(error.value, 'error')
  } finally {
    busy.value = false
  }
}
async function reset() {
  if (
    await ask({
      title: 'Reset settings to defaults?',
      message:
        'Load default settings into this editor. Existing fleet records are kept. Changes take effect only after Save Changes.',
      confirmLabel: 'Load Defaults',
      cancelLabel: 'Cancel',
    })
  ) {
    Object.assign(draft, defaultSettings())
    error.value = ''
  }
}
async function discard() {
  if (
    !dirty.value ||
    (await ask({
      title: 'Discard settings edits?',
      message: 'Restore the last saved configuration and discard staged changes.',
      confirmLabel: 'Discard Changes',
      cancelLabel: 'Keep Editing',
    }))
  ) {
    Object.assign(draft, clone(settingsState.value))
    baseline.value = JSON.stringify(draft)
    error.value = ''
  }
}
function shortcut(event) {
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 's') {
    event.preventDefault()
    if (tab.value === 'smtp') window.dispatchEvent(new Event('pastrack:save-smtp'))
    else save()
  }
}
onMounted(async () => {
  window.addEventListener('keydown', shortcut)
  if (!useMock) {
    loading.value = true
    try {
      const data = await settingsService.load()
      if (data) {
        commitSettings(data)
        Object.assign(draft, clone(data))
        baseline.value = JSON.stringify(draft)
        applyBranding()
      }
    } catch (cause) {
      error.value = `Settings API unavailable: ${cause.message}`
    } finally {
      loading.value = false
    }
  }
})
onBeforeUnmount(() => window.removeEventListener('keydown', shortcut))
</script>
<template>
  <div class="page">
    <section class="panel p-3">
      <div class="flex flex-wrap items-start justify-between gap-3">
        <div>
          <p class="mb-1 text-[9px] font-semibold uppercase tracking-wider text-slate-500">
            System › <span class="text-blue-700">Settings & Engine</span>
          </p>
          <div class="flex flex-wrap items-center gap-2">
            <h1 class="text-base font-bold tracking-tight">System Settings & Extension Engine</h1>
            <StatusBadge tone="blue" dot>Core v{{ coreVersion }}-STABLE</StatusBadge>
          </div>
          <p class="mt-1 text-[10px] text-slate-500">
            Manage modular runtime packages, enterprise branding, hardware audio alerts and role
            policies.
          </p>
        </div>
        <div class="flex gap-2">
          <button class="btn" :disabled="busy || loading" @click="reset">
            <AppIcon name="restart_alt" />Reset Defaults</button
          ><button
            class="btn btn-primary"
            :disabled="!dirty || busy || loading || tab === 'smtp' || tab === 'scope'"
            @click="save"
          >
            <AppIcon
              :name="busy ? 'progress_activity' : 'save'"
              :class="busy && 'animate-spin'"
            />{{ busy ? 'Saving…' : 'Save Changes'
            }}<kbd aria-hidden="true" class="hidden rounded bg-blue-800/30 px-1 text-[9px] sm:block"
              >⌘S</kbd
            >
          </button>
        </div>
      </div>
      <div
        class="mt-3 flex gap-1.5 overflow-x-auto border-t border-slate-100 pt-3"
        role="tablist"
        aria-label="Settings sections"
      >
        <button
          v-for="item in tabs"
          :key="item.id"
          :id="`settings-tab-${item.id}`"
          role="tab"
          :aria-selected="tab === item.id"
          :tabindex="tab === item.id ? 0 : -1"
          @keydown="tabKey($event, item.id)"
          :aria-controls="`settings-panel-${item.id}`"
          class="flex shrink-0 items-center gap-1.5 rounded-md border px-3 py-2 text-[11px] font-medium"
          :class="
            tab === item.id
              ? 'border-blue-600 bg-blue-600 text-white'
              : 'border-slate-200 bg-slate-50 text-slate-600 hover:bg-slate-100'
          "
          @click="changeTab(item.id)"
        >
          <AppIcon :name="item.icon" class="!text-base" />{{ item.name
          }}<span
            v-if="item.id === 'modules'"
            class="rounded-full px-1.5 py-0.5 text-[9px]"
            :class="tab === 'modules' ? 'bg-white/20' : 'bg-slate-200'"
            >{{ active }} Active</span
          ><span
            v-else-if="item.id === 'access'"
            class="rounded-full px-1.5 text-[9px]"
            :class="tab === 'access' ? 'bg-white/20' : 'bg-slate-200'"
            >{{ draft.roles.length }} Roles</span
          >
        </button>
      </div>
    </section>
    <div class="flex flex-wrap items-center justify-between gap-2 text-[10px]">
      <span class="text-slate-400">{{
        useMock
          ? 'Browser preview · settings stored locally. Backend installation and policy enforcement are not active.'
          : 'API settings · backend must enforce module lifecycle and permissions.'
      }}</span
      ><span v-if="dirty" role="status" class="font-medium text-amber-700"
        >Unsaved changes
        <button class="ml-2 underline" :disabled="busy" @click="discard">Discard</button></span
      ><span v-else-if="savedAt" class="text-emerald-700"
        >Saved {{ savedAt.toLocaleTimeString('en-GB', { timeZone: 'Asia/Jakarta' }) }}</span
      >
    </div>
    <p
      v-if="error"
      role="alert"
      class="rounded-lg border border-rose-200 bg-rose-50 p-3 text-xs text-rose-700"
    >
      {{ error }}
    </p>
    <div v-if="loading" class="panel p-5" role="status">
      <div class="skeleton h-32 w-full"></div>
      <p class="mt-3 text-xs text-slate-400">Loading settings…</p>
    </div>
    <div
      v-else
      :id="`settings-panel-${tab}`"
      role="tabpanel"
      :aria-labelledby="`settings-tab-${tab}`"
      :aria-busy="busy"
    >
      <fieldset :disabled="busy" class="min-w-0">
        <ModuleManager v-if="tab === 'modules'" v-model="draftProxy" /><BrandingSettings
          v-else-if="tab === 'branding'"
          v-model="draftProxy"
        /><AudioSettings v-else-if="tab === 'sound'" v-model="draftProxy" /><AccessSettings
          v-else-if="tab === 'access'"
          v-model="draftProxy"
        /><SmtpSettings
          v-else-if="tab === 'smtp'"
          @dirty="smtpDirty = $event"
        /><TelemetryScopeSettings v-else-if="tab === 'scope'" />
      </fieldset>
    </div>
  </div>
</template>
