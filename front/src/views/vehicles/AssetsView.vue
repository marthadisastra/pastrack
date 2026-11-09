<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import LiveValue from '../../components/feedback/LiveValue.vue'
import { useUnsavedChanges } from '../../composables/useUnsavedChanges'
import { assetsDemoMode, assetsService } from '../../services/assets'
import { useToast } from '../../composables/useToast'
import { useConfirm } from '../../composables/useConfirm'
import AppIcon from '../../components/ui/AppIcon.vue'
import StatusBadge from '../../components/ui/StatusBadge.vue'
import StatCard from '../../components/ui/StatCard.vue'
import AppModal from '../../components/ui/AppModal.vue'
import VehicleInspectModal from '../../components/fleet/VehicleInspectModal.vue'
const route = useRoute(),
  { notify } = useToast(),
  { ask } = useConfirm()
const vehicles = ref([])
const query = ref(String(route.query.q || '')),
  fleet = ref(''),
  gps = ref(''),
  view = ref('list'),
  page = ref(1),
  perPage = 6
const showAdd = ref(false),
  showPair = ref(false),
  showInspect = ref(false),
  editing = ref(false),
  selected = ref(null),
  imei = ref(''),
  busy = ref(false),
  error = ref('')
const initial = () => ({
  id: '',
  plate: '',
  model: '',
  fleetCode: '',
  capacity: 400,
})
const form = reactive(initial())
const assetDirty = computed(() => JSON.stringify(form) !== JSON.stringify(initial()))
useUnsavedChanges(
  computed(() => (showAdd.value && assetDirty.value) || (showPair.value && Boolean(imei.value))),
)
const fleets = computed(() => [...new Set(vehicles.value.map((v) => v.fleet_code).filter(Boolean))])
const loading = ref(false),
  loadError = ref(''),
  devices = ref([])
