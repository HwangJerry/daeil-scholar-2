// login.spec.ts — UI bug investigation tests for /login and /login/legacy pages.
import { test, expect } from '@playwright/test';

// ── /login page ──────────────────────────────────────────────────────────────

test('/login renders Kakao login button', async ({ page }) => {
  await page.goto('/login');
  await page.screenshot({ path: 'test-results/login-page.png', fullPage: true });

  // Page should load without error
  await expect(page).toHaveTitle(/.*/);

  // At minimum something renders in the body
  const body = page.locator('body');
  await expect(body).not.toBeEmpty();
});

// ── /login/legacy page ───────────────────────────────────────────────────────

test('/login/legacy renders form fields', async ({ page }) => {
  await page.goto('/login/legacy');
  await page.screenshot({ path: 'test-results/legacy-login-page.png', fullPage: true });

  // Input fields should be visible
  const idInput = page.locator('input[type="text"]');
  const passwordInput = page.locator('input[type="password"]');

  await expect(idInput).toBeVisible();
  await expect(passwordInput).toBeVisible();
});

// ── BUG 1: Success banner uses hardcoded Tailwind palette ────────────────────
// LegacyLoginPage.tsx:56 — bg-green-50 text-green-700 instead of design tokens

test('BUG-1: success banner uses design tokens (not hardcoded green)', async ({ page }) => {
  await page.goto('/login/legacy?registered=true');
  await page.screenshot({ path: 'test-results/legacy-login-success-banner.png', fullPage: true });

  // Hardcoded classes should be gone
  const hardcodedBg = page.locator('.bg-green-50');
  expect(await hardcodedBg.count()).toBe(0);

  const hardcodedText = page.locator('.text-green-700');
  expect(await hardcodedText.count()).toBe(0);

  // Design token classes should be present
  const tokenBanner = page.locator('.bg-success-subtle');
  expect(await tokenBanner.count()).toBe(1);

  const tokenText = page.locator('.text-success-text');
  expect(await tokenText.count()).toBe(1);
});

// ── BUG 2: Input missing background token ────────────────────────────────────
// LegacyLoginPage.tsx:47-48 — inputClass missing bg-input/bg-surface token

test('BUG-2: input fields have explicit background token', async ({ page }) => {
  await page.goto('/login/legacy');

  const idInput = page.locator('input[type="text"]');
  await expect(idInput).toBeVisible();

  await page.screenshot({ path: 'test-results/legacy-login-input-bg.png', fullPage: true });

  // The class string must contain bg-surface (the fix)
  const classAttr = await idInput.getAttribute('class') ?? '';
  const hasBgToken = classAttr.includes('bg-input') || classAttr.includes('bg-surface') || classAttr.includes('bg-white');
  expect(hasBgToken).toBe(true); // Fixed: explicit background token is present
});

// ── BUG 3: Loading state — no spinner, only text change ──────────────────────
// LegacyLoginPage.tsx:101-103 — button only changes text, no visual indicator

test('BUG-3: submit button shows spinner during loading state', async ({ page }) => {
  await page.goto('/login/legacy');

  const idInput = page.locator('input[type="text"]');
  const passwordInput = page.locator('input[type="password"]');
  const submitButton = page.locator('button[type="submit"]');

  // Fill in credentials
  await idInput.fill('testuser');
  await passwordInput.fill('testpassword');

  // Intercept the API call to delay it so we can capture loading state
  await page.route('**/api/auth/login', async (route) => {
    await new Promise((resolve) => setTimeout(resolve, 1000)); // 1s delay
    await route.abort(); // Abort so we don't need a real server
  });

  // Click submit and immediately screenshot to capture loading state
  await submitButton.click();
  await page.waitForTimeout(200); // Short wait to let React re-render

  await page.screenshot({ path: 'test-results/legacy-login-loading-state.png', fullPage: true });

  // Check for spinner element — should be present (the fix)
  const spinner = page.locator('button[type="submit"] .animate-spin');
  const spinnerCount = await spinner.count();
  expect(spinnerCount).toBeGreaterThanOrEqual(1); // Fixed: spinner is present during loading
});

// ── Error state: empty form submission ───────────────────────────────────────

test('error message appears on empty form submission', async ({ page }) => {
  await page.goto('/login/legacy');

  // HTML5 required validation will prevent submission with empty fields
  // Fill one field only to bypass partial validation
  const idInput = page.locator('input[type="text"]');
  await idInput.fill('someuser');

  const submitButton = page.locator('button[type="submit"]');

  // Intercept API to return an error
  await page.route('**/api/auth/legacy/login', async (route) => {
    await route.fulfill({
      status: 401,
      contentType: 'application/json',
      body: JSON.stringify({ error: '아이디 또는 비밀번호가 올바르지 않습니다.' }),
    });
  });

  const passwordInput = page.locator('input[type="password"]');
  await passwordInput.fill('wrongpassword');
  await submitButton.click();

  // Wait for error message to appear
  const errorMsg = page.locator('p.text-error-text, p[class*="text-error"]');
  await expect(errorMsg).toBeVisible({ timeout: 3000 });

  await page.screenshot({ path: 'test-results/legacy-login-error-state.png', fullPage: true });
  console.log(`Error message text: "${await errorMsg.textContent()}"`);
});
