import { test, expect } from '@playwright/test'

test('notifications have unread state, history and a contextual unit action', async ({ page }) => {
  await page.goto('/#/dashboard')
  await expect(page.getByRole('heading', { name: 'Operations Telematics Dashboard' })).toBeVisible()
  const bell = page.getByRole('button', { name: 'Notifications', exact: true })
  await bell.click()
  const panel = page.getByRole('dialog', { name: 'Notifications panel' })
  await expect(panel).toContainText('Fuel siphon detected · HAULER-04')
  await panel.getByRole('button', { name: 'Mark all read' }).click()
  await panel.getByRole('button', { name: 'unread', exact: true }).click()
  await expect(panel).toContainText('You are all caught up')
  // Reading must not clear operational incidents in the sidebar.
  await expect(
    page.getByRole('navigation').getByRole('link', { name: 'Alert Center', exact: true }),
  ).toContainText('1')
  await page.keyboard.press('Escape')
  await expect(bell).toBeFocused()
  await bell.click()
  await panel.getByRole('button', { name: 'all', exact: true }).click()
  await panel.getByRole('button', { name: /Fuel siphon detected/ }).click()
  await expect(page).toHaveURL(/tracking\?unit=HAULER-04/)
  await expect(page.locator('.tracking-hud')).toContainText('HAULER-04')
  await page.reload()
  await page.getByRole('button', { name: 'Notifications', exact: true }).click()
  await expect(page.getByRole('dialog', { name: 'Notifications panel' })).toContainText('0 unread')
})

test('live updates retain focus and row identity, pause and recover after offline', async ({
  page,
  context,
}) => {
  await page.clock.install()
  await page.goto('/#/dashboard')
  const row = page.locator('tbody tr').filter({ hasText: 'HAULER-01' })
  await expect(row).toBeVisible()
  await row.evaluate((el) => {
    el.dataset.retained = 'yes'
  })
  const fuel = row.locator('td').nth(3).locator('.live-value').first()
  const before = await fuel.innerText()
  await page.getByLabel('Filter vehicle or IMEI').focus()
  await page.clock.runFor(4300)
  await expect(fuel).not.toHaveText(before)
  await expect(row).toHaveAttribute('data-retained', 'yes')
  await expect(page.getByLabel('Filter vehicle or IMEI')).toBeFocused()
  await page.getByRole('button', { name: 'Pause live updates', exact: true }).click()
  const frozen = await fuel.innerText()
  await page.clock.runFor(12000)
  await expect(fuel).toHaveText(frozen)
  await page.getByRole('button', { name: 'Resume live updates', exact: true }).click()
  await page.clock.runFor(4300)
  await expect(fuel).not.toHaveText(frozen)
  const retained = await fuel.innerText()
  await context.setOffline(true)
  await expect(
    page.getByText('Network unavailable. Last received values are retained.'),
  ).toBeVisible()
  await page.clock.runFor(5000)
  await expect(fuel).toHaveText(retained)
  await context.setOffline(false)
  await expect(page.getByText('Demo live', { exact: true })).toBeVisible()
  await page.clock.runFor(4300)
  await expect(fuel).not.toHaveText(retained)
})

test('map image failure has a retry and a working vector alternative', async ({ page }) => {
  let failImage = true
  await page.route('**/src/assets/fleet-map.jpg', (route) =>
    failImage ? route.abort() : route.continue(),
  )
  await page.goto('/#/dashboard')
  await expect(page.getByText('Map image unavailable. Use vector view or retry.')).toBeVisible()
  failImage = false
  await page.getByRole('button', { name: 'Retry image' }).click()
  await expect(page.getByAltText('Satellite map illustration from the fleet mockup')).toHaveClass(
    /opacity-100/,
  )
  await page.getByRole('button', { name: 'Vector', exact: true }).click()
  await expect(page.getByAltText('Satellite map illustration from the fleet mockup')).toHaveCount(0)
  await expect(page.getByRole('button', { name: 'Select HAULER-01', exact: true })).toBeVisible()
})

test('unsaved input is protected and the user can keep editing', async ({ page }) => {
  await page.goto('/#/vehicles')
  await page.getByRole('button', { name: 'Add Asset', exact: true }).click()
  await page.getByLabel('Asset ID *', { exact: true }).fill('UNSAVED-01')
  await page.getByRole('button', { name: 'Cancel', exact: true }).click()
  await expect(page.getByText('Discard unsaved changes?', { exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'Keep editing', exact: true }).click()
  await expect(page.getByLabel('Asset ID *', { exact: true })).toHaveValue('UNSAVED-01')
  await page.getByRole('button', { name: 'Cancel', exact: true }).click()
  await page.getByRole('button', { name: 'Discard changes', exact: true }).click()
  await expect(page.getByRole('dialog')).toHaveCount(0)
  await expect(page.locator('tbody')).not.toContainText('UNSAVED-01')
})

test('realtime changes update the open inspection without replacing its focused button', async ({
  page,
}) => {
  await page.clock.install()
  await page.goto('/#/dashboard')
  await page
    .locator('tbody tr')
    .filter({ hasText: 'HAULER-01' })
    .getByRole('button', { name: 'Inspect', exact: true })
    .click()
  const dialog = page.getByRole('dialog', { name: 'Asset Inspection · HAULER-01' })
  await expect(dialog.getByText('342.0', { exact: true })).toBeVisible()
  await dialog.getByRole('button', { name: 'Close Inspection' }).focus()
  await page.clock.runFor(4300)
  await expect(dialog.getByText('342.0', { exact: true })).toHaveCount(0)
  await expect(dialog.getByRole('button', { name: 'Close Inspection' })).toBeFocused()
})

test('browser back navigation asks before discarding an edited form', async ({ page }) => {
  await page.goto('/#/dashboard')
  await expect(page.locator('h1')).toBeVisible()
  await page.getByRole('navigation').getByRole('link', { name: 'All Assets', exact: true }).click()
  await page.getByRole('button', { name: 'Add Asset', exact: true }).click()
  await page.getByLabel('Asset ID *', { exact: true }).fill('KEEP-MY-INPUT')
  await page.evaluate(() => history.back())
  const confirmation = page.getByRole('dialog', { name: 'Leave without saving?' })
  await expect(confirmation).toBeVisible()
  await confirmation.getByRole('button', { name: 'Keep editing' }).click()
  await expect(page.getByLabel('Asset ID *', { exact: true })).toHaveValue('KEEP-MY-INPUT')
  await expect(page).toHaveURL(/#\/vehicles$/)
})
