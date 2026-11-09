import { chromium } from '@playwright/test'
import { spawn } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import assert from 'node:assert/strict'
const cwd = fileURLToPath(new URL('../', import.meta.url))
const origin = 'http://127.0.0.1:5186'
const server = spawn(
  process.execPath,
  ['node_modules/vite/bin/vite.js', '--host', '127.0.0.1', '--port', '5186', '--strictPort'],
  {
    cwd,
    env: {
      ...process.env,
      VITE_AUTH_MODE: 'server',
      AUTH_PROXY_TARGET: process.env.AUTH_BACKEND_URL,
    },
    stdio: 'ignore',
  },
)
let browser, page
try {
  for (let i = 0; i < 100; i++) {
    try {
      if ((await fetch(origin)).ok) break
    } catch {}
    await new Promise((r) => setTimeout(r, 100))
  }
  browser = await chromium.launch()
  page = await browser.newPage()
  const errors = []
  page.on('pageerror', (e) => errors.push(e.message))
  await page.goto(origin + '/#/settings?tab=smtp')
  await page.waitForURL('**/#/login?redirect=**')
  await page.getByLabel('Email or Username').fill(process.env.AUTH_TEST_EMAIL)
  await page.getByLabel('Password', { exact: true }).fill(process.env.AUTH_TEST_PASSWORD)
  await page.getByRole('button', { name: 'Sign In to Pastrack', exact: true }).click()
  await page.waitForURL('**/#/settings?tab=smtp')
  await page.getByRole('heading', { name: /SMTP & Password Recovery Email/ }).waitFor()
  await page.reload()
  await page.getByRole('heading', { name: /SMTP & Password Recovery Email/ }).waitFor()
  await page.goto(origin + '/#/forgot-password')
  await page.getByLabel('Work Email Address').fill(process.env.AUTH_TEST_EMAIL)
  await page.getByRole('button', { name: /Send Reset Link/i }).click()
  await page.getByText(/Check your inbox/i).waitFor()
  const result = await (await fetch(process.env.AUTH_BACKEND_URL + '/__test/mail')).json()
  assert.ok(result.link.includes('/#/reset-password?token='))
  await page.goto(origin + '/' + new URL(result.link).hash)
  const fields = page.locator('input[autocomplete="new-password"]')
  await fields.nth(0).fill(process.env.AUTH_TEST_PASSWORD)
  await fields.nth(1).fill(process.env.AUTH_TEST_PASSWORD)
  await page.getByRole('button', { name: 'Update Password' }).click()
  await page
    .getByText(/Your password has been updated/i)
    .first()
    .waitFor()
  assert.equal((await page.request.get(origin + '/api/v1/auth/me')).status(), 401)
  await page.goto(origin + '/#/login')
  await page.getByLabel('Email or Username').fill(process.env.AUTH_TEST_EMAIL)
  await page.getByLabel('Password', { exact: true }).fill(process.env.AUTH_TEST_PASSWORD)
  await page.getByRole('button', { name: 'Sign In to Pastrack', exact: true }).click()
  await page.waitForURL('**/#/dashboard')
  await page.goto(origin + '/#/settings?tab=smtp')
  await page.getByRole('heading', { name: /SMTP & Password Recovery Email/ }).waitFor()
  await page.getByLabel('Sender Name').fill('QA Browser')
  await page.getByRole('button', { name: 'Save SMTP Settings' }).click()
  await page.getByText('SMTP settings saved securely on the server.').waitFor()
  await page.getByRole('button', { name: /Sign out/i }).click()
  await page.waitForURL('**/#/login')
  assert.equal((await page.request.get(origin + '/api/v1/auth/me')).status(), 401)
  assert.deepEqual(errors, [])
  console.log(
    'AUTH_BROWSER_OK: session reload, SMTP tab, forgot, reset, revoked sessions, login, SMTP save, logout',
  )
} catch (error) {
  console.error((await page?.locator('body').innerText())?.slice(-5000))
  throw error
} finally {
  await browser?.close()
  server.kill('SIGTERM')
}
