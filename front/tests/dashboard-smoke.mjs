import { chromium } from '@playwright/test'
import { spawn } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import assert from 'node:assert/strict'
const cwd = fileURLToPath(new URL('../', import.meta.url)),
  origin = 'http://127.0.0.1:5188'
const server = spawn(
  process.execPath,
  ['node_modules/vite/bin/vite.js', '--host', '127.0.0.1', '--port', '5188', '--strictPort'],
  {
    cwd,
    env: {
      ...process.env,
      VITE_AUTH_MODE: 'server',
      VITE_DASHBOARD_MODE: 'server',
      VITE_DASHBOARD_TRANSPORT: process.env.DASHBOARD_TEST_TRANSPORT || 'snapshot',
      VITE_USE_MOCK: 'true',
      AUTH_PROXY_TARGET: process.env.AUTH_BACKEND_URL,
    },
    stdio: 'ignore',
  },
)
let browser
try {
  for (let i = 0; i < 100; i++) {
    try {
      if ((await fetch(origin)).ok) break
    } catch {}
    await new Promise((r) => setTimeout(r, 100))
  }
  browser = await chromium.launch()
  const page = await browser.newPage()
  page.setDefaultTimeout(15000)
  const errors = [],
    business = []
  page.on('pageerror', (e) => errors.push(e.message))
  page.on('request', (r) => {
    if (r.url().includes('/api/v1/') && !r.url().includes('/auth/'))
      business.push(new URL(r.url()).pathname)
  })
  await page.goto(origin + '/#/dashboard')
  await page.waitForURL('**/#/login?redirect=**')
  await page.getByLabel('Email or Username').fill(process.env.AUTH_TEST_USERNAME)
  await page.getByLabel('Password', { exact: true }).fill(process.env.AUTH_TEST_PASSWORD)
  await page.getByRole('button', { name: 'Sign In to Pastrack', exact: true }).click()
  await page.waitForURL('**/#/dashboard')
  await page.getByText('DATABASE CONNECTED', { exact: true }).waitFor()
  await page.getByLabel('Filter vehicle or gateway').fill(process.env.DASHBOARD_TEST_PLATE)
  const row = page.getByRole('row').filter({ hasText: process.env.DASHBOARD_TEST_PLATE })
  await row.getByRole('button', { name: 'Inspect', exact: true }).waitFor()
  assert.match(await row.innerText(), /UNASSIGNED/)
  assert.match(await row.innerText(), /UNKNOWN/)
  await row.getByRole('button', { name: 'Inspect', exact: true }).click()
  await page.getByRole('dialog').waitFor()
  assert.match(await page.getByRole('dialog').innerText(), /Coordinates[\s\S]*—, —/)
  await page.getByRole('button', { name: 'Close Inspection', exact: true }).click()
  await new Promise((r) => setTimeout(r, 4500))
  assert.deepEqual(business, ['/api/v1/dashboard/snapshot'])
  const response = page.waitForResponse((r) => r.url().includes('/dashboard/snapshot'))
  await page.getByRole('button', { name: 'Refresh dashboard', exact: true }).click()
  assert.equal((await response).status(), 200)
  assert.deepEqual(business, ['/api/v1/dashboard/snapshot', '/api/v1/dashboard/snapshot'])
  assert.equal(await page.getByText('DEMO VECTOR', { exact: true }).count(), 0)
  assert.equal(await page.getByText(/NaN|Infinity/).count(), 0)
  assert.deepEqual(errors, [])
  await page.setViewportSize({ width: 390, height: 844 })
  assert.ok(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth))
  await page.setViewportSize({ width: 1280, height: 900 })
  await page.getByRole('button', { name: 'Sign out', exact: true }).click()
  await page.waitForURL('**/#/login')
  console.log(
    'DASHBOARD_DB_BROWSER_OK: existing Go login, SQL snapshot, nulls/states, inspection, explicit refresh, no demo/polling business requests, mobile, logout',
  )
} finally {
  await browser?.close()
  server.kill('SIGTERM')
}
