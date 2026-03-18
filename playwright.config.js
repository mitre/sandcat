// @ts-check
const { defineConfig, devices } = require('@playwright/test');

const CALDERA_URL = process.env.CALDERA_URL || 'http://localhost:8888';

// Credentials: require explicit env vars in CI; fall back to local dev defaults only
// when not running in CI to avoid accidentally committing insecure defaults.
const calderaUser = process.env.CALDERA_USER || (process.env.CI ? undefined : 'admin');
const calderaPass = process.env.CALDERA_PASS || (process.env.CI ? undefined : 'admin');

module.exports = defineConfig({
  testDir: './tests/e2e',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: [['html', { open: 'never' }], ['list']],
  timeout: 60_000,
  expect: { timeout: 15000 },
  use: {
    baseURL: CALDERA_URL,
    trace: 'on-first-retry',
    screenshot: 'only-on-failure',
    headless: true,
    httpCredentials: {
      username: calderaUser,
      password: calderaPass,
    },
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
});
