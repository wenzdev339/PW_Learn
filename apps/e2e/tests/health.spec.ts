import { test, expect } from '@playwright/test';

// เทสที่ง่ายที่สุดในโปรเจค: เช็คว่า backend ยังมีชีวิตอยู่ (health check)
test('health check returns ok', async ({ request }) => {
  // fixture "request" คือตัวยิง HTTP ตรงๆ โดยไม่ต้องเปิดเบราว์เซอร์เลย
  // ใช้ path แบบ relative ได้เพราะ baseURL ถูกตั้งไว้แล้วใน playwright.config.ts
  const res = await request.get('/health');

  // เช็ค HTTP status code ต้องเป็น 200 (สำเร็จ)
  expect(res.status()).toBe(200);

  // แปลง response body จาก JSON string ให้เป็น object ก่อน ถึงจะเช็คเนื้อหาข้างในได้
  const body = await res.json();
  expect(body.data.status).toBe('ok');
});
