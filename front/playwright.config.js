import { defineConfig, devices } from '@playwright/test'
export default defineConfig({
  testDir: './tests',
  testIgnore: 'dashboard-contract.spec.js',
  fullyParallel: false,
  use: { baseURL: 'http://127.0.0.1:5175', trace: 'retain-on-failure' },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: {
    command: 'npm run dev -- --port 5175 --strictPort',
    env: { VITE_AUTH_MODE: 'demo', VITE_DASHBOARD_MODE: 'demo' },
    url: 'http://127.0.0.1:5175',
    reuseExistingServer: !process.env.CI,
  },
})
