// receipt-work-admin.spec — Root review requires secured originals and independent cleanup confirmation.
import { expect, test } from '@playwright/test';

const adminURL = process.env.ADMIN_ERASURE_TEST_URL;
test.skip(!adminURL, 'Set ADMIN_ERASURE_TEST_URL to an isolated admin dev server.');
for (const width of [375, 1440]) {
  test(`receipt work lifecycle at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 1000 });
    let state = 'unreviewed';
    let calls = 0;
    await page.route(url => url.pathname.startsWith('/api/'), route => route.fulfill({ json: {} }));
    await page.route('**/api/auth/me', route => route.fulfill({ json: { usrSeq: 7, name: 'Synthetic Root', adminRole: 'root' } }));
    await page.route('**/api/admin/account-deletions**', async route => {
      if (route.request().method() === 'PUT') {
        const request = route.request().postDataJSON();
        expect(request.action).toBe('receipt_work');
        if (calls === 0) {
          expect(request.receiptWorkStatus).toBe('active');
          expect(request.contactSecured).toBe(true);
          expect(request.originalStorage).toBe('separate_excel');
        } else {
          expect(request.receiptWorkStatus).toBe('completed');
          expect(request.resultNotified).toBe(true);
          expect(request.contactErased).toBe(true);
        }
        state = request.receiptWorkStatus;
        calls++;
        await route.fulfill({ status: 204 });
        return;
      }
      await route.fulfill({ json: { items: [{
        requestId: 17, userSeq: 42, status: 'pending', processingMode: 'automatic',
        requestedAt: '2026-09-07T00:00:00Z', targetAt: '2026-09-10T00:00:00Z', dueAt: '2026-09-17T00:00:00Z',
        receiptWork: { status: state, originalStorage: 'separate_excel', evidenceReference: '', updatedAt: '2026-09-07T00:00:00Z' },
      }] } });
    });
    await page.goto(`${adminURL}/admin/account-deletions`);
    await expect(page.getByRole('heading', { name: '영수증 연락 업무 · 미확인' })).toBeVisible();
    await page.getByLabel('진행 중인 발급·정정·재발급 업무').selectOption('active');
    await page.getByLabel(/업무·원본 확인 또는 완료·정리 증빙 번호/).fill('synthetic-work-17');
    await expect(page.getByRole('button', { name: '영수증 업무 확인 저장' })).toBeDisabled();
    await page.getByLabel(/구체적인 진행 업무가 있으며/).check();
    await page.getByRole('button', { name: '영수증 업무 확인 저장' }).click();
    await expect(page.getByRole('heading', { name: '영수증 연락 업무 · 영수증 업무 진행 중' })).toBeVisible();
    await page.getByLabel(/업무·원본 확인 또는 완료·정리 증빙 번호/).fill('synthetic-cleanup-17');
    await page.getByLabel(/발급·정정·재발급 업무와 결과 전달을 모두 완료/).check();
    await expect(page.getByRole('button', { name: '결과 전달·연락처 정리 확인' })).toBeDisabled();
    await page.getByLabel(/해피나눔·회계 엑셀 등 해당 업무용 원본과 사본/).check();
    await page.getByRole('button', { name: '결과 전달·연락처 정리 확인' }).click();
    await expect(page.getByRole('heading', { name: '영수증 연락 업무 · 결과 전달·연락처 정리 완료' })).toBeVisible();
    expect(calls).toBe(2);
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    await page.screenshot({ path: `/tmp/dflh-receipt-work-${width}.png`, fullPage: true });
  });
}
