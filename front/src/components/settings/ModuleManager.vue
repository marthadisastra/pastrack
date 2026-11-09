<script setup>
import { computed, ref } from 'vue'
import { moduleCatalog, coreVersion, checkManifest } from '../../modules/registry'
import { installModule, uninstallModule, setModuleEnabled } from '../../modules/lifecycle'
import { useConfirm } from '../../composables/useConfirm'
import { useToast } from '../../composables/useToast'
import AppIcon from '../ui/AppIcon.vue'
import StatusBadge from '../ui/StatusBadge.vue'
import AppModal from '../ui/AppModal.vue'
const draft = defineModel({ required: true })
const basePath = import.meta.env.BASE_URL
const { ask } = useConfirm(),
  { notify } = useToast()
const query = ref(''),
  filter = ref('all'),
  upload = ref(false),
  chosen = ref(null),
  fileError = ref(''),
  inspecting = ref(false),
  selected = ref(null),
  configOpen = ref(false),
  drag = ref(false)
const records = computed(() =>
  moduleCatalog.filter(
    (module) =>
      `${module.name} ${module.id} ${module.description}`
        .toLowerCase()
        .includes(query.value.toLowerCase()) &&
      (filter.value === 'all' ||
        filter.value === module.kind ||
        (filter.value === 'available' && !draft.value.modules[module.id].installed)),
  ),
)
const installed = computed(
  () => moduleCatalog.filter((module) => draft.value.modules[module.id].installed).length,
)
function toggle(module) {
  try {
    setModuleEnabled(draft.value, module.id, !draft.value.modules[module.id].enabled)
  } catch (error) {
    notify(error.message, 'error')
  }
}
async function remove(module) {
  if (
    !(await ask({
      title: 'Uninstall bundled module?',
      message: `${module.name} will become unavailable after Save Changes. Existing vehicle and maintenance records are retained.`,
      confirmLabel: 'Stage uninstall',
      cancelLabel: 'Cancel',
    }))
  )
    return
  try {
    uninstallModule(draft.value, module.id)
  } catch (error) {
    notify(error.message, 'error')
  }
}
function install(manifest) {
  try {
    const module = installModule(draft.value, manifest)
    upload.value = false
    notify(`${module.name} is staged. Save Changes to activate its menu and page.`, 'info')
  } catch (error) {
    fileError.value = error.message
    notify(error.message, 'error')
  }
}
async function readFile(file) {
  chosen.value = null
  fileError.value = ''
  if (!file) return
  if (!file.name.toLowerCase().endsWith('.json')) {
    fileError.value =
      'Binary .ptm/.zip installation requires the Go package installer. This frontend preview validates catalog manifests (.json) only.'
    return
  }
  if (file.size > 128 * 1024) {
    fileError.value = 'Manifest is too large. Maximum size is 128 KB.'
    return
  }
  inspecting.value = true
  try {
    chosen.value = checkManifest(JSON.parse(await file.text()))
  } catch (error) {
    fileError.value = error instanceof SyntaxError ? 'The file is not valid JSON.' : error.message
  } finally {
    inspecting.value = false
  }
}
function drop(event) {
  drag.value = false
  readFile(event.dataTransfer.files[0])
}
function configure(module) {
  selected.value = module
  configOpen.value = true
}
function openUpload() {
  upload.value = true
  chosen.value = null
  fileError.value = ''
}
</script>
<template>
  <div class="space-y-3">
    <section class="panel flex flex-wrap items-center gap-2 p-3">
      <label class="relative min-w-48 flex-1"
        ><span class="sr-only">Search modules</span
        ><AppIcon name="search" class="absolute left-2 top-2 !text-base text-slate-400" /><input
          v-model="query"
          placeholder="Search installed packages, protocol decoders…"
          class="h-8 w-full pl-8" /></label
      ><label
        ><span class="sr-only">Module classification</span
        ><select v-model="filter" class="h-8">
          <option value="all">All Modules ({{ moduleCatalog.length }})</option>
          <option value="core">Native Core</option>
          <option value="extension">Extensions</option>
          <option value="available">Available to Install</option>
        </select></label
      ><button class="btn btn-primary ml-auto" @click="openUpload">
        <AppIcon name="cloud_upload" />Upload Module
      </button>
    </section>
    <div class="grid grid-cols-3 gap-2 sm:gap-3">
      <article class="panel flex items-center justify-between p-3">
        <div>
          <p class="text-[9px] font-semibold uppercase tracking-wider text-slate-500">
            Bundled Package Storage
          </p>
          <p class="mt-1 text-lg font-bold">
            {{ installed }}
            <span class="text-[10px] font-normal text-slate-500"
              >/ {{ moduleCatalog.length }} installed</span
            >
          </p>
          <p class="mt-1 text-[9px] text-slate-400">Disk usage requires runtime metrics API</p>
        </div>
        <AppIcon name="inventory_2" class="text-blue-600" />
      </article>
      <article class="panel flex items-center justify-between p-3">
        <div>
          <p class="text-[9px] font-semibold uppercase tracking-wider text-slate-500">
            BLE Telemetry Workers
          </p>
          <p class="mt-1 text-lg font-bold">
            — <span class="text-[10px] font-normal text-slate-500">Runtime status</span>
          </p>
          <p class="mt-1 text-[9px] text-slate-400">CFS300B worker metrics not connected</p>
        </div>
        <AppIcon name="bluetooth" class="text-cyan-600" />
      </article>
      <article class="panel flex items-center justify-between p-3">
        <div>
          <p class="text-[9px] font-semibold uppercase tracking-wider text-slate-500">
            Event Dispatchers
          </p>
          <p class="mt-1 text-lg font-bold">UI Ready</p>
          <p class="mt-1 text-[9px] text-slate-400">Backend worker health not connected</p>
        </div>
        <AppIcon name="bolt" class="text-emerald-600" />
      </article>
    </div>
    <section class="panel">
      <div class="panel-title">
        <h2 class="text-xs font-semibold">Runtime Modules & Subsystems</h2>
        <span class="font-mono text-[9px] text-slate-400"
          >Target runtime: PASTRACK Core {{ coreVersion }}</span
        >
      </div>
      <div class="overflow-x-auto">
        <table class="w-full min-w-[850px] text-xs">
          <thead class="border-b border-slate-200 bg-slate-50">
            <tr>
              <th>Module / Package Name</th>
              <th>Classification</th>
              <th>Hardware Target / Function</th>
              <th>Engine State</th>
              <th class="text-right">Actions</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="module in records" :key="module.id" class="hover:bg-slate-50/70">
              <td class="w-[29%]">
                <div class="flex items-center gap-2">
                  <span
                    class="flex h-7 w-7 shrink-0 items-center justify-center rounded-md border border-blue-100 bg-blue-50 text-blue-600"
                    ><AppIcon :name="module.icon" class="!text-base"
                  /></span>
                  <div>
                    <p class="text-[11px] font-semibold">{{ module.name }}</p>
                    <p class="mt-1 font-mono text-[9px] text-slate-400">
                      {{ module.id }} · v{{ module.version }}
                    </p>
                  </div>
                </div>
              </td>
              <td>
                <StatusBadge :tone="module.kind === 'core' ? 'blue' : 'slate'">{{
                  module.kind === 'core' ? 'Native Core' : 'Extension'
                }}</StatusBadge>
              </td>
              <td class="max-w-64 text-[10px] leading-4 text-slate-500">
                {{ module.description }}
              </td>
              <td>
                <StatusBadge v-if="module.kind === 'core'" tone="cyan" dot
                  >Active (Locked)</StatusBadge
                ><span
                  v-else-if="!draft.modules[module.id].installed"
                  class="text-[10px] text-slate-400"
                  >Not Installed</span
                ><button
                  v-else
                  type="button"
                  role="switch"
                  :aria-label="`Enable ${module.name}`"
                  :aria-checked="draft.modules[module.id].enabled"
                  class="relative inline-flex h-[18px] w-8 items-center rounded-full border transition-colors"
                  :class="
                    draft.modules[module.id].enabled
                      ? 'border-blue-600 bg-blue-600'
                      : 'border-slate-300 bg-slate-200'
                  "
                  @click="toggle(module)"
                >
                  <span
                    class="h-3.5 w-3.5 rounded-full bg-white shadow-sm transition-transform"
                    :class="
                      draft.modules[module.id].enabled ? 'translate-x-[15px]' : 'translate-x-px'
                    "
                  ></span>
                </button>
              </td>
              <td class="text-right">
                <div class="flex justify-end gap-1">
                  <button
                    v-if="!draft.modules[module.id].installed"
                    class="btn !text-blue-700"
                    :aria-label="`Install ${module.name}`"
                    @click="install(module)"
                  >
                    Install</button
                  ><template v-else
                    ><button
                      v-if="module.config.length"
                      class="btn !py-1 !text-[10px]"
                      :aria-label="`Configure ${module.name}`"
                      @click="configure(module)"
                    >
                      {{ module.kind === 'core' ? 'Telemetry Config' : 'Config' }}</button
                    ><button
                      v-if="module.kind === 'extension'"
                      class="btn !border-rose-200 !py-1 !text-[10px] !text-rose-600"
                      :aria-label="`Uninstall ${module.name}`"
                      @click="remove(module)"
                    >
                      Uninstall</button
                    ><span
                      v-else-if="!module.config.length"
                      class="text-[10px] italic text-slate-400"
                      >Protected Core</span
                    ></template
                  >
                </div>
              </td>
            </tr>
            <tr v-if="!records.length">
              <td colspan="5" class="py-10 text-center text-slate-400">
                No modules match this search.
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
    <p class="text-[10px] text-slate-400">
      Module changes are staged until Save Changes. Catalog modules are bundled with this frontend;
      backend binaries are not installed by the browser.
    </p>
    <AppModal v-model="upload" title="Upload Runtime Module Package"
      ><div class="space-y-4">
        <label
          class="flex cursor-pointer flex-col items-center gap-2 rounded-lg border-2 border-dashed p-6 text-center"
          :class="drag ? 'border-blue-500 bg-blue-50' : 'border-slate-300 bg-slate-50'"
          @dragover.prevent="drag = true"
          @dragleave.prevent="drag = false"
          @drop.prevent="drop"
          ><AppIcon name="upload_file" class="!text-3xl text-blue-600" /><span
            class="text-xs font-medium"
            >Drag a .json module manifest here or choose a file</span
          ><span class="text-[10px] text-slate-400"
            >Allowed extension: .json · Catalog manifest only</span
          ><input
            type="file"
            accept=".json"
            class="max-w-full text-xs"
            aria-label="Module package file"
            @change="readFile($event.target.files[0])"
        /></label>
        <p v-if="inspecting" role="status" class="text-xs text-blue-600">Inspecting manifest…</p>
        <p
          v-if="fileError"
          role="alert"
          class="rounded-md bg-rose-50 p-3 text-xs leading-5 text-rose-700"
        >
          {{ fileError }}
        </p>
        <div
          v-if="chosen"
          class="rounded-md border border-blue-200 bg-blue-50 p-3 text-xs leading-6 text-blue-800"
        >
          <strong>{{ chosen.name }} · v{{ chosen.version }}</strong>
          <p>Catalog manifest recognized. No binary signature has been verified.</p>
          <p>Dependencies: {{ chosen.dependencies.join(', ') || 'None' }}</p>
        </div>
        <p class="text-[10px] leading-5 text-slate-500">
          Production package handling needs signature verification, compatibility checks,
          migrations, health checks and rollback on the Go server.
        </p>
        <a
          :href="`${basePath}examples/maintenance.module.json`"
          download
          class="text-[10px] text-blue-700 underline"
          >Download example maintenance manifest</a
        >
      </div>
      <template #footer="{ close }"
        ><button class="btn" @click="close">Cancel</button
        ><button class="btn btn-primary" :disabled="!chosen || inspecting" @click="install(chosen)">
          Stage Catalog Install
        </button></template
      ></AppModal
    >
    <AppModal v-model="configOpen" :title="`Module Configuration · ${selected?.name || ''}`"
      ><div v-if="selected" class="space-y-4">
        <p class="text-[10px] text-slate-400">
          Configuration is staged with the rest of Settings. Hardware thresholds require backend
          enforcement.
        </p>
        <label v-for="field in selected.config" :key="field.key" class="field"
          >{{ field.label
          }}<input
            v-model.number="draft.modules[selected.id].config[field.key]"
            type="number"
            :min="field.min"
            :max="field.max"
            :step="field.step || 1"
        /></label>
        <div class="rounded bg-slate-50 p-3 text-[10px] leading-5 text-slate-500">
          <p>Dependencies: {{ selected.dependencies.join(', ') || 'None' }}</p>
          <p>Routes: {{ selected.paths.join(', ') }}</p>
          <p>Permissions: {{ selected.permissions.join(', ') }}</p>
          <p>Event topics: {{ selected.topics.join(', ') }}</p>
        </div>
      </div>
      <template #footer="{ close }"
        ><button class="btn btn-primary" @click="close">Done</button></template
      ></AppModal
    >
  </div>
</template>
