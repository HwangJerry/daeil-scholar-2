import { expect, test, type Page } from '@playwright/test';

async function mockPublicAPI(page: Page) {
  await page.route('**/api/feed/hero', (route) => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify({
      type: 'notice',
      seq: 1,
      subject: '장학회 주요 소식',
      summary: '후배들을 위한 장학사업 소식입니다.',
      thumbnailUrl: null,
      regDate: '2026-07-29T00:00:00Z',
      regName: '장학회',
      hit: 3,
      likeCnt: 0,
      commentCnt: 0,
      isPinned: 'Y',
      userLiked: false,
    }),
  }));
  await page.route('**/api/feed?**', (route) => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify({ items: [], nextCursor: null, hasMore: false }),
  }));
  await page.route('**/api/feed/1', (route) => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify({
      seq: 1,
      subject: '장학회 주요 소식',
      contentHtml: '<p>소식 본문</p>',
      contentFormat: 'MARKDOWN',
      summary: '후배들을 위한 장학사업 소식입니다.',
      thumbnailUrl: null,
      regDate: '2026-07-29T00:00:00Z',
      regName: '장학회',
      hit: 3,
      likeCnt: 0,
      commentCnt: 0,
      userLiked: false,
      files: [],
    }),
  }));
  await page.route('**/api/feed/1/siblings', (route) => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify({ prev: null, next: null }),
  }));
  await page.route('**/api/donation/summary', (route) => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify({ displayAmount: 123_456_789 }),
  }));
  await page.route('**/api/disclosure/1', (route) => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify({
      seq: 1,
      subject: '2025년 결산 공시',
      contentHtml: '<p>공시 본문</p>',
      contentFormat: 'MARKDOWN',
      summary: '공시 요약',
      thumbnailUrl: null,
      regDate: '2026-03-31T00:00:00Z',
      regName: '장학회',
      hit: 1,
      likeCnt: 0,
      commentCnt: 0,
      userLiked: false,
      files: [],
    }),
  }));
  await page.route('**/api/visit/beacon', (route) => route.fulfill({ status: 204 }));
}

test('desktop home exposes two public navigation links and canonical donation content', async ({ page }) => {
  await mockPublicAPI(page);
  await page.goto('/');

  const navigation = page.locator('header nav');
  await expect(navigation.getByRole('link')).toHaveCount(2);
  await expect(navigation.getByRole('link', { name: '소식' })).toBeVisible();
  await expect(navigation.getByRole('link', { name: '장학회 소개' })).toBeVisible();
  await expect(navigation.getByRole('link', { name: /누적 기부액|동문|쪽지|마이|로그인|광고/ })).toHaveCount(0);

  await page.keyboard.press('Tab');
  await page.keyboard.press('Tab');
  await expect(navigation.getByRole('link', { name: '소식' })).toBeFocused();
  await page.keyboard.press('Tab');
  await expect(navigation.getByRole('link', { name: '장학회 소개' })).toBeFocused();

  const summary = page.locator('#donation-summary');
  await expect(summary.getByRole('heading', { name: '누적 기부액' })).toBeVisible();
  await expect(summary.getByText('1억 2,345만원')).toBeVisible();
  await expect(summary.getByText(/명 참여|목표|달성률/)).toHaveCount(0);
});

test('mobile navigation has the same two public areas and retired routes redirect home', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await mockPublicAPI(page);
  await page.goto('/login/legacy');

  await expect(page).toHaveURL('/');
  const navigation = page.locator('nav.fixed');
  await expect(navigation.getByRole('link')).toHaveCount(2);
  await expect(navigation.getByRole('link', { name: '소식' })).toBeVisible();
  await expect(navigation.getByRole('link', { name: '장학회 소개' })).toBeVisible();
  await expect(navigation.getByRole('link', { name: '누적 기부액' })).toHaveCount(0);
  await expect(page.locator('input[type="password"]')).toHaveCount(0);
});

test('foundation information remains directly crawlable', async ({ page }) => {
  await mockPublicAPI(page);
  await page.goto('/about');

  await expect(page).toHaveURL('/about');
  await expect(page.getByRole('heading', { name: '대일외고 장학회' })).toBeVisible();
});

test('foundation detail pages keep their parent active and return directly to the about hub', async ({ page }) => {
  await mockPublicAPI(page);
  await page.goto('/vision');

  const navigation = page.locator('header nav');
  await expect(navigation.getByRole('link', { name: '장학회 소개' })).toHaveAttribute(
    'aria-current',
    'page',
  );
  await expect(navigation.getByRole('link', { name: '소식' })).not.toHaveAttribute('aria-current');

  const staticBackLink = page.getByRole('link', { name: '장학회 소개로 돌아가기' });
  await expect(staticBackLink).toHaveAttribute('href', '/about');
  await staticBackLink.click();
  await expect(page).toHaveURL('/about');

  await page.goto('/disclosure/1');
  await expect(page.getByRole('heading', { name: '2025년 결산 공시' })).toBeVisible();
  await expect(page.getByRole('link', { name: '장학회 소개로 돌아가기' })).toHaveAttribute(
    'href',
    '/about',
  );
  await expect(navigation.getByRole('link', { name: '장학회 소개' })).toHaveAttribute(
    'aria-current',
    'page',
  );
});

test('news detail keeps news active and returns directly to the news list', async ({ page }) => {
  await mockPublicAPI(page);
  await page.goto('/post/1');

  await expect(page.getByRole('heading', { name: '장학회 주요 소식' })).toBeVisible();

  const navigation = page.locator('header nav');
  await expect(navigation.getByRole('link', { name: '소식' })).toHaveAttribute(
    'aria-current',
    'page',
  );
  await expect(navigation.getByRole('link', { name: '장학회 소개' })).not.toHaveAttribute(
    'aria-current',
  );

  const backLink = page.getByRole('link', { name: '소식 목록으로 돌아가기' });
  await expect(backLink).toHaveAttribute('href', '/');
  await backLink.click();
  await expect(page).toHaveURL('/');

  await page.getByRole('link', { name: /장학회 주요 소식/ }).first().click();
  await expect(page.getByRole('button', { name: '닫기' })).toBeVisible();
  await expect(page.getByRole('link', { name: '소식 목록으로 돌아가기' })).toHaveAttribute(
    'href',
    '/',
  );
});