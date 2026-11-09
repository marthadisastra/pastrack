import { defineConfig, devices } from '@playwright/test'
export default defineConfig({
  testDir: './tests',
  testMatch: 'dashboard-contract.spec.js',
  fullyParallel: false,
  use: { baseURL: 'http://127.0.0.1:5187', trace: 'retain-on-failure' },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: {
    command: 'npm run dev -- --port 5187 --strictPort',
    env: {
      VITE_AUTH_MODE: 'demo',
      VITE_DASHBOARD_MODE: 'server',
      VITE_DASHBOARD_TRANSPORT: 'snapshot',
      VITE_USE_MOCK: 'true',
    },
    url: 'http://127.0.0.1:5187',
    reuseExistingServer: false,
  },
})
