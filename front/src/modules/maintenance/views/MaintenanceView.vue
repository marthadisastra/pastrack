<script setup>
import { computed, reactive, ref } from 'vue'
import { useFleet } from '../../../composables/useFleet'
import { useToast } from '../../../composables/useToast'
import AppIcon from '../../../components/ui/AppIcon.vue'
import StatCard from '../../../components/ui/StatCard.vue'
import StatusBadge from '../../../components/ui/StatusBadge.vue'
import AppModal from '../../../components/ui/AppModal.vue'
import { exportCsv } from '../../../services/export'
import LiveValue from '../../../components/feedback/LiveValue.vue'
import { useUnsavedChanges } from '../../../composables/useUnsavedChanges'
const { vehicles, orders, addOrder, updateOrder, deleteOrder } = useFleet(),
  { notify } = useToast()
const tab = ref('active'),
  query = ref(''),
  scope = ref(''),
  status = ref(''),
  showSchedule = ref(false),
  busy = ref(false),
  error = ref(''),
  actionOrder = ref(null),
  showAction = ref(false),
  editing = ref(false)
const scopes = ['Engine', 'Fuel Sensor', 'Brakes', 'Electrical']
const facilities = ['Tj. Priok Central Depot', 'Cikarang Telematics Bay', 'Marunda Logistics Yard']
const today = new Date().toLocaleDateString('en-CA', { timeZone: 'Asia/Jakarta' })
const defaults = () => ({
  vehicleId: vehicles.value[0]?.id || '',
  scope: 'Engine',
  title: '',
  description: '',
  trigger: 50000,
  unit: 'km',
  facility: facilities[0],
  budget: 0,
  date: today,
  priority: 'Normal',
})
const form = reactive(defaults())
const scheduleDirty = computed(() => JSON.stringify(form) !== JSON.stringify(defaults()))
useUnsavedChanges(computed(() => showSchedule.value && scheduleDirty.value))
const active = computed(() => orders.value.filter((o) => o.status !== 'Completed'))
const history = computed(() => orders.value.filter((o) => o.status === 'Completed'))
const overdue = computed(() => active.value.filter((o) => o.status === 'Overdue').length)
const workshop = computed(() => active.value.filter((o) => o.status === 'In Workshop').length)
const filtered = computed(() =>
  (tab.value === 'active' ? active.value : history.value).filter(
    (o) =>
      (!scope.value || o.scope === scope.value) &&
      (!status.value || o.status === status.value) &&
      `${o.id} ${o.vehicleId} ${o.title} ${vehicle(o.vehicleId)?.plate || ''}`
        .toLowerCase()
        .includes(query.value.toLowerCase()),
  ),
)
const telemetry = computed(() => vehicle(form.vehicleId))
const fleetAverage = (field, decimals = 1) => {
  const assets = vehicles.value.filter((v) => v.imei && Number.isFinite(v[field]))
  return assets.length
    ? (assets.reduce((sum, v) => sum + v[field], 0) / assets.length).toFixed(decimals)
    : '—'
}
const compliance = computed(() =>
  active.value.length
    ? (((active.value.length - overdue.value) / active.value.length) * 100).toFixed(1)
    : '100.0',
)
function vehicle(id) {
  return vehicles.value.find((v) => v.id === id)
}
function tone(value) {
  return (
    {
      Overdue: 'red',
      'In Workshop': 'blue',
      'Due Soon': 'cyan',
      Scheduled: 'slate',
      Completed: 'green',
    }[value] || 'slate'
  )
}
function reset() {
  query.value = ''
  scope.value = ''
  status.value = ''
}
function openSchedule() {
  Object.assign(form, defaults())
  editing.value = false
  error.value = ''
  showSchedule.value = true
}
function openEdit(order) {
  Object.assign(form, { ...defaults(), ...order })
  editing.value = true
  error.value = ''
  showSchedule.value = true
}
async function save() {
  if (busy.value) return
  busy.value = true
  error.value = ''
  try {
    if (!form.vehicleId) throw new Error('Select a vehicle asset first.')
    if (!form.date) throw new Error('Select a schedule date.')
    if (!form.title.trim()) throw new Error('Enter a service description.')
    const payload = {
      vehicleId: form.vehicleId,
      scope: form.scope,
      title: form.title.trim(),
      description: form.description,
      trigger: Number(form.trigger),
      unit: form.unit,
      facility: form.facility,
      bay: form.bay || 'Pending allocation',
      budget: Number(form.budget),
      date: form.date,
      priority: form.priority,
      status: form.status,
    }
    if (editing.value) await updateOrder(form.id, payload)
    else await addOrder(payload)
    showSchedule.value = false
    tab.value = 'active'
    reset()
    notify(editing.value ? 'Maintenance schedule updated.' : 'Preventive maintenance scheduled.')
    editing.value = false
  } catch (cause) {
    error.value = cause.message
  } finally {
    busy.value = false
  }
}
async function removeOrder(order) {
  if (busy.value || !window.confirm(`Delete ${order.id}? This cannot be undone.`)) return
  busy.value = true
  error.value = ''
  try {
    await deleteOrder(order.id)
    notify(`Work order ${order.id} deleted.`)
  } catch (cause) { error.value = cause.message } finally { busy.value = false }
}
function requestAction(order) {
  actionOrder.value = order
  error.value = ''
  showAction.value = true
}
async function confirmAction() {
  if (busy.value) return
  busy.value = true
  error.value = ''
  try {
    const next = actionOrder.value.status === 'In Workshop' ? 'Completed' : 'In Workshop'
    await updateOrder(actionOrder.value.id, { status: next })
    showAction.value = false
    notify(`Work order ${actionOrder.value.id} updated to ${next}.`)
  } catch (cause) {
    error.value = cause.message
  } finally {
    busy.value = false
  }
}
function selectTab(item) {
  tab.value = item
  status.value = ''
}
</script>
<template>
  <div class="page">
    <header class="panel flex flex-wrap items-center justify-between gap-3 p-3">
      <div>
        <p class="mb-1 text-[10px] text-slate-400">Fleet Management › Vehicles</p>
        <h1 class="flex items-center gap-2 text-sm font-bold tracking-tight text-slate-950">
          <AppIcon name="build_circle" class="text-blue-700" />Vehicles: Maintenance & Work Orders
        </h1>
        <p class="mt-1 max-w-36 truncate text-[10px] text-slate-400">
          Queclink GNSS ODO & DI1 Run-Hours Sync
        </p>
      </div>
      <div class="flex flex-wrap gap-2">
        <button
          class="btn"
          :disabled="!filtered.length"
          @click="exportCsv('maintenance-audit.csv', filtered)"
        >
          <AppIcon name="download" />Export PM Audit</button
        ><button type="button" class="btn btn-primary" @click="openSchedule">
          <AppIcon name="add" />Schedule Maintenance
        </button>
      </div>
    </header>
    <div class="grid grid-cols-2 gap-3 xl:grid-cols-4">
      <StatCard label="Active Work Orders" :value="`${active.length} Orders`" icon="car_repair"
        >{{ workshop }} in workshop · Bays active</StatCard
      ><StatCard label="Maintenance Overdue" :value="`${overdue} Units`" icon="warning" danger
        ><span class="text-red-600">Dispatch hold · Service required</span></StatCard
      ><StatCard
        label="Upcoming Services"
        :value="`${active.filter((o) => ['Scheduled', 'Due Soon'].includes(o.status)).length} Units`"
        icon="event_upcoming"
        >Nearest trigger in 380 km</StatCard
      ><StatCard label="PM Schedule Compliance" :value="`${compliance}%`" icon="verified"
        ><span class="text-emerald-600">Queclink ODO & ACC Automated Sync</span></StatCard
      >
    </div>
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex rounded-lg border border-slate-200 bg-slate-100 p-1">
        <button
          v-for="item in ['active', 'history']"
          :key="item"
          class="rounded-md px-3 py-2 text-[11px] font-semibold"
          :class="tab === item ? 'bg-white text-blue-700 shadow-sm' : 'text-slate-500'"
          :aria-pressed="tab === item"
          @click="selectTab(item)"
        >
          {{
            item === 'active'
              ? `Active & Upcoming (${active.length})`
              : `Past Service History (${history.length})`
          }}
        </button>
      </div>
      <div class="flex flex-wrap gap-2">
        <label
          ><span class="sr-only">Search work orders</span
          ><input
            v-model="query"
            class="h-8 w-48 !text-xs"
            placeholder="Search Unit ID, Plate, WO #…" /></label
        ><label
          ><span class="sr-only">Service scope</span
          ><select v-model="scope" class="h-8 !text-xs">
            <option value="">All Service Scopes</option>
            <option v-for="item in scopes" :key="item">{{ item }}</option>
          </select></label
        ><label
          ><span class="sr-only">Status</span
          ><select v-model="status" class="h-8 !text-xs">
            <option value="">All Statuses</option>
            <option
              v-for="item in tab === 'history'
                ? ['Completed']
                : ['Overdue', 'In Workshop', 'Due Soon', 'Scheduled']"
              :key="item"
            >
              {{ item }}
            </option>
          </select></label
        ><button class="btn !p-2" aria-label="Reset filters" @click="reset">
          <AppIcon name="restart_alt" />
        </button>
      </div>
    </div>
    <section class="panel">
      <div class="overflow-x-auto">
        <table class="w-full min-w-[1100px] text-xs">
          <thead class="border-b border-slate-200 bg-slate-50">
            <tr>
              <th>Asset Identity</th>
              <th>Work Order & Scope</th>
              <th>Trigger Logic & ODO/ACC Accumulation</th>
              <th>Hardware Telemetry Health</th>
              <th>Depot Bay & Cost</th>
              <th>Status</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="order in filtered" :key="order.id" class="hover:bg-slate-50/70">
              <td>
                <StatusBadge :tone="tone(order.status)">{{ order.vehicleId }}</StatusBadge>
                <p class="mt-2 font-mono text-[10px]">{{ vehicle(order.vehicleId)?.plate }}</p>
                <p class="mt-1 max-w-32 truncate text-[10px] text-slate-400">
                  {{ vehicle(order.vehicleId)?.model }}
                </p>
              </td>
              <td class="max-w-52">
                <p class="mb-1 font-mono text-[10px] font-semibold text-blue-600">
                  #{{ order.id }}
                </p>
                <p class="line-clamp-2 font-semibold leading-4">{{ order.title }}</p>
                <p class="mt-1 line-clamp-1 text-[10px] leading-4 text-slate-400">
                  {{ order.description || order.scope }}
                </p>
              </td>
              <td class="min-w-52">
                <div class="flex justify-between gap-3 font-mono text-[10px]">
                  <span
                    >{{ Number(order.current || 0).toLocaleString('en-US') }} /
                    {{ Number(order.trigger || 0).toLocaleString('en-US') }} {{ order.unit }}</span
                  ><span :class="order.current > order.trigger ? 'text-rose-500' : 'text-teal-600'"
                    >{{ order.current > order.trigger ? '+' : ''
                    }}{{
                      order.current > order.trigger
                        ? order.current - order.trigger
                        : order.trigger - order.current
                    }}
                    {{ order.unit }}</span
                  >
                </div>
                <div class="my-2 h-1.5 overflow-hidden rounded bg-slate-100">
                  <div
                    class="h-full rounded"
                    :class="
                      order.status === 'Overdue'
                        ? 'bg-rose-500'
                        : order.status === 'In Workshop'
                          ? 'bg-blue-600'
                          : 'bg-teal-500'
                    "
                    :style="{ width: `${Math.min(100, (order.current / order.trigger) * 100)}%` }"
                  ></div>
                </div>
                <p class="text-[9px] text-slate-400">
                  {{
                    order.unit === 'km'
                      ? 'Queclink GV30CEU GNSS Accumulation'
                      : 'DI1 Ignition Run-Hours Trigger'
                  }}
                </p>
              </td>
              <td class="font-mono text-[10px] leading-6">
                <p>
                  <span class="text-slate-400">GV30CEU:</span>
                  <LiveValue :value="vehicle(order.vehicleId)?.battery" suffix="V" /> Main
                </p>
                <p>
                  <span class="text-slate-400">CFS300B:</span>
                  {{ vehicle(order.vehicleId)?.probeBattery }}V
                  <span class="text-teal-600"
                    ><LiveValue :value="vehicle(order.vehicleId)?.rssi" suffix=" dBm"
                  /></span>
                </p>
              </td>
              <td>
                <p class="max-w-32 text-[11px] font-semibold">{{ order.facility }}</p>
                <p class="mt-1 max-w-36 truncate text-[10px] text-slate-400">
                  {{ order.bay }} · Est. ${{ order.budget.toFixed(2) }}
                </p>
                <p class="mt-1 font-mono text-[9px] text-slate-400">{{ order.date }}</p>
              </td>
              <td>
                <StatusBadge :tone="tone(order.status)" dot>{{
                  order.status.toUpperCase()
                }}</StatusBadge>
              </td>
              <td>
                <button
                  v-if="order.status !== 'Completed'"
                  class="btn !text-[10px]"
                  :class="
                    order.status === 'Overdue'
                      ? 'btn-danger'
                      : order.status === 'In Workshop'
                        ? 'btn-primary'
                        : ''
                  "
                  @click="requestAction(order)"
                >
                  {{
                    order.status === 'In Workshop'
                      ? 'Complete WO'
                      : order.status === 'Overdue'
                        ? 'Dispatch Bay'
                        : 'Start Service'
                  }}</button
                ><button class="btn ml-1 !px-2 !py-1 !text-[10px]" @click="openEdit(order)">Edit</button>
                <button class="btn ml-1 !px-2 !py-1 !text-[10px] text-red-600" @click="removeOrder(order)">Delete</button>
                <span v-if="order.status === 'Completed'" class="text-[10px] text-emerald-600">Completed</span>
              </td>
            </tr>
            <tr v-if="!filtered.length">
              <td colspan="7" class="py-14 text-center text-slate-400">
                No work orders match your filters.
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
    <div class="grid gap-4 lg:grid-cols-3">
      <section class="panel p-4">
        <h2 class="mb-4 flex items-center gap-2 text-xs font-bold">
          <AppIcon name="router" class="text-blue-600" />Queclink Gateway Roster
        </h2>
        <dl class="space-y-3 text-[11px]">
          <div
            v-for="(value, label) in {
              'Main Power Ingestion': `${fleetAverage('battery', 2)}V Avg DC`,
              'Internal Backup LiPo': `${fleetAverage('backupBattery', 2)}V Avg`,
              'Cellular Signal Strength': `LTE CSQ ${fleetAverage('csq')}/31`,
              'GNSS Lock Precision': `${fleetAverage('satellites', 0)} Sats · HDOP ${fleetAverage('hdop', 2)}`,
            }"
            :key="label"
            class="flex justify-between gap-3"
          >
            <dt class="text-slate-400">{{ label }}</dt>
            <dd class="text-right font-mono text-[10px]"><LiveValue :value="value" /></dd>
          </div>
        </dl>
      </section>
      <section class="panel p-4">
        <h2 class="mb-4 flex items-center gap-2 text-xs font-bold">
          <AppIcon name="sensors" class="text-teal-600" />CFS300B BLE 5.0 Probes
        </h2>
        <dl class="space-y-3 text-[11px]">
          <div
            v-for="(value, label) in {
              'Internal Li-SOCl2 Cell': '> 3.61V (All Safe)',
              'Zero-Point Frequency': '±0.4 Hz Tolerance',
              'BLE 5.0 Beaconing RSSI': `${fleetAverage('rssi')} dBm Avg In-Tank`,
              'Capacitive Resolution': '0.1 mm Level Accuracy',
            }"
            :key="label"
            class="flex justify-between gap-3"
          >
            <dt class="text-slate-400">{{ label }}</dt>
            <dd class="text-right font-mono text-[10px]"><LiveValue :value="value" /></dd>
          </div>
        </dl>
      </section>
      <section class="panel p-4">
        <h2 class="mb-3 flex items-center gap-2 text-xs font-bold">
          <AppIcon name="garage" class="text-blue-600" />Depot Bays Occupancy
        </h2>
        <div
          v-for="order in active
            .filter((o) => ['In Workshop', 'Overdue', 'Due Soon'].includes(o.status))
            .slice(0, 3)"
          :key="order.id"
          class="mb-2 flex items-center justify-between gap-2 rounded-lg border border-slate-200 bg-slate-50 p-2"
        >
          <div>
            <p class="text-[11px] font-semibold">{{ order.bay }}</p>
            <p class="text-[9px] text-slate-400">{{ order.facility }}</p>
          </div>
          <StatusBadge :tone="tone(order.status)">{{ order.vehicleId }}</StatusBadge>
        </div>
      </section>
    </div>
    <AppModal
      v-model="showSchedule"
      :dirty="scheduleDirty"
      :busy="busy"
      :title="editing ? 'Edit Preventive Maintenance' : 'Schedule Preventive Maintenance'"
      wide
      ><form id="schedule-form" class="space-y-5" @submit.prevent="save">
        <p v-if="error" role="alert" class="rounded-lg bg-red-50 p-3 text-xs text-red-600">
          {{ error }}
        </p>
        <div class="grid gap-4 sm:grid-cols-2">
          <label class="field"
            >Vehicle Asset *<select v-model="form.vehicleId" required>
              <option v-for="v in vehicles" :key="v.id" :value="v.id">
                {{ v.id }} · {{ v.plate }}
              </option>
            </select></label
          ><label class="field"
            >Service Scope<select v-model="form.scope">
              <option v-for="item in scopes" :key="item">{{ item }}</option>
            </select></label
          >
        </div>
        <div
          v-if="telemetry"
          class="rounded-lg border border-blue-100 bg-blue-50 p-3 font-mono text-[10px] leading-6 text-blue-700"
        >
          {{ telemetry.model }} · ODO {{ Number(telemetry.odo || 0).toLocaleString('en-US') }} km ·
          {{ telemetry.battery ?? '—' }}V Main · CFS300B {{ telemetry.probeBattery ?? '—' }}V
        </div>
        <label class="field"
          >Work Order Title *<input
            v-model="form.title"
            required
            maxlength="120"
            placeholder="50k Major Lubricants & Filters" /></label
        ><label class="field"
          >Service Details<textarea
            v-model="form.description"
            rows="2"
            maxlength="500"
            placeholder="Parts, inspection notes and service instructions"
          ></textarea>
        </label>
        <div class="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          <label class="field"
            >Trigger Threshold *<input
              v-model.number="form.trigger"
              type="number"
              min="1"
              required /></label
          ><label class="field"
            >Trigger Unit<select v-model="form.unit">
              <option value="km">GNSS Odometer (km)</option>
              <option value="hrs">DI1 Run Hours (hrs)</option>
            </select></label
          ><label class="field"
            >Depot / Facility<select v-model="form.facility">
              <option v-for="item in facilities" :key="item">{{ item }}</option>
            </select></label
          ><label class="field"
            >Schedule Date *<input v-model="form.date" type="date" :min="today" required /></label
          ><label class="field"
            >Priority<select v-model="form.priority">
              <option>Normal</option>
              <option>High</option>
              <option>Critical</option>
            </select></label
          ><label class="field"
            >Estimated Budget (USD)<input
              v-model.number="form.budget"
              type="number"
              min="0"
              step="0.01"
              required
          /></label>
        </div>
      </form>
      <template #footer="{ close }"
        ><button class="btn" :disabled="busy" @click="close">Cancel</button
        ><button form="schedule-form" class="btn btn-primary" :disabled="busy">
          {{ busy ? 'Saving…' : editing ? 'Save Changes' : 'Create Maintenance Schedule' }}
        </button></template
      ></AppModal
    ><AppModal
      v-model="showAction"
      :busy="busy"
      :title="
        actionOrder?.status === 'In Workshop' ? 'Complete Work Order' : 'Start Workshop Service'
      "
      ><p class="text-sm leading-6 text-slate-500">
        {{
          actionOrder?.status === 'In Workshop'
            ? 'Mark this service as completed and move it to past service history.'
            : 'Move this work order into the workshop.'
        }}
      </p>
      <div class="mt-4 rounded-lg border border-slate-200 bg-slate-50 p-4">
        <p class="font-mono text-xs font-semibold">
          {{ actionOrder?.id }} · {{ actionOrder?.vehicleId }}
        </p>
        <p class="mt-1 text-xs text-slate-500">{{ actionOrder?.title }}</p>
      </div>
      <p v-if="error" role="alert" class="mt-4 text-xs text-red-600">{{ error }}</p>
      <template #footer="{ close }"
        ><button class="btn" @click="close">Cancel</button
        ><button class="btn btn-primary" :disabled="busy" @click="confirmAction">
          {{ busy ? 'Updating…' : 'Confirm Update' }}
        </button></template
      ></AppModal
    >
  </div>
</template>
