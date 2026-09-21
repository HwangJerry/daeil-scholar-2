// notificationTemplateValidation — client mirror of the server's template rules
import type {
  NotificationTemplateDraft,
  NotificationTemplateFieldErrors,
  NotificationTemplateView,
} from '../types/notificationTemplate.ts';
import {
  eucKrByteLength,
  hasStrayBrace,
  missingRequiredPlaceholders,
  renderWithSamples,
  runeLength,
  unknownBraceTokens,
} from './notificationTemplateText.ts';

/**
 * Channel caps come from the response and are never assumed here: the gateway's
 * limits change independently of this screen, and a guessed cap would either
 * block text the server accepts or promise room it does not have. A cap the
 * response omits is simply not enforced client-side — the server still is.
 */
export function smsByteLimit(template: NotificationTemplateView): number | undefined {
  return template.limits.maxBodyEucKrBytes;
}

export function pushTitleLimit(template: NotificationTemplateView): number | undefined {
  return template.limits.maxTitleRunes;
}

export function pushBodyLimit(template: NotificationTemplateView): number | undefined {
  return template.limits.maxBodyRunes;
}

/** EUC-KR bytes the body will occupy once its placeholders are filled in. */
export function smsBodyBytes(template: NotificationTemplateView, body: string): number {
  return eucKrByteLength(renderWithSamples(body, template.placeholders));
}

/**
 * Rejects a draft the server would reject anyway, so the administrator sees the
 * problem while typing instead of after a failed save. The wording follows the
 * server's messages so the inline text does not change once it round-trips.
 */
export function validateTemplateDraft(
  template: NotificationTemplateView,
  draft: NotificationTemplateDraft,
): NotificationTemplateFieldErrors {
  const errors: NotificationTemplateFieldErrors = {};
  const addError = (field: 'title' | 'body', reason: string) => {
    if (!errors[field]) errors[field] = reason;
  };

  validateBraces(template, draft, addError);

  if (draft.body.trim() === '') addError('body', '본문을 입력해 주세요');
  for (const placeholder of missingRequiredPlaceholders(draft.body, template.placeholders)) {
    addError('body', `본문에 {${placeholder.name}} 을(를) 반드시 포함해야 합니다`);
  }

  if (template.channel === 'sms') {
    validateSmsDraft(template, draft, addError);
  } else {
    validatePushDraft(template, draft, addError);
  }
  return errors;
}

/**
 * A token this key does not support, or a brace that is not part of one, reaches
 * the member verbatim: `{cod}` in place of their code, `{{code}}` wrapped in
 * braces they were never meant to see.
 */
function validateBraces(
  template: NotificationTemplateView,
  draft: NotificationTemplateDraft,
  addError: (field: 'title' | 'body', reason: string) => void,
) {
  for (const field of ['title', 'body'] as const) {
    const text = draft[field];
    for (const token of unknownBraceTokens(text, template.placeholders)) {
      addError(field, `${token} 은(는) 사용할 수 없는 치환 항목입니다`);
    }
    if (hasStrayBrace(text)) {
      addError(field, '중괄호 { } 는 치환 항목에만 사용할 수 있습니다');
    }
  }
}

function validateSmsDraft(
  template: NotificationTemplateView,
  draft: NotificationTemplateDraft,
  addError: (field: 'title' | 'body', reason: string) => void,
) {
  if (draft.title.trim() !== '') {
    addError('title', '문자 템플릿에는 제목을 사용하지 않습니다');
  }
  const limit = smsByteLimit(template);
  if (limit === undefined) return;
  const bytes = smsBodyBytes(template, draft.body);
  if (bytes > limit) {
    addError('body', `치환 항목을 채운 길이가 ${bytes}바이트로 최대 ${limit}바이트를 넘습니다`);
  }
}

function validatePushDraft(
  template: NotificationTemplateView,
  draft: NotificationTemplateDraft,
  addError: (field: 'title' | 'body', reason: string) => void,
) {
  if (draft.title.trim() === '') addError('title', '제목을 입력해 주세요');
  const titleLimit = pushTitleLimit(template);
  if (titleLimit !== undefined && runeLength(draft.title) > titleLimit) {
    addError('title', `제목은 ${titleLimit}자 이하여야 합니다`);
  }
  const bodyLimit = pushBodyLimit(template);
  if (bodyLimit !== undefined && runeLength(draft.body) > bodyLimit) {
    addError('body', `본문은 ${bodyLimit}자 이하여야 합니다`);
  }
}
