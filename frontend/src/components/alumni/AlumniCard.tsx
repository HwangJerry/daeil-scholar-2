// AlumniCard — Compact table row view for alumni directory listing
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { MessageCircle } from 'lucide-react';
import { cn } from '../../lib/utils';
import type { AlumniItem } from '../../types/api';
import { SendMessageDialog } from '../message/SendMessageDialog';
import { Button } from '../ui/Button';
import { AlumniDetailModal } from './AlumniDetailModal';
import { useResponsive } from '../../hooks/useResponsive';

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
  const [showDetail, setShowDetail] = useState(false);
  const { isMobile } = useResponsive();
  const navigate = useNavigate();
  const isSelf = item.usrSeq === currentUsrSeq;
  // Dynamic calculation — inline style exception per design doc
  // fmSeq is 0 for WEO_MEMBER-only rows, fall back to usrSeq for stable coloring
  const avatarSeed = item.fmSeq || item.usrSeq;
  const avatarColor = AVATAR_COLORS[avatarSeed % AVATAR_COLORS.length];
  const bizDisplay = item.bizName || item.company || '—';
  const hasJobCat = !!item.jobCatName && item.jobCatName !== '미분류';

  return (
    <>
      <div
        className={cn(
          'group grid items-center px-5 py-3.5 gap-2 cursor-pointer',
          'grid-cols-[44px_1fr_48px] md:grid-cols-[44px_1fr_200px_160px_72px]',
          'hover:bg-background transition-colors duration-150',
          !isLast && 'border-b border-border-subtle',
        )}
        onClick={() => setShowDetail(true)}
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
          <span className="text-sm font-semibold text-text-primary font-serif truncate block">
            {item.fmName}
          </span>
          <span className="text-[11px] text-text-placeholder">
            {[item.fmFn && `${item.fmFn}기`, item.fmDept].filter(Boolean).join(' ')}
          </span>
        </div>

        {/* Col 3 — 소속 (hidden on mobile) */}
        <div className="hidden md:flex flex-col min-w-0">
          <span className="text-sm text-text-secondary truncate">{bizDisplay}</span>
          {hasJobCat && (
            <span className="text-[11px] text-text-placeholder truncate">{item.jobCatName}</span>
          )}
        </div>

        {/* Col 4 — 직책 (hidden on mobile) */}
        <span className="hidden md:block text-sm text-text-secondary truncate">
          {item.position || '—'}
        </span>

        {/* Col 5 — 쪽지 */}
        <div className="flex justify-end">
          {!isSelf && item.usrSeq > 0 && (
            <Button
              variant="default"
              size="sm"
              onClick={(e) => {
                e.stopPropagation();
                if (isMobile) {
                  navigate('/messages/new', {
                    state: { recipientSeq: item.usrSeq, recipientName: item.fmName },
                  });
                } else {
                  setShowMessageDialog(true);
                }
              }}
              className="flex items-center gap-1 whitespace-nowrap"
            >
              <MessageCircle size={12} />
              <span className="hidden md:inline">쪽지</span>
            </Button>
          )}
        </div>
      </div>

      {showMessageDialog && (
        <SendMessageDialog
          recipientSeq={item.usrSeq}
          recipientName={item.fmName}
          onClose={() => setShowMessageDialog(false)}
        />
      )}

      {showDetail && (
        <AlumniDetailModal
          item={item}
          currentUsrSeq={currentUsrSeq}
          onClose={() => setShowDetail(false)}
        />
      )}
    </>
  );
}
