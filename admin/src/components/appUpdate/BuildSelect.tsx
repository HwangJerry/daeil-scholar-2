// BuildSelect — threshold picker backed by build numbers the API has actually observed
import { useState } from 'react';
import { Input } from '../ui/Input.tsx';
import { Select } from '../ui/Select.tsx';
import { useAppClientBuilds } from '../../hooks/useAppClientBuilds.ts';
import type { AppPlatform } from '../../types/appUpdate.ts';
import { formatPolicyDateTime } from './formatPolicyDateTime.ts';

const MANUAL_OPTION = 'manual';

interface BuildSelectProps {
  id: string;
  platform: AppPlatform;
  value: number;
  disabled?: boolean;
  errorMessage?: string;
  /** isUnobserved is true when the chosen build has never been reported by a client. */
  onChange: (build: number, isUnobserved: boolean) => void;
}

export function BuildSelect({
  id,
  platform,
  value,
  disabled,
  errorMessage,
  onChange,
}: BuildSelectProps) {
  const buildsQuery = useAppClientBuilds(platform);
  const builds = buildsQuery.data?.builds ?? [];
  const isObserved = builds.some((build) => build.build === value);
  const [isManual, setIsManual] = useState(false);
  const showsManualInput = isManual || (value > 0 && !isObserved && !buildsQuery.isLoading);
  const errorId = `${id}-error`;

  // The server refuses a threshold no client has reported unless the
  // administrator confirms it, so report what the chosen build actually is
  // rather than which input produced it.
  const reportChange = (build: number) => {
    onChange(build, build > 0 && !builds.some((observed) => observed.build === build));
  };

  const handleSelectChange = (selected: string) => {
    if (selected === MANUAL_OPTION) {
      setIsManual(true);
      reportChange(value);
      return;
    }
    setIsManual(false);
    reportChange(Number(selected));
  };

  return (
    <div>
      <Select
        id={id}
        value={showsManualInput ? MANUAL_OPTION : String(value)}
        disabled={disabled}
        aria-describedby={errorMessage ? errorId : undefined}
        onChange={(event) => handleSelectChange(event.target.value)}
      >
        <option value="0">선택 안 함</option>
        {builds.map((build) => (
          <option key={build.build} value={build.build}>
            {build.versionName || '버전명 없음'} ({build.build}) · 최근 {formatPolicyDateTime(build.lastSeenAt)}
          </option>
        ))}
        <option value={MANUAL_OPTION}>직접 입력</option>
      </Select>

      {showsManualInput && (
        <div className="mt-2">
          <Input
            aria-label="빌드 번호 직접 입력"
            inputMode="numeric"
            value={value === 0 ? '' : String(value)}
            disabled={disabled}
            onChange={(event) => {
              // Latch manual mode so clearing the field does not unmount the input
              // mid-edit and snap the select back to "선택 안 함".
              setIsManual(true);
              reportChange(Number(event.target.value.replace(/\D/g, '')) || 0);
            }}
          />
          <p className="mt-1.5 text-xs text-warning-text">
            아직 확인된 적 없는 빌드입니다. 스토어에 배포된 빌드가 맞는지 확인해 주세요.
          </p>
        </div>
      )}

      {buildsQuery.isError && (
        <p className="mt-1.5 text-xs text-cool-gray">
          확인된 빌드 목록을 불러오지 못했습니다. 직접 입력으로 지정할 수 있습니다.
        </p>
      )}
      {errorMessage && (
        <p id={errorId} role="alert" className="mt-1.5 text-sm text-error-text">
          {errorMessage}
        </p>
      )}
    </div>
  );
}
