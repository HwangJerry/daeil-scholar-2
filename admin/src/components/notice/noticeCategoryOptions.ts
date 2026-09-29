// noticeCategoryOptions — which feed categories a post's category select offers
import type { AdminFeedCategory } from '../../types/api.ts';

export interface NoticeCategoryOption {
  seq: number;
  label: string;
}

/**
 * Open categories in admin order, plus the post's current category when it is
 * hidden (suffixed "(숨김)") so editing never silently reclassifies a post.
 */
export function noticeCategoryOptions(categories: AdminFeedCategory[], currentSeq: number | null): NoticeCategoryOption[] {
  return categories
    .filter((c) => c.openYn === 'Y' || c.seq === currentSeq)
    .map((c) => ({ seq: c.seq, label: c.openYn === 'Y' ? c.name : `${c.name} (숨김)` }));
}

export function defaultCategorySeq(categories: AdminFeedCategory[]): number | null {
  return categories.find((c) => c.isDefault === 'Y')?.seq ?? null;
}
