// useComments — React Query hooks for comment list, add, and delete
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';
import { ApiClientError } from '../api/client';
import type { Comment, CommentCreateRequest } from '../types/api';

const AUTH_ERROR_STATUS = 401;

export function useComments(seq: number) {
  return useQuery({
    queryKey: ['feed', 'comments', seq],
    queryFn: () => api.get<Comment[]>(`/api/feed/${seq}/comments`),
    staleTime: 30_000,
  });
}

interface UseAddCommentOptions {
  onAuthError?: () => void;
}

export function useAddComment(seq: number, options?: UseAddCommentOptions) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (body: CommentCreateRequest) =>
      api.post<Comment>(`/api/feed/${seq}/comments`, body),

    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['feed', 'comments', seq] });
      queryClient.invalidateQueries({ queryKey: ['feed', 'detail', String(seq)] });
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

export function useDeleteComment(seq: number) {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (cSeq: number) =>
      api.del(`/api/feed/${seq}/comments/${cSeq}`),

    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['feed', 'comments', seq] });
      queryClient.invalidateQueries({ queryKey: ['feed', 'detail', String(seq)] });
    },
  });
}
