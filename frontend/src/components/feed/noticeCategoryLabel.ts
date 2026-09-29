// noticeCategoryLabel — card label: the server's categoryName, else the built-in code mapping
import type { NoticeItem } from '../../types/api';
import { NOTICE_CATEGORY_LABELS } from './noticeCard.constants';

export function noticeCategoryLabel(item: Pick<NoticeItem, 'category' | 'categoryName'>): string {
  if (item.categoryName) return item.categoryName;
  return NOTICE_CATEGORY_LABELS[item.category ?? 'notice'] ?? '공지';
}
