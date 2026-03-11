// useNoticeLikeToggle — Local optimistic like toggle for notice cards in feed list
import { useEffect, useRef, useState } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api, ApiClientError } from '../api/client';
import type { LikeToggleResponse } from '../types/api';

interface UseNoticeLikeToggleOptions {
  onAuthError?: () => void;
}

export function useNoticeLikeToggle(
  seq: number,
  initialLikeCnt: number,
  initialLiked: boolean,
  options?: UseNoticeLikeToggleOptions,
) {
  const queryClient = useQueryClient();
  const [liked, setLiked] = useState(initialLiked);
  const [likeCnt, setLikeCnt] = useState(initialLikeCnt);
  const isMutating = useRef(false);

  useEffect(() => {
    if (!isMutating.current) {
      setLiked(initialLiked);
    }
  }, [initialLiked]);

  useEffect(() => {
    if (!isMutating.current) {
      setLikeCnt(initialLikeCnt);
    }
  }, [initialLikeCnt]);

  const mutation = useMutation({
    mutationFn: () => api.post<LikeToggleResponse>(`/api/feed/${seq}/like`),

    onMutate: () => {
      isMutating.current = true;
      const prevLiked = liked;
      const prevLikeCnt = likeCnt;
      setLiked(!prevLiked);
      setLikeCnt(prevLiked ? prevLikeCnt - 1 : prevLikeCnt + 1);
      return { prevLiked, prevLikeCnt };
    },

    onSuccess: (data) => {
      setLiked(data.liked);
      setLikeCnt(data.likeCnt);
      queryClient.invalidateQueries({ queryKey: ['feed'] });
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

    onSettled: () => {
      isMutating.current = false;
    },
  });

  return { liked, likeCnt, toggle: mutation.mutate };
}
