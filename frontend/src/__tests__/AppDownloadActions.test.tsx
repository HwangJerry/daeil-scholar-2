// AppDownloadActions — Compact marketplace icons and safe-link behavior
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { AppDownloadActions } from '../components/landing/AppDownloadActions';
import {
  createAppDownloadLink,
  type AppDownloadLinks,
} from '../constants/appDownload';

const AVAILABLE_DOWNLOAD_LINKS: AppDownloadLinks = {
  appStore: createAppDownloadLink('https://apps.example.com/download'),
  googlePlay: createAppDownloadLink('https://play.example.com/download'),
};

const UNAVAILABLE_DOWNLOAD_LINKS: AppDownloadLinks = {
  appStore: createAppDownloadLink(undefined),
  googlePlay: createAppDownloadLink(''),
};

function mockMobileViewport(isMobile: boolean) {
  vi.stubGlobal(
    'matchMedia',
    vi.fn().mockImplementation((query: string) => ({
      matches: query === '(max-width: 767px)' && isMobile,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })),
  );
}

describe('AppDownloadActions', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('renders compact icon-only actions with accessible labels', () => {
    render(<AppDownloadActions downloadLinks={AVAILABLE_DOWNLOAD_LINKS} />);

    const appStoreLink = screen.getByRole('link', {
      name: 'App Store에서 다운로드',
    });
    const googlePlayLink = screen.getByRole('link', {
      name: 'Google Play에서 다운로드',
    });

    expect(appStoreLink).toHaveClass('size-[52px]', 'bg-primary', 'text-surface');
    expect(googlePlayLink).toHaveClass('size-[52px]', 'bg-primary', 'text-surface');
    expect(appStoreLink.querySelector('svg')).toHaveAttribute('aria-hidden', 'true');
    expect(googlePlayLink.querySelector('svg')).toHaveAttribute('aria-hidden', 'true');
    expect(screen.queryAllByRole('img')).toHaveLength(0);
    expect(screen.queryByText('App Store에서 다운로드')).not.toBeInTheDocument();
    expect(screen.queryByText('Google Play에서 다운로드')).not.toBeInTheDocument();
  });

  it('opens available marketplace destinations in a new tab', () => {
    render(<AppDownloadActions downloadLinks={AVAILABLE_DOWNLOAD_LINKS} />);

    const appStoreLink = screen.getByRole('link', {
      name: 'App Store에서 다운로드',
    });
    const googlePlayLink = screen.getByRole('link', {
      name: 'Google Play에서 다운로드',
    });

    expect(appStoreLink).toHaveAttribute(
      'href',
      AVAILABLE_DOWNLOAD_LINKS.appStore.url,
    );
    expect(googlePlayLink).toHaveAttribute(
      'href',
      AVAILABLE_DOWNLOAD_LINKS.googlePlay.url,
    );
    expect(appStoreLink).toHaveAttribute('target', '_blank');
    expect(googlePlayLink).toHaveAttribute('target', '_blank');
    expect(appStoreLink).toHaveAttribute('rel', 'noreferrer');
    expect(googlePlayLink).toHaveAttribute('rel', 'noreferrer');
  });

  it('opens an AlertDialog for an unavailable marketplace on desktop', async () => {
    const user = userEvent.setup();
    mockMobileViewport(false);
    render(<AppDownloadActions downloadLinks={UNAVAILABLE_DOWNLOAD_LINKS} />);

    const appStoreButton = screen.getByRole('button', {
      name: 'App Store 출시 준비 중 안내 보기',
    });
    const googlePlayButton = screen.getByRole('button', {
      name: 'Google Play 출시 준비 중 안내 보기',
    });

    expect(appStoreButton).toBeEnabled();
    expect(googlePlayButton).toBeEnabled();
    expect(screen.queryByText('출시 준비 중')).not.toBeInTheDocument();
    expect(screen.queryAllByRole('link')).toHaveLength(0);

    await user.click(appStoreButton);

    expect(screen.getByText('출시 준비 중')).toBeInTheDocument();
    expect(
      screen.getByText(
        'App Store에 곧 출시될 예정입니다. 조금만 기다려 주세요!',
      ),
    ).toBeInTheDocument();
    expect(document.querySelector('.lucide-rocket')).not.toBeNull();

    await user.click(screen.getByRole('button', { name: '확인' }));
    expect(screen.queryByText('출시 준비 중')).not.toBeInTheDocument();
  });

  it('opens a BottomSheet for an unavailable marketplace on mobile', async () => {
    const user = userEvent.setup();
    mockMobileViewport(true);
    render(<AppDownloadActions downloadLinks={UNAVAILABLE_DOWNLOAD_LINKS} />);

    await user.click(
      screen.getByRole('button', {
        name: 'Google Play 출시 준비 중 안내 보기',
      }),
    );

    expect(screen.getByRole('dialog')).toBeInTheDocument();
    expect(
      screen.getByText(
        'Google Play에 곧 출시될 예정입니다. 조금만 기다려 주세요!',
      ),
    ).toBeInTheDocument();
  });
});
