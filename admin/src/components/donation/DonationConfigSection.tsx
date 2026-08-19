// DonationConfigSection — inline form for editing donation display configuration
import { useState } from 'react';
import { Settings } from 'lucide-react';
import { Input } from '../ui/Input.tsx';
import { Textarea } from '../ui/Textarea.tsx';
import { Button } from '../ui/Button.tsx';
import { ErrorState } from '../ui/ErrorState.tsx';
import { useDonationConfig } from '../../hooks/useDonationConfig.ts';
import { formatAmount } from '../../lib/formatAmount.ts';
import type { DonationConfig, DonationConfigUpdateRequest } from '../../types/api.ts';

type TierThresholdKey =
  | 'tierSproutMin'
  | 'tierSaplingMin'
  | 'tierTreeMin'
  | 'tierBloomingMin'
  | 'tierFruitingMin';

type TierThresholdFormValues = Record<TierThresholdKey, string>;

const TIER_THRESHOLD_FIELDS: ReadonlyArray<{
  key: TierThresholdKey;
  configKey: keyof Pick<
    DonationConfig,
    | 'dcTierSproutMin'
    | 'dcTierSaplingMin'
    | 'dcTierTreeMin'
    | 'dcTierBloomingMin'
    | 'dcTierFruitingMin'
  >;
  label: string;
}> = [
  { key: 'tierSproutMin', configKey: 'dcTierSproutMin', label: '새싹' },
  { key: 'tierSaplingMin', configKey: 'dcTierSaplingMin', label: '어린 나무' },
  { key: 'tierTreeMin', configKey: 'dcTierTreeMin', label: '나무' },
  { key: 'tierBloomingMin', configKey: 'dcTierBloomingMin', label: '꽃이 핀 나무' },
  { key: 'tierFruitingMin', configKey: 'dcTierFruitingMin', label: '열매 맺은 나무' },
];

const NON_NEGATIVE_INTEGER_ERROR = '각 단계 금액은 0 이상의 정수여야 합니다.';
const TIER_ORDER_ERROR = '각 단계 금액은 이전 단계보다 커야 합니다.';

function createTierThresholdFormValues(data: DonationConfig): TierThresholdFormValues {
  return Object.fromEntries(
    TIER_THRESHOLD_FIELDS.map(({ key, configKey }) => [key, String(data[configKey])]),
  ) as TierThresholdFormValues;
}

function parseTierThresholds(values: TierThresholdFormValues) {
  return {
    tierSproutMin: Number(values.tierSproutMin),
    tierSaplingMin: Number(values.tierSaplingMin),
    tierTreeMin: Number(values.tierTreeMin),
    tierBloomingMin: Number(values.tierBloomingMin),
    tierFruitingMin: Number(values.tierFruitingMin),
  };
}

function validateTierThresholds(thresholds: ReturnType<typeof parseTierThresholds>): string | null {
  const amounts = TIER_THRESHOLD_FIELDS.map(({ key }) => thresholds[key]);
  const areNonNegativeIntegers = amounts.every(
    (amount) => Number.isInteger(amount) && amount >= 0,
  );
  if (!areNonNegativeIntegers) return NON_NEGATIVE_INTEGER_ERROR;

  const areStrictlyIncreasing = amounts
    .slice(1)
    .every((amount, index) => amounts[index] < amount);
  return areStrictlyIncreasing ? null : TIER_ORDER_ERROR;
}

export function DonationConfigSection() {
  const { data, isLoading, isError, refetch, update, isUpdating } = useDonationConfig();

  if (isError) return <ErrorState onRetry={() => void refetch()} />;
  if (isLoading || !data) {
    return (
      <div className="rounded-2xl border border-border-light bg-white p-6 shadow-sm">
        <p className="text-sm text-cool-gray">로딩 중...</p>
      </div>
    );
  }

  // key={data.dcSeq} remounts the form when the underlying config row changes
  // so initial form state derives cleanly from props without a sync effect.
  return (
    <DonationConfigForm
      key={data.dcSeq}
      data={data}
      onSave={update}
      isUpdating={isUpdating}
    />
  );
}

interface DonationConfigFormProps {
  data: DonationConfig;
  onSave: (
    payload: DonationConfigUpdateRequest,
    options?: { onSuccess?: () => void },
  ) => void;
  isUpdating: boolean;
}

