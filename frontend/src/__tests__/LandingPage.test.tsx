// LandingPage.test — Landing composition order for the closing support and footer blocks
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { beforeEach, vi, describe, expect, it } from 'vitest';
import { LandingPage } from '../pages/LandingPage';

const pageMetaSpy = vi.hoisted(() => vi.fn());

vi.mock('../components/seo/PageMeta', () => ({
  PageMeta: (props: unknown) => {
    pageMetaSpy(props);
    return null;
  },
}));
vi.mock('../components/landing/LandingHeader', () => ({ LandingHeader: () => <header /> }));
vi.mock('../components/landing/LatestNewsSection', () => ({
  LatestNewsSection: () => <section data-landing-block="news" />,
}));
vi.mock('../components/feed/BannerAdSection', () => ({
  BannerAdSection: () => <section data-landing-block="banner" />,
}));
vi.mock('../components/landing/FoundationOverviewSection', () => ({
  FoundationOverviewSection: () => <section data-landing-block="about" />,
}));
vi.mock('../components/landing/GreetingSection', () => ({
  GreetingSection: () => <section data-landing-block="greeting" />,
}));
vi.mock('../components/landing/VisionSection', () => ({
  VisionSection: () => <section data-landing-block="vision" />,
}));
vi.mock('../components/landing/HistorySection', () => ({
  HistorySection: () => <section data-landing-block="history" />,
}));
vi.mock('../components/landing/OrganizationSection', () => ({
  OrganizationSection: () => <section data-landing-block="organization" />,
}));
vi.mock('../components/landing/ScholarshipBusinessSection', () => ({
  ScholarshipBusinessSection: () => <section data-landing-block="business" />,
}));
vi.mock('../components/landing/SupportSection', () => ({
  SupportSection: () => <section data-landing-block="support" />,
}));
vi.mock('../components/landing/LandingFooter', () => ({
  LandingFooter: () => <footer data-landing-block="footer" />,
}));

describe('LandingPage', () => {
  beforeEach(() => {
    pageMetaSpy.mockClear();
  });

  it('places support after scholarship business and the landing footer last', () => {
    const { container } = render(<LandingPage />);
    const blockOrder = Array.from(
      container.querySelectorAll<HTMLElement>('[data-landing-block]'),
      (element) => element.dataset.landingBlock,
    );

    expect(blockOrder).toEqual([
      'news',
      'banner',
      'about',
      'greeting',
      'vision',
      'history',
      'organization',
      'business',
      'support',
      'footer',
    ]);
  });

  it('has one h1 and wraps every landing section in main', () => {
    const { container } = render(<LandingPage />);
    const main = screen.getByRole('main');

    expect(container.querySelectorAll('h1')).toHaveLength(1);
    expect(screen.getByRole('heading', { level: 1 })).toHaveTextContent(
      '대일의 오늘과 내일을 잇습니다.',
    );
    expect(main).toHaveAttribute('id', 'main-content');
    expect(main.querySelectorAll('section')).toHaveLength(10);
    expect(container.querySelector('footer')).not.toBeNull();
    expect(main.querySelector('footer')).toBeNull();
  });

  it('moves keyboard focus to main through the skip link', async () => {
    const user = userEvent.setup();
    render(<LandingPage />);

    await user.tab();
    const skipLink = screen.getByRole('link', { name: '본문으로 건너뛰기' });
    expect(skipLink).toHaveFocus();

    await user.keyboard('{Enter}');
    expect(screen.getByRole('main')).toHaveFocus();
  });

  it('uses landing-specific metadata and keeps the root canonical path', () => {
    render(<LandingPage />);

    expect(pageMetaSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        title: '대일의 오늘과 내일을 잇습니다',
        description: expect.stringContaining('장학사업'),
        canonicalPath: '/',
      }),
    );
  });
});
