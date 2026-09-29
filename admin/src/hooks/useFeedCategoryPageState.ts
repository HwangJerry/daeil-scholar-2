// useFeedCategoryPageState — edit/delete/reorder interaction state for FeedCategoryPage
import { useState } from 'react';
import type { DragEndEvent } from '@dnd-kit/core';
import { arrayMove } from '@dnd-kit/sortable';
import { ApiClientError } from '../api/client.ts';
import { FEED_CATEGORY_MAX_COUNT } from './useFeedCategories.ts';
import { HAS_POSTS_CODE, useFeedCategoryList } from './useFeedCategoryList.ts';
import type { AdminFeedCategory, AdminFeedCategoryUpsert, FeedCategoryHasPostsError } from '../types/api.ts';

export interface FeedCategoryEditState {
  seq: number | null;
  draft: AdminFeedCategoryUpsert;
}

export interface FeedCategoryDeleteState {
  target: AdminFeedCategory;
  postCount: number;
}

export function useFeedCategoryPageState() {
  const list = useFeedCategoryList();
  const { createCategory, updateCategory, deleteCategory, reorderCategories } = list;
  const [edit, setEdit] = useState<FeedCategoryEditState | null>(null);
  const [deleting, setDeleting] = useState<FeedCategoryDeleteState | null>(null);

  const categories = list.data ?? [];
  const isSaving = list.isCreating || list.isUpdating;

  const startEdit = (cat: AdminFeedCategory) =>
    setEdit({ seq: cat.seq, draft: { name: cat.name, openYn: cat.openYn } });
  const startNew = () => setEdit({ seq: null, draft: { name: '', openYn: 'Y' } });
  const cancelEdit = () => setEdit(null);
  const setDraft = (patch: Partial<AdminFeedCategoryUpsert>) =>
    setEdit((prev) => (prev ? { ...prev, draft: { ...prev.draft, ...patch } } : prev));

  const save = () => {
    if (!edit) return;
    const body = { ...edit.draft, name: edit.draft.name.trim() };
    if (edit.seq === null) {
      createCategory(body, { onSuccess: () => setEdit(null) });
    } else {
      updateCategory({ seq: edit.seq, body }, { onSuccess: () => setEdit(null) });
    }
  };

  const toggleOpen = (cat: AdminFeedCategory, open: boolean) =>
    updateCategory({ seq: cat.seq, body: { name: cat.name, openYn: open ? 'Y' : 'N' } });

  const requestDelete = (cat: AdminFeedCategory) => setDeleting({ target: cat, postCount: cat.postCount });
  const closeDelete = () => setDeleting(null);

  const confirmDelete = (moveToSeq?: number) => {
    if (!deleting) return;
    const { target } = deleting;
    deleteCategory(
      { seq: target.seq, moveToSeq },
      {
        onSuccess: () => setDeleting(null),
        onError: (err) => {
          // Posts were added since the list loaded: ask where they go instead.
          if (err instanceof ApiClientError && err.code === HAS_POSTS_CODE) {
            setDeleting({ target, postCount: (err.payload as FeedCategoryHasPostsError).postCount });
            return;
          }
          setDeleting(null);
        },
      },
    );
  };

  const handleDragEnd = ({ active, over }: DragEndEvent) => {
    if (!over || active.id === over.id) return;
    const oldIndex = categories.findIndex((c) => c.seq === active.id);
    const newIndex = categories.findIndex((c) => c.seq === over.id);
    if (oldIndex < 0 || newIndex < 0) return;
    reorderCategories(arrayMove(categories, oldIndex, newIndex).map((c) => c.seq));
  };

  return {
    categories,
    isLoading: list.isLoading,
    isError: list.isError,
    refetch: list.refetch,
    errorMessage: list.errorMessage,
    isDeleting: list.isDeleting,
    isSaving,
    isFull: categories.length >= FEED_CATEGORY_MAX_COUNT,
    edit,
    deleting,
    startEdit,
    startNew,
    cancelEdit,
    setDraft,
    save,
    toggleOpen,
    requestDelete,
    closeDelete,
    confirmDelete,
    handleDragEnd,
  };
}
