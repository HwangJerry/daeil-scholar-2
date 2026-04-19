// AlumniDetailModal — Full-detail overlay for an alumni directory entry; bottom sheet on mobile
import { createPortal } from 'react-dom';
import { useNavigate } from 'react-router-dom';
import { X, Phone, Mail, Building2, MapPin, FileText, MessageCircle } from 'lucide-react';
import { Modal } from '../ui/Modal';
import { BottomSheet } from '../ui/BottomSheet';
import { Button } from '../ui/Button';
import { useResponsive } from '../../hooks/useResponsive';
import type { AlumniItem } from '../../types/api';

const AVATAR_COLORS = [
  '#5C6BC0', '#EF6C00', '#00897B', '#8E24AA', '#D81B60',
  '#039BE5', '#43A047', '#E53935', '#6D4C41', '#546E7A',
];

interface AlumniDetailModalProps {
  item: AlumniItem;
  currentUsrSeq?: number;
  onClose: () => void;
}

export function AlumniDetailModal({ item, currentUsrSeq, onClose }: AlumniDetailModalProps) {
  const { isMobile } = useResponsive();
  const navigate = useNavigate();

  const isSelf = item.usrSeq === currentUsrSeq;
  const avatarSeed = item.fmSeq || item.usrSeq;
  const avatarColor = AVATAR_COLORS[avatarSeed % AVATAR_COLORS.length];
  const hasJobCat = !!item.jobCatName && item.jobCatName !== '미분류';
  const bizDisplay = item.bizName || item.company;

  const handleSendMessage = () => {
    navigate(`/messages/${item.usrSeq}`);
    onClose();
  };

  const content = (
    <>
      {/* Header */}
      <div className="flex items-center justify-between px-5 pt-5 pb-0">
        <span className="text-xs text-text-placeholder">동문 정보</span>
        <button
          onClick={onClose}
          className="rounded-lg p-1.5 text-text-tertiary hover:bg-background transition-colors duration-150"
          aria-label="닫기"
        >
          <X size={20} />
        </button>
      </div>

      {/* Profile section */}
      <div className="px-5 pt-4 pb-5 flex items-start gap-4">
        <div className="flex-shrink-0">
          {item.photo ? (
            <img
              src={item.photo}
              alt={item.fmName}
              className="w-16 h-16 rounded-full object-cover"
            />
          ) : (
            <div
              className="w-16 h-16 rounded-full flex items-center justify-center text-white text-xl font-bold"
              style={{ backgroundColor: avatarColor }}
            >
              {item.fmName?.[0] ?? '?'}
            </div>
          )}
        </div>

        <div className="min-w-0 flex-1 pt-1">
          <span className="text-lg font-bold text-text-primary font-serif">
            {item.fmName}
          </span>
          {(item.fmFn || item.fmDept) && (
            <p className="text-sm text-text-secondary mt-0.5">
              {[item.fmFn && `${item.fmFn}기`, item.fmDept].filter(Boolean).join(' ')}
            </p>
          )}
        </div>
      </div>

      {/* Tags row */}
      {item.tags?.length > 0 && (
        <div className="px-5 pb-4 flex flex-wrap gap-1.5">
          {item.tags.map((tag) => (
            <span
              key={tag}
              className="rounded-full px-2.5 py-0.5 text-[11px] border bg-background text-text-secondary border-border"
            >
              #{tag}
            </span>
          ))}
        </div>
      )}

      {/* Divider */}
      <div className="border-t border-border-subtle mx-5" />

      {/* Detail fields */}
      <div className="px-5 py-4 space-y-3">
        {item.phone && (
          <div className="flex items-center gap-3">
            <Phone size={15} className="text-text-placeholder flex-shrink-0" />
            <span className="text-sm text-text-secondary">{item.phone}</span>
          </div>
        )}
        {item.email && (
          <div className="flex items-center gap-3">
            <Mail size={15} className="text-text-placeholder flex-shrink-0" />
            <span className="text-sm text-text-secondary break-all">{item.email}</span>
          </div>
        )}
        {(bizDisplay || item.position || hasJobCat) && (
          <div className="flex items-start gap-3">
            <Building2 size={15} className="text-text-placeholder flex-shrink-0 mt-0.5" />
            <div className="flex flex-col">
              {(bizDisplay || item.position) && (
                <span className="text-sm text-text-secondary">
                  {[bizDisplay, item.position].filter(Boolean).join(' / ')}
                </span>
              )}
              {hasJobCat && (
                <span className="text-[11px] text-text-placeholder">{item.jobCatName}</span>
              )}
            </div>
          </div>
        )}
        {item.bizAddr && (
          <div className="flex items-start gap-3">
            <MapPin size={15} className="text-text-placeholder flex-shrink-0 mt-0.5" />
            <span className="text-sm text-text-secondary">{item.bizAddr}</span>
          </div>
        )}
        {item.bizDesc && (
          <div className="flex items-start gap-3">
            <FileText size={15} className="text-text-placeholder flex-shrink-0 mt-0.5" />
            <span className="text-sm text-text-secondary whitespace-pre-line">{item.bizDesc}</span>
          </div>
        )}
      </div>

      {/* Biz card image */}
      {item.bizCard && (
        <>
          <div className="border-t border-border-subtle mx-5" />
          <div className="px-5 py-4">
            <p className="text-xs text-text-placeholder mb-2">명함</p>
            <img
              src={item.bizCard}
              alt="명함"
              className="w-full rounded-lg object-contain"
            />
          </div>
        </>
      )}

      {/* Action */}
      {!isSelf && item.usrSeq > 0 && (
        <>
          <div className="border-t border-border-subtle mx-5" />
          <div className="px-5 py-4 flex justify-center">
            <Button size="sm" className="flex items-center gap-1.5" onClick={handleSendMessage}>
              <MessageCircle size={14} />
              쪽지 보내기
            </Button>
          </div>
        </>
      )}
    </>
  );

  return createPortal(
    isMobile ? (
      <BottomSheet onClose={onClose}>{content}</BottomSheet>
    ) : (
      <Modal onClose={onClose} maxWidth="max-w-md">{content}</Modal>
    ),
    document.body
  );
}
