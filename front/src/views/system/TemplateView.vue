<script setup>
import AppIcon from '../../components/ui/AppIcon.vue'
import StatusBadge from '../../components/ui/StatusBadge.vue'
</script>
<template>
  <div class="h-full min-h-[650px] p-4 lg:p-6">
    <section
      class="blueprint-grid relative flex h-full min-h-[600px] flex-col rounded-xl border-2 border-dashed border-slate-300 bg-white p-4"
    >
      <div class="flex flex-wrap justify-between gap-2">
        <span
          class="rounded border border-slate-200 bg-slate-50 px-2 py-1 font-mono text-[9px] text-slate-500"
          >SLOT: MAIN_VIEWPORT_CONTAINER · W: 100% FLUID · 12-COL</span
        ><StatusBadge tone="blue" dot>STANDBY FOR MODULE INJECTION</StatusBadge>
      </div>
      <div class="flex flex-1 items-center justify-center py-12">
        <article class="panel w-full max-w-xl p-6 text-center sm:p-8">
          <span
            class="mx-auto mb-5 flex h-16 w-16 items-center justify-center rounded-2xl border border-blue-200 bg-blue-50 text-blue-700"
            ><AppIcon name="fullscreen" class="!text-4xl" /></span
          ><StatusBadge tone="blue" dot>MASTER CONTENT CONTAINER</StatusBadge>
          <h1 class="mt-4 text-xl font-bold tracking-tight text-slate-950">
            Area Konten Baku Siap Digunakan
          </h1>
          <p class="mt-3 text-xs leading-6 text-slate-500">
            Area viewport ini dirancang bersih dan modular sebagai penampung seluruh modul
            telematika: dashboard fleet, live tracking GPS GV30CEU, analisis sensor BBM CFS300B, dan
            telemetri IoT.
          </p>
          <div class="mt-6 grid gap-3 border-t border-slate-100 pt-4 text-left sm:grid-cols-3">
            <div
              v-for="(value, label) in {
                'DEVICE TARGET': 'GV30CEU Gateway',
                'FUEL SENSOR': 'CFS300B Probe',
                'LAYOUT RULE': '100% Fluid Canvas',
              }"
              :key="label"
              class="rounded-lg border border-slate-200 bg-slate-50 p-3"
            >
              <p class="font-mono text-[9px] text-slate-400">{{ label }}</p>
              <p class="mt-2 text-xs font-semibold">{{ value }}</p>
            </div>
          </div>
          <RouterLink to="/dashboard" class="btn btn-primary mt-6"
            >Open Dashboard<AppIcon name="arrow_forward"
          /></RouterLink>
        </article>
      </div>
      <div class="flex flex-wrap justify-between gap-2 font-mono text-[9px] text-slate-400">
        <span>PADDING: 24px · OVERFLOW: AUTO-SCROLL</span
        ><span>QUECLINK GV30CEU + BLE CFS300B</span>
      </div>
    </section>
  </div>
</template>
