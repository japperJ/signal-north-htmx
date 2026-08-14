import { defineConfig } from '@playwright/test';

const externalBaseURL = process.env.BASE_URL;
const baseURL = externalBaseURL ?? 'http://127.0.0.1:18080';

export default defineConfig({
  testDir: './tests/browser',
  timeout: 20_000,
  expect: { timeout: 8_000 },
  retries: process.env.CI ? 2 : 0,
  reporter: [['list'], ['html', { open: 'never' }]],
  use: {
    baseURL,
    headless: true,
    screenshot: 'only-on-failure',
    trace: 'on-first-retry',
  },
  webServer: externalBaseURL ? undefined : {
    command: 'go run ./cmd/server',
    url: baseURL,
    env: { ...process.env, PORT: '18080' },
    reuseExistingServer: false,
    timeout: 120_000,
  },
});
