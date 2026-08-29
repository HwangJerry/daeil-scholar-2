// AppDownloadActions — Shared safe marketplace actions for landing download CTAs
import type { LucideIcon } from 'lucide-react';
import { Apple, Play } from 'lucide-react';
import {
  APP_DOWNLOAD_LINKS,
  type AppDownloadLink,
  type AppDownloadLinks,
} from '../../constants/appDownload';
import { cn } from '../../lib/utils';
import { Button } from '../ui/Button';

interface DownloadActionProps {
  downloadLink: AppDownloadLink;
  icon: LucideIcon;
  label: string;
}

interface AppDownloadActionsProps {
  downloadLinks?: AppDownloadLinks;
  className?: string;
}

function DownloadAction({ downloadLink, icon: Icon, label }: DownloadActionProps) {
  const actionContent = (
    <>
      <Icon aria-hidden="true" className={cn('size-5 shrink-0')} />
      <span className={cn('flex flex-col items-start leading-tight')}>
        <span>{label}</span>
        {!downloadLink.available && (
          <span className={cn('mt-1 text-caption font-normal')}>출시 준비 중</span>
        )}
      </span>
    </>
  );

  if (!downloadLink.available || !downloadLink.url) {
    return (
      <Button
        type="button"
        variant="outline"
        size="lg"
        disabled
        aria-label={`${label}, 출시 준비 중`}
        className={cn(
          'h-auto min-h-14 flex-1 justify-start border-primary-muted/40 bg-primary-light/10 px-4 text-primary-muted',
        )}
      >
        {actionContent}
      </Button>
    );
  }

  return (
    <Button
      asChild
      variant="outline"
      size="lg"
      className={cn(
        'h-auto min-h-14 flex-1 justify-start border-surface bg-surface px-4 text-primary hover:bg-primary-light',
      )}
    >
      <a href={downloadLink.url} target="_blank" rel="noreferrer">
        {actionContent}
      </a>
    </Button>
  );
}

export function AppDownloadActions({
  downloadLinks = APP_DOWNLOAD_LINKS,
  className,
}: AppDownloadActionsProps) {
  return (
    <div
      aria-label="앱 다운로드"
      className={cn('flex flex-col gap-3 sm:flex-row', className)}
    >
      <DownloadAction
        downloadLink={downloadLinks.appStore}
        icon={Apple}
        label="App Store에서 다운로드"
      />
      <DownloadAction
        downloadLink={downloadLinks.googlePlay}
        icon={Play}
        label="Google Play에서 다운로드"
      />
    </div>
  );
}
