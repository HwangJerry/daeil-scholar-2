// appUpdatePolicyErrors — maps policy save failures onto per-field and summary messages
import { ApiClientError } from '../api/client.ts';
import type { AppUpdatePolicyFieldError } from '../types/appUpdate.ts';

interface PolicyErrorPayload {
  details?: { fields?: AppUpdatePolicyFieldError[] };
}

/** Field errors reported by the server, keyed by policy field name. */
export function policyFieldErrors(error: unknown): Record<string, string> {
  if (!(error instanceof ApiClientError)) return {};
  const fields = (error.payload as PolicyErrorPayload | undefined)?.details?.fields;
  if (!fields) return {};
  return fields.reduce<Record<string, string>>((messages, field) => {
    messages[field.field] = field.reason;
    return messages;
  }, {});
}

/** One sentence describing a failed save, shown above the form. */
export function policyErrorMessage(error: unknown): string {
  if (!(error instanceof ApiClientError)) {
    return '정책을 저장하지 못했습니다. 잠시 후 다시 시도해 주세요.';
  }
  switch (error.code) {
    case 'POLICY_CONFLICT':
      return '다른 관리자가 먼저 수정했습니다. 새로고침 후 다시 시도해 주세요.';
    case 'INVALID_POLICY':
      return '입력값을 확인해 주세요.';
    case 'POLICY_NOT_FOUND':
      return '정책을 찾을 수 없습니다. 마이그레이션 적용 여부를 확인해 주세요.';
    case 'INVALID_PLATFORM':
      return '지원하지 않는 플랫폼입니다.';
    default:
      return error.message || '정책을 저장하지 못했습니다.';
  }
}
