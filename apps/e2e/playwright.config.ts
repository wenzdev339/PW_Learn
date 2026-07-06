import { defineConfig } from '@playwright/test';

export default defineConfig({
  testDir: './tests',
  fullyParallel: true,
  reporter: [['html', { open: 'never' }], ['list']],
  use: {
    baseURL: process.env.API_BASE_URL ?? 'http://localhost:4000',
    extraHTTPHeaders: {
      'Content-Type': 'application/json',
    },
    trace: 'retain-on-failure',
  },
});
