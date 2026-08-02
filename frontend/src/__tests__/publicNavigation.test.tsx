// publicNavigation.test — Public parent navigation visibility and route-group state contracts
import { render, screen, within } from '@testing-library/react';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it, vi } from 'vitest';
import TopNav from '../components/layout/TopNav';
import BottomNav from '../components/layout/BottomNav';

vi.mock('../hooks/useUnreadMessages', () => ({
  useUnreadMessages: () => ({ unreadCount: 3 }),
}));

describe('public MVP navigation', () => {
  it('exposes only notices and foundation information', () => {
    render(
      <MemoryRouter>
        <TopNav />
        <BottomNav />
      </MemoryRouter>,
    );

    for (const navigation of screen.getAllByRole('navigation')) {
      expect(within(navigation).getAllByRole('link')).toHaveLength(2);
      expect(within(navigation).getByRole('link', { name: '소식' })).toHaveAttribute('href', '/');
      expect(within(navigation).getByRole('link', { name: '장학회 소개' })).toHaveAttribute('href', '/about');
      for (const forbidden of ['누적 기부액', '동문찾기', '쪽지', '마이페이지', 'MY', '로그인']) {
        expect(within(navigation).queryByText(forbidden)).not.toBeInTheDocument();
      }
    }
  });

  it.each([
    ['/post/42', '소식', '장학회 소개'],
    ['/vision', '장학회 소개', '소식'],
    ['/disclosure/42', '장학회 소개', '소식'],
  ])('keeps the parent navigation active at %s', (pathname, activeLabel, inactiveLabel) => {
    render(
      <MemoryRouter initialEntries={[pathname]}>
        <TopNav />
        <BottomNav />
      </MemoryRouter>,
    );

    for (const navigation of screen.getAllByRole('navigation')) {
      expect(within(navigation).getByRole('link', { name: activeLabel })).toHaveAttribute(
        'aria-current',
        'page',
      );
      expect(within(navigation).getByRole('link', { name: inactiveLabel })).not.toHaveAttribute(
        'aria-current',
      );
    }
  });
});
