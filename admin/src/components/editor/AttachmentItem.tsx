// AttachmentItem — single attachment row with name, size, download link, and remove button
import { X } from 'lucide-react';
import { Button } from '../ui/Button.tsx';
import { AttachmentIcon } from './AttachmentIcon.tsx';
import { formatFileSize } from './formatFileSize.ts';
import type { AttachmentDescriptor } from '../../hooks/useNoticeAttachments.ts';

interface AttachmentItemProps {
  file: AttachmentDescriptor;
  onRemove: (fSeq: number) => void;
  disabled?: boolean;
}

export function AttachmentItem({ file, onRemove, disabled }: AttachmentItemProps) {
  const downloadHref = file.filePath && file.fileName
    ? `${file.filePath}/${file.fileName}`
    : null;

  return (
    <li className="flex items-center justify-between gap-2 rounded-lg border border-border bg-surface px-3 py-2">
      <div className="flex min-w-0 items-center gap-2 text-sm">
        <AttachmentIcon
          fileName={file.fileName ?? file.fileOrgName}
          className="h-4 w-4 shrink-0 text-text-tertiary"
        />
        {downloadHref ? (
          <a
            href={downloadHref}
            download={file.fileOrgName}
            className="truncate text-royal-indigo hover:underline"
          >
            {file.fileOrgName}
          </a>
        ) : (
          <span className="truncate text-dark-slate">{file.fileOrgName}</span>
        )}
        <span className="shrink-0 text-text-placeholder">({formatFileSize(file.fileSize)})</span>
      </div>
      <Button
        variant="ghost"
        size="icon"
        onClick={() => onRemove(file.fSeq)}
        disabled={disabled}
        aria-label="첨부 삭제"
      >
        <X className="h-4 w-4" />
      </Button>
    </li>
  );
}
