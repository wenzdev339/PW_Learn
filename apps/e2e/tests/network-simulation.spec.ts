import { test, expect } from '@playwright/test';

// เทสกลุ่มนี้ไม่ได้เทส business logic ปกติ แต่เทส "ความสามารถจำลองปัญหาเครือข่าย"
// ของ backend เอง ผ่าน header พิเศษ X-Test-Scenario
// (ดูโค้ดจริงที่ apps/backend/internal/middleware/test_scenario.go)
test.describe('network condition simulation', () => {
  test('X-Test-Scenario: error forces a 500 response', async ({ request }) => {
    // แค่แนบ header นี้ไป backend ก็จะบังคับตอบ error กลับมาทันที
    // โดยไม่ต้องรอให้ backend พังขึ้นมาเองจริงๆ (ซึ่งควบคุมเวลาไม่ได้ และทำให้เทส flaky)
    const res = await request.get('/health', {
      headers: { 'X-Test-Scenario': 'error' },
    });
    expect(res.status()).toBe(500);
    const body = await res.json();
    expect(body.error.code).toBe('SIMULATED_ERROR');
  });

  test('X-Test-Scenario: flaky:2 fails twice then succeeds', async ({ request }) => {
    // X-Test-Run-Id ต้องไม่ซ้ำกับเทสอื่น เพราะ backend ใช้ค่านี้เป็น key
    // นับจำนวนครั้งที่ยิงมาแล้ว ถ้าใช้ id ซ้ำกับเทสอื่นจะนับปนกันจนผลลัพธ์ผิด
    const runId = `flaky-${Date.now()}`;
    const headers = { 'X-Test-Scenario': 'flaky:2', 'X-Test-Run-Id': runId };

    // "flaky:2" แปลว่า 2 ครั้งแรกที่ยิง (ด้วย runId เดียวกัน) ต้อง fail แล้วครั้งที่ 3
    // เป็นต้นไปต้องผ่าน — ยิง 3 ครั้งติดกันเพื่อพิสูจน์ pattern นี้ตรงตัว
    const first = await request.get('/health', { headers });
    const second = await request.get('/health', { headers });
    const third = await request.get('/health', { headers });

    expect(first.status()).toBe(500);
    expect(second.status()).toBe(500);
    expect(third.status()).toBe(200); // ครั้งที่ 3 = เกิน 2 ครั้งที่กำหนดไว้แล้ว ต้องผ่าน
  });
});
