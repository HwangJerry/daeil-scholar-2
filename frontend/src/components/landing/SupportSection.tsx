// SupportSection — Closing donation and app-download calls to action for the landing page
import { ArrowUpRight, HandHeart } from 'lucide-react';
import type { AppDownloadLinks } from '../../constants/appDownload';
import { APP_DOWNLOAD_LINKS } from '../../constants/appDownload';
import { EXTERNAL_DONATION_URL } from '../../constants/donation';
import { cn } from '../../lib/utils';
import { Button } from '../ui/Button';
import { AppDownloadActions } from './AppDownloadActions';

interface SupportSectionProps {
  downloadLinks?: AppDownloadLinks;
}

export function SupportSection({
  downloadLinks = APP_DOWNLOAD_LINKS,
}: SupportSectionProps) {
  return (
    <section
      aria-labelledby="support-heading"
      className={cn('border-t border-border-subtle bg-background px-5 py-16 sm:px-8 md:px-6 md:py-24')}
    >
      <div
        className={cn(
          'relative mx-auto max-w-[1080px] overflow-hidden rounded-2xl bg-gradient-to-br from-hero-from via-primary to-hero-to px-6 py-10 text-surface shadow-float sm:px-10 sm:py-12 md:px-14 md:py-16',
        )}
      >
        <div
          aria-hidden="true"
          className={cn(
            'absolute -right-20 -top-20 size-64 rounded-full bg-surface/10 blur-3xl',
          )}
        />

        <div
          className={cn(
            'relative grid gap-12 lg:grid-cols-[minmax(0,1fr)_minmax(360px,0.8fr)] lg:items-end lg:gap-16',
          )}
        >
          <div className={cn('max-w-2xl')}>
            <div className={cn('flex items-center gap-2 text-primary-muted')}>
              <HandHeart aria-hidden="true" className={cn('size-5')} />
              <p className={cn('text-xs font-semibold uppercase tracking-[0.24em]')}>
                Support the Future
              </p>
            </div>
            <h2
              id="support-heading"
              className={cn(
                'mt-4 font-serif text-3xl font-bold leading-tight tracking-tight text-surface sm:text-4xl md:text-5xl',
              )}
            >
              후배들의 가능성을 함께 열어주세요.
            </h2>
            <p
              className={cn(
                'mt-5 max-w-xl text-sm leading-7 text-primary-muted sm:text-base sm:leading-8',
              )}
            >
              동문의 응원은 한 사람의 배움에서 시작해 더 넓은 미래로 이어집니다.
              장학회의 든든한 여정에 함께해 주세요.
            </p>
            <Button
              asChild
              variant="outline"
              size="lg"
              className={cn(
                'mt-8 border-surface bg-surface text-primary hover:bg-primary-light',
              )}
            >
              <a
                href={EXTERNAL_DONATION_URL}
                target="_blank"
                rel="noopener noreferrer"
              >
                기부하기
                <ArrowUpRight aria-hidden="true" className={cn('size-4')} />
              </a>
            </Button>
          </div>

          <div className={cn('border-t border-primary-muted/30 pt-8 lg:border-l lg:border-t-0 lg:pl-10 lg:pt-0')}>
            <p className={cn('font-serif text-lg font-semibold text-surface')}>
              앱에서도 장학회와 계속 이어지세요.
            </p>
            <p className={cn('mt-2 text-body-sm leading-6 text-primary-muted')}>
              새로운 소식과 동문 네트워크를 한곳에서 만나보세요.
            </p>
            <AppDownloadActions
              downloadLinks={downloadLinks}
              className={cn('mt-6 lg:flex-col')}
            />
          </div>
        </div>
      </div>
    </section>
  );
}
