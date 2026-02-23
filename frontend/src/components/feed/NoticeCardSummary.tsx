// NoticeCardSummary — Expandable summary text with collapse/expand toggle
import { useState } from 'react';
import { cn } from '../../lib/utils';

const SUMMARY_COLLAPSE_THRESHOLD = 120;

interface NoticeCardSummaryProps {
  summary: string;
}

export function NoticeCardSummary({ summary }: NoticeCardSummaryProps) {
  const [isExpanded, setIsExpanded] = useState(false);
  const hasLongSummary = summary.length > SUMMARY_COLLAPSE_THRESHOLD;

  return (
    <div className="mb-1">
      <p
        className={cn(
          'text-body-sm text-text-tertiary leading-relaxed whitespace-pre-line',
          !isExpanded && 'line-clamp-3'
        )}
      >
        {summary}
      </p>
      {hasLongSummary && (
        <button
          onClick={() => setIsExpanded((prev) => !prev)}
          className="mt-1 text-body-sm text-text-placeholder hover:text-text-secondary transition-colors"
        >
          {isExpanded ? '접기' : '더 보기'}
        </button>
      )}
    </div>
  );
}
