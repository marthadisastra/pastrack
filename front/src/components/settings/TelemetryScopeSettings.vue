<script setup>
import { computed, onMounted, ref } from 'vue'
import { scopeService } from '../../services/scope'
import { useToast } from '../../composables/useToast'
const { notify } = useToast(),
  loading = ref(true),
  saving = ref(false),
  error = ref('')
const data = ref({ users: [], vehicles: [], scopes: [] }),
  selectedUser = ref(''),
  selected = ref(new Set())
const current = computed(() =>
  data.value.users.find((u) => String(u.id) === String(selectedUser.value)),
)
function loadSelection() {
  selected.value = new Set(
    data.value.scopes
      .filter((s) => String(s.user_id) === String(selectedUser.value))
      .map((s) => String(s.vehicle_id)),
  )
}
function toggle(id) {
  const next = new Set(selected.value)
  next.has(String(id)) ? next.delete(String(id)) : next.add(String(id))
  selected.value = next
}
async function save() {
  saving.value = true
  try {
    await scopeService.save(Number(selectedUser.value), [...selected.value].map(Number))
    notify('Telemetry scope saved.')
    data.value = await scopeService.load()
    loadSelection()
  } catch (e) {
    error.value = e.message
    notify(e.message, 'error')
  } finally {
    saving.value = false
  }
}
onMounted(async () => {
  try {
    data.value = await scopeService.load()
    selectedUser.value = data.value.users[0]?.id || ''
    loadSelection()
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
})
</script>
<template>
  <section class="panel p-3">
    <h2 class="text-sm font-semibold">Telemetry Scope</h2>
    <p class="mt-1 text-[10px] text-slate-500">Batasi kendaraan yang dapat dilihat setiap user.</p>
    <p v-if="error" class="mt-2 text-xs text-rose-600">{{ error }}</p>
    <div v-if="loading" class="skeleton mt-3 h-24" role="status" />
    <div v-else class="mt-3 grid gap-3 md:grid-cols-[220px_1fr]">
      <label class="text-xs"
        >User<select v-model="selectedUser" class="mt-1 w-full" @change="loadSelection">
          <option v-for="user in data.users" :key="user.id" :value="user.id">
            {{ user.username }} · {{ user.full_name }}
          </option>
        </select></label
      >
      <div>
        <p class="mb-2 text-xs text-slate-500">
          {{ current ? `Vehicles visible to ${current.username}` : 'Select a user' }}
        </p>
        <div class="grid gap-1 sm:grid-cols-2">
          <label
            v-for="vehicle in data.vehicles"
            :key="vehicle.id"
            class="flex items-center gap-2 rounded border border-slate-200 p-2 text-xs"
            ><input
              type="checkbox"
              :checked="selected.has(String(vehicle.id))"
              @change="toggle(vehicle.id)"
            />
            {{ vehicle.plate_number }} · {{ vehicle.vehicle_name || 'Vehicle' }}</label
          >
        </div>
        <button class="btn btn-primary mt-3" :disabled="!selectedUser || saving" @click="save">
          {{ saving ? 'Saving…' : 'Save scope' }}
        </button>
      </div>
    </div>
  </section>
</template>
