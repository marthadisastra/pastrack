import { test, expect } from '@playwright/test'
import { mergeFuelSamples } from '../src/services/fuelHistory'

test('fuel sample merge is vehicle-scoped, idempotent and event-time ordered', () => {
  const meta = { from: '2026-11-09T00:00:00Z', to: '2026-11-09T01:00:00Z' }
  const initial = [{ sampleId: '10', at: '2026-11-09T00:30:00Z', liters: 100 }]
  const samples = [
    { vehicleId: '11', sampleId: '12', at: '2026-11-09T01:02:00Z', liters: 102 },
    { vehicleId: '11', sampleId: '11', at: '2026-11-09T00:45:00Z', liters: 101 },
    { vehicleId: '12', sampleId: '99', at: '2026-11-09T01:03:00Z', liters: 999 },
    { vehicleId: '11', sampleId: '12', at: '2026-11-09T01:02:00Z', liters: 102 },
  ]
  const merged = mergeFuelSamples(initial, samples, '11', meta)
  expect(merged.points.map((point) => point.sampleId)).toEqual(['10', '11', '12'])
  expect(merged.to).toBe('2026-11-09T01:02:00.000Z')
})
const response = (fuel = 0, gps = {}) => ({
  meta: { schemaVersion: 1, asOf: '2026-11-07T03:00:00Z', mode: 'snapshot', realtime: false },
  data: {
    summary: {
      totalVehicles: 1,
      monitoredVehicles: 1,
      movementCounts: { MOVING: 0, STOPPED: 1, UNASSIGNED: 0 },
      devicesOnline: 1,
      activeAlerts: 0,
      gpsFreshVehicles: 1,
      fuelLiters: fuel,
      fuelCapacityLiters: 400,
      fuelFillPercent: fuel === null ? null : 0,
      fuelValidFreshVehicles: fuel === null ? 0 : 1,
      fuelCapacityEligibleVehicles: 1,
      calibrationUnknownVehicles: 0,
      averageHdop: 0.8,
      hdopSamples: 1,
    },
    vehicles: [
      {
        id: '411',
        code: 'UNIT-A',
        plate: 'DB 411',
        name: 'Contract fixture',
        deviceId: '9001',
        deviceIdentifier: '••••4321',
        monitored: true,
        movementStatus: 'STOPPED',
        connectivity: 'ONLINE',
        activeAlerts: 0,
        alertCodes: [],
        gps: {
          latitude: 0,
          longitude: 0,
          speedKmh: 0,
          headingDeg: null,
          satellites: 8,
          hdop: 0.8,
          fixStatus: 'valid',
          fresh: true,
          ...gps,
        },
        fuel: {
          liters: fuel,
          percent: 0,
          capacityLiters: 400,
          calibrationId: '8',
          validity: 'valid',
          fresh: true,
        },
        ignition: {
          status: 'UNKNOWN',
          source: null,
          reportedAt: null,
          physicalState: 'NOT_INSTALLED',
        },
      },
    ],
  },
})
const historyResponse = (range = '12h') => ({
  data: {
    vehicleId: '411',
    range,
    points: [
      {
        sampleId: '91',
        at: '2026-11-07T02:00:00Z',
        liters: 120,
        percent: 30,
        validity: 'valid',
        calibrationId: '8',
      },
      {
        sampleId: '92',
        at: '2026-11-07T02:30:00Z',
        liters: null,
        percent: null,
        validity: 'invalid',
        calibrationId: null,
      },
      {
        sampleId: '93',
        at: '2026-11-07T03:00:00Z',
        liters: 118,
        percent: 29.5,
        validity: 'valid',
        calibrationId: '8',
      },
    ],
  },
  meta: {
    schemaVersion: 1,
    from: '2026-11-06T15:00:00Z',
    to: '2026-11-07T03:00:00Z',
    limit: 5000,
  },
})
test.beforeEach(async ({ page }) => {
  await page.route('**/api/v1/dashboard/fuel-history**', (route) => {
    const range = new URL(route.request().url()).searchParams.get('range') || '12h'
    return route.fulfill({ json: historyResponse(range) })
  })
})
test('realtime map renders only valid geographic fixes and updates markers in place', async ({
  page,
}) => {
  let value = response(20, { latitude: -6.2146, longitude: 106.8451 })
  await page.route('**/api/v1/dashboard/snapshot', (route) => route.fulfill({ json: value }))
  await page.route('https://tile.openstreetmap.org/**', (route) => route.abort())
  await page.goto('/#/dashboard')
  await expect(page.getByRole('button', { name: 'Notifications' })).toBeVisible()
  const map = page.getByLabel('Realtime fleet map')
  await expect(map).toBeVisible()
  await expect(map.locator('.pastrack-map-marker')).toHaveCount(1)
  await expect(map.getByText('1 VALID FIX · LIVE WSS')).toBeVisible()
  await expect(page.getByRole('img', { name: '7d fuel history for UNIT-A' })).toBeVisible()
  const history12 = page.waitForRequest(
    (request) =>
      request.url().includes('/dashboard/fuel-history') && request.url().includes('range=12h'),
  )
  await page.getByRole('button', { name: '12h', exact: true }).click()
  await history12
  await expect(page.getByRole('img', { name: '12h fuel history for UNIT-A' })).toBeVisible()
  const history24 = page.waitForRequest(
    (request) =>
      request.url().includes('/dashboard/fuel-history') && request.url().includes('range=24h'),
  )
  await page.getByRole('button', { name: '24h', exact: true }).click()
  await history24
  await expect(page.getByRole('img', { name: '24h fuel history for UNIT-A' })).toBeVisible()
  value = response(20, { latitude: -6.22, longitude: 106.86 })
  await page.getByRole('button', { name: 'Refresh dashboard', exact: true }).click()
  await expect(map.locator('.pastrack-map-marker')).toHaveCount(1)
  value = response(20, { latitude: null, longitude: null, fixStatus: 'no_fix', fresh: false })
  await page.getByRole('button', { name: 'Refresh dashboard', exact: true }).click()
  await expect(map.locator('.pastrack-map-marker')).toHaveCount(0)
  await expect(
    map.getByText('No valid GPS fixes are available for the current filter.'),
  ).toBeVisible()
})
test('dashboard uses its own SQL contract and preserves null versus zero without demo requests', async ({
  page,
}) => {
  const paths = []
  page.on('request', (r) => {
    if (r.url().includes('/api/v1/')) paths.push(new URL(r.url()).pathname)
  })
  let value = response()
  await page.route('**/api/v1/dashboard/snapshot', (route) => route.fulfill({ json: value }))
  await page.goto('/#/dashboard')
  await expect(page.getByText('SNAPSHOT', { exact: true })).toBeVisible()
  const row = page.getByRole('row').filter({ hasText: 'UNIT-A' })
  await expect(row).toContainText('STOPPED')
  await expect(row).toContainText('0.0 / 400 L')
  await expect(row).toContainText('UNKNOWN')
  value = response(null)
  await page.getByRole('button', { name: 'Refresh dashboard', exact: true }).click()
  await expect(row).toContainText('— / 400 L')
  await expect(page.getByText(/NaN|Infinity/)).toHaveCount(0)
  await page.waitForTimeout(4500)
  expect(paths).toEqual([
    '/api/v1/dashboard/snapshot',
    '/api/v1/dashboard/fuel-history',
    '/api/v1/dashboard/snapshot',
  ])
  await page.setViewportSize({ width: 390, height: 844 })
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
})
test('dashboard distinguishes an API failure from a valid empty fleet and retains prior data', async ({
  page,
}) => {
  let failed = true
  await page.route('**/api/v1/dashboard/snapshot', (route) =>
    failed
      ? route.fulfill({ status: 503, json: { error: { message: 'Database unavailable' } } })
      : route.fulfill({ json: response() }),
  )
  await page.goto('/#/dashboard')
  await expect(page.getByRole('alert')).toContainText('Database unavailable')
  await expect(page.getByText('0 Monitored', { exact: true })).toHaveCount(0)
  failed = false
  await page.getByRole('button', { name: 'Retry', exact: true }).click()
  await expect(page.getByRole('row').filter({ hasText: 'UNIT-A' })).toBeVisible()
  failed = true
  await page.getByRole('button', { name: 'Refresh dashboard', exact: true }).click()
  await expect(page.getByRole('alert')).toContainText('previous snapshot is retained')
  await expect(page.getByRole('row').filter({ hasText: 'UNIT-A' })).toBeVisible()
})