function DonationConfigForm({ data, onSave, isUpdating }: DonationConfigFormProps) {
  const [goal, setGoal] = useState(() => String(data.dcGoal));
  const [manualAdj, setManualAdj] = useState(() => String(data.dcManualAdj));
  const [manualDonorCnt, setManualDonorCnt] = useState(() => String(data.dcManualDonorCnt));
  const [tierThresholds, setTierThresholds] = useState(() => createTierThresholdFormValues(data));
  const [tierThresholdError, setTierThresholdError] = useState<string | null>(null);
  const [note, setNote] = useState(() => data.dcNote ?? '');
  const [overwrite, setOverwrite] = useState(() => data.dcOverwrite === 'Y');
  const [isEditing, setIsEditing] = useState(false);

  const handleSave = () => {
    const parsedTierThresholds = parseTierThresholds(tierThresholds);
    const validationError = validateTierThresholds(parsedTierThresholds);
    if (validationError) {
      setTierThresholdError(validationError);
      return;
    }

    onSave(
      {
        goal: Number(goal),
        manualAdj: Number(manualAdj),
        manualDonorCnt: Number(manualDonorCnt),
        ...parsedTierThresholds,
        note,
        overwrite,
      },
      { onSuccess: () => setIsEditing(false) },
    );
  };

  const handleCancel = () => {
    setGoal(String(data.dcGoal));
    setManualAdj(String(data.dcManualAdj));
    setManualDonorCnt(String(data.dcManualDonorCnt));
    setTierThresholds(createTierThresholdFormValues(data));
    setTierThresholdError(null);
    setNote(data.dcNote ?? '');
    setOverwrite(data.dcOverwrite === 'Y');
    setIsEditing(false);
  };

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

      {isEditing ? (
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
          </div>
          <div>
            <label htmlFor="cfg-donor-cnt" className="mb-1 block text-sm font-medium text-dark-slate">
              기부자 수 (수동)
            </label>
            <Input
              id="cfg-donor-cnt"
              type="number"
              value={manualDonorCnt}
              onChange={(e) => setManualDonorCnt(e.target.value)}
              min={0}
            />
            <label className="mt-2 flex cursor-pointer items-center gap-2 text-sm text-dark-slate">
              <input
                type="checkbox"
                checked={overwrite}
                onChange={(e) => setOverwrite(e.target.checked)}
                className="h-4 w-4 rounded border-gray-300 accent-indigo-600"
              />
              덮어쓰기 — 체크 시 누적 금액·기부자 수를 수동 입력값으로 표시
            </label>
          </div>
          <fieldset>
            <legend className="text-sm font-semibold text-dark-slate">나무 성장 단계 임계값</legend>
            <p className="mt-1 text-xs text-cool-gray">
              씨앗 단계는 0원으로 고정됩니다. 이후 단계의 시작 금액을 원 단위로 입력해 주세요.
            </p>
            <div className="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-5">
              {TIER_THRESHOLD_FIELDS.map(({ key, label }) => (
                <div key={key}>
                  <label htmlFor={`cfg-${key}`} className="mb-1 block text-sm font-medium text-dark-slate">
                    {label} 기준 금액 (원)
                  </label>
                  <Input
                    id={`cfg-${key}`}
                    type="number"
                    value={tierThresholds[key]}
                    onChange={(event) => {
                      setTierThresholds((current) => ({ ...current, [key]: event.target.value }));
                      setTierThresholdError(null);
                    }}
                    min={0}
                    step={1}
                    aria-invalid={Boolean(tierThresholdError)}
                    aria-describedby={tierThresholdError ? 'cfg-tier-error' : undefined}
                  />
                </div>
              ))}
            </div>
            {tierThresholdError && (
              <p id="cfg-tier-error" role="alert" className="mt-2 text-sm text-red-600">
                {tierThresholdError}
              </p>
            )}
          </fieldset>
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
        <div className="space-y-5">
          <dl className="grid grid-cols-1 gap-3 sm:grid-cols-5">
            <div>
              <dt className="text-xs text-cool-gray">목표금액</dt>
              <dd className="text-sm font-medium text-dark-slate">₩{formatAmount(data.dcGoal)}</dd>
            </div>
            <div>
              <dt className="text-xs text-cool-gray">수동 조정액</dt>
              <dd className="text-sm font-medium text-dark-slate">₩{formatAmount(data.dcManualAdj)}</dd>
            </div>
            <div>
              <dt className="text-xs text-cool-gray">기부자 수(수동)</dt>
              <dd className="text-sm font-medium text-dark-slate">{data.dcManualDonorCnt.toLocaleString()}명</dd>
            </div>
            <div>
              <dt className="text-xs text-cool-gray">덮어쓰기</dt>
              <dd className="text-sm font-medium text-dark-slate">{data.dcOverwrite === 'Y' ? '✓ 적용 중' : '—'}</dd>
            </div>
            <div>
              <dt className="text-xs text-cool-gray">메모</dt>
              <dd className="text-sm text-dark-slate">{data.dcNote || '—'}</dd>
            </div>
          </dl>
          <div className="border-t border-border-light pt-4">
            <h4 className="text-sm font-semibold text-dark-slate">나무 성장 단계 임계값</h4>
            <dl className="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-5">
              {TIER_THRESHOLD_FIELDS.map(({ configKey, label }) => (
                <div key={configKey}>
                  <dt className="text-xs text-cool-gray">{label}</dt>
                  <dd className="text-sm font-medium text-dark-slate">
                    ₩{formatAmount(data[configKey])}
                  </dd>
                </div>
              ))}
            </dl>
          </div>
        </div>
      )}
    </div>
  );
}
