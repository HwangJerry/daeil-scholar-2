// useMessageComposition — Manages content state, send mutation, and success timer for message compose
import { useState, useRef, useEffect } from 'react';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { sendMessage } from '../api/messages';
import { ApiClientError } from '../api/client';

const MAX_CONTENT_LENGTH = 1000;
const SUCCESS_DISPLAY_DURATION_MS = 1500;

interface UseMessageCompositionOptions {
  recipientSeq: number;
  onSuccess: () => void;
}

export function useMessageComposition({ recipientSeq, onSuccess }: UseMessageCompositionOptions) {
  const [content, setContent] = useState('');
  const [showSuccess, setShowSuccess] = useState(false);
  const queryClient = useQueryClient();
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => {
    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, []);

  const sendMutation = useMutation({
    mutationFn: () => sendMessage(recipientSeq, content),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['messages'] });
      setContent('');
      setShowSuccess(true);
      timerRef.current = setTimeout(() => {
        setShowSuccess(false);
        onSuccess();
      }, SUCCESS_DISPLAY_DURATION_MS);
    },
  });

  const submit = () => {
    if (content.trim().length === 0) return;
    sendMutation.mutate();
  };

  const errorMessage = sendMutation.isError
    ? sendMutation.error instanceof ApiClientError
      ? sendMutation.error.message
      : '전송에 실패했습니다. 다시 시도해주세요.'
    : null;

  return {
    content,
    setContent,
    submit,
    showSuccess,
    isPending: sendMutation.isPending,
    isContentEmpty: content.trim().length === 0,
    errorMessage,
    maxLength: MAX_CONTENT_LENGTH,
  };
}
