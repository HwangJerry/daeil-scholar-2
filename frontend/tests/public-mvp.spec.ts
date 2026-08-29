import { expect, test, type Page } from '@playwright/test';

const LANDING_NAVIGATION = [
  { label: '앱 다운로드', href: '#download' },
  { label: '최근 소식', href: '#news' },
  { label: '장학회 소개', href: '#about' },
  { label: '장학사업', href: '#business' },
] as const;

const FOUNDATION_ROUTES = [
  { path: '/about', heading: '대일외고 장학회' },
  { path: '/greetings', heading: '인사말' },
  { path: '/vision', heading: '비전가치체계' },
  { path: '/history', heading: '연혁' },
  { path: '/organization', heading: '조직도' },
  { path: '/business', heading: '장학사업' },
] as const;

const HISTORY_EVENT = '후배 지원을 위한 새로운 장학사업을 시작했습니다.';

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
  await page.route('**/api/banner-ad/active', (route) => route.fulfill({
    contentType: 'application/json',
    body: 'null',
  }));
  await page.route('**/api/history', (route) => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify([
      {
        year: 2025,
        items: [
          {
            heSeq: 12,
            eventDate: '2025-11-03',
            text: HISTORY_EVENT,
            sortOrder: 1,
          },
        ],
      },
    ]),
  }));
  await page.route('**/api/disclosure?**', (route) => route.fulfill({
    contentType: 'application/json',
    body: JSON.stringify({
      items: [
        {
          seq: 1,
          subject: '2025년 결산 공시',
          summary: '공시 요약',
          regDate: '2026-03-31T00:00:00Z',
          regName: '장학회',
          hit: 1,
          attachmentCount: 0,
        },
      ],
      nextCursor: '',
      hasMore: false,
    }),
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

test('desktop landing exposes the hero, safe download CTAs, and valid anchor navigation', async ({ page }) => {
  await mockPublicAPI(page);
  await page.goto('/');

  const hero = page.locator('#download');
  await expect(hero.getByRole('heading', {
    name: '대일의 오늘과 내일을 잇습니다.',
    level: 1,
  })).toBeVisible();
  await expect(hero.getByRole('button', {
    name: 'App Store에서 다운로드, 출시 준비 중',
  })).toBeDisabled();
  await expect(hero.getByRole('button', {
    name: 'Google Play에서 다운로드, 출시 준비 중',
  })).toBeDisabled();
  await expect(hero.getByRole('link', { name: /App Store에서 다운로드/ })).toHaveCount(0);
  await expect(hero.getByRole('link', { name: /Google Play에서 다운로드/ })).toHaveCount(0);

  const orderedSectionIds = await page
    .locator('main > [data-scroll-reveal] > section[id]')
    .evaluateAll((sections) => sections.map((section) => section.id));
  expect(orderedSectionIds.slice(0, 2)).toEqual(['download', 'news']);

  const navigation = page.getByRole('navigation', { name: '랜딩 페이지' });
  await expect(navigation.getByRole('link')).toHaveCount(LANDING_NAVIGATION.length);
  for (const item of LANDING_NAVIGATION) {
    const link = navigation.getByRole('link', { name: item.label });
    await expect(link).toHaveAttribute('href', item.href);
    await expect(page.locator(item.href)).toHaveCount(1);
  }

  const summary = page.getByRole('complementary', { name: '장학회 누적 기부 현황' });
  await expect(summary.getByText('누적 기부액', { exact: true })).toBeVisible();
  await expect(summary.getByText('1억 2,345만원')).toBeVisible();
  await expect(summary.getByText(/명 참여|목표|달성률/)).toHaveCount(0);
});

test('unknown routes redirect to the completed landing page', async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await mockPublicAPI(page);
  await page.goto('/login/legacy');

  await expect(page).toHaveURL('/');
  await expect(page.getByRole('heading', {
    name: '대일의 오늘과 내일을 잇습니다.',
    level: 1,
  })).toBeVisible();
  await expect(page.getByRole('button', { name: '메뉴 열기' })).toBeVisible();
  await expect(page.locator('input[type="password"]')).toHaveCount(0);
});

// HistorySection.test.tsx owns deterministic loading, error, and retry coverage.
test('all foundation sections and successful history data share the landing page', async ({ page }) => {
  await mockPublicAPI(page);
  await page.goto('/');

  const sections = [
    { id: 'about', heading: /동문의 마음으로.*후배의 내일을 엽니다/ },
    { id: 'greeting', heading: '인사말' },
    { id: 'vision', heading: '비전과 핵심가치' },
    { id: 'history', heading: '연혁' },
    { id: 'organization', heading: '조직도' },
    { id: 'business', heading: '대일외고 장학회 장학사업 현황' },
  ] as const;

  for (const sectionDefinition of sections) {
    const section = page.locator(`#${sectionDefinition.id}`);
    await expect(section).toHaveCount(1);
    await expect(section.getByRole('heading', {
      name: sectionDefinition.heading,
      level: 2,
    })).toBeAttached();
  }

  const historySection = page.locator('#history');
  await expect(historySection.getByText(HISTORY_EVENT)).toBeAttached();
  await expect(historySection.getByRole('heading', { name: '2025' })).toBeAttached();
});

test('legacy foundation URLs remain directly accessible', async ({ page }) => {
  await mockPublicAPI(page);

  for (const route of FOUNDATION_ROUTES) {
    await test.step(route.path, async () => {
      await page.goto(route.path);
      await expect(page).toHaveURL(route.path);
      await expect(page.getByRole('heading', { name: route.heading, level: 1 })).toBeVisible();
    });
  }
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

test('the mandatory disclosure link, list, and detail route remain available', async ({ page }) => {
  await mockPublicAPI(page);
  await page.goto('/');

  const disclosureLink = page.getByRole('link', { name: '의무공시' });
  await expect(disclosureLink).toHaveAttribute('href', '/disclosure');
  await disclosureLink.click();

  await expect(page).toHaveURL('/disclosure');
  await expect(page.getByRole('heading', { name: '의무공시', level: 1 })).toBeVisible();

  const disclosureDetailLink = page.getByRole('link', { name: '2025년 결산 공시' });
  await expect(disclosureDetailLink).toHaveAttribute('href', '/disclosure/1');
  await disclosureDetailLink.click();

  await expect(page).toHaveURL('/disclosure/1');
  await expect(page.getByRole('heading', { name: '2025년 결산 공시' })).toBeVisible();
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

test('reduced-motion users receive revealed content without transitions or animations', async ({ page }) => {
  await page.emulateMedia({ reducedMotion: 'reduce' });
  await mockPublicAPI(page);
  await page.goto('/');

  const revealBoundaries = page.locator('[data-scroll-reveal]');
  await expect(revealBoundaries).toHaveCount(10);
  await expect(revealBoundaries.first()).toHaveAttribute('data-revealed', 'true');

  const hasDisabledRevealMotion = await revealBoundaries.evaluateAll((elements) =>
    elements.every((element) => {
      const styles = window.getComputedStyle(element);
      return styles.opacity === '1'
        && styles.transform === 'none'
        && styles.transitionDuration === '0s';
    }),
  );
  expect(hasDisabledRevealMotion).toBe(true);

  const heroNoticeAnimationDuration = await page
    .getByRole('link', { name: /장학회 주요 소식/ })
    .first()
    .evaluate((element) => window.getComputedStyle(element).animationDuration);
  expect(heroNoticeAnimationDuration).toBe('0s');
});

test.describe('mobile landing menu', () => {
  test.use({
    hasTouch: true,
    isMobile: true,
    viewport: { width: 390, height: 844 },
  });

  test('supports keyboard and touch input', async ({ page }) => {
    await mockPublicAPI(page);
    await page.goto('/');

    const menuButton = page.locator('button[aria-controls="landing-mobile-navigation"]');
    await menuButton.focus();
    await page.keyboard.press('Enter');

    await expect(menuButton).toHaveAttribute('aria-expanded', 'true');
    const mobileNavigation = page.getByRole('navigation', {
      name: '랜딩 페이지 모바일',
    });
    await expect(mobileNavigation.getByRole('link')).toHaveCount(LANDING_NAVIGATION.length);

    await page.keyboard.press('Tab');
    await expect(mobileNavigation.getByRole('link', { name: '앱 다운로드' })).toBeFocused();

    await page.keyboard.press('Escape');
    await expect(menuButton).toHaveAttribute('aria-expanded', 'false');
    await expect(menuButton).toBeFocused();

    await menuButton.tap();
    await expect(menuButton).toHaveAttribute('aria-expanded', 'true');
    await mobileNavigation.getByRole('link', { name: '최근 소식' }).tap();

    await expect(page).toHaveURL(/\/#news$/);
    await expect(menuButton).toHaveAttribute('aria-expanded', 'false');
  });
});
