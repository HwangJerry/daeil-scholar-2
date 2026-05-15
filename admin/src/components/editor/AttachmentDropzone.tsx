// AttachmentDropzone — drag-and-drop and file-picker zone for notice attachments
import { useRef, useState, useCallback } from 'react';
import { Upload, Loader2 } from 'lucide-react';
import { useToast } from '../../hooks/useToast.ts';
import { uploadAttachment, type AttachmentUploadResponse } from './uploadAttachment.ts';
import { isBlockedAttachment } from './isBlockedAttachment.ts';

interface AttachmentDropzoneProps {
  onUploaded: (items: AttachmentUploadResponse[]) => void;
  disabled?: boolean;
}

export function AttachmentDropzone({ onUploaded, disabled }: AttachmentDropzoneProps) {
  const inputRef = useRef<HTMLInputElement>(null);
  const [isDragOver, setIsDragOver] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const addToast = useToast((s) => s.addToast);

  const processFiles = useCallback(async (files: File[]) => {
    if (files.length === 0) return;
    const blocked = files.filter(isBlockedAttachment);
    const accepted = files.filter((f) => !isBlockedAttachment(f));
    if (blocked.length > 0) {
      addToast({
        variant: 'error',
        title: '실행파일은 업로드할 수 없습니다',
        description: blocked.map((f) => f.name).join(', '),
      });
    }
    if (accepted.length === 0) return;

    setIsUploading(true);
    try {
      const settled = await Promise.allSettled(accepted.map((f) => uploadAttachment(f)));
      const success: AttachmentUploadResponse[] = [];
      const failed: string[] = [];
      settled.forEach((s, i) => {
        if (s.status === 'fulfilled') success.push(s.value);
        else failed.push(accepted[i].name);
      });
      if (success.length > 0) onUploaded(success);
      if (failed.length > 0) {
        addToast({
          variant: 'error',
          title: '일부 파일 업로드 실패',
          description: failed.join(', '),
        });
      }
    } finally {
      setIsUploading(false);
    }
  }, [addToast, onUploaded]);

  const onDrop = useCallback((e: React.DragEvent<HTMLLabelElement>) => {
    e.preventDefault();
    setIsDragOver(false);
    if (disabled || isUploading) return;
    void processFiles(Array.from(e.dataTransfer.files));
  }, [disabled, isUploading, processFiles]);

  const onPicked = useCallback((e: React.ChangeEvent<HTMLInputElement>) => {
    if (disabled || isUploading) return;
    void processFiles(Array.from(e.target.files ?? []));
    e.target.value = '';
  }, [disabled, isUploading, processFiles]);

  const baseClass = 'flex flex-col items-center justify-center gap-2 rounded-xl border-2 border-dashed px-4 py-6 text-sm transition-colors cursor-pointer select-none';
  const stateClass = isDragOver
    ? 'border-royal-indigo bg-royal-indigo/5'
    : 'border-border hover:border-royal-indigo/50';
  const disabledClass = (disabled || isUploading) ? 'pointer-events-none opacity-60' : '';

  return (
    <label
      className={`${baseClass} ${stateClass} ${disabledClass}`}
      onDragOver={(e) => { e.preventDefault(); setIsDragOver(true); }}
      onDragLeave={() => setIsDragOver(false)}
      onDrop={onDrop}
    >
      <input
        ref={inputRef}
        type="file"
        multiple
        className="hidden"
        onChange={onPicked}
        disabled={disabled || isUploading}
      />
      {isUploading ? (
        <Loader2 className="h-5 w-5 animate-spin text-royal-indigo" />
      ) : (
        <Upload className="h-5 w-5 text-text-tertiary" />
      )}
      <span className="text-dark-slate">
        {isUploading ? '업로드 중...' : '파일을 드래그하거나 클릭하여 선택'}
      </span>
      <span className="text-xs text-text-placeholder">
        문서, 이미지, 압축파일 지원 (실행파일 제외)
      </span>
    </label>
  );
}
