// PlatformPolicyCard — independent update policy form for one platform
import { useState, type FormEvent } from 'react';
import { Badge } from '../ui/Badge.tsx';
import { Button } from '../ui/Button.tsx';
import { Input } from '../ui/Input.tsx';
import { useSaveAppUpdatePolicy } from '../../hooks/useSaveAppUpdatePolicy.ts';
import { policyErrorMessage, policyFieldErrors } from '../../lib/appUpdatePolicyErrors.ts';
import type { AppPlatform, AppUpdatePolicy, AppUpdatePolicyRecord } from '../../types/appUpdate.ts';
import { BuildSelect } from './BuildSelect.tsx';
import { formatOperator, formatPolicyDateTime } from './formatPolicyDateTime.ts';
import { PolicyChangeConfirmDialog } from './PolicyChangeConfirmDialog.tsx';
import { PolicyHistoryList } from './PolicyHistoryList.tsx';

const PLATFORM_LABELS: Record<AppPlatform, string> = { ios: 'iOS', android: 'Android' };
const OS_FIELD_LABELS: Record<AppPlatform, string> = {
  ios: '최소 지원 iOS 버전',
  android: '최소 지원 API 레벨',
};
const OS_FIELD_HINTS: Record<AppPlatform, string> = {
  ios: '현재 스토어 최신 빌드가 요구하는 iOS 버전입니다. 예: 17.0',
  android: '현재 스토어 최신 빌드의 minSdk입니다. OS 버전이 아니라 API 레벨입니다. 예: 26',
};

function PolicyStatusBadge({ policy }: { policy: AppUpdatePolicy }) {
  if (policy.forceEnabled) return <Badge variant="danger">강제 업데이트 사용 중</Badge>;
  if (policy.recommendEnabled) return <Badge variant="warning">권장 업데이트 사용 중</Badge>;
  return <Badge variant="muted">사용 안 함</Badge>;
}

interface PlatformPolicyCardProps {
  record: AppUpdatePolicyRecord;
}

