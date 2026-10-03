// noticeOfficialProfile.test — notice editor official-profile checkbox: default on for new posts, stored value on edit, sent on save
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { render, screen, cleanup, waitFor } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, expect, it, vi } from 'vitest';
import { api } from '../api/client';
import { NoticeEditPage } from '../pages/NoticeEditPage';
import type { AdminFeedCategory, NoticeDetail } from '../types/api';

// The rich Markdown editor is not under test; a textarea stands in for it.
vi.mock('../components/editor/MarkdownEditor', () => ({
  MarkdownEditor: ({ value, onChange }: { value: string; onChange: (v: string) => void }) => (
    <textarea aria-label="본문" value={value} onChange={(e) => onChange(e.target.value)} />
  ),
}));

afterEach(() => { cleanup(); vi.restoreAllMocks(); });

const LABEL = '공식 프로필 사용 (대일외고장학회 닉네임+아이콘)';

const CATEGORIES: AdminFeedCategory[] = [
  { seq: 1, code: 'notice', name: '공지', sortOrder: 1, openYn: 'Y', isDefault: 'Y', postCount: 1 },
];

const PERSONAL_NOTICE: NoticeDetail = {
  seq: 9, subject: '장학금 안내', contentHtml: '<p>본문</p>', contentFormat: 'MARKDOWN', contentMd: '본문',
  summary: '본문', thumbnailUrl: null, regDate: '2026-10-01T10:00:00+09:00', regName: '홍길동', hit: 0,
  likeCnt: 0, commentCnt: 0, isPinned: 'N', categorySeq: 1, categoryName: '공지', officialProfile: false, files: [],
};

function renderEditor(path: string) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <MemoryRouter initialEntries={[path]}>
      <QueryClientProvider client={client}>
        <Routes>
          <Route path="/notice/new" element={<NoticeEditPage />} />
          <Route path="/notice/:seq/edit" element={<NoticeEditPage />} />
          <Route path="/notice" element={<p>목록</p>} />
        </Routes>
      </QueryClientProvider>
    </MemoryRouter>,
  );
}

function mockGets(detail?: NoticeDetail) {
  vi.spyOn(api, 'get').mockImplementation((url: string) => {
    if (url === '/api/admin/feed-categories') return Promise.resolve(CATEGORIES);
    return Promise.resolve(detail);
  });
}

it('checks the official profile by default on a new post and sends it on save', async () => {
  mockGets();
  const post = vi.spyOn(api, 'post').mockResolvedValue({ seq: 10 });
  const user = userEvent.setup();
  renderEditor('/notice/new');

  const checkbox = screen.getByRole('checkbox', { name: LABEL });
  expect(checkbox).toBeChecked();

  await user.type(screen.getByRole('textbox', { name: '제목' }), '새 공지');
  await user.type(screen.getByRole('textbox', { name: '본문' }), '내용');
  await waitFor(() => expect(screen.getByRole('button', { name: '저장' })).toBeEnabled());
  await user.click(screen.getByRole('button', { name: '저장' }));

  await waitFor(() => expect(post).toHaveBeenCalledWith('/api/admin/feed', expect.objectContaining({ officialProfile: true })));
});

it('sends officialProfile false when unchecked on a new post', async () => {
  mockGets();
  const post = vi.spyOn(api, 'post').mockResolvedValue({ seq: 10 });
  const user = userEvent.setup();
  renderEditor('/notice/new');

  await user.click(screen.getByRole('checkbox', { name: LABEL }));
  await user.type(screen.getByRole('textbox', { name: '제목' }), '새 공지');
  await user.type(screen.getByRole('textbox', { name: '본문' }), '내용');
  await waitFor(() => expect(screen.getByRole('button', { name: '저장' })).toBeEnabled());
  await user.click(screen.getByRole('button', { name: '저장' }));

  await waitFor(() => expect(post).toHaveBeenCalledWith('/api/admin/feed', expect.objectContaining({ officialProfile: false })));
});

it('reflects the stored value on edit and sends the toggled choice', async () => {
  mockGets(PERSONAL_NOTICE);
  const put = vi.spyOn(api, 'put').mockResolvedValue(undefined);
  const user = userEvent.setup();
  renderEditor('/notice/9/edit');

  expect(await screen.findByDisplayValue('장학금 안내')).toBeInTheDocument();
  const checkbox = screen.getByRole('checkbox', { name: LABEL });
  expect(checkbox).not.toBeChecked();

  await user.click(checkbox);
  await waitFor(() => expect(screen.getByRole('button', { name: '저장' })).toBeEnabled());
  await user.click(screen.getByRole('button', { name: '저장' }));

  await waitFor(() => expect(put).toHaveBeenCalledWith('/api/admin/feed/9', expect.objectContaining({ officialProfile: true })));
});
