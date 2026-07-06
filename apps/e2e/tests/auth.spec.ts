import { test, expect } from '@playwright/test';

test.describe('auth', () => {
  test('registers a new user and returns an access token', async ({ request }) => {
    const email = `user-${Date.now()}@example.com`;
    const res = await request.post('/api/v1/auth/register', {
      data: { email, password: 'Password123!', name: 'Playwright User' },
    });
    expect(res.status()).toBe(201);
    const body = await res.json();
    expect(body.data.user.email).toBe(email);
    expect(typeof body.data.accessToken).toBe('string');
  });

  test('logs in with the seeded customer account', async ({ request }) => {
    const res = await request.post('/api/v1/auth/login', {
      data: { email: 'customer@example.com', password: 'Customer123!' },
    });
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(typeof body.data.accessToken).toBe('string');
  });

  test('rejects wrong password', async ({ request }) => {
    const res = await request.post('/api/v1/auth/login', {
      data: { email: 'customer@example.com', password: 'WrongPassword!' },
    });
    expect(res.status()).toBe(401);
  });
});
