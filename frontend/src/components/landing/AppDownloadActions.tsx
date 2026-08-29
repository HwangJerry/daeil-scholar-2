// AppDownloadActions — Shared safe marketplace icon actions for landing download CTAs
import type { ComponentType, SVGProps } from 'react';
import {
  APP_DOWNLOAD_LINKS,
  type AppDownloadLink,
  type AppDownloadLinks,
} from '../../constants/appDownload';
import { cn } from '../../lib/utils';
import { AppleLogoIcon } from '../icons/AppleLogoIcon';
import { GooglePlayLogoIcon } from '../icons/GooglePlayLogoIcon';
import { Button } from '../ui/Button';

interface DownloadActionProps {
  downloadLink: AppDownloadLink;
  icon: ComponentType<SVGProps<SVGSVGElement>>;
  label: string;
}

interface AppDownloadActionsProps {
  downloadLinks?: AppDownloadLinks;
  className?: string;
}

const APP_STORE_ACTION = {
  icon: AppleLogoIcon,
  label: 'App Store에서 다운로드',
} as const;

const GOOGLE_PLAY_ACTION = {
  icon: GooglePlayLogoIcon,
  label: 'Google Play에서 다운로드',
} as const;

function DownloadAction({
  downloadLink,
  icon: MarketplaceIcon,
  label,
}: DownloadActionProps) {
  const actionClassName = cn(
    'size-[52px] shrink-0 rounded-xl border border-surface/20 bg-primary p-0 text-surface shadow-card',
    'hover:border-surface/40 hover:bg-primary-hover hover:shadow-card-hover',
    'focus-visible:ring-primary-muted focus-visible:ring-offset-hero-from',
  );
  const actionIcon = (
    <MarketplaceIcon aria-hidden="true" className={cn('size-6')} />
  );

  if (!downloadLink.available || !downloadLink.url) {
    return (
      <Button
        type="button"
        variant="default"
        size="icon"
        disabled
        aria-label={`${label}, 출시 준비 중`}
        className={actionClassName}
      >
        {actionIcon}
      </Button>
    );
  }

  return (
    <Button
      asChild
      variant="default"
      size="icon"
      className={actionClassName}
    >
      <a
        href={downloadLink.url}
        target="_blank"
        rel="noreferrer"
        aria-label={label}
      >
        {actionIcon}
      </a>
    </Button>
  );
}

export function AppDownloadActions({
  downloadLinks = APP_DOWNLOAD_LINKS,
  className,
}: AppDownloadActionsProps) {
  const hasUnavailableDownload =
    !downloadLinks.appStore.available ||
    !downloadLinks.appStore.url ||
    !downloadLinks.googlePlay.available ||
    !downloadLinks.googlePlay.url;

  return (
    <div
      aria-label="앱 다운로드"
      className={cn('flex flex-col items-start gap-2', className)}
    >
      <div className={cn('flex items-center gap-3')}>
        <DownloadAction
          downloadLink={downloadLinks.appStore}
          {...APP_STORE_ACTION}
        />
        <DownloadAction
          downloadLink={downloadLinks.googlePlay}
          {...GOOGLE_PLAY_ACTION}
        />
      </div>
      {hasUnavailableDownload && (
        <p className={cn('text-caption text-primary-muted')}>출시 준비 중</p>
      )}
    </div>
  );
}
