// DisclosureDetailPage.test — Editorial document metadata and attachment-download contracts
import { render, screen } from '@testing-library/react';
import { MemoryRouter, Route, Routes } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useDisclosureDetail } from '../hooks/useDisclosureDetail';
import { DisclosureDetailPage } from '../pages/DisclosureDetailPage';
import type { NoticeDetail } from '../types/api';

vi.mock('../components/seo/PageMeta', () => ({ PageMeta: () => null }));
vi.mock('../hooks/useDisclosureDetail');

const DISCLOSURE_DETAIL: NoticeDetail = {
  seq: 188,
  subject: '2025 공시 자료',
  contentHtml: '<p>2025 공시 자료 게시합니다.</p>',
  contentFormat: 'MARKDOWN',
  summary: '2025년도 공시 문서입니다.',
  thumbnailUrl: null,
  regDate: '2026-04-30T00:00:00Z',
  regName: '장학회',
  hit: 12,
  likeCnt: 0,
  commentCnt: 0,
  userLiked: false,
  files: [
    {
      fSeq: 1,
      fGate: 'DISCLOSURE',
      fJoinSeq: 188,
      typeName: 'notice-attachment',
      fileName: 'financial-statement.pdf',
      fileSize: '152371',
      filePath: '/uploads/notice-attachment',
      fileOrgName: '재무상태표.pdf',
      openYn: 'Y',
    },
  ],
};

function renderPage() {
  return render(
    <MemoryRouter initialEntries={['/disclosure/188']}>
      <Routes>
        <Route path="/disclosure/:seq" element={<DisclosureDetailPage />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe('DisclosureDetailPage', () => {
  beforeEach(() => {
    vi.mocked(useDisclosureDetail).mockReturnValue({
      data: DISCLOSURE_DETAIL,
      isLoading: false,
      isError: false,
    } as ReturnType<typeof useDisclosureDetail>);
  });

  it('presents the disclosure as a readable document with explicit downloads', () => {
    renderPage();

    expect(screen.getByRole('heading', { level: 1, name: '2025 공시 자료' })).toBeInTheDocument();
    expect(screen.getByText('2025 공시 자료 게시합니다.')).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: '첨부 문서' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '재무상태표.pdf 다운로드' })).toHaveAttribute(
      'href',
      '/uploads/notice-attachment/financial-statement.pdf',
    );
  });

  it('keeps a clear route back to the disclosure archive', () => {
    renderPage();

    for (const link of screen.getAllByRole('link', { name: /의무공시 목록/ })) {
      expect(link).toHaveAttribute('href', '/disclosure');
    }
  });

  it('renders a recovery path when the disclosure is unavailable', () => {
    vi.mocked(useDisclosureDetail).mockReturnValue({
      data: undefined,
      isLoading: false,
      isError: true,
    } as ReturnType<typeof useDisclosureDetail>);
    renderPage();

    expect(
      screen.getByRole('heading', { level: 1, name: '공시 자료를 찾을 수 없습니다' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('link', { name: '의무공시 목록' })).toHaveAttribute(
      'href',
      '/disclosure',
    );
  });
});
