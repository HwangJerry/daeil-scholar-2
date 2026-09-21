// NotificationTemplatesPage.test — channel sections, live counters, pre-validation, and saves
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { cleanup, render, screen, waitFor, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, expect, it, vi } from 'vitest';
import { ApiClientError, api } from '../api/client';
import { ToastProvider } from '../components/ui/Toast';
import { useToast } from '../hooks/useToast';
import { NotificationTemplatesPage } from '../pages/NotificationTemplatesPage';
import type { NotificationTemplateView } from '../types/notificationTemplate';

afterEach(() => {
  cleanup();
  useToast.setState({ toasts: [] });
  vi.restoreAllMocks();
});

const SMS_KEY = 'sms.phone_verification';
const SMS_BODY = '[대일외고장학회] 인증번호 {code} 를 입력해 주세요.';

const SMS_TEMPLATE: NotificationTemplateView = {
  key: SMS_KEY,
  channel: 'sms',
  displayName: '휴대폰 인증번호 문자',
  description: '회원가입 시 발송되는 인증번호 문자입니다.',
  title: '',
  body: SMS_BODY,
  defaultTitle: '',
  defaultBody: SMS_BODY,
  isDefault: true,
  invalid: false,
  version: 3,
  updatedAt: null,
  updatedBy: null,
  placeholders: [{ name: 'code', required: true, description: '6자리 인증번호', sample: '000000' }],
  limits: { maxBodyEucKrBytes: 80 },
};

const PUSH_TEMPLATE: NotificationTemplateView = {
  key: 'push.notice.new',
  channel: 'push',
  displayName: '새 공지 푸시',
  description: '새 공지사항이 등록되었을 때 발송됩니다.',
  title: '새 소식',
  body: '{subject}',
  defaultTitle: '새 소식',
  defaultBody: '{subject}',
  isDefault: false,
  invalid: false,
  version: 2,
  updatedAt: '2026-09-16T10:00:00Z',
  updatedBy: 7,
  placeholders: [{ name: 'subject', required: true, description: '공지 제목', sample: '공지 제목' }],
  limits: { maxTitleRunes: 50, maxBodyRunes: 200 },
};

function stubList(templates: NotificationTemplateView[] = [SMS_TEMPLATE, PUSH_TEMPLATE]) {
  return vi.spyOn(api, 'get').mockResolvedValue(templates as never);
}

function mount() {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false }, mutations: { retry: false } },
  });
  return render(
    <QueryClientProvider client={client}>
      <ToastProvider>
        <NotificationTemplatesPage />
      </ToastProvider>
    </QueryClientProvider>,
  );
}

async function cardFor(displayName: string) {
  return (await screen.findByRole('heading', { name: displayName })).closest(
    'article',
  ) as HTMLElement;
}

async function replaceBody(card: HTMLElement, user: ReturnType<typeof userEvent.setup>, text: string) {
  const body = within(card).getByLabelText('본문');
  await user.clear(body);
  await user.click(body);
  await user.paste(text);
  return body;
}

it('lists each template under its channel section', async () => {
  stubList();
  mount();

  const smsSection = (await screen.findByRole('region', { name: '문자 (SMS)' })) as HTMLElement;
  const pushSection = screen.getByRole('region', { name: '푸시 알림' });

  expect(within(smsSection).getByRole('heading', { name: '휴대폰 인증번호 문자' })).toBeInTheDocument();
  expect(within(smsSection).queryByLabelText('제목')).not.toBeInTheDocument();
  expect(within(pushSection).getByRole('heading', { name: '새 공지 푸시' })).toBeInTheDocument();
  expect(within(pushSection).getByLabelText('제목')).toHaveValue('새 소식');
  expect(within(smsSection).getByText('기본값 사용 중')).toBeInTheDocument();
});

it('turns the sms byte counter red once the filled-in text passes the reported byte cap', async () => {
  stubList();
  const user = userEvent.setup();
  mount();

  const card = await cardFor('휴대폰 인증번호 문자');
  const counter = within(card).getByTestId('counter-본문');
  expect(counter).toHaveAttribute('data-over-limit', 'false');

  await replaceBody(card, user, `{code} ${'가'.repeat(50)}`);

  // 6 ascii sample bytes + a space + 100 bytes of Hangul.
  expect(counter).toHaveTextContent('107 / 80 바이트');
  expect(counter).toHaveAttribute('data-over-limit', 'true');
  expect(counter.className).toContain('text-error-text');
});

