// useNoticeLikeToggle — Local optimistic like toggle for notice cards in feed list
import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { api } from '../api/client';
import type { LikeToggleResponse } from '../types/api';

export function useNoticeLikeToggle(seq: number, initialLikeCnt: number) {
  const [liked, setLiked] = useState(false);
  const [likeCnt, setLikeCnt] = useState(initialLikeCnt);

  const mutation = useMutation({
    mutationFn: () => api.post<LikeToggleResponse>(`/api/feed/${seq}/like`),

    onMutate: () => {
      const prevLiked = liked;
      const prevLikeCnt = likeCnt;
      setLiked(!prevLiked);
      setLikeCnt(prevLiked ? prevLikeCnt - 1 : prevLikeCnt + 1);
      return { prevLiked, prevLikeCnt };
    },

    onSuccess: (data) => {
      setLiked(data.liked);
      setLikeCnt(data.likeCnt);
    },

    onError: (_err, _vars, context) => {
      if (context) {
        setLiked(context.prevLiked);
        setLikeCnt(context.prevLikeCnt);
      }
    },
  });

  return { liked, likeCnt, toggle: mutation.mutate };
}
