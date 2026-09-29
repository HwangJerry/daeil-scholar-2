// FeedCategoryDeleteDialogs — plain confirm for an empty category, move-then-delete dialog when it has posts
import { ConfirmDialog } from '../ui/ConfirmDialog.tsx';
import { FeedCategoryMoveDeleteDialog } from './FeedCategoryMoveDeleteDialog.tsx';
import type { AdminFeedCategory } from '../../types/api.ts';
import type { FeedCategoryDeleteState } from '../../hooks/useFeedCategoryPageState.ts';

export interface FeedCategoryDeleteDialogsProps {
  deleting: FeedCategoryDeleteState | null;
  categories: AdminFeedCategory[];
  onConfirm: (moveToSeq?: number) => void;
  onClose: () => void;
  isPending: boolean;
}

export function FeedCategoryDeleteDialogs({ deleting, categories, onConfirm, onClose, isPending }: FeedCategoryDeleteDialogsProps) {
  const hasPosts = deleting !== null && deleting.postCount > 0;
  return (
    <>
      <ConfirmDialog
        open={deleting !== null && !hasPosts}
        onOpenChange={(open) => { if (!open) onClose(); }}
        title="카테고리 삭제"
        description={`'${deleting?.target.name ?? ''}' 카테고리를 삭제합니다.`}
        confirmLabel="삭제"
        variant="destructive"
        onConfirm={() => onConfirm()}
        isPending={isPending}
      />
      {deleting !== null && hasPosts && (
        <FeedCategoryMoveDeleteDialog
          key={deleting.target.seq}
          target={deleting.target}
          postCount={deleting.postCount}
          categories={categories}
          onConfirm={onConfirm}
          onClose={onClose}
          isPending={isPending}
        />
      )}
    </>
  );
}
