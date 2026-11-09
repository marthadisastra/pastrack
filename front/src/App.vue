<script setup>
import ToastHost from './components/ui/ToastHost.vue'
import PageProgress from './components/feedback/PageProgress.vue'
import ConfirmHost from './components/feedback/ConfirmHost.vue'
</script>
<template><PageProgress /><RouterView /><ToastHost /><ConfirmHost /></template>