it('blocks a save that drops the required placeholder', async () => {
  stubList();
  const put = vi.spyOn(api, 'put');
  const user = userEvent.setup();
  mount();

  const card = await cardFor('휴대폰 인증번호 문자');
  await replaceBody(card, user, '[대일외고장학회] 인증번호를 입력해 주세요.');

  expect(
    within(card).getByText('본문에 {code} 을(를) 반드시 포함해야 합니다'),
  ).toBeInTheDocument();
  expect(within(card).getByRole('button', { name: '저장' })).toBeDisabled();
  expect(put).not.toHaveBeenCalled();
});

it('blocks a brace that is not part of a placeholder token', async () => {
  stubList();
  const user = userEvent.setup();
  mount();

  const card = await cardFor('휴대폰 인증번호 문자');
  await replaceBody(card, user, '[대일외고장학회] 인증번호 {{code}}');

  expect(within(card).getByText('중괄호 { } 는 치환 항목에만 사용할 수 있습니다')).toBeInTheDocument();
  expect(within(card).getByRole('button', { name: '저장' })).toBeDisabled();
});

it('blocks a misspelled placeholder token', async () => {
  stubList();
  const user = userEvent.setup();
  mount();

  const card = await cardFor('휴대폰 인증번호 문자');
  await replaceBody(card, user, '[대일외고장학회] 인증번호 {code} {cod}');

  expect(within(card).getByText('{cod} 은(는) 사용할 수 없는 치환 항목입니다')).toBeInTheDocument();
  expect(within(card).getByRole('button', { name: '저장' })).toBeDisabled();
});

it('previews the body with its sample values filled in', async () => {
  stubList();
  mount();

  const card = await cardFor('새 공지 푸시');
  expect(within(card).getByTestId('template-preview-title')).toHaveTextContent('새 소식');
  expect(within(card).getByTestId('template-preview-body')).toHaveTextContent('공지 제목');
});

it('saves with the loaded version and confirms with a toast', async () => {
  stubList();
  const put = vi
    .spyOn(api, 'put')
    .mockResolvedValue({ ...SMS_TEMPLATE, body: '[대일외고장학회] 인증번호 {code}', version: 4, isDefault: false } as never);
  const user = userEvent.setup();
  mount();

  const card = await cardFor('휴대폰 인증번호 문자');
  await replaceBody(card, user, '[대일외고장학회] 인증번호 {code}');
  await user.click(within(card).getByRole('button', { name: '저장' }));

  await waitFor(() =>
    expect(put).toHaveBeenCalledWith(`/api/admin/notification-templates/${SMS_KEY}`, {
      title: '',
      body: '[대일외고장학회] 인증번호 {code}',
      expectedVersion: 3,
    }),
  );
  expect(await screen.findByText('「휴대폰 인증번호 문자」 문구가 저장되었습니다.')).toBeInTheDocument();
  expect(within(await cardFor('휴대폰 인증번호 문자')).getByLabelText('본문')).toHaveValue(
    '[대일외고장학회] 인증번호 {code}',
  );
});

it('shows the server field message when the save is rejected', async () => {
  stubList();
  vi.spyOn(api, 'put').mockRejectedValue(
    new ApiClientError(400, 'INVALID_TEMPLATE', '알림 문구를 저장할 수 없습니다', [], {
      code: 'INVALID_TEMPLATE',
      message: '알림 문구를 저장할 수 없습니다',
      details: {
        fields: [
          { field: 'body', reason: '문자로 보낼 수 없는 문자가 있습니다' },
          { field: 'body', reason: '본문에 {code} 을(를) 반드시 포함해야 합니다' },
        ],
      },
    }),
  );
  const user = userEvent.setup();
  mount();

  const card = await cardFor('휴대폰 인증번호 문자');
  await replaceBody(card, user, '[대일외고장학회] 인증번호 {code}');
  await user.click(within(card).getByRole('button', { name: '저장' }));

  const inlineError = await within(card).findByText(/문자로 보낼 수 없는 문자가 있습니다/);
  // Both reasons for the same field are stacked rather than overwriting each other.
  expect(inlineError).toHaveTextContent('본문에 {code} 을(를) 반드시 포함해야 합니다');
});

it('raises a field the form has no input for to the card level', async () => {
  stubList();
  vi.spyOn(api, 'put').mockRejectedValue(
    new ApiClientError(400, 'INVALID_TEMPLATE', '알림 문구를 저장할 수 없습니다', [], {
      code: 'INVALID_TEMPLATE',
      message: '알림 문구를 저장할 수 없습니다',
      details: { fields: [{ field: 'expectedVersion', reason: '저장 행을 찾을 수 없습니다' }] },
    }),
  );
  const user = userEvent.setup();
  mount();

  const card = await cardFor('휴대폰 인증번호 문자');
  await replaceBody(card, user, '[대일외고장학회] 인증번호 {code}');
  await user.click(within(card).getByRole('button', { name: '저장' }));

  expect(await within(card).findByText('저장 행을 찾을 수 없습니다')).toBeInTheDocument();
});

