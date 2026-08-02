// playwright.config.ts — Playwright configuration for public MVP smoke tests.
import { defineConfig, devices } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  outputDir: './test-results',
  retries: 0,
  use: {
    baseURL: 'http://localhost:5173',
    screenshot: 'on',
    trace: 'off',
  },
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
  ],
  webServer: {
    command: 'npm run dev',
    url: 'http://localhost:5173',
    reuseExistingServer: false,
  },
});
