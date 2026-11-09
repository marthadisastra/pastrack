// Labels, order, groups and icons follow the master-layout mockup.
import { moduleForPath } from '../modules/registry'
const baseNavigation = [
  { group: 'Overview', items: [{ to: '/dashboard', label: 'Dashboard', icon: 'space_dashboard' }] },
  {
    group: 'Fleet Management',
    items: [
      { to: '/tracking', label: 'Live Tracking', icon: 'near_me' },
      { to: '/vehicles', label: 'All Assets', icon: 'local_shipping', badge: 'assets' },
      { to: '/maintenance', label: 'Maintenance', icon: 'build_circle' },
      { to: '/inspections', label: 'Inspections', icon: 'fact_check', placeholder: true },
      { to: '/trips', label: 'Trips & Journey', icon: 'route', placeholder: true },
      { to: '/geofence', label: 'Geofence', icon: 'fence', placeholder: true },
      { to: '/driving-behavior', label: 'Driving Behavior', icon: 'speed', placeholder: true },
    ],
  },
  {
    group: 'Intelligence',
    items: [
      {
        to: '/fuel-monitoring',
        label: 'Fuel Monitoring',
        icon: 'local_gas_station',
        dot: true,
        placeholder: true,
      },
      { to: '/alerts', label: 'Alert Center', icon: 'warning', badge: 'alerts', placeholder: true },
      { to: '/reports', label: 'Reports & Analytics', icon: 'analytics', placeholder: true },
    ],
  },
  {
    group: 'Hardware & System',
    items: [
      { to: '/devices/gps', label: 'Devices: GPS GV30CEU', icon: 'router', placeholder: true },
      { to: '/devices/fuel', label: 'Devices: Fuel CFS300B', icon: 'sensors', placeholder: true },
      { to: '/telemetry', label: 'Raw Telemetry Stream', icon: 'terminal', placeholder: true },
      { to: '/settings', label: 'Settings', icon: 'settings' },
    ],
  },
]
export const navigation = baseNavigation.map((group) => ({
  ...group,
  items: group.items.map((item) => ({ ...item, moduleId: moduleForPath(item.to)?.id })),
}))
export const pendingModules = navigation
  .flatMap((group) => group.items)
  .filter((item) => item.placeholder)