export function PlatformPolicyCard({ record }: PlatformPolicyCardProps) {
  const platformLabel = PLATFORM_LABELS[record.platform];
  const [policy, setPolicy] = useState<AppUpdatePolicy>(record.policy);
  const [unobservedBuilds, setUnobservedBuilds] = useState({ min: false, recommended: false });
  const allowUnobservedBuild = unobservedBuilds.min || unobservedBuilds.recommended;
  const [isConfirming, setIsConfirming] = useState(false);
  const savePolicy = useSaveAppUpdatePolicy(record.platform);
  const fieldErrors = policyFieldErrors(savePolicy.error);

  const isReadOnly = record.unavailable === true || savePolicy.isPending;
  // Turning force on, or raising the bar, is what can lock users out.
  const needsRolloutConfirmation =
    policy.forceEnabled && (!record.policy.forceEnabled || policy.minBuild > record.policy.minBuild);

  const updatePolicy = (changes: Partial<AppUpdatePolicy>) => {
    if (savePolicy.isError) savePolicy.reset();
    setPolicy((current) => ({ ...current, ...changes }));
  };

  const submitPolicy = () => {
    savePolicy.mutate(
      {
        platform: record.platform,
        policy,
        updatedAt: record.updatedAt,
        expectedPolicy: record.policy,
        allowUnobservedBuild,
      },
      { onSuccess: () => setIsConfirming(false) },
    );
  };

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (isReadOnly) return;
    if (needsRolloutConfirmation) {
      setIsConfirming(true);
      return;
    }
    submitPolicy();
  };

  if (record.unavailable) {
    return (
      <article className="rounded-2xl border border-border-light bg-surface p-5 shadow-sm md:p-6">
        <h3 className="font-semibold text-dark-slate">{platformLabel}</h3>
        <p className="mt-2 text-sm text-error-text">
          저장된 정책을 읽을 수 없습니다. 마이그레이션 적용 여부와 `app_settings` 행을 확인해 주세요.
        </p>
      </article>
    );
  }

  return (
    <article className="rounded-2xl border border-border-light bg-surface p-5 shadow-sm md:p-6">
      <form onSubmit={handleSubmit}>
        <div className="flex items-center justify-between gap-3">
          <h3 className="font-semibold text-dark-slate">{platformLabel}</h3>
          <PolicyStatusBadge policy={record.policy} />
        </div>

        <fieldset className="mt-5 space-y-3" disabled={isReadOnly}>
          <label className="flex items-center gap-2 text-sm font-medium text-dark-slate">
            <input
              type="checkbox"
              className="h-4 w-4"
              checked={policy.forceEnabled}
              onChange={(event) => updatePolicy({ forceEnabled: event.target.checked })}
            />
            강제 업데이트 사용
          </label>
          <div>
            <label
              htmlFor={`${record.platform}-min-build`}
              className="mb-1.5 block text-sm text-cool-gray"
            >
              최소 빌드 (이 빌드 미만이면 차단)
            </label>
            <BuildSelect
              id={`${record.platform}-min-build`}
              platform={record.platform}
              value={policy.minBuild}
              disabled={isReadOnly}
              errorMessage={fieldErrors.minBuild}
              onChange={(build, isUnobserved) => {
                updatePolicy({ minBuild: build });
                setUnobservedBuilds((current) => ({ ...current, min: isUnobserved }));
              }}
            />
          </div>

          <label className="flex items-center gap-2 pt-2 text-sm font-medium text-dark-slate">
            <input
              type="checkbox"
              className="h-4 w-4"
              checked={policy.recommendEnabled}
              onChange={(event) => updatePolicy({ recommendEnabled: event.target.checked })}
            />
            권장 업데이트 사용
          </label>
          <div>
            <label
              htmlFor={`${record.platform}-recommended-build`}
              className="mb-1.5 block text-sm text-cool-gray"
            >
              권장 빌드 (이 빌드 미만이면 안내)
            </label>
            <BuildSelect
              id={`${record.platform}-recommended-build`}
              platform={record.platform}
              value={policy.recommendedBuild}
              disabled={isReadOnly}
              errorMessage={fieldErrors.recommendedBuild}
              onChange={(build, isUnobserved) => {
                updatePolicy({ recommendedBuild: build });
                setUnobservedBuilds((current) => ({ ...current, recommended: isUnobserved }));
              }}
            />
          </div>

          <div className="pt-2">
            <label
              htmlFor={`${record.platform}-min-os`}
              className="mb-1.5 block text-sm font-medium text-dark-slate"
            >
              {OS_FIELD_LABELS[record.platform]}
            </label>
            <Input
              id={`${record.platform}-min-os`}
              value={policy.minOsVersion}
              inputMode="numeric"
              onChange={(event) => updatePolicy({ minOsVersion: event.target.value })}
            />
            <p className="mt-1.5 text-xs text-cool-gray">{OS_FIELD_HINTS[record.platform]}</p>
            {fieldErrors.minOsVersion && (
              <p role="alert" className="mt-1.5 text-sm text-error-text">
                {fieldErrors.minOsVersion}
              </p>
            )}
          </div>

          {record.platform === 'ios' && (
            <div className="pt-2">
              <label
                htmlFor="ios-store-url"
                className="mb-1.5 block text-sm font-medium text-dark-slate"
              >
                App Store URL
              </label>
              <Input
                id="ios-store-url"
                value={policy.storeUrl ?? ''}
                inputMode="url"
                autoCapitalize="none"
                autoCorrect="off"
                spellCheck={false}
                placeholder="https://apps.apple.com/app/id..."
                onChange={(event) => updatePolicy({ storeUrl: event.target.value })}
              />
              {fieldErrors.storeUrl && (
                <p role="alert" className="mt-1.5 text-sm text-error-text">
                  {fieldErrors.storeUrl}
                </p>
              )}
            </div>
          )}
        </fieldset>

        {savePolicy.isError && (
          <p role="alert" className="mt-4 text-sm text-error-text">
            {policyErrorMessage(savePolicy.error)}
          </p>
        )}

        <div className="mt-5 flex flex-col gap-4 border-t border-border-subtle pt-4 sm:flex-row sm:items-end sm:justify-between">
          <dl className="space-y-1 text-xs text-cool-gray">
            <div className="flex gap-1.5">
              <dt>마지막 수정</dt>
              <dd>{formatPolicyDateTime(record.updatedAt)}</dd>
            </div>
            <div className="flex gap-1.5">
              <dt>수정자</dt>
              <dd>{formatOperator(record.updatedBy)}</dd>
            </div>
          </dl>
          <Button type="submit" className="w-full sm:w-auto" disabled={isReadOnly}>
            {savePolicy.isPending ? '저장 중...' : `${platformLabel} 저장`}
          </Button>
        </div>
      </form>

      <PolicyHistoryList platform={record.platform} />

      <PolicyChangeConfirmDialog
        open={isConfirming}
        onOpenChange={setIsConfirming}
        platformLabel={platformLabel}
        minBuild={policy.minBuild}
        isPending={savePolicy.isPending}
        onConfirm={submitPolicy}
      />
    </article>
  );
}
