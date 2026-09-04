// DisclosureAttachmentList — Download-focused document cards for disclosure detail
import { Download, File, FileImage, FileSpreadsheet, FileText } from 'lucide-react';
import type { FileAttachment } from '../../types/api';

interface DisclosureAttachmentListProps {
  files: FileAttachment[];
}

const FILE_ICON_BY_EXTENSION: Record<string, typeof File> = {
  csv: FileSpreadsheet,
  doc: FileText,
  docx: FileText,
  gif: FileImage,
  hwp: FileText,
  jpeg: FileImage,
  jpg: FileImage,
  pdf: FileText,
  png: FileImage,
  webp: FileImage,
  xls: FileSpreadsheet,
  xlsx: FileSpreadsheet,
};

function getFileExtension(fileName: string): string {
  return fileName.split('.').pop()?.toLowerCase() ?? 'file';
}

function formatFileSize(size: string): string {
  const bytes = Number(size);
  if (Number.isNaN(bytes)) return size;
  if (bytes < 1024) return `${bytes}B`;
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)}KB`;
  return `${(bytes / (1024 * 1024)).toFixed(1)}MB`;
}

export function DisclosureAttachmentList({ files }: DisclosureAttachmentListProps) {
  if (!files || files.length === 0) return null;

  return (
    <section aria-labelledby="disclosure-attachments-heading" className="mt-14 border-t border-border pt-10">
      <div className="max-w-2xl">
        <p className="text-2xs font-semibold uppercase tracking-[0.2em] text-text-placeholder">
          Documents
        </p>
        <h2
          id="disclosure-attachments-heading"
          className="mt-2 font-serif text-2xl font-semibold text-text-primary"
        >
          첨부 문서
        </h2>
        <p className="mt-2 text-body-sm leading-relaxed text-text-tertiary">
          파일을 선택하면 원본 공시 문서를 내려받을 수 있습니다.
        </p>
      </div>

      <ul className="mt-6 grid gap-3 sm:grid-cols-2">
        {files.map((file) => {
          const extension = getFileExtension(file.fileOrgName);
          const Icon = FILE_ICON_BY_EXTENSION[extension] ?? File;

          return (
            <li key={file.fSeq}>
              <a
                href={`${file.filePath}/${file.fileName}`}
                download={file.fileOrgName}
                aria-label={`${file.fileOrgName} 다운로드`}
                className="group flex min-h-24 items-center gap-4 rounded-xl border border-border bg-surface p-4 shadow-xs transition-all hover:-translate-y-0.5 hover:border-border-hover hover:shadow-card focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary focus-visible:ring-offset-2 focus-visible:ring-offset-background"
              >
                <span className="inline-flex size-11 shrink-0 items-center justify-center rounded-lg bg-primary-light text-primary">
                  <Icon aria-hidden="true" className="size-5" />
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block break-words text-body-sm font-semibold leading-snug text-text-primary">
                    {file.fileOrgName}
                  </span>
                  <span className="mt-1 block text-caption uppercase tracking-wide text-text-placeholder">
                    {extension} · {formatFileSize(file.fileSize)}
                  </span>
                </span>
                <Download
                  aria-hidden="true"
                  className="size-4 shrink-0 text-text-placeholder transition-colors group-hover:text-primary"
                />
              </a>
            </li>
          );
        })}
      </ul>
    </section>
  );
}
