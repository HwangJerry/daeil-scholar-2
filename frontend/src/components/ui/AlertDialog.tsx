// AlertDialog — 단일 확인 버튼이 있는 알림 모달 (안내 문구 표시용)
import { CheckCircle2 } from 'lucide-react';
import { Modal } from './Modal';
import { Button } from './Button';

interface AlertDialogProps {
  open: boolean;
  title: string;
  message: string;
  confirmLabel?: string;
  onConfirm: () => void;
}

export function AlertDialog({
  open,
  title,
  message,
  confirmLabel = '확인',
  onConfirm,
}: AlertDialogProps) {
  if (!open) return null;

  return (
    <Modal onClose={onConfirm} maxWidth="max-w-sm">
      <div className="flex flex-col items-center gap-4 px-6 py-8 text-center">
        <div className="flex h-12 w-12 items-center justify-center rounded-full bg-success-subtle">
          <CheckCircle2 className="h-6 w-6 text-success-text" />
        </div>
        <div className="space-y-1.5">
          <h2 className="text-base font-semibold text-text-primary">{title}</h2>
          <p className="text-sm text-text-muted">{message}</p>
        </div>
        <Button onClick={onConfirm} className="mt-2 w-full">
          {confirmLabel}
        </Button>
      </div>
    </Modal>
  );
}
