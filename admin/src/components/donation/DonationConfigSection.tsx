// DonationConfigSection — inline form for editing donation goal, manual adjustment, and memo
import { useState, useEffect } from 'react';
import { Settings } from 'lucide-react';
import { Input } from '../ui/Input.tsx';
import { Textarea } from '../ui/Textarea.tsx';
import { Button } from '../ui/Button.tsx';
import { ErrorState } from '../ui/ErrorState.tsx';
import { useDonationConfig } from '../../hooks/useDonationConfig.ts';
import { formatAmount } from '../../lib/formatAmount.ts';

export function DonationConfigSection() {
  const { data, isLoading, isError, refetch, update, isUpdating } = useDonationConfig();

  const [goal, setGoal] = useState('');
  const [manualAdj, setManualAdj] = useState('');
  const [note, setNote] = useState('');
  const [overwrite, setOverwrite] = useState(false);
  const [isEditing, setIsEditing] = useState(false);

  useEffect(() => {
    if (data) {
      setGoal(String(data.dcGoal));
      setManualAdj(String(data.dcManualAdj));
      setNote(data.dcNote ?? '');
      setOverwrite(data.dcOverwrite === 'Y');
    }
  }, [data]);

  const handleSave = () => {
    update(
      { goal: Number(goal), manualAdj: Number(manualAdj), note, overwrite },
      { onSuccess: () => setIsEditing(false) },
    );
  };

  const handleCancel = () => {
    if (data) {
      setGoal(String(data.dcGoal));
      setManualAdj(String(data.dcManualAdj));
      setNote(data.dcNote ?? '');
      setOverwrite(data.dcOverwrite === 'Y');
    }
    setIsEditing(false);
  };

  if (isError) return <ErrorState onRetry={() => void refetch()} />;

  return (
    <div className="rounded-2xl border border-border-light bg-white p-6 shadow-sm">
      <div className="mb-4 flex items-center justify-between">
        <h3 className="flex items-center gap-2 font-semibold text-dark-slate">
          <Settings className="h-4 w-4" />
          기부 설정
        </h3>
        {!isEditing && (
          <Button variant="outline" size="sm" onClick={() => setIsEditing(true)}>
            수정
          </Button>
        )}
      </div>

      {isLoading ? (
        <p className="text-sm text-cool-gray">로딩 중...</p>
      ) : isEditing ? (
        <div className="space-y-4">
          <div>
            <label htmlFor="cfg-goal" className="mb-1 block text-sm font-medium text-dark-slate">
              목표금액 (원)
            </label>
            <Input
              id="cfg-goal"
              type="number"
              value={goal}
              onChange={(e) => setGoal(e.target.value)}
              min={0}
            />
          </div>
          <div>
            <label htmlFor="cfg-adj" className="mb-1 block text-sm font-medium text-dark-slate">
              수동 조정액 (원)
            </label>
            <Input
              id="cfg-adj"
              type="number"
              value={manualAdj}
              onChange={(e) => setManualAdj(e.target.value)}
            />
            <label className="mt-2 flex cursor-pointer items-center gap-2 text-sm text-dark-slate">
              <input
                type="checkbox"
                checked={overwrite}
                onChange={(e) => setOverwrite(e.target.checked)}
                className="h-4 w-4 rounded border-gray-300 accent-indigo-600"
              />
              덮어쓰기 — 체크 시 온라인 합산을 무시하고 수동 조정액을 누적 기부액으로 표시
            </label>
          </div>
          <div>
            <label htmlFor="cfg-note" className="mb-1 block text-sm font-medium text-dark-slate">
              메모
            </label>
            <Textarea
              id="cfg-note"
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder="관리자 메모..."
              rows={3}
            />
          </div>
          <div className="flex gap-2">
            <Button onClick={handleSave} disabled={isUpdating}>
              {isUpdating ? '저장 중...' : '저장'}
            </Button>
            <Button variant="outline" onClick={handleCancel} disabled={isUpdating}>
              취소
            </Button>
          </div>
        </div>
      ) : (
        <dl className="grid grid-cols-1 gap-3 sm:grid-cols-4">
          <div>
            <dt className="text-xs text-cool-gray">목표금액</dt>
            <dd className="text-sm font-medium text-dark-slate">₩{formatAmount(data?.dcGoal ?? 0)}</dd>
          </div>
          <div>
            <dt className="text-xs text-cool-gray">수동 조정액</dt>
            <dd className="text-sm font-medium text-dark-slate">₩{formatAmount(data?.dcManualAdj ?? 0)}</dd>
          </div>
          <div>
            <dt className="text-xs text-cool-gray">덮어쓰기</dt>
            <dd className="text-sm font-medium text-dark-slate">{data?.dcOverwrite === 'Y' ? '✓ 적용 중' : '—'}</dd>
          </div>
          <div>
            <dt className="text-xs text-cool-gray">메모</dt>
            <dd className="text-sm text-dark-slate">{data?.dcNote || '—'}</dd>
          </div>
        </dl>
      )}
    </div>
  );
}
