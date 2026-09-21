// notificationTemplateText.test — byte counting, placeholder scanning, and sample rendering
import { describe, expect, it } from 'vitest';
import {
  braceTokenName,
  braceTokens,
  eucKrByteLength,
  hasStrayBrace,
  insertPlaceholder,
  missingRequiredPlaceholders,
  renderWithSamples,
  runeLength,
  unknownBraceTokens,
} from '../lib/notificationTemplateText';
import { validateTemplateDraft } from '../lib/notificationTemplateValidation';
import type {
  NotificationTemplatePlaceholder,
  NotificationTemplateView,
} from '../types/notificationTemplate';

const CODE: NotificationTemplatePlaceholder = {
  name: 'code',
  required: true,
  description: '6자리 인증번호',
  sample: '000000',
};
const SENDER: NotificationTemplatePlaceholder = {
  name: 'senderName',
  required: false,
  description: '보낸 사람 이름',
  sample: '홍길동',
};

const SMS_TEMPLATE: NotificationTemplateView = {
  key: 'sms.phone_verification',
  channel: 'sms',
  displayName: '휴대폰 인증번호 문자',
  description: '인증번호 문자입니다.',
  title: '',
  body: '[대일외고장학회] 인증번호 {code} 를 입력해 주세요.',
  defaultTitle: '',
  defaultBody: '[대일외고장학회] 인증번호 {code} 를 입력해 주세요.',
  isDefault: true,
  invalid: false,
  version: 1,
  updatedAt: null,
  updatedBy: null,
  placeholders: [CODE],
  limits: { maxBodyEucKrBytes: 80 },
};

describe('eucKrByteLength', () => {
  it('counts ASCII as one byte and everything else as two', () => {
    expect(eucKrByteLength('abc')).toBe(3);
    expect(eucKrByteLength('가나다')).toBe(6);
    expect(eucKrByteLength('[대일] 1')).toBe(8);
    expect(eucKrByteLength('')).toBe(0);
  });

  it('counts a surrogate pair as one character', () => {
    expect(runeLength('🙂')).toBe(1);
    expect(eucKrByteLength('🙂')).toBe(2);
  });
});

describe('placeholder scanning', () => {
  it('finds every brace token, malformed ones included', () => {
    expect(braceTokens('a {code} b {cod} c')).toEqual(['{code}', '{cod}']);
    expect(braceTokenName('{code}')).toBe('code');
  });

  it('reports tokens the template does not allow, without duplicates', () => {
    expect(unknownBraceTokens('{code} {cod} {cod} {1x}', [CODE])).toEqual(['{cod}', '{1x}']);
    expect(unknownBraceTokens('{code}', [CODE])).toEqual([]);
  });

  it('reports a brace that is not part of a token', () => {
    expect(hasStrayBrace('인증번호 {code} 입니다')).toBe(false);
    expect(hasStrayBrace('인증번호 {{code}} 입니다')).toBe(true);
    expect(hasStrayBrace('인증번호 {code 입니다')).toBe(true);
    expect(hasStrayBrace('인증번호 code} 입니다')).toBe(true);
  });

  it('reports required placeholders the body is missing', () => {
    expect(missingRequiredPlaceholders('인증번호를 입력해 주세요', [CODE, SENDER])).toEqual([CODE]);
    expect(missingRequiredPlaceholders('{code}', [CODE])).toEqual([]);
  });
});

describe('renderWithSamples', () => {
  it('substitutes every allowed placeholder with its sample', () => {
    expect(renderWithSamples('{senderName}님: 인증번호 {code}', [SENDER, CODE])).toBe(
      '홍길동님: 인증번호 000000',
    );
  });

  it('leaves unknown tokens untouched so the preview shows the mistake', () => {
    expect(renderWithSamples('{cod}', [CODE])).toBe('{cod}');
  });

  it('substitutes in one pass, so a sample containing a token is not re-substituted', () => {
    const quoting: NotificationTemplatePlaceholder = {
      name: 'subject',
      required: true,
      description: '공지 제목',
      sample: '{code} 안내',
    };
    expect(renderWithSamples('{subject}', [quoting, CODE])).toBe('{code} 안내');
  });
});

