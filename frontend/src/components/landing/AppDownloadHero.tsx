// AppDownloadHero — App marketplace CTA hero with a responsive phone composition
import { ArrowDown, Download } from 'lucide-react';
import androidAppIcon from '../../assets/app-icons/app-icon-android.png';
import iosAppIcon from '../../assets/app-icons/app-icon-ios.png';
import {
  APP_DOWNLOAD_LINKS,
  type AppDownloadLinks,
} from '../../constants/appDownload';
import { cn } from '../../lib/utils';
import { AppDownloadActions } from './AppDownloadActions';

interface AppDownloadHeroProps {
  downloadLinks?: AppDownloadLinks;
}

function AppPreviewVisual() {
  return (
    <figure
      aria-label="대일외고 장학회 모바일 앱 미리보기"
      className={cn(
        'relative mx-auto flex h-[330px] w-full max-w-[360px] items-center justify-center md:h-[500px] md:max-w-[440px]',
      )}
    >
      <div
        aria-hidden="true"
        className={cn(
          'absolute size-64 rounded-full bg-surface/10 blur-3xl md:size-80',
        )}
      />

      <div
        aria-hidden="true"
        className={cn(
          'absolute bottom-0 left-1/2 z-0 h-3 w-32 -translate-x-1/2 rounded-full bg-primary/30 blur-xl md:bottom-3 md:h-4 md:w-52 md:blur-2xl',
        )}
      />

      <div
        className={cn(
          'relative z-10 h-[300px] w-[158px] rounded-[2.5rem] p-px shadow-float md:h-[440px] md:w-[226px] md:rounded-[3.25rem]',
        )}
      >
        <div
          aria-hidden="true"
          className={cn(
            'absolute inset-0 rounded-[2.5rem] bg-gradient-to-b from-primary-muted/80 via-primary-muted/30 to-primary-muted/60 md:rounded-[3.25rem]',
          )}
        />

        <div
          aria-hidden="true"
          className={cn(
            'absolute -left-[3px] top-[72px] flex flex-col gap-2 md:-left-[5px] md:top-28 md:gap-3',
          )}
        >
          <span className={cn('h-4 w-1 rounded-full bg-primary-muted md:h-6 md:w-1.5')} />
          <span className={cn('h-4 w-1 rounded-full bg-primary-muted md:h-6 md:w-1.5')} />
        </div>

        <div
          aria-hidden="true"
          className={cn(
            'absolute -right-[3px] top-24 h-7 w-1 rounded-full bg-primary-muted md:-right-[5px] md:top-36 md:h-10 md:w-1.5',
          )}
        />

        <div
          className={cn(
            'relative h-full w-full rounded-[calc(2.5rem-1px)] bg-primary p-[9px] md:rounded-[calc(3.25rem-1px)] md:p-[13px]',
          )}
        >
          <div
            className={cn(
              'relative flex h-full flex-col items-center overflow-hidden rounded-[2rem] bg-background px-3 pb-4 pt-8 text-center md:rounded-[2.6rem] md:px-5 md:pb-6 md:pt-12',
            )}
          >
            <div
              aria-hidden="true"
              className={cn(
                'absolute left-1/2 top-2.5 z-20 flex -translate-x-1/2 items-center gap-1 md:top-3 md:gap-1.5',
              )}
            >
              <span
                className={cn('h-2.5 w-9 rounded-full bg-primary md:h-3.5 md:w-12')}
              />
              <span
                className={cn('size-1.5 rounded-full bg-text-primary md:size-2')}
              />
            </div>

            <div
              aria-hidden="true"
              className={cn(
                'pointer-events-none absolute inset-0 z-10 bg-gradient-to-br from-surface/10 via-transparent to-transparent',
              )}
            />

            <img
              src={iosAppIcon}
              alt="대일외고 장학회 iOS 앱 아이콘"
              width={256}
              height={256}
              loading="eager"
              fetchPriority="high"
              decoding="sync"
              className={cn(
                'size-20 rounded-xl object-cover shadow-card md:size-28 md:rounded-2xl',
              )}
            />
            <p
              className={cn(
                'mt-4 font-serif text-xs font-bold text-text-primary md:mt-6 md:text-base',
              )}
            >
              대일외고 장학회
            </p>
            <p
              className={cn(
                'mt-1 text-[9px] leading-relaxed text-text-tertiary md:text-xs',
              )}
            >
              동문을 잇고, 미래를 응원합니다.
            </p>
            <div
              className={cn(
                'mt-auto flex w-full items-center gap-2 rounded-lg border border-border bg-surface p-2 text-left shadow-card md:rounded-xl md:p-3',
              )}
            >
              <span
                className={cn(
                  'flex size-7 shrink-0 items-center justify-center rounded-md bg-primary-light text-primary md:size-9',
                )}
              >
                <Download aria-hidden="true" className={cn('size-3.5 md:size-4')} />
              </span>
              <span className={cn('min-w-0')}>
                <span
                  className={cn(
                    'block truncate text-[8px] font-semibold text-text-primary md:text-caption',
                  )}
                >
                  장학회 소식
                </span>
                <span
                  className={cn(
                    'mt-0.5 block truncate text-[7px] text-text-tertiary md:text-[9px]',
                  )}
                >
                  새로운 이야기를 확인하세요
                </span>
              </span>
            </div>

            <div
              aria-hidden="true"
              className={cn(
                'absolute bottom-2 left-1/2 z-20 h-1 w-16 -translate-x-1/2 rounded-full bg-text-primary/20 md:bottom-3 md:h-1.5 md:w-20',
              )}
            />
          </div>
        </div>
      </div>

      <div
        className={cn(
          'absolute right-1 top-9 flex items-center gap-2 rounded-xl border border-primary-muted/50 bg-surface/95 p-2 shadow-float sm:right-4 md:right-0 md:top-16 md:gap-3 md:rounded-2xl md:p-3',
        )}
      >
        <img
          src={androidAppIcon}
          alt="대일외고 장학회 Android 앱 아이콘"
          width={192}
          height={192}
          loading="eager"
          fetchPriority="high"
          decoding="sync"
          className={cn('size-12 rounded-lg object-cover md:size-16 md:rounded-xl')}
        />
        <span className={cn('hidden pr-1 text-left sm:block')}>
          <span className={cn('block text-caption font-semibold text-text-primary')}>
            Android
          </span>
          <span className={cn('mt-0.5 block text-[9px] text-text-tertiary')}>
            하나의 동문 커뮤니티
          </span>
        </span>
      </div>
    </figure>
  );
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
          'absolute -left-24 top-1/4 size-72 rounded-full bg-hero-to/40 blur-3xl',
        )}
      />
      <div
        className={cn(
          'relative mx-auto flex min-h-[calc(100svh-var(--landing-header-height))] max-w-[1080px] flex-col px-5 py-8 sm:px-8 md:px-6 md:py-12',
        )}
      >
        <div
          className={cn(
            'grid flex-1 content-center gap-y-7 md:grid-cols-[minmax(0,1fr)_minmax(320px,0.8fr)] md:grid-rows-[auto_auto] md:items-center md:gap-x-12 md:gap-y-8 lg:gap-x-20',
          )}
        >
          <div className={cn('order-1 md:col-start-1 md:row-start-1 md:self-end')}>
            <p
              className={cn(
                'text-xs font-semibold uppercase tracking-[0.24em] text-primary-muted',
              )}
            >
              DAEIL ALUMNI COMMUNITY
            </p>
            <h1
              id="app-download-heading"
              className={cn(
                'mt-4 max-w-[680px] font-serif text-4xl font-bold leading-tight tracking-tight text-surface sm:text-5xl lg:text-6xl',
              )}
            >
              대일의 오늘과 내일을 잇습니다.
            </h1>
            <p
              className={cn(
                'mt-5 max-w-xl text-base leading-7 text-primary-muted sm:text-lg sm:leading-8',
              )}
            >
              장학회 소식부터 동문 네트워크까지, 대일외고 장학회 앱에서 더 가까이
              만나보세요.
            </p>
          </div>

          <div
            className={cn(
              'order-2 md:col-start-2 md:row-span-2 md:row-start-1',
            )}
          >
            <AppPreviewVisual />
          </div>

          <AppDownloadActions
            downloadLinks={downloadLinks}
            className={cn(
              'order-3 md:col-start-1 md:row-start-2 md:self-start',
            )}
          />
        </div>

        <a
          href="#news"
          className={cn(
            'mx-auto mt-8 inline-flex min-h-11 items-center gap-2 rounded-md px-3 py-2 text-xs font-medium text-primary-muted transition-colors hover:text-surface focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary-muted focus-visible:ring-offset-2 focus-visible:ring-offset-hero-from',
          )}
        >
          최근 소식 보기
          <ArrowDown aria-hidden="true" className={cn('size-4')} />
        </a>
      </div>
    </section>
  );
}
