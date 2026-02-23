// useLikeToggle — Optimistic like toggle mutation with rollback
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client';
import type { LikeToggleResponse, NoticeDetail } from '../types/api';

export function useLikeToggle(seq: number) {
  const queryClient = useQueryClient();
  const detailKey = ['feed', 'detail', String(seq)];

  return useMutation({
    mutationFn: () => api.post<LikeToggleResponse>(`/api/feed/${seq}/like`),

    onMutate: async () => {
      await queryClient.cancelQueries({ queryKey: detailKey });
      const previous = queryClient.getQueryData<NoticeDetail>(detailKey);

      if (previous) {
        queryClient.setQueryData<NoticeDetail>(detailKey, {
          ...previous,
          userLiked: !previous.userLiked,
          likeCnt: previous.userLiked ? previous.likeCnt - 1 : previous.likeCnt + 1,
        });
      }

      return { previous };
    },

    onSuccess: (data) => {
      queryClient.setQueryData<NoticeDetail>(detailKey, (prev) => {
        if (!prev) return prev;
        return { ...prev, userLiked: data.liked, likeCnt: data.likeCnt };
      });
    },

    onError: (_err, _vars, context) => {
      if (context?.previous) {
        queryClient.setQueryData(detailKey, context.previous);
      }
    },
  });
}
