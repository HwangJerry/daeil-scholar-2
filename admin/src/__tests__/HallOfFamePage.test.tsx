// HallOfFamePage tests — verifies list toggles, add/edit/delete, and the import preview/commit flow
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { useToast } from '../hooks/useToast.ts';
import { HallOfFamePage } from '../pages/HallOfFamePage.tsx';
import type { HallEntry, HallEntryPage, HallImportPreview } from '../types/hallOfFame.ts';

const ENTRY: HallEntry = {
  id: 7,
  displayName: '허재혁',
  kind: 'alumni',
  cohort: 1,
  department: '독일어과',
  tierManWon: 1000,
  contactName: null,
  roleLabel: null,
  consentPublic: true,
  visible: true,
  source: 'excel',
  createdAt: '2026-11-02T00:00:00+09:00',
  updatedAt: '2026-11-02T00:00:00+09:00',
};

const PAGE: HallEntryPage = {
  items: [ENTRY],
  total: 1,
  page: 1,
  size: 50,
  tierCounts: [{ tierManWon: 1000, count: 1 }, { tierManWon: null, count: 12 }],
};

const PREVIEW: HallImportPreview = {
  previewToken: 'signed-token',
  added: [{ rowIndex: 2, displayName: '김신규', kind: 'alumni', cohort: 15, department: '불어과', tierManWon: 100 }],
  updated: [{
    id: 7,
    displayName: '허재혁',
    changes: [
      { field: 'tierManWon', before: 500, after: 1000 },
      { field: 'roleLabel', before: null, after: '교장' },
      { field: 'phone', before: '01012345678', after: '01099998888' },
    ],
  }],
  unchangedCount: 330,
  alreadyCommitted: false,
  manualSkippedCount: 0,
  removalCandidates: [
    { id: 8, displayName: '이탈자', kind: 'guest', cohort: null, department: null, tierManWon: 300 },
    { id: 9, displayName: '유지자', kind: 'alumni', cohort: 3, department: null, tierManWon: 100 },
  ],
  warnings: [
    { rowIndex: 40, field: 'department', code: 'UNKNOWN_DEPARTMENT', message: '과 약어 "노"를 확인해 주세요.' },
    { rowIndex: 41, field: 'displayName', code: 'RESTRICTED_MATCH', message: '제한 회원과 일치합니다.' },
    { rowIndex: null, field: 'displayName', code: 'POSSIBLE_DUPLICATE', message: '중복일 수 있습니다.' },
    { rowIndex: 42, field: 'displayName', code: 'MANUAL_OVERLAP', message: '수동 항목과 겹칩니다.' },
  ],
  errors: [],
};

type Handler = (url: string, init?: RequestInit) => unknown;

const IMPORT_DISABLED_MESSAGE = '가져오기가 아직 설정되지 않았습니다.';

function importDisabledResponse() {
  return new Response(
    JSON.stringify({ code: 'HALL_OF_FAME_IMPORT_DISABLED', message: IMPORT_DISABLED_MESSAGE }),
    { status: 503, headers: { 'Content-Type': 'application/json' } },
  );
}

function jsonResponse(body: unknown) {
  return new Response(body === undefined ? null : JSON.stringify(body), {
    status: body === undefined ? 204 : 200,
    headers: { 'Content-Type': 'application/json' },
  });
}

function stubApi(handleWrite: Handler = () => undefined) {
  const fetchMock = vi.fn(async (url: string, init?: RequestInit) => {
    if ((init?.method ?? 'GET') === 'GET') return jsonResponse(PAGE);
    const body = handleWrite(url, init);
    return body instanceof Response ? body : jsonResponse(body);
  });
  vi.stubGlobal('fetch', fetchMock);
  return fetchMock;
}

function findWrite(fetchMock: ReturnType<typeof stubApi>, method: string) {
  const call = fetchMock.mock.calls.find(([, init]) => init?.method === method);
  if (!call) throw new Error(`no ${method} request`);
  return { url: call[0], body: call[1]?.body };
}

function renderPage() {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return render(
    <QueryClientProvider client={queryClient}>
      <HallOfFamePage />
    </QueryClientProvider>,
  );
}

