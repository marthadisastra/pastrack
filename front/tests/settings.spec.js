import { test, expect } from '@playwright/test'

const maintenance = 'Preventive Maintenance & Work Orders'
async function settings(page) {
  await page.goto('/#/settings')
  await expect(
    page.getByRole('heading', { name: 'System Settings & Extension Engine' }),
  ).toBeVisible()
}
async function save(page) {
  await page.getByRole('button', { name: 'Save Changes', exact: true }).click()
  await expect(page.getByRole('button', { name: 'Save Changes', exact: true })).toBeDisabled()
  await expect(
    page.getByText('Settings saved in this browser. Module menus updated.', { exact: true }),
  ).toBeVisible()
}

test('settings follows the mockup with an additional SMTP tab and stays compact on mobile', async ({
  page,
}) => {
  await settings(page)
  await expect(page.getByRole('tab')).toHaveCount(5)
  await expect(page.getByRole('table').locator('tbody tr')).toHaveCount(6)
  await expect(page.getByText('Active (Locked)', { exact: true })).toHaveCount(3)
  for (const tab of [
    'Branding & Web Assets',
    'Sound & Audio Alerts',
    'Access & RBAC',
    'Module Manager',
  ]) {
    await page.getByRole('tab', { name: new RegExp(tab.replaceAll('&', '&')) }).click()
    await expect(page.getByRole('tabpanel')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Save Changes', exact: true })).toBeDisabled()
  }
  await page.setViewportSize({ width: 390, height: 844 })
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true)
})

test('disabling a module affects navigation and rejects its direct URL after reload', async ({
  page,
}) => {
  await settings(page)
  await page.getByRole('switch', { name: `Enable ${maintenance}`, exact: true }).click()
  await expect(
    page
      .getByRole('navigation', { name: 'Main navigation' })
      .getByRole('link', { name: 'Maintenance', exact: true }),
  ).toBeVisible()
  await save(page)
  await expect(
    page
      .getByRole('navigation', { name: 'Main navigation' })
      .getByRole('link', { name: 'Maintenance', exact: true }),
  ).toHaveCount(0)
  await page.goto('/#/maintenance')
  await expect(page.getByRole('heading', { name: 'Module unavailable' })).toBeVisible()
  await page.getByRole('link', { name: 'Open Module Manager' }).click()
  await page.getByRole('switch', { name: `Enable ${maintenance}`, exact: true }).click()
  await save(page)
  await page
    .getByRole('navigation', { name: 'Main navigation' })
    .getByRole('link', { name: 'Maintenance', exact: true })
    .click()
  await expect(
    page.getByRole('heading', { name: 'Vehicles: Maintenance & Work Orders' }),
  ).toBeVisible()
  await expect(page.locator('tbody')).toContainText('WO-8821')
})

test('uninstall, package validation and catalog reinstall preserve module records', async ({
  page,
}) => {
  await settings(page)
  await page.getByRole('button', { name: `Uninstall ${maintenance}`, exact: true }).click()
  await page.getByRole('button', { name: 'Stage uninstall', exact: true }).click()
  await save(page)
  await page.getByRole('button', { name: 'Upload Module', exact: true }).click()
  const file = page.getByLabel('Module package file')
  await file.setInputFiles({
    name: 'unsafe.zip',
    mimeType: 'application/zip',
    buffer: Buffer.from('not executable'),
  })
  await expect(page.getByRole('dialog')).toContainText('requires the Go package installer')
  await file.setInputFiles({
    name: 'unknown.json',
    mimeType: 'application/json',
    buffer: Buffer.from(
      JSON.stringify({ schemaVersion: 1, id: 'ext.unknown', version: '1.0.0', kind: 'extension' }),
    ),
  })
  await expect(page.getByRole('dialog')).toContainText('not included in the current frontend build')
  const manifest = await (await page.request.get('/examples/maintenance.module.json')).json()
  await file.setInputFiles({
    name: 'maintenance.json',
    mimeType: 'application/json',
    buffer: Buffer.from(JSON.stringify({ ...manifest, version: '99.0.0' })),
  })
  await expect(page.getByRole('dialog')).toContainText('uploaded version is not available')
  await file.setInputFiles({
    name: 'maintenance.json',
    mimeType: 'application/json',
    buffer: Buffer.from(JSON.stringify(manifest)),
  })
  await page.getByRole('button', { name: 'Stage Catalog Install' }).click()
  await save(page)
  await page
    .getByRole('navigation', { name: 'Main navigation' })
    .getByRole('link', { name: 'Maintenance', exact: true })
    .click()
  await expect(page.locator('tbody')).toContainText('WO-8819')
})

