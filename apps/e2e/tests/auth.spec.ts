import { test, expect } from '@playwright/test';

// test.describe ใช้จัดกลุ่มเทสที่เกี่ยวข้องกันไว้ด้วยกัน (แค่จัดหมวดหมู่ใน report
// ไม่ได้มีผลกับการรันเทส)
test.describe('auth', () => {
  test('registers a new user and returns an access token', async ({ request }) => {
    // ต่อ Date.now() ท้าย email เพื่อให้ได้ email ใหม่ไม่ซ้ำทุกครั้งที่รันเทส
    // เพราะ backend กัน email ซ้ำ ถ้า hardcode email ตายตัว รันเทสรอบสองจะพัง
    const email = `user-${Date.now()}@example.com`;

    // ยิง POST พร้อมแนบ JSON body ผ่าน key "data"
    const res = await request.post('/api/v1/auth/register', {
      data: { email, password: 'Password123!', name: 'Playwright User' },
    });

    // 201 = Created (สร้างของใหม่สำเร็จ) ต่างจาก 200 ที่มักใช้กับ "อ่านข้อมูลสำเร็จ"
    expect(res.status()).toBe(201);
    const body = await res.json();

    // เช็คว่า email ที่ backend ตอบกลับมา ตรงกับที่เราส่งไปจริง
    expect(body.data.user.email).toBe(email);

    // เช็คแค่ "ชนิดข้อมูล" ว่าต้องเป็น string ก็พอ เพราะค่า token จริงเปลี่ยนทุกครั้งที่รัน
    expect(typeof body.data.accessToken).toBe('string');
  });

  test('logs in with the seeded customer account', async ({ request }) => {
    // ใช้บัญชีที่ seed ไว้ล่วงหน้าแล้ว (ดู `go run ./cmd/seed` ใน README)
    // ไม่ต้อง register ใหม่ เพราะบัญชีนี้มีอยู่ในฐานข้อมูลเสมอ
    const res = await request.post('/api/v1/auth/login', {
      data: { email: 'customer@example.com', password: 'Customer123!' },
    });
    // login สำเร็จ = 200 (ไม่ใช่ 201 เพราะไม่ได้ "สร้าง" อะไรใหม่ แค่ยืนยันตัวตน)
    expect(res.status()).toBe(200);
    const body = await res.json();
    expect(typeof body.data.accessToken).toBe('string');
  });

  test('rejects wrong password', async ({ request }) => {
    // เทส negative case: ต้องพิสูจน์ว่า "fail แบบถูกต้อง" ด้วย ไม่ใช่เทสแต่ happy path
    const res = await request.post('/api/v1/auth/login', {
      data: { email: 'customer@example.com', password: 'WrongPassword!' },
    });
    // 401 Unauthorized = รหัสผ่านผิด/ยืนยันตัวตนไม่ผ่าน
    expect(res.status()).toBe(401);
  });
});
