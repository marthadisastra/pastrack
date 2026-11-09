<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import AuthBrandPanel from '../components/fleet/AuthBrandPanel.vue'
import RecoveryContextPanel from '../components/fleet/RecoveryContextPanel.vue'
import AppIcon from '../components/ui/AppIcon.vue'
import StatusBadge from '../components/ui/StatusBadge.vue'
import { authDemo as useMock } from '../services/auth'
const route = useRoute()
const login = computed(() => route.name === 'login')
const reset = computed(() => route.name === 'reset-password')
</script>
<template>
  <div v-if="login" class="auth-login flex min-h-dvh bg-slate-50">
    <AuthBrandPanel />
    <section class="flex w-full flex-col items-center justify-center px-4 py-6 lg:w-[45%] lg:px-8">
      <RouterLink
        to="/dashboard"
        class="mb-5 text-base font-bold tracking-tight text-blue-700 lg:hidden"
        >PASTRACK
        <span class="text-[10px] font-normal text-slate-500">FLEET TELEMATICS</span></RouterLink
      >
      <div class="w-full max-w-sm">
        <p
          v-if="useMock"
          class="mb-3 rounded-md border border-blue-100 bg-blue-50 px-3 py-2 text-[10px] text-blue-700"
        >
          Template preview · Authentication uses demo data.
        </p>
        <RouterView />
        <p class="mt-4 text-center text-[9px] text-slate-400">
          © {{ new Date().getFullYear() }} PASTRACK Telematics
        </p>
      </div>
    </section>
  </div>
  <div v-else class="recovery-layout flex min-h-dvh flex-col bg-[#faf8ff]">
    <header class="mx-auto flex w-full max-w-[1200px] items-center justify-between gap-3 px-4 py-4">
      <RouterLink to="/login" class="flex items-center gap-2"
        ><span
          class="flex h-8 w-8 items-center justify-center rounded bg-blue-700 text-sm font-bold text-white"
          >P</span
        ><span class="text-sm font-bold"
          >PASTRACK<span
            class="block text-[8px] font-normal uppercase tracking-widest text-slate-500"
            >Fleet Telematics OS</span
          ></span
        ></RouterLink
      >
      <div class="flex items-center gap-2">
        <StatusBadge tone="blue">FLEET IDENTITY OS</StatusBadge
        ><StatusBadge v-if="useMock">DEMO</StatusBadge>
      </div>
    </header>
    <main
      class="mx-auto flex w-full max-w-[1120px] flex-1 flex-col justify-center px-4 py-3"
      :class="reset && '!max-w-[1040px]'"
    >
      <div
        v-if="reset"
        class="mb-3 flex flex-wrap items-center justify-between gap-2 rounded-lg bg-indigo-100/60 px-3 py-2.5 text-[10px]"
      >
        <span class="flex items-center gap-2"
          ><AppIcon name="shield_lock" class="!text-lg text-blue-700" />DISPATCH SECURITY ENCLAVE
          <span class="hidden text-slate-400 sm:inline">/ NODE::US-EAST-4A_GATEWAY</span></span
        ><StatusBadge tone="green" dot>{{
          useMock ? 'TOKEN PREVIEW' : 'CREDENTIAL UPDATE'
        }}</StatusBadge>
      </div>
      <div v-if="!reset" class="grid items-stretch gap-4 lg:grid-cols-[.8fr_1.2fr]">
        <RecoveryContextPanel mode="forgot" class="hidden lg:block" /><RouterView />
      </div>
      <div v-else class="grid items-start gap-4 lg:grid-cols-[1.8fr_1fr]">
        <RouterView /><RecoveryContextPanel mode="reset" />
      </div>
    </main>
    <footer
      class="mx-auto flex w-full max-w-[1200px] flex-wrap items-center justify-between gap-2 px-4 py-4 text-[9px] text-slate-400"
    >
      <span>● Operational Grid · Identity & Access</span
      ><span>© {{ new Date().getFullYear() }} PASTRACK Telematics</span>
    </footer>
  </div>
</template>
<style scoped>
.recovery-layout {
  background-image: radial-gradient(#d9d7e8 0.8px, transparent 0.8px);
  background-size: 24px 24px;
}
</style>
