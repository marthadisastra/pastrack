import { request, useMock } from './http'
import { authBaseURL } from './auth'
import { moduleCatalog } from '../modules/registry'
export function validateSettings(value) {
  if (
    !value ||
    !value.modules ||
    !value.branding ||
    !value.audio ||
    !Array.isArray(value.audio.events) ||
    !Array.isArray(value.roles) ||
    !Array.isArray(value.operators)
  )
    throw new Error(
      'Unsupported settings response. Normalize the API response to the Settings schema.',
    )
  if (typeof value.branding.name !== 'string') throw new Error('Application name must be text.')
  if (!value.branding.name.trim() || value.branding.name.length > 40)
    throw new Error('Application name is required and must be 40 characters or fewer.')
  if (!/^#[0-9a-f]{6}$/i.test(value.branding.color))
    throw new Error('Choose a valid six-digit brand color.')
  if (
    value.branding.domain &&
    !/^(?=.{1,253}$)([a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?\.)+[a-z]{2,63}$/i.test(
      value.branding.domain,
    )
  )
    throw new Error('Enter a domain name without https:// or a URL path.')
  for (const module of moduleCatalog) {
    const state = value.modules[module.id]
    if (!state) throw new Error(`Missing module state: ${module.name}.`)
    if (module.kind === 'core' && (!state.enabled || !state.installed))
      throw new Error(`${module.name} is a protected core module.`)
    if (state.installed && state.enabled)
      for (const dependency of module.dependencies)
        if (!value.modules[dependency]?.installed || !value.modules[dependency]?.enabled)
          throw new Error(`${module.name} requires ${dependency}.`)
    for (const field of module.config) {
      const number = Number(state.config[field.key])
      if (!Number.isFinite(number) || number < field.min || number > field.max)
        throw new Error(
          `${module.name}: ${field.label} must be between ${field.min} and ${field.max}.`,
        )
    }
  }
  if (!Number.isFinite(value.audio.volume) || value.audio.volume < 0 || value.audio.volume > 100)
    throw new Error('Master volume must be between 0 and 100.')
  if (!/^\d{2}:\d{2}$/.test(value.audio.quietFrom) || !/^\d{2}:\d{2}$/.test(value.audio.quietTo))
    throw new Error('Set valid quiet-window times.')
  for (const event of value.audio.events) {
    if (!Number.isFinite(event.gain) || event.gain < 0 || event.gain > 100)
      throw new Error('Event gain must be between 0 and 100.')
  }
  if (
    value.roles.some(
      (role) =>
        !Array.isArray(role.permissions) ||
        role.permissions.length !== 8 ||
        role.permissions.some((value) => typeof value !== 'boolean'),
    )
  )
    throw new Error('Every role must define all eight permission flags.')
  if (!value.roles.some((role) => role.name === 'Super Admin' && role.permissions.every(Boolean)))
    throw new Error('The Super Admin policy must retain all permissions.')
}
export const settingsService = {
  async load() {
    if (useMock) return null
    let value
    try {
      value = await request('/settings', { baseURL: authBaseURL })
    } catch (error) {
      // The v2 backend does not expose the legacy settings document yet.
      // Keep the local settings schema usable while backend modules are managed separately.
      if (error?.status === 404) return null
      throw error
    }
    validateSettings(value)
    return value
  },
  async save(value) {
    validateSettings(value)
    if (useMock) {
      await new Promise((resolve) => setTimeout(resolve, 280))
      return JSON.parse(JSON.stringify(value))
    }
    let saved
    try {
      saved = await request('/settings', { baseURL: authBaseURL, method: 'PUT', body: value })
    } catch (error) {
      if (error?.status === 404) return JSON.parse(JSON.stringify(value))
      throw error
    }
    const result = saved ?? JSON.parse(JSON.stringify(value))
    validateSettings(result)
    return result
  },
}
