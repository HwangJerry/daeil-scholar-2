// useFeedCategoryList — feed category query plus create/update/reorder/delete mutations with inline error text
import { useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api, ApiClientError } from '../api/client.ts';
import { useToast } from './useToast.ts';
import { FEED_CATEGORIES_QUERY_KEY, useFeedCategories } from './useFeedCategories.ts';
import type { AdminFeedCategory, AdminFeedCategoryUpsert } from '../types/api.ts';

const GENERIC_ERROR = '요청을 처리하지 못했습니다. 다시 시도해 주세요.';
export const HAS_POSTS_CODE = 'CATEGORY_HAS_POSTS';

/** Backend refusals carry Korean text meant to be shown as-is; anything else gets a generic line. */
function errorText(err: unknown): string {
  if (err instanceof ApiClientError && err.status < 500 && err.message) return err.message;
  return GENERIC_ERROR;
}

export interface DeleteFeedCategoryVars {
  seq: number;
  moveToSeq?: number;
}

export function useFeedCategoryList() {
  const queryClient = useQueryClient();
  const addToast = useToast((s) => s.addToast);
  const query = useFeedCategories();
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const invalidate = () => void queryClient.invalidateQueries({ queryKey: FEED_CATEGORIES_QUERY_KEY });
  const onError = (err: unknown) => setErrorMessage(errorText(err));
  const onMutate = () => setErrorMessage(null);

  const createMutation = useMutation({
    mutationFn: (body: AdminFeedCategoryUpsert) =>
      api.post<AdminFeedCategory>('/api/admin/feed-categories', body),
    onMutate,
    onSuccess: () => {
      invalidate();
      addToast({ variant: 'success', title: '카테고리가 추가되었습니다.' });
    },
    onError,
  });

  const updateMutation = useMutation({
    mutationFn: ({ seq, body }: { seq: number; body: AdminFeedCategoryUpsert }) =>
      api.put<AdminFeedCategory>(`/api/admin/feed-categories/${seq}`, body),
    onMutate,
    onSuccess: () => {
      invalidate();
      addToast({ variant: 'success', title: '저장되었습니다.' });
    },
    onError,
  });

  const deleteMutation = useMutation({
    mutationFn: ({ seq, moveToSeq }: DeleteFeedCategoryVars) =>
      api.del<void>(`/api/admin/feed-categories/${seq}`, moveToSeq ? { moveToSeq } : undefined),
    onMutate,
    onSuccess: () => {
      invalidate();
      // Moved posts show a new category in the notice list.
      void queryClient.invalidateQueries({ queryKey: ['admin', 'notices'] });
      addToast({ variant: 'success', title: '카테고리가 삭제되었습니다.' });
    },
    onError: (err: unknown) => {
      // The page reopens the move dialog for this one; it is not an inline error.
      if (err instanceof ApiClientError && err.code === HAS_POSTS_CODE) return;
      onError(err);
    },
  });

  const reorderMutation = useMutation<void, unknown, number[], { previous: AdminFeedCategory[] | undefined }>({
    mutationFn: (seqs: number[]) => api.put<void>('/api/admin/feed-categories/order', { seqs }),
    onMutate: async (seqs) => {
      setErrorMessage(null);
      await queryClient.cancelQueries({ queryKey: FEED_CATEGORIES_QUERY_KEY });
      const previous = queryClient.getQueryData<AdminFeedCategory[]>(FEED_CATEGORIES_QUERY_KEY);
      if (previous) {
        const bySeq = new Map(previous.map((c) => [c.seq, c]));
        const reordered = seqs
          .map((seq) => bySeq.get(seq))
          .filter((c): c is AdminFeedCategory => c !== undefined)
          .map((c, idx) => ({ ...c, sortOrder: idx + 1 }));
        queryClient.setQueryData(FEED_CATEGORIES_QUERY_KEY, reordered);
      }
      return { previous };
    },
    onError: (err, _seqs, context) => {
      if (context?.previous) {
        queryClient.setQueryData(FEED_CATEGORIES_QUERY_KEY, context.previous);
      }
      onError(err);
    },
    onSuccess: () => {
      addToast({ variant: 'success', title: '순서가 저장되었습니다.' });
    },
    onSettled: () => {
      invalidate();
    },
  });

  return {
    ...query,
    errorMessage,
    createCategory: createMutation.mutate,
    updateCategory: updateMutation.mutate,
    deleteCategory: deleteMutation.mutate,
    reorderCategories: reorderMutation.mutate,
    isCreating: createMutation.isPending,
    isUpdating: updateMutation.isPending,
    isDeleting: deleteMutation.isPending,
  };
}
