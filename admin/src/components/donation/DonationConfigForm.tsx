// DonationConfigForm — editable form for donation goal, manual adjustment, and note with a11y
import { useState } from 'react';
import { Button } from '../ui/Button.tsx';
import { Input } from '../ui/Input.tsx';
import { Textarea } from '../ui/Textarea.tsx';
import { useDonationConfig } from '../../hooks/useDonationConfig.ts';
import type { DonationConfig } from '../../types/api.ts';

function DonationConfigFormInner({
  config,
  onSave,
  isUpdating,
}: {
  config: DonationConfig;
  onSave: (data: { dcGoal: number; dcManualAdj: number; dcNote: string }) => void;
  isUpdating: boolean;
}) {
  const [goal, setGoal] = useState(String(config.dcGoal));
  const [manualAdj, setManualAdj] = useState(String(config.dcManualAdj));
  const [note, setNote] = useState(config.dcNote ?? '');

  const handleSave = () => {
    onSave({ dcGoal: Number(goal), dcManualAdj: Number(manualAdj), dcNote: note });
  };

  return (
    <div className="max-w-lg space-y-4 rounded-2xl border border-border-light bg-white p-6 shadow-sm">
      <div>
        <label htmlFor="dc-goal" className="mb-1 block text-sm font-medium text-dark-slate">목표 금액 (원)</label>
        <Input id="dc-goal" type="number" value={goal} onChange={(e) => setGoal(e.target.value)} />
      </div>
      <div>
        <label htmlFor="dc-manual-adj" className="mb-1 block text-sm font-medium text-dark-slate">수동 조정액 (원)</label>
        <Input id="dc-manual-adj" type="number" value={manualAdj} onChange={(e) => setManualAdj(e.target.value)} />
      </div>
      <div>
        <label htmlFor="dc-note" className="mb-1 block text-sm font-medium text-dark-slate">조정 메모</label>
        <Textarea id="dc-note" value={note} onChange={(e) => setNote(e.target.value)} placeholder="조정 사유를 입력하세요" />
      </div>
      <div className="flex justify-end">
        <Button onClick={handleSave} disabled={isUpdating}>
          {isUpdating ? '저장 중...' : '설정 저장'}
        </Button>
      </div>
    </div>
  );
}

export function DonationConfigForm() {
  const { config, isLoading, updateConfig, isUpdating } = useDonationConfig();

  if (isLoading || !config) {
    return <div className="py-8 text-center text-cool-gray">로딩 중...</div>;
  }

  return (
    <DonationConfigFormInner
      key={`${config.dcGoal}-${config.dcManualAdj}-${config.dcNote}`}
      config={config}
      onSave={updateConfig}
      isUpdating={isUpdating}
    />
  );
}
