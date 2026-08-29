// AppDownloadActions — Shared safe marketplace actions for landing download CTAs
import appleAppStoreBadge from '../../assets/app-store-badges/apple-app-store-badge-ko-black.svg';
import googlePlayBadge from '../../assets/app-store-badges/google-play-badge-ko.png';
import {
  APP_DOWNLOAD_LINKS,
  type AppDownloadLink,
  type AppDownloadLinks,
} from '../../constants/appDownload';
import { cn } from '../../lib/utils';
import { Button } from '../ui/Button';

interface DownloadActionProps {
  downloadLink: AppDownloadLink;
  badgeSrc: string;
  badgeAlt: string;
  badgeWidth: number;
  badgeHeight: number;
  badgeDisplayClassName: string;
  label: string;
}

interface AppDownloadActionsProps {
  downloadLinks?: AppDownloadLinks;
  className?: string;
}

const APP_STORE_BADGE = {
  badgeSrc: appleAppStoreBadge,
  badgeAlt: 'App Store에서 다운로드 하기',
  badgeWidth: 129.70071,
  badgeHeight: 40,
  badgeDisplayClassName: 'h-12',
  label: 'App Store에서 다운로드',
} as const;

const GOOGLE_PLAY_BADGE = {
  badgeSrc: googlePlayBadge,
  badgeAlt: 'Google Play에서 다운로드',
  badgeWidth: 646,
  badgeHeight: 250,
  badgeDisplayClassName: 'h-[63px]',
  label: 'Google Play에서 다운로드',
} as const;

function DownloadAction({
  downloadLink,
  badgeSrc,
  badgeAlt,
  badgeWidth,
  badgeHeight,
  badgeDisplayClassName,
  label,
}: DownloadActionProps) {
  const actionContent = (
    <span className={cn('flex flex-col items-center gap-1.5')}>
      <span className={cn('flex h-[63px] items-center justify-center')}>
        <img
          src={badgeSrc}
          alt={badgeAlt}
          width={badgeWidth}
          height={badgeHeight}
          decoding="async"
          className={cn(
            'w-auto max-w-full shrink-0 object-contain',
            badgeDisplayClassName,
            !downloadLink.available && 'opacity-50',
          )}
        />
      </span>
      {!downloadLink.available && (
        <span className={cn('text-caption font-normal leading-none text-text-tertiary')}>
          출시 준비 중
        </span>
      )}
    </span>
  );

  const actionClassName = cn(
    'h-auto min-h-[83px] flex-1 rounded-xl border-surface/80 bg-surface px-3 py-2.5 text-text-secondary shadow-card',
    'hover:border-border-hover hover:bg-background hover:shadow-card-hover',
    'focus-visible:ring-primary-muted focus-visible:ring-offset-hero-from',
  );

  if (!downloadLink.available || !downloadLink.url) {
    return (
      <Button
        type="button"
        variant="outline"
        size="lg"
        disabled
        aria-label={`${label}, 출시 준비 중`}
        className={cn(actionClassName, 'disabled:opacity-100')}
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
      className={actionClassName}
    >
      <a
        href={downloadLink.url}
        target="_blank"
        rel="noreferrer"
        aria-label={label}
      >
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
        {...APP_STORE_BADGE}
      />
      <DownloadAction
        downloadLink={downloadLinks.googlePlay}
        {...GOOGLE_PLAY_BADGE}
      />
    </div>
  );
}
