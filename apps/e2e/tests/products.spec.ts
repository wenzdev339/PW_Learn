import { test, expect } from '@playwright/test';

test.describe('products', () => {
  test('lists seeded products with pagination', async ({ request }) => {
    const res = await request.get('/api/v1/products');
    expect(res.status()).toBe(200);
    const body = await res.json();

    // ข้อมูลสินค้าถูก seed ไว้อย่างน้อย 20 ชิ้น (ดู README) เลยเช็คแบบ ">=" แทนที่จะ
    // hardcode ตัวเลขตายตัว เพื่อกันเทสพังถ้าในอนาคต seed เพิ่มสินค้ามากขึ้น
    expect(body.data.total).toBeGreaterThanOrEqual(20);
    expect(body.data.items.length).toBeGreaterThan(0);
  });

  test('filters by category', async ({ request }) => {
    // ส่ง query string แนบไปกับ URL ได้เลย ไม่ต้องมี object parameter แยก
    const res = await request.get('/api/v1/products?category=books');
    expect(res.status()).toBe(200);
    const body = await res.json();

    // วนเช็คทุก item ที่ได้กลับมาว่าต้องอยู่ใน category "books" ทั้งหมด
    // (พิสูจน์ว่า filter ทำงานถูกต้องจริง ไม่ใช่แค่เช็คว่า "ยิงแล้วไม่ error")
    for (const item of body.data.items) {
      expect(item.category.slug).toBe('books');
    }
  });
});
