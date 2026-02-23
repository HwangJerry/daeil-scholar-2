// NoticeDetailModal — Feed-to-modal notice detail view with sibling navigation
import { useParams, useNavigate, useLocation } from 'react-router-dom';
import { useQueryClient } from '@tanstack/react-query';
import { X, ChevronLeft, ChevronRight } from 'lucide-react';
import type { Location } from 'react-router-dom';
import { Modal } from '../ui/Modal';
import { PostContent } from '../post/PostContent';
import { usePostDetail } from '../../hooks/usePostDetail';
import { usePostSiblings } from '../../hooks/usePostSiblings';

function NoticeDetailSkeleton() {
  return (
    <div className="p-5 md:p-6 space-y-4">
      <div className="h-6 w-3/4 rounded skeleton-shimmer" />
      <div className="h-4 w-1/2 rounded skeleton-shimmer" />
      <div className="space-y-2">
        <div className="h-4 rounded skeleton-shimmer" />
        <div className="h-4 rounded skeleton-shimmer" />
        <div className="h-4 w-2/3 rounded skeleton-shimmer" />
      </div>
    </div>
  );
}

interface ModalPostNavigationProps {
  prev: { seq: number; subject: string } | null;
  next: { seq: number; subject: string } | null;
  onNavigate: (seq: number) => void;
}

function ModalPostNavigation({ prev, next, onNavigate }: ModalPostNavigationProps) {
  if (!prev && !next) return null;

  return (
    <nav className="border-t border-border-subtle">
      <div className="divide-y divide-border-subtle">
        {prev && (
          <button
            type="button"
            onClick={() => onNavigate(prev.seq)}
            className="w-full flex items-center gap-2 px-5 py-3 text-sm text-text-secondary hover:bg-background-secondary transition-colors text-left"
          >
            <ChevronLeft className="h-4 w-4 shrink-0 text-text-placeholder" />
            <span className="shrink-0 text-text-placeholder">이전글</span>
            <span className="truncate">{prev.subject}</span>
          </button>
        )}
        {next && (
          <button
            type="button"
            onClick={() => onNavigate(next.seq)}
            className="w-full flex items-center gap-2 px-5 py-3 text-sm text-text-secondary hover:bg-background-secondary transition-colors text-left"
          >
            <ChevronRight className="h-4 w-4 shrink-0 text-text-placeholder" />
            <span className="shrink-0 text-text-placeholder">다음글</span>
            <span className="truncate">{next.subject}</span>
          </button>
        )}
      </div>
    </nav>
  );
}

export function NoticeDetailModal() {
  const { seq } = useParams<{ seq: string }>();
  const navigate = useNavigate();
  const location = useLocation();
  const { data: post, isLoading, isError } = usePostDetail(seq);
  const { data: siblings } = usePostSiblings(post?.seq);

  const queryClient = useQueryClient();
  const handleClose = () => {
    queryClient.invalidateQueries({ queryKey: ['feed'] });
    navigate(-1);
  };

  const handleSiblingNav = (siblingSeq: number) => {
    const state = location.state as { backgroundLocation?: Location } | null;
    const backgroundLocation = state?.backgroundLocation ?? location;
    navigate(`/post/${siblingSeq}`, { state: { backgroundLocation } });
  };

  return (
    <Modal onClose={handleClose}>
      <div className="flex justify-end p-3 pb-0">
        <button
          onClick={handleClose}
          className="rounded-lg p-1.5 text-text-tertiary hover:bg-background transition-colors duration-150"
          aria-label="닫기"
        >
          <X size={20} />
        </button>
      </div>

      {isLoading && <NoticeDetailSkeleton />}

      {isError && (
        <div className="p-6 text-center">
          <p className="text-sm text-text-tertiary">게시글을 찾을 수 없습니다.</p>
        </div>
      )}

      {post && (
        <>
          <PostContent post={post} />
          <ModalPostNavigation
            prev={siblings?.prev ?? null}
            next={siblings?.next ?? null}
            onNavigate={handleSiblingNav}
          />
        </>
      )}
    </Modal>
  );
}
