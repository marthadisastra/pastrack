import { computed, ref } from 'vue'
import { moduleCatalog } from './registry'
import { useMock } from '../services/http'
const key = 'pastrack.demo.settings.v1'
export const permissionColumns = [
  'Live Tracking',
  'Asset Config',
  'Fuel CFS300B',
  'Maintenance',
  'Geofencing',
  'Shock/Kinematic',
  'Relay Cut (DO1)',
  'System Settings',
]
export function defaultSettings() {
  return {
    modules: Object.fromEntries(
      moduleCatalog.map((module) => [
        module.id,
        {
          installed: true,
          enabled: true,
          config: Object.fromEntries(module.config.map((field) => [field.key, field.default])),
        },
      ]),
    ),
    branding: {
      name: 'PASTRACK',
      tagline: 'FLEET TELEMATICS',
      organization: 'PASTRACK Telematics',
      color: '#1d4ed8',
      support: 'Dispatch NOC',
      domain: '',
      logo: '',
      favicon: '',
      touchIcon: '',
    },
    audio: {
      enabled: false,
      volume: 60,
      output: 'browser',
      quiet: false,
      quietFrom: '22:00',
      quietTo: '06:00',
      neverMuteCritical: true,
      events: [
        {
          id: 'siphon',
          title: 'Fuel Theft / Siphon',
          condition: 'Fuel drop > 5 L while ACC is off',
          severity: 'P1',
          tone: 'klaxon',
          gain: 100,
          enabled: true,
        },
        {
          id: 'power',
          title: 'Power Disconnect / Aki Cut',
          condition: 'Main power lost with backup battery engaged',
          severity: 'P1',
          tone: 'pulse',
          gain: 90,
          enabled: true,
        },
        {
          id: 'geofence',
          title: 'Geofence Perimeter Breach',
          condition: 'Boundary exit without route clearance',
          severity: 'P2',
          tone: 'chime',
          gain: 75,
          enabled: true,
        },
        {
          id: 'impact',
          title: 'Kinematic Shock / Severe Impact',
          condition: 'Accelerometer impact threshold exceeded',
          severity: 'P2',
          tone: 'thud',
          gain: 80,
          enabled: true,
        },
        {
          id: 'ble',
          title: 'CFS300B BLE Probe Drop',
          condition: 'Probe broadcast timeout > 180 seconds',
          severity: 'P3',
          tone: 'chime',
          gain: 50,
          enabled: true,
        },
        {
          id: 'battery',
          title: 'Low Probe Battery',
          condition: 'Probe cell voltage < 3.00 V',
          severity: 'P4',
          tone: 'beep',
          gain: 40,
          enabled: true,
        },
      ],
    },
    operators: [
      {
        id: 'admin',
        name: 'Super Admin',
        email: 'admin@pastrack.internal',
        role: 'Super Admin',
        twoFactor: true,
      },
      {
        id: 'dispatch',
        name: 'Fleet Operations Dispatcher',
        email: 'dispatch.lead@pastrack.internal',
        role: 'Dispatcher',
        twoFactor: true,
      },
      {
        id: 'workshop',
        name: 'Workshop Lead Specialist',
        email: 'sensor.tech@pastrack.internal',
        role: 'Workshop Lead',
        twoFactor: false,
      },
      {
        id: 'audit',
        name: 'Telematics Compliance Auditor',
        email: 'audit.external@compliance.corp',
        role: 'Auditor',
        twoFactor: true,
      },
    ],
    roles: [
      { name: 'Super Admin', permissions: permissionColumns.map(() => true) },
      { name: 'Dispatcher', permissions: [true, true, true, false, true, true, false, false] },
      { name: 'Workshop Lead', permissions: [true, true, true, true, false, false, false, false] },
      { name: 'Auditor', permissions: [true, false, true, false, true, false, false, false] },
    ],
  }
}
function restored() {
  const defaults = defaultSettings()
  if (!useMock) return defaults
  try {
    const value = JSON.parse(localStorage.getItem(key) || 'null')
    if (!value || value.schemaVersion !== 1 || !value.settings) return defaults
    const saved = value.settings
    for (const module of moduleCatalog) {
      const current = saved.modules?.[module.id]
      if (current)
        defaults.modules[module.id] = {
          installed: module.kind === 'core' || current.installed === true,
          enabled:
            module.kind === 'core' || (current.installed === true && current.enabled === true),
          config: { ...defaults.modules[module.id].config, ...current.config },
        }
    }
    if (
      saved.branding &&
      typeof saved.branding.name === 'string' &&
      /^#[0-9a-f]{6}$/i.test(saved.branding.color)
    )
      defaults.branding = { ...defaults.branding, ...saved.branding }
    if (
      saved.audio &&
      Array.isArray(saved.audio.events) &&
      Number.isFinite(saved.audio.volume) &&
      saved.audio.volume >= 0 &&
      saved.audio.volume <= 100 &&
      defaults.audio.events.every((event) =>
        saved.audio.events.some(
          (current) =>
            current.id === event.id &&
            Number.isFinite(current.gain) &&
            current.gain >= 0 &&
            current.gain <= 100,
        ),
      )
    )
      defaults.audio = saved.audio
    if (
      Array.isArray(saved.roles) &&
      saved.roles.every(
        (role) =>
          Array.isArray(role.permissions) &&
          role.permissions.length === permissionColumns.length &&
          role.permissions.every((value) => typeof value === 'boolean'),
      ) &&
      saved.roles.some((role) => role.name === 'Super Admin' && role.permissions.every(Boolean))
    )
      defaults.roles = saved.roles
    if (Array.isArray(saved.operators)) defaults.operators = saved.operators
  } catch {
    /* Invalid local preview state uses defaults. */
  }
  return defaults
}
export const settingsState = ref(restored())
export const enabledModuleCount = computed(
  () => moduleCatalog.filter((module) => isModuleActive(module.id)).length,
)
export function isModuleActive(id) {
  if (!id) return true
  const module = moduleCatalog.find((item) => item.id === id)
  return (
    module?.kind === 'core' ||
    Boolean(settingsState.value.modules[id]?.installed && settingsState.value.modules[id]?.enabled)
  )
}
export function commitSettings(value) {
  if (useMock) localStorage.setItem(key, JSON.stringify({ schemaVersion: 1, settings: value }))
  settingsState.value = JSON.parse(JSON.stringify(value))
}
