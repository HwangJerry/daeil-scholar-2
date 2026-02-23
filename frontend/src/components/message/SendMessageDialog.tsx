// SendMessageDialog — Modal overlay for composing and sending a message to an alumni member
import { useState } from 'react';
import { createPortal } from 'react-dom';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { X, Send, CheckCircle } from 'lucide-react';
import { api } from '../../api/client';
import { Button } from '../ui/Button';

interface SendMessageDialogProps {
  recipientSeq: number;
  recipientName: string;
  onClose: () => void;
}

const MAX_CONTENT_LENGTH = 1000;
const SUCCESS_DISPLAY_DURATION_MS = 1500;

export function SendMessageDialog({
  recipientSeq,
  recipientName,
  onClose,
}: SendMessageDialogProps) {
  const [content, setContent] = useState('');
  const [showSuccess, setShowSuccess] = useState(false);
  const queryClient = useQueryClient();

  const sendMutation = useMutation({
    mutationFn: () =>
      api.post('/api/messages', {
        recvrSeq: recipientSeq,
        content,
      }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['messages'] });
      setShowSuccess(true);
      setTimeout(() => {
        onClose();
      }, SUCCESS_DISPLAY_DURATION_MS);
    },
  });

  const handleSubmit = () => {
    const trimmed = content.trim();
    if (trimmed.length === 0) return;
    sendMutation.mutate();
  };

  const handleBackdropClick = (e: React.MouseEvent<HTMLDivElement>) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  const isContentEmpty = content.trim().length === 0;

  return createPortal(
    <div
      className="fixed inset-0 z-50 flex items-center justify-center px-4"
      onClick={handleBackdropClick}
    >
      {/* Backdrop */}
      <div className="absolute inset-0 bg-black/40 backdrop-blur-sm" />

      {/* Dialog */}
      <div className="relative w-full max-w-md rounded-[20px] bg-surface p-6 shadow-float animate-fade-in-up border border-border-subtle">
        {/* Header */}
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-lg font-bold text-text-primary font-serif">
            쪽지 보내기
          </h2>
          <button
            onClick={onClose}
            className="rounded-lg p-1.5 text-text-tertiary hover:bg-background transition-colors duration-150"
            aria-label="닫기"
          >
            <X size={20} />
          </button>
        </div>

        {/* Recipient */}
        <div className="mb-4 rounded-xl bg-background px-3 py-2.5">
          <span className="text-xs text-text-placeholder">받는 사람</span>
          <p className="text-sm font-semibold text-text-primary mt-0.5 font-serif">
            {recipientName}
          </p>
        </div>

        {/* Success state */}
        {showSuccess ? (
          <div className="flex flex-col items-center justify-center py-8 gap-3">
            <CheckCircle size={40} className="text-success" />
            <p className="text-sm font-medium text-text-primary">
              쪽지를 보냈습니다
            </p>
          </div>
        ) : (
          <>
            {/* Textarea */}
            <div className="mb-4">
              <textarea
                value={content}
                onChange={(e) => setContent(e.target.value)}
                maxLength={MAX_CONTENT_LENGTH}
                rows={6}
                placeholder="쪽지 내용을 입력하세요"
                className="w-full rounded-xl border border-border bg-background px-3 py-2.5 text-sm text-text-secondary outline-none transition-shadow duration-150 placeholder:text-text-placeholder focus:ring-2 focus:ring-primary/15 focus:border-primary/30 resize-none"
                disabled={sendMutation.isPending}
              />
              <div className="flex justify-end mt-1.5">
                <span className="text-xs text-text-placeholder">
                  {content.length} / {MAX_CONTENT_LENGTH}
                </span>
              </div>
            </div>

            {/* Error message */}
            {sendMutation.isError && (
              <p className="mb-3 text-xs text-error">
                전송에 실패했습니다. 다시 시도해주세요.
              </p>
            )}

            {/* Actions */}
            <div className="flex gap-2 justify-end">
              <Button
                variant="outline"
                size="sm"
                onClick={onClose}
                disabled={sendMutation.isPending}
              >
                취소
              </Button>
              <Button
                size="sm"
                onClick={handleSubmit}
                disabled={isContentEmpty || sendMutation.isPending}
              >
                {sendMutation.isPending ? (
                  <div className="h-4 w-4 animate-spin rounded-full border-2 border-white border-t-transparent" />
                ) : (
                  <>
                    <Send size={14} />
                    보내기
                  </>
                )}
              </Button>
            </div>
          </>
        )}
      </div>
    </div>,
    document.body
  );
}