describe('insertPlaceholder', () => {
  it('inserts the token at the caret and reports the caret after it', () => {
    expect(insertPlaceholder('인증번호  입니다', 'code', 5, 5)).toEqual({
      text: '인증번호 {code} 입니다',
      caret: 11,
    });
  });

  it('replaces the selected range', () => {
    expect(insertPlaceholder('abcd', 'code', 1, 3)).toEqual({ text: 'a{code}d', caret: 7 });
  });
});

describe('validateTemplateDraft', () => {
  it('accepts the default sms body', () => {
    expect(validateTemplateDraft(SMS_TEMPLATE, { title: '', body: SMS_TEMPLATE.defaultBody })).toEqual(
      {},
    );
  });

  it('rejects a missing required placeholder and an over-long sms body', () => {
    expect(validateTemplateDraft(SMS_TEMPLATE, { title: '', body: '인증번호를 입력해 주세요' }).body).toContain(
      '{code}',
    );
    const longBody = `{code} ${'가'.repeat(60)}`;
    expect(validateTemplateDraft(SMS_TEMPLATE, { title: '', body: longBody }).body).toContain(
      '바이트를 넘습니다',
    );
  });

  it('rejects a brace left outside an allowed token', () => {
    const strayBraceMessage = '중괄호 { } 는 치환 항목에만 사용할 수 있습니다';
    for (const body of ['{code} {{code}}', '{code} 인증번호 }', '{code} 인증번호 {']) {
      expect(validateTemplateDraft(SMS_TEMPLATE, { title: '', body }).body).toBe(strayBraceMessage);
    }
    expect(validateTemplateDraft(SMS_TEMPLATE, { title: '', body: '{code} 인증번호' })).toEqual({});
  });

  it('rejects a stray brace in a push title as well as the body', () => {
    const push: NotificationTemplateView = {
      ...SMS_TEMPLATE,
      key: 'push.notice.new',
      channel: 'push',
      placeholders: [],
      limits: { maxTitleRunes: 50, maxBodyRunes: 200 },
    };
    expect(validateTemplateDraft(push, { title: '새 소식 }', body: '내용' }).title).toBe(
      '중괄호 { } 는 치환 항목에만 사용할 수 있습니다',
    );
  });

  it('skips a cap the response does not report, leaving the server authoritative', () => {
    const noLimits = { ...SMS_TEMPLATE, limits: {} };
    expect(validateTemplateDraft(noLimits, { title: '', body: `{code} ${'가'.repeat(60)}` })).toEqual(
      {},
    );
  });

  it('rejects a title on an sms template and an unknown token', () => {
    expect(validateTemplateDraft(SMS_TEMPLATE, { title: '제목', body: '{code}' }).title).toBe(
      '문자 템플릿에는 제목을 사용하지 않습니다',
    );
    expect(validateTemplateDraft(SMS_TEMPLATE, { title: '', body: '{code} {cod}' }).body).toBe(
      '{cod} 은(는) 사용할 수 없는 치환 항목입니다',
    );
  });

  it('requires a title for push templates', () => {
    const push: NotificationTemplateView = {
      ...SMS_TEMPLATE,
      key: 'push.notice.new',
      channel: 'push',
      placeholders: [],
      limits: { maxTitleRunes: 50, maxBodyRunes: 200 },
    };
    expect(validateTemplateDraft(push, { title: '', body: '내용' }).title).toBe('제목을 입력해 주세요');
    expect(validateTemplateDraft(push, { title: '가'.repeat(51), body: '내용' }).title).toBe(
      '제목은 50자 이하여야 합니다',
    );
  });
});
