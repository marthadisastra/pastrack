// Isolated API/SSE fixture: validates the non-demo transport without contacting a live backend.
import http from 'node:http'
import { spawn } from 'node:child_process'
import { once } from 'node:events'
import { setTimeout as delay } from 'node:timers/promises'
import { fileURLToPath } from 'node:url'
import { chromium, expect } from '@playwright/test'
import { vehicles, workOrders } from '../src/mocks/fleet.js'
const root = fileURLToPath(new URL('../', import.meta.url))
const clients = new Set()
let connections = 0,
  browser,
  vite
const server = http.createServer((request, response) => {
  response.setHeader('Access-Control-Allow-Origin', 'http://127.0.0.1:5183')
  response.setHeader('Access-Control-Allow-Credentials', 'true')
  if (request.url.startsWith('/api/realtime')) {
    connections++
    response.writeHead(200, { 'Content-Type': 'text/event-stream', 'Cache-Control': 'no-cache' })
    response.write('event: ready\ndata: {}\n\n')
    clients.add(response)
    request.on('close', () => clients.delete(response))
    return
  }
  response.setHeader('Content-Type', 'application/json')
  if (request.url === '/api/vehicles') return response.end(JSON.stringify(vehicles))
  if (request.url === '/api/maintenance') return response.end(JSON.stringify(workOrders))
  response.writeHead(404)
  response.end(JSON.stringify({ message: 'Fixture route not found' }))
})
const send = (event, data) => {
  for (const client of clients) client.write(`event: ${event}\ndata: ${JSON.stringify(data)}\n\n`)
}
try {
  server.listen(0, '127.0.0.1')
  await once(server, 'listening')
  const port = server.address().port
  vite = spawn(
    process.execPath,
    ['node_modules/vite/bin/vite.js', '--host', '127.0.0.1', '--port', '5183', '--strictPort'],
    {
      cwd: root,
      env: {
        ...process.env,
        VITE_USE_MOCK: 'false',
        VITE_AUTH_MODE: 'demo',
        VITE_API_BASE_URL: `http://127.0.0.1:${port}/api`,
        VITE_REALTIME_URL: `http://127.0.0.1:${port}/api/realtime`,
      },
      stdio: 'pipe',
    },
  )
  let log = ''
  vite.stdout.on('data', (chunk) => {
    log += chunk
  })
  vite.stderr.on('data', (chunk) => {
    log += chunk
  })
  let ready = false
  for (let index = 0; index < 150; index++) {
    if (vite.exitCode !== null) throw new Error(`Fixture Vite exited: ${log}`)
    try {
      const response = await fetch('http://127.0.0.1:5183/')
      if (response.ok) {
        ready = true
        break
      }
    } catch {
      /* Wait for local server startup. */
    }
    await delay(100)
  }
  if (!ready) throw new Error(`Fixture Vite did not start: ${log}`)
  browser = await chromium.launch()
  const context = await browser.newContext()
  const page = await context.newPage()
  const errors = []
  page.on('pageerror', (error) => errors.push(error.message))
  await page.goto('http://127.0.0.1:5183/#/dashboard')
  await expect(page.getByText('Live', { exact: true })).toBeVisible()
  const row = page.locator('tbody tr').filter({ hasText: 'HAULER-01' })
  await expect(row).toContainText('342.0 L')
  send('vehicle', { id: 'HAULER-01', fuel: 331.2, speed: 72, updatedAt: new Date().toISOString() })
  await expect(row).toContainText('331.2 L')
  send('values', {
    'fleet/live': { upserts: { 'HAULER-01': { fuel_liters: 330.8, speed_kmh: 74 } }, removed: [] },
  })
  await expect(row).toContainText('330.8 L')
  await expect(row).toContainText('B 9123 XYZ')
  await expect(row).toContainText('74 km/h')
  send('notification', {
    id: 'server-warning-1',
    title: 'Server telemetry warning',
    message: 'An SSE notification arrived.',
    severity: 'warning',
    target: '/tracking?unit=HAULER-01',
  })
  await page.getByRole('button', { name: 'Notifications', exact: true }).click()
  await expect(page.getByRole('dialog', { name: 'Notifications panel' })).toContainText(
    'Server telemetry warning',
  )
  await page.keyboard.press('Escape')
  await page.getByRole('button', { name: 'Pause live updates' }).click()
  await expect(page.getByText('Values frozen. Resume to synchronize.')).toBeVisible()
  await page.getByRole('button', { name: 'Resume live updates' }).click()
  await expect(page.getByText('Live', { exact: true })).toBeVisible()
  await context.setOffline(true)
  await expect(
    page.getByText('Network unavailable. Last received values are retained.'),
  ).toBeVisible()
  await expect(row).toContainText('330.8 L')
  await context.setOffline(false)
  await expect(page.getByText('Live', { exact: true })).toBeVisible()
  send('vehicle', { id: 'HAULER-01', fuel: 329.5, updatedAt: new Date().toISOString() })
  await expect(row).toContainText('329.5 L')
  const before = connections
  for (const client of clients) client.destroy()
  await expect.poll(() => connections).toBeGreaterThan(before)
  await expect(page.getByText('Live', { exact: true })).toBeVisible()
  if (errors.length) throw new Error(errors.join('\n'))
  console.log(
    'PASS: normalized SSE events, Go field deltas, retained fields, notification delivery, pause/resume, offline retention, reconnection, and no browser errors.',
  )
} finally {
  await browser?.close()
  for (const client of clients) client.destroy()
  server.closeAllConnections()
  await new Promise((resolve) => server.close(resolve))
  if (vite && vite.exitCode === null) {
    vite.kill('SIGTERM')
    await once(vite, 'exit')
  }
}
