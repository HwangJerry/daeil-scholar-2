// FileAttachments — 첨부파일 목록 (아이콘 + 다운로드)
import { File, FileText, FileImage, FileSpreadsheet } from 'lucide-react';
import type { FileAttachment } from '../../types/api';

interface FileAttachmentsProps {
  files: FileAttachment[];
}

const EXT_ICON_MAP: Record<string, typeof File> = {
  pdf: FileText,
  doc: FileText,
  docx: FileText,
  hwp: FileText,
  jpg: FileImage,
  jpeg: FileImage,
  png: FileImage,
  gif: FileImage,
  webp: FileImage,
  xls: FileSpreadsheet,
  xlsx: FileSpreadsheet,
  csv: FileSpreadsheet,
};

function getFileIcon(fileName: string) {
  const ext = fileName.split('.').pop()?.toLowerCase() ?? '';
  return EXT_ICON_MAP[ext] ?? File;
}

function formatFileSize(size: string): string {
  const bytes = Number(size);
  if (isNaN(bytes)) return size;
  if (bytes < 1024) return `${bytes}B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)}MB`;
}

export function FileAttachments({ files }: FileAttachmentsProps) {
  if (!files || files.length === 0) return null;

  return (
    <div className="mt-6 border-t border-border-subtle pt-4">
      <h3 className="mb-2 text-sm font-medium text-text-tertiary">첨부파일</h3>
      <ul className="space-y-1">
        {files.map((file) => {
          const Icon = getFileIcon(file.fileName);
          return (
            <li key={file.fSeq}>
              <a
                href={`${file.filePath}/${file.fileName}`}
                download={file.fileOrgName}
                className="inline-flex items-center gap-2 py-2.5 text-sm text-primary hover:underline"
              >
                <Icon className="h-4 w-4 shrink-0" />
                <span>{file.fileOrgName}</span>
                <span className="text-text-placeholder">({formatFileSize(file.fileSize)})</span>
              </a>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