async function loadAssets() {
  loading.value = true
  loadError.value = ''
  try {
    vehicles.value = await assetsService.list()
  } catch (cause) {
    loadError.value = cause.message || 'Could not load vehicle registry.'
  } finally {
    loading.value = false
  }
}
onMounted(loadAssets)
const paired = computed(() => vehicles.value.filter((v) => v.imei).length)
const capacity = computed(() => vehicles.value.reduce((sum, v) => sum + v.capacity, 0))
const filtered = computed(() =>
  vehicles.value.filter(
    (v) =>
      (!fleet.value || v.fleet_code === fleet.value) &&
      (!gps.value || Boolean(v.imei) === (gps.value === 'paired')) &&
      `${v.id} ${v.plate} ${v.model} ${v.imei} ${v.fleet_code || ''}`
        .toLowerCase()
        .includes(query.value.toLowerCase()),
  ),
)
const pages = computed(() => Math.max(1, Math.ceil(filtered.value.length / perPage)))
const visible = computed(() =>
  filtered.value.slice((page.value - 1) * perPage, page.value * perPage),
)
watch([query, fleet, gps], () => {
  page.value = 1
})
watch(
  () => route.query.q,
  (q) => {
    query.value = String(q || '')
  },
)
function openAdd() {
  editing.value = false
  selected.value = null
  Object.assign(form, initial())
  error.value = ''
  showAdd.value = true
}
function openEdit(vehicle) {
  editing.value = true
  selected.value = vehicle
  Object.assign(form, {
    ...initial(),
    id: vehicle.id,
    plate: vehicle.plate,
    model: vehicle.model,
    fleetCode: vehicle.fleet_code || '',
    capacity: vehicle.capacity || 1,
  })
  error.value = ''
  showAdd.value = true
}
function describeSaveError(cause) {
  const code = cause.details?.error?.code
  const messages = {
    invalid_vehicle_field:
      'Periksa kode aset, nama kendaraan, plat nomor, kode armada, dan kapasitas tangki. Kode maks. 64 karakter, nama 100, plat 32; kapasitas harus lebih dari 0.',
    invalid_vehicle_form: 'Format data aset ditolak server. Periksa kembali isian lalu coba lagi.',
    invalid_or_conflicting_record: 'Kode aset atau plat nomor sudah dipakai, atau nilai tidak sesuai. Periksa data lalu coba lagi.',
    permission_denied: 'Akun ini tidak memiliki izin untuk mengubah registry kendaraan.',
    live_data_read_only: 'Server live sedang dalam mode hanya baca.',
    'Data kendaraan berubah. Muat ulang sebelum menyimpan.':
      'Data aset berubah sejak dibuka. Muat ulang halaman lalu edit kembali.',
  }
  return messages[code] || messages[cause.message] || cause.message || 'Aset gagal disimpan.'
}
function openPair(v) {
  selected.value = v
  imei.value = ''
  error.value = ''
  showPair.value = true
}
function inspect(v) {
  selected.value = v
  showInspect.value = true
}
async function remove(vehicle) {
  const confirmed = await ask({
    title: 'Delete fleet asset?',
    message: `Delete ${vehicle.id} (${vehicle.plate})? The asset will be removed from the active registry and its current gateway assignment will end.`,
    confirmLabel: 'Delete asset',
    cancelLabel: 'Keep asset',
  })
  if (!confirmed || busy.value) return
  busy.value = true
  error.value = ''
  try {
    await assetsService.remove(vehicle)
    vehicles.value = vehicles.value.filter(
      (item) => item.backendId !== vehicle.backendId && item.id !== vehicle.id,
    )
    await loadAssets()
    if (page.value > pages.value) page.value = pages.value
    notify('Asset deleted from the registry.')
  } catch (cause) {
    error.value = cause.details?.error?.message || cause.message || 'Asset could not be deleted.'
  } finally {
    busy.value = false
  }
}
async function save() {
  if (busy.value) return
  busy.value = true
  error.value = ''
  try {
    if (!form.id.trim() || !form.plate.trim() || !form.model.trim())
      throw new Error('Asset ID, license plate and model are required.')
    const payload = {
      code: form.id.trim().toUpperCase(),
      plate: form.plate.trim().toUpperCase(),
      model: form.model.trim(),
      fleetCode: form.fleetCode.trim(),
      capacity: Number(form.capacity),
    }
    if (editing.value) await assetsService.update(selected.value, payload)
    else await assetsService.create(payload)
    await loadAssets()
    showAdd.value = false
    query.value = ''
    fleet.value = ''
    gps.value = ''
    page.value = Math.ceil(vehicles.value.length / perPage)
    notify(editing.value ? 'Asset changes saved.' : 'Asset added to the registry.')
  } catch (cause) {
    error.value = describeSaveError(cause)
  } finally {
    busy.value = false
  }
}
async function pair() {
  if (busy.value) return
  busy.value = true
  error.value = ''
  try {
    const identifier = imei.value.trim()
    let deviceId = identifier
    if (!assetsDemoMode) {
      if (!/^\d{4}$/.test(identifier)) throw new Error('Enter the last 4 digits of the registered gateway IMEI.')
      const matches = devices.value.filter((device) => device.active && device.identifier.endsWith(identifier))
      if (matches.length !== 1) throw new Error(matches.length ? 'More than one gateway matches. Contact an administrator to resolve the duplicate suffix.' : 'No active registered gateway matches that suffix.')
      if (matches[0].vehicleId) throw new Error('This gateway is already assigned to a vehicle.')
      deviceId = matches[0].id
    } else if (!/^\d{15}$/.test(identifier)) {
      throw new Error('Enter a valid 15-digit gateway IMEI.')
    }
    await assetsService.pair(selected.value, deviceId, identifier)
    await loadAssets()
    showPair.value = false
    notify('Gateway paired to the asset.')
  } catch (cause) {
    error.value = cause.message
  } finally {
    busy.value = false
  }
}
async function openPairForAsset(vehicle) {
  openPair(vehicle)
  if (assetsDemoMode) return
  try {
    devices.value = await assetsService.devices()
  } catch (cause) {
    error.value = cause.message || 'Could not load registered gateways.'
  }
}
</script>
<template>
  <div class="page">
    <header class="panel flex flex-wrap items-center justify-between gap-3 p-3">
      <div>
        <p class="mb-1 text-[10px] text-slate-400">Fleet Management › Vehicles</p>
        <h1 class="text-sm font-bold text-slate-950">Vehicles: All Assets</h1>
      </div>
      <div class="flex flex-wrap items-center gap-3">
        <StatusBadge tone="green"
          >GPS GATEWAY: {{ paired }}/{{ vehicles.length }} PAIRED</StatusBadge
        ><button class="btn btn-primary" @click="openAdd"><AppIcon name="add" />Add Asset</button>
      </div>
    </header>
    <div class="grid grid-cols-2 gap-3 xl:grid-cols-4">
      <StatCard label="Total Fleet Assets" :value="vehicles.length" icon="local_shipping"
        >Registered vehicles in this workspace</StatCard
      ><StatCard label="GPS Gateways Paired" :value="paired" icon="sensors"
        ><span class="text-emerald-600">Registered GPS gateways</span></StatCard
      ><StatCard label="Unpaired GPS" :value="vehicles.length - paired" icon="link_off"
        ><span class="text-amber-600">Chassis idle / Needs pairing</span></StatCard
      ><StatCard
        label="Monitored Tank Capacity"
        :value="`${capacity.toLocaleString('en-US')} L`"
        icon="propane_tank"
        >Configured vehicle tank capacity</StatCard
      >
    </div>
    <section class="panel flex flex-wrap items-center gap-3 p-4">
      <label class="relative min-w-48 flex-1"
        ><span class="sr-only">Search assets</span
        ><AppIcon name="search" class="absolute left-3 top-2.5 text-slate-400" /><input
          v-model="query"
          placeholder="Search asset code, plate, vehicle name, or gateway suffix…"
          class="h-8 w-full pl-10 !text-xs" /></label
      ><label
        ><span class="sr-only">Fleet</span
        ><select v-model="fleet" class="h-8 !text-xs">
          <option value="">All Fleets</option>
          <option v-for="item in fleets" :key="item" :value="item">{{ item }}</option>
        </select></label
      ><label
        ><span class="sr-only">GPS status</span
        ><select v-model="gps" class="h-8 !text-xs">
          <option value="">All GPS Statuses</option>
          <option value="paired">Paired</option>
          <option value="unpaired">Unpaired</option>
        </select></label
      ><span class="text-[10px] text-slate-400">{{ filtered.length }} units</span>
      <div class="flex rounded-lg border border-slate-200 bg-slate-50 p-1">
        <button
          v-for="item in ['list', 'grid']"
          :key="item"
          class="flex items-center gap-1.5 rounded px-2 py-1.5 text-xs capitalize"
          :class="view === item && 'bg-white font-semibold text-blue-700 shadow-sm'"
          :aria-pressed="view === item"
          @click="view = item"
        >
          <AppIcon
            :name="item === 'list' ? 'format_list_bulleted' : 'grid_view'"
            class="!text-base"
          />{{ item }}
        </button>
      </div>
    </section>
    <section v-if="loadError" role="alert" class="panel flex items-center justify-between p-4 text-xs text-red-700">
      <span>{{ loadError }}</span><button class="btn" @click="loadAssets">Retry</button>
    </section>
    <section v-if="loading" class="panel p-10 text-center text-xs text-slate-400">Loading asset registry…</section>
    <section v-else-if="view === 'list'" class="panel">
      <div class="overflow-x-auto">
        <table class="w-full min-w-[1050px] text-xs">
          <thead class="border-b border-slate-200 bg-slate-50 font-mono">
            <tr>
              <th>Asset Identity</th>
              <th>Vehicle Specifications</th>
              <th>Tank Capacity</th>
              <th>GPS Gateway</th>
              <th>Fuel Sensor</th>
              <th>Operational Status</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr
              v-for="v in visible"
              :key="v.id"
              class="hover:bg-slate-50/60"
              :class="!v.imei && 'bg-amber-50/40'"
            >
              <td>
                <div class="flex items-center gap-3">
                  <div
                    class="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg font-mono font-semibold"
                    :class="v.imei ? 'bg-blue-100 text-blue-700' : 'bg-amber-100 text-amber-700'"
                  >
                    {{ v.id.slice(0, 2) }}
                  </div>
                  <div>
                    <p class="font-mono font-semibold">{{ v.id }}</p>
                    <p class="mt-1 text-[10px] text-slate-400">{{ v.plate }}</p>
                  </div>
                </div>
              </td>
              <td>
                <p class="max-w-40 font-semibold">{{ v.model }}</p>
                <p class="mt-1 text-[10px] text-slate-400">
                  {{ v.fleet_code || 'Fleet not assigned' }}
                </p>
              </td>
              <td>
                <p class="font-mono font-semibold">{{ v.capacity }} L</p>
                <p class="mt-1 text-[10px] text-slate-400">Registered capacity</p>
              </td>
              <td>
                <template v-if="v.imei"
                  ><p class="font-mono text-[11px]">● {{ v.imei }}</p>
                  <p class="mt-1 font-mono text-[9px] text-slate-400">{{ v.device_model || 'Registered gateway' }}</p></template
                ><button v-else class="btn !border-amber-200 !text-amber-700" @click="openPairForAsset(v)">
                  <AppIcon name="link" class="!text-base" />Pair GPS
                </button>
              </td>
              <td>
                <template v-if="assetsDemoMode && v.imei"
                  ><StatusBadge tone="blue">BLE CFS300B</StatusBadge>
                  <p class="mt-1 font-mono text-[9px] text-slate-400">
                    BLE RSSI: <LiveValue :value="v.rssi" suffix=" dBm" /></p></template
                ><StatusBadge v-else-if="v.sensor_status" tone="blue">{{ v.sensor_status }}</StatusBadge>
                <span v-else class="text-[10px] italic text-slate-400">No sensor telemetry</span>
              </td>
              <td>
                <StatusBadge :tone="v.tone" dot>{{ v.state.toUpperCase() }}</StatusBadge>
              </td>
              <td>
                <button
                  v-if="v.imei"
                  class="btn !p-1.5"
                  :aria-label="`Inspect ${v.id}`"
                  @click="inspect(v)"
                >
                  <AppIcon name="visibility" class="!text-base" /></button
                ><button v-else class="btn !text-blue-700" @click="openPairForAsset(v)">Pair GPS</button>
                <button class="btn !p-1.5" :aria-label="`Edit ${v.id}`" @click="openEdit(v)">
                  <AppIcon name="edit" class="!text-base" />
                </button>
                <button class="btn btn-danger !p-1.5" :aria-label="`Delete ${v.id}`" @click="remove(v)">
                  <AppIcon name="delete" class="!text-base" />
                </button>
              </td>
            </tr>
            <tr v-if="!visible.length">
              <td colspan="7" class="py-14 text-center text-slate-400">
                No assets match your filters.
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
    <div v-else class="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
      <article v-for="v in visible" :key="v.id" class="panel p-4">
        <div class="flex justify-between">
          <span
            class="flex h-10 w-10 items-center justify-center rounded-lg bg-blue-50 text-blue-600"
            ><AppIcon name="local_shipping" /></span
          ><StatusBadge :tone="v.tone" dot>{{ v.state.toUpperCase() }}</StatusBadge>
        </div>
        <h2 class="mt-4 font-mono font-semibold">{{ v.id }}</h2>
        <p class="mt-1 text-xs text-slate-400">{{ v.plate }} · {{ v.model }}</p>
        <div class="my-4 flex justify-between border-y border-slate-100 py-3 text-xs">
          <span class="text-slate-400">Tank Capacity</span
          ><span class="font-mono">{{ v.capacity }} L</span>
        </div>
        <p class="mb-4 truncate font-mono text-[10px] text-slate-500">
          {{ v.imei || 'No gateway linked' }}
        </p>
        <div class="grid grid-cols-2 gap-2">
          <button class="btn !text-blue-700" @click="v.imei ? inspect(v) : openPairForAsset(v)">
            {{ v.imei ? 'Inspect Asset' : 'Pair GPS Gateway' }}
          </button>
          <button class="btn" @click="openEdit(v)"><AppIcon name="edit" />Edit Asset</button>
        </div>
        <button class="btn btn-danger mt-2 w-full" @click="remove(v)"><AppIcon name="delete" />Delete Asset</button>
      </article>
      <p v-if="!visible.length" class="col-span-full py-12 text-center text-slate-400">
        No assets match your filters.
      </p>
    </div>
    <div class="flex items-center justify-between gap-3 text-xs text-slate-500">
      <span
        >Showing {{ filtered.length ? (page - 1) * perPage + 1 : 0 }}–{{
          Math.min(page * perPage, filtered.length)
        }}
        of {{ filtered.length }} assets</span
      >
      <div class="flex items-center gap-2">
        <button class="btn" :disabled="page <= 1" @click="page--">Previous</button
        ><span class="mono">{{ page }} / {{ pages }}</span
        ><button class="btn" :disabled="page >= pages" @click="page++">Next</button>
      </div>
    </div>
    <AppModal
      v-model="showAdd"
      :dirty="assetDirty"
      :busy="busy"
      :title="editing ? 'Edit Fleet Asset' : 'Register New Fleet Asset'"
      wide
      ><form id="asset-form" class="space-y-5" @submit.prevent="save">
        <p class="text-xs leading-5 text-slate-400">
          {{ editing ? 'Update the registry fields saved to the live vehicle inventory.' : 'Register the vehicle and tank capacity. Pair an existing GPS gateway after creating the asset.' }}
        </p>
        <p v-if="error" role="alert" class="rounded-lg bg-red-50 p-3 text-xs text-red-700">
          {{ error }}
        </p>
        <h3 class="border-b border-slate-100 pb-2 text-xs font-semibold">
          01 · Asset Identity & Vehicle Specifications
        </h3>
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <label class="field"
            >Asset ID *<input
              v-model="form.id"
              required
              maxlength="40"
              pattern="[A-Za-z0-9_-]+"
              title="Letters, numbers, hyphens and underscores"
              placeholder="HAULER-05" /></label
          ><label class="field"
            >License Plate *<input
              v-model="form.plate"
              required
              maxlength="20"
              placeholder="B 9844 UVD" /></label
          ><label class="field"
            >Vehicle Name / Model *<input
              v-model="form.model"
              required
              maxlength="100"
              placeholder="Scania R500" /></label
          ><label class="field"
            >Fleet Code<input
              v-model="form.fleetCode"
              maxlength="64"
              placeholder="Optional fleet code"
          /></label>
        </div>
        <h3 class="border-b border-slate-100 pb-2 text-xs font-semibold">
          02 · Fuel Tank Configuration
        </h3>
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <label class="field"
            >Tank Capacity (L) *<input
              v-model.number="form.capacity"
              type="number"
              min="1"
              max="50000"
              step="0.1"
              required /></label>
        </div>
      </form>
      <template #footer="{ close }"
        ><button class="btn" :disabled="busy" @click="close">Cancel</button
        ><button form="asset-form" type="submit" class="btn btn-primary" :disabled="busy">
          {{ busy ? (editing ? 'Saving…' : 'Registering…') : (editing ? 'Save Changes' : 'Register Asset') }}
        </button></template
      ></AppModal
    ><AppModal
      v-model="showPair"
      :dirty="Boolean(imei)"
      :busy="busy"
      title="Pair GPS Gateway"
      ><form id="pair-form" class="space-y-5" @submit.prevent="pair">
        <div class="rounded-lg border border-slate-200 bg-slate-50 p-4">
          <p class="font-mono font-semibold">{{ selected?.id }}</p>
          <p class="mt-1 text-xs text-slate-400">{{ selected?.plate }} · {{ selected?.model }}</p>
        </div>
        <p v-if="error" role="alert" class="text-xs text-red-600">{{ error }}</p>
        <label class="field"
          >Gateway IMEI (last 4 digits) *<input
            v-model="imei"
            required
            inputmode="numeric"
            :pattern="assetsDemoMode ? '[0-9]{15}' : '[0-9]{4}'"
            :maxlength="assetsDemoMode ? 15 : 4"
            :placeholder="assetsDemoMode ? '15-digit IMEI' : 'Last 4 digits'"
        /></label>
        <p class="text-xs leading-5 text-slate-400">
          {{ assetsDemoMode ? 'Demo pairing resets when the page reloads.' : 'Only active, unassigned gateways registered in the system can be linked.' }}
        </p>
      </form>
      <template #footer="{ close }"
        ><button class="btn" @click="close">Cancel</button
        ><button form="pair-form" class="btn btn-primary" :disabled="busy">
          {{ busy ? 'Pairing…' : 'Pair Gateway' }}
        </button></template
      ></AppModal
    ><VehicleInspectModal v-model="showInspect" :vehicle="selected" :live="!assetsDemoMode" />
  </div>
</template>
