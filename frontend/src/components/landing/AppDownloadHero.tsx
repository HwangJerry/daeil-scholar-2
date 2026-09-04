// AppDownloadHero — App marketplace CTA hero focused on the download message
import {
  APP_DOWNLOAD_LINKS,
  type AppDownloadLinks,
} from '../../constants/appDownload';
import { cn } from '../../lib/utils';
import { AppDownloadActions } from './AppDownloadActions';
import { AsciiGlobe } from './AsciiGlobe';

interface AppDownloadHeroProps {
  downloadLinks?: AppDownloadLinks;
}

export function AppDownloadHero({ downloadLinks = APP_DOWNLOAD_LINKS }: AppDownloadHeroProps) {
  return (
    <section
      id="download"
      aria-labelledby="app-download-heading"
      className={cn(
        'relative isolate mt-[var(--landing-header-height)] min-h-[calc(100svh-var(--landing-header-height))] scroll-mt-[var(--landing-header-height)] overflow-hidden bg-gradient-to-br from-hero-from via-hero-via to-hero-to text-surface',
      )}
    >
      <div
        aria-hidden="true"
        className={cn(
          'absolute -right-24 top-1/4 size-72 rounded-full bg-hero-to/40 blur-3xl',
        )}
      />
      <div
        className={cn(
          'relative mx-auto flex min-h-[calc(100svh-var(--landing-header-height))] max-w-[1080px] flex-col px-5 py-8 sm:px-8 md:px-6 md:py-12',
        )}
      >
        <div className={cn('flex flex-1 items-center')}>
          <div
            className={cn(
              'relative isolate grid w-full items-center md:grid-cols-[minmax(0,1fr)_340px] md:gap-8 lg:grid-cols-[minmax(0,1fr)_390px] lg:gap-10',
            )}
          >
            <div className={cn('relative z-10 w-full max-w-[620px]')}>
              <p
                className={cn(
                  'text-xs font-semibold uppercase tracking-[0.24em] text-primary-muted',
                )}
              >
                DAEIL SCHOLARSHIP FOUNDATION
              </p>
              <h1
                id="app-download-heading"
                className={cn(
                  'mt-4 font-serif text-4xl font-bold leading-tight tracking-tight text-surface sm:text-5xl lg:text-6xl',
                )}
              >
                대일의 오늘과 내일을 잇습니다.
              </h1>
              <p
                className={cn(
                  'mt-5 max-w-xl text-base leading-7 text-primary-muted sm:text-lg sm:leading-8',
                )}
              >
                장학회 소식부터 동문 네트워크까지, 대일외고 장학회 앱에서 더
                가까이 만나보세요.
              </p>

              <AppDownloadActions
                downloadLinks={downloadLinks}
                className={cn('mt-8')}
              />
            </div>

            <AsciiGlobe
              className={cn(
                'pointer-events-none absolute left-[62%] top-1/2 z-0 -translate-x-1/2 -translate-y-1/2 opacity-40 sm:left-[68%] sm:opacity-45 md:relative md:left-auto md:top-auto md:translate-x-0 md:translate-y-0 md:justify-self-end md:opacity-100',
              )}
            />
          </div>
        </div>

        <a
          href="#news"
          className={cn(
            'relative z-10 mx-auto mt-8 inline-flex min-h-11 self-center flex-col items-center justify-center gap-2 rounded-md px-3 py-2 text-xs font-medium text-primary-muted transition-colors hover:text-surface focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-muted focus-visible:ring-offset-2 focus-visible:ring-offset-hero-from',
          )}
        >
          최근 소식 보기
          <span
            aria-hidden="true"
            className={cn(
              'h-8 w-px origin-top bg-current animate-scroll-indicator motion-reduce:animate-none',
            )}
          />
        </a>
      </div>
    </section>
  );
}
