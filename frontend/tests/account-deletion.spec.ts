// account-deletion.spec — Private receipts stay public during maintenance without leaking tokens.
import { expect, test } from '@playwright/test';

const token = 'a'.repeat(64);
for (const width of [375, 1440]) {
  test(`private deletion receipt at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 });
    await page.route('**/api/visit/beacon', route => route.fulfill({ status: 204 }));
    let status = 'pending';
    const requestedURLs: string[] = [];
    page.on('request', request => requestedURLs.push(request.url()));
    await page.route('**/api/account-deletion/receipt', async route => {
      expect(route.request().method()).toBe('POST');
      expect(route.request().postDataJSON()).toEqual({ receiptToken: token });
      await route.fulfill({ json: {
        requestId: 17, status, receiptWorkPending: status === 'processing', databaseErased: status !== 'pending', requestedAt: '2026-09-06T09:00:00Z', targetAt: '2026-09-09T09:00:00Z', dueAt: '2026-09-16T09:00:00Z',
        completedAt: status === 'completed' ? '2026-09-07T09:00:00Z' : null, retainedRecords: '없음', retentionUntil: null,
      } });
    });
    await page.goto(`/account-deletion#receipt=${token}`);
    await expect(page.getByRole('heading', { name: '삭제 요청 접수', exact: true })).toBeVisible();
    await expect(page).toHaveURL(/\/account-deletion$/);
    expect(requestedURLs.every(url => !url.includes(token))).toBe(true);
    expect(await page.evaluate(value => !JSON.stringify(localStorage).includes(value) && !JSON.stringify(sessionStorage).includes(value), token)).toBe(true);
    await expect(page.locator('meta[name="robots"]')).toHaveAttribute('content', 'noindex, nofollow');
    status = 'processing';
    await page.getByRole('button', { name: '처리 현황 확인' }).click();
    await expect(page.getByText(/앱 운영 데이터는 삭제했습니다/)).toBeVisible();
    await expect(page.getByText(/진행 중인 영수증 업무를 처리하고 있습니다/)).toBeVisible();
    await expect(page.getByRole('heading', { name: '계정 삭제 완료', exact: true })).toHaveCount(0);
    status = 'completed';
    await page.getByRole('button', { name: '처리 현황 확인' }).click();
    await expect(page.getByRole('heading', { name: '계정 삭제 완료', exact: true })).toBeVisible();
    await expect(page.getByText('법정 보존 안내: 없음')).toBeVisible();
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
    await page.screenshot({ path: `/tmp/dflh-account-deletion-${width}.png`, fullPage: true });
  });
}

test('unknown receipt reports failure without showing completion', async ({ page }) => {
  await page.route('**/api/visit/beacon', route => route.fulfill({ status: 204 }));
  await page.route('**/api/account-deletion/receipt', route => route.fulfill({ status: 404, json: { code: 'DELETION_REQUEST_NOT_FOUND', message: '삭제 요청을 찾을 수 없습니다.' } }));
  await page.goto('/account-deletion');
  await page.getByLabel('삭제 요청 확인번호').fill(token);
  await page.getByRole('button', { name: '처리 현황 확인' }).click();
  await expect(page.getByRole('alert')).toHaveText('삭제 요청을 찾을 수 없습니다.');
  await expect(page.getByRole('heading', { name: '계정 삭제 완료', exact: true })).toHaveCount(0);
});
