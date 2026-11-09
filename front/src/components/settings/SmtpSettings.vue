<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { request } from '../../services/http'
import { authBaseURL, authDemo } from '../../services/auth'
import { useAuth } from '../../composables/useAuth'
import { useToast } from '../../composables/useToast'
import AppIcon from '../ui/AppIcon.vue'
const emit = defineEmits(['dirty'])
const { user } = useAuth(),
  { notify } = useToast()
const form = reactive({
  enabled: false,
  host: '',
  port: 587,
  tlsMode: 'starttls',
  username: '',
  password: '',
  fromEmail: '',
  fromName: 'PASTRACK',
  clearPassword: false,
})
const loading = ref(false),
  busy = ref(false),
  error = ref(''),
  configured = ref(false),
  ready = ref(false),
  baseline = ref(JSON.stringify(form))
const allowed = computed(() => !authDemo && user.value?.role === 'admin'),
  dirty = computed(() => JSON.stringify(form) !== baseline.value)
watch(dirty, (value) => emit('dirty', value))
const api = (path, options = {}) =>
  request(`/auth/smtp${path}`, { baseURL: authBaseURL, ...options })
async function load() {
  if (!allowed.value) return
  loading.value = true
  error.value = ''
  try {
    const result = await api('')
    Object.assign(form, result.data, { password: '', clearPassword: false })
    configured.value = result.passwordConfigured
    ready.value = result.recoveryReady
    baseline.value = JSON.stringify(form)
  } catch (cause) {
    error.value = cause.message
  } finally {
    loading.value = false
  }
}
async function save(test = false) {
  if (!allowed.value || busy.value) return
  busy.value = true
  error.value = ''
  try {
    await api(test ? '/test' : '', { method: test ? 'POST' : 'PUT', body: { ...form } })
    if (test) notify('SMTP connection and authentication succeeded. No email was sent.')
    else {
      await load()
      notify('SMTP settings saved securely on the server.')
    }
  } catch (cause) {
    error.value = cause.message
  } finally {
    busy.value = false
  }
}
const shortcut = () => save(false)
onMounted(() => {
  load()
  window.addEventListener('pastrack:save-smtp', shortcut)
})
onBeforeUnmount(() => {
  emit('dirty', false)
  window.removeEventListener('pastrack:save-smtp', shortcut)
})
</script>
<template>
  <section class="panel">
    <div class="panel-title">
      <div>
        <h2 class="flex items-center gap-2 text-xs font-semibold">
          <AppIcon name="mail" class="!text-base text-blue-600" />SMTP & Password Recovery Email
        </h2>
        <p class="mt-1 text-[10px] text-slate-400">
          Server configuration · admin access required · SMTP password is never stored in this
          browser.
        </p>
      </div>
      <span
        v-if="allowed"
        class="rounded px-2 py-1 text-[10px]"
        :class="ready ? 'bg-emerald-50 text-emerald-700' : 'bg-amber-50 text-amber-700'"
        >{{ ready ? 'Recovery email enabled' : 'Recovery email not ready' }}</span
      >
    </div>
    <p v-if="!allowed" class="p-4 text-xs leading-6 text-slate-500">
      Sign in with a Go backend administrator account to configure SMTP. SMTP credentials cannot be
      saved in demo mode.
    </p>
    <div v-else-if="loading" role="status" class="p-4 text-xs text-slate-500">
      Loading SMTP settings…
    </div>
    <form v-else class="space-y-4 p-4" @submit.prevent="save(false)">
      <p v-if="error" role="alert" class="rounded bg-rose-50 p-3 text-xs text-rose-700">
        {{ error }}
      </p>
      <label class="flex items-center gap-2 text-xs"
        ><input v-model="form.enabled" type="checkbox" class="rounded text-blue-700" />Enable SMTP
        password recovery</label
      >
      <fieldset :disabled="busy" class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        <label class="field"
          >SMTP Host<input
            v-model="form.host"
            :required="form.enabled"
            autocomplete="off"
            placeholder="smtp.example.com" /></label
        ><label class="field"
          >SMTP Port<input
            v-model.number="form.port"
            type="number"
            min="1"
            max="65535"
            required /></label
        ><label class="field"
          >Encryption<select v-model="form.tlsMode">
            <option value="starttls">STARTTLS (usually 587)</option>
            <option value="tls">TLS (usually 465)</option>
            <option value="none">None · loopback relay only</option>
          </select></label
        ><label class="field"
          >SMTP Username<input v-model="form.username" autocomplete="off" /></label
        ><label class="field"
          >SMTP Password<input
            v-model="form.password"
            type="password"
            autocomplete="new-password"
            :placeholder="
              configured ? 'Stored securely · leave blank to keep' : 'Enter SMTP password'
            "
          /><span class="text-[9px] text-slate-400">{{
            configured ? 'A password is stored on the server.' : 'No password has been stored.'
          }}</span></label
        ><label class="field"
          >Sender Email<input
            v-model="form.fromEmail"
            type="email"
            :required="form.enabled"
            placeholder="no-reply@example.com" /></label
        ><label class="field">Sender Name<input v-model="form.fromName" maxlength="100" /></label
        ><label v-if="configured" class="flex items-center gap-2 text-[11px]"
          ><input
            v-model="form.clearPassword"
            type="checkbox"
            class="rounded text-rose-600"
          />Remove stored SMTP password</label
        >
      </fieldset>
      <p class="text-[10px] leading-5 text-slate-400">
        Test Connection checks TLS and SMTP authentication without sending an email. Once enabled,
        users can request a one-time password-reset link through Forgot Password.
      </p>
      <div class="flex flex-wrap items-center gap-2">
        <button type="submit" class="btn btn-primary" :disabled="busy || !dirty">
          <AppIcon
            :name="busy ? 'progress_activity' : 'save'"
            :class="busy && 'animate-spin'"
          />Save SMTP Settings</button
        ><button type="button" class="btn" :disabled="busy || !form.enabled" @click="save(true)">
          Test Connection</button
        ><span v-if="dirty" class="text-[10px] text-amber-700">Unsaved SMTP changes</span>
      </div>
    </form>
  </section>
</template>
