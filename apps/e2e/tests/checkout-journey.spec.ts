import { test, expect } from '@playwright/test';

test('full journey: register, add to cart, checkout, and pay', async ({ request }) => {
  const email = `journey-${Date.now()}@example.com`;
  const registerRes = await request.post('/api/v1/auth/register', {
    data: { email, password: 'Password123!', name: 'Journey' },
  });
  const registerBody = await registerRes.json();
  const accessToken = registerBody.data.accessToken;
  const authHeaders = { Authorization: `Bearer ${accessToken}` };

  const productsRes = await request.get('/api/v1/products?pageSize=1');
  const productsBody = await productsRes.json();
  const product = productsBody.data.items[0];

  const addToCartRes = await request.post('/api/v1/cart/items', {
    headers: authHeaders,
    data: { productId: product.id, quantity: 1 },
  });
  expect(addToCartRes.status()).toBe(201);

  const addressRes = await request.post('/api/v1/addresses', {
    headers: authHeaders,
    data: { label: 'Home', line1: '1 Test Rd', city: 'Bangkok', postalCode: '10110', country: 'TH' },
  });
  const addressBody = await addressRes.json();

  const checkoutRes = await request.post('/api/v1/checkout', {
    headers: authHeaders,
    data: { shippingAddressId: addressBody.data.id },
  });
  expect(checkoutRes.status()).toBe(201);
  const checkoutBody = await checkoutRes.json();
  expect(checkoutBody.data.status).toBe('PENDING');

  const payRes = await request.post('/api/v1/payments/mock', {
    headers: authHeaders,
    data: { orderId: checkoutBody.data.id, cardNumber: '4242424242424242' },
  });
  expect(payRes.status()).toBe(200);

  const finalOrderRes = await request.get(`/api/v1/orders/${checkoutBody.data.id}`, { headers: authHeaders });
  const finalOrderBody = await finalOrderRes.json();
  expect(finalOrderBody.data.status).toBe('PAID');
});
