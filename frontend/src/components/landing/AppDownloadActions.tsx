// AppDownloadActions — Shared safe marketplace icon actions for landing download CTAs
import { useState, type ComponentType, type SVGProps } from 'react';
import { Rocket } from 'lucide-react';
import {
  APP_DOWNLOAD_LINKS,
  type AppDownloadLink,
  type AppDownloadLinks,
} from '../../constants/appDownload';
import { useResponsive } from '../../hooks/useResponsive';
import { cn } from '../../lib/utils';
import { AppleLogoIcon } from '../icons/AppleLogoIcon';
import { GooglePlayLogoIcon } from '../icons/GooglePlayLogoIcon';
import { AlertDialog, AlertDialogContent } from '../ui/AlertDialog';
import { BottomSheet } from '../ui/BottomSheet';
import { Button } from '../ui/Button';

interface DownloadActionProps {
  downloadLink: AppDownloadLink;
  icon: ComponentType<SVGProps<SVGSVGElement>>;
  label: string;
  marketplaceName: string;
  onComingSoon: (marketplaceName: string) => void;
}

interface AppDownloadActionsProps {
  downloadLinks?: AppDownloadLinks;
  className?: string;
}

const APP_STORE_ACTION = {
  icon: AppleLogoIcon,
  label: 'App Store에서 다운로드',
  marketplaceName: 'App Store',
} as const;

const GOOGLE_PLAY_ACTION = {
  icon: GooglePlayLogoIcon,
  label: 'Google Play에서 다운로드',
  marketplaceName: 'Google Play',
} as const;

interface AppDownloadComingSoonNoticeProps {
  marketplaceName: string;
  onClose: () => void;
}

function AppDownloadComingSoonNotice({
  marketplaceName,
  onClose,
}: AppDownloadComingSoonNoticeProps) {
  const { isMobile } = useResponsive();
  const contentProps = {
    title: '출시 준비 중',
    message: `${marketplaceName}에 곧 출시될 예정입니다. 조금만 기다려 주세요!`,
    icon: Rocket,
    iconTone: 'warning' as const,
    onConfirm: onClose,
  };

  if (isMobile) {
    return (
      <BottomSheet onClose={onClose}>
        <AlertDialogContent {...contentProps} />
      </BottomSheet>
    );
  }

  return <AlertDialog open {...contentProps} />;
}

function DownloadAction({
  downloadLink,
  icon: MarketplaceIcon,
  label,
  marketplaceName,
  onComingSoon,
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
        aria-label={`${marketplaceName} 출시 준비 중 안내 보기`}
        className={actionClassName}
        onClick={() => onComingSoon(marketplaceName)}
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
  const [comingSoonMarketplace, setComingSoonMarketplace] = useState<
    string | null
  >(null);

  return (
    <>
      <div
        aria-label="앱 다운로드"
        className={cn('flex items-center gap-3', className)}
      >
        <DownloadAction
          downloadLink={downloadLinks.appStore}
          onComingSoon={setComingSoonMarketplace}
          {...APP_STORE_ACTION}
        />
        <DownloadAction
          downloadLink={downloadLinks.googlePlay}
          onComingSoon={setComingSoonMarketplace}
          {...GOOGLE_PLAY_ACTION}
        />
      </div>
      {comingSoonMarketplace && (
        <AppDownloadComingSoonNotice
          marketplaceName={comingSoonMarketplace}
          onClose={() => setComingSoonMarketplace(null)}
        />
      )}
    </>
  );
}
