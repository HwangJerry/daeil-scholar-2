// DisclosureListItem — Responsive editorial row for a public disclosure
import { ArrowUpRight, Eye, Paperclip } from 'lucide-react';
import { Link } from 'react-router-dom';
import type { DisclosureItem } from '../../types/api';
import { formatAbsoluteDate } from '../../utils/date';

interface DisclosureListItemProps {
  item: DisclosureItem;
}

export function DisclosureListItem({ item }: DisclosureListItemProps) {
  return (
    <li className="border-b border-border last:border-b-0">
      <Link
        to={`/disclosure/${item.seq}`}
        className="group grid min-h-32 gap-6 py-7 focus-visible:rounded-md focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-4 focus-visible:ring-offset-background sm:py-9 md:grid-cols-[minmax(0,1fr)_auto_2.75rem] md:items-center md:gap-10"
      >
        <div className="min-w-0">
          <p className="text-2xs font-semibold uppercase tracking-[0.2em] text-text-placeholder">
            Disclosure Archive
          </p>
          <h2 className="mt-2 font-serif text-xl font-semibold leading-snug text-text-primary transition-colors group-hover:text-primary sm:text-2xl">
            {item.subject}
          </h2>
          {item.summary && (
            <p className="mt-2 line-clamp-2 max-w-2xl text-body-sm leading-relaxed text-text-tertiary">
              {item.summary}
            </p>
          )}
        </div>

        <dl className="flex flex-wrap items-center gap-x-4 gap-y-2 text-caption text-text-tertiary md:max-w-60 md:justify-end">
          <div>
            <dt className="sr-only">등록일</dt>
            <dd>
              <time dateTime={item.regDate}>{formatAbsoluteDate(item.regDate)}</time>
            </dd>
          </div>
          <div className="inline-flex items-center gap-1.5">
            <dt className="sr-only">첨부 문서 수</dt>
            <Paperclip aria-hidden="true" className="size-3.5" />
            <dd>{item.attachmentCount}건</dd>
          </div>
          <div className="inline-flex items-center gap-1.5">
            <dt className="sr-only">조회수</dt>
            <Eye aria-hidden="true" className="size-3.5" />
            <dd>{item.hit}</dd>
          </div>
        </dl>

        <span className="hidden size-11 items-center justify-center rounded-full border border-border bg-surface text-text-secondary transition-all group-hover:border-primary group-hover:bg-primary group-hover:text-white md:inline-flex">
          <ArrowUpRight aria-hidden="true" className="size-4" />
        </span>
      </Link>
    </li>
  );
}
