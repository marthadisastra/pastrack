<script setup>
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { moduleCatalog } from '../../modules/registry'
import AppIcon from '../../components/ui/AppIcon.vue'
const route = useRoute()
const module = computed(() => moduleCatalog.find((module) => module.id === route.query.module))
</script>
<template>
  <div class="page">
    <section class="panel mx-auto mt-6 max-w-xl p-6 text-center">
      <AppIcon name="extension_off" class="!text-4xl text-slate-300" />
      <h1 class="mt-3 text-lg font-bold">Module unavailable</h1>
      <p class="mt-2 text-xs leading-6 text-slate-500">
        {{ module?.name || 'This module' }} is disabled or not installed in this workspace. Existing
        records have been retained.
      </p>
      <RouterLink to="/settings?tab=modules" class="btn btn-primary mt-4"
        >Open Module Manager</RouterLink
      >
    </section>
  </div>
</template>
