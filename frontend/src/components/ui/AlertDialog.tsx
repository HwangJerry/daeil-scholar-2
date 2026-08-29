// AlertDialog — 단일 확인 버튼이 있는 알림 모달 (안내 문구 표시용)
import { CheckCircle2, type LucideIcon } from 'lucide-react';
import { cn } from '../../lib/utils';
import { Modal } from './Modal';
import { Button } from './Button';

type AlertDialogIconTone = 'success' | 'warning';

interface AlertDialogContentProps {
  title: string;
  message: string;
  confirmLabel?: string;
  icon?: LucideIcon;
  iconTone?: AlertDialogIconTone;
  onConfirm: () => void;
}

interface AlertDialogProps extends AlertDialogContentProps {
  open: boolean;
}

const ICON_TONE_CLASS_NAMES: Record<
  AlertDialogIconTone,
  { container: string; icon: string }
> = {
  success: {
    container: 'bg-success-subtle',
    icon: 'text-success-text',
  },
  warning: {
    container: 'bg-warning-subtle',
    icon: 'text-warning-text',
  },
};

export function AlertDialogContent({
  title,
  message,
  confirmLabel = '확인',
  icon: Icon = CheckCircle2,
  iconTone = 'success',
  onConfirm,
}: AlertDialogContentProps) {
  const iconToneClassNames = ICON_TONE_CLASS_NAMES[iconTone];

  return (
    <div className="flex flex-col items-center gap-4 px-6 py-8 text-center">
      <div
        className={cn(
          'flex size-12 items-center justify-center rounded-full',
          iconToneClassNames.container,
        )}
      >
        <Icon className={cn('size-6', iconToneClassNames.icon)} />
      </div>
      <div className="space-y-1.5">
        <h2 className="text-base font-semibold text-text-primary">{title}</h2>
        <p className="text-sm text-text-tertiary">{message}</p>
      </div>
      <Button onClick={onConfirm} className="mt-2 w-full">
        {confirmLabel}
      </Button>
    </div>
  );
}

export function AlertDialog({ open, ...contentProps }: AlertDialogProps) {
  if (!open) return null;

  return (
    <Modal onClose={contentProps.onConfirm} maxWidth="max-w-sm">
      <AlertDialogContent {...contentProps} />
    </Modal>
  );
}
