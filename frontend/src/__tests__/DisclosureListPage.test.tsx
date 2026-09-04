// DisclosureListPage.test — Editorial archive content and complete query-state coverage
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { MemoryRouter } from 'react-router-dom';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useDisclosureList } from '../hooks/useDisclosureList';
import { DisclosureListPage } from '../pages/DisclosureListPage';
import type { DisclosureItem } from '../types/api';

vi.mock('../components/seo/PageMeta', () => ({ PageMeta: () => null }));
vi.mock('../hooks/useDisclosureList');

const DISCLOSURE: DisclosureItem = {
  seq: 188,
  subject: '2025 공시 자료',
  summary: '2025년도 공시 문서입니다.',
  regDate: '2026-04-30T00:00:00Z',
  regName: '장학회',
  hit: 12,
  attachmentCount: 6,
};

const loadMore = vi.fn();
const refetch = vi.fn();

function mockDisclosureList(overrides: Record<string, unknown> = {}) {
  vi.mocked(useDisclosureList).mockReturnValue({
    items: [DISCLOSURE],
    hasMore: false,
    isError: false,
    isFetching: false,
    isLoading: false,
    loadMore,
    refetch,
    ...overrides,
  } as ReturnType<typeof useDisclosureList>);
}

function renderPage() {
  return render(
    <MemoryRouter>
      <DisclosureListPage />
    </MemoryRouter>,
  );
}

describe('DisclosureListPage', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockDisclosureList();
  });

  it('renders public disclosures as an editorial archive instead of a table', () => {
    renderPage();

    expect(screen.getByRole('heading', { level: 1, name: '의무공시' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: '연도별 공개 자료' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: /2025 공시 자료/ })).toHaveAttribute(
      'href',
      '/disclosure/188',
    );
    expect(screen.getByText('6건')).toBeInTheDocument();
    expect(screen.queryByRole('table')).not.toBeInTheDocument();
  });

  it('renders a labeled loading state', () => {
    mockDisclosureList({ items: [], isFetching: true, isLoading: true });
    renderPage();

    expect(screen.getByRole('status', { name: '의무공시 목록 불러오는 중' })).toBeInTheDocument();
  });

  it('offers a retry action when the initial request fails', async () => {
    const user = userEvent.setup();
    mockDisclosureList({ items: [], isError: true });
    renderPage();

    expect(screen.getByText('공시 자료를 불러오지 못했습니다')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: '다시 시도' }));
    expect(refetch).toHaveBeenCalledOnce();
  });
});
