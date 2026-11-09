<template>
  <div class="page" role="status" aria-label="Loading fleet data" aria-busy="true">
    <div class="panel p-4">
      <div class="skeleton h-4 w-64 max-w-full"></div>
      <div class="skeleton mt-3 h-2 w-44"></div>
    </div>
    <div class="grid grid-cols-2 gap-3 xl:grid-cols-4">
      <div v-for="n in 4" :key="n" class="panel space-y-3 p-3">
        <div class="skeleton h-2 w-24"></div>
        <div class="skeleton h-6 w-16"></div>
        <div class="skeleton h-2 w-32 max-w-full"></div>
      </div>
    </div>
    <div class="grid gap-3 lg:grid-cols-2">
      <div v-for="n in 2" :key="n" class="panel p-3">
        <div class="skeleton h-3 w-40"></div>
        <div class="skeleton mt-4 h-44 w-full"></div>
      </div>
    </div>
    <div class="panel space-y-3 p-3">
      <div v-for="n in 4" :key="n" class="skeleton h-8 w-full"></div>
    </div>
    <span class="sr-only">Loading vehicle and maintenance data.</span>
  </div>
</template>
