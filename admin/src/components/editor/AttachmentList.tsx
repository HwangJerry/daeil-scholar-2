// AttachmentList — vertical list of pending and existing notice attachments
import { AttachmentItem } from './AttachmentItem.tsx';
import type { AttachmentDescriptor } from '../../hooks/useNoticeAttachments.ts';

interface AttachmentListProps {
  attachments: AttachmentDescriptor[];
  onRemove: (fSeq: number) => void;
  disabled?: boolean;
}

export function AttachmentList({ attachments, onRemove, disabled }: AttachmentListProps) {
  if (attachments.length === 0) {
    return (
      <p className="text-sm text-text-placeholder">아직 첨부된 파일이 없습니다.</p>
    );
  }
  return (
    <ul className="space-y-1.5">
      {attachments.map((file) => (
        <AttachmentItem
          key={file.fSeq}
          file={file}
          onRemove={onRemove}
          disabled={disabled}
        />
      ))}
    </ul>
  );
}
