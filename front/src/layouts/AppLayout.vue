<script setup>
import { computed, ref, watch, onMounted, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useAuth } from '../composables/useAuth'
const { user, signOut, watchSession, authDemo } = useAuth()
const router = useRouter()
const signingOut = ref(false)
import AppIcon from '../components/ui/AppIcon.vue'
import StatusBadge from '../components/ui/StatusBadge.vue'
import { useFleet } from '../composables/useFleet'
import { useMock } from '../services/http'
import { dashboardDemo } from '../services/dashboard'
import { useDashboard } from '../composables/useDashboard'
import { navigation } from '../router/navigation'
import { isModuleActive, settingsState } from '../modules/state'
import { useToast } from '../composables/useToast'
import NotificationBell from '../components/notifications/NotificationBell.vue'
import PageSkeleton from '../components/feedback/PageSkeleton.vue'
import ConnectionStatus from '../components/feedback/ConnectionStatus.vue'
const route = useRoute()
const liveDashboard = computed(
  () => (route.path === '/dashboard' || route.path === '/tracking') && !dashboardDemo,
)
const dashboard = useDashboard()
const open = ref(false),
  collapsed = ref(false),
  clock = ref(''),
  query = ref('')
const { vehicles, loading, error, reconnect, startRealtime, stopRealtime } = useFleet()
const { notify } = useToast()
const monitoredCount = computed(() =>
  liveDashboard.value
    ? (dashboard.summary.value?.monitoredVehicles ?? 0)
    : vehicles.value.filter((v) => v.imei).length,
)
const meanCsq = computed(() => {
  const assets = vehicles.value.filter((v) => v.imei && Number.isFinite(v.csq))
  return assets.length
    ? (assets.reduce((sum, v) => sum + v.csq, 0) / assets.length).toFixed(1)
    : '—'
})
const links = computed(() =>
  navigation
    .map((group) => ({
      ...group,
      items: group.items.filter((item) => isModuleActive(item.moduleId)),
    }))
    .filter((group) => group.items.length),
)
const brand = computed(() => settingsState.value.branding)
const alertCount = computed(() => vehicles.value.filter((v) => v.state === 'alert').length)
watch(liveDashboard, (live) => (live ? stopRealtime() : startRealtime()), { immediate: true })
let timer, stopSessionWatch
onMounted(() => {
  if (!dashboardDemo) dashboard.start()
  stopSessionWatch = watchSession(() =>
    router.replace({ name: 'login', query: { redirect: route.fullPath } }),
  )
  const tick = () => {
    clock.value = new Date().toLocaleTimeString('en-GB', { timeZone: 'Asia/Jakarta' })
  }
  tick()
  timer = setInterval(tick, 1000)
})
onUnmounted(() => {
  if (!dashboardDemo) dashboard.stop()
  clearInterval(timer)
  stopRealtime()
  stopSessionWatch?.()
})
async function logout() {
  if (signingOut.value) return
  signingOut.value = true
  try {
    await signOut()
    await router.replace('/login')
  } catch (cause) {
    notify('Unable to sign out. Check your connection and try again.', 'error')
  } finally {
    signingOut.value = false
  }
}
async function refresh() {
  const ok = await (liveDashboard.value ? dashboard.load() : reconnect())
  notify(
    ok ? 'Fleet data synchronized.' : 'Synchronization failed. Last received values are retained.',
    ok ? 'success' : 'error',
  )
}
</script>
<template>
  <div class="workspace flex h-dvh overflow-hidden">
    <button
      v-if="open"
      class="fixed inset-0 z-40 bg-slate-950/40 lg:hidden"
      aria-label="Close navigation"
      @click="open = false"
    ></button>
    <aside
      class="sidebar fixed inset-y-0 left-0 z-50 flex w-[232px] shrink-0 flex-col border-r border-slate-200 bg-white transition-[width,transform] duration-200 lg:static lg:translate-x-0"
      :class="[
        open ? 'translate-x-0' : '-translate-x-full',
        collapsed ? 'sidebar-collapsed lg:w-14' : 'lg:w-[216px]',
      ]"
    >
      <RouterLink
        to="/dashboard"
        class="flex h-12 shrink-0 items-center gap-2.5 border-b border-slate-200 px-3"
        :aria-label="`${brand.name} Dashboard`"
        @click="open = false"
      >
        <span
          class="flex h-7 w-7 shrink-0 items-center justify-center rounded-lg bg-blue-700 text-white shadow-sm"
          ><img
            v-if="brand.logo"
            :src="brand.logo"
            :alt="brand.name"
            class="h-6 w-6 object-contain" /><AppIcon v-else name="radar" class="!text-lg"
        /></span>
        <span class="sidebar-label"
          ><span
            class="flex items-center gap-1.5 text-[13px] font-bold tracking-tight text-slate-950"
            >{{ brand.name }}
            <span
              class="rounded border border-blue-200 bg-blue-50 px-1 text-[8px] font-medium text-blue-700"
              >OS</span
            ></span
          ><span class="mt-0.5 block text-[8px] tracking-[.15em] text-slate-400">{{
            brand.tagline
          }}</span></span
        ><span class="sidebar-label ml-auto h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
      </RouterLink>
      <nav
        class="sidebar-nav min-h-0 flex-1 space-y-3 overflow-y-auto px-2 py-2.5"
        aria-label="Main navigation"
      >
        <div v-for="group in links" :key="group.group">
          <p
            class="sidebar-heading mb-1 px-2 text-[9px] font-bold uppercase tracking-wider text-slate-400"
          >
            {{ group.group }}
          </p>
          <RouterLink
            v-for="link in group.items"
            :key="link.to"
            :to="link.to"
            :title="link.label"
            :aria-label="link.label"
            class="sidebar-link mb-0.5 flex h-7 items-center gap-2 rounded-md px-2 text-[11px] font-medium transition-colors"
            :class="
              route.path === link.to
                ? 'bg-blue-700 text-white shadow-sm'
                : 'text-slate-600 hover:bg-slate-100 hover:text-slate-900'
            "
            @click="open = false"
          >
            <AppIcon
              :name="link.icon"
              class="shrink-0 !text-[17px]"
              :class="route.path !== link.to && 'text-slate-400'"
            /><span class="sidebar-label whitespace-nowrap">{{ link.label }}</span>
            <span
              v-if="link.badge"
              class="sidebar-label ml-auto rounded border px-1.5 py-px font-mono text-[8px]"
              :class="
                link.badge === 'alerts'
                  ? 'border-rose-200 bg-rose-50 text-rose-600'
                  : 'border-slate-200 bg-slate-100 text-slate-500'
              "
              >{{
                liveDashboard
                  ? link.badge === 'assets'
                    ? (dashboard.summary.value?.totalVehicles ?? '—')
                    : (dashboard.summary.value?.activeAlerts ?? '—')
                  : link.badge === 'assets'
                    ? vehicles.length
                    : alertCount
              }}</span
            >
            <span
              v-else-if="link.dot"
              class="sidebar-label ml-auto h-1.5 w-1.5 rounded-full bg-cyan-500"
            ></span
            ><span
              v-else-if="route.path === link.to"
              class="sidebar-label ml-auto h-1.5 w-1.5 rounded-full bg-white/80"
            ></span>
          </RouterLink>
        </div>
      </nav>
      <div class="sidebar-bottom shrink-0 space-y-1.5 border-t border-slate-200 bg-slate-50 p-2">
        <div
          v-if="!liveDashboard"
          class="sidebar-label rounded-lg border border-slate-200 bg-white p-2"
        >
          <div class="mb-1.5 flex items-center justify-between text-[8px] font-bold tracking-wide">
            <span class="flex items-center gap-1 text-slate-800"
              ><AppIcon name="hub" class="!text-sm text-blue-700" />HARDWARE BRIDGE</span
            ><span
              class="rounded border border-emerald-200 bg-emerald-50 px-1 text-[8px] text-emerald-700"
              >{{ useMock ? 'DEMO' : 'API' }}</span
            >
          </div>
          <div class="flex justify-between font-mono text-[9px]">
            <span>GV30CEU Gateway</span
            ><span class="text-blue-700">{{ useMock ? 'Demo' : 'SSE' }}</span>
          </div>
          <div
            class="mt-1.5 flex justify-between border-t border-slate-100 pt-1.5 font-mono text-[8px] text-slate-500"
          >
            <span>CSQ: {{ meanCsq }}/31</span><span>PROTO: GTFRI</span>
          </div>
          <div class="mt-1 flex justify-between font-mono text-[8px] text-slate-500">
            <span>BLE 5.0 LINK</span
            ><span class="text-emerald-700">{{ useMock ? 'SIMULATED' : 'CONFIGURED' }}</span>
          </div>
        </div>
        <button
          type="button"
          class="flex w-full items-center gap-2 rounded-md px-1 py-1 text-left hover:bg-white"
          :disabled="signingOut"
          aria-label="Sign out"
          title="Sign out"
          @click="logout"
        >
          <span
            class="flex h-7 w-7 shrink-0 items-center justify-center rounded-md bg-slate-900 text-white"
            ><AppIcon name="admin_panel_settings" class="!text-base" /></span
          ><span class="sidebar-label text-[11px] font-medium"
            >{{ user?.name || 'NOC Dispatcher'
            }}<span class="block text-[9px] text-slate-400">{{
              authDemo ? 'Demo workspace' : user?.role
            }}</span></span
          ><AppIcon name="logout" class="sidebar-label ml-auto !text-base text-slate-400" />
        </button>
      </div>
    </aside>
    <div class="flex min-w-0 flex-1 flex-col">
      <header
        class="workspace-header flex h-12 shrink-0 items-center gap-2 border-b border-slate-200 bg-white px-3"
      >
        <button class="btn !p-1 lg:hidden" aria-label="Open navigation" @click="open = true">
          <AppIcon name="menu" />
        </button>
        <button
          class="btn hidden !p-1 lg:inline-flex"
          :aria-label="collapsed ? 'Expand sidebar' : 'Collapse sidebar'"
          :aria-expanded="!collapsed"
          @click="collapsed = !collapsed"
        >
          <AppIcon :name="collapsed ? 'left_panel_open' : 'left_panel_close'" />
        </button>
        <span
          class="hidden text-[10px] font-bold uppercase tracking-wider text-blue-700 sm:block"
          >{{ brand.name }}</span
        ><span class="hidden h-4 w-px bg-slate-200 sm:block"></span>
        <StatusBadge :tone="useMock ? 'green' : 'blue'" dot>{{
          liveDashboard
            ? dashboard.connection.value.status === 'live'
              ? 'WebSocket live'
              : 'Realtime connecting'
            : useMock
              ? 'Demo'
              : 'API Mode'
        }}</StatusBadge>
        <span
          class="hidden rounded border border-slate-200 bg-slate-50 px-2 py-1 font-mono text-[10px] text-slate-500 xl:block"
          >{{ clock }} WIB</span
        >
        <span
          class="hidden items-center gap-1.5 rounded border border-blue-100 bg-blue-50 px-2 py-1 text-[10px] text-blue-700 md:flex"
          ><AppIcon name="local_shipping" class="!text-base" />{{ monitoredCount }}
          <span class="text-slate-500">Monitored</span></span
        >
        <form
          class="ml-auto hidden max-w-[280px] flex-1 md:block"
          @submit.prevent="$router.push({ path: '/vehicles', query: { q: query } })"
        >
          <label class="sr-only" for="global-search">Search assets</label>
          <div class="relative">
            <AppIcon
              class="absolute left-2 top-1.5 !text-base text-slate-400"
              name="search"
            /><input
              id="global-search"
              v-model="query"
              class="h-7 w-full !bg-slate-50 pl-7 !text-[11px]"
              placeholder="Search IMEI, Plate, Asset ID…"
            />
          </div>
        </form>
        <NotificationBell />
        <button
          class="btn !p-1"
          :disabled="liveDashboard ? dashboard.loading.value : loading"
          aria-label="Refresh fleet data"
          @click="refresh"
        >
          <AppIcon :class="loading && 'animate-spin'" name="refresh" />
        </button>
        <button
          class="btn btn-primary"
          @click="
            notify(
              'Dispatch is a template action. Connect your dispatch workflow when the backend is ready.',
              'info',
            )
          "
        >
          <AppIcon name="add" /><span>Dispatch</span>
        </button>
        <RouterLink
          to="/settings"
          class="ml-1 hidden h-7 w-7 items-center justify-center rounded-full border border-blue-200 bg-blue-50 text-blue-700 sm:flex"
          aria-label="User settings"
          ><AppIcon name="manage_accounts" class="!text-base"
        /></RouterLink>
      </header>
      <ConnectionStatus v-if="!liveDashboard" />
      <main
        id="main-content"
        class="min-h-0 flex-1 overflow-y-auto"
        :aria-busy="!liveDashboard && loading && !vehicles.length"
      >
        <div
          v-if="!liveDashboard && error"
          role="alert"
          class="m-3 flex items-center justify-between rounded-lg border border-red-200 bg-red-50 p-2 text-xs text-red-700"
        >
          <span>{{ error }}</span
          ><button class="btn" @click="refresh">Retry</button>
        </div>
        <PageSkeleton v-if="!liveDashboard && loading && !vehicles.length" />
        <RouterView v-else v-slot="{ Component }"
          ><Transition name="page" mode="out-in"
            ><component :is="Component" :key="route.path" /></Transition
        ></RouterView>
      </main>
      <footer
        class="flex h-6 shrink-0 items-center justify-between gap-2 overflow-hidden whitespace-nowrap border-t border-slate-200 bg-white px-3 font-mono text-[8px] text-slate-500"
      >
        <span
          ><span
            class="mr-1.5 inline-block h-1.5 w-1.5 rounded-full"
            :class="useMock ? 'bg-amber-400' : 'bg-blue-500'"
          ></span
          >{{
            liveDashboard
              ? 'WEBSOCKET · Event-driven dashboard updates'
              : useMock
                ? 'DEMO · No live telemetry connection'
                : 'Go API transport configured'
          }}</span
        ><span class="hidden md:block">Queclink GV30CEU · BLE CFS300B</span
        ><RouterLink to="/template" class="text-slate-400 hover:text-blue-700"
          >Master Layout</RouterLink
        >
      </footer>
    </div>
  </div>
</template>
<style scoped>
@media (min-width: 1024px) {
  .sidebar-collapsed .sidebar-label,
  .sidebar-collapsed .sidebar-heading {
    display: none;
  }
  .sidebar-collapsed .sidebar-nav {
    padding-left: 6px;
    padding-right: 6px;
  }
  .sidebar-collapsed .sidebar-link {
    justify-content: center;
    padding: 0;
    height: 30px;
  }
  .sidebar-collapsed .sidebar-bottom {
    padding: 8px 12px;
  }
}
</style>
