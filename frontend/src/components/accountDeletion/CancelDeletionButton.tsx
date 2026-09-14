// CancelDeletionButton — Explicit cancellation confirmation that also works in app webviews.
import { useId, useRef, useState } from 'react';
import { Button } from '../ui/Button';

interface CancelDeletionButtonProps {
  disabled: boolean;
  cancelling: boolean;
  onConfirm: () => Promise<void>;
}

export function CancelDeletionButton({ disabled, cancelling, onConfirm }: CancelDeletionButtonProps) {
  const [confirming, setConfirming] = useState(false);
  const headingId = useId();
  const trigger = useRef<HTMLButtonElement>(null);
  const submitting = useRef(false);

  async function confirmCancellation() {
    if (disabled || submitting.current) return;
    submitting.current = true;
    try {
      await onConfirm();
    } finally {
      submitting.current = false;
      setConfirming(false);
      trigger.current?.focus();
    }
  }

  return (
    <div className="space-y-4">
      <Button ref={trigger} type="button" disabled={disabled || cancelling || confirming} onClick={() => setConfirming(true)}>
        {cancelling ? '취소 확인 중…' : '탈퇴 신청 취소'}
      </Button>
      {confirming && (
        <div role="group" aria-labelledby={headingId} className="space-y-3 rounded-xl border border-border p-4">
          <h3 id={headingId} className="font-semibold text-text-primary">탈퇴 신청을 취소할까요?</h3>
          <p className="text-sm leading-7 text-text-secondary">취소하면 계정을 다시 이용할 수 있습니다. 취소 완료 후 다시 로그인해 주세요.</p>
          <div className="flex flex-wrap gap-3">
            <Button autoFocus type="button" variant="outline" disabled={cancelling} onClick={() => { setConfirming(false); trigger.current?.focus(); }}>
              신청 유지
            </Button>
            <Button type="button" disabled={disabled || cancelling} onClick={() => void confirmCancellation()}>
              {cancelling ? '취소 확인 중…' : '신청 취소 확인'}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
