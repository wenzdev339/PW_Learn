import { test, expect } from '@playwright/test';

test.describe('products', () => {
  test('lists seeded products with pagination', async ({ request }) => {
    const res = await request.get('/api/v1/products');
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(body.data.total).toBeGreaterThanOrEqual(20);
    expect(body.data.items.length).toBeGreaterThan(0);
  });

  test('filters by category', async ({ request }) => {
    const res = await request.get('/api/v1/products?category=books');
    expect(res.status()).toBe(200);
    const body = await res.json();
    for (const item of body.data.items) {
      expect(item.category.slug).toBe('books');
    }
  });
});
