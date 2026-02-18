// AlumniCard — Vertical card displaying alumni business info, location, tags, and message action
import { useState } from 'react';
import { Building2, MapPin, MessageCircle } from 'lucide-react';
import type { AlumniItem } from '../../types/api';
import { SendMessageDialog } from '../message/SendMessageDialog';

interface AlumniCardProps {
  item: AlumniItem;
  currentUsrSeq?: number;
}

export function AlumniCard({ item, currentUsrSeq }: AlumniCardProps) {
  const [showMessageDialog, setShowMessageDialog] = useState(false);

  const displayName = item.bizName || item.company;
  const hasBizInfo = !!displayName;
  const hasAddr = !!item.bizAddr;
  const hasTags = item.tags && item.tags.length > 0;
  const isSelf = item.usrSeq === currentUsrSeq;

  return (
    <>
      <div className="rounded-xl bg-surface p-5 shadow-card border border-border-subtle">
        {/* Header: name + job category badge */}
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <h3 className="text-lg font-bold text-text-primary">{item.fmName}</h3>
            <p className="text-xs text-text-tertiary mt-0.5">
              {item.fmFn}기 · {item.fmDept}
            </p>
          </div>
          {item.jobCatName ? (
            <span
              className="px-2.5 py-1 text-xs font-semibold rounded-full whitespace-nowrap"
              style={{
                backgroundColor: `${item.jobCatColor}18`,
                color: item.jobCatColor,
              }}
            >
              {item.jobCatName}
            </span>
          ) : (
            <span className="px-2.5 py-1 text-xs font-semibold rounded-full whitespace-nowrap bg-gray-100 text-gray-400">
              미분류
            </span>
          )}
        </div>

        {/* Business name */}
        <div className="flex items-center gap-2 text-sm text-text-secondary mb-1.5">
          <Building2 size={15} className="text-text-placeholder flex-shrink-0" />
          {hasBizInfo ? (
            <span className="truncate">{displayName}</span>
          ) : (
            <span className="italic text-text-placeholder text-xs">업체명 없음</span>
          )}
        </div>

        {/* Business description */}
        <p className="text-sm ml-[23px] mb-1.5 line-clamp-2">
          {item.bizDesc ? (
            <span className="text-text-tertiary">{item.bizDesc}</span>
          ) : (
            <span className="italic text-text-placeholder text-xs">업체 설명 없음</span>
          )}
        </p>

        {/* Address */}
        <div className="flex items-center gap-2 text-sm text-text-secondary mb-2">
          <MapPin size={15} className="text-text-placeholder flex-shrink-0" />
          {hasAddr ? (
            <span className="truncate">{item.bizAddr}</span>
          ) : (
            <span className="italic text-text-placeholder text-xs">주소 없음</span>
          )}
        </div>

        {/* Tags */}
        <div className="flex flex-wrap gap-1.5 mb-3">
          {hasTags ? (
            item.tags.map((tag) => (
              <span
                key={tag}
                className="px-2 py-0.5 text-xs font-medium rounded-md bg-primary-light text-primary"
              >
                #{tag}
              </span>
            ))
          ) : (
            <span className="italic text-text-placeholder text-xs">태그 없음</span>
          )}
        </div>

        {/* Message button — always visible, disabled for own card */}
        <button
          onClick={() => setShowMessageDialog(true)}
          disabled={isSelf}
          className={isSelf
            ? 'w-full flex items-center justify-center gap-2 py-2.5 rounded-lg bg-gray-100 text-gray-400 text-sm font-medium cursor-not-allowed'
            : 'w-full flex items-center justify-center gap-2 py-2.5 rounded-lg bg-primary/5 text-primary text-sm font-medium hover:bg-primary/10 transition-colors duration-150'
          }
        >
          <MessageCircle size={16} />
          쪽지보내기
        </button>
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
