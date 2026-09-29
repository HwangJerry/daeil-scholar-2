// FeedCategoryMoveDeleteDialog — delete a category that has posts after choosing where its posts move
import { useState } from 'react';
import { ConfirmDialog } from '../ui/ConfirmDialog.tsx';
import { Select } from '../ui/Select.tsx';
import type { AdminFeedCategory } from '../../types/api.ts';

export interface FeedCategoryMoveDeleteDialogProps {
  target: AdminFeedCategory;
  postCount: number;
  categories: AdminFeedCategory[];
  onConfirm: (moveToSeq: number) => void;
  onClose: () => void;
  isPending: boolean;
}

/** Mount with key={target.seq} so the preselection resets per target. */
export function FeedCategoryMoveDeleteDialog({ target, postCount, categories, onConfirm, onClose, isPending }: FeedCategoryMoveDeleteDialogProps) {
  const others = categories.filter((c) => c.seq !== target.seq);
  const initial = others.find((c) => c.isDefault === 'Y') ?? others[0];
  const [moveToSeq, setMoveToSeq] = useState<number | undefined>(initial?.seq);

  return (
    <ConfirmDialog
      open
      onOpenChange={(open) => { if (!open) onClose(); }}
      title={`'${target.name}' 카테고리 삭제`}
      description={`이 카테고리에 게시글 ${postCount}개가 있습니다. 게시글을 옮길 카테고리를 선택하면 옮긴 뒤 카테고리를 삭제합니다.`}
      confirmLabel="옮기고 삭제"
      cancelLabel="취소"
      variant="destructive"
      onConfirm={() => { if (moveToSeq !== undefined) onConfirm(moveToSeq); }}
      isPending={isPending}
      confirmDisabled={moveToSeq === undefined}
    >
      <label className="mt-4 block text-sm font-medium text-dark-slate" htmlFor="feed-category-move-to">
        옮길 카테고리
      </label>
      <Select
        id="feed-category-move-to"
        className="mt-1.5"
        value={moveToSeq ?? ''}
        onChange={(e) => setMoveToSeq(Number(e.target.value))}
        disabled={isPending}
      >
        {others.map((c) => (
          <option key={c.seq} value={c.seq}>
            {c.name}{c.openYn === 'N' ? ' (숨김)' : ''}
          </option>
        ))}
      </Select>
    </ConfirmDialog>
  );
}