it('refuses to save a template whose row has never been stored', async () => {
  stubList([{ ...SMS_TEMPLATE, version: 0 }]);
  const put = vi.spyOn(api, 'put');
  const user = userEvent.setup();
  mount();

  const card = await cardFor('휴대폰 인증번호 문자');
  await replaceBody(card, user, '[대일외고장학회] 인증번호 {code}');

  expect(
    within(card).getByText(
      '서버에 저장 행이 없어 수정할 수 없습니다. 마이그레이션 적용 여부를 확인해 주세요.',
    ),
  ).toBeInTheDocument();
  expect(within(card).getByRole('button', { name: '저장' })).toBeDisabled();
  expect(put).not.toHaveBeenCalled();
});

it('sends an empty title for an sms row that still has one stored', async () => {
  stubList([{ ...SMS_TEMPLATE, title: '쓰이지 않는 제목', invalid: true }]);
  const put = vi.spyOn(api, 'put').mockResolvedValue({ ...SMS_TEMPLATE, version: 4 } as never);
  const user = userEvent.setup();
  mount();

  const card = await cardFor('휴대폰 인증번호 문자');
  expect(
    within(card).getByText('저장된 문구가 유효하지 않아 기본 문구로 발송 중입니다.'),
  ).toBeInTheDocument();
  await replaceBody(card, user, '[대일외고장학회] 인증번호 {code}');

  const save = within(card).getByRole('button', { name: '저장' });
  expect(save).toBeEnabled();
  await user.click(save);

  await waitFor(() =>
    expect(put).toHaveBeenCalledWith(`/api/admin/notification-templates/${SMS_KEY}`, {
      title: '',
      body: '[대일외고장학회] 인증번호 {code}',
      expectedVersion: 3,
    }),
  );
});

it('keeps an unsaved draft when a reload brings a newer version', async () => {
  const get = stubList();
  vi.spyOn(api, 'put').mockRejectedValue(
    new ApiClientError(409, 'TEMPLATE_CONFLICT', '다른 관리자가 먼저 수정했습니다'),
  );
  const user = userEvent.setup();
  mount();

  const card = await cardFor('휴대폰 인증번호 문자');
  await replaceBody(card, user, '[대일외고장학회] 내 초안 {code}');
  await user.click(within(card).getByRole('button', { name: '저장' }));
  await within(card).findByRole('button', { name: '새로고침' });

  get.mockResolvedValue([
    { ...SMS_TEMPLATE, body: '[대일외고장학회] 다른 관리자 문구 {code}', version: 9 },
  ] as never);
  await user.click(within(card).getByRole('button', { name: '새로고침' }));

  // The draft is untouched, and the newly stored text is shown beside it.
  await waitFor(() =>
    expect(within(card).getByRole('region', { name: '서버 저장 문구' })).toHaveTextContent(
      '[대일외고장학회] 다른 관리자 문구 {code}',
    ),
  );
  expect(within(card).getByRole('region', { name: '서버 저장 문구' })).toHaveTextContent('버전 9');
  expect(within(card).getByLabelText('본문')).toHaveValue('[대일외고장학회] 내 초안 {code}');
});

it('follows the server on a card the administrator has not touched', async () => {
  const get = stubList();
  const user = userEvent.setup();
  mount();

  const card = await cardFor('새 공지 푸시');
  expect(within(card).getByLabelText('본문')).toHaveValue('{subject}');

  get.mockResolvedValue([{ ...PUSH_TEMPLATE, body: '{subject} 를 확인해 주세요', version: 5 }] as never);
  // A reload is reachable from the sms card; both cards read the same query.
  await replaceBody(await cardFor('휴대폰 인증번호 문자'), user, SMS_BODY);

  await waitFor(() => expect(get).toHaveBeenCalled());
});

it('explains a concurrent edit and offers a reload', async () => {
  const get = stubList();
  vi.spyOn(api, 'put').mockRejectedValue(
    new ApiClientError(409, 'TEMPLATE_CONFLICT', '다른 관리자가 먼저 수정했습니다'),
  );
  const user = userEvent.setup();
  mount();

  const card = await cardFor('휴대폰 인증번호 문자');
  await replaceBody(card, user, '[대일외고장학회] 인증번호 {code}');
  await user.click(within(card).getByRole('button', { name: '저장' }));

  expect(
    await within(card).findByText('다른 관리자가 먼저 수정했습니다. 새로고침 후 다시 시도해 주세요.'),
  ).toBeInTheDocument();

  get.mockClear();
  await user.click(within(card).getByRole('button', { name: '새로고침' }));
  await waitFor(() => expect(get).toHaveBeenCalled());
});
