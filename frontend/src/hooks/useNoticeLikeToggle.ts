// useNoticeLikeToggle — Local optimistic like toggle for notice cards in feed list
import { useState } from 'react';
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
  // State (not ref) so it can be read during render without violating react-hooks/refs.
  const [isMutating, setIsMutating] = useState(false);

  // React 19 pattern: sync state from props during render when the source prop changes,
  // bypassing the cascading-render effect. The mutation guard ensures optimistic updates
  // (which intentionally diverge from the prop) aren't clobbered by an in-flight refetch.
  const [prevInitialLiked, setPrevInitialLiked] = useState(initialLiked);
  if (prevInitialLiked !== initialLiked) {
    setPrevInitialLiked(initialLiked);
    if (!isMutating) {
      setLiked(initialLiked);
    }
  }

  const [prevInitialLikeCnt, setPrevInitialLikeCnt] = useState(initialLikeCnt);
  if (prevInitialLikeCnt !== initialLikeCnt) {
    setPrevInitialLikeCnt(initialLikeCnt);
    if (!isMutating) {
      setLikeCnt(initialLikeCnt);
    }
  }

  const mutation = useMutation({
    mutationFn: () => api.post<LikeToggleResponse>(`/api/feed/${seq}/like`),

    onMutate: () => {
      setIsMutating(true);
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
      setIsMutating(false);
    },
  });

  return { liked, likeCnt, toggle: mutation.mutate };
}
