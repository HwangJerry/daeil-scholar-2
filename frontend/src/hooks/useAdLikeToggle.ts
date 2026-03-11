// useAdLikeToggle — Local optimistic like toggle for ad cards
import { useState } from 'react';
import { useMutation } from '@tanstack/react-query';
import { api, ApiClientError } from '../api/client';
import type { LikeToggleResponse } from '../types/api';

interface UseAdLikeToggleOptions {
  onAuthError?: () => void;
}

export function useAdLikeToggle(maSeq: number, initialLikeCnt: number, initialLiked: boolean, options?: UseAdLikeToggleOptions) {
  const [liked, setLiked] = useState(initialLiked);
  const [likeCnt, setLikeCnt] = useState(initialLikeCnt);

  const mutation = useMutation({
    mutationFn: () => api.post<LikeToggleResponse>(`/api/ad/${maSeq}/like`),

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
      if (_err instanceof ApiClientError && _err.status === 401 && options?.onAuthError) {
        options.onAuthError();
      }
    },
  });

  return { liked, likeCnt, toggle: mutation.mutate };
}
