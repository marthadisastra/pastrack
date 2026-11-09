<script setup>
import AppIcon from '../ui/AppIcon.vue'
defineProps({ mode: { type: String, default: 'forgot' } })
</script>
<template>
  <aside v-if="mode === 'forgot'" class="space-y-3">
    <section class="panel p-3">
      <div class="mb-4 flex items-center justify-between gap-2 text-[9px]">
        <span class="font-semibold tracking-wide text-teal-700">● TELEMETRY LINK ACTIVE</span
        ><span class="rounded bg-indigo-50 px-1.5 py-1 text-slate-500">GRID: DEMO · 12ms</span>
      </div>
      <p class="text-[9px] uppercase tracking-wider text-slate-500">Active Operations Node</p>
      <h2 class="mt-1 text-lg font-semibold">Asset Security Grid</h2>
      <p class="mt-2 text-xs leading-4 text-slate-500">
        Real-time CAN-bus session monitor and identity management subsystem for Pastrack OS.
      </p>
      <div class="mt-3 rounded-lg bg-indigo-50/80 p-3">
        <div class="flex justify-between gap-2 text-[10px] font-semibold">
          <span class="flex items-center gap-1"
            ><AppIcon name="satellite_alt" class="!text-base text-blue-700" />Route Node
            #882-NORTH</span
          ><span class="text-blue-700">142 UNITS SYNCD</span>
        </div>
        <svg
          viewBox="0 0 340 90"
          class="my-2 h-20 w-full text-blue-700"
          fill="none"
          aria-label="Demo route telemetry"
        >
          <path
            d="M12 70 L65 52 L120 62 L180 28 L245 42 L310 18"
            stroke="currentColor"
            stroke-width="2.5"
          />
          <circle
            v-for="[x, y] in [
              [12, 70],
              [65, 52],
              [120, 62],
              [180, 28],
              [245, 42],
              [310, 18],
            ]"
            :key="x"
            :cx="x"
            :cy="y"
            r="3.5"
            fill="currentColor"
          />
          <circle cx="180" cy="28" r="9" fill="currentColor" opacity=".2" />
        </svg>
        <div class="grid grid-cols-3 border-t border-indigo-100 pt-2 text-[9px]">
          <div
            v-for="(value, label) in {
              LATENCY: '12.4 ms',
              'ENC LEVEL': 'AES-256',
              'AUTH GRID': 'DEMO',
            }"
            :key="label"
          >
            <p class="text-slate-400">{{ label }}</p>
            <p class="mt-1 font-mono font-medium">{{ value }}</p>
          </div>
        </div>
      </div>
      <h3 class="mb-2 mt-3 flex items-center gap-1.5 text-[10px] font-semibold tracking-wide">
        <AppIcon name="shield" class="!text-base text-blue-700" />FLEET ACCESS GOVERNANCE PROTOCOLS
      </h3>
      <div class="space-y-2">
        <div
          v-for="item in [
            {
              icon: 'key',
              title: '256-Bit Hardware Token Rotation',
              text: 'Time-bound password reset links for fleet operators.',
            },
            {
              icon: 'power_settings_new',
              title: 'CAN-bus Session Disconnect',
              text: 'Session revocation is handled by the identity backend.',
            },
            {
              icon: 'support_agent',
              title: 'Fleet Operations Security',
              text: 'Review suspicious requests with your dispatch administrator.',
            },
          ]"
          :key="item.title"
          class="flex gap-2 rounded-md bg-indigo-50/80 p-2.5"
        >
          <AppIcon :name="item.icon" class="shrink-0 !text-base text-teal-700" />
          <div>
            <h4 class="text-[11px] font-medium">{{ item.title }}</h4>
            <p class="mt-1 text-[10px] leading-4 text-slate-500">{{ item.text }}</p>
          </div>
        </div>
      </div>
    </section>
    <div
      class="flex items-center justify-between rounded-lg bg-indigo-100/50 px-3 py-2.5 text-[10px]"
    >
      <span class="flex items-center gap-2"
        ><AppIcon name="terminal" class="!text-base text-blue-700" />FLEET OS · v4.18.2-PROD</span
      ><span class="font-mono text-[8px] text-slate-500">NODE: US-EAST-04</span>
    </div>
  </aside>
  <aside v-else class="space-y-3">
    <section class="panel p-3">
      <h2 class="flex items-center gap-2 text-sm font-semibold">
        <AppIcon name="verified_user" class="!text-lg text-blue-700" />Telematics Compliance
      </h2>
      <p class="mt-3 text-xs leading-4 text-slate-500">
        Credential governance for dispatcher identities and fleet access.
      </p>
      <ul class="mt-3 space-y-2 text-[11px] leading-5 text-slate-600">
        <li
          v-for="text in [
            'Strong password complexity requirements.',
            'Refresh token and session lifecycle management.',
            'Secure gateway access policies.',
          ]"
          :key="text"
          class="flex gap-2"
        >
          <AppIcon name="check" class="mt-0.5 shrink-0 !text-base text-teal-700" />{{ text }}
        </li>
      </ul>
      <div class="mt-3 rounded bg-indigo-50 p-2.5 text-[9px] leading-5 text-slate-500">
        <p class="font-medium uppercase">Identity security</p>
        <p>Policy enforcement connects to the Go backend.</p>
      </div>
    </section>
    <section class="panel p-3">
      <h2 class="flex items-center gap-2 text-sm font-semibold">
        <AppIcon name="hub" class="!text-lg text-teal-700" />Telemetry Network Lock
      </h2>
      <p class="mt-3 text-xs leading-4 text-slate-500">
        Updating credentials can revoke dispatcher sessions across connected fleet terminals.
      </p>
      <dl class="mt-3 space-y-2 text-[10px]">
        <div
          v-for="(value, label) in {
            'Class-8 Heavy Haulers': '54 Linked',
            'Urban EV Vans': '32 Linked',
            'Regional Yard Relays': '12 Active',
          }"
          :key="label"
          class="flex justify-between rounded bg-indigo-50 px-2 py-1"
        >
          <dt>{{ label }}</dt>
          <dd>{{ value }}</dd>
        </div>
      </dl>
      <p class="mt-3 font-mono text-[9px] text-slate-400">DEMO NETWORK TOPOLOGY</p>
    </section>
    <div class="flex gap-2 rounded-lg bg-indigo-100/50 p-3">
      <AppIcon name="support_agent" class="text-blue-700" />
      <p class="text-[11px] leading-5 text-slate-500">
        Need operational clearance?<br />Contact your Dispatch NOC administrator.
      </p>
    </div>
  </aside>
</template>
