// notificationTemplateErrors — maps template save failures onto field and card-level messages
import { ApiClientError } from '../api/client.ts';
import type { NotificationTemplateFieldErrors } from '../types/notificationTemplate.ts';

/** One rejected field as `INVALID_TEMPLATE` reports it. */
interface TemplateFieldError {
  field: string;
  reason: string;
}

interface TemplateErrorPayload {
  details?: { fields?: TemplateFieldError[] };
}

/** Separates the reasons the form can show inline from the ones only the card can. */
const INLINE_FIELDS = ['title', 'body'] as const;

function fieldErrorEntries(error: unknown): TemplateFieldError[] {
  if (!(error instanceof ApiClientError)) return [];
  const fields = (error.payload as TemplateErrorPayload | undefined)?.details?.fields;
  return Array.isArray(fields) ? fields.filter((entry) => entry && entry.reason) : [];
}

function isInlineField(field: string): field is (typeof INLINE_FIELDS)[number] {
  return (INLINE_FIELDS as readonly string[]).includes(field);
}

/**
 * The server's per-field reasons, shown under the matching input. One field can
 * be rejected for several reasons at once — an over-long body that also dropped
 * a required placeholder — so the reasons are stacked rather than overwritten.
 */
export function templateFieldErrors(error: unknown): NotificationTemplateFieldErrors {
  return fieldErrorEntries(error).reduce<NotificationTemplateFieldErrors>((messages, entry) => {
    if (!isInlineField(entry.field)) return messages;
    const existing = messages[entry.field];
    messages[entry.field] = existing ? `${existing}\n${entry.reason}` : entry.reason;
    return messages;
  }, {});
}

/**
 * The message shown above the buttons, or undefined when every reason is already
 * rendered under its input. A rejection naming a field the form has no input for
 * — `expectedVersion`, say — would otherwise leave the screen silent.
 */
export function templateCardMessage(error: unknown): string | undefined {
  if (!error) return undefined;
  const entries = fieldErrorEntries(error);
  const unmapped = entries.filter((entry) => !isInlineField(entry.field));
  if (unmapped.length > 0) return unmapped.map((entry) => entry.reason).join(' ');
  return entries.length > 0 ? undefined : templateErrorMessage(error);
}

/** True when another administrator saved first and the card must be reloaded. */
export function isTemplateConflict(error: unknown): boolean {
  return error instanceof ApiClientError && error.code === 'TEMPLATE_CONFLICT';
}

/** One sentence describing a failed save, used when no field reason explains it. */
export function templateErrorMessage(error: unknown): string {
  if (!(error instanceof ApiClientError)) {
    return '문구를 저장하지 못했습니다. 잠시 후 다시 시도해 주세요.';
  }
  switch (error.code) {
    case 'TEMPLATE_CONFLICT':
      return '다른 관리자가 먼저 수정했습니다. 새로고침 후 다시 시도해 주세요.';
    case 'INVALID_TEMPLATE':
      return '입력값을 확인해 주세요.';
    case 'INVALID_BODY':
      return '요청 형식이 올바르지 않습니다. 새로고침 후 다시 시도해 주세요.';
    case 'TEMPLATE_NOT_FOUND':
      return '문구를 찾을 수 없습니다. 마이그레이션 적용 여부를 확인해 주세요.';
    default:
      return error.message || '문구를 저장하지 못했습니다.';
  }
}
