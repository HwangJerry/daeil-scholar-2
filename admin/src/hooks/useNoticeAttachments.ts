// useNoticeAttachments — manages attachment list state for the notice edit form
import { useState, useCallback, useMemo } from 'react';
import type { FileAttachment } from '../types/api.ts';
import type { AttachmentUploadResponse } from '../components/editor/uploadAttachment.ts';

export interface AttachmentDescriptor {
  fSeq: number;
  fileOrgName: string;
  fileSize: string;
  fileName?: string;
  filePath?: string;
}

export interface UseNoticeAttachmentsResult {
  attachments: AttachmentDescriptor[];
  addUploaded: (items: AttachmentUploadResponse[]) => void;
  remove: (fSeq: number) => void;
  fSeqs: number[];
}

function fromFileAttachment(f: FileAttachment): AttachmentDescriptor {
  return {
    fSeq: f.fSeq,
    fileOrgName: f.fileOrgName,
    fileSize: f.fileSize,
    fileName: f.fileName,
    filePath: f.filePath,
  };
}

function fromUploaded(u: AttachmentUploadResponse): AttachmentDescriptor {
  return {
    fSeq: u.fSeq,
    fileOrgName: u.fileOrgName,
    fileSize: u.fileSize,
    fileName: u.fileName,
    filePath: u.filePath,
  };
}

export function useNoticeAttachments(initial: FileAttachment[] | undefined): UseNoticeAttachmentsResult {
  const [attachments, setAttachments] = useState<AttachmentDescriptor[]>(
    () => (initial ?? []).map(fromFileAttachment),
  );

  const addUploaded = useCallback((items: AttachmentUploadResponse[]) => {
    setAttachments((prev) => [...prev, ...items.map(fromUploaded)]);
  }, []);

  const remove = useCallback((fSeq: number) => {
    setAttachments((prev) => prev.filter((a) => a.fSeq !== fSeq));
  }, []);

  const fSeqs = useMemo(() => attachments.map((a) => a.fSeq), [attachments]);

  return { attachments, addUploaded, remove, fSeqs };
}
