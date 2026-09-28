// accountErasureLabels.test — Erasure tables shown to operators get a readable Korean name.
import { expect, it } from 'vitest';
import { tableLabel } from '../components/accountErasureLabels';

it('names the notification inbox last-seen table', () => {
  expect(tableLabel('ALUMNI_NOTIFICATION_INBOX_STATE')).toBe('알림함 확인 기록');
});

it('keeps the push preference label distinct from the inbox state', () => {
  expect(tableLabel('ALUMNI_PUSH_PREFERENCE')).toBe('알림 설정');
});

it('falls back for unknown member tables', () => {
  expect(tableLabel('SOME_FUTURE_TABLE')).toBe('기타 회원 기록');
  expect(tableLabel('AUTH_SOMETHING')).toBe('인증 정보');
});