async function openPreview(user: ReturnType<typeof userEvent.setup>, container: HTMLElement) {
  await user.click(screen.getByRole('tab', { name: '엑셀 업로드' }));
  const input = container.querySelector<HTMLInputElement>('#hall-panel-upload input[type="file"]');
  await user.upload(input!, new File(['xlsx'], 'donors.xlsx'));
  await user.click(screen.getByRole('button', { name: '미리보기' }));
}

describe('HallOfFamePage list', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('shows the operator count and sends tier, kind, and search filters', async () => {
    const fetchMock = stubApi();
    const user = userEvent.setup();
    renderPage();

    expect(await screen.findByText('허재혁')).toBeInTheDocument();
    expect(screen.getByText('총 1명')).toBeInTheDocument();
    expect(screen.getByText('1,000만원 1명 · 미등재 12명')).toBeInTheDocument();
    expect(String(fetchMock.mock.calls[0][0])).toContain('size=50');

    await user.selectOptions(screen.getByRole('combobox', { name: '등급 필터' }), '300');
    await user.selectOptions(screen.getByRole('combobox', { name: '종류 필터' }), 'org');
    await user.type(screen.getByRole('textbox', { name: '이름 또는 기수 검색' }), '15');

    await waitFor(() => {
      const lastUrl = String(fetchMock.mock.lastCall?.[0]);
      expect(lastUrl).toContain('tier=300');
      expect(lastUrl).toContain('kind=org');
      expect(lastUrl).toContain('q=15');
    });
  });

  it('requests unlisted entries and clears the tier when all tiers are selected', async () => {
    const fetchMock = stubApi();
    const user = userEvent.setup();
    renderPage();

    const tierFilter = screen.getByRole('combobox', { name: '등급 필터' });
    await user.selectOptions(tierFilter, 'none');
    await waitFor(() => expect(String(fetchMock.mock.lastCall?.[0])).toContain('tier=none'));
    expect(tierFilter).toHaveValue('none');

    await user.selectOptions(tierFilter, '');
    await waitFor(() => expect(String(fetchMock.mock.lastCall?.[0])).not.toContain('tier='));
    expect(tierFilter).toHaveValue('');
  });

  it('patches only the toggled field', async () => {
    const fetchMock = stubApi(() => ENTRY);
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole('switch', { name: '허재혁 노출' }));
    await waitFor(() => expect(findWrite(fetchMock, 'PATCH').url).toBe('/api/admin/hall-of-fame/7'));
    expect(JSON.parse(String(findWrite(fetchMock, 'PATCH').body))).toEqual({ visible: false });
  });

  it('turns public consent off for one entry', async () => {
    const fetchMock = stubApi(() => ENTRY);
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole('switch', { name: '허재혁 공개 동의' }));
    await waitFor(() => {
      expect(JSON.parse(String(findWrite(fetchMock, 'PATCH').body))).toEqual({ consentPublic: false });
    });
  });

  it('edits an entry, clearing an emptied field with null', async () => {
    const fetchMock = stubApi(() => ENTRY);
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole('button', { name: '허재혁 수정' }));
    const dialog = await screen.findByRole('dialog');
    await user.clear(within(dialog).getByLabelText('과'));
    await user.selectOptions(within(dialog).getByLabelText('등급'), '500');
    await user.click(within(dialog).getByRole('button', { name: '수정 저장' }));

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(JSON.parse(String(findWrite(fetchMock, 'PATCH').body))).toEqual({
      displayName: '허재혁',
      kind: 'alumni',
      cohort: 1,
      department: null,
      tierManWon: 500,
      contactName: null,
      roleLabel: null,
    });
  });

  it('sends a role label only for teachers and hides the input for other kinds', async () => {
    const fetchMock = stubApi(() => ENTRY);
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole('button', { name: '허재혁 수정' }));
    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).queryByLabelText('직함(공개 라벨)')).not.toBeInTheDocument();

    await user.selectOptions(within(dialog).getByLabelText('종류'), 'teacher');
    await user.type(within(dialog).getByLabelText('직함(공개 라벨)'), '교장');
    await user.click(within(dialog).getByRole('button', { name: '수정 저장' }));

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(JSON.parse(String(findWrite(fetchMock, 'PATCH').body))).toMatchObject({
      kind: 'teacher',
      roleLabel: '교장',
    });
  });

  it('clears a stored role label when the entry stops being a teacher', async () => {
    const teacher: HallEntry = { ...ENTRY, kind: 'teacher', roleLabel: '교장' };
    const fetchMock = vi.fn(async (_url: string, init?: RequestInit) => (
      jsonResponse((init?.method ?? 'GET') === 'GET' ? { ...PAGE, items: [teacher] } : teacher)
    ));
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole('button', { name: '허재혁 수정' }));
    const dialog = await screen.findByRole('dialog');
    expect(within(dialog).getByLabelText('직함(공개 라벨)')).toHaveValue('교장');
    await user.selectOptions(within(dialog).getByLabelText('종류'), 'alumni');
    await user.click(within(dialog).getByRole('button', { name: '수정 저장' }));

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    const patch = fetchMock.mock.calls.find(([, init]) => init?.method === 'PATCH');
    expect(JSON.parse(String(patch?.[1]?.body))).toMatchObject({ roleLabel: null });
  });

  it('adds an organization with a private contact name and blocks an empty name', async () => {
    const fetchMock = stubApi(() => ENTRY);
    const user = userEvent.setup();
    renderPage();

    await screen.findByText('허재혁');
    await user.click(screen.getByRole('button', { name: '추가' }));
    const dialog = await screen.findByRole('dialog');
    await user.click(within(dialog).getByRole('button', { name: '추가' }));
    expect(within(dialog).getByRole('alert')).toHaveTextContent('이름을 입력해 주세요.');
    expect(fetchMock.mock.calls.some(([, init]) => init?.method === 'POST')).toBe(false);

    await user.type(within(dialog).getByLabelText('이름'), '주식회사 대일');
    await user.selectOptions(within(dialog).getByLabelText('종류'), 'org');
    await user.type(within(dialog).getByLabelText('담당자 (비공개)'), '홍길동');
    await user.click(within(dialog).getByRole('button', { name: '추가' }));

    await waitFor(() => expect(screen.queryByRole('dialog')).not.toBeInTheDocument());
    expect(JSON.parse(String(findWrite(fetchMock, 'POST').body))).toEqual({
      displayName: '주식회사 대일',
      kind: 'org',
      cohort: null,
      department: null,
      tierManWon: null,
      contactName: '홍길동',
      roleLabel: null,
      visible: true,
      consentPublic: true,
    });
  });

  it('soft deletes only after confirmation', async () => {
    const fetchMock = stubApi();
    const user = userEvent.setup();
    renderPage();

    await user.click(await screen.findByRole('button', { name: '허재혁 삭제' }));
    const dialog = await screen.findByRole('alertdialog');
    expect(fetchMock.mock.calls.some(([, init]) => init?.method === 'DELETE')).toBe(false);
    await user.click(within(dialog).getByRole('button', { name: '삭제' }));

    await waitFor(() => expect(findWrite(fetchMock, 'DELETE').url).toBe('/api/admin/hall-of-fame/7'));
  });
});

