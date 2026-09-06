// privacy-policy.spec — Footer access and section deep links survive navigation and reload.
import { expect, test } from '@playwright/test';

for (const width of [375, 1440]) {
  test(`privacy policy navigation at ${width}px`, async ({ page }) => {
    await page.setViewportSize({ width, height: 900 });
    await page.route('**/api/visit/beacon', (route) => route.fulfill({ status: 204 }));
    await page.goto('/support');
    await page.getByRole('contentinfo').getByRole('link', { name: '개인정보처리방침' }).click();
    await expect(page).toHaveURL(/\/privacy$/);
    await expect(page.getByRole('heading', { name: '개인정보처리방침', exact: true })).toBeVisible();

    await page.getByRole('navigation', { name: '개인정보처리방침 목차' })
      .getByRole('link', { name: '개인정보 문의는 어디로 하나요?' }).click();
    await expect(page).toHaveURL(/#contact$/);
    await page.reload();
    await expect(page.getByRole('heading', { name: '개인정보 문의는 어디로 하나요?' })).toBeInViewport();
    await expect.poll(() => page.locator('#contact').evaluate((element) => element.getBoundingClientRect().top)).toBeGreaterThanOrEqual(60);
    await expect(page.locator('#contact').getByRole('link', { name: 'ghkdwp018@gmail.com' })).toHaveAttribute('href', 'mailto:ghkdwp018@gmail.com');
    expect(await page.evaluate(() => document.documentElement.scrollWidth <= innerWidth)).toBe(true);
  });
}
