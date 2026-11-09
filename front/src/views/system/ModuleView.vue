<script setup>
import { useRoute } from 'vue-router'
import AppIcon from '../../components/ui/AppIcon.vue'
import StatusBadge from '../../components/ui/StatusBadge.vue'
const route = useRoute()
</script>
<template>
  <div class="page">
    <header class="panel flex items-center gap-3 p-3">
      <span class="flex h-8 w-8 items-center justify-center rounded-lg bg-blue-50 text-blue-700"
        ><AppIcon :name="route.meta.icon"
      /></span>
      <div>
        <p class="text-[9px] text-slate-400">PASTRACK Workspace</p>
        <h1 class="text-sm font-bold">{{ route.meta.title }}</h1>
      </div>
      <StatusBadge class="ml-auto">TEMPLATE</StatusBadge>
    </header>
    <section
      class="blueprint-grid flex min-h-[300px] items-center justify-center rounded-lg border border-dashed border-slate-300 bg-white p-5"
    >
      <div class="panel w-full max-w-md p-5 text-center">
        <AppIcon :name="route.meta.icon" class="mb-3 !text-3xl text-blue-600" />
        <h2 class="text-sm font-semibold">{{ route.meta.title }}</h2>
        <p class="mt-2 text-xs leading-5 text-slate-500">
          Menu ini tersedia pada master layout mockup. Desain detail dan integrasi modul belum
          tersedia.
        </p>
        <RouterLink to="/template" class="btn mt-4">Open Master Layout</RouterLink>
      </div>
    </section>
  </div>
</template>
