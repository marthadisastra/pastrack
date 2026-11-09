import { test, expect } from '@playwright/test'

test('dashboard inspection, state filter, map controls, and CSV export', async ({ page }) => {
  const errors = []
  page.on('pageerror', (error) => errors.push(error.message))
  await page.goto('/#/dashboard')
  await expect(page.getByRole('heading', { name: 'Operations Telematics Dashboard' })).toBeVisible()
  await page.getByRole('button', { name: 'Inspect', exact: true }).first().click()
  await expect(page.getByRole('dialog')).toContainText('HAULER-04')
  await page.keyboard.press('Escape')
  await expect(page.getByRole('dialog')).toHaveCount(0)
  await page.getByRole('button', { name: 'Filters', exact: true }).click()
  await page.getByLabel('Operational State').selectOption('alert')
  await expect(page.locator('tbody tr')).toHaveCount(1)
  const download = page.waitForEvent('download')
  await page.getByRole('button', { name: 'Export Log' }).click()
  expect((await download).suggestedFilename()).toBe('fleet-telemetry.csv')
  await expect(page.getByRole('button', { name: 'Zoom in' })).toBeVisible()
  expect(errors).toEqual([])
})

test('asset registration, search, grid and gateway pairing', async ({ page }) => {
  await page.goto('/#/vehicles')
  await page.getByRole('button', { name: 'Add Asset', exact: true }).click()
  await page.getByLabel('Asset ID *', { exact: true }).fill('TEST-07')
  await page.getByLabel('License Plate *').fill('B 7777 TEST')
  await page.getByLabel('Make / Model *').fill('Scania Test Truck')
  await page.getByRole('button', { name: 'Register Asset', exact: true }).click()
  await expect(page.getByRole('dialog')).toHaveCount(0)
  await page.getByPlaceholder('Search Unit ID, Plate, VIN, Make, or IMEI…').fill('TEST-07')
  await expect(page.locator('tbody tr')).toHaveCount(1)
  await expect(page.locator('tbody')).toContainText('TEST-07')
  await page.locator('tbody').getByRole('button', { name: 'Pair GPS', exact: true }).first().click()
  await page.getByLabel('GV30CEU Gateway IMEI *').fill('123456789012345')
  await page.getByRole('button', { name: 'Pair Gateway', exact: true }).click()
  await expect(page.getByRole('dialog')).toHaveCount(0)
  await expect(page.locator('tbody')).toContainText('123456789012345')
  await page.getByRole('button', { name: 'grid', exact: true }).click()
  await expect(page.getByRole('heading', { name: 'TEST-07' })).toBeVisible()
})

test('maintenance transitions into service history and schedules a new order', async ({ page }) => {
  await page.goto('/#/maintenance')
  const row = page.locator('tr').filter({ hasText: 'WO-8819' })
  await row.getByRole('button', { name: 'Complete WO' }).click()
  await page.getByRole('button', { name: 'Confirm Update' }).click()
  await page.getByRole('button', { name: /Past Service History/ }).click()
  await expect(page.locator('tbody')).toContainText('WO-8819')
  await page.getByRole('button', { name: 'Schedule Maintenance', exact: true }).click()
  await page.getByLabel('Work Order Title *').fill('Preview oil service')
  await page.getByRole('button', { name: 'Create Maintenance Schedule' }).click()
  await expect(page.getByRole('dialog')).toHaveCount(0)
  await expect(page.locator('tbody')).toContainText('Preview oil service')
})

test('tracking selects units and exposes the telemetry tab without mutations', async ({ page }) => {
  const mutations = []
  page.on('request', (request) => {
    if (['POST', 'PUT', 'PATCH', 'DELETE'].includes(request.method())) mutations.push(request.url())
  })
  await page.goto('/#/tracking')
  await page.locator('.tracking-roster button').filter({ hasText: 'HAULER-04' }).click()
  await expect(page.locator('.tracking-hud').getByText('HAULER-04', { exact: true })).toBeVisible()
  await page.getByRole('tab', { name: 'telemetry', exact: true }).click()
  await expect(page.getByRole('tabpanel', { name: 'Telemetry' })).toContainText('GPS TELEMETRY')
  await expect(page.getByRole('tabpanel', { name: 'Telemetry' })).toContainText(
    'FUEL / SENSOR TELEMETRY',
  )
  expect(mutations).toEqual([])
})

