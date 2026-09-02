// AppSettingsPage — card-based editor for remotely managed application settings
import { useState, type FormEvent } from 'react';
import { Clock3, Globe2, LockKeyhole, RefreshCw, Settings2 } from 'lucide-react';
import { ApiClientError } from '../api/client.ts';
import { Button } from '../components/ui/Button.tsx';
import { ErrorState } from '../components/ui/ErrorState.tsx';
import { Input } from '../components/ui/Input.tsx';
import { useAppSettings } from '../hooks/useAppSettings.ts';
import { useUpdateAppSetting } from '../hooks/useUpdateAppSetting.ts';
import type { AppSetting } from '../types/appSettings.ts';

const UPDATED_AT_FORMATTER = new Intl.DateTimeFormat('ko-KR', {
  dateStyle: 'medium',
  timeStyle: 'short',
  timeZone: 'Asia/Seoul',
});

function formatUpdatedAt(updatedAt: string) {
  const date = new Date(updatedAt);
  return Number.isNaN(date.getTime()) ? updatedAt : UPDATED_AT_FORMATTER.format(date);
}

function getUpdateErrorMessage(error: Error | null) {
  if (error instanceof ApiClientError && error.code === 'SETTING_NOT_FOUND') {
    return '설정 항목을 찾을 수 없습니다. 목록을 새로고침해 주세요.';
  }
  if (error instanceof ApiClientError && error.code === 'INVALID_SETTING') {
    return '입력한 설정값을 저장할 수 없습니다.';
  }
  if (error instanceof ApiClientError && error.message) {
    return `서버 응답: ${error.message}`;
  }
  return '설정을 저장하지 못했습니다. 잠시 후 다시 시도해 주세요.';
}

function SettingVisibility({ isPublic }: { isPublic: boolean }) {
  const Icon = isPublic ? Globe2 : LockKeyhole;
  const label = isPublic ? '사용자 앱 공개' : '관리자 전용';

  return (
    <span className="inline-flex items-center gap-1.5 rounded-xl bg-background px-2.5 py-1 text-xs font-medium text-cool-gray">
      <Icon aria-hidden="true" className="h-3.5 w-3.5" />
      {label}
    </span>
  );
}

interface AppSettingCardProps {
  setting: AppSetting;
}

function AppSettingCard({ setting }: AppSettingCardProps) {
  const [value, setValue] = useState(setting.value);
  const updateSetting = useUpdateAppSetting();
  const isValueUnchanged = value === setting.value;
  const inputId = `app-setting-${setting.key}`;
  const errorId = `${inputId}-error`;

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (isValueUnchanged || updateSetting.isPending) return;
    updateSetting.mutate({ key: setting.key, value });
  };

  const handleValueChange = (nextValue: string) => {
    if (updateSetting.isError) updateSetting.reset();
    setValue(nextValue);
  };

  return (
    <article className="rounded-2xl border border-border-light bg-surface p-5 shadow-sm md:p-6">
      <form className="flex h-full flex-col" onSubmit={handleSubmit}>
        <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
          <div className="min-w-0">
            <h3 className="break-all font-semibold text-dark-slate">{setting.key}</h3>
            <p className="mt-1 text-sm leading-6 text-cool-gray">
              {setting.description || '등록된 설명이 없습니다.'}
            </p>
          </div>
          <SettingVisibility isPublic={setting.public === 'Y'} />
        </div>

        <div className="mt-5 flex-1">
          <label htmlFor={inputId} className="mb-1.5 block text-sm font-medium text-dark-slate">
            현재 값
          </label>
          <Input
            id={inputId}
            name={setting.key}
            value={value}
            onChange={(event) => handleValueChange(event.target.value)}
            inputMode={setting.key.endsWith('_url') ? 'url' : 'text'}
            autoCapitalize="none"
            autoCorrect="off"
            spellCheck={false}
            disabled={updateSetting.isPending}
            aria-invalid={updateSetting.isError}
            aria-describedby={updateSetting.isError ? errorId : undefined}
          />
          {updateSetting.isError && (
            <p id={errorId} role="alert" className="mt-2 text-sm text-error-text">
              {getUpdateErrorMessage(updateSetting.error)}
            </p>
          )}
        </div>

        <div className="mt-5 flex flex-col gap-4 border-t border-border-subtle pt-4 sm:flex-row sm:items-end sm:justify-between">
          <dl className="space-y-1 text-xs text-cool-gray">
            <div className="flex items-center gap-1.5">
              <Clock3 aria-hidden="true" className="h-3.5 w-3.5" />
              <dt className="sr-only">마지막 수정일시</dt>
              <dd>마지막 수정 {formatUpdatedAt(setting.updatedAt)}</dd>
            </div>
            <div className="flex gap-1.5 pl-5">
              <dt>수정자</dt>
              <dd>{setting.updatedBy === null ? '시스템' : `관리자 #${setting.updatedBy}`}</dd>
            </div>
          </dl>
          <Button
            type="submit"
            className="w-full sm:w-auto"
            disabled={isValueUnchanged || updateSetting.isPending}
          >
            {updateSetting.isPending ? '저장 중...' : '저장'}
          </Button>
        </div>
      </form>
    </article>
  );
}

function AppSettingsLoadingState() {
  return (
    <div className="rounded-2xl border border-border-light bg-surface p-6 shadow-sm">
      <div className="flex items-center gap-2 text-sm text-cool-gray">
        <RefreshCw aria-hidden="true" className="h-4 w-4" />
        앱 설정을 불러오는 중입니다.
      </div>
    </div>
  );
}

export function AppSettingsPage() {
  const settingsQuery = useAppSettings();
  const settings = settingsQuery.data ?? [];

  return (
    <div className="space-y-6">
      <header>
        <div className="flex items-center gap-2">
          <Settings2 aria-hidden="true" className="h-5 w-5 text-royal-indigo" />
          <h2 className="text-xl font-bold text-dark-slate">앱 설정</h2>
        </div>
        <p className="mt-2 text-sm text-cool-gray">
          사용자 앱에서 사용하는 원격 설정값을 관리합니다.
        </p>
      </header>

      {settingsQuery.isLoading ? (
        <AppSettingsLoadingState />
      ) : settingsQuery.isError ? (
        <div className="rounded-2xl border border-border-light bg-surface shadow-sm">
          <ErrorState
            message="앱 설정을 불러오는 데 실패했습니다."
            onRetry={() => void settingsQuery.refetch()}
          />
        </div>
      ) : settings.length === 0 ? (
        <div className="rounded-2xl border border-border-light bg-surface px-6 py-10 text-center text-sm text-cool-gray shadow-sm">
          등록된 앱 설정이 없습니다.
        </div>
      ) : (
        <section aria-label="앱 설정 목록" className="grid grid-cols-1 gap-4 xl:grid-cols-2">
          {settings.map((setting) => (
            <AppSettingCard
              key={`${setting.key}:${setting.updatedAt}`}
              setting={setting}
            />
          ))}
        </section>
      )}
    </div>
  );
}
