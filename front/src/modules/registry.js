export const coreVersion = '4.8.2'
// Only manifests included in this reviewed build are executable modules.
const viewLoaders = import.meta.glob('./*/views/*.vue')
export function moduleView(module) {
  return module.entry ? viewLoaders[module.entry] : null
}
const manifests = import.meta.glob('./*/manifest.json', { eager: true, import: 'default' })
const order = [
  'core.gis.tracker',
  'telemetry.cfs300b.ble',
  'ext.queclink.kinematics',
  'ext.fleet.maintenance',
  'core.geofence.spatial',
  'ext.socket.queclink.gtfri',
]
export const moduleCatalog = Object.values(manifests).sort(
  (a, b) => order.indexOf(a.id) - order.indexOf(b.id),
)
export function moduleForPath(path) {
  return moduleCatalog.find((module) => module.paths.includes(path))
}
export function checkManifest(input) {
  if (!input || typeof input !== 'object' || Array.isArray(input))
    throw new Error('The manifest must be a JSON object.')
  if (input.schemaVersion !== 1)
    throw new Error('Unsupported manifest schema. Expected schemaVersion 1.')
  const known = moduleCatalog.find((module) => module.id === input.id)
  if (!known)
    throw new Error(
      'This module is not included in the current frontend build. Publish its reviewed bundle through the backend installer first.',
    )
  if (known.version !== input.version)
    throw new Error(
      `This build supports ${known.id} v${known.version}. The uploaded version is not available.`,
    )
  if (input.kind !== known.kind)
    throw new Error('Module classification does not match the bundled catalog.')
  const current = coreVersion.split('.').map(Number),
    required = known.minCore.split('.').map(Number)
  const belowMinimum =
    current[0] < required[0] ||
    (current[0] === required[0] &&
      (current[1] < required[1] || (current[1] === required[1] && current[2] < required[2])))
  if (current[0] !== known.coreMajor || belowMinimum)
    throw new Error('This package is incompatible with the current core runtime.')
  return known // Uploaded declarations never replace trusted routes, permissions or executable code.
}
