import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests/e2e',
  timeout: 60_000,
  expect: { timeout: 25_000 },
  workers: 1,
  use: {
    baseURL: process.env.TEST_BASE_URL || 'http://127.0.0.1:5173',
    viewport: { width: 1440, height: 1000 },
    trace: 'retain-on-failure',
  },
  webServer: process.env.TEST_BASE_URL
    ? undefined
    : [
        {
          command: 'go run ./cmd/server',
          cwd: './backend',
          url: 'http://127.0.0.1:8080/api/health',
          reuseExistingServer: !process.env.CI,
          timeout: 120_000,
        },
        {
          command: 'npm run dev --workspace frontend',
          url: 'http://127.0.0.1:5173',
          reuseExistingServer: !process.env.CI,
          env: { VITE_API_URL: 'http://127.0.0.1:8080', VITE_WS_URL: 'ws://127.0.0.1:8080' },
        },
      ],
});
