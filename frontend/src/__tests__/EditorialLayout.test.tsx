// EditorialLayout.test — Shared public-site chrome and home-anchor contracts
import { render, screen, within } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import { EditorialLayout } from '../components/layout/EditorialLayout';

vi.mock('../components/landing/LandingFooter', () => ({
  LandingFooter: () => <footer>공통 푸터</footer>,
}));

describe('EditorialLayout', () => {
  it('uses the current site navigation with home-qualified section links', () => {
    render(
      <MemoryRouter initialEntries={['/disclosure']}>
        <Routes>
          <Route element={<EditorialLayout />}>
            <Route path="/disclosure" element={<h1>의무공시</h1>} />
          </Route>
        </Routes>
      </MemoryRouter>,
    );

    const navigation = screen.getByRole('navigation', { name: '주요 메뉴' });
    expect(within(navigation).getByRole('link', { name: '다운로드' })).toHaveAttribute(
      'href',
      '/#download',
    );
    expect(within(navigation).getByRole('link', { name: '최근 소식' })).toHaveAttribute(
      'href',
      '/#news',
    );
    expect(within(navigation).getByRole('link', { name: '장학회 소개' })).toHaveAttribute(
      'href',
      '/#about',
    );
    expect(screen.getByRole('link', { name: '본문으로 건너뛰기' })).toHaveAttribute(
      'href',
      '#main-content',
    );
    expect(screen.getByRole('contentinfo')).toHaveTextContent('공통 푸터');
  });
});
