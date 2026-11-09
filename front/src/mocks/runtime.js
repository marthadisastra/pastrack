import { vehicles, workOrders } from './fleet'
// Shared demo records; individual module services own their operations.
export const demoVehicles = structuredClone(vehicles)
export const demoOrders = structuredClone(workOrders)
