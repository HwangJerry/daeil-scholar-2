// useJobCategoryList — query and mutations for job category admin CRUD (incl. drag-and-drop reorder)
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, ApiClientError } from '../api/client.ts';
import { useToast } from './useToast.ts';
import type { AdminJobCategory, AdminJobCategoryUpsert } from '../types/api.ts';

const QUERY_KEY = ['admin', 'job-categories'] as const;

export function useJobCategoryList() {
  const queryClient = useQueryClient();
  const addToast = useToast((s) => s.addToast);

  const query = useQuery<AdminJobCategory[]>({
    queryKey: QUERY_KEY,
    queryFn: () => api.get<AdminJobCategory[]>('/api/admin/job-category'),
  });

  const invalidate = () => void queryClient.invalidateQueries({ queryKey: QUERY_KEY });

  const onDuplicateName = () =>
    addToast({ variant: 'error', title: '중복 이름', description: '이미 존재하는 카테고리 이름입니다.' });

  const createMutation = useMutation({
    mutationFn: (body: AdminJobCategoryUpsert) =>
      api.post<{ seq: number }>('/api/admin/job-category', body),
    onSuccess: () => {
      invalidate();
      addToast({ variant: 'success', title: '카테고리가 추가되었습니다.' });
    },
    onError: (err: unknown) => {
      if (err instanceof ApiClientError && err.code === 'DUPLICATE_NAME') {
        onDuplicateName();
        return;
      }
      addToast({ variant: 'error', title: '추가 실패', description: '다시 시도해 주세요.' });
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ seq, body }: { seq: number; body: AdminJobCategoryUpsert }) =>
      api.put<void>(`/api/admin/job-category/${seq}`, body),
    onSuccess: () => {
      invalidate();
      addToast({ variant: 'success', title: '저장되었습니다.' });
    },
    onError: (err: unknown) => {
      if (err instanceof ApiClientError && err.code === 'DUPLICATE_NAME') {
        onDuplicateName();
        return;
      }
      addToast({ variant: 'error', title: '저장 실패', description: '다시 시도해 주세요.' });
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (seq: number) => api.del<void>(`/api/admin/job-category/${seq}`),
    onSuccess: () => {
      invalidate();
      addToast({ variant: 'success', title: '카테고리가 비활성화되었습니다.' });
    },
    onError: () => {
      addToast({ variant: 'error', title: '삭제 실패', description: '다시 시도해 주세요.' });
    },
  });

  const reorderMutation = useMutation<
    void,
    unknown,
    number[],
    { previous: AdminJobCategory[] | undefined }
  >({
    mutationFn: (order: number[]) =>
      api.post<void>('/api/admin/job-category/reorder', { order }),
    onMutate: async (order) => {
      await queryClient.cancelQueries({ queryKey: QUERY_KEY });
      const previous = queryClient.getQueryData<AdminJobCategory[]>(QUERY_KEY);
      if (previous) {
        const bySeq = new Map(previous.map((c) => [c.seq, c]));
        const reordered = order
          .map((seq) => bySeq.get(seq))
          .filter((c): c is AdminJobCategory => c !== undefined)
          .map((c, idx) => ({ ...c, index: idx + 1 }));
        queryClient.setQueryData(QUERY_KEY, reordered);
      }
      return { previous };
    },
    onError: (_err, _order, context) => {
      if (context?.previous) {
        queryClient.setQueryData(QUERY_KEY, context.previous);
      }
      addToast({ variant: 'error', title: '순서 저장 실패', description: '다시 시도해 주세요.' });
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
    createCategory: createMutation.mutate,
    updateCategory: updateMutation.mutate,
    deleteCategory: deleteMutation.mutate,
    reorderCategories: reorderMutation.mutate,
    isCreating: createMutation.isPending,
    isUpdating: updateMutation.isPending,
    isDeleting: deleteMutation.isPending,
    isReordering: reorderMutation.isPending,
  };
}
