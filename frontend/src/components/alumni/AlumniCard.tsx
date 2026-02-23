// AlumniCard — Compact table row view for alumni directory listing
import { useState } from 'react';
import { MessageCircle } from 'lucide-react';
import { cn } from '../../lib/utils';
import type { AlumniItem } from '../../types/api';
import { SendMessageDialog } from '../message/SendMessageDialog';
import { Button } from '../ui/Button';

const AVATAR_COLORS = [
  '#5C6BC0', '#EF6C00', '#00897B', '#8E24AA', '#D81B60',
  '#039BE5', '#43A047', '#E53935', '#6D4C41', '#546E7A',
];

interface AlumniCardProps {
  item: AlumniItem;
  currentUsrSeq?: number;
  isLast?: boolean;
}

export function AlumniCard({ item, currentUsrSeq, isLast }: AlumniCardProps) {
  const [showMessageDialog, setShowMessageDialog] = useState(false);
  const isSelf = item.usrSeq === currentUsrSeq;
  // Dynamic calculation — inline style exception per design doc
  const avatarColor = AVATAR_COLORS[item.fmSeq % AVATAR_COLORS.length];
  const bizDisplay = item.bizName || item.company || '—';
  const hasJobCat = !!item.jobCatName && item.jobCatName !== '미분류';

  return (
    <>
      <div
        className={cn(
          'group grid items-center px-5 py-3.5 gap-2 cursor-pointer',
          'grid-cols-[44px_1fr_88px_48px] md:grid-cols-[44px_1fr_200px_160px_88px_72px]',
          'hover:bg-background transition-colors duration-150',
          !isLast && 'border-b border-border-subtle',
        )}
      >
        {/* Col 1 — Avatar */}
        <div
          className="w-8 h-8 rounded-full flex items-center justify-center text-white text-xs font-bold flex-shrink-0"
          style={{ backgroundColor: avatarColor }}
        >
          {item.fmName?.[0] ?? '?'}
        </div>

        {/* Col 2 — 동문 */}
        <div className="min-w-0">
          <div className="flex items-center gap-1.5 flex-wrap">
            <span className="text-sm font-semibold text-text-primary font-serif truncate">
              {item.fmName}
            </span>
            <span className="bg-border-subtle text-text-placeholder rounded-full px-2 py-0.5 text-[10px] font-medium whitespace-nowrap flex-shrink-0">
              {item.fmFn}기
            </span>
          </div>
          <span className="text-[11px] text-text-placeholder">{item.fmDept}</span>
        </div>

        {/* Col 3 — 소속 (hidden on mobile) */}
        <span className="hidden md:block text-sm text-text-secondary truncate">
          {bizDisplay}
        </span>

        {/* Col 4 — 직책 (hidden on mobile) */}
        <span className="hidden md:block text-sm text-text-secondary truncate">
          {item.position || '—'}
        </span>

        {/* Col 5 — 업종 */}
        <span
          className={cn(
            'rounded-full px-2 py-0.5 text-[11px] border truncate block max-w-full',
            hasJobCat
              ? 'bg-cat-notice-bg text-cat-notice-text border-cat-notice-border'
              : 'bg-border-subtle text-text-placeholder border-border',
          )}
        >
          {item.jobCatName || '미분류'}
        </span>

        {/* Col 6 — 쪽지 */}
        <div className="flex justify-end">
          <Button
            variant="default"
            size="sm"
            onClick={() => setShowMessageDialog(true)}
            disabled={isSelf}
            className={cn(
              'opacity-0 group-hover:opacity-100 transition-opacity',
              'flex items-center gap-1 whitespace-nowrap',
            )}
          >
            <MessageCircle size={12} />
            <span className="hidden md:inline">쪽지</span>
          </Button>
        </div>
      </div>

      {showMessageDialog && (
        <SendMessageDialog
          recipientSeq={item.usrSeq}
          recipientName={item.fmName}
          onClose={() => setShowMessageDialog(false)}
        />
      )}
    </>
  );
}