test('branding changes are applied only after save and survive reload', async ({ page }) => {
  await settings(page)
  await page.getByRole('tab', { name: 'Branding & Web Assets' }).click()
  await page.getByLabel('Application Name', { exact: true }).fill('Atlas Fleet')
  await expect(page.locator('.sidebar')).toContainText('PASTRACK')
  await save(page)
  await expect(page.locator('.sidebar')).toContainText('Atlas Fleet')
  await expect(page).toHaveTitle('Settings · Atlas Fleet')
  await page.reload()
  await expect(page.getByLabel('Application Name', { exact: true })).toHaveValue('Atlas Fleet')
  const download = page.waitForEvent('download')
  await page.getByRole('button', { name: 'Download JSON', exact: true }).click()
  expect((await download).suggestedFilename()).toBe('manifest.webmanifest')
})

test('access policies protect the administrator and invitations stay explicit drafts', async ({
  page,
}) => {
  await settings(page)
  await page.getByRole('tab', { name: /Access & RBAC/ }).click()
  await expect(
    page.getByRole('checkbox', { name: 'Super Admin: System Settings', exact: true }),
  ).toBeDisabled()
  await page.getByRole('checkbox', { name: 'Auditor: Asset Config', exact: true }).check()
  await page.getByRole('button', { name: 'Invite Operator' }).click()
  await page.getByLabel('Operator Name', { exact: true }).fill('New Operator')
  await page.getByLabel('Work Email', { exact: true }).fill('operator@example.com')
  await page.getByRole('button', { name: 'Add Invitation Draft' }).click()
  await expect(page.getByRole('table').first()).toContainText('Invite Draft')
  await save(page)
  await page.reload()
  await expect(
    page.getByRole('checkbox', { name: 'Auditor: Asset Config', exact: true }),
  ).toBeChecked()
  await expect(page.getByRole('table').first()).toContainText('operator@example.com')
})

test('audio preferences can be staged, tested and saved', async ({ page }) => {
  await settings(page)
  await page.getByRole('tab', { name: 'Sound & Audio Alerts' }).click()
  await page.getByRole('switch', { name: 'Enable audio alerts', exact: true }).click()
  await page.getByRole('switch', { name: 'Scheduled audio mute', exact: true }).click()
  await page.getByRole('button', { name: 'Test tone Fuel Theft / Siphon', exact: true }).click()
  await expect(page.getByRole('alert')).toHaveCount(0)
  await save(page)
  await page.reload()
  await expect(
    page.getByRole('switch', { name: 'Enable audio alerts', exact: true }),
  ).toHaveAttribute('aria-checked', 'true')
  await expect(
    page.getByRole('switch', { name: 'Scheduled audio mute', exact: true }),
  ).toHaveAttribute('aria-checked', 'true')
})

test('leaving dirty settings keeps the draft when navigation is cancelled', async ({ page }) => {
  await settings(page)
  await page.getByRole('tab', { name: 'Branding & Web Assets' }).click()
  await page.getByLabel('Application Name', { exact: true }).fill('Unsaved Brand')
  await page
    .getByRole('navigation', { name: 'Main navigation' })
    .getByRole('link', { name: 'Dashboard', exact: true })
    .click()
  await page
    .getByRole('dialog', { name: 'Leave without saving?' })
    .getByRole('button', { name: 'Keep editing' })
    .click()
  await expect(page.getByLabel('Application Name', { exact: true })).toHaveValue('Unsaved Brand')
})
