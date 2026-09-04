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
  it('renders coming-soon controls without fake marketplace links when URLs are unset', () => {
    render(<AppDownloadHero downloadLinks={UNAVAILABLE_DOWNLOAD_LINKS} />);

    expect(
      screen.getByRole('button', {
        name: 'App Store 출시 준비 중 안내 보기',
      }),
    ).toBeEnabled();
    expect(
      screen.getByRole('button', {
        name: 'Google Play 출시 준비 중 안내 보기',
      }),
    ).toBeEnabled();
    expect(screen.queryByText('출시 준비 중')).not.toBeInTheDocument();
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

  it('keeps the hero focused on a single text column without a phone preview', () => {
    render(<AppDownloadHero downloadLinks={UNAVAILABLE_DOWNLOAD_LINKS} />);

    expect(
      screen.getByRole('heading', { name: '대일의 오늘과 내일을 잇습니다.' }),
    ).toBeInTheDocument();
    expect(screen.queryByRole('figure')).not.toBeInTheDocument();
    expect(screen.queryAllByRole('img')).toHaveLength(0);
  });

  it('layers an enlarged mobile globe behind the hero content', () => {
    const { container } = render(
      <AppDownloadHero downloadLinks={UNAVAILABLE_DOWNLOAD_LINKS} />,
    );
    const globe = container.querySelector('canvas')?.parentElement;
    const heading = screen.getByRole('heading', {
      name: '대일의 오늘과 내일을 잇습니다.',
    });

    expect(globe).toHaveClass(
      'absolute',
      'z-0',
      'size-[440px]',
      'sm:size-[500px]',
      'md:relative',
    );
    expect(heading.parentElement).toHaveClass('relative', 'z-10');
  });
});
