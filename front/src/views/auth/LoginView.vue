<script setup>
import { ref } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useAuth } from '../../composables/useAuth'
const { signIn, sessionError } = useAuth()
const route = useRoute()
import AppIcon from '../../components/ui/AppIcon.vue'
import { authService } from '../../services/auth'
import { authDemo as useMock } from '../../services/auth'
import { useToast } from '../../composables/useToast'
const { notify } = useToast()
function sso(provider) {
  notify(
    `${provider} SSO is not configured. Connect your identity provider when integrating the backend.`,
    'info',
  )
}
const router = useRouter(),
  email = ref(''),
  password = ref(''),
  visible = ref(false),
  remember = ref(true),
  busy = ref(false),
  error = ref('')
async function submit() {
  if (busy.value) return
  busy.value = true
  error.value = ''
  try {
    await signIn({
      email: email.value.trim(),
      password: password.value,
      remember: remember.value,
    })
    const target = String(route.query.redirect || '/dashboard')
    await router.push(
      target.startsWith('/') && !target.startsWith('//') && !target.startsWith('/login')
        ? target
        : '/dashboard',
    )
  } catch (cause) {
    error.value = cause.message
  } finally {
    busy.value = false
  }
}
</script>
<template>
  <div class="panel p-5">
    <h1 class="text-xl font-bold tracking-tight text-slate-950">Welcome Back</h1>
    <p class="mt-2 text-xs leading-5 text-slate-500">
      Sign in to access your Pastrack fleet telematics dashboard.
    </p>
    <div
      v-if="error"
      role="alert"
      class="mt-5 rounded-lg border border-red-200 bg-red-50 p-3 text-xs text-red-700"
    >
      {{ error }}
    </div>
    <p v-if="sessionError" role="alert" class="mt-3 text-xs text-rose-600">{{ sessionError }}</p>
    <form class="mt-5 space-y-4" @submit.prevent="submit">
      <label class="field"
        >Email or Username
        <div class="relative">
          <AppIcon name="mail" class="absolute left-2.5 top-2 !text-base text-slate-400" /><input
            v-model="email"
            type="text"
            name="email"
            autocomplete="username"
            required
            class="h-8 w-full pl-8"
            placeholder="Email address or existing username"
          /></div></label
      ><label class="field"
        ><span class="flex justify-between"
          >Password<RouterLink
            to="/forgot-password"
            class="font-normal text-blue-700 hover:underline"
            >Forgot Password?</RouterLink
          ></span
        >
        <div class="relative">
          <AppIcon name="lock" class="absolute left-2.5 top-2 !text-base text-slate-400" /><input
            v-model="password"
            :type="visible ? 'text' : 'password'"
            name="password"
            aria-label="Password"
            autocomplete="current-password"
            required
            class="h-8 w-full px-8"
            placeholder="Enter your password"
          /><button
            type="button"
            class="absolute right-2.5 top-2 !text-base text-slate-400"
            :aria-label="visible ? 'Hide password' : 'Show password'"
            @click="visible = !visible"
          >
            <AppIcon :name="visible ? 'visibility_off' : 'visibility'" />
          </button></div></label
      ><label class="flex items-center gap-2 text-xs text-slate-500"
        ><input
          v-model="remember"
          type="checkbox"
          class="rounded border-slate-300 text-blue-700"
        />Remember device (30 days)</label
      ><button type="submit" class="btn btn-primary w-full !py-2" :disabled="busy">
        <AppIcon :name="busy ? 'progress_activity' : 'login'" :class="busy && 'animate-spin'" />{{
          busy ? 'Signing in…' : 'Sign In to Pastrack'
        }}
      </button>
    </form>
    <div class="mt-4 space-y-2 border-t border-slate-100 pt-3">
      <p class="mb-3 text-center font-mono text-[8px] tracking-wider text-slate-400">
        OR CONTINUE WITH ENTERPRISE SSO
      </p>
      <button class="btn w-full !py-2" @click="sso('Google Workspace')">
        <span class="text-base font-bold text-blue-600">G</span>Continue with Google Workspace</button
      ><button class="btn w-full !py-2" @click="sso('Okta / SAML')">
        <AppIcon name="domain" class="text-blue-700" />Corporate Okta / SAML Single Sign-On
      </button>
    </div>
    <div
      v-if="useMock"
      class="mt-4 border-t border-slate-100 pt-3 text-[10px] leading-4 text-slate-400"
    >
      Preview with any valid email and password. Use password <code>fail</code> to preview an error.
    </div>
    <RouterLink
      v-if="useMock"
      to="/dashboard"
      class="mt-3 flex items-center justify-center gap-2 text-xs font-medium text-blue-700"
      >Explore demo workspace <AppIcon name="arrow_forward"
    /></RouterLink>
  </div>
</template>
