// AppDownloadActions — Compact marketplace icons and safe-link behavior
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
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

describe('AppDownloadActions', () => {
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

  it('keeps unavailable destinations disabled with a coming-soon caption', () => {
    render(<AppDownloadActions downloadLinks={UNAVAILABLE_DOWNLOAD_LINKS} />);

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
    expect(screen.getByText('출시 준비 중')).toBeInTheDocument();
    expect(screen.queryAllByRole('link')).toHaveLength(0);
  });
});
