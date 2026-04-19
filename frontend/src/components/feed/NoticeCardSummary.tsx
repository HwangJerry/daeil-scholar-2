// NoticeCardSummary — Summary text truncated to 3 lines with fade gradient indicator

const SUMMARY_TRUNCATE_THRESHOLD = 120;

interface NoticeCardSummaryProps {
  summary: string;
}

export function NoticeCardSummary({ summary }: NoticeCardSummaryProps) {
  const isTruncated = summary.length > SUMMARY_TRUNCATE_THRESHOLD;

  return (
    <div className="mb-1 relative">
      <p className="text-body-sm text-text-tertiary leading-relaxed whitespace-pre-line line-clamp-3">
        {summary}
      </p>
      {isTruncated && (
        <div className="absolute bottom-0 left-0 right-0 h-6 bg-gradient-to-t from-surface to-transparent pointer-events-none" />
      )}
    </div>
  );
}
