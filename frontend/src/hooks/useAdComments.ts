// useAdComments — React Query hooks for ad comment list, add, and delete
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api, ApiClientError } from '../api/client';
import type { AdComment, CommentCreateRequest } from '../types/api';

const AUTH_ERROR_STATUS = 401;

export function useAdComments(maSeq: number) {
  return useQuery({
    queryKey: ['ad', 'comments', maSeq],
    queryFn: () => api.get<AdComment[]>(`/api/ad/${maSeq}/comments`),
    staleTime: 30_000,
  });
}

interface UseAddAdCommentOptions {
  onAuthError?: () => void;
}

export function useAddAdComment(maSeq: number, options?: UseAddAdCommentOptions) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (body: CommentCreateRequest) =>
      api.post<AdComment>(`/api/ad/${maSeq}/comments`, body),

    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['ad', 'comments', maSeq] });
    },

    onError: (error: Error) => {
      if (
        error instanceof ApiClientError &&
        error.status === AUTH_ERROR_STATUS &&
        options?.onAuthError
      ) {
        options.onAuthError();
      }
    },
  });
}

export function useDeleteAdComment(maSeq: number) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (acSeq: number) =>
      api.del(`/api/ad/${maSeq}/comments/${acSeq}`),

    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['ad', 'comments', maSeq] });
    },
  });
}
