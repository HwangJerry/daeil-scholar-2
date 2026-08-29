// SupportSection.test — Closing CTA links and unavailable marketplace safety contracts
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { SupportSection } from '../components/landing/SupportSection';
import {
  createAppDownloadLink,
  type AppDownloadLinks,
} from '../constants/appDownload';
import { EXTERNAL_DONATION_URL } from '../constants/donation';

const UNAVAILABLE_DOWNLOAD_LINKS: AppDownloadLinks = {
  appStore: createAppDownloadLink(undefined),
  googlePlay: createAppDownloadLink(''),
};

describe('SupportSection', () => {
  it('uses the shared donation destination and required closing headline', () => {
    render(<SupportSection downloadLinks={UNAVAILABLE_DOWNLOAD_LINKS} />);

    expect(
      screen.getByRole('heading', {
        name: '후배들의 가능성을 함께 열어주세요.',
      }),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '기부하기' })).toHaveAttribute(
      'href',
      EXTERNAL_DONATION_URL,
    );
  });

  it('does not render fake marketplace links when app URLs are unavailable', () => {
    render(<SupportSection downloadLinks={UNAVAILABLE_DOWNLOAD_LINKS} />);

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
  });
});
