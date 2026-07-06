import { test, expect } from '@playwright/test';

test.describe('network condition simulation', () => {
  test('X-Test-Scenario: error forces a 500 response', async ({ request }) => {
    const res = await request.get('/health', {
      headers: { 'X-Test-Scenario': 'error' },
    });
    expect(res.status()).toBe(500);
    const body = await res.json();
    expect(body.error.code).toBe('SIMULATED_ERROR');
  });

  test('X-Test-Scenario: flaky:2 fails twice then succeeds', async ({ request }) => {
    const runId = `flaky-${Date.now()}`;
    const headers = { 'X-Test-Scenario': 'flaky:2', 'X-Test-Run-Id': runId };

    const first = await request.get('/health', { headers });
    const second = await request.get('/health', { headers });
    const third = await request.get('/health', { headers });

    expect(first.status()).toBe(500);
    expect(second.status()).toBe(500);
    expect(third.status()).toBe(200);
  });
});
