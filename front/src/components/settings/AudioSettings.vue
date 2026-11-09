<script setup>
import { ref } from 'vue'
import AppIcon from '../ui/AppIcon.vue'
import SettingToggle from './SettingToggle.vue'
import { playTone, unlockAudio } from '../../modules/audio'
const draft = defineModel({ required: true })
const playing = ref(''),
  error = ref('')
async function enableAudio(value) {
  if (!value) return
  try {
    await unlockAudio()
  } catch (cause) {
    error.value = cause.message
  }
}
async function test(event) {
  error.value = ''
  playing.value = event.id
  try {
    await playTone(event.tone, draft.value.audio.volume, event.gain)
  } catch (cause) {
    error.value = cause.message
  } finally {
    setTimeout(() => {
      playing.value = ''
    }, 600)
  }
}
</script>
<template>
  <div class="space-y-3">
    <section class="panel p-3">
      <div class="flex items-center justify-between gap-3">
        <div>
          <h2 class="text-sm font-semibold">Enable Hardware Telemetry Audio Alerts</h2>
          <p class="mt-1 text-[10px] text-slate-400">
            Browser audio feedback for new telemetry events. Test a tone once after opening this
            browser session.
          </p>
        </div>
        <SettingToggle
          v-model="draft.audio.enabled"
          label="Enable audio alerts"
          @update:model-value="enableAudio"
        />
      </div>
      <div class="mt-4 flex flex-wrap items-center gap-3 border-t border-slate-100 pt-3">
        <AppIcon name="volume_up" class="text-slate-400" /><label
          class="flex min-w-40 flex-1 items-center gap-3"
          ><span class="sr-only">Master volume</span
          ><input
            v-model.number="draft.audio.volume"
            type="range"
            min="0"
            max="100"
            class="w-full accent-blue-600"
          /><span class="w-9 font-mono text-xs">{{ draft.audio.volume }}%</span></label
        ><select v-model="draft.audio.output" aria-label="Audio output" class="h-8">
          <option value="browser">Dispatch Console Speaker (Browser)</option>
        </select>
      </div>
      <p class="mt-2 text-[9px] text-slate-400">
        External siren / GPIO and hardware DAC routing require a backend/device integration.
      </p>
    </section>
    <p v-if="error" role="alert" class="rounded-md bg-rose-50 p-3 text-xs text-rose-700">
      {{ error }}
    </p>
    <section class="panel">
      <div class="panel-title">
        <h3 class="text-xs font-semibold">Hardware Incident Audio Configuration Matrix</h3>
        <span class="text-[9px] text-slate-400">Queclink GV30CEU & CFS300B Event Triggers</span>
      </div>
      <div class="overflow-x-auto">
        <table class="w-full min-w-[760px]">
          <thead class="border-b border-slate-200 bg-slate-50">
            <tr>
              <th>Telemetry Event / Incident</th>
              <th>Hardware Sensor Trigger Condition</th>
              <th>Synthesized Tone</th>
              <th>Gain</th>
              <th>Test Tone</th>
              <th>State</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-slate-100">
            <tr v-for="event in draft.audio.events" :key="event.id">
              <td class="text-[11px] font-medium">
                {{ event.title }}
                <span
                  class="text-[9px]"
                  :class="event.severity === 'P1' ? 'text-rose-600' : 'text-slate-400'"
                  >{{ event.severity }}</span
                >
              </td>
              <td class="max-w-52 text-[10px] text-slate-500">{{ event.condition }}</td>
              <td>
                <select
                  v-model="event.tone"
                  :aria-label="`Tone for ${event.title}`"
                  class="h-7 !py-1 !text-[10px]"
                >
                  <option value="klaxon">Siren Klaxon</option>
                  <option value="pulse">Rapid Pulse</option>
                  <option value="chime">Double Chime</option>
                  <option value="thud">Low Warning</option>
                  <option value="beep">Soft Beep</option>
                </select>
              </td>
              <td>
                <input
                  v-model.number="event.gain"
                  type="number"
                  min="0"
                  max="100"
                  class="h-7 w-16 !py-1 !text-[10px]"
                  :aria-label="`Gain for ${event.title}`"
                />
              </td>
              <td>
                <button
                  class="btn"
                  :disabled="Boolean(playing)"
                  :aria-label="`Test tone ${event.title}`"
                  @click="test(event)"
                >
                  <AppIcon :name="playing === event.id ? 'volume_up' : 'play_arrow'" />Test
                </button>
              </td>
              <td><SettingToggle v-model="event.enabled" :label="`Audio for ${event.title}`" /></td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
    <section class="panel p-3">
      <div class="flex items-center gap-2">
        <AppIcon name="bedtime" class="!text-lg text-blue-600" />
        <h3 class="text-xs font-semibold">Scheduled Audio Mute / Night Protocol</h3>
        <SettingToggle v-model="draft.audio.quiet" label="Scheduled audio mute" class="ml-auto" />
      </div>
      <div class="mt-3 flex flex-wrap items-end gap-3">
        <label class="field">From<input v-model="draft.audio.quietFrom" type="time" /></label
        ><span class="pb-2 text-xs text-slate-400">to</span
        ><label class="field">Until<input v-model="draft.audio.quietTo" type="time" /></label
        ><span class="pb-2 text-[10px] text-slate-400">Asia/Jakarta</span
        ><label class="flex items-center gap-2 pb-2 text-[11px]"
          ><input
            v-model="draft.audio.neverMuteCritical"
            type="checkbox"
            class="rounded text-blue-700"
          />Never mute P1 Critical</label
        >
      </div>
      <p class="mt-2 text-[9px] text-slate-400">
        Only browser audio is muted. Visual notifications and incident history remain available.
      </p>
    </section>
  </div>
</template>
