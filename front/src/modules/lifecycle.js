import { moduleCatalog, checkManifest } from './registry'
export function setModuleEnabled(draft, id, enabled) {
  const module = moduleCatalog.find((module) => module.id === id)
  if (!module) throw new Error('Unknown module.')
  if (module.kind === 'core') throw new Error('Core modules cannot be disabled.')
  if (!draft.modules[id].installed) throw new Error('Install this module before enabling it.')
  if (!enabled) {
    const dependents = moduleCatalog.filter(
      (module) => module.dependencies.includes(id) && draft.modules[module.id].enabled,
    )
    if (dependents.length)
      throw new Error(
        `Disable dependent modules first: ${dependents.map((module) => module.name).join(', ')}.`,
      )
  }
  draft.modules[id].enabled = enabled
}
export function uninstallModule(draft, id) {
  setModuleEnabled(draft, id, false)
  draft.modules[id].installed = false
}
export function installModule(draft, manifest) {
  const module = checkManifest(manifest)
  if (draft.modules[module.id].installed) throw new Error('This module is already installed.')
  for (const id of module.dependencies)
    if (!draft.modules[id]?.installed || !draft.modules[id]?.enabled)
      throw new Error(`Required dependency is inactive: ${id}.`)
  draft.modules[module.id].installed = true
  draft.modules[module.id].enabled = true
  return module
}
