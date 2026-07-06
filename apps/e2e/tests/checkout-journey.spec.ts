import { test, expect } from '@playwright/test';

// เทสนี้ไม่แยกเป็นหลาย test() ย่อย เพราะแต่ละขั้นตอนต้องพึ่งผลลัพธ์จากขั้นก่อนหน้า
// (เช่น ต้องมี accessToken ก่อน ถึงจะเรียก add-to-cart ได้) เลยรวมเป็นเทสยาวเทสเดียว
// ที่จำลอง "user เดินเรื่องซื้อของทั้ง flow" ตั้งแต่ต้นจนจบ
test('full journey: register, add to cart, checkout, and pay', async ({ request }) => {
  // ขั้น 1: สมัครสมาชิกใหม่ เพื่อเอา accessToken มาใช้ยืนยันตัวตนในขั้นถัดๆ ไป
  const email = `journey-${Date.now()}@example.com`;
  const registerRes = await request.post('/api/v1/auth/register', {
    data: { email, password: 'Password123!', name: 'Journey' },
  });
  const registerBody = await registerRes.json();
  const accessToken = registerBody.data.accessToken;

  // เตรียม header ไว้ใช้ซ้ำกับทุก request ถัดจากนี้ที่ต้อง login ก่อน
  const authHeaders = { Authorization: `Bearer ${accessToken}` };

  // ขั้น 2: ดึงสินค้ามา 1 ชิ้น (pageSize=1) เอาแค่ id ไปใส่ตะกร้า ไม่สนว่าเป็นชิ้นไหน
  const productsRes = await request.get('/api/v1/products?pageSize=1');
  const productsBody = await productsRes.json();
  const product = productsBody.data.items[0];

  // ขั้น 3: เพิ่มสินค้าลงตะกร้า — endpoint นี้ต้อง login เลยต้องแนบ authHeaders ไปด้วย
  const addToCartRes = await request.post('/api/v1/cart/items', {
    headers: authHeaders,
    data: { productId: product.id, quantity: 1 },
  });
  expect(addToCartRes.status()).toBe(201);

  // ขั้น 4: สร้างที่อยู่จัดส่ง เพราะขั้น checkout ต่อไปต้องใช้ shippingAddressId
  const addressRes = await request.post('/api/v1/addresses', {
    headers: authHeaders,
    data: { label: 'Home', line1: '1 Test Rd', city: 'Bangkok', postalCode: '10110', country: 'TH' },
  });
  const addressBody = await addressRes.json();

  // ขั้น 5: checkout ตะกร้า — backend จะตัด stock, คำนวณราคารวม, สร้าง order
  // สถานะ PENDING ทั้งหมดนี้ทำในทรานแซคชันเดียว (ดู README หัวข้อ sequence diagram)
  const checkoutRes = await request.post('/api/v1/checkout', {
    headers: authHeaders,
    data: { shippingAddressId: addressBody.data.id },
  });
  expect(checkoutRes.status()).toBe(201);
  const checkoutBody = await checkoutRes.json();
  expect(checkoutBody.data.status).toBe('PENDING'); // ยังไม่จ่ายเงิน สถานะต้องเป็น PENDING ก่อนเสมอ

  // ขั้น 6: จ่ายเงินจำลอง — เลขบัตร 4242... คือเลขที่ mock payment กำหนดให้ "ผ่านเสมอ"
  const payRes = await request.post('/api/v1/payments/mock', {
    headers: authHeaders,
    data: { orderId: checkoutBody.data.id, cardNumber: '4242424242424242' },
  });
  expect(payRes.status()).toBe(200);

  // ขั้น 7: ยืนยันผลลัพธ์สุดท้าย — โหลด order ใหม่จาก backend อีกครั้ง (ไม่เชื่อ response
  // ของขั้นจ่ายเงินตรงๆ) เพื่อพิสูจน์ว่าสถานะถูกอัปเดตเป็น PAID จริงในฐานข้อมูลแล้ว
  const finalOrderRes = await request.get(`/api/v1/orders/${checkoutBody.data.id}`, { headers: authHeaders });
  const finalOrderBody = await finalOrderRes.json();
  expect(finalOrderBody.data.status).toBe('PAID');
});
