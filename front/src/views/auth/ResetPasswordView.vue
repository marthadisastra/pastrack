<script setup>
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useAuth } from '../../composables/useAuth'
const { clearSession } = useAuth()
import AppIcon from '../../components/ui/AppIcon.vue'
import { authService } from '../../services/auth'
import { authDemo as useMock } from '../../services/auth'
const route = useRoute(),
  password = ref(''),
  confirm = ref(''),
  visible = ref(false),
  terminate = ref(true),
  busy = ref(false),
  success = ref(false),
  error = ref('')
const rules = computed(() => [
  { label: 'At least 12 characters', ok: password.value.length >= 12 },
  {
    label: 'Uppercase and lowercase letters',
    ok: /[A-Z]/.test(password.value) && /[a-z]/.test(password.value),
  },
  { label: 'At least one number', ok: /\d/.test(password.value) },
  { label: 'At least one special character', ok: /[^\w\s]/.test(password.value) },
])
const strength = computed(() => rules.value.filter((rule) => rule.ok).length)
const valid = computed(() => strength.value === 4 && password.value === confirm.value)
async function submit() {
  if (busy.value) return
  if (!valid.value) return
  busy.value = true
  error.value = ''
  try {
    if (!useMock && !route.query.token)
      throw new Error('Reset token is missing. Request a new reset link.')
    await authService.resetPassword({
      token: route.query.token,
      password: password.value,
      terminateSessions: terminate.value,
    })
    clearSession()
    success.value = true
    password.value = ''
    confirm.value = ''
  } catch (cause) {
    error.value = cause.message
  } finally {
    busy.value = false
  }
}
</script>
<template>
  <section class="panel p-5 sm:p-6">
    <template v-if="success"
      ><div class="mb-4 text-emerald-600"><AppIcon name="verified_user" class="!text-4xl" /></div>
      <h1 class="text-xl font-bold">Credentials Synchronized</h1>
      <p class="mt-3 text-xs leading-6 text-slate-500">
        {{
          useMock
            ? 'Demo password reset complete. No account credentials were changed.'
            : 'Your password has been updated. Sign in with your new credentials.'
        }}
      </p>
      <RouterLink to="/login" class="btn btn-primary mt-5 w-full"
        >Return to Sign In</RouterLink
      ></template
    >
    <template v-else>
      <p class="mb-2 text-[9px] uppercase tracking-wider text-blue-700">
        <span class="rounded bg-indigo-100 px-1.5 py-1">Security Credential Update</span
        ><span class="ml-2 text-slate-400">REQ_ID: #TK-88912-FLT</span>
      </p>
      <h1 class="mt-3 text-xl font-bold tracking-tight text-slate-950">Set New Password</h1>
      <p class="mt-2 text-xs leading-5 text-slate-500">
        Create a resilient security key for your dispatcher account and restore access to your fleet
        workspace.
      </p>
      <p v-if="error" role="alert" class="mt-3 text-xs text-red-600">{{ error }}</p>
      <form class="mt-5 space-y-4" @submit.prevent="submit">
        <div>
          <div class="mb-1.5 flex items-center justify-between">
            <label for="new-password" class="text-[11px] font-medium"
              >New Password <span class="text-rose-500">*</span></label
            ><span class="text-[9px] text-slate-400"
              >STRENGTH: {{ ['LOW', 'LOW', 'FAIR', 'GOOD', 'STRONG'][strength] }}</span
            >
          </div>
          <div class="relative">
            <input
              id="new-password"
              aria-label="New Password"
              v-model="password"
              :type="visible ? 'text' : 'password'"
              autocomplete="new-password"
              required
              minlength="12"
              maxlength="72"
              class="h-9 w-full !border-indigo-100 !bg-indigo-50/70 pr-10"
              placeholder="Enter 12+ characters"
            /><button
              type="button"
              class="absolute right-3 top-2 text-slate-500"
              :aria-label="visible ? 'Hide passwords' : 'Show passwords'"
              @click="visible = !visible"
            >
              <AppIcon :name="visible ? 'visibility_off' : 'visibility'" class="!text-lg" />
            </button>
          </div>
          <div class="mt-2 flex gap-1.5">
            <div
              v-for="n in 4"
              :key="n"
              class="h-1 flex-1 rounded-full"
              :class="
                n <= strength
                  ? strength === 4
                    ? 'bg-emerald-500'
                    : 'bg-blue-500'
                  : 'bg-indigo-100'
              "
            ></div>
          </div>
        </div>
        <div class="rounded-lg bg-indigo-50/80 p-3">
          <div class="mb-3 flex justify-between gap-2 text-[9px]">
            <h2 class="font-medium uppercase tracking-wide">
              Complexity Matrix (Telematics Standard)
            </h2>
            <span>{{ strength }} / 4 Satisfied</span>
          </div>
          <ul class="grid gap-x-3 gap-y-2 sm:grid-cols-2">
            <li
              v-for="rule in rules"
              :key="rule.label"
              class="flex items-center gap-1.5 text-[10px]"
              :class="rule.ok ? 'text-emerald-700' : 'text-slate-500'"
            >
              <AppIcon
                :name="rule.ok ? 'check_circle' : 'radio_button_unchecked'"
                class="shrink-0 !text-sm"
              />{{ rule.label }}
            </li>
            <li class="col-span-full text-[9px] text-slate-400">
              Previous password history is verified by the identity backend.
            </li>
          </ul>
        </div>
        <label class="field"
          >Confirm New Password *
          <div class="relative">
            <input
              aria-label="Confirm Password"
              v-model="confirm"
              :type="visible ? 'text' : 'password'"
              autocomplete="new-password"
              required
              class="h-9 w-full !border-indigo-100 !bg-indigo-50/70 pr-10"
              placeholder="Re-enter your password"
            /><button
              type="button"
              class="absolute right-3 top-2 text-slate-500"
              :aria-label="visible ? 'Hide confirmation password' : 'Show confirmation password'"
              @click="visible = !visible"
            >
              <AppIcon :name="visible ? 'visibility_off' : 'visibility'" class="!text-lg" />
            </button>
          </div>
          <span v-if="confirm && confirm !== password" class="text-red-600"
            >Passwords do not match.</span
          ></label
        >
        <label class="flex items-start gap-2 rounded-lg bg-indigo-50/80 p-3"
          ><input
            v-model="terminate"
            disabled
            type="checkbox"
            class="mt-0.5 rounded border-slate-300 text-blue-700"
          /><span class="text-[11px]"
            >Revoke existing login sessions<span
              class="mt-1 block text-[10px] leading-4 text-slate-500"
              >All existing sessions will be revoked when the password is updated.</span
            ></span
          ></label
        >
        <div class="flex flex-wrap items-center justify-between gap-3 pt-1">
          <RouterLink to="/login" class="flex items-center gap-1 text-[10px] text-slate-500"
            ><AppIcon name="arrow_back" class="!text-base" />Cancel & Return to Sign In</RouterLink
          ><button class="btn btn-primary !px-5 !py-2" :disabled="!valid || busy">
            {{ busy ? 'Updating credentials…' : 'Update Password' }}
          </button>
        </div>
      </form>
    </template>
  </section>
</template>
