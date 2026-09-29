// noticeCategory.test — notice editor category select options and the notice list category filter
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { render, screen, cleanup, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, expect, it, vi } from 'vitest';
import { api } from '../api/client';
import { NoticeCategorySelect } from '../components/notice/NoticeCategorySelect';
import { defaultCategorySeq, noticeCategoryOptions } from '../components/notice/noticeCategoryOptions';
import { NoticeListPage } from '../pages/NoticeListPage';
import type { AdminFeedCategory } from '../types/api';

afterEach(() => { cleanup(); vi.restoreAllMocks(); });

const CATEGORIES: AdminFeedCategory[] = [
  { seq: 3, code: 'c3', name: '장학', sortOrder: 1, openYn: 'Y', isDefault: 'N', postCount: 1 },
  { seq: 1, code: 'notice', name: '공지', sortOrder: 2, openYn: 'Y', isDefault: 'Y', postCount: 4 },
  { seq: 2, code: 'etc', name: '기타', sortOrder: 3, openYn: 'N', isDefault: 'N', postCount: 2 },
];

function withProviders(ui: React.ReactNode, path = '/') {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(<MemoryRouter initialEntries={[path]}><QueryClientProvider client={client}>{ui}</QueryClientProvider></MemoryRouter>);
}

it('offers open categories in order plus the post’s own hidden one', () => {
  expect(noticeCategoryOptions(CATEGORIES, null)).toEqual([{ seq: 3, label: '장학' }, { seq: 1, label: '공지' }]);
  expect(noticeCategoryOptions(CATEGORIES, 2)).toEqual([
    { seq: 3, label: '장학' }, { seq: 1, label: '공지' }, { seq: 2, label: '기타 (숨김)' },
  ]);
  expect(defaultCategorySeq(CATEGORIES)).toBe(1);
});

it('ends the select with 카테고리 관리, which opens the manager in a new tab', async () => {
  vi.spyOn(api, 'get').mockResolvedValue(CATEGORIES);
  const open = vi.spyOn(window, 'open').mockReturnValue(null);
  const onChange = vi.fn();
  const user = userEvent.setup();
  withProviders(<NoticeCategorySelect value={1} savedSeq={null} onChange={onChange} />);
  const select = await screen.findByRole('combobox', { name: '카테고리' });
  await waitFor(() => expect(within(select).getAllByRole('option').map((o) => o.textContent)).toEqual(['장학', '공지', '카테고리 관리']));
  await user.selectOptions(select, '카테고리 관리');
  expect(open).toHaveBeenCalledWith('/feed-categories', '_blank', 'noopener');
  expect(onChange).not.toHaveBeenCalled();
  await user.selectOptions(select, '장학');
  expect(onChange).toHaveBeenCalledWith(3);
});

it('filters the notice list by the category kept in the URL and shows the category column', async () => {
  const get = vi.spyOn(api, 'get').mockImplementation((url: string) => {
    if (url === '/api/admin/feed-categories') return Promise.resolve(CATEGORIES);
    return Promise.resolve({
      items: [{ seq: 9, subject: '장학금 안내', regDate: '2026-09-29T10:00:00+09:00', regName: '관리자', hit: 0, openYn: 'Y', isPinned: 'N', contentFormat: 'MARKDOWN', categorySeq: 3, categoryName: '장학' }],
      total: 1,
    });
  });
  const user = userEvent.setup();
  withProviders(<NoticeListPage />, '/notice?category=3');
  expect(await screen.findByText('장학금 안내')).toBeInTheDocument();
  expect(get).toHaveBeenCalledWith('/api/admin/feed?page=1&size=20&category=3');
  expect(screen.getByRole('columnheader', { name: /카테고리/ })).toBeInTheDocument();

  const filter = screen.getByRole('combobox', { name: '카테고리 필터' }) as HTMLSelectElement;
  await waitFor(() => expect(filter.value).toBe('3'));
  expect(within(filter).getByRole('option', { name: '기타 (숨김)' })).toBeInTheDocument();
  await user.selectOptions(filter, '카테고리: 전체');
  await waitFor(() => expect(get).toHaveBeenCalledWith('/api/admin/feed?page=1&size=20'));
});
