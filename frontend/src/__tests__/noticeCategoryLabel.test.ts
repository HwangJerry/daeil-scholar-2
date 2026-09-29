// noticeCategoryLabel.test — server categoryName wins; older payloads keep the built-in mapping
import { expect, it } from 'vitest';
import { noticeCategoryLabel } from '../components/feed/noticeCategoryLabel';

it('uses the server category name when present', () => {
  expect(noticeCategoryLabel({ category: 'c3', categoryName: '장학' })).toBe('장학');
  expect(noticeCategoryLabel({ category: 'notice', categoryName: '새 소식' })).toBe('새 소식');
});

it('falls back to the built-in mapping without a name', () => {
  expect(noticeCategoryLabel({ category: 'scholarship' })).toBe('장학');
  expect(noticeCategoryLabel({})).toBe('공지');
  expect(noticeCategoryLabel({ category: 'unknown', categoryName: '' })).toBe('공지');
});