test('demo authentication and password recovery routes work', async ({ page }) => {
  await page.goto('/#/login')
  await page.getByRole('textbox', { name: 'Email or Username' }).fill('demo@example.com')
  await page.locator('input[name=password]').fill('fail')
  await page.getByRole('button', { name: 'Sign In to Pastrack' }).click()
  await expect(page.getByRole('alert')).toContainText('Demo invalid credentials')
  await page.locator('input[name=password]').fill('FleetPreview123!')
  await page.getByRole('button', { name: 'Sign In to Pastrack' }).click()
  await expect(page).toHaveURL(/#\/dashboard$/)
  await page.goto('/#/forgot-password')
  await page.getByRole('textbox', { name: 'Work Email Address' }).fill('demo@example.com')
  await page.getByRole('button', { name: 'Send Reset Link' }).click()
  await page.getByRole('link', { name: /Preview set new password/ }).click()
  await page.getByLabel('New Password', { exact: true }).fill('FleetPreview123!')
  await page.getByLabel('Confirm Password', { exact: true }).fill('FleetPreview123!')
  await page.getByRole('button', { name: 'Update Password' }).click()
  await expect(page.getByRole('heading', { name: 'Credentials Synchronized' })).toBeVisible()
})

test('mobile navigation and all routes render without page overflow', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 })
  for (const path of [
    'dashboard',
    'vehicles',
    'maintenance',
    'tracking',
    'template',
    'login',
    'forgot-password',
    'reset-password',
  ]) {
    await page.goto(`/#/${path}`)
    await expect(page.locator('h1')).toBeVisible()
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
  }
  await page.goto('/#/dashboard')
  await page.getByRole('button', { name: 'Open navigation' }).click()
  await page.getByRole('link', { name: 'All Assets' }).click()
  await expect(page).toHaveURL(/#\/vehicles$/)
})

test('sidebar matches all mockup groups and supports compact icon mode', async ({ page }) => {
  await page.setViewportSize({ width: 1366, height: 768 })
  await page.goto('/#/dashboard')
  const nav = page.getByRole('navigation', { name: 'Main navigation' })
  await expect(nav.getByRole('link')).toHaveCount(15)
  expect(
    await nav
      .getByRole('link')
      .evaluateAll((links) => links.map((link) => link.getAttribute('aria-label'))),
  ).toEqual([
    'Dashboard',
    'Live Tracking',
    'All Assets',
    'Maintenance',
    'Inspections',
    'Trips & Journey',
    'Geofence',
    'Driving Behavior',
    'Fuel Monitoring',
    'Alert Center',
    'Reports & Analytics',
    'Devices: GPS GV30CEU',
    'Devices: Fuel CFS300B',
    'Raw Telemetry Stream',
    'Settings',
  ])
  for (const group of ['Overview', 'Fleet Management', 'Intelligence', 'Hardware & System']) {
    await expect(nav.getByText(group, { exact: true })).toBeVisible()
  }
  expect(await nav.evaluate((el) => el.scrollHeight - el.clientHeight)).toBeLessThanOrEqual(1)
  await page.getByRole('button', { name: 'Collapse sidebar', exact: true }).click()
  await expect.poll(async () => (await page.locator('.sidebar').boundingBox()).width).toBe(56)
  await nav.getByRole('link', { name: 'Inspections', exact: true }).click()
  await expect(
    page.getByRole('heading', { name: 'Inspections', exact: true, level: 1 }),
  ).toBeVisible()
  await expect(page.getByText(/Menu ini tersedia pada master layout mockup/)).toBeVisible()
  await page.getByRole('button', { name: 'Expand sidebar', exact: true }).click()
  await expect.poll(async () => (await page.locator('.sidebar').boundingBox()).width).toBe(216)
})

test('tracking fits a laptop viewport and minimizes both panels', async ({ page }) => {
  await page.setViewportSize({ width: 1366, height: 768 })
  await page.goto('/#/tracking')
  await expect(page.getByRole('heading', { name: 'Live Tracking', exact: true })).toBeVisible()
  const main = page.locator('#main-content')
  expect(await main.evaluate((el) => el.scrollHeight - el.clientHeight)).toBeLessThanOrEqual(1)
  await expect(page.getByLabel('Realtime fleet map')).toBeVisible()
  await page.getByRole('button', { name: 'Minimize fleet roster' }).click()
  await expect(page.locator('.tracking-roster')).toHaveCount(0)
  await page.getByRole('button', { name: 'Minimize telemetry panel' }).click()
  await expect(page.locator('.tracking-hud')).toHaveCount(0)
  await page.getByRole('button', { name: 'Roster', exact: true }).click()
  await page.getByRole('button', { name: 'Telemetry', exact: true }).click()
  await expect(page.locator('.tracking-roster')).toBeVisible()
  await expect(page.locator('.tracking-hud')).toBeVisible()
})
