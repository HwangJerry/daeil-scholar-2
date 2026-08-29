// landing-organization.spec — Mobile overflow and organization-route regression checks
import { expect, test, type Page } from '@playwright/test';

const MOBILE_VIEWPORT = { width: 390, height: 844 } as const;

async function mockLandingAPI(page: Page) {
  await page.route('**/api/feed/hero', (route) =>
    route.fulfill({ contentType: 'application/json', body: 'null' }),
  );
  await page.route('**/api/feed?**', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({ items: [], nextCursor: null, hasMore: false }),
    }),
  );
  await page.route('**/api/banner-ad/active', (route) =>
    route.fulfill({ contentType: 'application/json', body: 'null' }),
  );
  await page.route('**/api/donation/summary', (route) =>
    route.fulfill({
      contentType: 'application/json',
      body: JSON.stringify({ displayAmount: 123_456_789 }),
    }),
  );
  await page.route('**/api/history', (route) =>
    route.fulfill({ contentType: 'application/json', body: '[]' }),
  );
  await page.route('**/api/visit/beacon', (route) => route.fulfill({ status: 204 }));
}

test('organization section fits 390px and the detail route remains available', async (
  { page },
  testInfo,
) => {
  await page.setViewportSize(MOBILE_VIEWPORT);
  await mockLandingAPI(page);

  const consoleErrors: string[] = [];
  page.on('console', (message) => {
    if (message.type() === 'error') consoleErrors.push(message.text());
  });

  await page.goto('/');

  const organizationSection = page.locator('#organization');
  await expect(organizationSection).toBeVisible();
  await organizationSection.scrollIntoViewIfNeeded();
  await expect(organizationSection.getByRole('heading', { name: '조직도' })).toBeVisible();
  await expect(organizationSection.getByRole('heading', { name: '엄은숙' })).toBeVisible();

  const documentWidths = await page.evaluate(() => ({
    clientWidth: document.documentElement.clientWidth,
    scrollWidth: document.documentElement.scrollWidth,
  }));
  expect(documentWidths.clientWidth).toBe(MOBILE_VIEWPORT.width);
  expect(documentWidths.scrollWidth).toBeLessThanOrEqual(documentWidths.clientWidth);
  expect(consoleErrors).toEqual([]);

  await organizationSection.screenshot({ path: testInfo.outputPath('organization-390.png') });

  await page.goto('/organization');
  await expect(page).toHaveURL('/organization');
  await expect(page.getByRole('heading', { name: '조직도', level: 1 })).toBeVisible();
  await expect(page.getByRole('heading', { name: '이사회' })).toBeVisible();
});
