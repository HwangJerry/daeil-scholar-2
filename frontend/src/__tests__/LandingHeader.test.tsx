// LandingHeader.test — Accessibility and keyboard contracts for landing navigation
import { act, render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { LandingHeader } from '../components/landing/LandingHeader';

const NAVIGATION_LABELS = ['앱 다운로드', '최근 소식', '장학회 소개', '장학사업'];
let intersectionObserverCallback: IntersectionObserverCallback | undefined;

class IntersectionObserverMock {
  constructor(callback: IntersectionObserverCallback) {
    intersectionObserverCallback = callback;
  }

  observe() {}
  unobserve() {}
  disconnect() {}
}

describe('LandingHeader', () => {
  afterEach(() => {
    intersectionObserverCallback = undefined;
    vi.unstubAllGlobals();
  });

  it('renders safely when landing sections do not exist yet', () => {
    render(<LandingHeader />);

    const desktopNavigation = screen.getByRole('navigation', { name: '랜딩 페이지' });
    expect(within(desktopNavigation).getAllByRole('link')).toHaveLength(4);
    expect(screen.getByRole('button', { name: '메뉴 열기' })).toHaveAttribute(
      'aria-expanded',
      'false',
    );
  });

  it('opens and closes the mobile navigation with mouse input', async () => {
    const user = userEvent.setup();
    render(<LandingHeader />);

    const menuButton = screen.getByRole('button', { name: '메뉴 열기' });
    await user.click(menuButton);

    expect(menuButton).toHaveAttribute('aria-expanded', 'true');
    const mobileNavigation = screen.getByRole('navigation', {
      name: '랜딩 페이지 모바일',
    });
    for (const label of NAVIGATION_LABELS) {
      expect(within(mobileNavigation).getByRole('link', { name: label })).toBeInTheDocument();
    }

    await user.click(screen.getByRole('button', { name: '메뉴 닫기' }));
    expect(
      screen.queryByRole('navigation', { name: '랜딩 페이지 모바일' }),
    ).not.toBeInTheDocument();
  });

  it('supports Enter, Tab, link activation, and Escape', async () => {
    const user = userEvent.setup();
    render(<LandingHeader />);

    const menuButton = screen.getByRole('button', { name: '메뉴 열기' });
    menuButton.focus();
    await user.keyboard('{Enter}');

    const mobileNavigation = screen.getByRole('navigation', {
      name: '랜딩 페이지 모바일',
    });
    const downloadLink = within(mobileNavigation).getByRole('link', {
      name: '앱 다운로드',
    });
    await user.tab();
    expect(downloadLink).toHaveFocus();
    await user.keyboard('{Enter}');
    expect(
      screen.queryByRole('navigation', { name: '랜딩 페이지 모바일' }),
    ).not.toBeInTheDocument();

    menuButton.focus();
    await user.keyboard('{Enter}');
    await user.keyboard('{Escape}');
    expect(menuButton).toHaveFocus();
    expect(menuButton).toHaveAttribute('aria-expanded', 'false');
  });

  it('marks an observed landing section as the current navigation item', () => {
    vi.stubGlobal('IntersectionObserver', IntersectionObserverMock);
    render(
      <>
        <LandingHeader />
        <section id="download" />
        <section id="news" />
      </>,
    );

    const newsSection = document.getElementById('news');
    expect(newsSection).not.toBeNull();
    act(() => {
      intersectionObserverCallback?.(
        [
          {
            isIntersecting: true,
            target: newsSection,
          } as unknown as IntersectionObserverEntry,
        ],
        {} as IntersectionObserver,
      );
    });

    const desktopNavigation = screen.getByRole('navigation', { name: '랜딩 페이지' });
    expect(within(desktopNavigation).getByRole('link', { name: '최근 소식' })).toHaveAttribute(
      'aria-current',
      'page',
    );
    expect(
      within(desktopNavigation).getByRole('link', { name: '앱 다운로드' }),
    ).not.toHaveAttribute('aria-current');
  });
});
