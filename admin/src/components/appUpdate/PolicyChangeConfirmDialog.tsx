// PolicyChangeConfirmDialog — blocks a lockout-causing save until rollout is confirmed
import { useState } from 'react';
import { ConfirmDialog } from '../ui/ConfirmDialog.tsx';

interface PolicyChangeConfirmDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  platformLabel: string;
  minBuild: number;
  isPending?: boolean;
  onConfirm: () => void;
}

export function PolicyChangeConfirmDialog({
  open,
  onOpenChange,
  platformLabel,
  minBuild,
  isPending,
  onConfirm,
}: PolicyChangeConfirmDialogProps) {
  const [isRolloutConfirmed, setIsRolloutConfirmed] = useState(false);

  const handleOpenChange = (next: boolean) => {
    if (!next) setIsRolloutConfirmed(false);
    onOpenChange(next);
  };

  return (
    <ConfirmDialog
      open={open}
      onOpenChange={handleOpenChange}
      title={`${platformLabel} 강제 업데이트 적용`}
      description={`빌드 ${minBuild} 미만의 ${platformLabel} 앱은 저장 즉시 사용이 차단됩니다.`}
      confirmLabel="적용"
      variant="destructive"
      isPending={isPending}
      confirmDisabled={!isRolloutConfirmed}
      onConfirm={onConfirm}
    >
      <label className="mt-4 flex items-start gap-2 rounded-xl bg-warning-subtle p-3 text-sm text-warning-text">
        <input
          type="checkbox"
          className="mt-0.5 h-4 w-4"
          checked={isRolloutConfirmed}
          onChange={(event) => setIsRolloutConfirmed(event.target.checked)}
        />
        <span>
          해당 빌드가 스토어에 100% 배포 완료되었음을 확인했습니다. 단계적 출시가 진행 중이면 업데이트를
          받을 수 없는 사용자가 차단됩니다.
        </span>
      </label>
    </ConfirmDialog>
  );
}
