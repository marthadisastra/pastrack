<script setup>
import { dashboardDemo } from '../../services/dashboard'
import DashboardDemoView from './DashboardDemoView.vue'
import DashboardSnapshotView from './DashboardSnapshotView.vue'
</script>
<template>
  <DashboardDemoView v-if="dashboardDemo" />
  <DashboardSnapshotView v-else />
</template>
