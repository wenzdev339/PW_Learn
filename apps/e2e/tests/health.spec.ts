import { test, expect } from '@playwright/test';

test('health check returns ok', async ({ request }) => {
  const res = await request.get('/health');
  expect(res.status()).toBe(200);
  const body = await res.json();
  expect(body.data.status).toBe('ok');
});
