<script setup>
import { ref } from 'vue'
import AppIcon from '../../components/ui/AppIcon.vue'
import { authService } from '../../services/auth'
import { authDemo as useMock } from '../../services/auth'
import { useToast } from '../../composables/useToast'
const { notify } = useToast()
const email = ref(''),
  busy = ref(false),
  sent = ref(false),
  error = ref('')
async function submit() {
  if (busy.value) return
  busy.value = true
  error.value = ''
  try {
    await authService.forgotPassword(email.value.trim())
    sent.value = true
  } catch (cause) {
    error.value = cause.message
  } finally {
    busy.value = false
  }
}
</script>
<template>
  <section class="panel flex min-h-[480px] flex-col p-5 sm:p-6">
    <div class="mb-6 flex items-center justify-between gap-2">
      <RouterLink to="/login" class="flex items-center gap-1.5 text-[11px] text-blue-700"
        ><AppIcon name="arrow_back" class="!text-base" />Return to Login</RouterLink
      ><span class="rounded bg-indigo-50 px-2 py-1 text-[9px] text-slate-500"
        >FLEET IDENTITY OS</span
      >
    </div>
    <p class="mb-2 text-[10px] font-semibold tracking-widest text-blue-700">CREDENTIAL RECOVERY</p>
    <h1 class="text-2xl font-bold tracking-tight text-slate-950">Recover Pastrack Access</h1>
    <p class="mt-3 text-xs leading-6 text-slate-500">
      Enter your enterprise fleet work email. If an active operator or dispatcher account is
      registered, a password reset link will be dispatched.
    </p>
    <div
      v-if="sent"
      role="status"
      class="mt-4 rounded-lg border border-emerald-200 bg-emerald-50 p-3 text-[11px] leading-5 text-emerald-800"
    >
      <strong>{{ useMock ? 'Demo reset request received.' : 'Check your inbox.' }}</strong>
      <p>
        {{
          useMock
            ? 'No email has been sent. Continue below to preview the reset screen.'
            : 'If an account exists for this address, a reset link will be sent.'
        }}
      </p>
      <RouterLink
        v-if="useMock"
        to="/reset-password?token=demo-preview"
        class="mt-2 inline-block font-semibold underline"
        >Preview set new password →</RouterLink
      >
    </div>
    <p v-if="error" role="alert" class="mt-4 text-xs text-red-600">{{ error }}</p>
    <form class="mt-6 space-y-4" @submit.prevent="submit">
      <label for="recovery-email" class="flex justify-between text-[11px] font-medium"
        >Work Email Address<span class="text-[8px] font-normal text-slate-400"
          >OPERATOR DOMAIN REQUIRED</span
        ></label
      >
      <div class="relative !mt-2">
        <AppIcon name="mail" class="absolute left-3 top-2.5 !text-lg text-slate-500" /><input
          id="recovery-email"
          v-model="email"
          type="email"
          autocomplete="email"
          required
          placeholder="dispatcher@enterprise-fleet.com"
          class="h-10 w-full !border-indigo-100 !bg-indigo-50/70 pl-10 pr-16"
        /><button
          v-if="useMock"
          type="button"
          class="absolute right-2 top-2 rounded bg-indigo-100/70 px-2 py-1 text-[9px]"
          @click="email = 'dispatcher@enterprise-fleet.com'"
        >
          Sample
        </button>
      </div>
      <p v-if="useMock" class="flex items-start gap-2 text-[10px] leading-4 text-slate-500">
        <AppIcon name="info" class="shrink-0 !text-base text-blue-700" />Single Sign-On users:
        authenticate directly via your Okta, Azure AD, or Google Workspace dashboard.
      </p>
      <div
        class="flex items-center justify-between gap-2 rounded-md bg-indigo-50 px-3 py-2.5 text-[9px]"
      >
        <span class="flex items-center gap-1"
          ><AppIcon name="verified_user" class="!text-base text-teal-700" />Credential Recovery
          Gateway</span
        ><span class="font-semibold text-teal-700">{{
          useMock ? 'DEMO PREVIEW' : 'IDENTITY ACCESS'
        }}</span>
      </div>
      <div class="grid gap-2 sm:grid-cols-[1.4fr_1fr]">
        <button class="btn btn-primary !py-2.5" :disabled="busy">
          <AppIcon name="send" />{{
            busy ? 'Requesting link…' : sent ? 'Request Another Link' : 'Send Reset Link'
          }}</button
        ><button
          v-if="useMock"
          type="button"
          class="btn !border-indigo-100 !bg-indigo-50 !py-2.5"
          @click="
            notify(
              'SSO gateway is not configured. Connect your identity provider with the backend.',
              'info',
            )
          "
        >
          <AppIcon name="domain" />SSO Enterprise Gateway
        </button>
      </div>
    </form>
    <div
      class="mt-auto flex items-start gap-2 border-t border-indigo-100 pt-4 text-[10px] leading-5 text-slate-400"
    >
      <AppIcon name="lock" class="mt-0.5 shrink-0 !text-base text-blue-700" />
      <p>
        Fleet credential recovery and hardware security policies are managed by your identity
        provider.
      </p>
    </div>
  </section>
</template>
