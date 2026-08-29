// AppDownloadHero.test — Marketplace availability and safe-link rendering contracts
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { AppDownloadHero } from '../components/landing/AppDownloadHero';
import {
  createAppDownloadLink,
  type AppDownloadLinks,
} from '../constants/appDownload';

const UNAVAILABLE_DOWNLOAD_LINKS: AppDownloadLinks = {
  appStore: createAppDownloadLink(undefined),
  googlePlay: createAppDownloadLink('   '),
};

describe('AppDownloadHero', () => {
  it('renders disabled controls without fake marketplace links when URLs are unset', () => {
    render(<AppDownloadHero downloadLinks={UNAVAILABLE_DOWNLOAD_LINKS} />);

    expect(
      screen.getByRole('button', {
        name: 'App Store에서 다운로드, 출시 준비 중',
      }),
    ).toBeDisabled();
    expect(
      screen.getByRole('button', {
        name: 'Google Play에서 다운로드, 출시 준비 중',
      }),
    ).toBeDisabled();
    expect(
      screen.queryByRole('link', { name: /App Store에서 다운로드/ }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('link', { name: /Google Play에서 다운로드/ }),
    ).not.toBeInTheDocument();
    expect(screen.getByRole('link', { name: '최근 소식 보기' })).toHaveAttribute(
      'href',
      '#news',
    );
  });

  it('only accepts valid HTTPS marketplace URLs', () => {
    expect(createAppDownloadLink('javascript:alert(1)')).toEqual({
      url: null,
      available: false,
    });
    expect(createAppDownloadLink('https://apps.example.com/download')).toEqual({
      url: 'https://apps.example.com/download',
      available: true,
    });
  });

  it('loads both app icon assets eagerly', () => {
    render(<AppDownloadHero downloadLinks={UNAVAILABLE_DOWNLOAD_LINKS} />);

    expect(
      screen.getByRole('img', { name: '대일외고 장학회 iOS 앱 아이콘' }),
    ).toHaveAttribute('loading', 'eager');
    expect(
      screen.getByRole('img', { name: '대일외고 장학회 Android 앱 아이콘' }),
    ).toHaveAttribute('loading', 'eager');
  });
});
