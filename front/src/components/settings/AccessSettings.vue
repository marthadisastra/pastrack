<script setup>
import { reactive, ref } from 'vue'
import AppIcon from '../ui/AppIcon.vue'
import StatusBadge from '../ui/StatusBadge.vue'
import AppModal from '../ui/AppModal.vue'
import { permissionColumns } from '../../modules/state'
import { downloadJson } from '../../modules/branding'
const draft = defineModel({ required: true })
const invite = ref(false),
  inviteError = ref(''),
  form = reactive({ name: '', email: '', role: 'Dispatcher' })
function addOperator() {
  inviteError.value = ''
  if (!form.name.trim() || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(form.email)) {
    inviteError.value = 'Provide an operator name and valid email address.'
    return
  }
  if (draft.value.operators.some((user) => user.email.toLowerCase() === form.email.toLowerCase())) {
    inviteError.value = 'This email address is already in the operator list.'
    return
  }
  draft.value.operators.push({
    id: crypto.randomUUID(),
    name: form.name.trim(),
    email: form.email.trim(),
    role: form.role,
    twoFactor: false,
    pending: true,
  })
  invite.value = false
  form.name = ''
  form.email = ''
}
function duplicate() {
  const base = draft.value.roles.find((role) => role.name === 'Dispatcher'),
    name = `Dispatcher Copy ${draft.value.roles.filter((role) => role.name.startsWith('Dispatcher Copy')).length + 1}`
  draft.value.roles.push({ name, permissions: [...base.permissions] })
}
function openInvite() {
  invite.value = true
  inviteError.value = ''
}
</script>
<template>
  <div class="space-y-3">
    <div
      class="rounded-md border border-blue-100 bg-blue-50 px-3 py-2 text-[10px] leading-5 text-blue-700"
    >
      This matrix is a policy editor preview. Authentication, invitations, 2FA verification and
      permission enforcement must be implemented on the Go backend.
    </div>
    <section class="panel">
      <div class="panel-title">
        <div>
          <h2 class="text-xs font-semibold">Authorized Telematics Operators</h2>
          <p class="mt-1 text-[9px] text-slate-400">
            Review identities, role assignments and 2FA policy.
          </p>
        </div>
        <button class="btn btn-primary" @click="openInvite">
          <AppIcon name="person_add" />Invite Operator
        </button>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full min-w-[680px]">
          <thead class="border-b border-slate-200 bg-slate-50">
            <tr>
              <th>System Identity</th>
              <th>Assigned Role</th>
              <th>2FA Security</th>
              <th>State</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="user in draft.operators" :key="user.id">
              <td>
                <p class="text-[11px] font-semibold">{{ user.name }}</p>
                <p class="mt-1 text-[10px] text-slate-400">{{ user.email }}</p>
              </td>
              <td>
                <select
                  v-model="user.role"
                  :aria-label="`Role for ${user.name}`"
                  :disabled="user.id === 'admin'"
                  class="h-7 !py-1 !text-[10px]"
                >
                  <option v-for="role in draft.roles" :key="role.name">{{ role.name }}</option>
                </select>
              </td>
              <td>
                <label class="flex items-center gap-2 text-[10px]"
                  ><input
                    v-model="user.twoFactor"
                    type="checkbox"
                    :disabled="user.id === 'admin'"
                    class="rounded text-blue-700"
                    :aria-label="`Require 2FA for ${user.name}`"
                  />{{ user.twoFactor ? 'Required policy' : 'Optional policy' }}</label
                >
              </td>
              <td>
                <StatusBadge :tone="user.pending ? 'amber' : 'slate'">{{
                  user.pending ? 'Invite Draft' : 'Demo Identity'
                }}</StatusBadge>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <p class="border-t border-slate-100 p-3 text-[9px] text-slate-400">
        Invite drafts do not send email. No active session or security credential is changed by this
        preview.
      </p>
    </section>
    <section class="panel">
      <div class="panel-title">
        <div>
          <h3 class="text-xs font-semibold">Subsystem & Telemetry Command Permission Matrix</h3>
          <p class="mt-1 text-[9px] text-slate-400">
            Control policy for tracking, asset configuration, calibration and relay commands.
          </p>
        </div>
        <div class="flex gap-2">
          <button class="btn" @click="duplicate">
            <AppIcon name="content_copy" />Duplicate Role</button
          ><button
            class="btn"
            @click="downloadJson('pastrack-rbac-policy.json', { roles: draft.roles })"
          >
            <AppIcon name="download" />Export JSON
          </button>
        </div>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full min-w-[880px]">
          <thead class="border-b border-slate-200 bg-slate-50">
            <tr>
              <th>Role Classification</th>
              <th v-for="column in permissionColumns" :key="column" class="max-w-20 text-center">
                {{ column }}
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="role in draft.roles" :key="role.name">
              <td class="text-[11px] font-medium">{{ role.name }}</td>
              <td v-for="(column, index) in permissionColumns" :key="column" class="text-center">
                <input
                  v-model="role.permissions[index]"
                  type="checkbox"
                  :disabled="role.name === 'Super Admin'"
                  :aria-label="`${role.name}: ${column}`"
                  class="rounded border-slate-300 text-blue-700"
                />
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
    <AppModal v-model="invite" title="Invite Operator · Draft"
      ><form id="operator-form" class="space-y-3" @submit.prevent="addOperator">
        <p class="text-[10px] leading-5 text-slate-400">
          Add an operator invitation draft. Sending an invitation requires the backend mail and
          identity service.
        </p>
        <p v-if="inviteError" role="alert" class="text-xs text-rose-600">{{ inviteError }}</p>
        <label class="field"
          >Operator Name<input v-model="form.name" required maxlength="100" /></label
        ><label class="field"
          >Work Email<input v-model="form.email" required type="email" maxlength="160" /></label
        ><label class="field"
          >Assigned Role<select v-model="form.role">
            <option
              v-for="role in draft.roles.filter((role) => role.name !== 'Super Admin')"
              :key="role.name"
            >
              {{ role.name }}
            </option>
          </select></label
        >
      </form>
      <template #footer="{ close }"
        ><button class="btn" @click="close">Cancel</button
        ><button class="btn btn-primary" form="operator-form">
          Add Invitation Draft
        </button></template
      ></AppModal
    >
  </div>
</template>
