// notificationTemplateErrors.test — parsing the array-shaped INVALID_TEMPLATE details
import { describe, expect, it } from 'vitest';
import { ApiClientError } from '../api/client';
import {
  isTemplateConflict,
  templateCardMessage,
  templateErrorMessage,
  templateFieldErrors,
} from '../lib/notificationTemplateErrors';

function invalidTemplate(fields: { field: string; reason: string }[]) {
  return new ApiClientError(400, 'INVALID_TEMPLATE', '알림 문구를 저장할 수 없습니다', [], {
    code: 'INVALID_TEMPLATE',
    message: '알림 문구를 저장할 수 없습니다',
    details: { fields },
  });
}

describe('templateFieldErrors', () => {
  it('maps title and body entries onto their inputs', () => {
    expect(
      templateFieldErrors(
        invalidTemplate([
          { field: 'title', reason: '제목을 입력해 주세요' },
          { field: 'body', reason: '본문을 입력해 주세요' },
        ]),
      ),
    ).toEqual({ title: '제목을 입력해 주세요', body: '본문을 입력해 주세요' });
  });

  it('stacks several reasons for the same field instead of dropping one', () => {
    expect(
      templateFieldErrors(
        invalidTemplate([
          { field: 'body', reason: '첫째 이유' },
          { field: 'body', reason: '둘째 이유' },
        ]),
      ).body,
    ).toBe('첫째 이유\n둘째 이유');
  });

  it('ignores fields the form has no input for, and non-API errors', () => {
    expect(templateFieldErrors(invalidTemplate([{ field: 'expectedVersion', reason: '없음' }]))).toEqual(
      {},
    );
    expect(templateFieldErrors(new Error('boom'))).toEqual({});
  });

  it('tolerates a details payload that is missing or still map-shaped', () => {
    const mapShaped = new ApiClientError(400, 'INVALID_TEMPLATE', '거절', [], {
      details: { fields: { body: '이유' } },
    });
    expect(templateFieldErrors(mapShaped)).toEqual({});
    expect(templateCardMessage(mapShaped)).toBe('입력값을 확인해 주세요.');
  });
});

describe('templateCardMessage', () => {
  it('is undefined when every reason is shown inline', () => {
    expect(templateCardMessage(invalidTemplate([{ field: 'body', reason: '본문을 입력해 주세요' }]))).toBeUndefined();
  });

  it('surfaces reasons for fields the form cannot show', () => {
    expect(
      templateCardMessage(
        invalidTemplate([
          { field: 'body', reason: '본문을 입력해 주세요' },
          { field: 'expectedVersion', reason: '저장 행을 찾을 수 없습니다' },
        ]),
      ),
    ).toBe('저장 행을 찾을 수 없습니다');
  });

  it('falls back to the code-based sentence when no fields are reported', () => {
    const conflict = new ApiClientError(409, 'TEMPLATE_CONFLICT', '충돌');
    expect(templateCardMessage(conflict)).toBe(
      '다른 관리자가 먼저 수정했습니다. 새로고침 후 다시 시도해 주세요.',
    );
    expect(isTemplateConflict(conflict)).toBe(true);
    expect(templateErrorMessage(new Error('boom'))).toBe(
      '문구를 저장하지 못했습니다. 잠시 후 다시 시도해 주세요.',
    );
  });

  it('is undefined when there is no error at all', () => {
    expect(templateCardMessage(undefined)).toBeUndefined();
  });
});
