// FeedCategoryPage.test — default-row guards, 6-category cap, inline server errors and move-then-delete
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter } from 'react-router-dom';
import { render, screen, cleanup, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, expect, it, vi } from 'vitest';
import { api, ApiClientError } from '../api/client';
import { FeedCategoryPage } from '../pages/FeedCategoryPage';
import type { AdminFeedCategory } from '../types/api';

afterEach(() => { cleanup(); vi.restoreAllMocks(); });

const category = (seq: number, name: string, extra: Partial<AdminFeedCategory> = {}): AdminFeedCategory => ({
  seq, code: seq === 1 ? 'notice' : `c${seq}`, name, sortOrder: seq, openYn: 'Y', isDefault: 'N', postCount: 0, ...extra,
});

const SEEDED = [
  category(1, '공지', { isDefault: 'Y', postCount: 12 }),
  category(2, '기타', { code: 'etc', postCount: 3 }),
  category(3, '행사', { openYn: 'N' }),
];

function renderPage() {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false }, mutations: { retry: false } } });
  return render(<MemoryRouter><QueryClientProvider client={client}><FeedCategoryPage /></QueryClientProvider></MemoryRouter>);
}

function rowOf(name: string) {
  return screen.getByText(name).closest('tr') as HTMLElement;
}

it('guards the default row and shows the count against the maximum', async () => {
  vi.spyOn(api, 'get').mockResolvedValue(SEEDED);
  renderPage();
  expect(await screen.findByText('공지')).toBeInTheDocument();
  expect(screen.getByText('(3 / 최대 6개)')).toBeInTheDocument();

  const defaultRow = rowOf('공지');
  expect(within(defaultRow).getByText('기본')).toBeInTheDocument();
  expect(within(defaultRow).getByRole('button', { name: '기본 카테고리는 삭제할 수 없습니다' })).toBeDisabled();
  expect(within(defaultRow).getByRole('switch')).toBeDisabled();
  expect(within(rowOf('행사')).getByRole('switch')).toHaveAttribute('aria-checked', 'false');
  expect(within(rowOf('기타')).getByRole('button', { name: '삭제' })).toBeEnabled();
  expect(screen.getByRole('button', { name: '추가' })).toBeEnabled();
});

it('disables adding at six categories', async () => {
  vi.spyOn(api, 'get').mockResolvedValue([...SEEDED, category(4, '장학'), category(5, '동문'), category(6, '모집')]);
  renderPage();
  expect(await screen.findByText('(6 / 최대 6개)')).toBeInTheDocument();
  expect(screen.getByRole('button', { name: '추가' })).toBeDisabled();
});

it('shows the server refusal under the table as-is', async () => {
  vi.spyOn(api, 'get').mockResolvedValue(SEEDED);
  const post = vi.spyOn(api, 'post').mockRejectedValue(new ApiClientError(400, 'DUPLICATE_NAME', '이미 있는 카테고리 이름입니다.'));
  const user = userEvent.setup();
  renderPage();
  await user.click(await screen.findByRole('button', { name: '추가' }));
  await user.type(screen.getByLabelText('카테고리 이름'), ' 기타 ');
  await user.click(screen.getByRole('button', { name: '저장' }));
  await waitFor(() => expect(post).toHaveBeenCalledWith('/api/admin/feed-categories', { name: '기타', openYn: 'Y' }));
  expect(await screen.findByRole('alert')).toHaveTextContent('이미 있는 카테고리 이름입니다.');
});

it('toggles app tab visibility with the current name', async () => {
  vi.spyOn(api, 'get').mockResolvedValue(SEEDED);
  const put = vi.spyOn(api, 'put').mockResolvedValue(category(3, '행사'));
  const user = userEvent.setup();
  renderPage();
  await user.click(within(await screen.findByText('행사').then(() => rowOf('행사'))).getByRole('switch'));
  await waitFor(() => expect(put).toHaveBeenCalledWith('/api/admin/feed-categories/3', { name: '행사', openYn: 'Y' }));
});

it('deletes an empty category after a plain confirm', async () => {
  vi.spyOn(api, 'get').mockResolvedValue(SEEDED);
  const del = vi.spyOn(api, 'del').mockResolvedValue(undefined);
  const user = userEvent.setup();
  renderPage();
  await screen.findByText('행사');
  await user.click(within(rowOf('행사')).getByRole('button', { name: '삭제' }));
  const dialog = screen.getByRole('alertdialog');
  expect(within(dialog).queryByLabelText('옮길 카테고리')).toBeNull();
  await user.click(within(dialog).getByRole('button', { name: '삭제' }));
  await waitFor(() => expect(del).toHaveBeenCalledWith('/api/admin/feed-categories/3', undefined));
});

it('moves posts to the chosen category before deleting', async () => {
  vi.spyOn(api, 'get').mockResolvedValue(SEEDED);
  const del = vi.spyOn(api, 'del').mockResolvedValue(undefined);
  const user = userEvent.setup();
  renderPage();
  await screen.findByText('기타');
  await user.click(within(rowOf('기타')).getByRole('button', { name: '삭제' }));
  const dialog = screen.getByRole('alertdialog');
  expect(within(dialog).getByText("'기타' 카테고리 삭제")).toBeInTheDocument();
  expect(within(dialog).getByText(/게시글 3개/)).toBeInTheDocument();
  const target = within(dialog).getByLabelText('옮길 카테고리') as HTMLSelectElement;
  expect(target.value).toBe('1'); // default preselected
  expect(within(target).queryByRole('option', { name: '기타' })).toBeNull();
  await user.selectOptions(target, '3');
  await user.click(within(dialog).getByRole('button', { name: '옮기고 삭제' }));
  await waitFor(() => expect(del).toHaveBeenCalledWith('/api/admin/feed-categories/2', { moveToSeq: 3 }));
});

it('asks for a target when posts appeared after the list loaded', async () => {
  vi.spyOn(api, 'get').mockResolvedValue(SEEDED);
  vi.spyOn(api, 'del').mockRejectedValueOnce(
    new ApiClientError(409, 'CATEGORY_HAS_POSTS', '게시글이 있는 카테고리는 옮길 카테고리를 선택해야 삭제할 수 있습니다.', [], { postCount: 2 }),
  );
  const user = userEvent.setup();
  renderPage();
  await screen.findByText('행사');
  await user.click(within(rowOf('행사')).getByRole('button', { name: '삭제' }));
  await user.click(within(screen.getByRole('alertdialog')).getByRole('button', { name: '삭제' }));
  expect(await screen.findByText(/게시글 2개/)).toBeInTheDocument();
  expect(screen.getByLabelText('옮길 카테고리')).toBeInTheDocument();
  expect(screen.queryByRole('alert')).toBeNull();
});