describe('HallOfFamePage upload', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('commits the preview token with only the removal candidates the operator checked', async () => {
    const fetchMock = stubApi((url) => (
      url.endsWith('/import/preview')
        ? PREVIEW
        : { addedCount: 1, updatedCount: 1, removedCount: 1 }
    ));
    const user = userEvent.setup();
    const { container } = renderPage();

    await openPreview(user, container);

    expect(await screen.findByText('변경 없음 330')).toBeInTheDocument();
    expect(screen.getByText('추가 1')).toBeInTheDocument();
    expect(screen.getByText('김신규')).toBeInTheDocument();
    expect(screen.getByText(/등급: 500만원/)).toHaveTextContent('등급: 500만원 → 1,000만원');
    expect(screen.getByText(/직함\(공개 라벨\):/)).toHaveTextContent('직함(공개 라벨): — → 교장');
    expect(screen.getByText(/과 약어 "노"/)).toBeInTheDocument();
    expect(screen.getByText('41행 제한 회원과 일치합니다.')).toBeInTheDocument();
    expect(screen.getByText('중복일 수 있습니다.')).toBeInTheDocument();
    expect(screen.getByText('42행 수동 항목과 겹칩니다.')).toBeInTheDocument();
    expect(screen.getByRole('checkbox', { name: '이탈자 삭제' })).not.toBeChecked();

    await user.click(screen.getByRole('checkbox', { name: '이탈자 삭제' }));
    await user.click(screen.getByRole('button', { name: '반영' }));
    const dialog = await screen.findByRole('alertdialog');
    expect(dialog).toHaveTextContent('추가 1건, 수정 1건, 삭제 1건을 반영합니다.');
    await user.click(within(dialog).getByRole('button', { name: '반영' }));

    expect(await screen.findByRole('status')).toHaveTextContent('추가 1건, 수정 1건, 삭제 1건');
    const commit = fetchMock.mock.calls.find(([url]) => url.endsWith('/import/commit'));
    expect(JSON.parse(String(commit?.[1]?.body))).toEqual({
      previewToken: 'signed-token',
      removeIds: [8],
    });
    expect(screen.queryByRole('checkbox', { name: '이탈자 삭제' })).not.toBeInTheDocument();
  });

  it('never renders phone values, even if a diff contains them', async () => {
    stubApi(() => PREVIEW);
    const user = userEvent.setup();
    const { container } = renderPage();

    await openPreview(user, container);

    await screen.findByText('변경 없음 330');
    expect(container).not.toHaveTextContent('01012345678');
    expect(container).not.toHaveTextContent('01099998888');
  });

  it('shows a quiet notice instead of an error toast when preview is disabled', async () => {
    stubApi(() => importDisabledResponse());
    const addToast = vi.fn();
    useToast.setState({ addToast });
    const user = userEvent.setup();
    const { container } = renderPage();

    await openPreview(user, container);

    const notice = await screen.findByText(IMPORT_DISABLED_MESSAGE);
    expect(notice).toHaveClass('text-cool-gray');
    expect(addToast).not.toHaveBeenCalled();
  });

  it('shows the same notice when commit is disabled and keeps the preview', async () => {
    stubApi((url) => (url.endsWith('/import/preview') ? PREVIEW : importDisabledResponse()));
    const addToast = vi.fn();
    useToast.setState({ addToast });
    const user = userEvent.setup();
    const { container } = renderPage();

    await openPreview(user, container);
    await user.click(await screen.findByRole('button', { name: '반영' }));
    await user.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: '반영' }));

    expect(await screen.findByText(IMPORT_DISABLED_MESSAGE)).toHaveClass('text-cool-gray');
    expect(addToast).not.toHaveBeenCalled();
    expect(screen.getByText('변경 없음 330')).toBeInTheDocument();
  });

  it('blocks commit while the preview has errors', async () => {
    stubApi(() => ({ ...PREVIEW, previewToken: '', errors: [{ rowIndex: 5, message: '등급 값을 알 수 없습니다.' }] }));
    const user = userEvent.setup();
    const { container } = renderPage();

    await openPreview(user, container);

    expect(await screen.findByText(/등급 값을 알 수 없습니다/)).toHaveTextContent('5행');
    expect(screen.getByRole('button', { name: '반영' })).toBeDisabled();
  });

  it('explains an already committed file and protected manual entries', async () => {
    stubApi(() => ({ ...PREVIEW, alreadyCommitted: true, manualSkippedCount: 2 }));
    const user = userEvent.setup();
    const { container } = renderPage();

    await openPreview(user, container);

    expect(await screen.findByText('이미 반영한 파일입니다. 현재 명단과 비교한 결과를 확인해 주세요.')).toBeInTheDocument();
    expect(screen.getByText('수동 수정 보호 2')).toBeInTheDocument();
    expect(screen.getByText('수동 수정한 항목은 덮어쓰거나 삭제하지 않습니다.')).toBeInTheDocument();
    // A repeated file can still have a diff after other imports; the flag is informational.
    expect(screen.getByRole('button', { name: '반영' })).toBeEnabled();
  });

  it('blocks commit when the file changes nothing', async () => {
    stubApi(() => ({ ...PREVIEW, added: [], updated: [], removalCandidates: [] }));
    const user = userEvent.setup();
    const { container } = renderPage();

    await openPreview(user, container);

    expect(await screen.findByText('반영할 변경이 없습니다.')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '반영' })).toBeDisabled();
  });
});
