// PostBody — Renders post HTML content and file attachments
import { HtmlContent } from '../common/HtmlContent';
import { FileAttachments } from './FileAttachments';
import type { FileAttachment } from '../../types/api';

interface PostBodyProps {
  contentHtml: string;
  files: FileAttachment[];
}

export function PostBody({ contentHtml, files }: PostBodyProps) {
  return (
    <>
      <HtmlContent html={contentHtml} className="text-text-secondary" />

      <FileAttachments files={files} />
    </>
  );
}
